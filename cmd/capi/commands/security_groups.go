package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// validateFilePath validates that a file path is safe to read.
func validateFilePathSecurity(filePath string) error {
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

	// Check if file exists and is readable
	_, err := os.Stat(cleanPath)
	if err != nil {
		return fmt.Errorf("file not accessible: %w", err)
	}

	return nil
}

// NewSecurityGroupsCommand creates the security groups command group.
func NewSecurityGroupsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "security-groups",
		Aliases: []string{"security-group", "sg"},
		Short:   "Manage security groups",
		Long:    "List and manage Cloud Foundry security groups",
	}

	cmd.AddCommand(newSecurityGroupsListCommand())
	cmd.AddCommand(newSecurityGroupsGetCommand())
	cmd.AddCommand(newSecurityGroupsCreateCommand())
	cmd.AddCommand(newSecurityGroupsUpdateCommand())
	cmd.AddCommand(newSecurityGroupsDeleteCommand())
	cmd.AddCommand(newSecurityGroupsBindCommand())
	cmd.AddCommand(newSecurityGroupsUnbindCommand())
	cmd.AddCommand(newSecurityGroupsRunningCommand())
	cmd.AddCommand(newSecurityGroupsStagingCommand())

	return cmd
}

func newSecurityGroupsListCommand() *cobra.Command {
	var (
		allPages        bool
		perPage         int
		globallyEnabled bool
		runningSpaces   bool
		stagingSpaces   bool
	)

	cmd := &cobra.Command{
		Use:   List,
		Short: "List security groups",
		Long:  "List all security groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeSecurityGroupsList(cmd, allPages, perPage, globallyEnabled)
		},
	}

	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "results per page")
	cmd.Flags().BoolVar(&globallyEnabled, "globally-enabled", false, "filter by globally enabled for running")
	cmd.Flags().BoolVar(&runningSpaces, "running-spaces", false, "show running spaces bindings")
	cmd.Flags().BoolVar(&stagingSpaces, "staging-spaces", false, "show staging spaces bindings")

	return cmd
}

// executeSecurityGroupsList handles the security groups list logic.
func executeSecurityGroupsList(cmd *cobra.Command, allPages bool, perPage int, globallyEnabled bool) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()
	params := capi.NewQueryParams()
	params.PerPage = perPage

	// Apply filters based on flags
	if globallyEnabled {
		params.WithFilter("globally_enabled_running", "true")
	}

	// Fetch security groups
	allGroups, pagination, err := fetchAllSecurityGroups(ctx, client, params, allPages)
	if err != nil {
		return err
	}

	return outputSecurityGroups(allGroups, pagination, allPages)
}

// fetchAllSecurityGroups fetches all security groups with pagination.
func fetchAllSecurityGroups(ctx context.Context, client capi.Client, params *capi.QueryParams, allPages bool) ([]capi.SecurityGroup, *capi.Pagination, error) {
	securityGroups, err := client.SecurityGroups().List(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list security groups: %w", err)
	}

	allGroups := securityGroups.Resources
	if allPages && securityGroups.Pagination.TotalPages > 1 {
		for page := 2; page <= securityGroups.Pagination.TotalPages; page++ {
			params.Page = page

			moreGroups, err := client.SecurityGroups().List(ctx, params)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
			}

			allGroups = append(allGroups, moreGroups.Resources...)
		}
	}

	return allGroups, &securityGroups.Pagination, nil
}

// outputSecurityGroups outputs security groups in the requested format.
func outputSecurityGroups(allGroups []capi.SecurityGroup, pagination *capi.Pagination, allPages bool) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(allGroups)
		if err != nil {
			return fmt.Errorf("failed to encode security groups: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(allGroups)
		if err != nil {
			return fmt.Errorf("failed to encode security groups: %w", err)
		}

		return nil
	default:
		return renderSecurityGroupsTable(allGroups, pagination, allPages)
	}
}

// renderSecurityGroupsTable renders security groups as a table.
func renderSecurityGroupsTable(allGroups []capi.SecurityGroup, pagination *capi.Pagination, allPages bool) error {
	if len(allGroups) == 0 {
		_, _ = os.Stdout.WriteString("No security groups found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "GUID", "Running Spaces", "Staging Spaces", "Global Running", "Global Staging")

	for _, securityGroup := range allGroups {
		runningSpacesCount := len(securityGroup.Relationships.RunningSpaces.Data)
		stagingSpacesCount := len(securityGroup.Relationships.StagingSpaces.Data)

		globalRunning := "no"
		if securityGroup.GloballyEnabled.Running {
			globalRunning = Yes
		}

		globalStaging := "no"
		if securityGroup.GloballyEnabled.Staging {
			globalStaging = Yes
		}

		_ = table.Append(securityGroup.Name, securityGroup.GUID,
			strconv.Itoa(runningSpacesCount),
			strconv.Itoa(stagingSpacesCount),
			globalRunning, globalStaging)
	}

	_ = table.Render()

	if !allPages && pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", pagination.TotalPages)
	}

	return nil
}

func printSecurityGroupDetails(securityGroup *capi.SecurityGroup) {
	_, _ = fmt.Fprintf(os.Stdout, "Security Group: %s\n", securityGroup.Name)
	_, _ = fmt.Fprintf(os.Stdout, "  GUID:     %s\n", securityGroup.GUID)
	_, _ = fmt.Fprintf(os.Stdout, "  Created:  %s\n", securityGroup.CreatedAt.Format(TimeFormatDisplay))
	_, _ = fmt.Fprintf(os.Stdout, "  Updated:  %s\n", securityGroup.UpdatedAt.Format(TimeFormatDisplay))

	_, _ = os.Stdout.WriteString("  Globally Enabled:\n")
	_, _ = fmt.Fprintf(os.Stdout, "    Running: %t\n", securityGroup.GloballyEnabled.Running)
	_, _ = fmt.Fprintf(os.Stdout, "    Staging: %t\n", securityGroup.GloballyEnabled.Staging)

	if len(securityGroup.Rules) > 0 {
		_, _ = os.Stdout.WriteString("  Rules:\n")

		for i, rule := range securityGroup.Rules {
			printSecurityGroupRule(i+1, rule)
		}
	}

	// Show bound spaces
	runningSpacesCount := len(securityGroup.Relationships.RunningSpaces.Data)
	stagingSpacesCount := len(securityGroup.Relationships.StagingSpaces.Data)
	_, _ = fmt.Fprintf(os.Stdout, "  Bound to %d running spaces, %d staging spaces\n", runningSpacesCount, stagingSpacesCount)
}

func printSecurityGroupRule(index int, rule capi.SecurityGroupRule) {
	_, _ = fmt.Fprintf(os.Stdout, "    Rule %d:\n", index)
	_, _ = fmt.Fprintf(os.Stdout, "      Protocol:    %s\n", rule.Protocol)
	_, _ = fmt.Fprintf(os.Stdout, "      Destination: %s\n", rule.Destination)

	if rule.Ports != nil {
		_, _ = fmt.Fprintf(os.Stdout, "      Ports:       %s\n", *rule.Ports)
	}

	if rule.Description != nil {
		_, _ = fmt.Fprintf(os.Stdout, "      Description: %s\n", *rule.Description)
	}

	if rule.Log != nil {
		_, _ = fmt.Fprintf(os.Stdout, "      Log:         %t\n", *rule.Log)
	}
}

func newSecurityGroupsGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get SECURITY_GROUP_NAME_OR_GUID",
		Short: "Get security group details",
		Long:  "Display detailed information about a specific security group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Try to get by GUID first
			securityGroup, err := client.SecurityGroups().Get(ctx, nameOrGUID)
			if err != nil {
				// If not found by GUID, try by name
				params := capi.NewQueryParams()
				params.WithFilter("names", nameOrGUID)

				groups, err := client.SecurityGroups().List(ctx, params)
				if err != nil {
					return fmt.Errorf("failed to find security group: %w", err)
				}

				if len(groups.Resources) == 0 {
					return fmt.Errorf("security group '%s': %w", nameOrGUID, ErrSecurityGroupNotFound)
				}

				securityGroup = &groups.Resources[0]
			}

			// Output results
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				return encoder.Encode(securityGroup)
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				return encoder.Encode(securityGroup)
			default:
				printSecurityGroupDetails(securityGroup)
			}

			return nil
		},
	}
}

func newSecurityGroupsCreateCommand() *cobra.Command {
	var (
		name          string
		rulesFile     string
		globalRunning bool
		globalStaging bool
	)

	cmd := &cobra.Command{
		Use:   Create,
		Short: "Create a security group",
		Long:  "Create a new security group with rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return ErrSecurityGroupNameRequired
			}

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			createReq := &capi.SecurityGroupCreateRequest{
				Name: name,
			}

			// Set global enablement
			if globalRunning || globalStaging {
				createReq.GloballyEnabled = &capi.SecurityGroupGloballyEnabled{
					Running: globalRunning,
					Staging: globalStaging,
				}
			}

			// Load rules from file if specified
			if rulesFile != "" {
				// Validate file path to prevent directory traversal
				err := validateFilePathSecurity(rulesFile)
				if err != nil {
					return fmt.Errorf("invalid rules file: %w", err)
				}

				rulesContent, err := os.ReadFile(filepath.Clean(rulesFile))
				if err != nil {
					return fmt.Errorf("failed to read rules file: %w", err)
				}

				var rules []capi.SecurityGroupRule

				err = json.Unmarshal(rulesContent, &rules)
				if err != nil {
					return fmt.Errorf("failed to parse rules JSON: %w", err)
				}

				createReq.Rules = rules
			}

			createdSecurityGroup, err := client.SecurityGroups().Create(ctx, createReq)
			if err != nil {
				return fmt.Errorf("failed to create security group: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully created security group '%s'\n", createdSecurityGroup.Name)
			_, _ = fmt.Fprintf(os.Stdout, "  GUID: %s\n", createdSecurityGroup.GUID)
			_, _ = fmt.Fprintf(os.Stdout, "  Rules: %d\n", len(createdSecurityGroup.Rules))

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "security group name (required)")
	cmd.Flags().StringVarP(&rulesFile, "rules", "r", "", "JSON file containing security group rules")
	cmd.Flags().BoolVar(&globalRunning, "global-running", false, "enable globally for running applications")
	cmd.Flags().BoolVar(&globalStaging, "global-staging", false, "enable globally for staging applications")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func loadSecurityGroupRules(rulesFile string) ([]capi.SecurityGroupRule, error) {
	// Validate file path to prevent directory traversal
	err := validateFilePathSecurity(rulesFile)
	if err != nil {
		return nil, fmt.Errorf("invalid rules file: %w", err)
	}

	rulesContent, err := os.ReadFile(filepath.Clean(rulesFile))
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file: %w", err)
	}

	var rules []capi.SecurityGroupRule

	err = json.Unmarshal(rulesContent, &rules)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rules JSON: %w", err)
	}

	return rules, nil
}

// findSecurityGroupByNameOrGUID finds a security group by name or GUID.
func findSecurityGroupByNameOrGUID(ctx context.Context, client capi.Client, nameOrGUID string) (*capi.SecurityGroup, error) {
	securityGroup, err := client.SecurityGroups().Get(ctx, nameOrGUID)
	if err != nil {
		// Try by name
		params := capi.NewQueryParams()
		params.WithFilter("names", nameOrGUID)

		groups, err := client.SecurityGroups().List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to find security group: %w", err)
		}

		if len(groups.Resources) == 0 {
			return nil, fmt.Errorf("security group '%s': %w", nameOrGUID, ErrSecurityGroupNotFound)
		}

		securityGroup = &groups.Resources[0]
	}

	return securityGroup, nil
}

// findSpaceByName finds a space by name with optional org filter.
func findSpaceByName(ctx context.Context, client capi.Client, spaceName string, orgGUID string) (*capi.Space, error) {
	params := capi.NewQueryParams()
	params.WithFilter("names", spaceName)

	if orgGUID != "" {
		params.WithFilter("organization_guids", orgGUID)
	}

	spaces, err := client.Spaces().List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find space '%s': %w", spaceName, err)
	}

	if len(spaces.Resources) == 0 {
		return nil, fmt.Errorf("space '%s': %w", spaceName, ErrSpaceNotFound)
	}

	return &spaces.Resources[0], nil
}

// buildUpdateRequest builds an update request for a security group.
func buildUpdateRequest(cmd *cobra.Command, securityGroup *capi.SecurityGroup, newName, rulesFile string, globalRunning, globalStaging bool) (*capi.SecurityGroupUpdateRequest, error) {
	updateReq := &capi.SecurityGroupUpdateRequest{}

	if newName != "" {
		updateReq.Name = &newName
	}

	// Set global enablement if specified
	if cmd.Flags().Changed("global-running") || cmd.Flags().Changed("global-staging") {
		updateReq.GloballyEnabled = &capi.SecurityGroupGloballyEnabled{
			Running: securityGroup.GloballyEnabled.Running,
			Staging: securityGroup.GloballyEnabled.Staging,
		}
		if cmd.Flags().Changed("global-running") {
			updateReq.GloballyEnabled.Running = globalRunning
		}

		if cmd.Flags().Changed("global-staging") {
			updateReq.GloballyEnabled.Staging = globalStaging
		}
	}

	// Load rules from file if specified
	if rulesFile != "" {
		rules, err := loadSecurityGroupRules(rulesFile)
		if err != nil {
			return nil, err
		}

		updateReq.Rules = rules
	}

	return updateReq, nil
}

func newSecurityGroupsUpdateCommand() *cobra.Command {
	var (
		newName       string
		rulesFile     string
		globalRunning bool
		globalStaging bool
	)

	cmd := &cobra.Command{
		Use:   "update SECURITY_GROUP_NAME_OR_GUID",
		Short: "Update a security group",
		Long:  "Update an existing security group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			securityGroup, err := findSecurityGroupByNameOrGUID(ctx, client, args[0])
			if err != nil {
				return err
			}

			updateReq, err := buildUpdateRequest(cmd, securityGroup, newName, rulesFile, globalRunning, globalStaging)
			if err != nil {
				return err
			}

			updatedSG, err := client.SecurityGroups().Update(ctx, securityGroup.GUID, updateReq)
			if err != nil {
				return fmt.Errorf("failed to update security group: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully updated security group '%s'\n", updatedSG.Name)

			return nil
		},
	}

	cmd.Flags().StringVar(&newName, "name", "", "new security group name")
	cmd.Flags().StringVarP(&rulesFile, "rules", "r", "", "JSON file containing security group rules")
	cmd.Flags().BoolVar(&globalRunning, "global-running", false, "enable globally for running applications")
	cmd.Flags().BoolVar(&globalStaging, "global-staging", false, "enable globally for staging applications")

	return cmd
}

func newSecurityGroupsDeleteCommand() *cobra.Command {
	config := DeleteConfig{
		Use:         "delete SECURITY_GROUP_NAME_OR_GUID",
		Short:       "Delete a security group",
		Long:        "Delete a security group",
		EntityType:  "security group",
		GetResource: CreateSecurityGroupDeleteResourceFunc(),
		DeleteFunc: func(ctx context.Context, client any, guid string) (*string, error) {
			capiClient, ok := client.(capi.Client)
			if !ok {
				return nil, constants.ErrInvalidClientType
			}

			job, err := capiClient.SecurityGroups().Delete(ctx, guid)
			if err != nil {
				return nil, fmt.Errorf("failed to delete security group: %w", err)
			}

			if job != nil {
				return &job.GUID, nil
			}

			return nil, nil
		},
	}

	return createDeleteCommand(config)
}

// findSpaceGUIDs finds GUIDs for the given space names.
func findSpaceGUIDs(ctx context.Context, client capi.Client, spaceNames []string, orgGUID string) ([]string, error) {
	spaceGUIDs := make([]string, 0, len(spaceNames))

	for _, spaceName := range spaceNames {
		space, err := findSpaceByName(ctx, client, spaceName, orgGUID)
		if err != nil {
			return nil, err
		}

		spaceGUIDs = append(spaceGUIDs, space.GUID)
	}

	return spaceGUIDs, nil
}

// bindSecurityGroupToSpaces handles binding logic for both running and staging.
func bindSecurityGroupToSpaces(ctx context.Context, client capi.Client, sgGUID, sgName string, spaceGUIDs, spaceNames []string, running, staging bool) error {
	if running {
		_, err := client.SecurityGroups().BindRunningSpaces(ctx, sgGUID, spaceGUIDs)
		if err != nil {
			return fmt.Errorf("failed to bind security group to running spaces: %w", err)
		}

		_, _ = fmt.Fprintf(os.Stdout, "Successfully bound security group '%s' to running spaces: %s\n", sgName, strings.Join(spaceNames, ", "))
	}

	if staging {
		_, err := client.SecurityGroups().BindStagingSpaces(ctx, sgGUID, spaceGUIDs)
		if err != nil {
			return fmt.Errorf("failed to bind security group to staging spaces: %w", err)
		}

		_, _ = fmt.Fprintf(os.Stdout, "Successfully bound security group '%s' to staging spaces: %s\n", sgName, strings.Join(spaceNames, ", "))
	}

	return nil
}

func newSecurityGroupsBindCommand() *cobra.Command {
	var (
		spaceNames []string
		running    bool
		staging    bool
	)

	cmd := &cobra.Command{
		Use:   "bind SECURITY_GROUP_NAME_OR_GUID",
		Short: "Bind security group to spaces",
		Long:  "Bind a security group to spaces for running or staging",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(spaceNames) == 0 {
				return ErrAtLeastOneSpaceRequired
			}

			if !running && !staging {
				return ErrMustSpecifyRunningOrStaging
			}

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			securityGroup, err := findSecurityGroupByNameOrGUID(ctx, client, args[0])
			if err != nil {
				return err
			}

			orgGUID := viper.GetString("organization_guid")

			spaceGUIDs, err := findSpaceGUIDs(ctx, client, spaceNames, orgGUID)
			if err != nil {
				return err
			}

			return bindSecurityGroupToSpaces(ctx, client, securityGroup.GUID, securityGroup.Name,
				spaceGUIDs, spaceNames, running, staging)
		},
	}

	cmd.Flags().StringArrayVarP(&spaceNames, "spaces", "s", nil, "spaces to bind to (required)")
	cmd.Flags().BoolVar(&running, LifecycleRunning, false, "bind for running applications")
	cmd.Flags().BoolVar(&staging, LifecycleStaging, false, "bind for staging applications")
	_ = cmd.MarkFlagRequired("spaces")

	return cmd
}

// unbindSecurityGroupFromSpace handles unbinding logic for both running and staging.
func unbindSecurityGroupFromSpace(ctx context.Context, client capi.Client, sgGUID, sgName, spaceGUID, spaceName string, running, staging bool) error {
	if running {
		err := client.SecurityGroups().UnbindRunningSpace(ctx, sgGUID, spaceGUID)
		if err != nil {
			return fmt.Errorf("failed to unbind security group from running space: %w", err)
		}

		_, _ = fmt.Fprintf(os.Stdout, "Successfully unbound security group '%s' from running space '%s'\n", sgName, spaceName)
	}

	if staging {
		err := client.SecurityGroups().UnbindStagingSpace(ctx, sgGUID, spaceGUID)
		if err != nil {
			return fmt.Errorf("failed to unbind security group from staging space: %w", err)
		}

		_, _ = fmt.Fprintf(os.Stdout, "Successfully unbound security group '%s' from staging space '%s'\n", sgName, spaceName)
	}

	return nil
}

func newSecurityGroupsUnbindCommand() *cobra.Command {
	var (
		spaceName string
		running   bool
		staging   bool
	)

	cmd := &cobra.Command{
		Use:   "unbind SECURITY_GROUP_NAME_OR_GUID",
		Short: "Unbind security group from a space",
		Long:  "Remove a security group binding from a space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if spaceName == "" {
				return ErrSpaceNameRequired
			}

			if !running && !staging {
				return ErrMustSpecifyRunningOrStaging
			}

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			securityGroup, err := findSecurityGroupByNameOrGUID(ctx, client, args[0])
			if err != nil {
				return err
			}

			orgGUID := viper.GetString("organization_guid")

			space, err := findSpaceByName(ctx, client, spaceName, orgGUID)
			if err != nil {
				return err
			}

			return unbindSecurityGroupFromSpace(ctx, client, securityGroup.GUID, securityGroup.Name,
				space.GUID, spaceName, running, staging)
		},
	}

	cmd.Flags().StringVarP(&spaceName, spaceKey, "s", "", "space to unbind from (required)")
	cmd.Flags().BoolVar(&running, LifecycleRunning, false, "unbind from running applications")
	cmd.Flags().BoolVar(&staging, LifecycleStaging, false, "unbind from staging applications")
	_ = cmd.MarkFlagRequired(spaceKey)

	return cmd
}

func newSecurityGroupsRunningCommand() *cobra.Command {
	return createSecurityGroupListCommand(SecurityGroupListConfig{
		Use:        LifecycleRunning,
		Short:      "List running security groups",
		Long:       "List all security groups that are globally enabled for running applications",
		FilterKey:  "globally_enabled_running",
		NoItemsMsg: "No globally enabled running security groups found",
		ListTitle:  "Globally enabled running security groups:",
	})
}

func newSecurityGroupsStagingCommand() *cobra.Command {
	return createSecurityGroupListCommand(SecurityGroupListConfig{
		Use:        LifecycleStaging,
		Short:      "List staging security groups",
		Long:       "List all security groups that are globally enabled for staging applications",
		FilterKey:  "globally_enabled_staging",
		NoItemsMsg: "No globally enabled staging security groups found",
		ListTitle:  "Globally enabled staging security groups:",
	})
}
