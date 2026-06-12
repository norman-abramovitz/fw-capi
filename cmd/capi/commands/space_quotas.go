package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"os"
	"strconv"
	"strings"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewSpaceQuotasCommand creates the space quotas command group.
func NewSpaceQuotasCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "space-quotas",
		Aliases: []string{"space-quota", "sq"},
		Short:   "Manage space quotas",
		Long:    "List, create, update, delete, apply, and remove space quotas",
	}

	cmd.AddCommand(newSpaceQuotasListCommand())
	cmd.AddCommand(newSpaceQuotasGetCommand())
	cmd.AddCommand(newSpaceQuotasCreateCommand())
	cmd.AddCommand(newSpaceQuotasUpdateCommand())
	cmd.AddCommand(newSpaceQuotasDeleteCommand())
	cmd.AddCommand(newSpaceQuotasApplyCommand())
	cmd.AddCommand(newSpaceQuotasRemoveCommand())

	return cmd
}

func newSpaceQuotasListCommand() *cobra.Command {
	var (
		allPages bool
		perPage  int
		orgName  string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List space quotas",
		Long:  "List all space quotas",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSpaceQuotasList(cmd, allPages, perPage, orgName)
		},
	}

	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "results per page")
	cmd.Flags().StringVarP(&orgName, "org", "o", "", "filter by organization name")

	return cmd
}

func runSpaceQuotasList(cmd *cobra.Command, allPages bool, perPage int, orgName string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()
	params := capi.NewQueryParams()
	params.PerPage = perPage

	// Filter by organization if specified
	if orgName != "" {
		orgGUID, err := resolveOrganizationGUID(ctx, client, orgName)
		if err != nil {
			return err
		}

		params.WithFilter("organization_guids", orgGUID)
	}

	quotas, err := client.SpaceQuotas().List(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list space quotas: %w", err)
	}

	// Handle pagination
	allQuotas, err := handleSpaceQuotasPagination(ctx, client, params, quotas, allPages)
	if err != nil {
		return err
	}

	// Output results
	return renderSpaceQuotasOutput(allQuotas, quotas.Pagination, allPages)
}

func resolveOrganizationGUID(ctx context.Context, client capi.Client, orgName string) (string, error) {
	orgParams := capi.NewQueryParams()
	orgParams.WithFilter("names", orgName)

	orgs, err := client.Organizations().List(ctx, orgParams)
	if err != nil {
		return "", fmt.Errorf("failed to find organization: %w", err)
	}

	if len(orgs.Resources) == 0 {
		return "", fmt.Errorf("organization '%s': %w", orgName, ErrOrganizationNotFound)
	}

	return orgs.Resources[0].GUID, nil
}

//nolint:dupl // Acceptable duplication - each pagination handler works with different resource types and endpoints
func handleSpaceQuotasPagination(ctx context.Context, client capi.Client, params *capi.QueryParams, quotas *capi.ListResponse[capi.SpaceQuotaV3], allPages bool) ([]capi.SpaceQuotaV3, error) {
	if !allPages || quotas.Pagination.TotalPages <= 1 {
		return quotas.Resources, nil
	}

	handler := &PaginationHandler[capi.SpaceQuotaV3]{
		FetchPage: func(ctx context.Context, params *capi.QueryParams, page int) ([]capi.SpaceQuotaV3, *capi.Pagination, error) {
			params.Page = page

			moreQuotas, err := client.SpaceQuotas().List(ctx, params)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to list space quotas: %w", err)
			}

			return moreQuotas.Resources, &moreQuotas.Pagination, nil
		},
	}

	return handler.FetchAllPages(ctx, params, allPages, quotas.Resources, &quotas.Pagination)
}

func renderSpaceQuotasOutput(allQuotas []capi.SpaceQuotaV3, pagination capi.Pagination, allPages bool) error {
	renderer := &StandardOutputRenderer[capi.SpaceQuotaV3]{
		RenderTable: renderSpaceQuotasTable,
	}

	output := viper.GetString("output")

	return renderer.Render(allQuotas, &pagination, allPages, output)
}

func renderSpaceQuotasTable(quotas []capi.SpaceQuotaV3, pagination *capi.Pagination, allPages bool) error {
	if len(quotas) == 0 {
		_, _ = os.Stdout.WriteString("No space quotas found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "GUID", "Total Memory", "Services", "Routes", "Created")

	for _, quota := range quotas {
		memoryStr := formatMemoryLimit(quota.Apps)
		servicesStr := formatServicesLimit(quota.Services)
		routesStr := formatRoutesLimit(quota.Routes)

		_ = table.Append(quota.Name, quota.GUID, memoryStr, servicesStr, routesStr,
			quota.CreatedAt.Format("2006-01-02"))
	}

	_ = table.Render()

	if !allPages && pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", pagination.TotalPages)
	}

	return nil
}

func formatMemoryLimit(apps *capi.SpaceQuotaApps) string {
	if apps != nil && apps.TotalMemoryInMB != nil {
		return fmt.Sprintf("%d MB", *apps.TotalMemoryInMB)
	}

	return Unlimited
}

func formatServicesLimit(services *capi.SpaceQuotaServices) string {
	if services != nil && services.TotalServiceInstances != nil {
		return strconv.Itoa(*services.TotalServiceInstances)
	}

	return Unlimited
}

func formatRoutesLimit(routes *capi.SpaceQuotaRoutes) string {
	if routes != nil && routes.TotalRoutes != nil {
		return strconv.Itoa(*routes.TotalRoutes)
	}

	return Unlimited
}

func newSpaceQuotasGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get QUOTA_NAME_OR_GUID",
		Short: "Get space quota details",
		Long:  "Display detailed information about a specific space quota",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()
			quotaClient := client.SpaceQuotas()

			// Try to get by GUID first
			quota, err := quotaClient.Get(ctx, nameOrGUID)
			if err != nil {
				// If not found by GUID, try by name
				params := capi.NewQueryParams()
				params.WithFilter("names", nameOrGUID)

				quotas, err := quotaClient.List(ctx, params)
				if err != nil {
					return fmt.Errorf("failed to find space quota: %w", err)
				}

				if len(quotas.Resources) == 0 {
					return fmt.Errorf("space quota '%s': %w", nameOrGUID, ErrSpaceQuotaNotFound)
				}

				quota = &quotas.Resources[0]
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
				return displaySpaceQuotaTable(quota)
			}
		},
	}
}

// displaySpaceQuotaTable displays space quota information in table format.
func displaySpaceQuotaTable(quota *capi.SpaceQuotaV3) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")
	_ = table.Append("Name", quota.Name)
	_ = table.Append("GUID", quota.GUID)
	_ = table.Append("Created", quota.CreatedAt.Format("2006-01-02 15:04:05"))

	_ = table.Append("Updated", quota.UpdatedAt.Format("2006-01-02 15:04:05"))

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	err = displayAppLimitsTable(quota.Apps)
	if err != nil {
		return err
	}

	err = displayServiceLimitsTable(quota.Services)
	if err != nil {
		return err
	}

	err = displayRouteLimitsTable(quota.Routes)
	if err != nil {
		return err
	}

	return nil
}

// displayAppLimitsTable displays app limits if present.
func displayAppLimitsTable(apps *capi.SpaceQuotaApps) error {
	if apps == nil {
		return nil
	}

	_, _ = os.Stdout.WriteString("\nApp Limits:\n")

	appTable := tablewriter.NewWriter(os.Stdout)
	appTable.Header("Limit", "Value")

	addAppLimitRow(appTable, "Total Memory", apps.TotalMemoryInMB, func(v int) string { return fmt.Sprintf("%d MB", v) })
	addAppLimitRow(appTable, "Instance Memory", apps.PerProcessMemoryInMB, func(v int) string { return fmt.Sprintf("%d MB", v) })
	addAppLimitRow(appTable, "Total Instances", apps.TotalInstances, strconv.Itoa)
	addAppLimitRow(appTable, "Total App Tasks", apps.PerAppTasks, strconv.Itoa)

	if apps.LogRateLimitInBytesPerSecond != nil {
		_ = appTable.Append("Log Rate Limit", fmt.Sprintf("%d bytes/sec", *apps.LogRateLimitInBytesPerSecond))
	} else {
		_ = appTable.Append("Log Rate Limit", Unlimited)
	}

	err := appTable.Render()
	if err != nil {
		return fmt.Errorf("failed to render app limits table: %w", err)
	}

	return nil
}

// displayServiceLimitsTable displays service limits if present.
func displayServiceLimitsTable(services *capi.SpaceQuotaServices) error {
	if services == nil {
		return nil
	}

	_, _ = os.Stdout.WriteString("\nService Limits:\n")

	serviceTable := tablewriter.NewWriter(os.Stdout)
	serviceTable.Header("Limit", "Value")

	if services.PaidServicesAllowed != nil {
		_ = serviceTable.Append("Paid Services", strconv.FormatBool(*services.PaidServicesAllowed))
	}

	addServiceLimitRow(serviceTable, "Total Service Instances", services.TotalServiceInstances)
	addServiceLimitRow(serviceTable, "Total Service Keys", services.TotalServiceKeys)

	err := serviceTable.Render()
	if err != nil {
		return fmt.Errorf("failed to render service limits table: %w", err)
	}

	return nil
}

// displayRouteLimitsTable displays route limits if present.
func displayRouteLimitsTable(routes *capi.SpaceQuotaRoutes) error {
	if routes == nil {
		return nil
	}

	_, _ = os.Stdout.WriteString("\nRoute Limits:\n")

	routeTable := tablewriter.NewWriter(os.Stdout)
	routeTable.Header("Limit", "Value")

	addRouteLimitRow(routeTable, "Total Routes", routes.TotalRoutes)
	addRouteLimitRow(routeTable, "Total Reserved Ports", routes.TotalReservedPorts)

	err := routeTable.Render()
	if err != nil {
		return fmt.Errorf("failed to render route limits table: %w", err)
	}

	return nil
}

// addAppLimitRow adds a row with proper nil checking and formatting.
func addAppLimitRow(table *tablewriter.Table, label string, value *int, formatter func(int) string) {
	if value != nil {
		_ = table.Append(label, formatter(*value))
	} else {
		_ = table.Append(label, Unlimited)
	}
}

// addServiceLimitRow adds a service limit row with proper nil checking.
func addServiceLimitRow(table *tablewriter.Table, label string, value *int) {
	if value != nil {
		_ = table.Append(label, strconv.Itoa(*value))
	} else {
		_ = table.Append(label, Unlimited)
	}
}

// addRouteLimitRow adds a route limit row with proper nil checking.
func addRouteLimitRow(table *tablewriter.Table, label string, value *int) {
	if value != nil {
		_ = table.Append(label, strconv.Itoa(*value))
	} else {
		_ = table.Append(label, Unlimited)
	}
}

// spaceQuotaCreateConfig holds configuration for creating space quotas.
type spaceQuotaCreateConfig struct {
	name                         string
	orgName                      string
	totalMemoryInMB              int
	totalInstanceMemoryInMB      int
	totalInstances               int
	totalAppTasks                int
	paidServicesAllowed          bool
	totalServiceInstances        int
	totalServiceKeys             int
	totalRoutes                  int
	totalReservedPorts           int
	logRateLimitInBytesPerSecond int
}

// findOrgGUIDForSpaceQuota finds the organization GUID by name.
func findOrgGUIDForSpaceQuota(ctx context.Context, client capi.Client, orgName string) (string, error) {
	orgParams := capi.NewQueryParams()
	orgParams.WithFilter("names", orgName)

	orgs, err := client.Organizations().List(ctx, orgParams)
	if err != nil {
		return "", fmt.Errorf("failed to find organization: %w", err)
	}

	if len(orgs.Resources) == 0 {
		return "", fmt.Errorf("organization '%s': %w", orgName, ErrOrganizationNotFound)
	}

	return orgs.Resources[0].GUID, nil
}

// buildSpaceQuotaCreateRequest creates the create request based on changed flags.
func buildSpaceQuotaCreateRequest(cmd *cobra.Command, config *spaceQuotaCreateConfig, orgGUID string) *capi.SpaceQuotaV3CreateRequest {
	createReq := &capi.SpaceQuotaV3CreateRequest{
		Name: config.name,
		Relationships: capi.SpaceQuotaRelationships{
			Organization: capi.Relationship{
				Data: &capi.RelationshipData{GUID: orgGUID},
			},
			Spaces: capi.ToManyRelationship{
				Data: []capi.RelationshipData{},
			},
		},
	}

	// Build app limits if any app flags are set
	createReq.Apps = buildSpaceQuotaAppLimits(cmd, config)

	// Build service limits if any service flags are set
	createReq.Services = buildSpaceQuotaServiceLimits(cmd, config)

	// Build route limits if any route flags are set
	createReq.Routes = buildSpaceQuotaRouteLimits(cmd, config)

	return createReq
}

// buildSpaceQuotaAppLimits creates app limits section for space quota.
func buildSpaceQuotaAppLimits(cmd *cobra.Command, config *spaceQuotaCreateConfig) *capi.SpaceQuotaApps {
	return buildSpaceQuotaAppLimitsGeneric(cmd, config)
}

// AppLimitsConfig interface for extracting app limits from config structs

// Implement AppLimitsConfig for spaceQuotaCreateConfig.
func (c *spaceQuotaCreateConfig) GetTotalMemoryInMB() int      { return c.totalMemoryInMB }
func (c *spaceQuotaCreateConfig) GetPerProcessMemoryInMB() int { return c.totalInstanceMemoryInMB }
func (c *spaceQuotaCreateConfig) GetTotalInstances() int       { return c.totalInstances }
func (c *spaceQuotaCreateConfig) GetPerAppTasks() int          { return c.totalAppTasks }
func (c *spaceQuotaCreateConfig) GetLogRateLimitInBytesPerSecond() int {
	return c.logRateLimitInBytesPerSecond
}

// Implement AppLimitsConfig for spaceQuotaUpdateConfig.
func (c *spaceQuotaUpdateConfig) GetTotalMemoryInMB() int      { return c.totalMemoryInMB }
func (c *spaceQuotaUpdateConfig) GetPerProcessMemoryInMB() int { return c.totalInstanceMemoryInMB }
func (c *spaceQuotaUpdateConfig) GetTotalInstances() int       { return c.totalInstances }
func (c *spaceQuotaUpdateConfig) GetPerAppTasks() int          { return c.totalAppTasks }
func (c *spaceQuotaUpdateConfig) GetLogRateLimitInBytesPerSecond() int {
	return c.logRateLimitInBytesPerSecond
}

// buildSpaceQuotaAppLimitsGeneric builds app limits for space quotas using any config that implements AppLimitsConfig.
func buildSpaceQuotaAppLimitsGeneric(cmd *cobra.Command, config AppLimitsConfig) *capi.SpaceQuotaApps {
	return BuildSpaceQuotaApps(cmd, config)
}

// buildSpaceQuotaServiceLimits creates service limits section for space quota.
func buildSpaceQuotaServiceLimits(cmd *cobra.Command, config *spaceQuotaCreateConfig) *capi.SpaceQuotaServices {
	if !cmd.Flags().Changed("paid-services") && !cmd.Flags().Changed("service-instances") &&
		!cmd.Flags().Changed("service-keys") {
		return nil
	}

	services := &capi.SpaceQuotaServices{}

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

// buildSpaceQuotaRouteLimits creates route limits section for space quota.
func buildSpaceQuotaRouteLimits(cmd *cobra.Command, config *spaceQuotaCreateConfig) *capi.SpaceQuotaRoutes {
	if !cmd.Flags().Changed("routes") && !cmd.Flags().Changed("reserved-ports") {
		return nil
	}

	routes := &capi.SpaceQuotaRoutes{}

	if cmd.Flags().Changed("routes") {
		routes.TotalRoutes = &config.totalRoutes
	}

	if cmd.Flags().Changed("reserved-ports") {
		routes.TotalReservedPorts = &config.totalReservedPorts
	}

	return routes
}

func newSpaceQuotasCreateCommand() *cobra.Command {
	config := &spaceQuotaCreateConfig{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new space quota",
		Long:  "Create a new Cloud Foundry space quota",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config.name == "" {
				return ErrQuotaNameRequired
			}

			if config.orgName == "" {
				return ErrOrganizationNameRequired
			}

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find organization GUID
			orgGUID, err := findOrgGUIDForSpaceQuota(ctx, client, config.orgName)
			if err != nil {
				return err
			}

			// Build create request based on changed flags
			createReq := buildSpaceQuotaCreateRequest(cmd, config, orgGUID)

			// Create space quota
			quota, err := client.SpaceQuotas().Create(ctx, createReq)
			if err != nil {
				return fmt.Errorf("failed to create space quota: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully created space quota '%s' with GUID %s\n", quota.Name, quota.GUID)

			return nil
		},
	}

	cmd.Flags().StringVarP(&config.name, "name", "n", "", "quota name (required)")
	cmd.Flags().StringVarP(&config.orgName, "org", "o", "", "organization name (required)")
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
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("org")

	return cmd
}

// spaceQuotaUpdateConfig holds configuration for updating space quotas.
type spaceQuotaUpdateConfig struct {
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
	logRateLimitInBytesPerSecond int
}

// findSpaceQuotaGUID resolves the space quota GUID from name or GUID.
func findSpaceQuotaGUID(ctx context.Context, quotaClient capi.SpaceQuotasClient, nameOrGUID string) (string, error) {
	quota, err := quotaClient.Get(ctx, nameOrGUID)
	if err == nil {
		return quota.GUID, nil
	}

	// Try by name
	params := capi.NewQueryParams()
	params.WithFilter("names", nameOrGUID)

	quotas, err := quotaClient.List(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to find space quota: %w", err)
	}

	if len(quotas.Resources) == 0 {
		return "", fmt.Errorf("space quota '%s': %w", nameOrGUID, ErrSpaceQuotaNotFound)
	}

	return quotas.Resources[0].GUID, nil
}

// buildSpaceQuotaUpdateRequest creates the update request based on changed flags.
func buildSpaceQuotaUpdateRequest(cmd *cobra.Command, config *spaceQuotaUpdateConfig) *capi.SpaceQuotaV3UpdateRequest {
	updateReq := &capi.SpaceQuotaV3UpdateRequest{}

	if config.newName != "" {
		updateReq.Name = &config.newName
	}

	// Build app limits if any app flags are set
	updateReq.Apps = buildSpaceQuotaUpdateAppLimits(cmd, config)

	// Build service limits if any service flags are set
	updateReq.Services = buildSpaceQuotaUpdateServiceLimits(cmd, config)

	// Build route limits if any route flags are set
	updateReq.Routes = buildSpaceQuotaUpdateRouteLimits(cmd, config)

	return updateReq
}

// buildSpaceQuotaUpdateAppLimits creates app limits section for space quota update.
func buildSpaceQuotaUpdateAppLimits(cmd *cobra.Command, config *spaceQuotaUpdateConfig) *capi.SpaceQuotaApps {
	return buildSpaceQuotaAppLimitsGeneric(cmd, config)
}

// buildSpaceQuotaUpdateServiceLimits creates service limits section for space quota update.
func buildSpaceQuotaUpdateServiceLimits(cmd *cobra.Command, config *spaceQuotaUpdateConfig) *capi.SpaceQuotaServices {
	if !cmd.Flags().Changed("paid-services") && !cmd.Flags().Changed("service-instances") &&
		!cmd.Flags().Changed("service-keys") {
		return nil
	}

	services := &capi.SpaceQuotaServices{}

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

// buildSpaceQuotaUpdateRouteLimits creates route limits section for space quota update.
func buildSpaceQuotaUpdateRouteLimits(cmd *cobra.Command, config *spaceQuotaUpdateConfig) *capi.SpaceQuotaRoutes {
	if !cmd.Flags().Changed("routes") && !cmd.Flags().Changed("reserved-ports") {
		return nil
	}

	routes := &capi.SpaceQuotaRoutes{}

	if cmd.Flags().Changed("routes") {
		routes.TotalRoutes = &config.totalRoutes
	}

	if cmd.Flags().Changed("reserved-ports") {
		routes.TotalReservedPorts = &config.totalReservedPorts
	}

	return routes
}

func newSpaceQuotasUpdateCommand() *cobra.Command {
	config := &spaceQuotaUpdateConfig{}

	cmd := &cobra.Command{
		Use:   "update QUOTA_NAME_OR_GUID",
		Short: "Update a space quota",
		Long:  "Update an existing Cloud Foundry space quota",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()
			quotaClient := client.SpaceQuotas()

			// Find quota GUID
			quotaGUID, err := findSpaceQuotaGUID(ctx, quotaClient, nameOrGUID)
			if err != nil {
				return err
			}

			// Build update request based on changed flags
			updateReq := buildSpaceQuotaUpdateRequest(cmd, config)

			// Update quota
			updatedQuota, err := quotaClient.Update(ctx, quotaGUID, updateReq)
			if err != nil {
				return fmt.Errorf("failed to update space quota: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully updated space quota '%s'\n", updatedQuota.Name)

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

	return cmd
}

func newSpaceQuotasDeleteCommand() *cobra.Command {
	config := DeleteConfig{
		Use:         "delete QUOTA_NAME_OR_GUID",
		Short:       "Delete a space quota",
		Long:        "Delete a Cloud Foundry space quota",
		EntityType:  "space quota",
		GetResource: CreateSpaceQuotaDeleteResourceFunc(),
		DeleteFunc: func(ctx context.Context, client interface{}, guid string) (*string, error) {
			capiClient, ok := client.(capi.Client)
			if !ok {
				return nil, constants.ErrInvalidClientType
			}

			job, err := capiClient.SpaceQuotas().Delete(ctx, guid)
			if err != nil {
				return nil, fmt.Errorf("failed to delete space quota: %w", err)
			}

			return &job.GUID, nil
		},
	}

	return createDeleteCommand(config)
}

// resolveSpaceQuota finds a space quota by GUID or name and returns its GUID and name.
func resolveSpaceQuota(ctx context.Context, client capi.Client, quotaNameOrGUID string) (string, string, error) {
	quotaClient := client.SpaceQuotas()

	// Try to get by GUID first
	quota, err := quotaClient.Get(ctx, quotaNameOrGUID)
	if err == nil {
		return quota.GUID, quota.Name, nil
	}

	// Try by name
	params := capi.NewQueryParams()
	params.WithFilter("names", quotaNameOrGUID)

	quotas, err := quotaClient.List(ctx, params)
	if err != nil {
		return "", "", fmt.Errorf("failed to find space quota: %w", err)
	}

	if len(quotas.Resources) == 0 {
		return "", "", fmt.Errorf("space quota '%s': %w", quotaNameOrGUID, ErrSpaceQuotaNotFound)
	}

	return quotas.Resources[0].GUID, quotas.Resources[0].Name, nil
}

// resolveSpace finds a space by GUID or name and returns its GUID and name.
func resolveSpace(ctx context.Context, client capi.Client, spaceNameOrGUID string) (string, string, error) {
	spacesClient := client.Spaces()

	// Try to get by GUID first
	space, err := spacesClient.Get(ctx, spaceNameOrGUID)
	if err == nil {
		return space.GUID, space.Name, nil
	}

	// Try by name
	params := capi.NewQueryParams()
	params.WithFilter("names", spaceNameOrGUID)
	// Add org filter if targeted
	if orgGUID := viper.GetString("organization_guid"); orgGUID != "" {
		params.WithFilter("organization_guids", orgGUID)
	}

	spaces, err := spacesClient.List(ctx, params)
	if err != nil {
		return "", "", fmt.Errorf("failed to find space '%s': %w", spaceNameOrGUID, err)
	}

	if len(spaces.Resources) == 0 {
		return "", "", fmt.Errorf("space '%s': %w", spaceNameOrGUID, ErrSpaceNotFound)
	}

	return spaces.Resources[0].GUID, spaces.Resources[0].Name, nil
}

// resolveSpaces resolves multiple space names or GUIDs to their GUIDs and names.
func resolveSpaces(ctx context.Context, client capi.Client, spaceNamesOrGUIDs []string) ([]string, []string, error) {
	spaceGUIDs := make([]string, 0, len(spaceNamesOrGUIDs))
	spaceNames := make([]string, 0, len(spaceNamesOrGUIDs))

	for _, spaceNameOrGUID := range spaceNamesOrGUIDs {
		guid, name, err := resolveSpace(ctx, client, spaceNameOrGUID)
		if err != nil {
			return nil, nil, err
		}

		spaceGUIDs = append(spaceGUIDs, guid)
		spaceNames = append(spaceNames, name)
	}

	return spaceGUIDs, spaceNames, nil
}

func newSpaceQuotasApplyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "apply QUOTA_NAME_OR_GUID SPACE_NAME_OR_GUID...",
		Short: "Apply quota to spaces",
		Long:  "Apply a space quota to one or more spaces",
		Args:  cobra.MinimumNArgs(constants.TwoArgumentsMax),
		RunE: func(cmd *cobra.Command, args []string) error {
			quotaNameOrGUID := args[0]
			spaceNamesOrGUIDs := args[1:]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find quota
			quotaGUID, quotaName, err := resolveSpaceQuota(ctx, client, quotaNameOrGUID)
			if err != nil {
				return err
			}

			// Resolve space GUIDs
			spaceGUIDs, spaceNames, err := resolveSpaces(ctx, client, spaceNamesOrGUIDs)
			if err != nil {
				return err
			}

			// Apply quota to spaces
			_, err = client.SpaceQuotas().ApplyToSpaces(ctx, quotaGUID, spaceGUIDs)
			if err != nil {
				return fmt.Errorf("failed to apply quota to spaces: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully applied quota '%s' to spaces: %s\n",
				quotaName, strings.Join(spaceNames, ", "))

			return nil
		},
	}
}

func newSpaceQuotasRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove QUOTA_NAME_OR_GUID SPACE_NAME_OR_GUID",
		Short: "Remove quota from space",
		Long:  "Remove a space quota from a specific space",
		Args:  cobra.ExactArgs(constants.TwoArgumentsMax),
		RunE:  runSpaceQuotasRemove,
	}
}

func runSpaceQuotasRemove(cmd *cobra.Command, args []string) error {
	quotaNameOrGUID := args[0]
	spaceNameOrGUID := args[1]

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()
	quotaClient := client.SpaceQuotas()
	spacesClient := client.Spaces()

	// Find quota
	var (
		quotaGUID string
		quotaName string
	)

	quota, err := quotaClient.Get(ctx, quotaNameOrGUID)
	if err != nil {
		// Try by name
		params := capi.NewQueryParams()
		params.WithFilter("names", quotaNameOrGUID)

		quotas, err := quotaClient.List(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to find space quota: %w", err)
		}

		if len(quotas.Resources) == 0 {
			return fmt.Errorf("space quota '%s': %w", quotaNameOrGUID, ErrSpaceQuotaNotFound)
		}

		quotaGUID = quotas.Resources[0].GUID
		quotaName = quotas.Resources[0].Name
	} else {
		quotaGUID = quota.GUID
		quotaName = quota.Name
	}

	// Find space
	var (
		spaceGUID string
		spaceName string
	)

	space, err := spacesClient.Get(ctx, spaceNameOrGUID)
	if err != nil {
		// Try by name
		params := capi.NewQueryParams()
		params.WithFilter("names", spaceNameOrGUID)

		// Add org filter if targeted
		if orgGUID := viper.GetString("organization_guid"); orgGUID != "" {
			params.WithFilter("organization_guids", orgGUID)
		}

		spaces, err := spacesClient.List(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to find space: %w", err)
		}

		if len(spaces.Resources) == 0 {
			return fmt.Errorf("space '%s': %w", spaceNameOrGUID, ErrSpaceNotFound)
		}

		spaceGUID = spaces.Resources[0].GUID
		spaceName = spaces.Resources[0].Name
	} else {
		spaceGUID = space.GUID
		spaceName = space.Name
	}

	// Remove quota from space
	err = quotaClient.RemoveFromSpace(ctx, quotaGUID, spaceGUID)
	if err != nil {
		return fmt.Errorf("failed to remove quota from space: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully removed quota '%s' from space '%s'\n", quotaName, spaceName)

	return nil
}
