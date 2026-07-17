package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"os"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// flagUser is the shared "--user" flag name used by role list/create commands.
const flagUser = "user"

// NewRolesCommand creates the roles command group.
func NewRolesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "roles",
		Aliases: []string{"role"},
		Short:   "Manage roles",
		Long:    "List and manage Cloud Foundry user roles",
	}

	cmd.AddCommand(newRolesListCommand())
	cmd.AddCommand(newRolesGetCommand())
	cmd.AddCommand(newRolesCreateCommand())
	cmd.AddCommand(newRolesDeleteCommand())

	return cmd
}

func newRolesListCommand() *cobra.Command {
	var (
		allPages  bool
		perPage   int
		userGUID  string
		orgGUID   string
		spaceGUID string
		roleType  string
	)

	cmd := &cobra.Command{
		Use:   List,
		Short: "List roles",
		Long:  "List all roles the user has access to",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()
			params := buildRoleListParams(perPage, userGUID, orgGUID, spaceGUID, roleType)

			roles, err := client.Roles().List(ctx, params)
			if err != nil {
				return fmt.Errorf("failed to list roles: %w", err)
			}

			allRoles, err := fetchAllRolePages(ctx, client, params, roles, allPages)
			if err != nil {
				return err
			}

			return renderRoles(allRoles)
		},
	}

	cmd.Flags().BoolVar(&allPages, "all-pages", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "number of results per page")
	cmd.Flags().StringVar(&userGUID, flagUser, "", "filter by user GUID")
	cmd.Flags().StringVar(&orgGUID, "org", "", "filter by organization GUID")
	cmd.Flags().StringVar(&spaceGUID, spaceKey, "", "filter by space GUID")
	cmd.Flags().StringVar(&roleType, "type", "", "filter by role type")

	return cmd
}

func buildRoleListParams(perPage int, userGUID, orgGUID, spaceGUID, roleType string) *capi.QueryParams {
	params := capi.NewQueryParams()
	params.PerPage = perPage

	if userGUID != "" {
		params.WithFilter("user_guids", userGUID)
	}

	if orgGUID != "" {
		params.WithFilter("organization_guids", orgGUID)
	}

	if spaceGUID != "" {
		params.WithFilter("space_guids", spaceGUID)
	}

	if roleType != "" {
		params.WithFilter("types", roleType)
	}

	return params
}

func fetchAllRolePages(ctx context.Context, client capi.Client, params *capi.QueryParams, roles *capi.ListResponse[capi.Role], allPages bool) ([]capi.Role, error) {
	allRoles := roles.Resources
	if allPages && roles.Pagination.TotalPages > 1 {
		for page := 2; page <= roles.Pagination.TotalPages; page++ {
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

func renderRoles(allRoles []capi.Role) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(allRoles)
		if err != nil {
			return fmt.Errorf("failed to encode roles as JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(allRoles)
		if err != nil {
			return fmt.Errorf("failed to encode roles as YAML: %w", err)
		}

		return nil
	default:
		return renderRolesTable(allRoles)
	}
}

func renderRolesTable(allRoles []capi.Role) error {
	if len(allRoles) == 0 {
		_, _ = os.Stdout.WriteString("No roles found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("GUID", "Type", "User GUID", "Organization GUID", "Space GUID", "Created", "Updated")

	for _, role := range allRoles {
		orgGUID := ""
		if role.Relationships.Organization != nil {
			orgGUID = role.Relationships.Organization.Data.GUID
		}

		spaceGUID := ""
		if role.Relationships.Space != nil {
			spaceGUID = role.Relationships.Space.Data.GUID
		}

		_ = table.Append([]string{
			role.GUID,
			role.Type,
			role.Relationships.User.Data.GUID,
			orgGUID,
			spaceGUID,
			role.CreatedAt.Format(TimeFormatDisplay),
			role.UpdatedAt.Format(TimeFormatDisplay),
		})
	}

	_ = table.Render()

	return nil
}

func newRolesGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get ROLE_GUID",
		Short: "Get role details",
		Long:  "Display detailed information about a specific role",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			roleGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			role, err := client.Roles().Get(ctx, roleGUID)
			if err != nil {
				return fmt.Errorf("failed to get role: %w", err)
			}

			// Output results
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				return encoder.Encode(role)
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				return encoder.Encode(role)
			default:
				table := tablewriter.NewWriter(os.Stdout)
				table.Header("Property", "Value")

				_ = table.Append("GUID", role.GUID)
				_ = table.Append("Type", role.Type)
				_ = table.Append("User GUID", role.Relationships.User.Data.GUID)

				if role.Relationships.Organization != nil {
					_ = table.Append("Organization GUID", role.Relationships.Organization.Data.GUID)
				}

				if role.Relationships.Space != nil {
					_ = table.Append("Space GUID", role.Relationships.Space.Data.GUID)
				}

				_ = table.Append("Created", role.CreatedAt.Format(TimeFormatDisplay))
				_ = table.Append("Updated", role.UpdatedAt.Format(TimeFormatDisplay))

				_, _ = os.Stdout.WriteString("Role details:\n\n")
				_ = table.Render()
			}

			return nil
		},
	}
}

func newRolesCreateCommand() *cobra.Command {
	var (
		userGUID  string
		orgGUID   string
		spaceGUID string
	)

	cmd := &cobra.Command{
		Use:   "create ROLE_TYPE",
		Short: "Create a role",
		Long:  "Create a new role assignment for a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			roleType := args[0]

			if userGUID == "" {
				return ErrUserGUIDRequired
			}

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()
			request := buildRoleCreateRequest(roleType, userGUID, orgGUID, spaceGUID)

			role, err := client.Roles().Create(ctx, request)
			if err != nil {
				return fmt.Errorf("failed to create role: %w", err)
			}

			return renderCreatedRole(role)
		},
	}

	cmd.Flags().StringVar(&userGUID, flagUser, "", "user GUID (required)")
	cmd.Flags().StringVar(&orgGUID, "org", "", "organization GUID")
	cmd.Flags().StringVar(&spaceGUID, spaceKey, "", "space GUID")
	_ = cmd.MarkFlagRequired(flagUser)

	return cmd
}

func buildRoleCreateRequest(roleType, userGUID, orgGUID, spaceGUID string) *capi.RoleCreateRequest {
	relationships := capi.RoleRelationships{
		User: capi.Relationship{
			Data: &capi.RelationshipData{GUID: userGUID},
		},
	}

	if orgGUID != "" {
		relationships.Organization = &capi.Relationship{
			Data: &capi.RelationshipData{GUID: orgGUID},
		}
	}

	if spaceGUID != "" {
		relationships.Space = &capi.Relationship{
			Data: &capi.RelationshipData{GUID: spaceGUID},
		}
	}

	return &capi.RoleCreateRequest{
		Type:          roleType,
		Relationships: relationships,
	}
}

func renderCreatedRole(role *capi.Role) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(role)
		if err != nil {
			return fmt.Errorf("failed to encode role as JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(role)
		if err != nil {
			return fmt.Errorf("failed to encode role as YAML: %w", err)
		}

		return nil
	default:
		_, _ = fmt.Fprintf(os.Stdout, "Created role: %s\n", role.GUID)
		_, _ = fmt.Fprintf(os.Stdout, "  Type:      %s\n", role.Type)
		_, _ = fmt.Fprintf(os.Stdout, "  User GUID: %s\n", role.Relationships.User.Data.GUID)

		if role.Relationships.Organization != nil {
			_, _ = fmt.Fprintf(os.Stdout, "  Org GUID:  %s\n", role.Relationships.Organization.Data.GUID)
		}

		if role.Relationships.Space != nil {
			_, _ = fmt.Fprintf(os.Stdout, "  Space GUID: %s\n", role.Relationships.Space.Data.GUID)
		}
	}

	return nil
}

func newRolesDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete ROLE_GUID",
		Short: "Delete a role",
		Long:  "Delete a role assignment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			roleGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			_, err = client.Roles().Delete(ctx, roleGUID)
			if err != nil {
				return fmt.Errorf("failed to delete role: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Role %s deleted successfully\n", roleGUID)

			return nil
		},
	}
}
