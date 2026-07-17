package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewAuditEventsCommand creates the audit events command group.
func NewAuditEventsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "audit-events",
		Aliases: []string{"audit", "events", "ae"},
		Short:   "Manage audit events",
		Long:    "View audit events for tracking system changes and user actions",
	}

	cmd.AddCommand(newAuditEventsListCommand())
	cmd.AddCommand(newAuditEventsGetCommand())

	return cmd
}

func newAuditEventsListCommand() *cobra.Command {
	var (
		allPages    bool
		perPage     int
		eventTypes  []string
		targetTypes []string
		actorTypes  []string
		spaceName   string
		orgName     string
		startTime   string
		endTime     string
	)

	cmd := &cobra.Command{
		Use:   List,
		Short: "List audit events",
		Long:  "List audit events with optional filtering",
		RunE: func(cmd *cobra.Command, args []string) error {
			filters := &auditEventFilters{
				allPages:    allPages,
				perPage:     perPage,
				eventTypes:  eventTypes,
				targetTypes: targetTypes,
				actorTypes:  actorTypes,
				spaceName:   spaceName,
				orgName:     orgName,
				startTime:   startTime,
				endTime:     endTime,
			}

			return runAuditEventsList(cmd, filters)
		},
	}

	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.DefaultPageSize, "results per page")
	cmd.Flags().StringSliceVar(&eventTypes, "event-types", nil, "filter by event types (comma-separated)")
	cmd.Flags().StringSliceVar(&targetTypes, "target-types", nil, "filter by target types (comma-separated)")
	cmd.Flags().StringSliceVar(&actorTypes, "actor-types", nil, "filter by actor types (comma-separated)")
	cmd.Flags().StringVar(&spaceName, "space-name", "", "filter by space name")
	cmd.Flags().StringVar(&orgName, "org-name", "", "filter by organization name")
	cmd.Flags().StringVar(&startTime, "start-time", "", "filter events after this time (RFC3339 format)")
	cmd.Flags().StringVar(&endTime, "end-time", "", "filter events before this time (RFC3339 format)")

	return cmd
}

type auditEventFilters struct {
	allPages    bool
	perPage     int
	eventTypes  []string
	targetTypes []string
	actorTypes  []string
	spaceName   string
	orgName     string
	startTime   string
	endTime     string
}

func runAuditEventsList(cmd *cobra.Command, filters *auditEventFilters) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	params, err := buildAuditEventsParams(ctx, client, filters)
	if err != nil {
		return err
	}

	events, err := client.AuditEvents().List(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list audit events: %w", err)
	}

	allEvents, err := fetchAllAuditEventsPages(ctx, client, events, params, filters.allPages)
	if err != nil {
		return err
	}

	return outputAuditEventsList(allEvents, events, filters.allPages)
}

func buildAuditEventsParams(ctx context.Context, client capi.Client, filters *auditEventFilters) (*capi.QueryParams, error) {
	params := capi.NewQueryParams()
	params.PerPage = filters.perPage

	addStringSliceFilter(params, "types", filters.eventTypes)
	addStringSliceFilter(params, "target_types", filters.targetTypes)
	addStringSliceFilter(params, "actor_types", filters.actorTypes)

	err := addSpaceFilter(ctx, client, params, filters.spaceName)
	if err != nil {
		return nil, err
	}

	err = addOrgFilter(ctx, client, params, filters.orgName)
	if err != nil {
		return nil, err
	}

	addTimeFilter(params, "created_ats[gte]", filters.startTime)
	addTimeFilter(params, "created_ats[lte]", filters.endTime)

	return params, nil
}

func addStringSliceFilter(params *capi.QueryParams, filterName string, values []string) {
	if len(values) > 0 {
		params.WithFilter(filterName, strings.Join(values, ","))
	}
}

func addSpaceFilter(ctx context.Context, client capi.Client, params *capi.QueryParams, spaceName string) error {
	if spaceName == "" {
		return nil
	}

	spaceParams := capi.NewQueryParams()
	spaceParams.WithFilter("names", spaceName)

	if orgGUID := viper.GetString("organization_guid"); orgGUID != "" {
		spaceParams.WithFilter("organization_guids", orgGUID)
	}

	spaces, err := client.Spaces().List(ctx, spaceParams)
	if err != nil {
		return fmt.Errorf("failed to find space: %w", err)
	}

	if len(spaces.Resources) == 0 {
		return fmt.Errorf("%w: '%s'", capi.ErrSpaceNotFound, spaceName)
	}

	params.WithFilter("space_guids", spaces.Resources[0].GUID)

	return nil
}

func addOrgFilter(ctx context.Context, client capi.Client, params *capi.QueryParams, orgName string) error {
	if orgName == "" {
		return nil
	}

	orgParams := capi.NewQueryParams()
	orgParams.WithFilter("names", orgName)

	orgs, err := client.Organizations().List(ctx, orgParams)
	if err != nil {
		return fmt.Errorf("failed to find organization: %w", err)
	}

	if len(orgs.Resources) == 0 {
		return fmt.Errorf("%w: '%s'", capi.ErrOrganizationNotFound, orgName)
	}

	params.WithFilter("organization_guids", orgs.Resources[0].GUID)

	return nil
}

func addTimeFilter(params *capi.QueryParams, filterName, timeValue string) {
	if timeValue != "" {
		params.WithFilter(filterName, timeValue)
	}
}

func fetchAllAuditEventsPages(ctx context.Context, client capi.Client, events *capi.AuditEventsList, params *capi.QueryParams, allPages bool) ([]capi.AuditEvent, error) {
	allEvents := events.Resources
	if !allPages || events.Pagination.TotalPages <= 1 {
		return allEvents, nil
	}

	for page := 2; page <= events.Pagination.TotalPages; page++ {
		params.Page = page

		moreEvents, err := client.AuditEvents().List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		allEvents = append(allEvents, moreEvents.Resources...)
	}

	return allEvents, nil
}

func outputAuditEventsList(allEvents []capi.AuditEvent, events *capi.AuditEventsList, allPages bool) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return outputAuditEventsJSON(allEvents)
	case OutputFormatYAML:
		return outputAuditEventsYAML(allEvents)
	default:
		return outputAuditEventsTable(allEvents, events, allPages)
	}
}

func outputAuditEventsJSON(events []capi.AuditEvent) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(events)
	if err != nil {
		return fmt.Errorf("failed to encode audit events as JSON: %w", err)
	}

	return nil
}

func outputAuditEventsYAML(events []capi.AuditEvent) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(events)
	if err != nil {
		return fmt.Errorf("failed to encode audit events as YAML: %w", err)
	}

	return nil
}

func outputAuditEventsTable(allEvents []capi.AuditEvent, events *capi.AuditEventsList, allPages bool) error {
	if len(allEvents) == 0 {
		_, _ = os.Stdout.WriteString("No audit events found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("GUID", "Type", "Actor", "Target", "Space", "Organization", "Created")

	for _, event := range allEvents {
		actorInfo := formatActorInfo(event.Actor.Name, event.Actor.Type)
		targetInfo := formatTargetInfo(event.Target.Name, event.Target.Type)
		spaceName := formatSpaceName(event.Space)
		orgName := formatOrgName(event.Organization)

		_ = table.Append(
			event.GUID,
			event.Type,
			actorInfo,
			targetInfo,
			spaceName,
			orgName,
			event.CreatedAt.Format(TimeFormatDisplay),
		)
	}

	_ = table.Render()

	if !allPages && events.Pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", events.Pagination.TotalPages)
	}

	return nil
}

func formatActorInfo(name, actorType string) string {
	actorInfo := fmt.Sprintf("%s (%s)", name, actorType)
	if len(actorInfo) > constants.ActorInfoDisplayLength {
		return actorInfo[:27] + "..."
	}

	return actorInfo
}

func formatTargetInfo(name, targetType string) string {
	targetInfo := fmt.Sprintf("%s (%s)", name, targetType)
	if len(targetInfo) > constants.TargetInfoDisplayLength {
		return targetInfo[:27] + "..."
	}

	return targetInfo
}

func formatSpaceName(space any) string {
	if space == nil {
		return constants.NotAvailable
	}

	switch s := space.(type) {
	case *capi.Space:
		return s.Name
	case *capi.AuditEventSpace:
		return s.Name
	default:
		return constants.NotAvailable
	}
}

func formatOrgName(org any) string {
	if org == nil {
		return constants.NotAvailable
	}

	switch o := org.(type) {
	case *capi.Organization:
		return o.Name
	case *capi.AuditEventOrganization:
		return o.Name
	default:
		return constants.NotAvailable
	}
}

func newAuditEventsGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   UseGetEventGUID,
		Short: "Get audit event details",
		Long:  "Display detailed information about a specific audit event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuditEventsGet(cmd, args[0])
		},
	}
}

func runAuditEventsGet(cmd *cobra.Command, eventGUID string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	event, err := client.AuditEvents().Get(ctx, eventGUID)
	if err != nil {
		return fmt.Errorf("failed to get audit event: %w", err)
	}

	return outputAuditEventDetails(event)
}

func outputAuditEventDetails(event *capi.AuditEvent) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return outputAuditEventDetailsJSON(event)
	case OutputFormatYAML:
		return outputAuditEventDetailsYAML(event)
	default:
		return outputAuditEventDetailsTable(event)
	}
}

func outputAuditEventDetailsJSON(event *capi.AuditEvent) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(event)
	if err != nil {
		return fmt.Errorf("failed to encode audit event details as JSON: %w", err)
	}

	return nil
}

func outputAuditEventDetailsYAML(event *capi.AuditEvent) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(event)
	if err != nil {
		return fmt.Errorf("failed to encode audit event details as YAML: %w", err)
	}

	return nil
}

func outputAuditEventDetailsTable(event *capi.AuditEvent) error {
	printEventBasicInfo(event)
	printActorInfo(event.Actor)
	printTargetInfo(event.Target)
	printSpaceInfo(event.Space)
	printOrganizationInfo(event.Organization)
	printEventData(event.Data)

	return nil
}

func printEventBasicInfo(event *capi.AuditEvent) {
	_, _ = fmt.Fprintf(os.Stdout, "Audit Event: %s\n", event.GUID)
	_, _ = fmt.Fprintf(os.Stdout, "  Type: %s\n", event.Type)
	_, _ = fmt.Fprintf(os.Stdout, "  Created: %s\n", event.CreatedAt.Format(TimeFormatDisplay))
	_, _ = fmt.Fprintf(os.Stdout, "  Updated: %s\n", event.UpdatedAt.Format(TimeFormatDisplay))
	_, _ = os.Stdout.WriteString("\n")
}

func printActorInfo(actor capi.Actor) {
	_, _ = os.Stdout.WriteString("Actor Information:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  GUID: %s\n", actor.GUID)
	_, _ = fmt.Fprintf(os.Stdout, "  Type: %s\n", actor.Type)
	_, _ = fmt.Fprintf(os.Stdout, "  Name: %s\n", actor.Name)
	_, _ = os.Stdout.WriteString("\n")
}

func printTargetInfo(target capi.Target) {
	_, _ = os.Stdout.WriteString("Target Information:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  GUID: %s\n", target.GUID)
	_, _ = fmt.Fprintf(os.Stdout, "  Type: %s\n", target.Type)
	_, _ = fmt.Fprintf(os.Stdout, "  Name: %s\n", target.Name)
	_, _ = os.Stdout.WriteString("\n")
}

func printSpaceInfo(space any) {
	if space == nil {
		return
	}

	_, _ = os.Stdout.WriteString("Space Information:\n")

	switch spaceInfo := space.(type) {
	case *capi.Space:
		_, _ = fmt.Fprintf(os.Stdout, "  GUID: %s\n", spaceInfo.GUID)
		_, _ = fmt.Fprintf(os.Stdout, "  Name: %s\n", spaceInfo.Name)
	case *capi.AuditEventSpace:
		_, _ = fmt.Fprintf(os.Stdout, "  GUID: %s\n", spaceInfo.GUID)
		_, _ = fmt.Fprintf(os.Stdout, "  Name: %s\n", spaceInfo.Name)
	}

	_, _ = os.Stdout.WriteString("\n")
}

func printOrganizationInfo(org any) {
	if org == nil {
		return
	}

	_, _ = os.Stdout.WriteString("Organization Information:\n")

	switch orgInfo := org.(type) {
	case *capi.Organization:
		_, _ = fmt.Fprintf(os.Stdout, "  GUID: %s\n", orgInfo.GUID)
		_, _ = fmt.Fprintf(os.Stdout, "  Name: %s\n", orgInfo.Name)
	case *capi.AuditEventOrganization:
		_, _ = fmt.Fprintf(os.Stdout, "  GUID: %s\n", orgInfo.GUID)
		_, _ = fmt.Fprintf(os.Stdout, "  Name: %s\n", orgInfo.Name)
	}

	_, _ = os.Stdout.WriteString("\n")
}

func printEventData(data map[string]any) {
	if len(data) == 0 {
		return
	}

	_, _ = os.Stdout.WriteString("Event Data:\n")

	for key, value := range data {
		printEventDataValue(key, value, "  ")
	}
}

func printEventDataValue(key string, value any, indent string) {
	if valueMap, ok := value.(map[string]any); ok {
		_, _ = fmt.Fprintf(os.Stdout, "%s%s:\n", indent, key)

		for subKey, subValue := range valueMap {
			printEventDataValue(subKey, subValue, indent+"  ")
		}
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "%s%s: %v\n", indent, key, value)
	}
}
