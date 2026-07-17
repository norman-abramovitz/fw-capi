package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewAPIsCommand creates the apis command group.
func NewAPIsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "apis",
		Aliases: []string{"api"},
		Short:   "Manage Cloud Foundry API endpoints",
		Long:    "Add, list, delete, and target Cloud Foundry API endpoints",
	}

	cmd.AddCommand(newAPIsAddCommand())
	cmd.AddCommand(newAPIsListCommand())
	cmd.AddCommand(newAPIsDeleteCommand())
	cmd.AddCommand(newAPIsTargetCommand())

	return cmd
}

func newAPIsAddCommand() *cobra.Command {
	var skipSSLValidation bool

	cmd := &cobra.Command{
		Use:   "add NAME ENDPOINT",
		Short: "Add a new Cloud Foundry API endpoint",
		Long:  "Add a new Cloud Foundry API endpoint to the configuration",
		Args:  cobra.ExactArgs(constants.MinimumArgumentCount),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			endpoint := args[1]

			// Validate and normalize the endpoint
			normalizedEndpoint, err := normalizeEndpoint(endpoint)
			if err != nil {
				return fmt.Errorf("invalid endpoint: %w", err)
			}

			// Load current configuration
			config := loadConfig()

			// Initialize APIs map if it doesn't exist
			if config.APIs == nil {
				config.APIs = make(map[string]*APIConfig)
			}

			// Extract domain for use as key
			domain := extractDomainFromEndpoint(normalizedEndpoint)

			// Check if API already exists
			if _, exists := config.APIs[domain]; exists {
				return fmt.Errorf("%w for domain '%s': '%s'", capi.ErrAPIAlreadyExists, domain, name)
			}

			// Create new API configuration
			apiConfig := &APIConfig{
				Endpoint:          normalizedEndpoint,
				SkipSSLValidation: skipSSLValidation,
			}

			// Add to configuration
			config.APIs[domain] = apiConfig

			// If this is the first API, make it current
			if config.CurrentAPI == "" {
				config.CurrentAPI = domain
				_, _ = fmt.Fprintf(os.Stdout, "API '%s' (%s) added and set as current target\n", name, normalizedEndpoint)
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "API '%s' (%s) added\n", name, normalizedEndpoint)
			}

			// Save configuration
			err = saveConfigStruct(config)
			if err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&skipSSLValidation, "skip-ssl-validation", false, "Skip SSL certificate validation")

	return cmd
}

func newAPIsListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   List,
		Short: "List all Cloud Foundry API endpoints",
		Long:  "Display all configured Cloud Foundry API endpoints",
		RunE:  runAPIsList,
	}
}

func runAPIsList(cmd *cobra.Command, args []string) error {
	config := loadConfig()

	if len(config.APIs) == 0 {
		_, _ = os.Stdout.WriteString("No APIs configured. Use 'capi apis add' to add one.\n")

		return nil
	}

	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return outputAPIsJSON(config)
	case OutputFormatYAML:
		return outputAPIsYAML(config)
	default:
		return outputAPIsTable(config)
	}
}

type apiInfo struct {
	Domain            string `json:"domain"`
	Endpoint          string `json:"endpoint"`
	Username          string `json:"username,omitempty"`
	Organization      string `json:"organization,omitempty"`
	Space             string `json:"space,omitempty"`
	SkipSSLValidation bool   `json:"skip_ssl_validation"`
	Current           bool   `json:"current"`
}

func outputAPIsJSON(config *Config) error {
	apis := make([]apiInfo, 0, len(config.APIs))
	for domain, apiConfig := range config.APIs {
		apis = append(apis, apiInfo{
			Domain:            domain,
			Endpoint:          apiConfig.Endpoint,
			Username:          apiConfig.Username,
			Organization:      apiConfig.Organization,
			Space:             apiConfig.Space,
			SkipSSLValidation: apiConfig.SkipSSLValidation,
			Current:           domain == config.CurrentAPI,
		})
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(apis)
	if err != nil {
		return fmt.Errorf("failed to encode APIs as JSON: %w", err)
	}

	return nil
}

func outputAPIsYAML(config *Config) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(config.APIs)
	if err != nil {
		return fmt.Errorf("failed to encode APIs as YAML: %w", err)
	}

	return nil
}

func outputAPIsTable(config *Config) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Domain", "Endpoint", "User", "Org", "Space", "Current")

	for domain, apiConfig := range config.APIs {
		current := ""
		if domain == config.CurrentAPI {
			current = constants.CheckMarkSymbol
		}

		_ = table.Append(
			domain,
			apiConfig.Endpoint,
			defaultIfEmpty(apiConfig.Username, "-"),
			defaultIfEmpty(apiConfig.Organization, "-"),
			defaultIfEmpty(apiConfig.Space, "-"),
			current,
		)
	}

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render APIs table: %w", err)
	}

	return nil
}

func defaultIfEmpty(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}

	return value
}

func newAPIsDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete DOMAIN",
		Short: "Delete a Cloud Foundry API endpoint",
		Long:  "Remove a Cloud Foundry API endpoint from the configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			// Load current configuration
			config := loadConfig()

			// Check if API exists
			if _, exists := config.APIs[domain]; !exists {
				return fmt.Errorf("%w: '%s'", capi.ErrAPINotFound, domain)
			}

			// Don't allow deleting the last API if it's current
			if len(config.APIs) == 1 && config.CurrentAPI == domain {
				return capi.ErrCannotDeleteOnlyAPI
			}

			// Remove from configuration
			delete(config.APIs, domain)

			// If this was the current API, switch to another one
			if config.CurrentAPI == domain {
				if len(config.APIs) > 0 {
					// Set the first remaining API as current
					for newDomain := range config.APIs {
						config.CurrentAPI = newDomain

						break
					}

					_, _ = fmt.Fprintf(os.Stdout, "API '%s' deleted. Current API switched to '%s'\n", domain, config.CurrentAPI)
				} else {
					config.CurrentAPI = ""
					_, _ = fmt.Fprintf(os.Stdout, "API '%s' deleted. No APIs remaining.\n", domain)
				}
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "API '%s' deleted\n", domain)
			}

			// Save configuration
			err := saveConfigStruct(config)
			if err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			return nil
		},
	}
}

func newAPIsTargetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "target DOMAIN",
		Short: "Target a Cloud Foundry API endpoint",
		Long:  "Set a Cloud Foundry API endpoint as the current target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			// Load current configuration
			config := loadConfig()

			// Check if API exists
			if _, exists := config.APIs[domain]; !exists {
				return fmt.Errorf("%w: '%s'. Use 'capi apis list' to see available APIs", capi.ErrAPINotFound, domain)
			}

			// Set as current
			config.CurrentAPI = domain

			// Save configuration
			err := saveConfigStruct(config)
			if err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "API '%s' is now the current target\n", domain)

			return nil
		},
	}
}

// normalizeEndpoint validates and normalizes a CF API endpoint URL.
func normalizeEndpoint(endpoint string) (string, error) {
	// Add https:// if no protocol is specified
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	// Parse URL to validate
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	// Ensure we have a host
	if parsedURL.Host == "" {
		return "", capi.ErrNoHostInURL
	}

	// Remove trailing slash and path for consistency
	normalizedURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	return normalizedURL, nil
}
