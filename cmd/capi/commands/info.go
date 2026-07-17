package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewInfoCommand creates the info command.
func runInfo(cmd *cobra.Command, args []string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	info, err := client.GetInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get API info: %w", err)
	}

	return renderInfo(info)
}

func renderInfo(info *capi.Info) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(info)
		if err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(info)
		if err != nil {
			return fmt.Errorf("failed to encode YAML: %w", err)
		}

		return nil
	default:
		return renderInfoTable(info)
	}
}

func renderInfoTable(info *capi.Info) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	_ = table.Append("Name", info.Name)
	_ = table.Append("Build", info.Build)
	_ = table.Append("Version", strconv.Itoa(info.Version))
	_ = table.Append("Description", info.Description)
	_ = table.Append("CF on K8s", strconv.FormatBool(info.CFOnK8s))
	_ = table.Append("CLI Minimum", info.CLIVersion.Minimum)
	_ = table.Append("CLI Recommended", info.CLIVersion.Recommended)

	// Add links
	if len(info.Links) > 0 {
		linkStrings := make([]string, 0, len(info.Links))
		for name, link := range info.Links {
			linkStrings = append(linkStrings, fmt.Sprintf("%s: %s", name, link.Href))
		}

		_ = table.Append("Links", strings.Join(linkStrings, "\n"))
	}

	// Add custom fields if present
	if len(info.Custom) > 0 {
		customStrings := make([]string, 0, len(info.Custom))
		for key, value := range info.Custom {
			customStrings = append(customStrings, fmt.Sprintf("%s: %v", key, value))
		}

		_ = table.Append("Custom", strings.Join(customStrings, "\n"))
	}

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	return nil
}

func NewInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   Info,
		Short: "Display API endpoint information",
		Long:  "Display information about the Cloud Foundry API endpoint",
		RunE:  runInfo,
	}
}
