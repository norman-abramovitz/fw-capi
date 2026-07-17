package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewResourceMatchesCommand creates the resource matches command group.
func NewResourceMatchesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "resource-matches",
		Aliases: []string{"resources", "resource", "rm"},
		Short:   "Manage resource matches",
		Long:    "Create resource matches for optimizing package uploads",
	}

	cmd.AddCommand(newResourceMatchesCreateCommand())

	return cmd
}

// validateFilePathResourceMatches validates that a file path is safe to read.
func validateFilePathResourceMatches(filePath string) error {
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

func newResourceMatchesCreateCommand() *cobra.Command {
	var (
		fromFile  string
		resources []string
	)

	cmd := &cobra.Command{
		Use:   Create,
		Short: "Create resource matches",
		Long:  "Create resource matches to check which resources already exist on the platform",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			resourceList, err := buildResourceList(fromFile, resources)
			if err != nil {
				return err
			}

			if len(resourceList) == 0 {
				return ErrNoResourcesSpecified
			}

			// Create resource matches request
			request := &capi.ResourceMatchesRequest{
				Resources: resourceList,
			}

			matches, err := client.ResourceMatches().Create(ctx, request)
			if err != nil {
				return fmt.Errorf("failed to create resource matches: %w", err)
			}

			return outputResourceMatches(matches, resourceList)
		},
	}

	cmd.Flags().StringVarP(&fromFile, "from-file", "f", "", "load resources from file (JSON or YAML format)")
	cmd.Flags().StringSliceVarP(&resources, "resource", "r", nil, "resource specification (sha1:size:path:mode)")

	return cmd
}

// parseResourceString parses a resource string in the format "sha1:size:path:mode".
func parseResourceString(resourceStr string) (capi.ResourceMatch, error) {
	parts := strings.Split(resourceStr, ":")
	if len(parts) != constants.FilePathPartsCount {
		return capi.ResourceMatch{}, ErrInvalidResourceFormat
	}

	sha1 := parts[0]
	sizeStr := parts[1]
	path := parts[2]
	mode := parts[3]

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return capi.ResourceMatch{}, fmt.Errorf("invalid size '%s': %w", sizeStr, err)
	}

	return capi.ResourceMatch{
		SHA1: sha1,
		Size: size,
		Path: path,
		Mode: mode,
	}, nil
}

// buildResourceList builds the resource list from file and command-line arguments.
func buildResourceList(fromFile string, resources []string) ([]capi.ResourceMatch, error) {
	resourceList := make([]capi.ResourceMatch, 0, len(resources))

	// Load resources from file if specified
	if fromFile != "" {
		// Validate file path to prevent directory traversal
		err := validateFilePathResourceMatches(fromFile)
		if err != nil {
			return nil, fmt.Errorf("invalid resource file: %w", err)
		}

		fileResources, err := loadResourcesFromFile(fromFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load resources from file: %w", err)
		}

		resourceList = append(resourceList, fileResources...)
	}

	// Parse individual resource entries
	for _, resourceStr := range resources {
		resource, err := parseResourceString(resourceStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse resource '%s': %w", resourceStr, err)
		}

		resourceList = append(resourceList, resource)
	}

	return resourceList, nil
}

// outputResourceMatches handles the output formatting for resource matches.
func outputResourceMatches(matches *capi.ResourceMatches, resourceList []capi.ResourceMatch) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(matches)
		if err != nil {
			return fmt.Errorf("failed to encode matches to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(matches)
		if err != nil {
			return fmt.Errorf("failed to encode matches to YAML: %w", err)
		}

		return nil
	default:
		return renderResourceMatchesTable(matches, resourceList)
	}
}

// renderResourceMatchesTable renders the resource matches in table format.
func renderResourceMatchesTable(matches *capi.ResourceMatches, resourceList []capi.ResourceMatch) error {
	if len(matches.Resources) == 0 {
		_, _ = os.Stdout.WriteString("No matching resources found on the platform\n")
		_, _ = fmt.Fprintf(os.Stdout, "All %d resources will need to be uploaded\n", len(resourceList))

		return nil
	}

	_, _ = os.Stdout.WriteString("Resource Matches Summary:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  Total resources checked: %d\n", len(resourceList))
	_, _ = fmt.Fprintf(os.Stdout, "  Matching resources found: %d\n", len(matches.Resources))
	_, _ = fmt.Fprintf(os.Stdout, "  Resources to upload: %d\n", len(resourceList)-len(matches.Resources))
	_, _ = os.Stdout.WriteString("\n")

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("SHA1", "Size", "Path", "Mode")

	for _, resource := range matches.Resources {
		_ = table.Append(
			resource.SHA1,
			strconv.FormatInt(resource.Size, 10),
			resource.Path,
			resource.Mode,
		)
	}

	_, _ = os.Stdout.WriteString("Matching resources (already exist on platform):\n")

	_ = table.Render()

	printUploadOptimization(resourceList, matches.Resources)

	return nil
}

// printUploadOptimization prints the upload optimization statistics.
func printUploadOptimization(resourceList []capi.ResourceMatch, matchedResources []capi.ResourceMatch) {
	var (
		totalSizeToUpload int64
		totalSizeMatched  int64
	)

	for _, resource := range resourceList {
		totalSizeToUpload += resource.Size
	}

	for _, resource := range matchedResources {
		totalSizeMatched += resource.Size
	}

	sizeSavings := totalSizeMatched

	_, _ = os.Stdout.WriteString("\nUpload optimization:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  Total size without optimization: %d bytes\n", totalSizeToUpload)
	_, _ = fmt.Fprintf(os.Stdout, "  Size of matching resources: %d bytes\n", totalSizeMatched)
	_, _ = fmt.Fprintf(os.Stdout, "  Actual upload size needed: %d bytes\n", totalSizeToUpload-totalSizeMatched)

	if totalSizeToUpload > 0 {
		percentSaved := float64(sizeSavings) / float64(totalSizeToUpload) * constants.PercentageMultiplierFloat
		_, _ = fmt.Fprintf(os.Stdout, "  Bandwidth saved: %.1f%%\n", percentSaved)
	}
}

// loadResourcesFromFile loads resources from a JSON or YAML file.
func loadResourcesFromFile(filename string) ([]capi.ResourceMatch, error) {
	data, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var resources []capi.ResourceMatch

	// Try to parse as JSON first
	err = json.Unmarshal(data, &resources)
	if err == nil {
		return resources, nil
	}

	// Try to parse as YAML
	err = yaml.Unmarshal(data, &resources)
	if err == nil {
		return resources, nil
	}

	// Try to parse as a ResourceMatchesRequest (with resources field)
	var request struct {
		Resources []capi.ResourceMatch `json:"resources" yaml:"resources"`
	}

	err = json.Unmarshal(data, &request)
	if err == nil && len(request.Resources) > 0 {
		return request.Resources, nil
	}

	err = yaml.Unmarshal(data, &request)
	if err == nil && len(request.Resources) > 0 {
		return request.Resources, nil
	}

	return nil, ErrFailedToParseResourceFile
}
