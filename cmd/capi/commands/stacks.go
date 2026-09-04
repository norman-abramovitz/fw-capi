package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"os"
	"strconv"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewStacksCommand creates the stacks command group.
// fetchAllStackPages fetches all pages of stacks if allPages is true.
func fetchAllStackPages(ctx context.Context, client capi.Client, params *capi.QueryParams, initialStacks *capi.ListResponse[capi.Stack], allPages bool) ([]capi.Stack, error) {
	allStacks := initialStacks.Resources
	if allPages && initialStacks.Pagination.TotalPages > 1 {
		for page := 2; page <= initialStacks.Pagination.TotalPages; page++ {
			params.Page = page

			moreStacks, err := client.Stacks().List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
			}

			allStacks = append(allStacks, moreStacks.Resources...)
		}
	}

	return allStacks, nil
}

// fetchAllAppPages fetches all pages of apps if allPages is true.
func fetchAllStackAppPages(ctx context.Context, client capi.Client, stackGUID string, params *capi.QueryParams, initialApps *capi.ListResponse[capi.App], allPages bool) ([]capi.App, error) {
	allApps := initialApps.Resources
	if allPages && initialApps.Pagination.TotalPages > 1 {
		for page := 2; page <= initialApps.Pagination.TotalPages; page++ {
			params.Page = page

			moreApps, err := client.Stacks().ListApps(ctx, stackGUID, params)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
			}

			allApps = append(allApps, moreApps.Resources...)
		}
	}

	return allApps, nil
}

// renderStacksOutput handles the output rendering for stacks based on the format.
func renderStacksOutput(output string, stacks []capi.Stack) error {
	switch output {
	case OutputFormatJSON:
		return renderJSONOutput(stacks)
	case OutputFormatYAML:
		return renderYAMLOutput(stacks)
	default:
		return renderStacksTable(stacks)
	}
}

// renderAppsOutput handles the output rendering for apps based on the format.
func renderAppsOutput(output string, apps []capi.App) error {
	switch output {
	case OutputFormatJSON:
		return renderJSONOutput(apps)
	case OutputFormatYAML:
		return renderYAMLOutput(apps)
	default:
		return renderAppsTable(apps)
	}
}

// renderJSONOutput renders data as JSON.
func renderJSONOutput(data any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(data)
	if err != nil {
		return fmt.Errorf("encoding data to JSON: %w", err)
	}

	return nil
}

// renderYAMLOutput renders data as YAML.
func renderYAMLOutput(data any) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode data to YAML: %w", err)
	}

	return nil
}

// renderStacksTable renders stacks in table format.
func renderStacksTable(stacks []capi.Stack) error {
	if len(stacks) == 0 {
		_, _ = os.Stdout.WriteString("No stacks found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "Description", "Default", "Build Image", "Run Image", "Created", "Updated")

	for _, stack := range stacks {
		defaultStack := False
		if stack.Default {
			defaultStack = True
		}

		buildImage := truncateString(stack.BuildRootfsImage, constants.ShortCommandDisplayLength)
		runImage := truncateString(stack.RunRootfsImage, constants.ShortCommandDisplayLength)

		_ = table.Append([]string{
			stack.Name,
			stack.Description,
			defaultStack,
			buildImage,
			runImage,
			stack.CreatedAt.Format(TimeFormatDisplay),
			stack.UpdatedAt.Format(TimeFormatDisplay),
		})
	}

	_ = table.Render()

	return nil
}

// renderAppsTable renders apps in table format.
func renderAppsTable(apps []capi.App) error {
	if len(apps) == 0 {
		_, _ = os.Stdout.WriteString("No apps found using this stack\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "GUID", "State", "Lifecycle", "Created", "Updated")

	for _, app := range apps {
		lifecycle := formatAppLifecycle(app)
		_ = table.Append([]string{
			app.Name,
			app.GUID,
			app.State,
			lifecycle,
			app.CreatedAt.Format(TimeFormatDisplay),
			app.UpdatedAt.Format(TimeFormatDisplay),
		})
	}

	_ = table.Render()

	return nil
}

// truncateString truncates a string to maxLength with ellipsis.
func truncateString(s string, maxLength int) string {
	if len(s) > maxLength {
		return s[:37] + "..."
	}

	return s
}

// formatAppLifecycle formats the lifecycle information for an app.
func formatAppLifecycle(app capi.App) string {
	lifecycle := app.Lifecycle.Type
	if len(app.Lifecycle.Data) > 0 {
		if stackName, ok := app.Lifecycle.Data["stack"]; ok {
			lifecycle += fmt.Sprintf(" (%v)", stackName)
		}
	}

	return lifecycle
}

func NewStacksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stacks",
		Aliases: []string{"stack"},
		Short:   "Manage stacks",
		Long:    "List and manage Cloud Foundry stacks",
	}

	cmd.AddCommand(newStacksListCommand())
	cmd.AddCommand(newStacksGetCommand())
	cmd.AddCommand(newStacksCreateCommand())
	cmd.AddCommand(newStacksUpdateCommand())
	cmd.AddCommand(newStacksDeleteCommand())
	cmd.AddCommand(newStacksListAppsCommand())

	return cmd
}

func newStacksListCommand() *cobra.Command {
	var (
		allPages bool
		perPage  int
		name     string
	)

	cmd := &cobra.Command{
		Use:   List,
		Short: "List stacks",
		Long:  "List all stacks available in the platform",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			params := capi.NewQueryParams()
			params.PerPage = perPage

			// Apply name filter if specified
			if name != "" {
				params.WithFilter("names", name)
			}

			stacks, err := client.Stacks().List(ctx, params)
			if err != nil {
				return fmt.Errorf("failed to list stacks: %w", err)
			}

			// Fetch all pages if requested
			allStacks, err := fetchAllStackPages(ctx, client, params, stacks, allPages)
			if err != nil {
				return err
			}

			// Output results
			return renderStacksOutput(viper.GetString("output"), allStacks)
		},
	}

	cmd.Flags().BoolVar(&allPages, "all-pages", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "number of results per page")
	cmd.Flags().StringVar(&name, "name", "", "filter by stack name")

	return cmd
}

func newStacksGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get STACK_GUID",
		Short: "Get stack details",
		Long:  "Display detailed information about a specific stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stackGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			stack, err := client.Stacks().Get(ctx, stackGUID)
			if err != nil {
				return fmt.Errorf("failed to get stack: %w", err)
			}

			// Output results
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				err := encoder.Encode(stack)
				if err != nil {
					return fmt.Errorf("encoding stack to JSON: %w", err)
				}

				return nil
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				err := encoder.Encode(stack)
				if err != nil {
					return fmt.Errorf("encoding stack to YAML: %w", err)
				}

				return nil
			default:
				table := tablewriter.NewWriter(os.Stdout)
				table.Header("Property", "Value")
				_ = table.Append("GUID", stack.GUID)
				_ = table.Append("Name", stack.Name)
				_ = table.Append("Description", stack.Description)
				_ = table.Append("Default", strconv.FormatBool(stack.Default))
				_ = table.Append("Build Image", stack.BuildRootfsImage)
				_ = table.Append("Run Image", stack.RunRootfsImage)
				_ = table.Append("Created", stack.CreatedAt.Format(TimeFormatDisplay))
				_ = table.Append("Updated", stack.UpdatedAt.Format(TimeFormatDisplay))

				err := table.Render()
				if err != nil {
					return fmt.Errorf("failed to render table: %w", err)
				}
			}

			return nil
		},
	}
}

func newStacksCreateCommand() *cobra.Command {
	var (
		description      string
		buildRootfsImage string
		runRootfsImage   string
	)

	cmd := &cobra.Command{
		Use:   "create STACK_NAME",
		Short: "Create a stack",
		Long:  "Create a new stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stackName := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			request := &capi.StackCreateRequest{
				Name:             stackName,
				Description:      description,
				BuildRootfsImage: buildRootfsImage,
				RunRootfsImage:   runRootfsImage,
			}

			stack, err := client.Stacks().Create(ctx, request)
			if err != nil {
				return fmt.Errorf("failed to create stack: %w", err)
			}

			// Output results
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				err := encoder.Encode(stack)
				if err != nil {
					return fmt.Errorf("encoding stack to JSON: %w", err)
				}

				return nil
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				err := encoder.Encode(stack)
				if err != nil {
					return fmt.Errorf("encoding stack to YAML: %w", err)
				}

				return nil
			default:
				_, _ = fmt.Fprintf(os.Stdout, "Created stack: %s\n", stack.GUID)
				_, _ = fmt.Fprintf(os.Stdout, "  Name:        %s\n", stack.Name)
				_, _ = fmt.Fprintf(os.Stdout, "  Description: %s\n", stack.Description)
				_, _ = fmt.Fprintf(os.Stdout, "  Build Image: %s\n", stack.BuildRootfsImage)
				_, _ = fmt.Fprintf(os.Stdout, "  Run Image:   %s\n", stack.RunRootfsImage)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "stack description")
	cmd.Flags().StringVar(&buildRootfsImage, "build-image", "", "build rootfs image")
	cmd.Flags().StringVar(&runRootfsImage, "run-image", "", "run rootfs image")

	return cmd
}

func newStacksUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update STACK_GUID",
		Short: "Update a stack",
		Long:  "Update stack metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stackGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// For now, stacks only support metadata updates
			request := &capi.StackUpdateRequest{
				Metadata: &capi.Metadata{
					Labels:      make(map[string]*string),
					Annotations: make(map[string]*string),
				},
			}

			stack, err := client.Stacks().Update(ctx, stackGUID, request)
			if err != nil {
				return fmt.Errorf("failed to update stack: %w", err)
			}

			// Output results
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				err := encoder.Encode(stack)
				if err != nil {
					return fmt.Errorf("encoding stack to JSON: %w", err)
				}

				return nil
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				err := encoder.Encode(stack)
				if err != nil {
					return fmt.Errorf("encoding stack to YAML: %w", err)
				}

				return nil
			default:
				_, _ = fmt.Fprintf(os.Stdout, "Updated stack: %s\n", stack.GUID)
				_, _ = fmt.Fprintf(os.Stdout, "  Name:        %s\n", stack.Name)
				_, _ = fmt.Fprintf(os.Stdout, "  Description: %s\n", stack.Description)
			}

			return nil
		},
	}

	return cmd
}

func newStacksDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete STACK_GUID",
		Short: "Delete a stack",
		Long:  "Delete a stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stackGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			err = client.Stacks().Delete(ctx, stackGUID)
			if err != nil {
				return fmt.Errorf("failed to delete stack: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Stack %s deleted successfully\n", stackGUID)

			return nil
		},
	}
}

func newStacksListAppsCommand() *cobra.Command {
	var (
		allPages bool
		perPage  int
	)

	cmd := &cobra.Command{
		Use:   "list-apps STACK_GUID",
		Short: "List apps using a stack",
		Long:  "List all applications that are using the specified stack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stackGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			params := capi.NewQueryParams()
			params.PerPage = perPage

			apps, err := client.Stacks().ListApps(ctx, stackGUID, params)
			if err != nil {
				return fmt.Errorf("failed to list apps for stack: %w", err)
			}

			// Fetch all pages if requested
			allApps, err := fetchAllStackAppPages(ctx, client, stackGUID, params, apps, allPages)
			if err != nil {
				return err
			}

			// Output results
			return renderAppsOutput(viper.GetString("output"), allApps)
		},
	}

	cmd.Flags().BoolVar(&allPages, "all-pages", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "number of results per page")

	return cmd
}
