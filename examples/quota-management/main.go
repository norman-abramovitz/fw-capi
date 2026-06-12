package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

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

	log.Println("=== Cloud Foundry Quota Management Examples ===")
	log.Println()

	// Organization Quota Management
	log.Println("1. Organization Quota Management")
	demonstrateOrgQuotaManagement(ctx, client)

	// Space Quota Management
	log.Println("\n2. Space Quota Management")
	demonstrateSpaceQuotaManagement(ctx, client)
}

func demonstrateOrgQuotaManagement(ctx context.Context, client capi.Client) {
	listExistingOrgQuotas(ctx, client)

	quota := createDemoOrgQuota(ctx, client)
	if quota == nil {
		return
	}

	showOrgQuotaDetails(ctx, client, quota)

	updatedQuota := updateOrgQuota(ctx, client, quota)
	if updatedQuota != nil {
		cleanupOrgQuota(ctx, client, updatedQuota)
	}
}

func listExistingOrgQuotas(ctx context.Context, client capi.Client) {
	log.Println("   Listing existing organization quotas...")

	quotas, err := client.OrganizationQuotas().List(ctx, nil)
	if err != nil {
		log.Printf("   Failed to list organization quotas: %v", err)

		return
	}

	printOrgQuotasList(quotas)
}

func printOrgQuotasList(quotas *capi.ListResponse[capi.OrganizationQuota]) {
	log.Printf("   Found %d organization quotas\n", len(quotas.Resources))

	for _, quota := range quotas.Resources {
		memoryStr := formatMemoryLimit(quota.Apps)
		servicesStr := formatServicesLimit(quota.Services)
		log.Printf("     - %s: Memory=%s, Services=%s\n", quota.Name, memoryStr, servicesStr)
	}
}

func formatMemoryLimit(apps *capi.OrganizationQuotaApps) string {
	if apps != nil && apps.TotalMemoryInMB != nil {
		return strconv.Itoa(*apps.TotalMemoryInMB) + " MB"
	}

	return "unlimited"
}

func formatServicesLimit(services *capi.OrganizationQuotaServices) string {
	if services != nil && services.TotalServiceInstances != nil {
		return strconv.Itoa(*services.TotalServiceInstances)
	}

	return "unlimited"
}

func createDemoOrgQuota(ctx context.Context, client capi.Client) *capi.OrganizationQuota {
	log.Println("\n   Creating new organization quota...")

	createReq := buildOrgQuotaCreateRequest()

	quota, err := client.OrganizationQuotas().Create(ctx, createReq)
	if err != nil {
		log.Printf("   Failed to create organization quota: %v", err)

		return nil
	}

	log.Printf("   Created organization quota: %s (GUID: %s)\n", quota.Name, quota.GUID)

	return quota
}

func buildOrgQuotaCreateRequest() *capi.OrganizationQuotaCreateRequest {
	totalMemory := 2048
	instanceMemory := 512
	totalInstances := 10
	totalAppTasks := 5
	totalServices := 10
	totalServiceKeys := 20
	totalRoutes := 50
	totalReservedPorts := 5
	totalDomains := 5
	paidServices := true

	return &capi.OrganizationQuotaCreateRequest{
		Name: "demo-org-quota",
		Apps: &capi.OrganizationQuotaApps{
			TotalMemoryInMB:      &totalMemory,
			PerProcessMemoryInMB: &instanceMemory,
			TotalInstances:       &totalInstances,
			PerAppTasks:          &totalAppTasks,
		},
		Services: &capi.OrganizationQuotaServices{
			PaidServicesAllowed:   &paidServices,
			TotalServiceInstances: &totalServices,
			TotalServiceKeys:      &totalServiceKeys,
		},
		Routes: &capi.OrganizationQuotaRoutes{
			TotalRoutes:        &totalRoutes,
			TotalReservedPorts: &totalReservedPorts,
		},
		Domains: &capi.OrganizationQuotaDomains{
			TotalDomains: &totalDomains,
		},
	}
}

func showOrgQuotaDetails(ctx context.Context, client capi.Client, quota *capi.OrganizationQuota) {
	log.Println("\n   Getting quota details...")

	detailedQuota, err := client.OrganizationQuotas().Get(ctx, quota.GUID)
	if err != nil {
		log.Printf("   Failed to get organization quota: %v", err)

		return
	}

	printOrgQuotaDetails(detailedQuota)
}

func printOrgQuotaDetails(quota *capi.OrganizationQuota) {
	log.Printf("   Quota Details:\n")
	log.Printf("     Name: %s\n", quota.Name)
	log.Printf("     GUID: %s\n", quota.GUID)

	if quota.Apps != nil {
		if quota.Apps.TotalMemoryInMB != nil {
			log.Printf("     Total Memory: %d MB\n", *quota.Apps.TotalMemoryInMB)
		}

		if quota.Apps.TotalInstances != nil {
			log.Printf("     Total Instances: %d\n", *quota.Apps.TotalInstances)
		}
	}
}

func updateOrgQuota(ctx context.Context, client capi.Client, quota *capi.OrganizationQuota) *capi.OrganizationQuota {
	log.Println("\n   Updating quota...")

	updateReq := buildOrgQuotaUpdateRequest()

	updatedQuota, err := client.OrganizationQuotas().Update(ctx, quota.GUID, updateReq)
	if err != nil {
		log.Printf("   Failed to update organization quota: %v", err)

		return nil
	}

	log.Printf("   Updated quota: %s (Memory: %d MB)\n", updatedQuota.Name, *updatedQuota.Apps.TotalMemoryInMB)

	return updatedQuota
}

func buildOrgQuotaUpdateRequest() *capi.OrganizationQuotaUpdateRequest {
	newMemory := 4096
	newName := "demo-org-quota-updated"

	return &capi.OrganizationQuotaUpdateRequest{
		Name: &newName,
		Apps: &capi.OrganizationQuotaApps{
			TotalMemoryInMB: &newMemory,
		},
	}
}

func cleanupOrgQuota(ctx context.Context, client capi.Client, quota *capi.OrganizationQuota) {
	log.Println("\n   Cleaning up - deleting demo quota...")

	job, err := client.OrganizationQuotas().Delete(ctx, quota.GUID)
	if err != nil {
		log.Printf("   Failed to delete organization quota: %v", err)

		return
	}

	log.Printf("   Deleted quota: %s (job: %s)\n", quota.Name, job.GUID)
}

func findOrganizationForSpaceQuota(ctx context.Context, client capi.Client) (string, string, error) {
	orgs, err := client.Organizations().List(ctx, capi.NewQueryParams().WithPerPage(1))
	if err != nil {
		return "", "", fmt.Errorf("failed to list organizations: %w", err)
	}

	if len(orgs.Resources) == 0 {
		return "", "", constants.ErrNoOrganizationsFound
	}

	return orgs.Resources[0].GUID, orgs.Resources[0].Name, nil
}

func listSpaceQuotasForOrg(ctx context.Context, client capi.Client, orgGUID string) error {
	log.Println("\n   Listing space quotas for organization...")

	params := capi.NewQueryParams()
	params.WithFilter("organization_guids", orgGUID)

	spaceQuotas, err := client.SpaceQuotas().List(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list space quotas: %w", err)
	}

	log.Printf("   Found %d space quotas in organization\n", len(spaceQuotas.Resources))

	return nil
}

func createDemoSpaceQuota(ctx context.Context, client capi.Client, orgGUID string) (*capi.SpaceQuotaV3, error) {
	log.Println("\n   Creating new space quota...")

	totalMemory := 1024
	totalInstances := 5
	totalRoutes := 20
	logRateLimit := 1000

	createReq := &capi.SpaceQuotaV3CreateRequest{
		Name: "demo-space-quota",
		Relationships: capi.SpaceQuotaRelationships{
			Organization: capi.Relationship{
				Data: &capi.RelationshipData{GUID: orgGUID},
			},
			Spaces: capi.ToManyRelationship{
				Data: []capi.RelationshipData{},
			},
		},
		Apps: &capi.SpaceQuotaApps{
			TotalMemoryInMB:              &totalMemory,
			TotalInstances:               &totalInstances,
			LogRateLimitInBytesPerSecond: &logRateLimit,
		},
		Routes: &capi.SpaceQuotaRoutes{
			TotalRoutes: &totalRoutes,
		},
	}

	spaceQuota, err := client.SpaceQuotas().Create(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create space quota: %w", err)
	}

	log.Printf("   Created space quota: %s (GUID: %s)\n", spaceQuota.Name, spaceQuota.GUID)

	return spaceQuota, nil
}

func showSpaceQuotaDetails(ctx context.Context, client capi.Client, spaceQuotaGUID string) error {
	log.Println("\n   Getting space quota details...")

	spaceQuota, err := client.SpaceQuotas().Get(ctx, spaceQuotaGUID)
	if err != nil {
		return fmt.Errorf("failed to get space quota: %w", err)
	}

	log.Printf("   Space Quota Details:\n")
	log.Printf("     Name: %s\n", spaceQuota.Name)

	if spaceQuota.Apps != nil {
		if spaceQuota.Apps.TotalMemoryInMB != nil {
			log.Printf("     Total Memory: %d MB\n", *spaceQuota.Apps.TotalMemoryInMB)
		}

		if spaceQuota.Apps.TotalInstances != nil {
			log.Printf("     Total Instances: %d\n", *spaceQuota.Apps.TotalInstances)
		}
	}

	return nil
}

func cleanupSpaceQuota(ctx context.Context, client capi.Client, spaceQuota *capi.SpaceQuotaV3) error {
	log.Println("\n   Cleaning up - deleting demo space quota...")

	job, err := client.SpaceQuotas().Delete(ctx, spaceQuota.GUID)
	if err != nil {
		return fmt.Errorf("failed to delete space quota: %w", err)
	}

	log.Printf("   Deleted space quota: %s (job: %s)\n", spaceQuota.Name, job.GUID)

	return nil
}

func demonstrateSpaceQuotaManagement(ctx context.Context, client capi.Client) {
	// Find an organization to create the space quota in
	log.Println("   Finding an organization...")

	orgGUID, orgName, err := findOrganizationForSpaceQuota(ctx, client)
	if err != nil {
		log.Printf("   %v", err)

		return
	}

	log.Printf("   Using organization: %s (%s)\n", orgName, orgGUID)

	// List existing space quotas for this organization
	err = listSpaceQuotasForOrg(ctx, client, orgGUID)
	if err != nil {
		log.Printf("   %v", err)

		return
	}

	// Create a new space quota
	spaceQuota, err := createDemoSpaceQuota(ctx, client, orgGUID)
	if err != nil {
		log.Printf("   %v", err)

		return
	}

	// Get and display the space quota details
	err = showSpaceQuotaDetails(ctx, client, spaceQuota.GUID)
	if err != nil {
		log.Printf("   %v", err)

		return
	}

	// Clean up - delete the demo space quota
	err = cleanupSpaceQuota(ctx, client, spaceQuota)
	if err != nil {
		log.Printf("   %v", err)

		return
	}
}
