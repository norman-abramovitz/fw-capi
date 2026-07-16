package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"os"
	"path/filepath"
	"strings"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	statusDisabled = "disabled"
	statusEnabled  = "enabled"
)

func validateFilePathSpaces(filePath string) error {
	// Clean the path to resolve any path traversal attempts
	cleanPath := filepath.Clean(filePath)

	// Check for path traversal attempts
	if filepath.IsAbs(filePath) {
		// Allow absolute paths but ensure they're clean
		if cleanPath != filePath {
			return ErrDirectoryTraversalDetected
		}
	} else {
		// For relative paths, ensure they don't escape the current directory
		if len(cleanPath) > 0 && cleanPath[0] == '.' && len(cleanPath) > 1 && cleanPath[1] == '.' {
			return ErrDirectoryTraversalDetected
		}
	}

	return nil
}

// NewSpacesCommand creates the spaces command group.
func NewSpacesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "spaces",
		Aliases: []string{"space"},
		Short:   "Manage spaces",
		Long:    "List and manage Cloud Foundry spaces",
	}

	cmd.AddCommand(newSpacesListCommand())
	cmd.AddCommand(newSpacesGetCommand())
	cmd.AddCommand(newSpacesCreateCommand())
	cmd.AddCommand(newSpacesUpdateCommand())
	cmd.AddCommand(newSpacesDeleteCommand())
	cmd.AddCommand(newSpacesFeaturesCommand())
	cmd.AddCommand(newSpacesSetQuotaCommand())
	cmd.AddCommand(newSpacesListUsersCommand())
	cmd.AddCommand(newSpacesSetRoleCommand())
	cmd.AddCommand(newSpacesUnsetRoleCommand())
	cmd.AddCommand(newSpacesListAppsCommand())
	cmd.AddCommand(newSpacesListServicesCommand())
	cmd.AddCommand(newSpacesApplyManifestCommand())

	return cmd
}

// findOrganizationGUID resolves an organization name to its GUID.
func findOrganizationGUID(ctx context.Context, client capi.Client, orgName string) (string, error) {
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

// fetchAllSpacePagesForList fetches all pages of spaces from the API for list command.
func fetchAllSpacePagesForList(ctx context.Context, client capi.Client, params *capi.QueryParams, firstPage *capi.ListResponse[capi.Space]) ([]capi.Space, error) {
	allSpaces := firstPage.Resources

	if firstPage.Pagination.TotalPages > 1 {
		for page := 2; page <= firstPage.Pagination.TotalPages; page++ {
			params.Page = page

			moreSpaces, err := client.Spaces().List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
			}

			allSpaces = append(allSpaces, moreSpaces.Resources...)
		}
	}

	return allSpaces, nil
}

// outputSpacesAsJSON outputs spaces in JSON format.
func outputSpacesAsJSON(spaces []capi.Space) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(spaces)
	if err != nil {
		return fmt.Errorf("encoding spaces to JSON: %w", err)
	}

	return nil
}

// outputSpacesAsYAML outputs spaces in YAML format.
func outputSpacesAsYAML(spaces []capi.Space) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(spaces)
	if err != nil {
		return fmt.Errorf("encoding spaces to YAML: %w", err)
	}

	return nil
}

// outputSpacesAsTable outputs spaces in table format.
func outputSpacesAsTable(ctx context.Context, client capi.Client, spaces []capi.Space, allPages bool, pagination capi.Pagination) error {
	if len(spaces) == 0 {
		_, _ = os.Stdout.WriteString("No spaces found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "GUID", "Organization", "Created", "Updated")

	for _, space := range spaces {
		// Get org name if available
		orgName := ""

		if space.Relationships.Organization.Data != nil {
			org, _ := client.Organizations().Get(ctx, space.Relationships.Organization.Data.GUID)
			if org != nil {
				orgName = org.Name
			}
		}

		_ = table.Append(space.Name, space.GUID, orgName,
			space.CreatedAt.Format("2006-01-02"),
			space.UpdatedAt.Format("2006-01-02"))
	}

	_ = table.Render()

	if !allPages && pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", pagination.TotalPages)
	}

	return nil
}

func newSpacesListCommand() *cobra.Command {
	var (
		orgName  string
		allPages bool
		perPage  int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List spaces",
		Long:  "List all spaces the user has access to",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()
			params := capi.NewQueryParams()
			params.PerPage = perPage

			// Filter by organization if specified
			if orgName != "" {
				orgGUID, err := findOrganizationGUID(ctx, client, orgName)
				if err != nil {
					return err
				}

				params.WithFilter("organization_guids", orgGUID)
			} else if orgGUID := viper.GetString("organization_guid"); orgGUID != "" {
				// Use targeted organization
				params.WithFilter("organization_guids", orgGUID)
			}

			spaces, err := client.Spaces().List(ctx, params)
			if err != nil {
				return fmt.Errorf("failed to list spaces: %w", err)
			}

			// Fetch all pages if requested
			allSpaces := spaces.Resources
			if allPages {
				allSpaces, err = fetchAllSpacePagesForList(ctx, client, params, spaces)
				if err != nil {
					return err
				}
			}

			// Output results
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				return outputSpacesAsJSON(allSpaces)
			case OutputFormatYAML:
				return outputSpacesAsYAML(allSpaces)
			default:
				return outputSpacesAsTable(ctx, client, allSpaces, allPages, spaces.Pagination)
			}
		},
	}

	cmd.Flags().StringVarP(&orgName, "org", "o", "", "filter by organization name")
	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "results per page")

	return cmd
}

// findSpaceByNameOrGUID finds a space by GUID or name.
func findSpaceByNameOrGUID(ctx context.Context, client capi.Client, nameOrGUID string) (*capi.Space, error) {
	spacesClient := client.Spaces()

	// Try to get by GUID first
	space, err := spacesClient.Get(ctx, nameOrGUID)
	if err == nil {
		return space, nil
	}

	// If not found by GUID, try by name
	params := capi.NewQueryParams()
	params.WithFilter("names", nameOrGUID)

	spaces, err := spacesClient.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}

	if len(spaces.Resources) == 0 {
		return nil, fmt.Errorf("space '%s': %w", nameOrGUID, ErrSpaceNotFound)
	}

	return &spaces.Resources[0], nil
}

// outputSpaceAsJSON outputs a single space in JSON format.
func outputSpaceAsJSON(space *capi.Space) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(space)
	if err != nil {
		return fmt.Errorf("encoding space to JSON: %w", err)
	}

	return nil
}

// outputSpaceAsYAML outputs a single space in YAML format.
func outputSpaceAsYAML(space *capi.Space) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(space)
	if err != nil {
		return fmt.Errorf("encoding space to YAML: %w", err)
	}

	return nil
}

// outputSpaceAsDetailedTable outputs a single space as a detailed table.
func outputSpaceAsDetailedTable(ctx context.Context, client capi.Client, space *capi.Space) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	status := Active
	if space.Suspended {
		status = Suspended
	}

	_ = table.Append("Name", space.Name)
	_ = table.Append("GUID", space.GUID)
	_ = table.Append("Status", status)
	_ = table.Append("Created", space.CreatedAt.Format(TimeFormatDisplay))
	_ = table.Append("Updated", space.UpdatedAt.Format(TimeFormatDisplay))

	if space.Relationships.Organization.Data != nil {
		org, _ := client.Organizations().Get(ctx, space.Relationships.Organization.Data.GUID)
		if org != nil {
			_ = table.Append("Organization", fmt.Sprintf("%s (%s)", org.Name, org.GUID))
		}
	}

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	// Output labels if present
	err = outputMetadataTable("Labels", space.Metadata.Labels)
	if err != nil {
		return err
	}

	// Output annotations if present
	err = outputMetadataTable("Annotations", space.Metadata.Annotations)
	if err != nil {
		return err
	}

	return nil
}

// outputMetadataTable outputs a metadata table for labels or annotations.
func outputMetadataTable(title string, metadata map[string]string) error {
	if len(metadata) == 0 {
		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "\n%s:\n", title)

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Key", "Value")

	for k, v := range metadata {
		_ = table.Append(k, v)
	}

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	return nil
}

func newSpacesGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get SPACE_NAME_OR_GUID",
		Short: "Get space details",
		Long:  "Display detailed information about a specific space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			space, err := findSpaceByNameOrGUID(ctx, client, nameOrGUID)
			if err != nil {
				return err
			}

			// Output results
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				return outputSpaceAsJSON(space)
			case OutputFormatYAML:
				return outputSpaceAsYAML(space)
			default:
				return outputSpaceAsDetailedTable(ctx, client, space)
			}
		},
	}
}

func newSpacesCreateCommand() *cobra.Command {
	var (
		name      string
		orgName   string
		suspended bool
		labels    map[string]string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new space",
		Long:  "Create a new Cloud Foundry space",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return ErrSpaceNameRequired
			}

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find organization
			var orgGUID string

			if orgName != "" {
				params := capi.NewQueryParams()
				params.WithFilter("names", orgName)

				orgs, err := client.Organizations().List(ctx, params)
				if err != nil {
					return fmt.Errorf("failed to find organization: %w", err)
				}

				if len(orgs.Resources) == 0 {
					return fmt.Errorf("organization '%s': %w", orgName, ErrOrganizationNotFound)
				}

				orgGUID = orgs.Resources[0].GUID
			} else {
				return ErrOrganizationRequired
			}

			createReq := &capi.SpaceCreateRequest{
				Name: name,
				Relationships: capi.SpaceRelationships{
					Organization: capi.Relationship{
						Data: &capi.RelationshipData{GUID: orgGUID},
					},
				},
			}

			if suspended {
				createReq.Suspended = &suspended
			}

			if labels != nil {
				createReq.Metadata = &capi.Metadata{
					Labels: labels,
				}
			}

			space, err := client.Spaces().Create(ctx, createReq)
			if err != nil {
				return fmt.Errorf("failed to create space: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully created space '%s' with GUID %s\n", space.Name, space.GUID)

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "space name (required)")
	cmd.Flags().StringVarP(&orgName, "org", "o", "", "organization name (required)")
	cmd.Flags().BoolVar(&suspended, "suspended", false, "create the space in a suspended state (admin only)")
	cmd.Flags().StringToStringVar(&labels, "labels", nil, "labels to apply (key=value)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("org")

	return cmd
}

func newSpacesUpdateCommand() *cobra.Command {
	config := UpdateConfig{
		Use:         "update SPACE_NAME_OR_GUID",
		Short:       "Update a space",
		Long:        "Update an existing Cloud Foundry space",
		EntityType:  "space",
		GetResource: CreateSpaceUpdateResourceFunc(),
		UpdateFunc: func(ctx context.Context, client interface{}, guid, newName string, labels map[string]string) (string, error) {
			capiClient, ok := client.(capi.Client)
			if !ok {
				return "", constants.ErrInvalidClientType
			}

			spacesClient := capiClient.Spaces()

			updateReq := &capi.SpaceUpdateRequest{}

			if newName != "" {
				updateReq.Name = &newName
			}

			if labels != nil {
				updateReq.Metadata = &capi.Metadata{
					Labels: labels,
				}
			}

			updatedSpace, err := spacesClient.Update(ctx, guid, updateReq)
			if err != nil {
				return "", fmt.Errorf("failed to update space: %w", err)
			}

			return updatedSpace.Name, nil
		},
	}

	return createUpdateCommand(config)
}

func newSpacesDeleteCommand() *cobra.Command {
	return createDeleteCommand(DeleteConfig{
		Use:         "delete SPACE_NAME_OR_GUID",
		Short:       "Delete a space",
		Long:        "Delete a Cloud Foundry space",
		EntityType:  "space",
		GetResource: CreateSpaceDeleteResourceFunc(),
		DeleteFunc: func(ctx context.Context, client interface{}, guid string) (*string, error) {
			capiClient, ok := client.(capi.Client)
			if !ok {
				return nil, constants.ErrInvalidClientType
			}

			job, err := capiClient.Spaces().Delete(ctx, guid)
			if err != nil {
				return nil, fmt.Errorf("failed to delete space: %w", err)
			}

			if job != nil {
				return &job.GUID, nil
			}

			return nil, nil
		},
	})
}

func newSpacesSetQuotaCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set-quota SPACE_NAME_OR_GUID QUOTA_NAME_OR_GUID",
		Short: "Set space quota",
		Long:  "Assign a quota to a space",
		Args:  cobra.ExactArgs(constants.TwoArgumentsMax),
		RunE:  runSpacesSetQuota,
	}
}

func runSpacesSetQuota(cmd *cobra.Command, args []string) error {
	spaceNameOrGUID := args[0]
	quotaNameOrGUID := args[1]

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()
	quotaClient := client.SpaceQuotas()
	spacesClient := client.Spaces()

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
			return fmt.Errorf("failed to find space '%s': %w", spaceNameOrGUID, err)
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
			return fmt.Errorf("failed to find space quota '%s': %w", quotaNameOrGUID, err)
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

	// Apply quota to space
	_, err = quotaClient.ApplyToSpaces(ctx, quotaGUID, []string{spaceGUID})
	if err != nil {
		return fmt.Errorf("failed to set quota for space: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully set quota '%s' for space '%s'\n", quotaName, spaceName)

	return nil
}

// fetchAllRolePagesForSpace fetches all pages of roles from the API.
func fetchAllRolePagesForSpace(ctx context.Context, client capi.Client, params *capi.QueryParams, firstPage *capi.ListResponse[capi.Role]) ([]capi.Role, error) {
	allRoles := firstPage.Resources

	if firstPage.Pagination.TotalPages > 1 {
		for page := 2; page <= firstPage.Pagination.TotalPages; page++ {
			params.Page = page

			moreRoles, err := client.Roles().List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
			}

			allRoles = append(allRoles, moreRoles.Resources...)
		}
	}

	return allRoles, nil
}

// outputRolesAsJSON outputs roles in JSON format.
func outputRolesAsJSON(roles []capi.Role) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(roles)
	if err != nil {
		return fmt.Errorf("encoding roles to JSON: %w", err)
	}

	return nil
}

// outputRolesAsYAML outputs roles in YAML format.
func outputRolesAsYAML(roles []capi.Role) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(roles)
	if err != nil {
		return fmt.Errorf("encoding roles to YAML: %w", err)
	}

	return nil
}

// groupRolesByUser groups roles by user GUID.
func groupRolesByUser(roles []capi.Role) map[string][]string {
	userRoles := make(map[string][]string)

	for _, role := range roles {
		if role.Relationships.User.Data != nil {
			userGUID := role.Relationships.User.Data.GUID
			userRoles[userGUID] = append(userRoles[userGUID], role.Type)
		}
	}

	return userRoles
}

// outputSpaceUsersAsTable outputs space users and their roles in table format.
func outputSpaceUsersAsTable(ctx context.Context, client capi.Client, spaceName string, roles []capi.Role, allPages bool, pagination capi.Pagination) error {
	if len(roles) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No users found in space '%s'\n", spaceName)

		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "Users in space '%s':\n\n", spaceName)

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("User GUID", "Role", "Created")

	// Group by user
	userRoles := groupRolesByUser(roles)

	for userGUID, roleList := range userRoles {
		rolesStr := strings.Join(roleList, ", ")
		// Get user details if possible
		user, _ := client.Users().Get(ctx, userGUID)

		username := userGUID
		if user != nil {
			username = user.Username
		}

		_ = table.Append(username, rolesStr, "")
	}

	_ = table.Render()

	if !allPages && pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", pagination.TotalPages)
	}

	return nil
}

func newSpacesListUsersCommand() *cobra.Command {
	var (
		allPages bool
		perPage  int
	)

	cmd := &cobra.Command{
		Use:   "list-users SPACE_NAME_OR_GUID",
		Short: "List users in space",
		Long:  "List all users with roles in the space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceNameOrGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find space
			space, err := findSpaceByNameOrGUID(ctx, client, spaceNameOrGUID)
			if err != nil {
				return err
			}

			// Get users with roles in space
			params := capi.NewQueryParams()
			params.PerPage = perPage
			params.WithFilter("space_guids", space.GUID)

			roles, err := client.Roles().List(ctx, params)
			if err != nil {
				return fmt.Errorf("failed to list space roles: %w", err)
			}

			// Fetch all pages if requested
			allRoles := roles.Resources
			if allPages {
				allRoles, err = fetchAllRolePagesForSpace(ctx, client, params, roles)
				if err != nil {
					return err
				}
			}

			// Output results
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				return outputRolesAsJSON(allRoles)
			case OutputFormatYAML:
				return outputRolesAsYAML(allRoles)
			default:
				return outputSpaceUsersAsTable(ctx, client, space.Name, allRoles, allPages, roles.Pagination)
			}
		},
	}

	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "results per page")

	return cmd
}

func newSpacesSetRoleCommand() *cobra.Command {
	roleContext := RoleContext{
		ResourceType:   "space",
		DefaultRole:    "space_developer",
		ValidRoles:     []string{"space_developer", "space_manager", "space_auditor", "space_supporter"},
		SuccessMessage: "Successfully set user role '%s' in space\n",
	}

	return CreateRoleCommand("set-role", "space", roleContext)()
}

func newSpacesUnsetRoleCommand() *cobra.Command {
	roleContext := RoleContext{
		ResourceType:   "space",
		DefaultRole:    "",
		ValidRoles:     []string{"space_developer", "space_manager", "space_auditor", "space_supporter"},
		SuccessMessage: "Successfully removed user role from space\n",
	}

	return CreateRoleCommand("unset-role", "space", roleContext)()
}

// fetchAllAppPagesForSpace fetches all pages of apps from the API.
func fetchAllAppPagesForSpace(ctx context.Context, client capi.Client, params *capi.QueryParams, firstPage *capi.ListResponse[capi.App]) ([]capi.App, error) {
	allApps := firstPage.Resources

	if firstPage.Pagination.TotalPages > 1 {
		for page := 2; page <= firstPage.Pagination.TotalPages; page++ {
			params.Page = page

			moreApps, err := client.Apps().List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
			}

			allApps = append(allApps, moreApps.Resources...)
		}
	}

	return allApps, nil
}

// outputAppsAsJSON outputs apps in JSON format.
func outputAppsAsJSON(apps []capi.App) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(apps)
	if err != nil {
		return fmt.Errorf("encoding apps to JSON: %w", err)
	}

	return nil
}

// outputAppsAsYAML outputs apps in YAML format.
func outputAppsAsYAML(apps []capi.App) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(apps)
	if err != nil {
		return fmt.Errorf("encoding apps to YAML: %w", err)
	}

	return nil
}

// outputSpaceAppsAsTable outputs space applications in table format.
func outputSpaceAppsAsTable(spaceName string, apps []capi.App, allPages bool, pagination capi.Pagination) error {
	if len(apps) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No applications found in space '%s'\n", spaceName)

		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "Applications in space '%s':\n\n", spaceName)

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "GUID", "State", "Created", "Updated")

	for _, app := range apps {
		_ = table.Append(app.Name, app.GUID, app.State,
			app.CreatedAt.Format("2006-01-02"),
			app.UpdatedAt.Format("2006-01-02"))
	}

	_ = table.Render()

	if !allPages && pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", pagination.TotalPages)
	}

	return nil
}

func newSpacesListAppsCommand() *cobra.Command {
	var (
		allPages bool
		perPage  int
	)

	cmd := &cobra.Command{
		Use:   "list-apps SPACE_NAME_OR_GUID",
		Short: "List applications in space",
		Long:  "List all applications in a space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceNameOrGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find space
			space, err := findSpaceByNameOrGUID(ctx, client, spaceNameOrGUID)
			if err != nil {
				return err
			}

			// List apps in space
			params := capi.NewQueryParams()
			params.PerPage = perPage
			params.WithFilter("space_guids", space.GUID)

			apps, err := client.Apps().List(ctx, params)
			if err != nil {
				return fmt.Errorf("failed to list applications: %w", err)
			}

			// Fetch all pages if requested
			allApps := apps.Resources
			if allPages {
				allApps, err = fetchAllAppPagesForSpace(ctx, client, params, apps)
				if err != nil {
					return err
				}
			}

			// Output results
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				return outputAppsAsJSON(allApps)
			case OutputFormatYAML:
				return outputAppsAsYAML(allApps)
			default:
				return outputSpaceAppsAsTable(space.Name, allApps, allPages, apps.Pagination)
			}
		},
	}

	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "results per page")

	return cmd
}

// fetchAllServicePagesForSpace fetches all pages of service instances from the API.
func fetchAllServicePagesForSpace(ctx context.Context, client capi.Client, params *capi.QueryParams, firstPage *capi.ListResponse[capi.ServiceInstance]) ([]capi.ServiceInstance, error) {
	allServices := firstPage.Resources

	if firstPage.Pagination.TotalPages > 1 {
		for page := 2; page <= firstPage.Pagination.TotalPages; page++ {
			params.Page = page

			moreServices, err := client.ServiceInstances().List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
			}

			allServices = append(allServices, moreServices.Resources...)
		}
	}

	return allServices, nil
}

// outputServicesAsJSON outputs service instances in JSON format.
func outputServicesAsJSON(services []capi.ServiceInstance) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(services)
	if err != nil {
		return fmt.Errorf("encoding services to JSON: %w", err)
	}

	return nil
}

// outputServicesAsYAML outputs service instances in YAML format.
func outputServicesAsYAML(services []capi.ServiceInstance) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(services)
	if err != nil {
		return fmt.Errorf("encoding services to YAML: %w", err)
	}

	return nil
}

// outputSpaceServicesAsTable outputs space service instances in table format.
func outputSpaceServicesAsTable(spaceName string, services []capi.ServiceInstance, allPages bool, pagination capi.Pagination) error {
	if len(services) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No service instances found in space '%s'\n", spaceName)

		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "Service instances in space '%s':\n\n", spaceName)

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "GUID", "Type", "State", "Created")

	for _, service := range services {
		state := "ready"
		if service.LastOperation != nil {
			state = service.LastOperation.State
		}

		_ = table.Append(service.Name, service.GUID, service.Type, state,
			service.CreatedAt.Format("2006-01-02"))
	}

	_ = table.Render()

	if !allPages && pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", pagination.TotalPages)
	}

	return nil
}

func newSpacesListServicesCommand() *cobra.Command {
	var (
		allPages bool
		perPage  int
	)

	cmd := &cobra.Command{
		Use:   "list-services SPACE_NAME_OR_GUID",
		Short: "List service instances in space",
		Long:  "List all service instances in a space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceNameOrGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find space
			space, err := findSpaceByNameOrGUID(ctx, client, spaceNameOrGUID)
			if err != nil {
				return err
			}

			// List service instances in space
			params := capi.NewQueryParams()
			params.PerPage = perPage
			params.WithFilter("space_guids", space.GUID)

			services, err := client.ServiceInstances().List(ctx, params)
			if err != nil {
				return fmt.Errorf("failed to list service instances: %w", err)
			}

			// Fetch all pages if requested
			allServices := services.Resources
			if allPages {
				allServices, err = fetchAllServicePagesForSpace(ctx, client, params, services)
				if err != nil {
					return err
				}
			}

			// Output results
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				return outputServicesAsJSON(allServices)
			case OutputFormatYAML:
				return outputServicesAsYAML(allServices)
			default:
				return outputSpaceServicesAsTable(space.Name, allServices, allPages, services.Pagination)
			}
		},
	}

	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "results per page")

	return cmd
}

func newSpacesApplyManifestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "apply-manifest SPACE_NAME_OR_GUID MANIFEST_FILE",
		Short: "Apply manifest to space",
		Long:  "Apply an application manifest to a space",
		Args:  cobra.ExactArgs(constants.TwoArgumentsMax),
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceNameOrGUID := args[0]
			manifestPath := args[1]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()
			spacesClient := client.Spaces()

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
					return fmt.Errorf("failed to find space '%s': %w", spaceNameOrGUID, err)
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

			// Validate and read manifest file
			err = validateFilePathSpaces(manifestPath)
			if err != nil {
				return fmt.Errorf("invalid manifest file: %w", err)
			}

			manifestContent, err := os.ReadFile(filepath.Clean(manifestPath))
			if err != nil {
				return fmt.Errorf("failed to read manifest file '%s': %w", manifestPath, err)
			}

			// Apply manifest
			job, err := spacesClient.ApplyManifest(ctx, spaceGUID, string(manifestContent))
			if err != nil {
				return fmt.Errorf("failed to apply manifest: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully applied manifest to space '%s'\n", spaceName)
			if job != nil {
				_, _ = fmt.Fprintf(os.Stdout, "Job GUID: %s\n", job.GUID)
				_, _ = fmt.Fprintf(os.Stdout, "Monitor job status with: capi jobs get %s\n", job.GUID)
			}

			return nil
		},
	}
}

func newSpacesFeaturesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "features",
		Short: "Manage space features",
		Long:  "Manage space features including listing, getting, enabling, and disabling features",
	}

	cmd.AddCommand(newSpacesFeaturesListCommand())
	cmd.AddCommand(newSpacesFeaturesGetCommand())
	cmd.AddCommand(newSpacesFeaturesEnableCommand())
	cmd.AddCommand(newSpacesFeaturesDisableCommand())

	return cmd
}

func newSpacesFeaturesListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list SPACE_NAME_OR_GUID",
		Short: "List space features",
		Long:  "List all features for a space",
		Args:  cobra.ExactArgs(1),
		RunE:  runSpacesFeaturesListCommand,
	}
}

func runSpacesFeaturesListCommand(cmd *cobra.Command, args []string) error {
	spaceNameOrGUID := args[0]

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Find space
	space, err := client.Spaces().Get(ctx, spaceNameOrGUID)
	if err != nil {
		// Try by name in targeted org
		params := capi.NewQueryParams()
		params.WithFilter("names", spaceNameOrGUID)

		if orgGUID := viper.GetString("organization_guid"); orgGUID != "" {
			params.WithFilter("organization_guids", orgGUID)
		}

		spaces, err := client.Spaces().List(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to find space: %w", err)
		}

		if len(spaces.Resources) == 0 {
			return fmt.Errorf("space '%s': %w", spaceNameOrGUID, ErrSpaceNotFound)
		}

		space = &spaces.Resources[0]
	}

	features, err := client.Spaces().GetFeatures(ctx, space.GUID)
	if err != nil {
		return fmt.Errorf("getting space features: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(features)
		if err != nil {
			return fmt.Errorf("encoding features to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(features)
		if err != nil {
			return fmt.Errorf("encoding features to YAML: %w", err)
		}

		return nil
	default:
		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Feature", "Status")

		status := constants.StatusDisabled
		if features.SSHEnabled {
			status = constants.StatusEnabled
		}

		_ = table.Append("ssh", status)

		_, _ = fmt.Fprintf(os.Stdout, "Features for space '%s':\n\n", space.Name)

		err := table.Render()
		if err != nil {
			return fmt.Errorf("failed to render table: %w", err)
		}
	}

	return nil
}

func findSpaceForFeature(ctx context.Context, client capi.Client, spaceNameOrGUID string) (*capi.Space, error) {
	space, err := client.Spaces().Get(ctx, spaceNameOrGUID)
	if err == nil {
		return space, nil
	}

	// Try by name in targeted org
	params := capi.NewQueryParams()
	params.WithFilter("names", spaceNameOrGUID)

	if orgGUID := viper.GetString("organization_guid"); orgGUID != "" {
		params.WithFilter("organization_guids", orgGUID)
	}

	spaces, err := client.Spaces().List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}

	if len(spaces.Resources) == 0 {
		return nil, fmt.Errorf("space '%s': %w", spaceNameOrGUID, ErrSpaceNotFound)
	}

	return &spaces.Resources[0], nil
}

func displaySpaceFeature(feature *capi.SpaceFeature, space *capi.Space, output string) error {
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(feature)
		if err != nil {
			return fmt.Errorf("encoding feature to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(feature)
		if err != nil {
			return fmt.Errorf("encoding feature to YAML: %w", err)
		}

		return nil
	default:
		return displaySpaceFeatureTable(feature, space)
	}
}

func displaySpaceFeatureTable(feature *capi.SpaceFeature, space *capi.Space) error {
	status := statusDisabled
	if feature.Enabled {
		status = statusEnabled
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")
	_ = table.Append("Name", feature.Name)

	_ = table.Append("Status", status)
	if feature.Description != "" {
		_ = table.Append("Description", feature.Description)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Feature '%s' for space '%s':\n\n", feature.Name, space.Name)

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	return nil
}

func newSpacesFeaturesGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get SPACE_NAME_OR_GUID FEATURE_NAME",
		Short: "Get details for a specific space feature",
		Long:  "Get detailed information about a specific space feature",
		Args:  cobra.ExactArgs(constants.TwoArgumentsMax),
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceNameOrGUID := args[0]
			featureName := args[1]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find space
			space, err := findSpaceForFeature(ctx, client, spaceNameOrGUID)
			if err != nil {
				return err
			}

			// Get feature
			feature, err := client.Spaces().GetFeature(ctx, space.GUID, featureName)
			if err != nil {
				return fmt.Errorf("getting space feature '%s': %w", featureName, err)
			}

			// Display feature
			return displaySpaceFeature(feature, space, viper.GetString("output"))
		},
	}
}

func newSpacesFeaturesEnableCommand() *cobra.Command {
	return createSpaceFeatureToggleCommand(
		"enable SPACE_NAME_OR_GUID FEATURE_NAME",
		"Enable a specific space feature",
		"Enable a specific space feature",
		true,
		"✓ Feature '%s' has been "+statusEnabled+" for space '%s'\n",
	)
}

func newSpacesFeaturesDisableCommand() *cobra.Command {
	return createSpaceFeatureToggleCommand(
		"disable SPACE_NAME_OR_GUID FEATURE_NAME",
		"Disable a specific space feature",
		"Disable a specific space feature",
		false,
		"✓ Feature '%s' has been "+statusDisabled+" for space '%s'\n",
	)
}
