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

// NewDomainsCommand creates the domains command group.
func NewDomainsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "domains",
		Aliases: []string{"domain"},
		Short:   "Manage domains",
		Long:    "List and manage Cloud Foundry domains",
	}

	cmd.AddCommand(newDomainsListCommand())
	cmd.AddCommand(newDomainsGetCommand())
	cmd.AddCommand(newDomainsCreateCommand())
	cmd.AddCommand(newDomainsUpdateCommand())
	cmd.AddCommand(newDomainsDeleteCommand())
	cmd.AddCommand(newDomainsShareCommand())
	cmd.AddCommand(newDomainsUnshareCommand())

	return cmd
}

func newDomainsListCommand() *cobra.Command {
	var (
		allPages bool
		perPage  int
		orgName  string
		internal bool
	)

	cmd := &cobra.Command{
		Use:   List,
		Short: "List domains",
		Long:  "List all domains the user has access to",
		RunE: func(cmd *cobra.Command, args []string) error {
			filters := &domainsListFilters{
				allPages: allPages,
				perPage:  perPage,
				orgName:  orgName,
				internal: internal,
				cmd:      cmd,
			}

			return runDomainsList(filters)
		},
	}

	cmd.Flags().StringVarP(&orgName, "org", "o", "", "filter by organization name")
	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "results per page")
	cmd.Flags().BoolVar(&internal, "internal", false, "filter by internal domains")

	return cmd
}

type domainsListFilters struct {
	allPages bool
	perPage  int
	orgName  string
	internal bool
	cmd      *cobra.Command
}

func runDomainsList(filters *domainsListFilters) error {
	client, err := CreateClientWithAPI(filters.cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	params, err := buildDomainsListParams(ctx, client, filters)
	if err != nil {
		return err
	}

	domains, err := client.Domains().List(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list domains: %w", err)
	}

	allDomains, err := fetchAllDomainsPages(ctx, client, domains, params, filters.allPages)
	if err != nil {
		return err
	}

	return outputDomainsList(allDomains, domains, filters.allPages)
}

func buildDomainsListParams(ctx context.Context, client capi.Client, filters *domainsListFilters) (*capi.QueryParams, error) {
	params := capi.NewQueryParams()
	params.PerPage = filters.perPage

	err := addOrgFilterToDomains(ctx, client, params, filters.orgName)
	if err != nil {
		return nil, err
	}

	addInternalFilterToDomains(params, filters)

	return params, nil
}

func addOrgFilterToDomains(ctx context.Context, client capi.Client, params *capi.QueryParams, orgName string) error {
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
		return fmt.Errorf("organization '%s': %w", orgName, ErrOrganizationNotFound)
	}

	params.WithFilter("organization_guids", orgs.Resources[0].GUID)

	return nil
}

func addInternalFilterToDomains(params *capi.QueryParams, filters *domainsListFilters) {
	if !filters.cmd.Flags().Changed("internal") {
		return
	}

	if filters.internal {
		params.WithFilter("internal", "true")
	} else {
		params.WithFilter("internal", "false")
	}
}

func fetchAllDomainsPages(ctx context.Context, client capi.Client, domains *capi.DomainsList, params *capi.QueryParams, allPages bool) ([]capi.Domain, error) {
	allDomains := domains.Resources
	if !allPages || domains.Pagination.TotalPages <= 1 {
		return allDomains, nil
	}

	for page := 2; page <= domains.Pagination.TotalPages; page++ {
		params.Page = page

		moreDomains, err := client.Domains().List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		allDomains = append(allDomains, moreDomains.Resources...)
	}

	return allDomains, nil
}

func outputDomainsList(allDomains []capi.Domain, domains *capi.DomainsList, allPages bool) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return outputDomainsListJSON(allDomains)
	case OutputFormatYAML:
		return outputDomainsListYAML(allDomains)
	default:
		return outputDomainsListTable(allDomains, domains, allPages)
	}
}

func outputDomainsListJSON(domains []capi.Domain) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(domains)
	if err != nil {
		return fmt.Errorf("failed to encode domains as JSON: %w", err)
	}

	return nil
}

func outputDomainsListYAML(domains []capi.Domain) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(domains)
	if err != nil {
		return fmt.Errorf("failed to encode domains as YAML: %w", err)
	}

	return nil
}

func outputDomainsListTable(allDomains []capi.Domain, domains *capi.DomainsList, allPages bool) error {
	if len(allDomains) == 0 {
		_, _ = os.Stdout.WriteString("No domains found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "Type", "Protocols", "Router Group", "Created")

	for _, domain := range allDomains {
		info := buildDomainTableInfo(domain)
		_ = table.Append(domain.Name, info.domainType, info.protocols, info.routerGroup, domain.CreatedAt.Format("2006-01-02"))
	}

	_ = table.Render()

	if !allPages && domains.Pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", domains.Pagination.TotalPages)
	}

	return nil
}

type domainTableInfo struct {
	domainType  string
	protocols   string
	routerGroup string
}

func buildDomainTableInfo(domain capi.Domain) domainTableInfo {
	return domainTableInfo{
		domainType:  formatDomainType(domain.Internal),
		protocols:   strings.Join(domain.SupportedProtocols, ", "),
		routerGroup: formatRouterGroup(domain.RouterGroup),
	}
}

func formatDomainType(internal bool) string {
	if internal {
		return "internal"
	}

	return "shared"
}

func formatRouterGroup(routerGroup *capi.RouterGroup) string {
	if routerGroup != nil {
		return routerGroup.GUID
	}

	return "default"
}

func newDomainsGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get DOMAIN_NAME_OR_GUID",
		Short: "Get domain details",
		Long:  "Display detailed information about a specific domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDomainsGet(cmd, args[0])
		},
	}
}

func runDomainsGet(cmd *cobra.Command, nameOrGUID string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	domain, err := resolveDomain(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	return outputDomainDetails(ctx, client, domain)
}

func resolveDomain(ctx context.Context, client capi.Client, nameOrGUID string) (*capi.Domain, error) {
	// Try to get by GUID first
	domain, err := client.Domains().Get(ctx, nameOrGUID)
	if err == nil {
		return domain, nil
	}

	// If not found by GUID, try by name
	params := capi.NewQueryParams()
	params.WithFilter("names", nameOrGUID)

	domains, err := client.Domains().List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find domain: %w", err)
	}

	if len(domains.Resources) == 0 {
		return nil, fmt.Errorf("domain '%s': %w", nameOrGUID, ErrDomainNotFound)
	}

	return &domains.Resources[0], nil
}

func outputDomainDetails(ctx context.Context, client capi.Client, domain *capi.Domain) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return outputDomainDetailsJSON(domain)
	case OutputFormatYAML:
		return outputDomainDetailsYAML(domain)
	default:
		return outputDomainDetailsTable(ctx, client, domain)
	}
}

func outputDomainDetailsJSON(domain *capi.Domain) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(domain)
	if err != nil {
		return fmt.Errorf("failed to encode domain as JSON: %w", err)
	}

	return nil
}

func outputDomainDetailsYAML(domain *capi.Domain) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(domain)
	if err != nil {
		return fmt.Errorf("failed to encode domain as YAML: %w", err)
	}

	return nil
}

func outputDomainDetailsTable(ctx context.Context, client capi.Client, domain *capi.Domain) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	addDomainBasicInfo(table, domain)
	addDomainOptionalInfo(table, domain)
	addDomainOrganizationInfo(ctx, client, table, domain)

	_, _ = fmt.Fprintf(os.Stdout, "Domain: %s\n\n", domain.Name)

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render domain table: %w", err)
	}

	return nil
}

func addDomainBasicInfo(table *tablewriter.Table, domain *capi.Domain) {
	_ = table.Append("Name", domain.Name)
	_ = table.Append("GUID", domain.GUID)
	_ = table.Append("Internal", strconv.FormatBool(domain.Internal))
	_ = table.Append("Protocols", strings.Join(domain.SupportedProtocols, ", "))
	_ = table.Append("Created", domain.CreatedAt.Format(TimeFormatDisplay))
	_ = table.Append("Updated", domain.UpdatedAt.Format(TimeFormatDisplay))
}

func addDomainOptionalInfo(table *tablewriter.Table, domain *capi.Domain) {
	if domain.RouterGroup != nil {
		_ = table.Append("Router Group", domain.RouterGroup.GUID)
	}

	if domain.EnforceRoutePolicies {
		_ = table.Append("Enforce Route Policies", True)
		_ = table.Append("Route Policies Scope", string(domain.RoutePoliciesScope))
	}
}

func addDomainOrganizationInfo(ctx context.Context, client capi.Client, table *tablewriter.Table, domain *capi.Domain) {
	addDomainOwnerOrg(ctx, client, table, domain)
	addDomainSharedOrgs(ctx, client, table, domain)
}

func addDomainOwnerOrg(ctx context.Context, client capi.Client, table *tablewriter.Table, domain *capi.Domain) {
	if domain.Relationships.Organization == nil || domain.Relationships.Organization.Data == nil {
		_ = table.Append("Organization", "shared (platform)")

		return
	}

	org, _ := client.Organizations().Get(ctx, domain.Relationships.Organization.Data.GUID)
	if org != nil {
		_ = table.Append("Organization", org.Name)
	}
}

func addDomainSharedOrgs(ctx context.Context, client capi.Client, table *tablewriter.Table, domain *capi.Domain) {
	if domain.Relationships.SharedOrganizations == nil || len(domain.Relationships.SharedOrganizations.Data) == 0 {
		return
	}

	var sharedOrgs []string

	for _, orgData := range domain.Relationships.SharedOrganizations.Data {
		org, _ := client.Organizations().Get(ctx, orgData.GUID)
		if org != nil {
			sharedOrgs = append(sharedOrgs, org.Name)
		}
	}

	if len(sharedOrgs) > 0 {
		_ = table.Append("Shared With", strings.Join(sharedOrgs, ", "))
	}
}

func newDomainsCreateCommand() *cobra.Command {
	var (
		name                 string
		internal             bool
		enforceRoutePolicies bool
		routePoliciesScope   string
		routerGroup          string
		orgName              string
		labels               map[string]string
	)

	cmd := &cobra.Command{
		Use:   Create,
		Short: "Create a domain",
		Long:  "Create a new Cloud Foundry domain",
		RunE:  runDomainsCreate,
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "domain name (required)")
	cmd.Flags().BoolVar(&internal, "internal", false, "create as internal domain")
	cmd.Flags().BoolVar(&enforceRoutePolicies, "enforce-route-policies", false, "create as identity-aware domain (experimental; immutable)")
	cmd.Flags().StringVar(&routePoliciesScope, "route-policies-scope", "", "allowed-caller boundary: any, org, or space (required with --enforce-route-policies)")
	cmd.Flags().StringVar(&routerGroup, "router-group", "", "router group for TCP domains")
	cmd.Flags().StringVarP(&orgName, "org", "o", "", "organization name for private domains")
	cmd.Flags().StringToStringVar(&labels, "labels", nil, "labels to apply (key=value)")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func runDomainsCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		return ErrDomainNameRequired
	}

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	createReq, err := buildDomainCreateRequest(ctx, client, cmd)
	if err != nil {
		return err
	}

	domain, err := client.Domains().Create(ctx, createReq)
	if err != nil {
		return fmt.Errorf("failed to create domain: %w", err)
	}

	displayCreatedDomain(domain)

	return nil
}

func buildDomainCreateRequest(ctx context.Context, client capi.Client, cmd *cobra.Command) (*capi.DomainCreateRequest, error) {
	name, _ := cmd.Flags().GetString("name")
	internal, _ := cmd.Flags().GetBool("internal")
	routerGroup, _ := cmd.Flags().GetString("router-group")
	orgName, _ := cmd.Flags().GetString("org")
	labels, _ := cmd.Flags().GetStringToString("labels")

	createReq := &capi.DomainCreateRequest{
		Name:     name,
		Internal: &internal,
	}

	enforceRoutePolicies, _ := cmd.Flags().GetBool("enforce-route-policies")
	routePoliciesScope, _ := cmd.Flags().GetString("route-policies-scope")

	if routePoliciesScope != "" && !enforceRoutePolicies {
		return nil, ErrScopeRequiresEnforce
	}

	if enforceRoutePolicies {
		if internal {
			return nil, ErrEnforceIncompatibleInternal
		}

		if routePoliciesScope == "" {
			return nil, ErrScopeRequiredWithEnforce
		}

		scope := capi.RoutePoliciesScope(routePoliciesScope)
		if scope != capi.RoutePoliciesScopeAny && scope != capi.RoutePoliciesScopeOrg && scope != capi.RoutePoliciesScopeSpace {
			return nil, ErrInvalidRoutePoliciesScope
		}

		createReq.EnforceRoutePolicies = &enforceRoutePolicies
		createReq.RoutePoliciesScope = &scope
	}

	if routerGroup != "" {
		createReq.RouterGroup = &routerGroup
	}

	if orgName != "" {
		orgRelationship, err := findOrganizationForDomain(ctx, client, orgName)
		if err != nil {
			return nil, err
		}

		createReq.Relationships = &capi.DomainRelationships{
			Organization: orgRelationship,
		}
	}

	if labels != nil {
		createReq.Metadata = &capi.Metadata{
			Labels: capi.StringMap(labels),
		}
	}

	return createReq, nil
}

func findOrganizationForDomain(ctx context.Context, client capi.Client, orgName string) (*capi.Relationship, error) {
	params := capi.NewQueryParams()
	params.WithFilter("names", orgName)

	orgs, err := client.Organizations().List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find organization: %w", err)
	}

	if len(orgs.Resources) == 0 {
		return nil, fmt.Errorf("organization '%s': %w", orgName, ErrOrganizationNotFound)
	}

	return &capi.Relationship{
		Data: &capi.RelationshipData{GUID: orgs.Resources[0].GUID},
	}, nil
}

func displayCreatedDomain(domain *capi.Domain) {
	_, _ = fmt.Fprintf(os.Stdout, "Successfully created domain '%s'\n", domain.Name)
	_, _ = fmt.Fprintf(os.Stdout, "  GUID: %s\n", domain.GUID)
	_, _ = fmt.Fprintf(os.Stdout, "  Type: %s\n", func() string {
		if domain.Internal {
			return "internal"
		}

		return "shared"
	}())
}

func newDomainsUpdateCommand() *cobra.Command {
	var labels map[string]string

	cmd := &cobra.Command{
		Use:   "update DOMAIN_NAME_OR_GUID",
		Short: "Update a domain",
		Long:  "Update domain metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find domain
			var domainGUID string

			domain, err := client.Domains().Get(ctx, nameOrGUID)
			if err != nil {
				// Try by name
				params := capi.NewQueryParams()
				params.WithFilter("names", nameOrGUID)

				domains, err := client.Domains().List(ctx, params)
				if err != nil {
					return fmt.Errorf("failed to find domain: %w", err)
				}

				if len(domains.Resources) == 0 {
					return fmt.Errorf("domain '%s': %w", nameOrGUID, ErrDomainNotFound)
				}

				domain = &domains.Resources[0]
			}

			domainGUID = domain.GUID

			// Build update request
			updateReq := &capi.DomainUpdateRequest{}

			if labels != nil {
				updateReq.Metadata = &capi.Metadata{
					Labels: capi.StringMap(labels),
				}
			}

			updatedDomain, err := client.Domains().Update(ctx, domainGUID, updateReq)
			if err != nil {
				return fmt.Errorf("failed to update domain: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully updated domain '%s'\n", updatedDomain.Name)

			return nil
		},
	}

	cmd.Flags().StringToStringVar(&labels, "labels", nil, "labels to apply (key=value)")

	return cmd
}

func newDomainsDeleteCommand() *cobra.Command {
	config := DeleteConfig{
		Use:         "delete DOMAIN_NAME_OR_GUID",
		Short:       "Delete a domain",
		Long:        "Delete a Cloud Foundry domain",
		EntityType:  "domain",
		GetResource: CreateDomainDeleteResourceFunc(),
		DeleteFunc: func(ctx context.Context, client any, guid string) (*string, error) {
			capiClient, ok := client.(capi.Client)
			if !ok {
				return nil, constants.ErrInvalidClientType
			}

			job, err := capiClient.Domains().Delete(ctx, guid)
			if err != nil {
				return nil, fmt.Errorf("failed to delete domain: %w", err)
			}

			if job != nil {
				return &job.GUID, nil
			}

			return nil, nil
		},
	}

	return createDeleteCommand(config)
}

func newDomainsShareCommand() *cobra.Command {
	var orgNames []string

	cmd := &cobra.Command{
		Use:   "share DOMAIN_NAME_OR_GUID",
		Short: "Share a domain with organizations",
		Long:  "Share a private domain with specified organizations",
		Args:  cobra.ExactArgs(1),
		RunE:  runDomainsShare,
	}

	cmd.Flags().StringArrayVarP(&orgNames, "orgs", "o", nil, "organizations to share with (required)")
	_ = cmd.MarkFlagRequired("orgs")

	return cmd
}

func runDomainsShare(cmd *cobra.Command, args []string) error {
	nameOrGUID := args[0]

	orgNames, _ := cmd.Flags().GetStringArray("orgs")
	if len(orgNames) == 0 {
		return ErrAtLeastOneOrgRequired
	}

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	domain, err := findDomainByNameOrGUID(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	orgGUIDs, err := findOrganizationGUIDs(ctx, client, orgNames)
	if err != nil {
		return err
	}

	_, err = client.Domains().ShareWithOrganization(ctx, domain.GUID, orgGUIDs)
	if err != nil {
		return fmt.Errorf("failed to share domain: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully shared domain '%s' with organizations: %s\n", domain.Name, strings.Join(orgNames, ", "))

	return nil
}

func findDomainByNameOrGUID(ctx context.Context, client capi.Client, nameOrGUID string) (*capi.Domain, error) {
	domain, err := client.Domains().Get(ctx, nameOrGUID)
	if err != nil {
		params := capi.NewQueryParams()
		params.WithFilter("names", nameOrGUID)

		domains, err := client.Domains().List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to find domain: %w", err)
		}

		if len(domains.Resources) == 0 {
			return nil, fmt.Errorf("domain '%s': %w", nameOrGUID, ErrDomainNotFound)
		}

		domain = &domains.Resources[0]
	}

	return domain, nil
}

func findOrganizationGUIDs(ctx context.Context, client capi.Client, orgNames []string) ([]string, error) {
	orgGUIDs := make([]string, 0, len(orgNames))

	for _, orgName := range orgNames {
		params := capi.NewQueryParams()
		params.WithFilter("names", orgName)

		orgs, err := client.Organizations().List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to find organization '%s': %w", orgName, err)
		}

		if len(orgs.Resources) == 0 {
			return nil, fmt.Errorf("organization '%s': %w", orgName, ErrOrganizationNotFound)
		}

		orgGUIDs = append(orgGUIDs, orgs.Resources[0].GUID)
	}

	return orgGUIDs, nil
}

func newDomainsUnshareCommand() *cobra.Command {
	var orgName string

	cmd := &cobra.Command{
		Use:   "unshare DOMAIN_NAME_OR_GUID",
		Short: "Unshare a domain from an organization",
		Long:  "Remove sharing of a domain from a specified organization",
		Args:  cobra.ExactArgs(1),
		RunE:  runDomainsUnshare,
	}

	cmd.Flags().StringVarP(&orgName, "org", "o", "", "organization to unshare from (required)")
	_ = cmd.MarkFlagRequired("org")

	return cmd
}

func runDomainsUnshare(cmd *cobra.Command, args []string) error {
	nameOrGUID := args[0]

	orgName, _ := cmd.Flags().GetString("org")
	if orgName == "" {
		return ErrOrganizationNameRequired
	}

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	domain, err := findDomainByNameOrGUID(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	orgGUID, err := findSingleOrganizationGUID(ctx, client, orgName)
	if err != nil {
		return err
	}

	err = client.Domains().UnshareFromOrganization(ctx, domain.GUID, orgGUID)
	if err != nil {
		return fmt.Errorf("failed to unshare domain: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully unshared domain '%s' from organization '%s'\n", domain.Name, orgName)

	return nil
}

func findSingleOrganizationGUID(ctx context.Context, client capi.Client, orgName string) (string, error) {
	params := capi.NewQueryParams()
	params.WithFilter("names", orgName)

	orgs, err := client.Organizations().List(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to find organization: %w", err)
	}

	if len(orgs.Resources) == 0 {
		return "", fmt.Errorf("organization '%s': %w", orgName, ErrOrganizationNotFound)
	}

	return orgs.Resources[0].GUID, nil
}
