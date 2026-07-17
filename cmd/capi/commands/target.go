package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// TargetInfo represents the current target information.
type TargetInfo struct {
	API          string `json:"api,omitempty"          yaml:"api,omitempty"`
	Endpoint     string `json:"endpoint,omitempty"     yaml:"endpoint,omitempty"`
	User         string `json:"user,omitempty"         yaml:"user,omitempty"`
	Organization string `json:"organization,omitempty" yaml:"organization,omitempty"`
	Space        string `json:"space,omitempty"        yaml:"space,omitempty"`
}

// NewTargetCommand creates the target command.
func NewTargetCommand() *cobra.Command {
	var (
		orgName   string
		spaceName string
	)

	cmd := &cobra.Command{
		Use:   "target",
		Short: "Set or show the targeted organization and space",
		Long:  "Set or display the currently targeted Cloud Foundry organization and space",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTargetCommand(cmd, orgName, spaceName)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&orgName, "org", "o", "", "target organization")
	cmd.Flags().StringVarP(&spaceName, spaceKey, "s", "", "target space")

	return cmd
}

func runTargetCommand(cmd *cobra.Command, orgName, spaceName string) error {
	// If no flags provided, show current target
	if orgName == "" && spaceName == "" {
		return showTarget()
	}

	// Create client
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Get current API config
	config := loadConfig()

	apiConfig, err := getCurrentAPIConfig()
	if err != nil {
		return err
	}

	// Target organization
	if orgName != "" {
		err := targetOrganization(ctx, client, orgName, apiConfig)
		if err != nil {
			return err
		}

		// Clear space if only org is being targeted
		if spaceName == "" {
			apiConfig.Space = ""
			apiConfig.SpaceGUID = ""

			// List available spaces
			listAvailableSpaces(ctx, client, apiConfig.OrganizationGUID)
		}
	}

	// Target space (requires organization to be set)
	if spaceName != "" {
		if apiConfig.OrganizationGUID == "" {
			return ErrOrganizationMustBeTargeted
		}

		spacesClient := client.Spaces()
		spaceParams := capi.NewQueryParams()
		spaceParams.WithFilter("names", spaceName)
		spaceParams.WithFilter("organization_guids", apiConfig.OrganizationGUID)

		spaces, err := spacesClient.List(ctx, spaceParams)
		if err != nil {
			return fmt.Errorf("failed to find space: %w", err)
		}

		if len(spaces.Resources) == 0 {
			return fmt.Errorf("space '%s' not found in organization: %w", spaceName, ErrSpaceNotFound)
		}

		space := spaces.Resources[0]
		apiConfig.Space = space.Name
		apiConfig.SpaceGUID = space.GUID
		_, _ = fmt.Fprintf(os.Stdout, "Targeted space: %s\n", space.Name)
	}

	// Update config and save
	config.APIs[config.CurrentAPI] = apiConfig

	err = saveConfigStruct(config)
	if err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

// targetOrganization targets the specified organization and updates the config.
func targetOrganization(ctx context.Context, client capi.Client, orgName string, apiConfig *APIConfig) error {
	orgsClient := client.Organizations()
	params := capi.NewQueryParams()
	params.WithFilter("names", orgName)

	orgs, err := orgsClient.List(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to find organization: %w", err)
	}

	if len(orgs.Resources) == 0 {
		return fmt.Errorf("organization '%s': %w", orgName, ErrOrganizationNotFound)
	}

	org := orgs.Resources[0]
	apiConfig.Organization = org.Name
	apiConfig.OrganizationGUID = org.GUID
	_, _ = fmt.Fprintf(os.Stdout, "Targeted organization: %s\n", org.Name)

	return nil
}

// listAvailableSpaces lists available spaces for the targeted organization.
func listAvailableSpaces(ctx context.Context, client capi.Client, orgGUID string) {
	spacesClient := client.Spaces()
	spaceParams := capi.NewQueryParams()
	spaceParams.WithFilter("organization_guids", orgGUID)

	spaces, err := spacesClient.List(ctx, spaceParams)
	if err == nil && len(spaces.Resources) > 0 {
		_, _ = os.Stdout.WriteString("\nAvailable spaces:\n")

		for _, space := range spaces.Resources {
			_, _ = fmt.Fprintf(os.Stdout, "  - %s\n", space.Name)
		}

		_, _ = os.Stdout.WriteString("\nUse 'capi target -s <space>' to target a space\n")
	}
}

func showTarget() error {
	config := loadConfig()

	if config.CurrentAPI == "" || len(config.APIs) == 0 {
		_, _ = os.Stdout.WriteString("No API targeted. Use 'capi apis add' to add an API endpoint.\n")

		return nil
	}

	apiConfig, exists := config.APIs[config.CurrentAPI]
	if !exists {
		_, _ = fmt.Fprintf(os.Stdout, "Current API '%s' not found in configuration.\n", config.CurrentAPI)

		return nil
	}

	// Create target info struct
	targetInfo := TargetInfo{
		API:      config.CurrentAPI,
		Endpoint: apiConfig.Endpoint,
	}

	if apiConfig.Username != "" {
		targetInfo.User = apiConfig.Username
	}

	if apiConfig.Organization != "" {
		targetInfo.Organization = apiConfig.Organization
	}

	if apiConfig.Space != "" {
		targetInfo.Space = apiConfig.Space
	}

	return outputTargetInfo(targetInfo)
}

func outputTargetInfo(targetInfo TargetInfo) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(targetInfo)
		if err != nil {
			return fmt.Errorf("failed to encode target info as JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(targetInfo)
		if err != nil {
			return fmt.Errorf("failed to encode target info as YAML: %w", err)
		}

		return nil
	default:
		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Property", "Value")

		_ = table.Append("API", targetInfo.API)
		_ = table.Append("Endpoint", targetInfo.Endpoint)

		if targetInfo.User != "" {
			_ = table.Append("User", targetInfo.User)
		}

		if targetInfo.Organization != "" {
			_ = table.Append("Organization", targetInfo.Organization)
		}

		if targetInfo.Space != "" {
			_ = table.Append("Space", targetInfo.Space)
		}

		_ = table.Render()
	}

	return nil
}
