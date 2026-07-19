package main

import (
	"context"
	"fmt"
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

	log.Println("=== Cloud Foundry Application Lifecycle Management Examples ===")
	log.Println()

	// Revisions
	log.Println("1. Application Revisions")
	demonstrateRevisions(ctx, client)

	// Sidecars
	log.Println("\n2. Application Sidecars")
	demonstrateSidecars(ctx, client)

	// Resource Matches
	log.Println("\n3. Resource Matches")
	demonstrateResourceMatches(ctx, client)
}

func demonstrateRevisions(ctx context.Context, client capi.Client) {
	// First, find an application to work with
	log.Println("   Finding an application...")

	apps, err := client.Apps().List(ctx, capi.NewQueryParams().WithPerPage(1))
	if err != nil {
		log.Printf("   Failed to list applications: %v", err)

		return
	}

	if len(apps.Resources) == 0 {
		log.Println("   No applications found. Skipping revision demo.")

		return
	}

	app := apps.Resources[0]
	log.Printf("   Using application: %s (%s)\n", app.Name, app.GUID)

	// List revisions for the application
	log.Println("\n   Listing revisions for application...")

	params := capi.NewQueryParams()
	params.WithFilter("app_guids", app.GUID)
	params.WithPerPage(constants.SmallPageSize)

	// Note: We would need to implement a revisions list method for apps
	// For now, we'll demonstrate individual revision operations

	// If we had a revision GUID, we could demonstrate:
	log.Println("\n   Revision operations example (requires revision GUID):")
	log.Println("   // Get revision details")
	log.Println("   // revision, err := client.Revisions().Get(ctx, \"revision-guid\")")
	log.Printf("   // log.Printf(\"Version: %%d, Deployable: %%t\", revision.Version, revision.Deployable)\n")

	log.Println("   // Get revision environment variables")
	log.Println("   // envVars, err := client.Revisions().GetEnvironmentVariables(ctx, \"revision-guid\")")
	log.Println("   // for key, value := range envVars {")
	log.Printf("   //     log.Printf(\"%%s=%%v\", key, value)\n")
	log.Println("   // }")

	log.Println("   // Update revision metadata")
	log.Println("   // updateReq := &capi.RevisionUpdateRequest{")
	log.Println("   //     Metadata: &capi.Metadata{")
	log.Println("   //         Labels: capi.StringMap(map[string]string{")
	log.Println("   //             \"version\": \"1.2.0\",")
	log.Println("   //             \"team\":    \"backend\",")
	log.Println("   //         }),")
	log.Println("   //     },")
	log.Println("   // }")
	log.Println("   // revision, err := client.Revisions().Update(ctx, \"revision-guid\", updateReq)")
}

func demonstrateSidecars(ctx context.Context, client capi.Client) {
	app := findApplication(ctx, client)
	if app == nil {
		return
	}

	process := findProcess(ctx, client, app)
	if process == nil {
		return
	}

	sidecars := listSidecars(ctx, client, process)
	if sidecars == nil {
		return
	}

	printSidecarsList(sidecars)
	demonstrateSidecarOperations(ctx, client, sidecars)
}

func findApplication(ctx context.Context, client capi.Client) *capi.App {
	log.Println("   Finding an application process...")

	apps, err := client.Apps().List(ctx, capi.NewQueryParams().WithPerPage(1))
	if err != nil {
		log.Printf("   Failed to list applications: %v", err)

		return nil
	}

	if len(apps.Resources) == 0 {
		log.Println("   No applications found. Skipping sidecar demo.")

		return nil
	}

	app := &apps.Resources[0]
	log.Printf("   Using application: %s (%s)\n", app.Name, app.GUID)

	return app
}

func findProcess(ctx context.Context, client capi.Client, app *capi.App) *capi.Process {
	processes, err := client.Processes().List(ctx, capi.NewQueryParams().WithFilter("app_guids", app.GUID))
	if err != nil {
		log.Printf("   Failed to list processes: %v", err)

		return nil
	}

	if len(processes.Resources) == 0 {
		log.Println("   No processes found for application. Skipping sidecar demo.")

		return nil
	}

	process := &processes.Resources[0]
	log.Printf("   Using process: %s (%s)\n", process.Type, process.GUID)

	return process
}

func listSidecars(ctx context.Context, client capi.Client, process *capi.Process) *capi.ListResponse[capi.Sidecar] {
	log.Println("\n   Listing sidecars for process...")

	sidecars, err := client.Sidecars().ListForProcess(ctx, process.GUID, capi.NewQueryParams().WithPerPage(constants.DefaultPageSize))
	if err != nil {
		log.Printf("   Failed to list sidecars: %v", err)

		return nil
	}

	return sidecars
}

func printSidecarsList(sidecars *capi.ListResponse[capi.Sidecar]) {
	log.Printf("   Found %d sidecars for process\n", len(sidecars.Resources))

	for _, sidecar := range sidecars.Resources {
		memoryStr := formatMemorySize(sidecar.MemoryInMB)
		log.Printf("     - %s: %s (Memory: %s, Types: %v)\n",
			sidecar.Name, sidecar.Command, memoryStr, sidecar.ProcessTypes)
	}
}

func formatMemorySize(memoryInMB *int) string {
	if memoryInMB == nil {
		return "default"
	}

	return fmt.Sprintf("%d MB", *memoryInMB)
}

func demonstrateSidecarOperations(ctx context.Context, client capi.Client, sidecars *capi.ListResponse[capi.Sidecar]) {
	if len(sidecars.Resources) > 0 {
		showSidecarDetails(ctx, client, sidecars.Resources[0])
	} else {
		showSidecarExamples()
	}
}

func showSidecarDetails(ctx context.Context, client capi.Client, sidecar capi.Sidecar) {
	log.Printf("\n   Getting detailed sidecar information for: %s\n", sidecar.Name)

	detailedSidecar, err := client.Sidecars().Get(ctx, sidecar.GUID)
	if err != nil {
		log.Printf("   Failed to get sidecar: %v", err)

		return
	}

	printSidecarDetails(detailedSidecar)
}

func printSidecarDetails(sidecar *capi.Sidecar) {
	log.Printf("   Sidecar Details:\n")
	log.Printf("     Name: %s\n", sidecar.Name)
	log.Printf("     GUID: %s\n", sidecar.GUID)
	log.Printf("     Command: %s\n", sidecar.Command)
	log.Printf("     Process Types: %v\n", sidecar.ProcessTypes)
	log.Printf("     Origin: %s\n", sidecar.Origin)
	log.Printf("     Created: %s\n", sidecar.CreatedAt.Format(time.RFC3339))

	if sidecar.MemoryInMB != nil {
		log.Printf("     Memory: %d MB\n", *sidecar.MemoryInMB)
	}
}

func showSidecarExamples() {
	log.Println("\n   No sidecars found for demonstration.")
	log.Println("   Sidecar operations example:")
	printSidecarOperationExamples()
}

func printSidecarOperationExamples() {
	log.Println("   // Get sidecar")
	log.Println("   // sidecar, err := client.Sidecars().Get(ctx, \"sidecar-guid\")")
	log.Println("")
	log.Println("   // Update sidecar")
	log.Println("   // newName := \"updated-sidecar\"")
	log.Println("   // newCommand := \"./updated-command\"")
	log.Println("   // newMemory := 256")
	log.Println("   // updateReq := &capi.SidecarUpdateRequest{")
	log.Println("   //     Name:         &newName,")
	log.Println("   //     Command:      &newCommand,")
	log.Println("   //     ProcessTypes: []string{\"web\", \"worker\"},")
	log.Println("   //     MemoryInMB:   &newMemory,")
	log.Println("   // }")
	log.Println("   // sidecar, err := client.Sidecars().Update(ctx, \"sidecar-guid\", updateReq)")
}

func demonstrateResourceMatches(ctx context.Context, client capi.Client) {
	log.Println("   Demonstrating resource matches...")

	// Create example resource list
	resources := []capi.ResourceMatch{
		{
			Path: "app.js",
			SHA1: "da39a3ee5e6b4b0d3255bfef95601890afd80709", // Empty file SHA1
			Size: constants.MediumMemorySize,
			Mode: "0644",
		},
		{
			Path: "package.json",
			SHA1: "356a192b7913b04c54574d18c28d46e6395428ab", // "1" SHA1
			Size: constants.SmallMemorySize,
			Mode: "0644",
		},
		{
			Path: "server.js",
			SHA1: "da4b9237bacccdf19c0760cab7aec4a8359010b0", // "hello" SHA1
			Size: constants.LargeMemorySize,
			Mode: "0644",
		},
	}

	createReq := &capi.ResourceMatchesRequest{
		Resources: resources,
	}

	log.Printf("   Checking %d resources for matches...\n", len(resources))

	for _, resource := range resources {
		log.Printf("     - %s (SHA1: %s, Size: %d bytes)\n",
			resource.Path, resource.SHA1, resource.Size)
	}

	// Create resource matches request
	matches, err := client.ResourceMatches().Create(ctx, createReq)
	if err != nil {
		log.Printf("   Failed to create resource matches: %v", err)

		return
	}

	// Calculate which resources need to be uploaded
	matchedCount := len(matches.Resources)
	totalCount := len(resources)
	uploadCount := totalCount - matchedCount

	log.Printf("\n   Resource Match Results:\n")
	log.Printf("     Total resources: %d\n", totalCount)
	log.Printf("     Matched resources: %d\n", matchedCount)
	log.Printf("     Resources to upload: %d\n", uploadCount)

	if matchedCount > 0 {
		log.Printf("   Matched resources:\n")

		for _, match := range matches.Resources {
			log.Printf("     - %s (already exists on platform)\n", match.Path)
		}
	}

	// Show which resources would need to be uploaded
	if uploadCount > 0 {
		log.Printf("   Resources that need to be uploaded:\n")

		for _, resource := range resources {
			found := false

			for _, match := range matches.Resources {
				if resource.SHA1 == match.SHA1 {
					found = true

					break
				}
			}

			if !found {
				log.Printf("     - %s (new file)\n", resource.Path)
			}
		}
	}

	log.Printf("\n   This optimization could save %.1f%% of upload bandwidth!\n",
		float64(matchedCount)/float64(totalCount)*constants.PercentageMultiplier)
}
