package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewManifestsCommand creates the manifests command group.
func NewManifestsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manifests",
		Short: "Manage application manifests",
		Long:  "Manage Cloud Foundry application manifests including applying, generating, and diffing manifests",
	}

	cmd.AddCommand(newManifestsApplyCommand())
	cmd.AddCommand(newManifestsGenerateCommand())
	cmd.AddCommand(newManifestsDiffCommand())

	return cmd
}

// readManifestFile reads and validates the manifest file from the given path.
func readManifestFileBytes(manifestPath string) ([]byte, error) {
	cleanPath := filepath.Clean(manifestPath)
	if !filepath.IsAbs(cleanPath) {
		// Make relative paths relative to current directory
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}

		cleanPath = filepath.Join(wd, cleanPath)
	}

	manifestContent, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	return manifestContent, nil
}

// handleManifestOutput handles the output formatting for manifest operations.
func handleManifestOutput(job *capi.Job, wait bool, client capi.Client, ctx context.Context) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(job)
		if err != nil {
			return fmt.Errorf("failed to encode job to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(job)
		if err != nil {
			return fmt.Errorf("failed to encode job to YAML: %w", err)
		}

		return nil
	default:
		_, _ = os.Stdout.WriteString("Manifest application started\n")
		_, _ = fmt.Fprintf(os.Stdout, "Job ID: %s\n", job.GUID)
		_, _ = fmt.Fprintf(os.Stdout, "State: %s\n", job.State)

		if wait {
			_, _ = os.Stdout.WriteString("\nWaiting for job to complete...\n")

			completedJob, err := client.Jobs().PollUntilComplete(ctx, job.GUID)
			if err != nil {
				return fmt.Errorf("failed to wait for job: %w", err)
			}

			handleJobCompletion(completedJob)
		}
	}

	return nil
}

func newManifestsApplyCommand() *cobra.Command {
	var (
		manifestPath string
		wait         bool
	)

	cmd := &cobra.Command{
		Use:   "apply SPACE_GUID",
		Short: "Apply a manifest to a space",
		Long:  "Apply an application manifest to deploy or update applications in a space",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceGUID := args[0]

			// Read manifest file
			manifestContent, err := readManifestFileBytes(manifestPath)
			if err != nil {
				return err
			}

			// Create client
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			ctx := context.Background()

			// Apply manifest
			job, err := client.Manifests().ApplyManifest(ctx, spaceGUID, manifestContent)
			if err != nil {
				return fmt.Errorf("failed to apply manifest: %w", err)
			}

			// Handle output
			return handleManifestOutput(job, wait, client, ctx)
		},
	}

	cmd.Flags().StringVarP(&manifestPath, "file", "f", "manifest.yml", "Path to manifest file")
	cmd.Flags().BoolVarP(&wait, "wait", "w", false, "Wait for the job to complete")

	return cmd
}

func newManifestsGenerateCommand() *cobra.Command {
	var outputFile string

	cmd := &cobra.Command{
		Use:   "generate APP_GUID",
		Short: "Generate a manifest for an app",
		Long:  "Generate a manifest file from an existing application's current configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appGUID := args[0]

			// Create client
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			ctx := context.Background()

			// Generate manifest
			manifestContent, err := client.Manifests().GenerateManifest(ctx, appGUID)
			if err != nil {
				return fmt.Errorf("failed to generate manifest: %w", err)
			}

			// Write to file or stdout
			if outputFile != "" {
				err = os.WriteFile(outputFile, manifestContent, constants.ConfigFilePerm)
				if err != nil {
					return fmt.Errorf("failed to write manifest file: %w", err)
				}

				_, _ = fmt.Fprintf(os.Stdout, "Manifest written to %s\n", outputFile)
			} else {
				_, err := os.Stdout.Write(manifestContent)
				if err != nil {
					return fmt.Errorf("failed to write manifest to stdout: %w", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output-file", "o", "", "Output file path (defaults to stdout)")

	return cmd
}

// handleDiffOutput handles the output formatting for manifest diff operations.
func handleDiffOutput(diff *capi.ManifestDiff) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(diff)
		if err != nil {
			return fmt.Errorf("failed to encode diff to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(diff)
		if err != nil {
			return fmt.Errorf("failed to encode diff to YAML: %w", err)
		}

		return nil
	default:
		if len(diff.Diff) == 0 {
			_, _ = os.Stdout.WriteString("No differences found\n")
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "Found %d difference(s):\n\n", len(diff.Diff))

			table := tablewriter.NewWriter(os.Stdout)
			table.Header("Operation", "Path", "Current Value", "New Value")

			for _, entry := range diff.Diff {
				wasStr := formatValue(entry.Was)
				valueStr := formatValue(entry.Value)
				_ = table.Append(entry.Op, entry.Path, wasStr, valueStr)
			}

			err := table.Render()
			if err != nil {
				return fmt.Errorf("failed to render table: %w", err)
			}
		}
	}

	return nil
}

func newManifestsDiffCommand() *cobra.Command {
	var manifestPath string

	cmd := &cobra.Command{
		Use:   "diff SPACE_GUID",
		Short: "Create a diff between current and proposed manifest",
		Long:  "Compare the current state of applications in a space with a proposed manifest to see what would change",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceGUID := args[0]

			// Read manifest file
			manifestContent, err := readManifestFileBytes(manifestPath)
			if err != nil {
				return err
			}

			// Create client
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			ctx := context.Background()

			// Create diff
			diff, err := client.Manifests().CreateManifestDiff(ctx, spaceGUID, manifestContent)
			if err != nil {
				return fmt.Errorf("failed to create manifest diff: %w", err)
			}

			// Handle output
			return handleDiffOutput(diff)
		},
	}

	cmd.Flags().StringVarP(&manifestPath, "file", "f", "manifest.yml", "Path to manifest file")

	return cmd
}

// handleJobCompletion handles the display of job completion results.
func handleJobCompletion(completedJob any) {
	// Use reflection to access job fields since we don't know the exact type
	jobValue := reflect.ValueOf(completedJob)

	stateField := jobValue.FieldByName("State")
	if !stateField.IsValid() {
		return
	}

	state := stateField.String()
	if state == "COMPLETE" {
		_, _ = os.Stdout.WriteString("✓ Manifest applied successfully\n")

		return
	}

	_, _ = fmt.Fprintf(os.Stdout, "Job finished with state: %s\n", state)

	// Check for errors
	errorsField := jobValue.FieldByName("Errors")
	if errorsField.IsValid() && errorsField.Len() > 0 {
		_, _ = os.Stdout.WriteString("\nErrors:\n")

		for i := range errorsField.Len() {
			errorItem := errorsField.Index(i)
			title := errorItem.FieldByName("Title").String()
			detail := errorItem.FieldByName("Detail").String()
			_, _ = fmt.Fprintf(os.Stdout, "  - %s: %s\n", title, detail)
		}
	}
}

// formatValue formats a value for display in the diff table.
func formatValue(v any) string {
	if v == nil {
		return "-"
	}

	switch val := v.(type) {
	case string:
		return val
	case bool:
		return strconv.FormatBool(val)
	case float64:
		// Check if it's an integer
		if val == float64(int(val)) {
			return strconv.Itoa(int(val))
		}

		return fmt.Sprintf("%g", val)
	default:
		// For complex types, use JSON
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}

		return string(b)
	}
}
