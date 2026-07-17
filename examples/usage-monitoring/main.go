package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/fivetwenty-io/capi/v3/pkg/cfclient"
)

func main() {
	if len(os.Args) < constants.MinimumArgumentCount {
		log.Println("Usage: go run main.go <cf-api-endpoint>")
		log.Println("Example: go run main.go https://api.cf.example.com")
		os.Exit(1)
	}

	endpoint := os.Args[1]

	ctx := context.Background()

	client, err := cfclient.NewWithPassword(ctx,
		endpoint,
		os.Getenv("CF_USERNAME"),
		os.Getenv("CF_PASSWORD"),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	log.Println("=== Cloud Foundry Usage Monitoring Examples ===")
	log.Println()

	// Application Usage Events
	log.Println("1. Application Usage Events")
	demonstrateAppUsageEvents(ctx, client)

	// Service Usage Events
	log.Println("\n2. Service Usage Events")
	demonstrateServiceUsageEvents(ctx, client)

	// Audit Events
	log.Println("\n3. Audit Events")
	demonstrateAuditEvents(ctx, client)

	// Environment Variable Groups
	log.Println("\n4. Environment Variable Groups")
	demonstrateEnvironmentVariableGroups(ctx, client)
}

func demonstrateAppUsageEvents(ctx context.Context, client capi.Client) {
	events := listAppUsageEvents(ctx, client)
	if events == nil {
		return
	}

	showDetailedAppEvent(ctx, client, events)
	filterAppEventsByTimeRange(ctx, client)
}

func listAppUsageEvents(ctx context.Context, client capi.Client) *capi.ListResponse[capi.AppUsageEvent] {
	log.Println("   Listing recent application usage events...")

	params := capi.NewQueryParams()
	params.WithPerPage(constants.DefaultPageSize)

	events, err := client.AppUsageEvents().List(ctx, params)
	if err != nil {
		log.Printf("   Failed to list app usage events: %v", err)

		return nil
	}

	log.Printf("   Found %d app usage events\n", len(events.Resources))

	for i, event := range events.Resources {
		if i >= constants.DemoDisplayLimit { // Show only first 3 for demo
			break
		}

		previousState := "N/A"
		if event.PreviousState != nil {
			previousState = *event.PreviousState
		}

		log.Printf("     - App: %s, State: %s -> %s, Instances: %d, Memory: %d MB\n",
			event.AppName,
			previousState,
			event.State,
			event.InstanceCount,
			event.MemoryInMBPerInstance)
	}

	return events
}

func showDetailedAppEvent(ctx context.Context, client capi.Client, events *capi.ListResponse[capi.AppUsageEvent]) {
	if len(events.Resources) == 0 {
		return
	}

	log.Println("\n   Getting detailed event information...")

	event, err := client.AppUsageEvents().Get(ctx, events.Resources[0].GUID)
	if err != nil {
		log.Printf("   Failed to get app usage event: %v", err)

		return
	}

	printAppEventDetails(event)
}

func printAppEventDetails(event *capi.AppUsageEvent) {
	log.Printf("   Event Details:\n")
	log.Printf("     GUID: %s\n", event.GUID)
	log.Printf("     App: %s (%s)\n", event.AppName, event.AppGUID)
	log.Printf("     Space: %s (%s)\n", event.SpaceName, event.SpaceGUID)
	log.Printf("     Organization: %s (%s)\n", event.OrganizationName, event.OrganizationGUID)
	log.Printf("     Process Type: %s\n", event.ProcessType)
	log.Printf("     Created: %s\n", event.CreatedAt.Format(time.RFC3339))

	if event.BuildpackName != nil {
		log.Printf("     Buildpack: %s\n", *event.BuildpackName)
	}

	if event.TaskName != nil {
		log.Printf("     Task: %s\n", *event.TaskName)
	}
}

func filterAppEventsByTimeRange(ctx context.Context, client capi.Client) {
	log.Println("\n   Filtering events by time range (last 24 hours)...")

	yesterday := time.Now().Add(-24 * time.Hour)
	timeParams := capi.NewQueryParams()
	timeParams.WithFilter("created_ats[gte]", yesterday.Format(time.RFC3339))
	timeParams.WithPerPage(constants.SmallPageSize)

	recentEvents, err := client.AppUsageEvents().List(ctx, timeParams)
	if err != nil {
		log.Printf("   Failed to filter app usage events: %v", err)

		return
	}

	log.Printf("   Found %d events in the last 24 hours\n", len(recentEvents.Resources))
}

func demonstrateServiceUsageEvents(ctx context.Context, client capi.Client) {
	// List recent service usage events
	log.Println("   Listing recent service usage events...")

	params := capi.NewQueryParams()
	params.WithPerPage(constants.DefaultPageSize)

	events, err := client.ServiceUsageEvents().List(ctx, params)
	if err != nil {
		log.Printf("   Failed to list service usage events: %v", err)

		return
	}

	log.Printf("   Found %d service usage events\n", len(events.Resources))

	for i, event := range events.Resources {
		if i >= constants.DemoDisplayLimit { // Show only first 3 for demo
			break
		}

		log.Printf("     - Service: %s (%s), State: %s, Plan: %s\n",
			event.ServiceInstanceName,
			event.ServiceInstanceType,
			event.State,
			event.ServicePlanName)
	}

	// Get detailed information for the first event
	if len(events.Resources) > 0 {
		log.Println("\n   Getting detailed service event information...")

		event, err := client.ServiceUsageEvents().Get(ctx, events.Resources[0].GUID)
		if err != nil {
			log.Printf("   Failed to get service usage event: %v", err)

			return
		}

		log.Printf("   Service Event Details:\n")
		log.Printf("     GUID: %s\n", event.GUID)
		log.Printf("     Service Instance: %s (%s)\n", event.ServiceInstanceName, event.ServiceInstanceGUID)
		log.Printf("     Service Type: %s\n", event.ServiceInstanceType)
		log.Printf("     Service Offering: %s\n", event.ServiceOfferingName)
		log.Printf("     Service Plan: %s\n", event.ServicePlanName)
		log.Printf("     Service Broker: %s\n", event.ServiceBrokerName)
		log.Printf("     Space: %s (%s)\n", event.SpaceName, event.SpaceGUID)
		log.Printf("     Organization: %s (%s)\n", event.OrganizationName, event.OrganizationGUID)
	}
}

func demonstrateAuditEvents(ctx context.Context, client capi.Client) {
	events := listRecentAuditEvents(ctx, client)
	if events == nil {
		return
	}

	printAuditEventsSummary(events)
	showDetailedAuditEvent(ctx, client, events)
	filterAuditEventsByType(ctx, client)
}

func listRecentAuditEvents(ctx context.Context, client capi.Client) *capi.ListResponse[capi.AuditEvent] {
	log.Println("   Listing recent audit events...")

	params := capi.NewQueryParams()
	params.WithPerPage(constants.DefaultPageSize)

	events, err := client.AuditEvents().List(ctx, params)
	if err != nil {
		log.Printf("   Failed to list audit events: %v", err)

		return nil
	}

	return events
}

func printAuditEventsSummary(events *capi.ListResponse[capi.AuditEvent]) {
	log.Printf("   Found %d audit events\n", len(events.Resources))

	for i, event := range events.Resources {
		if i >= constants.MaxDemoItems { // Show only first 5 for demo
			break
		}

		log.Printf("     - Type: %s, Actor: %s, Target: %s (%s)\n",
			event.Type,
			event.Actor.Name,
			event.Target.Name,
			event.Target.Type)
	}
}

func showDetailedAuditEvent(ctx context.Context, client capi.Client, events *capi.ListResponse[capi.AuditEvent]) {
	if len(events.Resources) == 0 {
		return
	}

	log.Println("\n   Getting detailed audit event information...")

	event, err := client.AuditEvents().Get(ctx, events.Resources[0].GUID)
	if err != nil {
		log.Printf("   Failed to get audit event: %v", err)

		return
	}

	printAuditEventDetails(event)
}

func printAuditEventDetails(event *capi.AuditEvent) {
	log.Printf("   Audit Event Details:\n")
	log.Printf("     GUID: %s\n", event.GUID)
	log.Printf("     Type: %s\n", event.Type)
	log.Printf("     Actor: %s (%s) - %s\n", event.Actor.Name, event.Actor.Type, event.Actor.GUID)
	log.Printf("     Target: %s (%s) - %s\n", event.Target.Name, event.Target.Type, event.Target.GUID)
	log.Printf("     Created: %s\n", event.CreatedAt.Format(time.RFC3339))

	printAuditEventContext(event)
	printAuditEventData(event)
}

func printAuditEventContext(event *capi.AuditEvent) {
	if event.Space != nil {
		log.Printf("     Space: %s (%s)\n", event.Space.Name, event.Space.GUID)
	}

	if event.Organization != nil {
		log.Printf("     Organization: %s (%s)\n", event.Organization.Name, event.Organization.GUID)
	}
}

func printAuditEventData(event *capi.AuditEvent) {
	if len(event.Data) == 0 {
		return
	}

	log.Printf("     Event Data:\n")

	for key, value := range event.Data {
		log.Printf("       %s: %v\n", key, value)
	}
}

func filterAuditEventsByType(ctx context.Context, client capi.Client) {
	log.Println("\n   Filtering audit events by type (app events)...")

	appEventParams := buildAppEventParams()

	appEvents, err := client.AuditEvents().List(ctx, appEventParams)
	if err != nil {
		log.Printf("   Failed to filter audit events: %v", err)

		return
	}

	printFilteredAuditEvents(appEvents)
}

func buildAppEventParams() *capi.QueryParams {
	params := capi.NewQueryParams()
	params.WithFilter("types", "audit.app.create,audit.app.update,audit.app.delete")
	params.WithPerPage(constants.SmallPageSize)

	return params
}

func printFilteredAuditEvents(appEvents *capi.ListResponse[capi.AuditEvent]) {
	log.Printf("   Found %d app-related audit events\n", len(appEvents.Resources))

	for _, event := range appEvents.Resources {
		log.Printf("     - %s: %s\n", event.Type, event.Target.Name)
	}
}

func demonstrateEnvironmentVariableGroups(ctx context.Context, client capi.Client) {
	// Get running environment variables
	log.Println("   Getting running environment variables...")

	runningEnvVars, err := client.EnvironmentVariableGroups().Get(ctx, "running")
	if err != nil {
		log.Printf("   Failed to get running environment variables: %v", err)

		return
	}

	log.Printf("   Running Environment Variables (%d variables):\n", len(runningEnvVars.Var))

	for key, value := range runningEnvVars.Var {
		log.Printf("     %s=%v\n", key, value)
	}

	// Get staging environment variables
	log.Println("\n   Getting staging environment variables...")

	stagingEnvVars, err := client.EnvironmentVariableGroups().Get(ctx, "staging")
	if err != nil {
		log.Printf("   Failed to get staging environment variables: %v", err)

		return
	}

	log.Printf("   Staging Environment Variables (%d variables):\n", len(stagingEnvVars.Var))

	for key, value := range stagingEnvVars.Var {
		log.Printf("     %s=%v\n", key, value)
	}

	// Demonstrate updating environment variables (commented out to avoid affecting real environment)
	log.Println("\n   Environment variable update example (commented out for safety):")
	log.Println("   // Update running environment variables")
	log.Println("   // newVars := map[string]any{")
	log.Println("   //     \"DEMO_LOG_LEVEL\": \"debug\",")
	log.Println("   //     \"DEMO_FEATURE_FLAG\": true,")
	log.Println("   // }")
	log.Println("   // runningEnvVars, err := client.EnvironmentVariableGroups().Update(ctx, \"running\", newVars)")

	/*
		// Uncomment this section if you want to actually update environment variables
		// WARNING: This will affect all applications in the CF deployment

		newVars := map[string]any{
			"DEMO_LOG_LEVEL":    "debug",
			"DEMO_FEATURE_FLAG": true,
			"DEMO_TIMESTAMP":    time.Now().Format(time.RFC3339),
		}

		fmt.Println("   Updating running environment variables...")
		updatedRunningEnvVars, err := client.EnvironmentVariableGroups().Update(ctx, "running", newVars)
		if err != nil {
			log.Printf("   Failed to update running environment variables: %v", err)
			return
		}

		fmt.Printf("   Updated running environment variables (%d variables)\n", len(updatedRunningEnvVars.Var))

		// Restore original environment variables
		fmt.Println("   Restoring original environment variables...")
		_, err = client.EnvironmentVariableGroups().Update(ctx, "running", runningEnvVars.Var)
		if err != nil {
			log.Printf("   Failed to restore running environment variables: %v", err)
			return
		}

		fmt.Println("   Restored original environment variables")
	*/
}
