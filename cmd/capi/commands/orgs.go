package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fivetwenty-io/capi/v3/internal/constants"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewOrgsCommand creates the organizations command group.
func NewOrgsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "orgs",
		Aliases: []string{"organizations", "org"},
		Short:   "Manage organizations",
		Long:    "List, create, update, and delete Cloud Foundry organizations",
	}

	cmd.AddCommand(newOrgsListCommand())
	cmd.AddCommand(newOrgsGetCommand())
	cmd.AddCommand(newOrgsCreateCommand())
	cmd.AddCommand(newOrgsUpdateCommand())
	cmd.AddCommand(newOrgsDeleteCommand())
	cmd.AddCommand(newOrgsSetQuotaCommand())
	cmd.AddCommand(newOrgsListUsersCommand())
	cmd.AddCommand(newOrgsAddUserCommand())
	cmd.AddCommand(newOrgsRemoveUserCommand())
	cmd.AddCommand(newOrgsListSpacesCommand())

	return cmd
}

func newOrgsListCommand() *cobra.Command {
	var (
		allPages bool
		perPage  int
	)

	cmd := &cobra.Command{
		Use:   List,
		Short: "List organizations",
		Long:  "List all organizations the user has access to",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrgsListCommand(cmd, allPages, perPage)
		},
	}

	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "results per page")

	return cmd
}

func runOrgsListCommand(cmd *cobra.Command, allPages bool, perPage int) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	params := capi.NewQueryParams()
	params.PerPage = perPage

	orgs, err := client.Organizations().List(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list organizations: %w", err)
	}

	allOrgs := orgs.Resources
	if allPages && orgs.Pagination.TotalPages > 1 {
		moreOrgs, err := fetchAllOrgPages(ctx, client, params, orgs.Pagination.TotalPages)
		if err != nil {
			return err
		}

		allOrgs = append(allOrgs, moreOrgs...)
	}

	return outputOrganizations(allOrgs, orgs.Pagination, allPages)
}

func fetchAllOrgPages(ctx context.Context, client capi.Client, params *capi.QueryParams, totalPages int) ([]capi.Organization, error) {
	var allOrgs []capi.Organization

	for page := 2; page <= totalPages; page++ {
		params.Page = page

		moreOrgs, err := client.Organizations().List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		allOrgs = append(allOrgs, moreOrgs.Resources...)
	}

	return allOrgs, nil
}

func outputOrganizations(orgs []capi.Organization, pagination capi.Pagination, allPages bool) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return StandardJSONRenderer(orgs)
	case OutputFormatYAML:
		return StandardYAMLRenderer(orgs)
	default:
		return renderOrganizationTable(orgs, pagination, allPages)
	}
}

func renderOrganizationTable(orgs []capi.Organization, pagination capi.Pagination, allPages bool) error {
	if len(orgs) == 0 {
		_, _ = os.Stdout.WriteString("No organizations found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "GUID", "Status", "Created", "Updated")

	for _, org := range orgs {
		status := Active
		if org.Suspended {
			status = "suspended"
		}

		_ = table.Append(org.Name, org.GUID, status,
			org.CreatedAt.Format("2006-01-02"),
			org.UpdatedAt.Format("2006-01-02"))
	}

	_ = table.Render()

	if !allPages && pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", pagination.TotalPages)
	}

	return nil
}

func newOrgsGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get ORG_NAME_OR_GUID",
		Short: "Get organization details",
		Long:  "Display detailed information about a specific organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrgsGetCommand(cmd, args[0])
		},
	}
}

func runOrgsGetCommand(cmd *cobra.Command, nameOrGUID string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	org, err := findOrganizationByNameOrGUID(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	return outputOrganizationDetails(org)
}

func findOrganizationByNameOrGUID(ctx context.Context, client capi.Client, nameOrGUID string) (*capi.Organization, error) {
	orgsClient := client.Organizations()

	org, err := orgsClient.Get(ctx, nameOrGUID)
	if err == nil {
		return org, nil
	}

	params := capi.NewQueryParams()
	params.WithFilter("names", nameOrGUID)

	orgs, err := orgsClient.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find organization: %w", err)
	}

	if len(orgs.Resources) == 0 {
		return nil, fmt.Errorf("organization '%s': %w", nameOrGUID, ErrOrganizationNotFound)
	}

	return &orgs.Resources[0], nil
}

func outputOrganizationDetails(org *capi.Organization) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return StandardJSONRenderer(org)
	case OutputFormatYAML:
		return StandardYAMLRenderer(org)
	default:
		return renderOrganizationDetailsTable(org)
	}
}

func renderOrganizationDetailsTable(org *capi.Organization) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	_ = table.Append("Name", org.Name)
	_ = table.Append("GUID", org.GUID)

	status := Active
	if org.Suspended {
		status = "suspended"
	}

	_ = table.Append("Status", status)
	_ = table.Append("Created", org.CreatedAt.Format(TimeFormatDisplay))
	_ = table.Append("Updated", org.UpdatedAt.Format(TimeFormatDisplay))

	_, _ = os.Stdout.WriteString("Organization details:\n\n")

	_ = table.Render()

	renderMetadataTables(org.Metadata.Labels, org.Metadata.Annotations)

	return nil
}

func renderMetadataTables(labels, annotations map[string]string) {
	if len(labels) > 0 {
		_, _ = os.Stdout.WriteString("\nLabels:\n")

		labelTable := tablewriter.NewWriter(os.Stdout)
		labelTable.Header("Key", "Value")

		for k, v := range labels {
			_ = labelTable.Append(k, v)
		}

		_ = labelTable.Render()
	}

	if len(annotations) > 0 {
		_, _ = os.Stdout.WriteString("\nAnnotations:\n")

		annotationTable := tablewriter.NewWriter(os.Stdout)
		annotationTable.Header("Key", "Value")

		for k, v := range annotations {
			_ = annotationTable.Append(k, v)
		}

		_ = annotationTable.Render()
	}
}

func newOrgsCreateCommand() *cobra.Command {
	return createGenericCreateCommand(CreateConfig{
		Use:        Create,
		Short:      "Create a new organization",
		Long:       "Create a new Cloud Foundry organization",
		EntityType: organizationKey,
		NameError:  ErrOrganizationNameRequired,
		CreateFunc: func(ctx context.Context, client any, name string, labels map[string]string) (string, string, error) {
			createReq := &capi.OrganizationCreateRequest{
				Name: name,
			}

			if labels != nil {
				createReq.Metadata = &capi.Metadata{
					Labels: labels,
				}
			}

			capiClient, ok := client.(capi.Client)
			if !ok {
				return "", "", constants.ErrInvalidClientType
			}

			org, err := capiClient.Organizations().Create(ctx, createReq)
			if err != nil {
				return "", "", fmt.Errorf("failed to create organization: %w", err)
			}

			return org.GUID, org.Name, nil
		},
	})
}

func newOrgsUpdateCommand() *cobra.Command {
	var (
		newName string
		labels  map[string]string
	)

	cmd := &cobra.Command{
		Use:   "update ORG_NAME_OR_GUID",
		Short: "Update an organization",
		Long:  "Update an existing Cloud Foundry organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrgsUpdateCommand(cmd, args[0], newName, labels)
		},
	}

	cmd.Flags().StringVar(&newName, "name", "", "new organization name")
	cmd.Flags().StringToStringVar(&labels, "labels", nil, "labels to apply (key=value)")

	return cmd
}

func runOrgsUpdateCommand(cmd *cobra.Command, nameOrGUID, newName string, labels map[string]string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	org, err := findOrganizationByNameOrGUID(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	updateRequest := buildOrgUpdateRequest(newName, labels)
	if updateRequest == nil {
		_, _ = os.Stdout.WriteString("No updates specified\n")

		return nil
	}

	updatedOrg, err := client.Organizations().Update(ctx, org.GUID, updateRequest)
	if err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}

	return outputOrganizationUpdateResult(updatedOrg)
}

func buildOrgUpdateRequest(newName string, labels map[string]string) *capi.OrganizationUpdateRequest {
	updateRequest := &capi.OrganizationUpdateRequest{}
	hasUpdate := false

	if newName != "" {
		updateRequest.Name = &newName
		hasUpdate = true
	}

	if len(labels) > 0 {
		updateRequest.Metadata = &capi.Metadata{
			Labels: labels,
		}
		hasUpdate = true
	}

	if !hasUpdate {
		return nil
	}

	return updateRequest
}

func outputOrganizationUpdateResult(org *capi.Organization) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return StandardJSONRenderer(org)
	case OutputFormatYAML:
		return StandardYAMLRenderer(org)
	default:
		_, _ = fmt.Fprintf(os.Stdout, "Successfully updated organization '%s' (GUID: %s)\n", org.Name, org.GUID)

		return nil
	}
}

func newOrgsDeleteCommand() *cobra.Command {
	return createDeleteCommand(DeleteConfig{
		Use:         "delete ORG_NAME_OR_GUID",
		Short:       "Delete an organization",
		Long:        "Delete a Cloud Foundry organization",
		EntityType:  organizationKey,
		GetResource: CreateOrganizationDeleteResourceFunc(),
		DeleteFunc: func(ctx context.Context, client any, guid string) (*string, error) {
			capiClient, ok := client.(capi.Client)
			if !ok {
				return nil, constants.ErrInvalidClientType
			}

			job, err := capiClient.Organizations().Delete(ctx, guid)
			if err != nil {
				return nil, fmt.Errorf("failed to delete organization: %w", err)
			}

			if job != nil {
				return &job.GUID, nil
			}

			return nil, nil
		},
	})
}

func newOrgsSetQuotaCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set-quota ORG_NAME_OR_GUID QUOTA_NAME_OR_GUID",
		Short: "Set organization quota",
		Long:  "Assign a quota to an organization",
		Args:  cobra.ExactArgs(constants.MinimumArgumentCount),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrgsSetQuotaCommand(cmd, args[0], args[1])
		},
	}
}

func runOrgsSetQuotaCommand(cmd *cobra.Command, orgNameOrGUID, quotaNameOrGUID string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	org, err := findOrganizationByNameOrGUID(ctx, client, orgNameOrGUID)
	if err != nil {
		return err
	}

	quota, err := findOrganizationQuota(ctx, client, quotaNameOrGUID)
	if err != nil {
		return err
	}

	err = applyOrganizationQuota(ctx, client, org.GUID, quota.GUID)
	if err != nil {
		return fmt.Errorf("failed to apply quota: %w", err)
	}

	return outputOrganizationQuotaResult(org.Name, quota.Name)
}

func findOrganizationQuota(ctx context.Context, client capi.Client, nameOrGUID string) (*capi.OrganizationQuota, error) {
	quotasClient := client.OrganizationQuotas()

	quota, err := quotasClient.Get(ctx, nameOrGUID)
	if err == nil {
		return quota, nil
	}

	params := capi.NewQueryParams()
	params.WithFilter("names", nameOrGUID)

	quotas, err := quotasClient.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find organization quota: %w", err)
	}

	if len(quotas.Resources) == 0 {
		return nil, fmt.Errorf("organization quota '%s': %w", nameOrGUID, ErrOrganizationQuotaNotFound)
	}

	return &quotas.Resources[0], nil
}

func applyOrganizationQuota(ctx context.Context, client capi.Client, orgGUID, quotaGUID string) error {
	_, err := client.OrganizationQuotas().ApplyToOrganizations(ctx, quotaGUID, []string{orgGUID})
	if err != nil {
		return fmt.Errorf("failed to apply organization quota: %w", err)
	}

	return nil
}

func outputOrganizationQuotaResult(orgName, quotaName string) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		result := map[string]string{
			organizationKey: orgName,
			"quota":         quotaName,
			keyStatus:       Applied,
		}

		return StandardJSONRenderer(result)
	case OutputFormatYAML:
		result := map[string]string{
			organizationKey: orgName,
			"quota":         quotaName,
			keyStatus:       Applied,
		}

		return StandardYAMLRenderer(result)
	default:
		_, _ = fmt.Fprintf(os.Stdout, "Successfully applied quota '%s' to organization '%s'\n", quotaName, orgName)

		return nil
	}
}

func newOrgsListUsersCommand() *cobra.Command {
	var (
		allPages bool
		perPage  int
		role     string
	)

	cmd := &cobra.Command{
		Use:   "list-users ORG_NAME_OR_GUID",
		Short: "List organization users",
		Long:  "List all users in an organization with their roles",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrgsListUsersCommand(cmd, args[0], allPages, perPage, role)
		},
	}

	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "results per page")
	cmd.Flags().StringVar(&role, "role", "", "filter by role type")

	return cmd
}

func runOrgsListUsersCommand(cmd *cobra.Command, orgNameOrGUID string, allPages bool, perPage int, roleFilter string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	org, err := findOrganizationByNameOrGUID(ctx, client, orgNameOrGUID)
	if err != nil {
		return err
	}

	params := buildOrgUsersQueryParams(org.GUID, perPage, roleFilter)

	roles, err := client.Roles().List(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list organization users: %w", err)
	}

	allRoles := roles.Resources
	if allPages && roles.Pagination.TotalPages > 1 {
		moreRoles, err := fetchAllOrgUserPages(ctx, client, params, roles.Pagination.TotalPages)
		if err != nil {
			return err
		}

		allRoles = append(allRoles, moreRoles...)
	}

	userMap := buildUserMap(ctx, client, allRoles)

	return outputOrganizationUsers(userMap, roles.Pagination, allPages)
}

func buildOrgUsersQueryParams(orgGUID string, perPage int, roleFilter string) *capi.QueryParams {
	params := capi.NewQueryParams()
	params.PerPage = perPage
	params.WithFilter("organization_guids", orgGUID)
	params.WithInclude("user")

	if roleFilter != "" {
		params.WithFilter("types", roleFilter)
	}

	return params
}

func fetchAllOrgUserPages(ctx context.Context, client capi.Client, params *capi.QueryParams, totalPages int) ([]capi.Role, error) {
	var allRoles []capi.Role

	for page := 2; page <= totalPages; page++ {
		params.Page = page

		moreRoles, err := client.Roles().List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		allRoles = append(allRoles, moreRoles.Resources...)
	}

	return allRoles, nil
}

func buildUserMap(ctx context.Context, client capi.Client, roles []capi.Role) map[string]*UserRoleInfo {
	userMap := make(map[string]*UserRoleInfo)

	for _, role := range roles {
		userGUID := ""
		if role.Relationships.User.Data != nil {
			userGUID = role.Relationships.User.Data.GUID
		}

		if userGUID == "" {
			continue
		}

		if _, exists := userMap[userGUID]; !exists {
			username := getUsernameFromIncluded(role, userGUID)
			if username == "" {
				user, err := client.Users().Get(ctx, userGUID)
				if err == nil {
					username = user.Username
				}
			}

			userMap[userGUID] = &UserRoleInfo{
				Username: username,
				GUID:     userGUID,
				Roles:    []string{},
			}
		}

		userMap[userGUID].Roles = append(userMap[userGUID].Roles, role.Type)
	}

	return userMap
}

func getUsernameFromIncluded(role capi.Role, userGUID string) string {
	// Note: The current Role type doesn't have an Included field.
	// This would need to be implemented if/when included resources are supported.
	// For now, return empty string to trigger the fallback to fetching the user.
	return ""
}

func outputOrganizationUsers(userMap map[string]*UserRoleInfo, pagination capi.Pagination, allPages bool) error {
	output := viper.GetString("output")

	users := make([]*UserRoleInfo, 0, len(userMap))
	for _, user := range userMap {
		users = append(users, user)
	}

	switch output {
	case OutputFormatJSON:
		return StandardJSONRenderer(users)
	case OutputFormatYAML:
		return StandardYAMLRenderer(users)
	default:
		return renderOrganizationUsersTable(users, pagination, allPages)
	}
}

func renderOrganizationUsersTable(users []*UserRoleInfo, pagination capi.Pagination, allPages bool) error {
	if len(users) == 0 {
		_, _ = os.Stdout.WriteString("No users found in this organization\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Username", "GUID", "Roles")

	for _, user := range users {
		rolesStr := strings.Join(user.Roles, ", ")
		_ = table.Append(user.Username, user.GUID, rolesStr)
	}

	_ = table.Render()

	if !allPages && pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", pagination.TotalPages)
	}

	return nil
}

// RoleOrganizationUser is the default organization role assigned to a user
// added to an organization.
const RoleOrganizationUser = "organization_user"

type UserRoleInfo struct {
	Username string   `json:"username" yaml:"username"`
	GUID     string   `json:"guid"     yaml:"guid"`
	Roles    []string `json:"roles"    yaml:"roles"`
}

func newOrgsAddUserCommand() *cobra.Command {
	roleContext := RoleContext{
		ResourceType:   organizationKey,
		DefaultRole:    RoleOrganizationUser,
		ValidRoles:     []string{RoleOrganizationUser, "organization_manager", "organization_auditor", "organization_billing_manager"},
		SuccessMessage: "Successfully added user to organization with role '%s'\n",
	}

	return CreateRoleCommand(roleOperationAddUser, organizationKey, roleContext)()
}

func newOrgsRemoveUserCommand() *cobra.Command {
	roleContext := RoleContext{
		ResourceType:   organizationKey,
		DefaultRole:    "",
		ValidRoles:     []string{RoleOrganizationUser, "organization_manager", "organization_auditor", "organization_billing_manager"},
		SuccessMessage: "Successfully removed user role from organization\n",
	}

	return CreateRoleCommand("remove-user", organizationKey, roleContext)()
}

func newOrgsListSpacesCommand() *cobra.Command {
	var (
		allPages bool
		perPage  int
	)

	cmd := &cobra.Command{
		Use:   "list-spaces ORG_NAME_OR_GUID",
		Short: "List organization spaces",
		Long:  "List all spaces in an organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrgsListSpacesCommand(cmd, args[0], allPages, perPage)
		},
	}

	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "results per page")

	return cmd
}

func runOrgsListSpacesCommand(cmd *cobra.Command, orgNameOrGUID string, allPages bool, perPage int) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	org, err := findOrganizationByNameOrGUID(ctx, client, orgNameOrGUID)
	if err != nil {
		return err
	}

	params := capi.NewQueryParams()
	params.PerPage = perPage
	params.WithFilter("organization_guids", org.GUID)

	spaces, err := client.Spaces().List(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list spaces: %w", err)
	}

	allSpaces := spaces.Resources
	if allPages && spaces.Pagination.TotalPages > 1 {
		moreSpaces, err := fetchAllSpacePages(ctx, client, params, spaces.Pagination.TotalPages)
		if err != nil {
			return err
		}

		allSpaces = append(allSpaces, moreSpaces...)
	}

	return outputOrganizationSpaces(allSpaces, spaces.Pagination, allPages)
}

func fetchAllSpacePages(ctx context.Context, client capi.Client, params *capi.QueryParams, totalPages int) ([]capi.Space, error) {
	var allSpaces []capi.Space

	for page := 2; page <= totalPages; page++ {
		params.Page = page

		moreSpaces, err := client.Spaces().List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		allSpaces = append(allSpaces, moreSpaces.Resources...)
	}

	return allSpaces, nil
}

func outputOrganizationSpaces(spaces []capi.Space, pagination capi.Pagination, allPages bool) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return StandardJSONRenderer(spaces)
	case OutputFormatYAML:
		return StandardYAMLRenderer(spaces)
	default:
		return renderOrganizationSpacesTable(spaces, pagination, allPages)
	}
}

func renderOrganizationSpacesTable(spaces []capi.Space, pagination capi.Pagination, allPages bool) error {
	if len(spaces) == 0 {
		_, _ = os.Stdout.WriteString("No spaces found in this organization\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "GUID", "Created", "Updated")

	for _, space := range spaces {
		_ = table.Append(space.Name, space.GUID,
			space.CreatedAt.Format("2006-01-02"),
			space.UpdatedAt.Format("2006-01-02"))
	}

	_ = table.Render()

	if !allPages && pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", pagination.TotalPages)
	}

	return nil
}
