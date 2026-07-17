package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"os"
	"strings"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewSidecarsCommand creates the sidecars command group.
func NewSidecarsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sidecars",
		Aliases: []string{"sidecar", "sc"},
		Short:   "Manage application sidecars",
		Long:    "View, update, and delete sidecars for applications",
	}

	cmd.AddCommand(newSidecarsGetCommand())
	cmd.AddCommand(newSidecarsUpdateCommand())
	cmd.AddCommand(newSidecarsDeleteCommand())
	cmd.AddCommand(newSidecarsListForProcessCommand())

	return cmd
}

func newSidecarsGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get SIDECAR_GUID",
		Short: "Get sidecar details",
		Long:  "Display detailed information about a specific sidecar",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sidecarGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			sidecar, err := client.Sidecars().Get(ctx, sidecarGUID)
			if err != nil {
				return fmt.Errorf("failed to get sidecar: %w", err)
			}

			// Output results
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				return encoder.Encode(sidecar)
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				return encoder.Encode(sidecar)
			default:
				table := tablewriter.NewWriter(os.Stdout)
				table.Header("Property", "Value")

				_ = table.Append("Name", sidecar.Name)
				_ = table.Append("GUID", sidecar.GUID)
				_ = table.Append("Command", sidecar.Command)
				_ = table.Append("Process Types", strings.Join(sidecar.ProcessTypes, ", "))

				if sidecar.MemoryInMB != nil {
					_ = table.Append("Memory", fmt.Sprintf("%d MB", *sidecar.MemoryInMB))
				} else {
					_ = table.Append("Memory", "default")
				}

				_ = table.Append("Origin", sidecar.Origin)
				_ = table.Append("Created", sidecar.CreatedAt.Format(TimeFormatDisplay))
				_ = table.Append("Updated", sidecar.UpdatedAt.Format(TimeFormatDisplay))

				if sidecar.Relationships.App.Data != nil {
					_ = table.Append("App GUID", sidecar.Relationships.App.Data.GUID)
				}

				_, _ = os.Stdout.WriteString("Sidecar details:\n\n")
				_ = table.Render()
			}

			return nil
		},
	}
}

func newSidecarsUpdateCommand() *cobra.Command {
	var (
		name         string
		command      string
		processTypes []string
		memoryInMB   int
	)

	cmd := &cobra.Command{
		Use:   "update SIDECAR_GUID",
		Short: "Update a sidecar",
		Long:  "Update an existing sidecar configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sidecarGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Build update request
			updateReq := &capi.SidecarUpdateRequest{}

			if name != "" {
				updateReq.Name = &name
			}

			if command != "" {
				updateReq.Command = &command
			}

			if len(processTypes) > 0 {
				updateReq.ProcessTypes = processTypes
			}

			if cmd.Flags().Changed("memory") {
				updateReq.MemoryInMB = &memoryInMB
			}

			// Update sidecar
			updatedSidecar, err := client.Sidecars().Update(ctx, sidecarGUID, updateReq)
			if err != nil {
				return fmt.Errorf("failed to update sidecar: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully updated sidecar '%s'\n", updatedSidecar.Name)

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "sidecar name")
	cmd.Flags().StringVar(&command, "command", "", "sidecar command")
	cmd.Flags().StringSliceVar(&processTypes, "process-types", nil, "process types (comma-separated)")
	cmd.Flags().IntVar(&memoryInMB, "memory", 0, "memory limit in MB")

	return cmd
}

func newSidecarsDeleteCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete SIDECAR_GUID",
		Short: "Delete a sidecar",
		Long:  "Delete a sidecar from an application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sidecarGUID := args[0]

			if !force {
				_, _ = fmt.Fprintf(os.Stdout, "Really delete sidecar '%s'? (y/N): ", sidecarGUID)

				var response string

				_, _ = fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					_, _ = os.Stdout.WriteString("Cancelled\n")

					return nil
				}
			}

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Get sidecar name for confirmation
			sidecar, err := client.Sidecars().Get(ctx, sidecarGUID)
			if err != nil {
				return fmt.Errorf("failed to get sidecar: %w", err)
			}

			// Delete sidecar
			err = client.Sidecars().Delete(ctx, sidecarGUID)
			if err != nil {
				return fmt.Errorf("failed to delete sidecar: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully deleted sidecar '%s'\n", sidecar.Name)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "force deletion without confirmation")

	return cmd
}

func newSidecarsListForProcessCommand() *cobra.Command {
	var (
		allPages bool
		perPage  int
	)

	cmd := &cobra.Command{
		Use:   "list-for-process PROCESS_GUID",
		Short: "List sidecars for a process",
		Long:  "List all sidecars associated with a specific process",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			processGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()
			params := capi.NewQueryParams()
			params.PerPage = perPage

			sidecars, err := client.Sidecars().ListForProcess(ctx, processGUID, params)
			if err != nil {
				return fmt.Errorf("failed to list sidecars for process: %w", err)
			}

			allSidecars, err := fetchAllSidecarPages(ctx, client, processGUID, sidecars, params, allPages)
			if err != nil {
				return err
			}

			return outputSidecarsForProcess(processGUID, allSidecars, sidecars.Pagination, allPages)
		},
	}

	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.StandardPageSize, "results per page")

	return cmd
}

// fetchAllSidecarPages fetches all pages of sidecars if requested.
func fetchAllSidecarPages(ctx context.Context, client capi.Client, processGUID string, initial *capi.ListResponse[capi.Sidecar], params *capi.QueryParams, allPages bool) ([]capi.Sidecar, error) {
	allSidecars := initial.Resources

	if !allPages || initial.Pagination.TotalPages <= 1 {
		return allSidecars, nil
	}

	for page := 2; page <= initial.Pagination.TotalPages; page++ {
		params.Page = page

		moreSidecars, err := client.Sidecars().ListForProcess(ctx, processGUID, params)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		allSidecars = append(allSidecars, moreSidecars.Resources...)
	}

	return allSidecars, nil
}

// outputSidecarsForProcess handles output formatting for sidecars.
func outputSidecarsForProcess(processGUID string, sidecars []capi.Sidecar, pagination capi.Pagination, allPages bool) error {
	output := viper.GetString("output")

	switch output {
	case OutputFormatJSON:
		return outputSidecarsJSON(sidecars)
	case OutputFormatYAML:
		return outputSidecarsYAML(sidecars)
	default:
		return outputSidecarsTable(processGUID, sidecars, &pagination, allPages)
	}
}

// outputSidecarsJSON outputs sidecars in JSON format.
func outputSidecarsJSON(sidecars []capi.Sidecar) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(sidecars)
	if err != nil {
		return fmt.Errorf("encoding sidecars to JSON: %w", err)
	}

	return nil
}

// outputSidecarsYAML outputs sidecars in YAML format.
func outputSidecarsYAML(sidecars []capi.Sidecar) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(sidecars)
	if err != nil {
		return fmt.Errorf("encoding sidecars to YAML: %w", err)
	}

	return nil
}

// outputSidecarsTable outputs sidecars in table format.
func outputSidecarsTable(processGUID string, sidecars []capi.Sidecar, pagination *capi.Pagination, allPages bool) error {
	if len(sidecars) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No sidecars found for process %s\n", processGUID)

		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "Sidecars for process %s:\n\n", processGUID)

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "GUID", "Command", "Process Types", "Memory", "Origin", "Created")

	for _, sidecar := range sidecars {
		_ = table.Append(formatSidecarRow(sidecar)...)
	}

	_ = table.Render()

	if !allPages && pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", pagination.TotalPages)
	}

	return nil
}

// formatSidecarRow formats a single sidecar for table display.
func formatSidecarRow(sidecar capi.Sidecar) []any {
	memoryStr := formatSidecarMemory(sidecar.MemoryInMB)
	processTypesStr := truncateSidecarString(strings.Join(sidecar.ProcessTypes, ", "), constants.ProcessTypesDisplayLength)
	commandStr := truncateSidecarString(sidecar.Command, constants.ShortCommandDisplayLength)

	return []any{
		sidecar.Name,
		sidecar.GUID,
		commandStr,
		processTypesStr,
		memoryStr,
		sidecar.Origin,
		sidecar.CreatedAt.Format("2006-01-02"),
	}
}

// formatSidecarMemory formats memory value for display.
func formatSidecarMemory(memoryInMB *int) string {
	if memoryInMB == nil {
		return "default"
	}

	return fmt.Sprintf("%d MB", *memoryInMB)
}

// truncateSidecarString truncates a string to specified length with ellipsis.
func truncateSidecarString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-3] + "..."
}
