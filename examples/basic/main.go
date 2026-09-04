package main

import (
	"context"
	"log"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/fivetwenty-io/capi/v3/pkg/cfclient"
)

func main() {
	client := createClient()
	ctx := context.Background()

	runBasicExamples(client, ctx)
}

func createClient() capi.Client {
	ctx := context.Background()

	client, err := cfclient.NewWithPassword(ctx,
		"https://api.your-cf-domain.com",
		"your-username",
		"your-password",
	)
	if err != nil {
		log.Fatalf("Failed to create CF client: %v", err)
	}

	return client
}

func runBasicExamples(client capi.Client, ctx context.Context) {
	getAPIInfoExample(client, ctx)
	orgs := listOrganizationsExample(client, ctx)
	listSpacesExample(client, ctx, orgs)
	listApplicationsExample(client, ctx)
}

func getAPIInfoExample(client capi.Client, ctx context.Context) {
	log.Println("=== API Info ===")

	info, err := client.GetInfo(ctx)
	if err != nil {
		log.Fatalf("Failed to get API info: %v", err)
	}

	printAPIInfo(info)
	log.Println()
}

func printAPIInfo(info *capi.Info) {
	log.Printf("API Version: %d\n", info.Version)
	log.Printf("API Description: %s\n", info.Description)
}

func listOrganizationsExample(client capi.Client, ctx context.Context) *capi.ListResponse[capi.Organization] {
	log.Println("=== Organizations ===")

	orgs, err := client.Organizations().List(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to list organizations: %v", err)
	}

	printOrganizations(orgs)
	log.Println()

	return orgs
}

func printOrganizations(orgs *capi.ListResponse[capi.Organization]) {
	log.Printf("Found %d organizations:\n", len(orgs.Resources))

	for _, org := range orgs.Resources {
		log.Printf("  - %s (GUID: %s)\n", org.Name, org.GUID)
		printOrgMetadata(org.Metadata)
	}
}

func printOrgMetadata(metadata *capi.Metadata) {
	if metadata == nil || len(metadata.Labels) == 0 {
		return
	}

	log.Println("    Labels:")

	for key, value := range metadata.Labels {
		log.Printf("      %s: %s\n", key, capi.StringValue(value))
	}
}

func listSpacesExample(client capi.Client, ctx context.Context, orgs *capi.ListResponse[capi.Organization]) {
	if len(orgs.Resources) == 0 {
		return
	}

	log.Println("=== Spaces ===")

	firstOrg := orgs.Resources[0]
	spaces := getSpacesForOrganization(client, ctx, firstOrg.GUID)
	printSpaces(spaces, firstOrg.Name)
	log.Println()
}

func getSpacesForOrganization(client capi.Client, ctx context.Context, orgGUID string) *capi.ListResponse[capi.Space] {
	params := capi.NewQueryParams()
	params.WithFilter("organization_guids", orgGUID)

	spaces, err := client.Spaces().List(ctx, params)
	if err != nil {
		log.Fatalf("Failed to list spaces: %v", err)
	}

	return spaces
}

func printSpaces(spaces *capi.ListResponse[capi.Space], orgName string) {
	log.Printf("Found %d spaces in organization '%s':\n", len(spaces.Resources), orgName)

	for _, space := range spaces.Resources {
		log.Printf("  - %s (GUID: %s)\n", space.Name, space.GUID)
	}
}

func listApplicationsExample(client capi.Client, ctx context.Context) {
	log.Println("=== Applications (with pagination) ===")

	params := buildAppListParams()
	apps := getApplications(client, ctx, params)
	printApplications(apps)
}

func buildAppListParams() *capi.QueryParams {
	params := capi.NewQueryParams()
	params.WithPerPage(constants.SmallPageSize) // Small page size for demonstration

	return params
}

func getApplications(client capi.Client, ctx context.Context, params *capi.QueryParams) []capi.App {
	appList, err := client.Apps().List(ctx, params)
	if err != nil {
		log.Fatalf("Failed to list applications: %v", err)
	}

	return appList.Resources
}

func printApplications(apps []capi.App) {
	log.Printf("Total applications found: %d (first page only)\n", len(apps))

	for _, app := range apps {
		log.Printf("  - %s (State: %s)\n", app.Name, app.State)
	}
}
