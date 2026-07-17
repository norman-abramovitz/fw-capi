package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/fivetwenty-io/capi/v3/pkg/cfclient"
)

func main() {
	client, err := createClient()
	if err != nil {
		log.Fatalf("Failed to create CF client: %v", err)
	}

	ctx := context.Background()
	spaceGUID := "your-space-guid" // Replace with actual space GUID

	app := runAppLifecycleExamples(client, ctx, spaceGUID)
	cleanup(client, ctx, app)
}

func createClient() (capi.Client, error) {
	ctx := context.Background()

	client, err := cfclient.NewWithPassword(ctx,
		"https://api.your-cf-domain.com",
		"your-username",
		"your-password",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CF client: %w", err)
	}

	return client, nil
}

func runAppLifecycleExamples(client capi.Client, ctx context.Context, spaceGUID string) *capi.App {
	app := createApplicationExample(client, ctx, spaceGUID)
	getApplicationDetailsExample(client, ctx, app)
	updateApplicationExample(client, ctx, app)
	processes := listApplicationProcessesExample(client, ctx, app)
	scaleApplicationExample(client, ctx, processes)
	manageEnvironmentVariablesExample(client, ctx, app)
	getApplicationStatsExample(client, ctx, processes)
	startStopApplicationExample(client, ctx, app)

	return app
}

func createApplicationExample(client capi.Client, ctx context.Context, spaceGUID string) *capi.App {
	log.Println("=== Creating Application ===")

	createAppReq := buildAppCreateRequest(spaceGUID)

	app, err := client.Apps().Create(ctx, createAppReq)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	log.Printf("Created application: %s (GUID: %s)\n", app.Name, app.GUID)
	log.Println()

	return app
}

func buildAppCreateRequest(spaceGUID string) *capi.AppCreateRequest {
	return &capi.AppCreateRequest{
		Name: "example-app",
		Relationships: capi.AppRelationships{
			Space: capi.Relationship{
				Data: &capi.RelationshipData{GUID: spaceGUID},
			},
		},
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				"team":        "platform",
				"environment": "demo",
			},
			Annotations: map[string]string{
				"created-by": "capi-client-example",
			},
		},
	}
}

func getApplicationDetailsExample(client capi.Client, ctx context.Context, app *capi.App) {
	log.Println("=== Getting Application Details ===")

	updatedApp, err := client.Apps().Get(ctx, app.GUID)
	if err != nil {
		log.Fatalf("Failed to get application: %v", err)
	}

	printApplicationDetails(updatedApp)
	log.Println()
}

func printApplicationDetails(app *capi.App) {
	log.Printf("Application Details:\n")
	log.Printf("  Name: %s\n", app.Name)
	log.Printf("  GUID: %s\n", app.GUID)
	log.Printf("  State: %s\n", app.State)
	log.Printf("  Created: %s\n", app.CreatedAt.Format(time.RFC3339))
	log.Printf("  Updated: %s\n", app.UpdatedAt.Format(time.RFC3339))

	printAppMetadata(app.Metadata)
}

func printAppMetadata(metadata *capi.Metadata) {
	if metadata == nil {
		return
	}

	if len(metadata.Labels) > 0 {
		log.Println("  Labels:")

		for key, value := range metadata.Labels {
			log.Printf("    %s: %s\n", key, value)
		}
	}

	if len(metadata.Annotations) > 0 {
		log.Println("  Annotations:")

		for key, value := range metadata.Annotations {
			log.Printf("    %s: %s\n", key, value)
		}
	}
}

func updateApplicationExample(client capi.Client, ctx context.Context, app *capi.App) {
	log.Println("=== Updating Application ===")

	updateReq := buildAppUpdateRequest()

	updatedApp, err := client.Apps().Update(ctx, app.GUID, updateReq)
	if err != nil {
		log.Fatalf("Failed to update application: %v", err)
	}

	log.Printf("Updated application name to: %s\n", updatedApp.Name)
	log.Println()

	*app = *updatedApp // Update the app reference
}

func buildAppUpdateRequest() *capi.AppUpdateRequest {
	newName := "example-app-updated"

	return &capi.AppUpdateRequest{
		Name: &newName,
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				"version": "1.0.0",
				"updated": "true",
			},
			Annotations: map[string]string{
				"updated-by": "capi-client-example",
			},
		},
	}
}

func listApplicationProcessesExample(client capi.Client, ctx context.Context, app *capi.App) *capi.ProcessList {
	log.Println("=== Listing Application Processes ===")

	processParams := capi.NewQueryParams().WithFilter("app_guids", app.GUID)

	processes, err := client.Processes().List(ctx, processParams)
	if err != nil {
		log.Fatalf("Failed to list processes: %v", err)
	}

	printProcesses(processes)
	log.Println()

	return processes
}

func printProcesses(processes *capi.ProcessList) {
	log.Printf("Found %d processes:\n", len(processes.Resources))

	for _, process := range processes.Resources {
		log.Printf("  - Type: %s, Instances: %d, Memory: %d MB, Disk: %d MB\n",
			process.Type, process.Instances, process.MemoryInMB, process.DiskInMB)
	}
}

func scaleApplicationExample(client capi.Client, ctx context.Context, processes *capi.ProcessList) {
	if len(processes.Resources) == 0 {
		return
	}

	log.Println("=== Scaling Application ===")

	webProcess := processes.Resources[0] // Usually the 'web' process

	scaleReq := buildScaleRequest()

	// Scale is async — returns a job whose GUID we log. Callers that need
	// terminal state should poll Jobs().Get.
	job, err := client.Processes().Scale(ctx, webProcess.GUID, scaleReq)
	if err != nil {
		log.Fatalf("Failed to scale process: %v", err)
	}

	log.Printf("Queued scale of %s process (job %s)\n", webProcess.Type, job.GUID)
}

func buildScaleRequest() *capi.ProcessScaleRequest {
	instances := 2
	memory := 512
	disk := 1024

	return &capi.ProcessScaleRequest{
		Instances:  &instances,
		MemoryInMB: &memory,
		DiskInMB:   &disk,
	}
}

func manageEnvironmentVariablesExample(client capi.Client, ctx context.Context, app *capi.App) {
	getEnvironmentVariablesExample(client, ctx, app)
	setEnvironmentVariablesExample(client, ctx, app)
}

func getEnvironmentVariablesExample(client capi.Client, ctx context.Context, app *capi.App) {
	log.Println("=== Application Environment Variables ===")

	envVars, err := client.Apps().GetEnv(ctx, app.GUID)
	if err != nil {
		log.Fatalf("Failed to get environment variables: %v", err)
	}

	printEnvironmentVariables(envVars)
	log.Println()
}

func printEnvironmentVariables(envVars *capi.AppEnv) {
	if len(envVars.SystemEnvJSON) > 0 {
		log.Println("System Environment Variables:")

		for key, value := range envVars.SystemEnvJSON {
			log.Printf("  %s: %v\n", key, value)
		}
	}

	if len(envVars.ApplicationEnvJSON) > 0 {
		log.Println("Application Environment Variables:")

		for key, value := range envVars.ApplicationEnvJSON {
			log.Printf("  %s: %v\n", key, value)
		}
	}
}

func setEnvironmentVariablesExample(client capi.Client, ctx context.Context, app *capi.App) {
	log.Println("=== Setting Environment Variables ===")

	newEnvVars := map[string]any{
		"EXAMPLE_VAR": "example-value",
		"DEBUG":       "true",
	}

	_, err := client.Apps().UpdateEnvVars(ctx, app.GUID, newEnvVars)
	if err != nil {
		log.Fatalf("Failed to set environment variables: %v", err)
	}

	log.Println("Environment variables updated successfully")
	log.Println()
}

func getApplicationStatsExample(client capi.Client, ctx context.Context, processes *capi.ProcessList) {
	log.Println("=== Application Stats ===")

	stats, err := getProcessStats(client, ctx, processes)
	if err != nil {
		log.Printf("Failed to get stats (app may not be running): %v", err)
	} else {
		printApplicationStats(stats)
	}

	log.Println()
}

func getProcessStats(client capi.Client, ctx context.Context, processes *capi.ProcessList) (*capi.ProcessStats, error) {
	if len(processes.Resources) == 0 {
		return nil, capi.ErrNoProcessesFound
	}

	stats, err := client.Processes().GetStats(ctx, processes.Resources[0].GUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get process stats: %w", err)
	}

	return stats, nil
}

func printApplicationStats(stats *capi.ProcessStats) {
	log.Printf("Application Stats:\n")

	for _, stat := range stats.Resources {
		log.Printf("  Instance %d:\n", stat.Index)
		log.Printf("    State: %s\n", stat.State)

		if stat.Usage != nil {
			log.Printf("    CPU: %.2f%%\n", stat.Usage.CPU*constants.PercentageMultiplier)
			log.Printf("    Memory: %d bytes\n", stat.Usage.Mem)
			log.Printf("    Disk: %d bytes\n", stat.Usage.Disk)
		}
	}
}

func startStopApplicationExample(client capi.Client, ctx context.Context, app *capi.App) {
	startApplicationExample(client, ctx, app)
	time.Sleep(constants.DefaultPollInterval)
	stopApplicationExample(client, ctx, app)
}

func startApplicationExample(client capi.Client, ctx context.Context, app *capi.App) {
	log.Println("=== Starting Application ===")

	// CF v3 /actions/start is async — returns a job whose GUID we can
	// poll for terminal state. For the example we just log it and fall
	// through to the next step.
	job, err := client.Apps().Start(ctx, app.GUID)
	if err != nil {
		log.Printf("Failed to start application: %v", err)
	} else {
		log.Printf("Queued start of application (job %s)\n", job.GUID)
	}
}

func stopApplicationExample(client capi.Client, ctx context.Context, app *capi.App) {
	log.Println("=== Stopping Application ===")

	job, err := client.Apps().Stop(ctx, app.GUID)
	if err != nil {
		log.Printf("Failed to stop application: %v", err)
	} else {
		log.Printf("Queued stop of application (job %s)\n", job.GUID)
	}

	log.Println()
}

func cleanup(client capi.Client, ctx context.Context, app *capi.App) {
	log.Println("=== Deleting Application ===")

	job, err := client.Apps().Delete(ctx, app.GUID)
	if err != nil {
		log.Fatalf("Failed to delete application: %v", err)
	}

	log.Printf("Application delete queued (job %s) — poll Jobs().Get to observe completion.", job.GUID)
}
