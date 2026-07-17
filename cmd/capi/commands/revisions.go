package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewRevisionsCommand creates the revisions command group.
func NewRevisionsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "revisions",
		Aliases: []string{"revision", "rev"},
		Short:   "Manage application revisions",
		Long:    "View and manage application revisions",
	}

	cmd.AddCommand(newRevisionsGetCommand())
	cmd.AddCommand(newRevisionsUpdateCommand())
	cmd.AddCommand(newRevisionsGetEnvCommand())

	return cmd
}

func newRevisionsGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get REVISION_GUID",
		Short: "Get revision details",
		Long:  "Display detailed information about a specific revision",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			revisionGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			revision, err := client.Revisions().Get(ctx, revisionGUID)
			if err != nil {
				return fmt.Errorf("failed to get revision: %w", err)
			}

			return renderRevision(revision)
		},
	}
}

func renderRevision(revision *capi.Revision) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(revision)
		if err != nil {
			return fmt.Errorf("failed to encode revision as JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(revision)
		if err != nil {
			return fmt.Errorf("failed to encode revision as YAML: %w", err)
		}

		return nil
	default:
		renderRevisionTable(revision)
		renderRevisionProcesses(revision)
		renderRevisionSidecars(revision)
	}

	return nil
}

func renderRevisionTable(revision *capi.Revision) {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	_ = table.Append("Version", strconv.Itoa(revision.Version))
	_ = table.Append("GUID", revision.GUID)

	_ = table.Append("Deployable", strconv.FormatBool(revision.Deployable))
	if revision.Description != nil {
		_ = table.Append("Description", *revision.Description)
	}

	_ = table.Append("Created", revision.CreatedAt.Format(TimeFormatDisplay))
	_ = table.Append("Updated", revision.UpdatedAt.Format(TimeFormatDisplay))
	_ = table.Append("Droplet GUID", revision.Droplet.GUID)

	if revision.Relationships.App.Data != nil {
		_ = table.Append("App GUID", revision.Relationships.App.Data.GUID)
	}

	_, _ = os.Stdout.WriteString("Revision Details:\n\n")

	_ = table.Render()
}

func renderRevisionProcesses(revision *capi.Revision) {
	if len(revision.Processes) == 0 {
		return
	}

	_, _ = os.Stdout.WriteString("\nProcesses:\n")

	processTable := tablewriter.NewWriter(os.Stdout)
	processTable.Header("Type", "GUID", "Instances", "Memory", "Disk", "Command")

	for processType, process := range revision.Processes {
		command := ""
		if process.Command != nil {
			command = *process.Command
			if len(command) > constants.ShortCommandDisplayLength {
				command = command[:37] + "..."
			}
		}

		_ = processTable.Append(
			processType,
			process.GUID,
			strconv.Itoa(process.Instances),
			fmt.Sprintf("%d MB", process.MemoryInMB),
			fmt.Sprintf("%d MB", process.DiskInMB),
			command,
		)
	}

	_ = processTable.Render()
}

func renderRevisionSidecars(revision *capi.Revision) {
	if len(revision.Sidecars) == 0 {
		return
	}

	_, _ = os.Stdout.WriteString("\nSidecars:\n")

	sidecarTable := tablewriter.NewWriter(os.Stdout)
	sidecarTable.Header("Name", "GUID", "Command", "Memory")

	for _, sidecar := range revision.Sidecars {
		memory := NotAvailable
		if sidecar.MemoryInMB != nil {
			memory = fmt.Sprintf("%d MB", *sidecar.MemoryInMB)
		}

		command := sidecar.Command
		if len(command) > constants.ShortCommandDisplayLength {
			command = command[:37] + "..."
		}

		_ = sidecarTable.Append(sidecar.Name, sidecar.GUID, command, memory)
	}

	_ = sidecarTable.Render()
}

func newRevisionsUpdateCommand() *cobra.Command {
	var (
		metadata map[string]string
	)

	cmd := &cobra.Command{
		Use:   "update REVISION_GUID",
		Short: "Update a revision",
		Long:  "Update a revision's metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			revisionGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Build update request
			updateReq := &capi.RevisionUpdateRequest{}

			if len(metadata) > 0 {
				updateReq.Metadata = &capi.Metadata{
					Labels: metadata,
				}
			}

			// Update revision
			updatedRevision, err := client.Revisions().Update(ctx, revisionGUID, updateReq)
			if err != nil {
				return fmt.Errorf("failed to update revision: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully updated revision %d\n", updatedRevision.Version)

			return nil
		},
	}

	cmd.Flags().StringToStringVar(&metadata, "metadata", nil, "metadata labels to apply (key=value)")

	return cmd
}

func newRevisionsGetEnvCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get-env REVISION_GUID",
		Short: "Get revision environment variables",
		Long:  "Display environment variables for a specific revision",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			revisionGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			envVars, err := client.Revisions().GetEnvironmentVariables(ctx, revisionGUID)
			if err != nil {
				return fmt.Errorf("failed to get revision environment variables: %w", err)
			}

			return renderRevisionEnvVars(revisionGUID, envVars)
		},
	}
}

func renderRevisionEnvVars(revisionGUID string, envVars map[string]any) error {
	envVarsList := make([]EnvVar, 0, len(envVars))
	for key, value := range envVars {
		envVarsList = append(envVarsList, EnvVar{
			Name:  key,
			Value: value,
		})
	}

	// Output results
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(envVarsList)
		if err != nil {
			return fmt.Errorf("failed to encode environment variables to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(envVarsList)
		if err != nil {
			return fmt.Errorf("failed to encode environment variables to YAML: %w", err)
		}

		return nil
	default:
		return renderRevisionEnvVarsTable(revisionGUID, envVarsList)
	}
}

func renderRevisionEnvVarsTable(revisionGUID string, envVarsList []EnvVar) error {
	if len(envVarsList) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No environment variables found for revision %s\n", revisionGUID)

		return nil
	}

	_, _ = fmt.Fprintf(os.Stdout, "Environment variables for revision %s:\n\n", revisionGUID)

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "Value")

	for _, envVar := range envVarsList {
		valueStr := fmt.Sprintf("%v", envVar.Value)
		// Truncate long values for table display
		if len(valueStr) > constants.StringTruncationLength {
			valueStr = valueStr[:77] + "..."
		}

		_ = table.Append(envVar.Name, valueStr)
	}

	_ = table.Render()

	return nil
}
