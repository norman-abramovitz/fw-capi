package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewAdminCommand creates the admin command group.
func NewAdminCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Administrative operations",
		Long:  "Administrative operations for Cloud Foundry platform management",
	}

	cmd.AddCommand(newAdminClearCacheCommand())
	cmd.AddCommand(newAdminUsageSummaryCommand())
	cmd.AddCommand(newAdminInfoCommand())

	return cmd
}

func newAdminClearCacheCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear-cache",
		Short: "Clear platform buildpack cache",
		Long:  "Clear the buildpack cache for the entire Cloud Foundry platform",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Clear buildpack cache
			job, err := client.ClearBuildpackCache(ctx)
			if err != nil {
				return fmt.Errorf("clearing buildpack cache: %w", err)
			}

			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				return encoder.Encode(job)
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				return encoder.Encode(job)
			default:
				_, _ = os.Stdout.WriteString("✓ Buildpack cache clear initiated\n")
				if job != nil {
					_, _ = fmt.Fprintf(os.Stdout, "Job GUID: %s\n", job.GUID)
					_, _ = fmt.Fprintf(os.Stdout, "Monitor job status with: capi jobs get %s\n", job.GUID)
				}
			}

			return nil
		},
	}

	return cmd
}

func newAdminUsageSummaryCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "usage-summary",
		Short: "Get platform usage summary",
		Long:  "Get usage summary for the entire Cloud Foundry platform",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Get usage summary
			summary, err := client.GetUsageSummary(ctx)
			if err != nil {
				return fmt.Errorf("getting usage summary: %w", err)
			}

			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				return encoder.Encode(summary)
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				return encoder.Encode(summary)
			default:
				table := tablewriter.NewWriter(os.Stdout)
				table.Header("Metric", "Value")

				_ = table.Append("Memory (MB)", strconv.Itoa(summary.UsageSummary.MemoryInMB))
				_ = table.Append("Started Instances", strconv.Itoa(summary.UsageSummary.StartedInstances))

				_, _ = os.Stdout.WriteString("Platform Usage Summary:\n")
				_, _ = os.Stdout.WriteString("\n")

				err := table.Render()
				if err != nil {
					return fmt.Errorf("failed to render table: %w", err)
				}
			}

			return nil
		},
	}
}

func newAdminInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   Info,
		Short: "Get extended platform information",
		Long:  "Get extended information about the Cloud Foundry platform",
		RunE:  runAdminInfo,
	}
}

func runAdminInfo(cmd *cobra.Command, args []string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	extendedInfo, err := getExtendedPlatformInfo(ctx, client)
	if err != nil {
		return err
	}

	return outputAdminInfo(extendedInfo)
}

func getExtendedPlatformInfo(ctx context.Context, client capi.Client) (*extendedInfo, error) {
	// Get platform info
	info, err := client.GetInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting platform info: %w", err)
	}

	// Get usage summary for additional context
	summary, err := client.GetUsageSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting usage summary: %w", err)
	}

	return &extendedInfo{
		Name:        info.Name,
		Build:       info.Build,
		Version:     info.Version,
		Description: info.Description,
		Links:       info.Links,
		Usage:       summary.UsageSummary,
	}, nil
}

type extendedInfo struct {
	Name        string                `json:"name"        yaml:"name"`
	Build       string                `json:"build"       yaml:"build"`
	Version     int                   `json:"version"     yaml:"version"`
	Description string                `json:"description" yaml:"description"`
	Links       capi.Links            `json:"links"       yaml:"links"`
	Usage       capi.UsageSummaryData `json:"usage"       yaml:"usage"`
}

func outputAdminInfo(info *extendedInfo) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return outputAdminInfoJSON(info)
	case OutputFormatYAML:
		return outputAdminInfoYAML(info)
	default:
		return outputAdminInfoTable(info)
	}
}

func outputAdminInfoJSON(info *extendedInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(info)
	if err != nil {
		return fmt.Errorf("failed to encode admin info as JSON: %w", err)
	}

	return nil
}

func outputAdminInfoYAML(info *extendedInfo) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(info)
	if err != nil {
		return fmt.Errorf("failed to encode admin info as YAML: %w", err)
	}

	return nil
}

func outputAdminInfoTable(info *extendedInfo) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	_ = table.Append("Name", info.Name)
	_ = table.Append("Build", info.Build)
	_ = table.Append("API Version", strconv.Itoa(info.Version))
	_ = table.Append("Description", info.Description)
	_ = table.Append("Memory Usage (MB)", strconv.Itoa(info.Usage.MemoryInMB))
	_ = table.Append("Started Instances", strconv.Itoa(info.Usage.StartedInstances))

	addLinkToTable(table, info.Links, "authorization_endpoint", "Auth Endpoint")
	addLinkToTable(table, info.Links, "token_endpoint", "Token Endpoint")
	addLinkToTable(table, info.Links, "logging_endpoint", "Logging Endpoint")

	_, _ = os.Stdout.WriteString("Extended Platform Information:\n")
	_, _ = os.Stdout.WriteString("\n")

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render admin info table: %w", err)
	}

	return nil
}

func addLinkToTable(table *tablewriter.Table, links capi.Links, key, label string) {
	if href, exists := links[key]; exists && href.Href != "" {
		_ = table.Append(label, href.Href)
	}
}
