package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// createUsersContextCommand creates the UAA context command.
func createUsersContextCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "context",
		Aliases: []string{"ctx", Status},
		Short:   "Display current UAA context",
		Long:    "Show information about the currently active UAA context including endpoint, authentication status, and user information",
		Example: `  # Show current UAA context
  capi uaa context

  # Show context in JSON format
  capi uaa context --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := loadConfig()
			output := viper.GetString("output")

			// Create UAA client to get context information
			uaaClient, err := NewUAAClient(config)
			if err != nil {
				// Show basic context even if UAA client creation fails
				switch output {
				case OutputFormatJSON:
					return showContextJSON(config, err)
				case OutputFormatYAML:
					return showContextYAML(config, err)
				default:
					return showContextTable(config, err)
				}
			}

			// Test connection and get server info with caching
			ctx := context.Background()

			var (
				serverInfo      map[string]any
				connectionError error
			)

			err = uaaClient.TestConnection(ctx)
			if err != nil {
				connectionError = err
			} else {
				_ = WithPerformanceTracking("get-server-info", func() error {
					var infoErr error

					serverInfo, infoErr = CachedServerInfo(uaaClient)

					return infoErr
				})
			}

			// Display context based on output format
			switch output {
			case OutputFormatJSON:
				return showContextWithServerInfoJSON(uaaClient, serverInfo, connectionError)
			case OutputFormatYAML:
				return showContextWithServerInfoYAML(uaaClient, serverInfo, connectionError)
			default:
				return showContextWithServerInfoTable(uaaClient, serverInfo, connectionError)
			}
		},
	}
}

// createUsersTargetCommand creates the UAA target command.
func createUsersTargetCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "target <url>",
		Aliases: []string{"t", "set-target"},
		Short:   "Set UAA endpoint URL",
		Long:    "Set the URL of the UAA service to target for user management operations",
		Example: `  # Target UAA with full URL
  capi uaa target https://uaa.your-domain.com

  # Target UAA with hostname (https:// added automatically)
  capi uaa target uaa.your-domain.com`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetURL := args[0]

			// Validate URL format
			if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
				targetURL = "https://" + targetURL
			}

			// Update configuration
			viper.Set(uaaEndpointKey, targetURL)

			config := loadConfig()
			config.UAAEndpoint = targetURL

			// Test connection to the new endpoint
			uaaClient, err := NewUAAClientWithEndpoint(targetURL, config)
			if err != nil {
				return fmt.Errorf("failed to create UAA client: %w", err)
			}

			ctx := context.Background()

			err = uaaClient.TestConnection(ctx)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stdout, "Warning: Failed to connect to UAA at %s: %v\n", targetURL, err)
				_, _ = os.Stdout.WriteString("Target set but connection test failed. Please verify the URL and network connectivity.\n")
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "Successfully targeted UAA at %s\n", targetURL)
			}

			// Save configuration
			err = saveConfigStruct(config)
			if err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			return nil
		},
	}
}

// createUsersInfoCommand creates the UAA info command.
func createUsersInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   Info,
		Short: "Display UAA server information",
		Long:  "Show version and configuration information for the targeted UAA server",
		RunE: func(cmd *cobra.Command, args []string) error {
			config := loadConfig()

			if GetEffectiveUAAEndpoint(config) == "" {
				return constants.ErrNoUAAConfigured
			}

			uaaClient, err := NewUAAClient(config)
			if err != nil {
				return fmt.Errorf("failed to create UAA client: %w", err)
			}

			ctx := context.Background()

			serverInfo, err := uaaClient.GetServerInfo(ctx)
			if err != nil {
				return fmt.Errorf("failed to get server info: %w", err)
			}

			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				err := encoder.Encode(serverInfo)
				if err != nil {
					return fmt.Errorf("encoding server info to JSON: %w", err)
				}

				return nil
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				err := encoder.Encode(serverInfo)
				if err != nil {
					return fmt.Errorf("encoding server info to YAML: %w", err)
				}

				return nil
			default:
				return displayServerInfoTable(serverInfo)
			}
		},
	}
}

// createUsersVersionCommand creates the UAA version command.
func createUsersVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   Version,
		Short: "Display UAA server version",
		Long:  "Show the version of the targeted UAA server",
		RunE: func(cmd *cobra.Command, args []string) error {
			config := loadConfig()

			if GetEffectiveUAAEndpoint(config) == "" {
				return constants.ErrNoUAAConfigured
			}

			uaaClient, err := NewUAAClient(config)
			if err != nil {
				return fmt.Errorf("failed to create UAA client: %w", err)
			}

			ctx := context.Background()

			serverInfo, err := uaaClient.GetServerInfo(ctx)
			if err != nil {
				return fmt.Errorf("failed to get server info: %w", err)
			}

			// Extract version information
			version := Unknown

			if app, ok := serverInfo[uaaInfoKeyApp].(map[string]any); ok {
				if v, ok := app["version"].(string); ok && v != "" {
					version = v
				}
			}

			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				result := map[string]string{
					keyVersion:  version,
					keyEndpoint: GetEffectiveUAAEndpoint(config),
				}
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				err := encoder.Encode(result)
				if err != nil {
					return fmt.Errorf("failed to encode result: %w", err)
				}

				return nil
			case OutputFormatYAML:
				result := map[string]string{
					keyVersion:  version,
					keyEndpoint: GetEffectiveUAAEndpoint(config),
				}
				encoder := yaml.NewEncoder(os.Stdout)

				err := encoder.Encode(result)
				if err != nil {
					return fmt.Errorf("failed to encode result: %w", err)
				}

				return nil
			default:
				table := tablewriter.NewWriter(os.Stdout)
				table.Header("Property", "Value")
				_ = table.Append("Version", version)
				_ = table.Append("Endpoint", GetEffectiveUAAEndpoint(config))

				err := table.Render()
				if err != nil {
					return fmt.Errorf("failed to render table: %w", err)
				}

				return nil
			}
		},
	}
}

// Helper functions for context display

func showContextJSON(config *Config, clientError error) error {
	context := map[string]any{
		uaaEndpointKey:   config.UAAEndpoint,
		keyAuthenticated: config.UAAToken != "" || config.Token != "",
		"client_error":   nil,
	}

	if clientError != nil {
		context["client_error"] = clientError.Error()
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(context)
	if err != nil {
		return fmt.Errorf("failed to encode context: %w", err)
	}

	return nil
}

func showContextYAML(config *Config, clientError error) error {
	context := map[string]any{
		uaaEndpointKey:   config.UAAEndpoint,
		keyAuthenticated: config.UAAToken != "" || config.Token != "",
		"client_error":   nil,
	}

	if clientError != nil {
		context["client_error"] = clientError.Error()
	}

	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(context)
	if err != nil {
		return fmt.Errorf("failed to encode context: %w", err)
	}

	return nil
}

func showContextTable(config *Config, clientError error) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	_ = table.Append("Endpoint", getValueOrEmpty(config.UAAEndpoint))

	authenticated := False
	if config.UAAToken != "" || config.Token != "" {
		authenticated = True
	}

	_ = table.Append("Authenticated", authenticated)

	if clientError != nil {
		_ = table.Append("Error", clientError.Error())
	}

	_, _ = os.Stdout.WriteString("UAA Context:\n")
	_, _ = os.Stdout.WriteString("\n")

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	return nil
}

func showContextWithServerInfoJSON(client *UAAClientWrapper, serverInfo map[string]any, connectionError error) error {
	context := map[string]any{
		uaaEndpointKey:      client.Endpoint(),
		keyAuthenticated:    client.IsAuthenticated(),
		"connection_status": connectionError == nil,
		keyServerInfo:       serverInfo,
	}
	if connectionError != nil {
		context["connection_error"] = connectionError.Error()
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(context)
	if err != nil {
		return fmt.Errorf("encoding context to JSON: %w", err)
	}

	return nil
}

func showContextWithServerInfoYAML(client *UAAClientWrapper, serverInfo map[string]any, connectionError error) error {
	context := map[string]any{
		uaaEndpointKey:      client.Endpoint(),
		keyAuthenticated:    client.IsAuthenticated(),
		"connection_status": connectionError == nil,
		keyServerInfo:       serverInfo,
	}
	if connectionError != nil {
		context["connection_error"] = connectionError.Error()
	}

	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(context)
	if err != nil {
		return fmt.Errorf("encoding context to YAML: %w", err)
	}

	return nil
}

func showContextWithServerInfoTable(client *UAAClientWrapper, serverInfo map[string]any, connectionError error) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	// Add basic context information
	_ = table.Append("Endpoint", getValueOrEmpty(client.Endpoint()))

	authenticated := False
	if client.IsAuthenticated() {
		authenticated = True
	}

	_ = table.Append("Authenticated", authenticated)

	connected := True
	if connectionError != nil {
		connected = False
	}

	_ = table.Append("Connected", connected)

	if connectionError != nil {
		_ = table.Append("Connection Error", connectionError.Error())
	}

	// Add server information if available
	addServerInfoToTable(table, serverInfo)

	_, _ = os.Stdout.WriteString("UAA Context:\n")
	_, _ = os.Stdout.WriteString("\n")

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	return nil
}

func displayServerInfoTable(serverInfo map[string]any) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	// Add server info to table
	for key, value := range serverInfo {
		if valueStr, ok := value.(string); ok && valueStr != "" {
			_ = table.Append(key, valueStr)
		} else if valueMap, ok := value.(map[string]any); ok {
			// Handle nested objects like "app"
			for nestedKey, nestedValue := range valueMap {
				if nestedStr, ok := nestedValue.(string); ok && nestedStr != "" {
					_ = table.Append(fmt.Sprintf("%s.%s", key, nestedKey), nestedStr)
				}
			}
		}
	}

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	return nil
}

// addServerInfoToTable adds server information to the table if available.
func addServerInfoToTable(table *tablewriter.Table, serverInfo map[string]any) {
	if len(serverInfo) == 0 {
		return
	}

	addAppInfoToTable(table, serverInfo)
	addServerPropertyIfExists(table, serverInfo, "commit_id", "Commit ID")
	addServerPropertyIfExists(table, serverInfo, "zone_name", "Zone Name")
}

// addAppInfoToTable adds application information from server info.
func addAppInfoToTable(table *tablewriter.Table, serverInfo map[string]any) {
	app, ok := serverInfo[uaaInfoKeyApp].(map[string]any)
	if !ok {
		return
	}

	if name, ok := app["name"].(string); ok && name != "" {
		_ = table.Append("Server Name", name)
	}

	if version, ok := app["version"].(string); ok && version != "" {
		_ = table.Append("Server Version", version)
	}
}

// addServerPropertyIfExists adds a server property to the table if it exists and is not empty.
func addServerPropertyIfExists(table *tablewriter.Table, serverInfo map[string]any, key, displayName string) {
	if value, ok := serverInfo[key].(string); ok && value != "" {
		_ = table.Append(displayName, value)
	}
}

func getValueOrEmpty(value string) string {
	if value == "" {
		return "(not set)"
	}

	return value
}
