package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewAppUsageEventsCommand creates the app usage events command group.
func NewAppUsageEventsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "app-usage-events",
		Aliases: []string{"app-usage", "app-events", "aue"},
		Short:   "Manage application usage events",
		Long:    "View and manage application usage events for monitoring and billing",
	}

	cmd.AddCommand(newAppUsageEventsListCommand())
	cmd.AddCommand(newAppUsageEventsGetCommand())
	cmd.AddCommand(newAppUsageEventsPurgeReseedCommand())

	return cmd
}

func newAppUsageEventsListCommand() *cobra.Command {
	var opts appUsageEventListOptions

	cmd := &cobra.Command{
		Use:   List,
		Short: "List application usage events",
		Long:  "List application usage events with optional filtering",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAppUsageEventsList(cmd, &opts)
		},
	}

	cmd.Flags().BoolVar(&opts.allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&opts.perPage, "per-page", constants.DefaultPageSize, "results per page")
	cmd.Flags().StringVar(&opts.afterGUID, "after-guid", "", "return events after this GUID")
	cmd.Flags().StringVar(&opts.appName, "app-name", "", "filter by application name")
	cmd.Flags().StringVar(&opts.spaceName, "space-name", "", "filter by space name")
	cmd.Flags().StringVar(&opts.orgName, "org-name", "", "filter by organization name")
	cmd.Flags().StringVar(&opts.startTime, "start-time", "", "filter events after this time (RFC3339 format)")
	cmd.Flags().StringVar(&opts.endTime, "end-time", "", "filter events before this time (RFC3339 format)")

	return cmd
}

type appUsageEventListOptions struct {
	allPages  bool
	perPage   int
	afterGUID string
	appName   string
	spaceName string
	orgName   string
	startTime string
	endTime   string
}

func runAppUsageEventsList(cmd *cobra.Command, opts *appUsageEventListOptions) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()
	params := buildAppUsageEventParams(opts)

	events, err := client.AppUsageEvents().List(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list app usage events: %w", err)
	}

	allEvents := events.Resources
	if opts.allPages && events.Pagination.TotalPages > 1 {
		moreEvents, err := fetchAllAppUsageEventPages(ctx, client, params, events.Pagination.TotalPages)
		if err != nil {
			return err
		}

		allEvents = append(allEvents, moreEvents...)
	}

	return outputAppUsageEvents(allEvents, events.Pagination, opts.allPages)
}

func buildAppUsageEventParams(opts *appUsageEventListOptions) *capi.QueryParams {
	params := capi.NewQueryParams()
	params.PerPage = opts.perPage

	if opts.afterGUID != "" {
		params.WithFilter("guids", opts.afterGUID)
	}

	if opts.appName != "" {
		params.WithFilter("app_names", opts.appName)
	}

	if opts.spaceName != "" {
		params.WithFilter("space_names", opts.spaceName)
	}

	if opts.orgName != "" {
		params.WithFilter("organization_names", opts.orgName)
	}

	if opts.startTime != "" {
		params.WithFilter("created_ats[gte]", opts.startTime)
	}

	if opts.endTime != "" {
		params.WithFilter("created_ats[lte]", opts.endTime)
	}

	return params
}

func fetchAllAppUsageEventPages(ctx context.Context, client capi.Client, params *capi.QueryParams, totalPages int) ([]capi.AppUsageEvent, error) {
	var allEvents []capi.AppUsageEvent

	for page := 2; page <= totalPages; page++ {
		params.Page = page

		moreEvents, err := client.AppUsageEvents().List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		allEvents = append(allEvents, moreEvents.Resources...)
	}

	return allEvents, nil
}

func outputAppUsageEvents(events []capi.AppUsageEvent, pagination capi.Pagination, allPages bool) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return outputAppUsageEventsJSON(events)
	case OutputFormatYAML:
		return outputAppUsageEventsYAML(events)
	default:
		return outputAppUsageEventsTable(events, pagination, allPages)
	}
}

func outputAppUsageEventsJSON(events []capi.AppUsageEvent) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(events)
	if err != nil {
		return fmt.Errorf("failed to encode app usage events as JSON: %w", err)
	}

	return nil
}

func outputAppUsageEventsYAML(events []capi.AppUsageEvent) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(events)
	if err != nil {
		return fmt.Errorf("failed to encode app usage events as YAML: %w", err)
	}

	return nil
}

func outputAppUsageEventsTable(events []capi.AppUsageEvent, pagination capi.Pagination, allPages bool) error {
	if len(events) == 0 {
		_, _ = os.Stdout.WriteString("No app usage events found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("GUID", "App Name", "Space", "Organization", "State", "Instances", "Memory MB", "Created")

	for _, event := range events {
		stateTransition := formatStateTransition(event.PreviousState, event.State)
		_ = table.Append(
			event.GUID,
			event.AppName,
			event.SpaceName,
			event.OrganizationName,
			stateTransition,
			strconv.Itoa(event.InstanceCount),
			strconv.Itoa(event.MemoryInMBPerInstance),
			event.CreatedAt.Format(TimeFormatDisplay),
		)
	}

	_ = table.Render()

	if !allPages && pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", pagination.TotalPages)
	}

	return nil
}

func formatStateTransition(previousState *string, currentState string) string {
	previous := NotAvailable
	if previousState != nil {
		previous = *previousState
	}

	return fmt.Sprintf("%s -> %s", previous, currentState)
}

func newAppUsageEventsGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   UseGetEventGUID,
		Short: "Get app usage event details",
		Long:  "Display detailed information about a specific app usage event",
		Args:  cobra.ExactArgs(1),
		RunE:  runAppUsageEventGet,
	}
}

func runAppUsageEventGet(cmd *cobra.Command, args []string) error {
	eventGUID := args[0]

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	event, err := client.AppUsageEvents().Get(ctx, eventGUID)
	if err != nil {
		return fmt.Errorf("failed to get app usage event: %w", err)
	}

	return outputAppUsageEvent(event)
}

func outputAppUsageEvent(event *capi.AppUsageEvent) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return outputAppUsageEventJSON(event)
	case OutputFormatYAML:
		return outputAppUsageEventYAML(event)
	default:
		return outputAppUsageEventText(event)
	}
}

func outputAppUsageEventJSON(event *capi.AppUsageEvent) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(event)
	if err != nil {
		return fmt.Errorf("failed to encode app usage event as JSON: %w", err)
	}

	return nil
}

func outputAppUsageEventYAML(event *capi.AppUsageEvent) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(event)
	if err != nil {
		return fmt.Errorf("failed to encode app usage event as YAML: %w", err)
	}

	return nil
}

func outputAppUsageEventText(event *capi.AppUsageEvent) error {
	printAppUsageEventHeader(event)
	printAppUsageEventAppInfo(event)
	printAppUsageEventStateInfo(event)
	printAppUsageEventResourceUsage(event)
	printAppUsageEventBuildInfo(event)
	printAppUsageEventProcessInfo(event)

	return nil
}

func printAppUsageEventHeader(event *capi.AppUsageEvent) {
	_, _ = fmt.Fprintf(os.Stdout, "App Usage Event: %s\n", event.GUID)
	_, _ = fmt.Fprintf(os.Stdout, "  Created: %s\n", event.CreatedAt.Format(TimeFormatDisplay))
	_, _ = fmt.Fprintf(os.Stdout, "  Updated: %s\n", event.UpdatedAt.Format(TimeFormatDisplay))
	_, _ = os.Stdout.WriteString("\n")
}

func printAppUsageEventAppInfo(event *capi.AppUsageEvent) {
	_, _ = os.Stdout.WriteString("Application Information:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  App Name: %s\n", event.AppName)
	_, _ = fmt.Fprintf(os.Stdout, "  App GUID: %s\n", event.AppGUID)
	_, _ = fmt.Fprintf(os.Stdout, "  Space Name: %s\n", event.SpaceName)
	_, _ = fmt.Fprintf(os.Stdout, "  Space GUID: %s\n", event.SpaceGUID)
	_, _ = fmt.Fprintf(os.Stdout, "  Organization Name: %s\n", event.OrganizationName)
	_, _ = fmt.Fprintf(os.Stdout, "  Organization GUID: %s\n", event.OrganizationGUID)
	_, _ = os.Stdout.WriteString("\n")
}

func printAppUsageEventStateInfo(event *capi.AppUsageEvent) {
	_, _ = os.Stdout.WriteString("State Information:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  Current State: %s\n", event.State)

	if event.PreviousState != nil {
		_, _ = fmt.Fprintf(os.Stdout, "  Previous State: %s\n", *event.PreviousState)
	}

	_, _ = os.Stdout.WriteString("\n")
}

func printAppUsageEventResourceUsage(event *capi.AppUsageEvent) {
	_, _ = os.Stdout.WriteString("Resource Usage:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  Instance Count: %d\n", event.InstanceCount)

	if event.PreviousInstanceCount != nil {
		_, _ = fmt.Fprintf(os.Stdout, "  Previous Instance Count: %d\n", *event.PreviousInstanceCount)
	}

	_, _ = fmt.Fprintf(os.Stdout, "  Memory per Instance: %d MB\n", event.MemoryInMBPerInstance)

	if event.PreviousMemoryInMBPerInstance != nil {
		_, _ = fmt.Fprintf(os.Stdout, "  Previous Memory per Instance: %d MB\n", *event.PreviousMemoryInMBPerInstance)
	}

	_, _ = os.Stdout.WriteString("\n")
}

func printAppUsageEventBuildInfo(event *capi.AppUsageEvent) {
	_, _ = os.Stdout.WriteString("Build Information:\n")

	if event.BuildpackName != nil {
		_, _ = fmt.Fprintf(os.Stdout, "  Buildpack Name: %s\n", *event.BuildpackName)
	}

	if event.BuildpackGUID != nil {
		_, _ = fmt.Fprintf(os.Stdout, "  Buildpack GUID: %s\n", *event.BuildpackGUID)
	}

	_, _ = fmt.Fprintf(os.Stdout, "  Package State: %s\n", event.Package.State)
	_, _ = os.Stdout.WriteString("\n")
}

func printAppUsageEventProcessInfo(event *capi.AppUsageEvent) {
	_, _ = os.Stdout.WriteString("Process Information:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  Process Type: %s\n", event.ProcessType)

	if event.TaskName != nil {
		_, _ = fmt.Fprintf(os.Stdout, "  Task Name: %s\n", *event.TaskName)
	}

	if event.TaskGUID != nil {
		_, _ = fmt.Fprintf(os.Stdout, "  Task GUID: %s\n", *event.TaskGUID)
	}

	if event.ParentAppName != nil {
		_, _ = fmt.Fprintf(os.Stdout, "  Parent App Name: %s\n", *event.ParentAppName)
	}

	if event.ParentAppGUID != nil {
		_, _ = fmt.Fprintf(os.Stdout, "  Parent App GUID: %s\n", *event.ParentAppGUID)
	}
}

func newAppUsageEventsPurgeReseedCommand() *cobra.Command {
	config := PurgeReseedConfig{
		EntityType:       "app usage events",
		EntityTypePlural: "applications",
		PurgeFunc: func(ctx context.Context, client any) error {
			if capiClient, ok := client.(interface {
				AppUsageEvents() interface {
					PurgeAndReseed(ctx context.Context) error
				}
			}); ok {
				return capiClient.AppUsageEvents().PurgeAndReseed(ctx)
			}

			return capi.ErrInvalidClientType
		},
	}

	return createPurgeAndReseedCommand(config)
}
