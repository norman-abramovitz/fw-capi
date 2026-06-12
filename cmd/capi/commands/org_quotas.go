package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewOrgQuotasCommand creates the organization quotas command group.
func NewOrgQuotasCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "org-quotas",
		Aliases: []string{"organization-quotas", "org-quota", "quotas"},
		Short:   "Manage organization quotas",
		Long:    "List, create, update, delete, and apply organization quotas",
	}

	cmd.AddCommand(newOrgQuotasListCommand())
	cmd.AddCommand(newOrgQuotasGetCommand())
	cmd.AddCommand(newOrgQuotasCreateCommand())
	cmd.AddCommand(newOrgQuotasUpdateCommand())
	cmd.AddCommand(newOrgQuotasDeleteCommand())
	cmd.AddCommand(newOrgQuotasApplyCommand())

	return cmd
}

// fetchAllOrgQuotasPages fetches all pages of organization quotas.
func fetchAllOrgQuotasPages(ctx context.Context, client capi.Client, params *capi.QueryParams, quotas *capi.ListResponse[capi.OrganizationQuota]) ([]capi.OrganizationQuota, error) {
	allQuotas := quotas.Resources
	if quotas.Pagination.TotalPages > 1 {
		for page := 2; page <= quotas.Pagination.TotalPages; page++ {
			params.Page = page

			moreQuotas, err := client.OrganizationQuotas().List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
			}

			allQuotas = append(allQuotas, moreQuotas.Resources...)
		}
	}

	return allQuotas, nil
}

// renderOrgQuotasTable renders organization quotas in table format.
func renderOrgQuotasTable(allQuotas []capi.OrganizationQuota, allPages bool, pagination capi.Pagination) error {
	if len(allQuotas) == 0 {
		_, _ = os.Stdout.WriteString("No organization quotas found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "GUID", "Total Memory", "Services", "Routes", "Created")

	for _, quota := range allQuotas {
		memoryStr := constants.Unlimited
		if quota.Apps != nil && quota.Apps.TotalMemoryInMB != nil {
			memoryStr = fmt.Sprintf("%d MB", *quota.Apps.TotalMemoryInMB)
		}

		servicesStr := constants.Unlimited
		if quota.Services != nil && quota.Services.TotalServiceInstances != nil {
			servicesStr = strconv.Itoa(*quota.Services.TotalServiceInstances)
		}

		routesStr := constants.Unlimited
		if quota.Routes != nil && quota.Routes.TotalRoutes != nil {
			routesStr = strconv.Itoa(*quota.Routes.TotalRoutes)
		}

		_ = table.Append(quota.Name, quota.GUID, memoryStr, servicesStr, routesStr,
			quota.CreatedAt.Format("2006-01-02"))
	}

	_ = table.Render()

	if !allPages && pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", pagination.TotalPages)
	}

	return nil
}

func newOrgQuotasListCommand() *cobra.Command {
	var (
		allPages bool
		perPage  int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List organization quotas",
		Long:  "List all organization quotas",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()
			params := capi.NewQueryParams()
			params.PerPage = perPage

			quotas, err := client.OrganizationQuotas().List(ctx, params)
			if err != nil {
				return fmt.Errorf("failed to list organization quotas: %w", err)
			}

			// Fetch all pages if requested
			allQuotas := quotas.Resources
			if allPages {
				allQuotas, err = fetchAllOrgQuotasPages(ctx, client, params, quotas)
				if err != nil {
					return err
				}
			}

			// Output results
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				return encoder.Encode(allQuotas)
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				return encoder.Encode(allQuotas)
			default:
				return renderOrgQuotasTable(allQuotas, allPages, quotas.Pagination)
			}
		},
	}

	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "results per page")

	return cmd
}

// findOrgQuota finds an organization quota by name or GUID.
func findOrgQuota(ctx context.Context, client capi.Client, nameOrGUID string) (*capi.OrganizationQuota, error) {
	quotaClient := client.OrganizationQuotas()

	// Try to get by GUID first
	quota, err := quotaClient.Get(ctx, nameOrGUID)
	if err == nil {
		return quota, nil
	}

	// If not found by GUID, try by name
	params := capi.NewQueryParams()
	params.WithFilter("names", nameOrGUID)

	quotas, err := quotaClient.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find organization quota: %w", err)
	}

	if len(quotas.Resources) == 0 {
		return nil, fmt.Errorf("organization quota '%s': %w", nameOrGUID, ErrOrganizationQuotaNotFound)
	}

	return &quotas.Resources[0], nil
}

// renderOrgQuotaDetails renders detailed organization quota information.
func renderOrgQuotaDetails(quota *capi.OrganizationQuota) {
	// Basic quota information
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	_ = table.Append("Name", quota.Name)
	_ = table.Append("GUID", quota.GUID)
	_ = table.Append("Created", quota.CreatedAt.Format("2006-01-02 15:04:05"))
	_ = table.Append("Updated", quota.UpdatedAt.Format("2006-01-02 15:04:05"))

	_, _ = os.Stdout.WriteString("Organization Quota Details:\n\n")

	_ = table.Render()

	// App limits
	renderOrgQuotaAppLimits(quota.Apps)

	// Service limits
	renderOrgQuotaServiceLimits(quota.Services)

	// Route limits
	renderOrgQuotaRouteLimits(quota.Routes)

	// Domain limits
	renderOrgQuotaDomainLimits(quota.Domains)
}

// renderOrgQuotaAppLimits renders app limits for organization quota.
func renderOrgQuotaAppLimits(apps *capi.OrganizationQuotaApps) {
	if apps == nil {
		return
	}

	_, _ = os.Stdout.WriteString("\nApp Limits:\n")

	appTable := tablewriter.NewWriter(os.Stdout)
	appTable.Header("Limit", "Value")

	totalMemory := constants.Unlimited
	if apps.TotalMemoryInMB != nil {
		totalMemory = fmt.Sprintf("%d MB", *apps.TotalMemoryInMB)
	}

	_ = appTable.Append("Total Memory", totalMemory)

	instanceMemory := constants.Unlimited
	if apps.TotalInstanceMemoryInMB != nil {
		instanceMemory = fmt.Sprintf("%d MB", *apps.TotalInstanceMemoryInMB)
	}

	_ = appTable.Append("Instance Memory", instanceMemory)

	totalInstances := constants.Unlimited
	if apps.TotalInstances != nil {
		totalInstances = strconv.Itoa(*apps.TotalInstances)
	}

	_ = appTable.Append("Total Instances", totalInstances)

	totalAppTasks := constants.Unlimited
	if apps.TotalAppTasks != nil {
		totalAppTasks = strconv.Itoa(*apps.TotalAppTasks)
	}

	_ = appTable.Append("Total App Tasks", totalAppTasks)

	_ = appTable.Render()
}

// renderOrgQuotaServiceLimits renders service limits for organization quota.
func renderOrgQuotaServiceLimits(services *capi.OrganizationQuotaServices) {
	if services == nil {
		return
	}

	_, _ = os.Stdout.WriteString("\nService Limits:\n")

	serviceTable := tablewriter.NewWriter(os.Stdout)
	serviceTable.Header("Limit", "Value")

	if services.PaidServicesAllowed != nil {
		paidServices := strconv.FormatBool(*services.PaidServicesAllowed)
		_ = serviceTable.Append("Paid Services Allowed", paidServices)
	}

	totalServiceInstances := constants.Unlimited
	if services.TotalServiceInstances != nil {
		totalServiceInstances = strconv.Itoa(*services.TotalServiceInstances)
	}

	_ = serviceTable.Append("Total Service Instances", totalServiceInstances)

	totalServiceKeys := constants.Unlimited
	if services.TotalServiceKeys != nil {
		totalServiceKeys = strconv.Itoa(*services.TotalServiceKeys)
	}

	_ = serviceTable.Append("Total Service Keys", totalServiceKeys)

	_ = serviceTable.Render()
}

// renderOrgQuotaRouteLimits renders route limits for organization quota.
func renderOrgQuotaRouteLimits(routes *capi.OrganizationQuotaRoutes) {
	if routes == nil {
		return
	}

	_, _ = os.Stdout.WriteString("\nRoute Limits:\n")

	routeTable := tablewriter.NewWriter(os.Stdout)
	routeTable.Header("Limit", "Value")

	totalRoutes := constants.Unlimited
	if routes.TotalRoutes != nil {
		totalRoutes = strconv.Itoa(*routes.TotalRoutes)
	}

	_ = routeTable.Append("Total Routes", totalRoutes)

	totalReservedPorts := constants.Unlimited
	if routes.TotalReservedPorts != nil {
		totalReservedPorts = strconv.Itoa(*routes.TotalReservedPorts)
	}

	_ = routeTable.Append("Total Reserved Ports", totalReservedPorts)

	_ = routeTable.Render()
}

// renderOrgQuotaDomainLimits renders domain limits for organization quota.
func renderOrgQuotaDomainLimits(domains *capi.OrganizationQuotaDomains) {
	if domains == nil {
		return
	}

	_, _ = os.Stdout.WriteString("\nDomain Limits:\n")

	domainTable := tablewriter.NewWriter(os.Stdout)
	domainTable.Header("Limit", "Value")

	totalDomains := constants.Unlimited
	if domains.TotalDomains != nil {
		totalDomains = strconv.Itoa(*domains.TotalDomains)
	}

	_ = domainTable.Append("Total Domains", totalDomains)

	_ = domainTable.Render()
}

func newOrgQuotasGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get QUOTA_NAME_OR_GUID",
		Short: "Get organization quota details",
		Long:  "Display detailed information about a specific organization quota",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find the quota
			quota, err := findOrgQuota(ctx, client, nameOrGUID)
			if err != nil {
				return err
			}

			// Output results
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				return encoder.Encode(quota)
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				return encoder.Encode(quota)
			default:
				renderOrgQuotaDetails(quota)
			}

			return nil
		},
	}
}

// orgQuotaCreateConfig holds configuration for creating org quotas.
type orgQuotaCreateConfig struct {
	name                         string
	totalMemoryInMB              int
	totalInstanceMemoryInMB      int
	totalInstances               int
	totalAppTasks                int
	paidServicesAllowed          bool
	totalServiceInstances        int
	totalServiceKeys             int
	totalRoutes                  int
	totalReservedPorts           int
	totalDomains                 int
	logRateLimitInBytesPerSecond int
}

// buildOrgQuotaCreateRequest creates the create request based on changed flags.
func buildOrgQuotaCreateRequest(cmd *cobra.Command, config *orgQuotaCreateConfig) *capi.OrganizationQuotaCreateRequest {
	createReq := &capi.OrganizationQuotaCreateRequest{
		Name: config.name,
	}

	// Build app limits if any app flags are set
	createReq.Apps = buildOrgQuotaCreateAppLimits(cmd, config)

	// Build service limits if any service flags are set
	createReq.Services = buildOrgQuotaCreateServiceLimits(cmd, config)

	// Build route limits if any route flags are set
	createReq.Routes = buildOrgQuotaCreateRouteLimits(cmd, config)

	// Build domain limits if domain flags are set
	createReq.Domains = buildOrgQuotaCreateDomainLimits(cmd, config)

	return createReq
}

// buildOrgQuotaCreateAppLimits creates app limits section for org quota create.
// AppLimitsConfig provides a common interface for extracting app limit values.

// Implement AppLimitsConfig for orgQuotaCreateConfig.
func (c *orgQuotaCreateConfig) GetTotalMemoryInMB() int         { return c.totalMemoryInMB }
func (c *orgQuotaCreateConfig) GetTotalInstanceMemoryInMB() int { return c.totalInstanceMemoryInMB }
func (c *orgQuotaCreateConfig) GetTotalInstances() int          { return c.totalInstances }
func (c *orgQuotaCreateConfig) GetTotalAppTasks() int           { return c.totalAppTasks }
func (c *orgQuotaCreateConfig) GetLogRateLimitInBytesPerSecond() int {
	return c.logRateLimitInBytesPerSecond
}

// Implement AppLimitsConfig for orgQuotaUpdateConfig.
func (c *orgQuotaUpdateConfig) GetTotalMemoryInMB() int         { return c.totalMemoryInMB }
func (c *orgQuotaUpdateConfig) GetTotalInstanceMemoryInMB() int { return c.totalInstanceMemoryInMB }
func (c *orgQuotaUpdateConfig) GetTotalInstances() int          { return c.totalInstances }
func (c *orgQuotaUpdateConfig) GetTotalAppTasks() int           { return c.totalAppTasks }
func (c *orgQuotaUpdateConfig) GetLogRateLimitInBytesPerSecond() int {
	return c.logRateLimitInBytesPerSecond
}

// buildGenericAppLimits creates app limits section if app flags are changed.
func buildGenericAppLimits(cmd *cobra.Command, config AppLimitsConfig) *capi.OrganizationQuotaApps {
	return BuildOrganizationQuotaApps(cmd, config)
}

func buildOrgQuotaCreateAppLimits(cmd *cobra.Command, config *orgQuotaCreateConfig) *capi.OrganizationQuotaApps {
	return buildGenericAppLimits(cmd, config)
}

// buildOrgQuotaCreateServiceLimits creates service limits section for org quota create.
func buildOrgQuotaCreateServiceLimits(cmd *cobra.Command, config *orgQuotaCreateConfig) *capi.OrganizationQuotaServices {
	if !cmd.Flags().Changed("paid-services") && !cmd.Flags().Changed("service-instances") &&
		!cmd.Flags().Changed("service-keys") {
		return nil
	}

	services := &capi.OrganizationQuotaServices{}

	if cmd.Flags().Changed("paid-services") {
		services.PaidServicesAllowed = &config.paidServicesAllowed
	}

	if cmd.Flags().Changed("service-instances") {
		services.TotalServiceInstances = &config.totalServiceInstances
	}

	if cmd.Flags().Changed("service-keys") {
		services.TotalServiceKeys = &config.totalServiceKeys
	}

	return services
}

// buildOrgQuotaCreateRouteLimits creates route limits section for org quota create.
func buildOrgQuotaCreateRouteLimits(cmd *cobra.Command, config *orgQuotaCreateConfig) *capi.OrganizationQuotaRoutes {
	if !cmd.Flags().Changed("routes") && !cmd.Flags().Changed("reserved-ports") {
		return nil
	}

	routes := &capi.OrganizationQuotaRoutes{}

	if cmd.Flags().Changed("routes") {
		routes.TotalRoutes = &config.totalRoutes
	}

	if cmd.Flags().Changed("reserved-ports") {
		routes.TotalReservedPorts = &config.totalReservedPorts
	}

	return routes
}

// buildOrgQuotaCreateDomainLimits creates domain limits section for org quota create.
func buildOrgQuotaCreateDomainLimits(cmd *cobra.Command, config *orgQuotaCreateConfig) *capi.OrganizationQuotaDomains {
	if !cmd.Flags().Changed("domains") {
		return nil
	}

	domains := &capi.OrganizationQuotaDomains{}
	domains.TotalDomains = &config.totalDomains

	return domains
}

func newOrgQuotasCreateCommand() *cobra.Command {
	config := &orgQuotaCreateConfig{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new organization quota",
		Long:  "Create a new Cloud Foundry organization quota",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config.name == "" {
				return ErrQuotaNameRequired
			}

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Build create request based on changed flags
			createReq := buildOrgQuotaCreateRequest(cmd, config)

			// Create organization quota
			quota, err := client.OrganizationQuotas().Create(ctx, createReq)
			if err != nil {
				return fmt.Errorf("failed to create organization quota: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully created organization quota '%s' with GUID %s\n", quota.Name, quota.GUID)

			return nil
		},
	}

	cmd.Flags().StringVarP(&config.name, "name", "n", "", "quota name (required)")
	cmd.Flags().IntVar(&config.totalMemoryInMB, "total-memory", 0, "total memory limit in MB")
	cmd.Flags().IntVar(&config.totalInstanceMemoryInMB, "instance-memory", 0, "instance memory limit in MB")
	cmd.Flags().IntVar(&config.totalInstances, "instances", 0, "total instances limit")
	cmd.Flags().IntVar(&config.totalAppTasks, "app-tasks", 0, "total app tasks limit")
	cmd.Flags().IntVar(&config.logRateLimitInBytesPerSecond, "log-rate-limit", 0, "log rate limit in bytes per second")
	cmd.Flags().BoolVar(&config.paidServicesAllowed, "paid-services", true, "allow paid services")
	cmd.Flags().IntVar(&config.totalServiceInstances, "service-instances", 0, "total service instances limit")
	cmd.Flags().IntVar(&config.totalServiceKeys, "service-keys", 0, "total service keys limit")
	cmd.Flags().IntVar(&config.totalRoutes, "routes", 0, "total routes limit")
	cmd.Flags().IntVar(&config.totalReservedPorts, "reserved-ports", 0, "total reserved ports limit")
	cmd.Flags().IntVar(&config.totalDomains, "domains", 0, "total domains limit")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

// orgQuotaUpdateConfig holds all the configuration for updating org quota.
type orgQuotaUpdateConfig struct {
	newName                      string
	totalMemoryInMB              int
	totalInstanceMemoryInMB      int
	totalInstances               int
	totalAppTasks                int
	paidServicesAllowed          bool
	totalServiceInstances        int
	totalServiceKeys             int
	totalRoutes                  int
	totalReservedPorts           int
	totalDomains                 int
	logRateLimitInBytesPerSecond int
}

// findOrgQuotaGUID resolves the quota GUID from name or GUID.
func findOrgQuotaGUID(ctx context.Context, quotaClient capi.OrganizationQuotasClient, nameOrGUID string) (string, error) {
	quota, err := quotaClient.Get(ctx, nameOrGUID)
	if err == nil {
		return quota.GUID, nil
	}

	// Try by name
	params := capi.NewQueryParams()
	params.WithFilter("names", nameOrGUID)

	quotas, err := quotaClient.List(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to find organization quota: %w", err)
	}

	if len(quotas.Resources) == 0 {
		return "", fmt.Errorf("organization quota '%s': %w", nameOrGUID, ErrOrganizationQuotaNotFound)
	}

	return quotas.Resources[0].GUID, nil
}

// buildOrgQuotaUpdateRequest creates the update request based on changed flags.
func buildOrgQuotaUpdateRequest(cmd *cobra.Command, config *orgQuotaUpdateConfig) *capi.OrganizationQuotaUpdateRequest {
	updateReq := &capi.OrganizationQuotaUpdateRequest{}

	if config.newName != "" {
		updateReq.Name = &config.newName
	}

	// Build app limits if any app flags are set
	updateReq.Apps = buildAppLimits(cmd, config)

	// Build service limits if any service flags are set
	updateReq.Services = buildServiceLimits(cmd, config)

	// Build route limits if any route flags are set
	updateReq.Routes = buildRouteLimits(cmd, config)

	// Build domain limits if domain flags are set
	updateReq.Domains = buildDomainLimits(cmd, config)

	return updateReq
}

// buildAppLimits creates app limits section if app flags are changed.
func buildAppLimits(cmd *cobra.Command, config *orgQuotaUpdateConfig) *capi.OrganizationQuotaApps {
	return buildGenericAppLimits(cmd, config)
}

// buildServiceLimits creates service limits section if service flags are changed.
func buildServiceLimits(cmd *cobra.Command, config *orgQuotaUpdateConfig) *capi.OrganizationQuotaServices {
	if !cmd.Flags().Changed("paid-services") && !cmd.Flags().Changed("service-instances") &&
		!cmd.Flags().Changed("service-keys") {
		return nil
	}

	services := &capi.OrganizationQuotaServices{}

	if cmd.Flags().Changed("paid-services") {
		services.PaidServicesAllowed = &config.paidServicesAllowed
	}

	if cmd.Flags().Changed("service-instances") {
		services.TotalServiceInstances = &config.totalServiceInstances
	}

	if cmd.Flags().Changed("service-keys") {
		services.TotalServiceKeys = &config.totalServiceKeys
	}

	return services
}

// buildRouteLimits creates route limits section if route flags are changed.
func buildRouteLimits(cmd *cobra.Command, config *orgQuotaUpdateConfig) *capi.OrganizationQuotaRoutes {
	if !cmd.Flags().Changed("routes") && !cmd.Flags().Changed("reserved-ports") {
		return nil
	}

	routes := &capi.OrganizationQuotaRoutes{}

	if cmd.Flags().Changed("routes") {
		routes.TotalRoutes = &config.totalRoutes
	}

	if cmd.Flags().Changed("reserved-ports") {
		routes.TotalReservedPorts = &config.totalReservedPorts
	}

	return routes
}

// buildDomainLimits creates domain limits section if domain flags are changed.
func buildDomainLimits(cmd *cobra.Command, config *orgQuotaUpdateConfig) *capi.OrganizationQuotaDomains {
	if !cmd.Flags().Changed("domains") {
		return nil
	}

	domains := &capi.OrganizationQuotaDomains{}
	domains.TotalDomains = &config.totalDomains

	return domains
}

func newOrgQuotasUpdateCommand() *cobra.Command {
	config := &orgQuotaUpdateConfig{}

	cmd := &cobra.Command{
		Use:   "update QUOTA_NAME_OR_GUID",
		Short: "Update an organization quota",
		Long:  "Update an existing Cloud Foundry organization quota",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()
			quotaClient := client.OrganizationQuotas()

			// Find quota GUID
			quotaGUID, err := findOrgQuotaGUID(ctx, quotaClient, nameOrGUID)
			if err != nil {
				return err
			}

			// Build update request based on changed flags
			updateReq := buildOrgQuotaUpdateRequest(cmd, config)

			// Update quota
			updatedQuota, err := quotaClient.Update(ctx, quotaGUID, updateReq)
			if err != nil {
				return fmt.Errorf("failed to update organization quota: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully updated organization quota '%s'\n", updatedQuota.Name)

			return nil
		},
	}

	cmd.Flags().StringVar(&config.newName, "name", "", "new quota name")
	cmd.Flags().IntVar(&config.totalMemoryInMB, "total-memory", 0, "total memory limit in MB")
	cmd.Flags().IntVar(&config.totalInstanceMemoryInMB, "instance-memory", 0, "instance memory limit in MB")
	cmd.Flags().IntVar(&config.totalInstances, "instances", 0, "total instances limit")
	cmd.Flags().IntVar(&config.totalAppTasks, "app-tasks", 0, "total app tasks limit")
	cmd.Flags().IntVar(&config.logRateLimitInBytesPerSecond, "log-rate-limit", 0, "log rate limit in bytes per second")
	cmd.Flags().BoolVar(&config.paidServicesAllowed, "paid-services", true, "allow paid services")
	cmd.Flags().IntVar(&config.totalServiceInstances, "service-instances", 0, "total service instances limit")
	cmd.Flags().IntVar(&config.totalServiceKeys, "service-keys", 0, "total service keys limit")
	cmd.Flags().IntVar(&config.totalRoutes, "routes", 0, "total routes limit")
	cmd.Flags().IntVar(&config.totalReservedPorts, "reserved-ports", 0, "total reserved ports limit")
	cmd.Flags().IntVar(&config.totalDomains, "domains", 0, "total domains limit")

	return cmd
}

func newOrgQuotasDeleteCommand() *cobra.Command {
	config := DeleteConfig{
		Use:         "delete QUOTA_NAME_OR_GUID",
		Short:       "Delete an organization quota",
		Long:        "Delete a Cloud Foundry organization quota",
		EntityType:  "organization quota",
		GetResource: CreateOrganizationQuotaDeleteResourceFunc(),
		DeleteFunc: func(ctx context.Context, client interface{}, guid string) (*string, error) {
			capiClient, ok := client.(capi.Client)
			if !ok {
				return nil, constants.ErrInvalidClientType
			}

			job, err := capiClient.OrganizationQuotas().Delete(ctx, guid)
			if err != nil {
				return nil, fmt.Errorf("failed to delete organization quota: %w", err)
			}

			return &job.GUID, nil
		},
	}

	return createDeleteCommand(config)
}

// resolveOrgQuotaGUID finds an organization quota and returns its GUID and name.
func resolveOrgQuotaGUID(ctx context.Context, quotaClient capi.OrganizationQuotasClient, nameOrGUID string) (string, string, error) {
	quota, err := quotaClient.Get(ctx, nameOrGUID)
	if err == nil {
		return quota.GUID, quota.Name, nil
	}

	// Try by name
	params := capi.NewQueryParams()
	params.WithFilter("names", nameOrGUID)

	quotas, err := quotaClient.List(ctx, params)
	if err != nil {
		return "", "", fmt.Errorf("failed to find organization quota: %w", err)
	}

	if len(quotas.Resources) == 0 {
		return "", "", fmt.Errorf("organization quota '%s': %w", nameOrGUID, ErrOrganizationQuotaNotFound)
	}

	return quotas.Resources[0].GUID, quotas.Resources[0].Name, nil
}

// resolveOrganizationGUIDs resolves a list of organization names/GUIDs to GUIDs and names.
func resolveOrganizationGUIDs(ctx context.Context, orgsClient capi.OrganizationsClient, namesOrGUIDs []string) ([]string, []string, error) {
	var (
		orgGUIDs []string
		orgNames []string
	)

	for _, nameOrGUID := range namesOrGUIDs {
		org, err := orgsClient.Get(ctx, nameOrGUID)
		if err != nil {
			// Try by name
			params := capi.NewQueryParams()
			params.WithFilter("names", nameOrGUID)

			orgs, err := orgsClient.List(ctx, params)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to find organization '%s': %w", nameOrGUID, err)
			}

			if len(orgs.Resources) == 0 {
				return nil, nil, fmt.Errorf("organization '%s': %w", nameOrGUID, ErrOrganizationNotFound)
			}

			orgGUIDs = append(orgGUIDs, orgs.Resources[0].GUID)
			orgNames = append(orgNames, orgs.Resources[0].Name)
		} else {
			orgGUIDs = append(orgGUIDs, org.GUID)
			orgNames = append(orgNames, org.Name)
		}
	}

	return orgGUIDs, orgNames, nil
}

func newOrgQuotasApplyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "apply QUOTA_NAME_OR_GUID ORG_NAME_OR_GUID...",
		Short: "Apply quota to organizations",
		Long:  "Apply an organization quota to one or more organizations",
		Args:  cobra.MinimumNArgs(constants.TwoArgumentsMax),
		RunE: func(cmd *cobra.Command, args []string) error {
			quotaNameOrGUID := args[0]
			orgNamesOrGUIDs := args[1:]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()
			quotaClient := client.OrganizationQuotas()
			orgsClient := client.Organizations()

			// Find quota
			quotaGUID, quotaName, err := resolveOrgQuotaGUID(ctx, quotaClient, quotaNameOrGUID)
			if err != nil {
				return err
			}

			// Resolve organization GUIDs
			orgGUIDs, orgNames, err := resolveOrganizationGUIDs(ctx, orgsClient, orgNamesOrGUIDs)
			if err != nil {
				return err
			}

			// Apply quota to organizations
			_, err = quotaClient.ApplyToOrganizations(ctx, quotaGUID, orgGUIDs)
			if err != nil {
				return fmt.Errorf("failed to apply quota to organizations: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully applied quota '%s' to organizations: %s\n",
				quotaName, strings.Join(orgNames, ", "))

			return nil
		},
	}
}
