package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"os"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewServiceUsageEventsCommand creates the service usage events command group.
func NewServiceUsageEventsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "service-usage-events",
		Aliases: []string{"service-usage", "service-events", "sue"},
		Short:   "Manage service usage events",
		Long:    "View and manage service usage events for monitoring and billing",
	}

	cmd.AddCommand(newServiceUsageEventsListCommand())
	cmd.AddCommand(newServiceUsageEventsGetCommand())
	cmd.AddCommand(newServiceUsageEventsPurgeReseedCommand())

	return cmd
}

func newServiceUsageEventsListCommand() *cobra.Command {
	var (
		allPages            bool
		perPage             int
		afterGUID           string
		serviceInstanceName string
		servicePlanName     string
		serviceOfferingName string
		serviceBrokerName   string
		spaceName           string
		orgName             string
		startTime           string
		endTime             string
	)

	cmd := &cobra.Command{
		Use:   List,
		Short: "List service usage events",
		Long:  "List service usage events with optional filtering",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServiceUsageEventsList(cmd, allPages, perPage, afterGUID, serviceInstanceName, servicePlanName, serviceOfferingName, serviceBrokerName, spaceName, orgName, startTime, endTime)
		},
	}

	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "results per page")
	cmd.Flags().StringVar(&afterGUID, "after-guid", "", "return events after this GUID")
	cmd.Flags().StringVar(&serviceInstanceName, "service-instance-name", "", "filter by service instance name")
	cmd.Flags().StringVar(&servicePlanName, "service-plan-name", "", "filter by service plan name")
	cmd.Flags().StringVar(&serviceOfferingName, "service-offering-name", "", "filter by service offering name")
	cmd.Flags().StringVar(&serviceBrokerName, "service-broker-name", "", "filter by service broker name")
	cmd.Flags().StringVar(&spaceName, "space-name", "", "filter by space name")
	cmd.Flags().StringVar(&orgName, "org-name", "", "filter by organization name")
	cmd.Flags().StringVar(&startTime, "start-time", "", "filter events after this time (RFC3339 format)")
	cmd.Flags().StringVar(&endTime, "end-time", "", "filter events before this time (RFC3339 format)")

	return cmd
}

func runServiceUsageEventsList(cmd *cobra.Command, allPages bool, perPage int, afterGUID, serviceInstanceName, servicePlanName, serviceOfferingName, serviceBrokerName, spaceName, orgName, startTime, endTime string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Build filters using FilterBuilder
	params := buildServiceUsageEventsFilters(perPage, afterGUID, serviceInstanceName, servicePlanName, serviceOfferingName, serviceBrokerName, spaceName, orgName, startTime, endTime)

	events, err := client.ServiceUsageEvents().List(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list service usage events: %w", err)
	}

	// Handle pagination
	allEvents, err := handleServiceUsageEventsPagination(ctx, client, params, events, allPages)
	if err != nil {
		return err
	}

	// Output results
	return renderServiceUsageEventsOutput(allEvents, events.Pagination, allPages)
}

func buildServiceUsageEventsFilters(perPage int, afterGUID, serviceInstanceName, servicePlanName, serviceOfferingName, serviceBrokerName, spaceName, orgName, startTime, endTime string) *capi.QueryParams {
	return NewFilterBuilder().
		SetPerPage(perPage).
		AddFilterIf("guids", afterGUID).
		AddFilterIf("service_instance_names", serviceInstanceName).
		AddFilterIf("service_plan_names", servicePlanName).
		AddFilterIf("service_offering_names", serviceOfferingName).
		AddFilterIf("service_broker_names", serviceBrokerName).
		AddFilterIf("space_names", spaceName).
		AddFilterIf("organization_names", orgName).
		AddFilterIf("created_ats[gte]", startTime).
		AddFilterIf("created_ats[lte]", endTime).
		Build()
}

//nolint:dupl // Acceptable duplication - each pagination handler works with different resource types and endpoints
func handleServiceUsageEventsPagination(ctx context.Context, client capi.Client, params *capi.QueryParams, events *capi.ListResponse[capi.ServiceUsageEvent], allPages bool) ([]capi.ServiceUsageEvent, error) {
	if !allPages || events.Pagination.TotalPages <= 1 {
		return events.Resources, nil
	}

	handler := &PaginationHandler[capi.ServiceUsageEvent]{
		FetchPage: func(ctx context.Context, params *capi.QueryParams, page int) ([]capi.ServiceUsageEvent, *capi.Pagination, error) {
			params.Page = page

			moreEvents, err := client.ServiceUsageEvents().List(ctx, params)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to list service usage events: %w", err)
			}

			return moreEvents.Resources, &moreEvents.Pagination, nil
		},
	}

	return handler.FetchAllPages(ctx, params, allPages, events.Resources, &events.Pagination)
}

func renderServiceUsageEventsOutput(allEvents []capi.ServiceUsageEvent, pagination capi.Pagination, allPages bool) error {
	renderer := &StandardOutputRenderer[capi.ServiceUsageEvent]{
		RenderTable: renderServiceUsageEventsTable,
	}

	output := viper.GetString("output")

	return renderer.Render(allEvents, &pagination, allPages, output)
}

func renderServiceUsageEventsTable(events []capi.ServiceUsageEvent, pagination *capi.Pagination, allPages bool) error {
	if len(events) == 0 {
		_, _ = os.Stdout.WriteString("No service usage events found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("GUID", "Service Instance", "Service Plan", "Service Offering", "State", "Space", "Organization", "Created")

	for _, event := range events {
		previousState := NotAvailable
		if event.PreviousState != nil {
			previousState = *event.PreviousState
		}

		stateTransition := fmt.Sprintf("%s -> %s", previousState, event.State)

		_ = table.Append(
			event.GUID,
			event.ServiceInstanceName,
			event.ServicePlanName,
			event.ServiceOfferingName,
			stateTransition,
			event.SpaceName,
			event.OrganizationName,
			event.CreatedAt.Format(TimeFormatDisplay),
		)
	}

	_ = table.Render()

	if !allPages && pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", pagination.TotalPages)
	}

	return nil
}

func newServiceUsageEventsGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   UseGetEventGUID,
		Short: "Get service usage event details",
		Long:  "Display detailed information about a specific service usage event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			event, err := client.ServiceUsageEvents().Get(ctx, eventGUID)
			if err != nil {
				return fmt.Errorf("failed to get service usage event: %w", err)
			}

			// Output results
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				return encoder.Encode(event)
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				return encoder.Encode(event)
			default:
				_, _ = fmt.Fprintf(os.Stdout, "Service Usage Event: %s\n", event.GUID)
				_, _ = fmt.Fprintf(os.Stdout, "  Created: %s\n", event.CreatedAt.Format(TimeFormatDisplay))
				_, _ = fmt.Fprintf(os.Stdout, "  Updated: %s\n", event.UpdatedAt.Format(TimeFormatDisplay))
				_, _ = os.Stdout.WriteString("\n")

				_, _ = os.Stdout.WriteString("Service Instance Information:\n")
				_, _ = fmt.Fprintf(os.Stdout, "  Service Instance Name: %s\n", event.ServiceInstanceName)
				_, _ = fmt.Fprintf(os.Stdout, "  Service Instance GUID: %s\n", event.ServiceInstanceGUID)
				_, _ = fmt.Fprintf(os.Stdout, "  Service Instance Type: %s\n", event.ServiceInstanceType)
				_, _ = os.Stdout.WriteString("\n")

				_, _ = os.Stdout.WriteString("Service Plan Information:\n")
				_, _ = fmt.Fprintf(os.Stdout, "  Service Plan Name: %s\n", event.ServicePlanName)
				_, _ = fmt.Fprintf(os.Stdout, "  Service Plan GUID: %s\n", event.ServicePlanGUID)
				_, _ = os.Stdout.WriteString("\n")

				_, _ = os.Stdout.WriteString("Service Offering Information:\n")
				_, _ = fmt.Fprintf(os.Stdout, "  Service Offering Name: %s\n", event.ServiceOfferingName)
				_, _ = fmt.Fprintf(os.Stdout, "  Service Offering GUID: %s\n", event.ServiceOfferingGUID)
				_, _ = os.Stdout.WriteString("\n")

				_, _ = os.Stdout.WriteString("Service Broker Information:\n")
				_, _ = fmt.Fprintf(os.Stdout, "  Service Broker Name: %s\n", event.ServiceBrokerName)
				_, _ = fmt.Fprintf(os.Stdout, "  Service Broker GUID: %s\n", event.ServiceBrokerGUID)
				_, _ = os.Stdout.WriteString("\n")

				_, _ = os.Stdout.WriteString("Location Information:\n")
				_, _ = fmt.Fprintf(os.Stdout, "  Space Name: %s\n", event.SpaceName)
				_, _ = fmt.Fprintf(os.Stdout, "  Space GUID: %s\n", event.SpaceGUID)
				_, _ = fmt.Fprintf(os.Stdout, "  Organization Name: %s\n", event.OrganizationName)
				_, _ = fmt.Fprintf(os.Stdout, "  Organization GUID: %s\n", event.OrganizationGUID)
				_, _ = os.Stdout.WriteString("\n")

				_, _ = os.Stdout.WriteString("State Information:\n")

				_, _ = fmt.Fprintf(os.Stdout, "  Current State: %s\n", event.State)
				if event.PreviousState != nil {
					_, _ = fmt.Fprintf(os.Stdout, "  Previous State: %s\n", *event.PreviousState)
				}
			}

			return nil
		},
	}
}

func newServiceUsageEventsPurgeReseedCommand() *cobra.Command {
	config := PurgeReseedConfig{
		EntityType:       "service usage events",
		EntityTypePlural: "service instances",
		PurgeFunc: func(ctx context.Context, client any) error {
			if capiClient, ok := client.(interface {
				ServiceUsageEvents() interface {
					PurgeAndReseed(ctx context.Context) error
				}
			}); ok {
				return capiClient.ServiceUsageEvents().PurgeAndReseed(ctx)
			}

			return capi.ErrInvalidClientType
		},
	}

	return createPurgeAndReseedCommand(config)
}
