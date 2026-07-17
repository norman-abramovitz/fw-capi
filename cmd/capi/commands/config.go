package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fivetwenty-io/capi/v3/internal/auth"
	"github.com/fivetwenty-io/capi/v3/internal/client"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/fivetwenty-io/capi/v3/pkg/cfclient"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	organizationKey = "organization"
	// spaceKey is the canonical literal for the space resource: used as the
	// viper config key, CLI flag name/alias, and role ResourceType/EntityType
	// discriminator, mirroring organizationKey's usage.
	spaceKey = "space"

	// UAA-related viper/persisted-config key names.
	uaaEndpointKey     = "uaa_endpoint"
	uaaTokenKey        = "uaa_token"         //#nosec G101 -- nolint:gosec -- config key name, not a credential
	uaaRefreshTokenKey = "uaa_refresh_token" //#nosec G101 -- nolint:gosec -- config key name, not a credential
	uaaClientIDKey     = "uaa_client_id"
	uaaClientSecretKey = "uaa_client_secret" //#nosec G101 -- nolint:gosec -- config key name, not a credential

	// tokenKey is the viper/persisted-config key for the legacy CF token.
	tokenKey = "token"
)

// Config represents the CLI configuration.
type Config struct {
	// Multi-API configuration
	APIs       map[string]*APIConfig `json:"apis,omitempty"        yaml:"apis,omitempty"`
	CurrentAPI string                `json:"current_api,omitempty" yaml:"current_api,omitempty"`

	// Global settings
	Output  string `json:"output"   yaml:"output"`
	NoColor bool   `json:"no_color" yaml:"no_color"`

	// Legacy fields for backward compatibility (will be migrated to APIs map)
	API               string            `json:"api,omitempty"               yaml:"api,omitempty"`
	Token             string            `json:"token,omitempty"             yaml:"token,omitempty"`
	RefreshToken      string            `json:"refresh_token,omitempty"     yaml:"refresh_token,omitempty"`
	Username          string            `json:"username,omitempty"          yaml:"username,omitempty"`
	Organization      string            `json:"organization,omitempty"      yaml:"organization,omitempty"`
	OrganizationGUID  string            `json:"organization_guid,omitempty" yaml:"organization_guid,omitempty"`
	Space             string            `json:"space,omitempty"             yaml:"space,omitempty"`
	SpaceGUID         string            `json:"space_guid,omitempty"        yaml:"space_guid,omitempty"`
	SkipSSLValidation bool              `json:"skip_ssl_validation"         yaml:"skip_ssl_validation"`
	Targets           map[string]Target `json:"targets,omitempty"           yaml:"targets,omitempty"`
	CurrentTarget     string            `json:"current_target,omitempty"    yaml:"current_target,omitempty"`
	UAAEndpoint       string            `json:"uaa_endpoint,omitempty"      yaml:"uaa_endpoint,omitempty"`
	UAAToken          string            `json:"uaa_token,omitempty"         yaml:"uaa_token,omitempty"`
	UAARefreshToken   string            `json:"uaa_refresh_token,omitempty" yaml:"uaa_refresh_token,omitempty"`
	UAAClientID       string            `json:"uaa_client_id,omitempty"     yaml:"uaa_client_id,omitempty"`
	UAAClientSecret   string            `json:"uaa_client_secret,omitempty" yaml:"uaa_client_secret,omitempty"`
}

// APIConfig represents configuration for a single Cloud Foundry API endpoint.
type APIConfig struct {
	Endpoint          string            `json:"endpoint"                    yaml:"endpoint"`
	Token             string            `json:"token,omitempty"             yaml:"token,omitempty"`
	TokenExpiresAt    *time.Time        `json:"token_expires_at,omitempty"  yaml:"token_expires_at,omitempty"`
	RefreshToken      string            `json:"refresh_token,omitempty"     yaml:"refresh_token,omitempty"`
	LastRefreshed     *time.Time        `json:"last_refreshed,omitempty"    yaml:"last_refreshed,omitempty"`
	Username          string            `json:"username,omitempty"          yaml:"username,omitempty"`
	Organization      string            `json:"organization,omitempty"      yaml:"organization,omitempty"`
	OrganizationGUID  string            `json:"organization_guid,omitempty" yaml:"organization_guid,omitempty"`
	Space             string            `json:"space,omitempty"             yaml:"space,omitempty"`
	SpaceGUID         string            `json:"space_guid,omitempty"        yaml:"space_guid,omitempty"`
	SkipSSLValidation bool              `json:"skip_ssl_validation"         yaml:"skip_ssl_validation"`
	UAAEndpoint       string            `json:"uaa_endpoint,omitempty"      yaml:"uaa_endpoint,omitempty"`
	UAAToken          string            `json:"uaa_token,omitempty"         yaml:"uaa_token,omitempty"`
	UAARefreshToken   string            `json:"uaa_refresh_token,omitempty" yaml:"uaa_refresh_token,omitempty"`
	UAAClientID       string            `json:"uaa_client_id,omitempty"     yaml:"uaa_client_id,omitempty"`
	UAAClientSecret   string            `json:"uaa_client_secret,omitempty" yaml:"uaa_client_secret,omitempty"`
	APILinks          map[string]string `json:"api_links,omitempty"         yaml:"api_links,omitempty"`
}

// Target represents a saved CF target.
type Target struct {
	API               string `json:"api"                     yaml:"api"`
	Token             string `json:"token,omitempty"         yaml:"token,omitempty"`
	RefreshToken      string `json:"refresh_token,omitempty" yaml:"refresh_token,omitempty"`
	Username          string `json:"username,omitempty"      yaml:"username,omitempty"`
	Organization      string `json:"organization,omitempty"  yaml:"organization,omitempty"`
	Space             string `json:"space,omitempty"         yaml:"space,omitempty"`
	SkipSSLValidation bool   `json:"skip_ssl_validation"     yaml:"skip_ssl_validation"`
	// UAA-specific fields for targets
	UAAEndpoint     string `json:"uaa_endpoint,omitempty"      yaml:"uaa_endpoint,omitempty"`
	UAAToken        string `json:"uaa_token,omitempty"         yaml:"uaa_token,omitempty"`
	UAARefreshToken string `json:"uaa_refresh_token,omitempty" yaml:"uaa_refresh_token,omitempty"`
	UAAClientID     string `json:"uaa_client_id,omitempty"     yaml:"uaa_client_id,omitempty"`
	UAAClientSecret string `json:"uaa_client_secret,omitempty" yaml:"uaa_client_secret,omitempty"`
}

// NewConfigCommand creates the config command group.
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long:  "Manage CAPI CLI configuration including targets and settings",
	}

	cmd.AddCommand(newConfigShowCommand())
	cmd.AddCommand(newConfigSetCommand())
	cmd.AddCommand(newConfigUnsetCommand())
	cmd.AddCommand(newConfigClearCommand())

	return cmd
}

func newConfigShowCommand() *cobra.Command {
	var apiFlag string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long:  "Display the current CLI configuration (global or API-specific)",
		RunE: func(cmd *cobra.Command, args []string) error {
			config := loadConfig()

			// If --api flag is provided, show only that API's configuration
			if apiFlag != "" {
				return showAPISpecificConfig(config, apiFlag)
			}

			output := viper.GetString("output")
			switch output {
			case constants.FormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				return encoder.Encode(config) //#nosec G117 -- nolint:gosec -- intentional CLI config persistence; refresh_token required for session restore
			case constants.FormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				return encoder.Encode(config) //#nosec G117 -- nolint:gosec -- intentional CLI config persistence; refresh_token required for session restore
			default:
				return displayConfigTable(config)
			}
		},
	}

	cmd.Flags().StringVar(&apiFlag, "api", "", "show configuration for specific API")

	return cmd
}

func newConfigSetCommand() *cobra.Command {
	var apiFlag string

	cmd := &cobra.Command{
		Use:   "set KEY VALUE",
		Short: "Set a configuration value",
		Long:  "Set a specific configuration value (global or API-specific)",
		Args:  cobra.ExactArgs(constants.MinimumArgumentCount),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			config := loadConfig()

			// If --api flag is provided, set API-specific configuration
			if apiFlag != "" {
				return setAPISpecificConfig(config, apiFlag, key, value)
			}

			// Otherwise set global configuration
			return setGlobalConfig(config, key, value)
		},
	}

	cmd.Flags().StringVar(&apiFlag, "api", "", "target specific API for configuration")

	return cmd
}

func newConfigUnsetCommand() *cobra.Command {
	var apiFlag string

	cmd := &cobra.Command{
		Use:   "unset KEY",
		Short: "Unset a configuration value",
		Long:  "Remove a specific configuration value (global or API-specific)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			config := loadConfig()

			// If --api flag is provided, unset API-specific configuration
			if apiFlag != "" {
				return unsetAPISpecificConfig(config, apiFlag, key)
			}

			// Otherwise unset global configuration
			return unsetGlobalConfig(config, key)
		},
	}

	cmd.Flags().StringVar(&apiFlag, "api", "", "target specific API for configuration")

	return cmd
}

func newConfigClearCommand() *cobra.Command {
	var apiFlag string

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear configuration",
		Long:  "Remove all configuration settings (global or API-specific)",
		RunE: func(cmd *cobra.Command, args []string) error {
			config := loadConfig()

			// If --api flag is provided, clear only that API's configuration
			if apiFlag != "" {
				return clearAPISpecificConfig(config, apiFlag)
			}

			// Otherwise clear all configuration
			configFile := viper.ConfigFileUsed()
			if configFile == "" {
				home, _ := os.UserHomeDir()
				configFile = filepath.Join(home, ".capi", "config.yml")
			}

			err := os.Remove(configFile)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove config file: %w", err)
			}

			return outputConfigUpdateResult("Cleared", "all configuration", "", "")
		},
	}

	cmd.Flags().StringVar(&apiFlag, "api", "", "clear configuration for specific API only")

	return cmd
}

func loadConfig() *Config {
	config := createBaseConfig()

	loadAPIConfigurations(config)
	handleLegacyMigration(config)
	loadLegacyTargets(config)

	return config
}

// createBaseConfig creates a base config with global settings and legacy fields.
func createBaseConfig() *Config {
	return &Config{
		// Global settings
		Output:  viper.GetString("output"),
		NoColor: viper.GetBool("no_color"),

		// Initialize APIs map
		APIs: make(map[string]*APIConfig),

		// Legacy fields for migration
		API:               viper.GetString("api"),
		Token:             viper.GetString(tokenKey),
		RefreshToken:      viper.GetString("refresh_token"),
		Username:          viper.GetString("username"),
		Organization:      viper.GetString(organizationKey),
		OrganizationGUID:  viper.GetString("organization_guid"),
		Space:             viper.GetString(spaceKey),
		SpaceGUID:         viper.GetString("space_guid"),
		SkipSSLValidation: viper.GetBool("skip_ssl_validation"),
		CurrentTarget:     viper.GetString("current_target"),
		Targets:           make(map[string]Target),
		UAAEndpoint:       viper.GetString(uaaEndpointKey),
		UAAToken:          viper.GetString(uaaTokenKey),
		UAARefreshToken:   viper.GetString(uaaRefreshTokenKey),
		UAAClientID:       viper.GetString(uaaClientIDKey),
		UAAClientSecret:   viper.GetString(uaaClientSecretKey),
	}
}

// loadAPIConfigurations loads multi-API configuration from viper.
func loadAPIConfigurations(config *Config) {
	config.CurrentAPI = viper.GetString("current_api")

	apisRaw := viper.GetStringMap("apis")
	if apisRaw == nil {
		return
	}

	for domain, apiRaw := range apisRaw {
		if apiMap, ok := apiRaw.(map[string]any); ok {
			apiConfig := parseAPIConfig(apiMap)
			config.APIs[domain] = apiConfig
		}
	}
}

// parseAPIConfig parses API configuration from a map.
func parseAPIConfig(apiMap map[string]any) *APIConfig {
	apiConfig := &APIConfig{}

	parseAPIBasicFields(apiConfig, apiMap)
	parseAPIAuthFields(apiConfig, apiMap)
	parseAPIOrganizationSpaceFields(apiConfig, apiMap)
	parseAPIUAAFields(apiConfig, apiMap)
	parseAPITimestampFields(apiConfig, apiMap)

	return apiConfig
}

// parseAPIBasicFields parses basic API configuration fields.
func parseAPIBasicFields(apiConfig *APIConfig, apiMap map[string]any) {
	if endpoint, ok := apiMap["endpoint"].(string); ok {
		apiConfig.Endpoint = endpoint
	}

	if skipSSL, ok := apiMap["skip_ssl_validation"].(bool); ok {
		apiConfig.SkipSSLValidation = skipSSL
	}
}

// parseAPIAuthFields parses authentication-related API configuration fields.
func parseAPIAuthFields(apiConfig *APIConfig, apiMap map[string]any) {
	if token, ok := apiMap[tokenKey].(string); ok {
		apiConfig.Token = token
	}

	if refreshToken, ok := apiMap["refresh_token"].(string); ok {
		apiConfig.RefreshToken = refreshToken
	}

	if username, ok := apiMap["username"].(string); ok {
		apiConfig.Username = username
	}
}

// parseAPIUAAFields parses UAA-related API configuration fields.
func parseAPIUAAFields(apiConfig *APIConfig, apiMap map[string]any) {
	uaaFields := map[string]*string{
		uaaEndpointKey:     &apiConfig.UAAEndpoint,
		uaaTokenKey:        &apiConfig.UAAToken,
		uaaRefreshTokenKey: &apiConfig.UAARefreshToken,
		uaaClientIDKey:     &apiConfig.UAAClientID,
		uaaClientSecretKey: &apiConfig.UAAClientSecret,
	}

	for key, field := range uaaFields {
		if value, ok := apiMap[key].(string); ok {
			*field = value
		}
	}
}

// parseAPIOrganizationSpaceFields parses organization and space fields.
func parseAPIOrganizationSpaceFields(apiConfig *APIConfig, apiMap map[string]any) {
	if org, ok := apiMap[organizationKey].(string); ok {
		apiConfig.Organization = org
	}

	if orgGUID, ok := apiMap["organization_guid"].(string); ok {
		apiConfig.OrganizationGUID = orgGUID
	}

	if space, ok := apiMap[spaceKey].(string); ok {
		apiConfig.Space = space
	}

	if spaceGUID, ok := apiMap["space_guid"].(string); ok {
		apiConfig.SpaceGUID = spaceGUID
	}
}

// parseAPITimestampFields parses timestamp fields in API configuration.
func parseAPITimestampFields(apiConfig *APIConfig, apiMap map[string]any) {
	if tokenExpiresAtStr, ok := apiMap["token_expires_at"].(string); ok && tokenExpiresAtStr != "" {
		t, err := time.Parse(time.RFC3339, tokenExpiresAtStr)
		if err == nil {
			apiConfig.TokenExpiresAt = &t
		}
	}

	if lastRefreshedStr, ok := apiMap["last_refreshed"].(string); ok && lastRefreshedStr != "" {
		t, err := time.Parse(time.RFC3339, lastRefreshedStr)
		if err == nil {
			apiConfig.LastRefreshed = &t
		}
	}
}

// handleLegacyMigration handles migration from legacy single-API config.
func handleLegacyMigration(config *Config) {
	if config.API != "" && len(config.APIs) == 0 {
		migrateFromLegacyConfig(config)
	}
}

// loadLegacyTargets loads legacy target configurations from viper.
func loadLegacyTargets(config *Config) {
	targetsRaw := viper.GetStringMap("targets")
	if targetsRaw == nil {
		return
	}

	for name, targetRaw := range targetsRaw {
		if targetMap, ok := targetRaw.(map[string]any); ok {
			target := parseTarget(targetMap)
			config.Targets[name] = target
		}
	}
}

// parseTarget parses a target configuration from a map.
func parseTarget(targetMap map[string]any) Target {
	target := Target{}

	parseTargetBasicFields(&target, targetMap)
	parseTargetAuthFields(&target, targetMap)
	parseTargetUAAFields(&target, targetMap)

	return target
}

// parseTargetBasicFields parses basic target fields.
func parseTargetBasicFields(target *Target, targetMap map[string]any) {
	if api, ok := targetMap["api"].(string); ok {
		target.API = api
	}

	if org, ok := targetMap[organizationKey].(string); ok {
		target.Organization = org
	}

	if space, ok := targetMap[spaceKey].(string); ok {
		target.Space = space
	}

	if skipSSL, ok := targetMap["skip_ssl_validation"].(bool); ok {
		target.SkipSSLValidation = skipSSL
	}
}

// parseTargetAuthFields parses authentication fields for targets.
func parseTargetAuthFields(target *Target, targetMap map[string]any) {
	if token, ok := targetMap[tokenKey].(string); ok {
		target.Token = token
	}

	if refreshToken, ok := targetMap["refresh_token"].(string); ok {
		target.RefreshToken = refreshToken
	}

	if username, ok := targetMap["username"].(string); ok {
		target.Username = username
	}
}

// parseTargetUAAFields parses UAA-related fields for targets.
func parseTargetUAAFields(target *Target, targetMap map[string]any) {
	uaaFields := map[string]*string{
		uaaEndpointKey:     &target.UAAEndpoint,
		uaaTokenKey:        &target.UAAToken,
		uaaRefreshTokenKey: &target.UAARefreshToken,
		uaaClientIDKey:     &target.UAAClientID,
		uaaClientSecretKey: &target.UAAClientSecret,
	}

	for key, field := range uaaFields {
		if value, ok := targetMap[key].(string); ok {
			*field = value
		}
	}
}

func saveConfig() error {
	// Load current config to check for migration needs
	config := loadConfig()

	// Use the struct-based save which handles migration and backup
	return saveConfigStruct(config)
}

func saveConfigStruct(config *Config) error {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}

		configDir := filepath.Join(home, ".capi")

		err = os.MkdirAll(configDir, constants.ConfigDirPerm)
		if err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		configFile = filepath.Join(configDir, "config.yml")
	}

	// Create backup before migration if APIs are being used and legacy fields exist
	if len(config.APIs) > 0 && config.API != "" {
		backupFile := configFile + ".backup"

		_, err := os.Stat(backupFile)
		if os.IsNotExist(err) {
			// Read current config and save as backup
			// configFile is securely constructed from user home dir and is not user-controllable
			// #nosec G304
			currentData, err := os.ReadFile(configFile)
			if err == nil {
				// backupFile is derived from configFile, itself built from the
				// user home dir and not externally controllable — same trust
				// basis as the #nosec G304 read above.
				_ = os.WriteFile(backupFile, currentData, constants.ConfigFilePerm) // #nosec G703
			}
		}
	}

	// Clear legacy fields if migration occurred
	if len(config.APIs) > 0 {
		config.API = ""
		config.Token = ""
		config.RefreshToken = ""
		config.Username = ""
		config.Organization = ""
		config.OrganizationGUID = ""
		config.Space = ""
		config.SpaceGUID = ""
		config.SkipSSLValidation = false
		config.UAAEndpoint = ""
		config.UAAToken = ""
		config.UAARefreshToken = ""
		config.UAAClientID = ""
		config.UAAClientSecret = ""
		// Keep legacy targets for now but they're deprecated
	}

	data, err := yaml.Marshal(config) //#nosec G117 -- nolint:gosec -- intentional CLI config persistence; refresh_token required for session restore
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	err = os.WriteFile(configFile, data, constants.ConfigFilePerm)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// migrateFromLegacyConfig converts legacy single-API config to new multi-API format.
func migrateFromLegacyConfig(config *Config) *Config {
	if config.API == "" {
		return config
	}

	// Extract domain from API endpoint for use as key
	domain := extractDomainFromEndpoint(config.API)

	// Create new API config from legacy fields
	apiConfig := &APIConfig{
		Endpoint:          config.API,
		Token:             config.Token,
		RefreshToken:      config.RefreshToken,
		Username:          config.Username,
		Organization:      config.Organization,
		OrganizationGUID:  config.OrganizationGUID,
		Space:             config.Space,
		SpaceGUID:         config.SpaceGUID,
		SkipSSLValidation: config.SkipSSLValidation,
		UAAEndpoint:       config.UAAEndpoint,
		UAAToken:          config.UAAToken,
		UAARefreshToken:   config.UAARefreshToken,
		UAAClientID:       config.UAAClientID,
		UAAClientSecret:   config.UAAClientSecret,
	}

	// Add to APIs map and set as current
	config.APIs[domain] = apiConfig
	config.CurrentAPI = domain

	// Clear legacy fields after migration
	config.API = ""
	config.Token = ""
	config.RefreshToken = ""
	config.Username = ""
	config.Organization = ""
	config.OrganizationGUID = ""
	config.Space = ""
	config.SpaceGUID = ""
	config.SkipSSLValidation = false
	config.UAAEndpoint = ""
	config.UAAToken = ""
	config.UAARefreshToken = ""
	config.UAAClientID = ""
	config.UAAClientSecret = ""

	return config
}

// extractDomainFromEndpoint extracts the domain portion from a CF API endpoint.
func extractDomainFromEndpoint(endpoint string) string {
	// Remove protocol if present
	domain := endpoint
	if strings.HasPrefix(domain, "https://") {
		domain = strings.TrimPrefix(domain, "https://")
	} else if strings.HasPrefix(domain, "http://") {
		domain = strings.TrimPrefix(domain, "http://")
	}

	// Remove path if present
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}

	// Remove port if present
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}

	return domain
}

// getCurrentAPIConfig returns the configuration for the currently targeted API.
func getCurrentAPIConfig() (*APIConfig, error) {
	config := loadConfig()

	if config.CurrentAPI == "" {
		if len(config.APIs) == 0 {
			return nil, fmt.Errorf("%w, use 'capi apis add' to add one", capi.ErrNoAPIsConfigured)
		}
		// If no current API set but APIs exist, use the first one
		for domain := range config.APIs {
			config.CurrentAPI = domain

			break
		}
	}

	apiConfig, exists := config.APIs[config.CurrentAPI]
	if !exists {
		return nil, fmt.Errorf("%w in configuration: '%s'", capi.ErrCurrentAPINotFound, config.CurrentAPI)
	}

	return apiConfig, nil
}

// getAPIConfigByFlag returns API config based on command line flag or current API.
func getAPIConfigByFlag(apiFlag string) (*APIConfig, error) {
	config := loadConfig()

	// If --api flag is provided, use that specific API
	if apiFlag != "" {
		// First try to resolve it as a short name or endpoint
		resolvedEndpoint, err := ResolveAPIEndpoint(apiFlag)
		if err != nil {
			return nil, err
		}

		// Check if the original apiFlag is a short name in our config
		if apiConfig, exists := config.APIs[apiFlag]; exists {
			return apiConfig, nil
		}

		// Otherwise look for it by resolved endpoint
		for _, apiConfig := range config.APIs {
			if apiConfig.Endpoint == resolvedEndpoint {
				return apiConfig, nil
			}
		}

		return nil, fmt.Errorf("%w in configuration, use 'capi apis list' to see available APIs: '%s'", capi.ErrAPINotFound, apiFlag)
	}

	// Otherwise use current API
	return getCurrentAPIConfig()
}

// ResolveAPIEndpoint resolves a short name or returns the endpoint if it's already a URL.
func ResolveAPIEndpoint(apiNameOrEndpoint string) (string, error) {
	if apiNameOrEndpoint == "" {
		return "", capi.ErrAPINameOrEndpointRequired
	}

	config := loadConfig()

	// Check if it's a short name in the APIs map
	if apiConfig, exists := config.APIs[apiNameOrEndpoint]; exists {
		return apiConfig.Endpoint, nil
	}

	// If not found in config, treat as direct endpoint URL
	return apiNameOrEndpoint, nil
}

// GetEffectiveUAAEndpoint returns the effective UAA endpoint from either legacy config or current API.
func GetEffectiveUAAEndpoint(config *Config) string {
	// Check legacy UAA endpoint first
	if config.UAAEndpoint != "" {
		return config.UAAEndpoint
	}

	// Check current API configuration for UAA endpoint
	if config.CurrentAPI != "" {
		if apiConfig, exists := config.APIs[config.CurrentAPI]; exists && apiConfig.UAAEndpoint != "" {
			return apiConfig.UAAEndpoint
		}
	}

	return ""
}

// discoverUAAEndpoint discovers the UAA endpoint from a CF API endpoint.
func discoverUAAEndpoint(apiEndpoint string) string {
	// This is a simplified implementation - in a real scenario you'd make an HTTP request
	// to the API root and parse the UAA link from the response
	// For now, use a common pattern
	if strings.Contains(apiEndpoint, "api.") {
		return strings.Replace(apiEndpoint, "api.", "uaa.", 1)
	}

	// Fallback: assume UAA is at the same domain with /uaa path
	return strings.TrimSuffix(apiEndpoint, "/") + "/uaa"
}

// newClientFunc is the seam through which command RunE handlers obtain a CAPI
// client. It defaults to the real token-refreshing constructor. Tests reassign
// it to inject a fake client so RunE behavior can be exercised without standing
// up on-disk config or a live UAA. Production code must never reassign it.
//
//nolint:gochecknoglobals // intentional test seam for dependency injection
var newClientFunc = CreateClientWithTokenRefresh

// CreateClientWithAPI creates a CAPI client using the specified API or current API.
func CreateClientWithAPI(apiFlag string) (capi.Client, error) {
	return newClientFunc(apiFlag)
}

// CreateClientWithTokenRefresh creates a CAPI client with automatic token refresh.
func CreateClientWithTokenRefresh(apiFlag string) (capi.Client, error) {
	apiConfig, apiDomain, err := prepareClientConfig(apiFlag)
	if err != nil {
		return nil, err
	}

	setViperAPIConfig(apiConfig)
	tokenManager := createTokenManager(apiConfig, apiDomain)
	capiConfig := buildCAPIConfig(apiConfig)

	return createFinalClient(capiConfig, tokenManager, apiConfig)
}

func prepareClientConfig(apiFlag string) (*APIConfig, string, error) {
	apiConfig, err := getAPIConfigByFlag(apiFlag)
	if err != nil {
		return nil, "", err
	}

	if apiConfig.Endpoint == "" {
		return nil, "", fmt.Errorf("%w, use 'capi apis add' first", capi.ErrNoAPIEndpointConfigured)
	}

	apiDomain, err := findAPIDomain(apiConfig)
	if err != nil {
		return nil, "", err
	}

	return apiConfig, apiDomain, nil
}

func findAPIDomain(apiConfig *APIConfig) (string, error) {
	config := loadConfig()

	for domain, cfg := range config.APIs {
		if cfg.Endpoint == apiConfig.Endpoint {
			return domain, nil
		}
	}

	return "", capi.ErrCouldNotDetermineAPIDomain
}

func setViperAPIConfig(apiConfig *APIConfig) {
	viper.Set("api", apiConfig.Endpoint)
	viper.Set(organizationKey, apiConfig.Organization)
	viper.Set("organization_guid", apiConfig.OrganizationGUID)
	viper.Set(spaceKey, apiConfig.Space)
	viper.Set("space_guid", apiConfig.SpaceGUID)
	viper.Set("username", apiConfig.Username)
	viper.Set(uaaEndpointKey, apiConfig.UAAEndpoint)
}

func createTokenManager(apiConfig *APIConfig, apiDomain string) auth.TokenManager {
	if !hasAuthInfo(apiConfig) {
		return nil
	}

	uaaEndpoint := resolveUAAEndpoint(apiConfig)
	oauth2Config := buildOAuth2Config(apiConfig, uaaEndpoint)
	configPersister := NewConfigPersister()
	initialExpiry := getInitialTokenExpiry(apiConfig)

	return auth.NewConfigTokenManager(oauth2Config, configPersister, apiDomain, apiConfig.Token, initialExpiry)
}

func hasAuthInfo(apiConfig *APIConfig) bool {
	return apiConfig.Token != "" || apiConfig.RefreshToken != "" || apiConfig.Username != ""
}

func resolveUAAEndpoint(apiConfig *APIConfig) string {
	if apiConfig.UAAEndpoint != "" {
		return apiConfig.UAAEndpoint
	}

	return discoverUAAEndpoint(apiConfig.Endpoint)
}

func buildOAuth2Config(apiConfig *APIConfig, uaaEndpoint string) *auth.OAuth2Config {
	return &auth.OAuth2Config{
		TokenURL:     strings.TrimSuffix(uaaEndpoint, "/") + "/oauth/token",
		ClientID:     "cf", // Default CF CLI client ID
		ClientSecret: "",
		Username:     apiConfig.Username,
		RefreshToken: apiConfig.RefreshToken,
		AccessToken:  apiConfig.Token,
	}
}

func getInitialTokenExpiry(apiConfig *APIConfig) time.Time {
	if apiConfig.TokenExpiresAt != nil {
		return *apiConfig.TokenExpiresAt
	}

	return time.Time{}
}

func buildCAPIConfig(apiConfig *APIConfig) *capi.Config {
	return &capi.Config{
		APIEndpoint:   apiConfig.Endpoint,
		SkipTLSVerify: apiConfig.SkipSSLValidation,
		Username:      apiConfig.Username,
		TokenURL:      strings.TrimSuffix(apiConfig.UAAEndpoint, "/") + "/oauth/token",
	}
}

func createFinalClient(capiConfig *capi.Config, tokenManager auth.TokenManager, apiConfig *APIConfig) (capi.Client, error) {
	if tokenManager != nil {
		return createClientWithTokenManager(capiConfig, tokenManager)
	}

	if apiConfig.Token != "" {
		capiConfig.AccessToken = apiConfig.Token

		ctx := context.Background()

		client, err := cfclient.New(ctx, capiConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create CF client: %w", err)
		}

		return client, nil
	}

	return nil, fmt.Errorf("%w, use 'capi login' first", capi.ErrNotAuthenticated)
}

// createClientWithTokenManager creates a client with a custom token manager.
func createClientWithTokenManager(config *capi.Config, tokenManager auth.TokenManager) (capi.Client, error) {
	// Use the internal client package to create a client with token manager
	client, err := client.NewWithTokenManager(config, tokenManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create client with token manager: %w", err)
	}

	return client, nil
}

// setGlobalConfig sets a global configuration value.
func setGlobalConfig(config *Config, key, value string) error {
	switch key {
	case "output":
		config.Output = value
	case "no_color":
		if value == constants.BooleanTrue || value == "1" {
			config.NoColor = true
		} else {
			config.NoColor = false
		}
	default:
		return fmt.Errorf("%w: %s. Use --api flag for API-specific settings", capi.ErrUnknownConfigKey, key)
	}

	err := saveConfigStruct(config)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return outputConfigUpdateResult("Set global", key, value, "")
}

// setAPISpecificConfig sets configuration for a specific API.
func setAPISpecificConfig(config *Config, apiDomain, key, value string) error {
	apiConfig, err := validateAPIExists(config, apiDomain)
	if err != nil {
		return err
	}

	err = setAPIConfigValue(apiConfig, key, value)
	if err != nil {
		return err
	}

	config.APIs[apiDomain] = apiConfig

	err = saveConfigStruct(config)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return outputConfigUpdateResult("Set", key, value, apiDomain)
}

// validateAPIExists validates that an API exists in the configuration.
func validateAPIExists(config *Config, apiDomain string) (*APIConfig, error) {
	apiConfig, exists := config.APIs[apiDomain]
	if !exists {
		return nil, fmt.Errorf("%w. Use 'capi apis list' to see available APIs: '%s'", capi.ErrAPINotFound, apiDomain)
	}

	return apiConfig, nil
}

// setAPIConfigValue sets a specific configuration value for an API.
func setAPIConfigValue(apiConfig *APIConfig, key, value string) error {
	if handler, exists := getAPIConfigHandler(key); exists {
		handler(apiConfig, value)

		return nil
	}

	return fmt.Errorf("%w: %s", capi.ErrUnknownConfigKey, key)
}

// getAPIConfigHandler returns a handler function for a given config key.
func getAPIConfigHandler(key string) (func(*APIConfig, string), bool) {
	handlers := map[string]func(*APIConfig, string){
		"username":            func(c *APIConfig, v string) { c.Username = v },
		organizationKey:       func(c *APIConfig, v string) { c.Organization = v },
		"organization_guid":   func(c *APIConfig, v string) { c.OrganizationGUID = v },
		spaceKey:              func(c *APIConfig, v string) { c.Space = v },
		"space_guid":          func(c *APIConfig, v string) { c.SpaceGUID = v },
		"skip_ssl_validation": func(c *APIConfig, v string) { c.SkipSSLValidation = parseBoolValue(v) },
		uaaEndpointKey:        func(c *APIConfig, v string) { c.UAAEndpoint = v },
		uaaClientIDKey:        func(c *APIConfig, v string) { c.UAAClientID = v },
		uaaClientSecretKey:    func(c *APIConfig, v string) { c.UAAClientSecret = v },
	}
	handler, exists := handlers[key]

	return handler, exists
}

// parseBoolValue parses a boolean value from string.
func parseBoolValue(value string) bool {
	return value == constants.BooleanTrue || value == "1"
}

// unsetGlobalConfig unsets a global configuration value.
func unsetGlobalConfig(config *Config, key string) error {
	switch key {
	case "output":
		config.Output = "table"
	case "no_color":
		config.NoColor = false
	default:
		return fmt.Errorf("%w: %s. Use --api flag for API-specific settings", capi.ErrUnknownConfigKey, key)
	}

	err := saveConfigStruct(config)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return outputConfigUpdateResult("Unset global", key, "", "")
}

// unsetAPISpecificConfig unsets configuration for a specific API.
func unsetAPISpecificConfig(config *Config, apiDomain, key string) error {
	apiConfig, exists := config.APIs[apiDomain]
	if !exists {
		return fmt.Errorf("API '%s': %w. Use 'capi apis list' to see available APIs", apiDomain, capi.ErrAPINotFound)
	}

	switch key {
	case "username":
		apiConfig.Username = ""
	case organizationKey:
		apiConfig.Organization = ""
	case "organization_guid":
		apiConfig.OrganizationGUID = ""
	case spaceKey:
		apiConfig.Space = ""
	case "space_guid":
		apiConfig.SpaceGUID = ""
	case "skip_ssl_validation":
		apiConfig.SkipSSLValidation = false
	case uaaEndpointKey:
		apiConfig.UAAEndpoint = ""
	case uaaClientIDKey:
		apiConfig.UAAClientID = ""
	case uaaClientSecretKey:
		apiConfig.UAAClientSecret = ""
	// Token fields should not be unset via config command for security
	case tokenKey, "refresh_token", uaaTokenKey, uaaRefreshTokenKey:
		return fmt.Errorf("%w. Use 'capi logout' instead", capi.ErrTokenFieldsCannotUnset)
	default:
		return fmt.Errorf("%w: %s", capi.ErrUnknownConfigKey, key)
	}

	// Update the API config in the main config
	config.APIs[apiDomain] = apiConfig

	err := saveConfigStruct(config)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return outputConfigUpdateResult("Unset", key, "", apiDomain)
}

// showAPISpecificConfig shows configuration for a specific API.
func showAPISpecificConfig(config *Config, apiDomain string) error {
	apiConfig, exists := config.APIs[apiDomain]
	if !exists {
		return fmt.Errorf("API '%s': %w. Use 'capi apis list' to see available APIs", apiDomain, capi.ErrAPINotFound)
	}

	output := viper.GetString("output")
	switch output {
	case constants.FormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(apiConfig) //#nosec G117 -- nolint:gosec -- intentional CLI config persistence; refresh_token required for session restore
		if err != nil {
			return fmt.Errorf("failed to encode API config as JSON: %w", err)
		}

		return nil
	case constants.FormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(apiConfig) //#nosec G117 -- nolint:gosec -- intentional CLI config persistence; refresh_token required for session restore
		if err != nil {
			return fmt.Errorf("failed to encode API config as YAML: %w", err)
		}

		return nil
	default:
		return displayAPISpecificConfigTable(config, apiDomain, apiConfig)
	}
}

// clearAPISpecificConfig clears configuration for a specific API.
func clearAPISpecificConfig(config *Config, apiDomain string) error {
	apiConfig, exists := config.APIs[apiDomain]
	if !exists {
		return fmt.Errorf("API '%s': %w. Use 'capi apis list' to see available APIs", apiDomain, capi.ErrAPINotFound)
	}

	// Clear all configuration except the endpoint
	apiConfig.Token = ""
	apiConfig.TokenExpiresAt = nil
	apiConfig.RefreshToken = ""
	apiConfig.LastRefreshed = nil
	apiConfig.Username = ""
	apiConfig.Organization = ""
	apiConfig.OrganizationGUID = ""
	apiConfig.Space = ""
	apiConfig.SpaceGUID = ""
	apiConfig.SkipSSLValidation = false
	apiConfig.UAAEndpoint = ""
	apiConfig.UAAToken = ""
	apiConfig.UAARefreshToken = ""
	apiConfig.UAAClientID = ""
	apiConfig.UAAClientSecret = ""

	// Update the API config in the main config
	config.APIs[apiDomain] = apiConfig

	err := saveConfigStruct(config)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return outputConfigUpdateResult("Cleared configuration for API", apiDomain, "", "")
}

// displayConfigTable displays configuration in a table format.
func displayConfigTable(config *Config) error {
	err := displayGlobalConfigTable(config)
	if err != nil {
		return err
	}

	err = displayAPIsConfigTable(config)
	if err != nil {
		return err
	}

	err = displayLegacyConfigTable(config)
	if err != nil {
		return err
	}

	return displayLegacyTargetsTable(config)
}

func displayGlobalConfigTable(config *Config) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	addGlobalConfigRows(table, config)

	_, _ = os.Stdout.WriteString("Global Configuration:\n")

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	return nil
}

func addGlobalConfigRows(table *tablewriter.Table, config *Config) {
	_ = table.Append([]string{"Output", config.Output})
	_ = table.Append([]string{"No Color", strconv.FormatBool(config.NoColor)})

	if config.CurrentAPI != "" {
		_ = table.Append([]string{"Current API", config.CurrentAPI})
	}
}

func displayAPIsConfigTable(config *Config) error {
	if len(config.APIs) == 0 {
		_, _ = os.Stdout.WriteString("\nNo APIs configured. Use 'capi apis add' to add one.\n")

		return nil
	}

	_, _ = os.Stdout.WriteString("\nConfigured APIs:\n")

	apiTable := tablewriter.NewWriter(os.Stdout)
	apiTable.Header("Domain", "Endpoint", "Username", "Organization", "Space", "Current", "UAA Endpoint")

	addAPIConfigRows(apiTable, config)

	err := apiTable.Render()
	if err != nil {
		return fmt.Errorf("failed to render API config table: %w", err)
	}

	return nil
}

func addAPIConfigRows(table *tablewriter.Table, config *Config) {
	for domain, apiConfig := range config.APIs {
		row := buildAPIConfigRow(domain, apiConfig, config.CurrentAPI)
		_ = table.Append(row)
	}
}

func buildAPIConfigRow(domain string, apiConfig *APIConfig, currentAPI string) []string {
	return []string{
		domain,
		apiConfig.Endpoint,
		formatConfigValue(apiConfig.Username),
		formatConfigValue(apiConfig.Organization),
		formatConfigValue(apiConfig.Space),
		formatCurrentIndicator(domain == currentAPI),
		formatConfigValue(apiConfig.UAAEndpoint),
	}
}

func formatConfigValue(value string) string {
	if value == "" {
		return "-"
	}

	return value
}

func formatCurrentIndicator(isCurrent bool) string {
	if isCurrent {
		return "✓"
	}

	return ""
}

func displayLegacyConfigTable(config *Config) error {
	if config.API == "" {
		return nil
	}

	_, _ = os.Stdout.WriteString("\nLegacy Configuration (will be migrated):\n")

	legacyTable := tablewriter.NewWriter(os.Stdout)
	legacyTable.Header("Property", "Value")

	addLegacyConfigRows(legacyTable, config)

	err := legacyTable.Render()
	if err != nil {
		return fmt.Errorf("failed to render legacy config table: %w", err)
	}

	return nil
}

func addLegacyConfigRows(table *tablewriter.Table, config *Config) {
	_ = table.Append([]string{"API", config.API})

	if config.Username != "" {
		_ = table.Append([]string{"Username", config.Username})
	}

	if config.Organization != "" {
		_ = table.Append([]string{"Organization", config.Organization})
	}

	if config.Space != "" {
		_ = table.Append([]string{"Space", config.Space})
	}
}

func displayLegacyTargetsTable(config *Config) error {
	if len(config.Targets) == 0 {
		return nil
	}

	_, _ = os.Stdout.WriteString("\nLegacy Targets (deprecated):\n")

	targetsTable := tablewriter.NewWriter(os.Stdout)
	targetsTable.Header("Name", "API", "Username", "Organization", "Space")

	addLegacyTargetRows(targetsTable, config)

	err := targetsTable.Render()
	if err != nil {
		return fmt.Errorf("failed to render targets table: %w", err)
	}

	return nil
}

func addLegacyTargetRows(table *tablewriter.Table, config *Config) {
	for name, target := range config.Targets {
		row := buildLegacyTargetRow(name, target)
		_ = table.Append(row)
	}
}

func buildLegacyTargetRow(name string, target Target) []string {
	return []string{
		name,
		target.API,
		formatConfigValue(target.Username),
		formatConfigValue(target.Organization),
		formatConfigValue(target.Space),
	}
}

// displayAPISpecificConfigTable displays configuration for a specific API in table format.
func displayAPISpecificConfigTable(config *Config, apiDomain string, apiConfig *APIConfig) error {
	displayAPITableHeader(config, apiDomain)

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	err := populateAPIConfigTable(table, apiConfig)
	if err != nil {
		return err
	}

	err = table.Render()
	if err != nil {
		return fmt.Errorf("failed to render API config table: %w", err)
	}

	return nil
}

// displayAPITableHeader displays the header for API configuration table.
func displayAPITableHeader(config *Config, apiDomain string) {
	current := ""
	if apiDomain == config.CurrentAPI {
		current = " (current)"
	}

	_, _ = fmt.Fprintf(os.Stdout, "Configuration for API '%s'%s:\n", apiDomain, current)
}

// populateAPIConfigTable populates the API configuration table with rows.
func populateAPIConfigTable(table *tablewriter.Table, apiConfig *APIConfig) error {
	err := addBasicConfigRows(table, apiConfig)
	if err != nil {
		return err
	}

	err = addOrganizationSpaceRows(table, apiConfig)
	if err != nil {
		return err
	}

	err = addUAAConfigRows(table, apiConfig)
	if err != nil {
		return err
	}

	return addTokenRows(table, apiConfig)
}

// addBasicConfigRows adds basic configuration rows to the table.
func addBasicConfigRows(table *tablewriter.Table, apiConfig *APIConfig) error {
	err := table.Append([]string{"Endpoint", apiConfig.Endpoint})
	if err != nil {
		return fmt.Errorf("failed to append endpoint to config table: %w", err)
	}

	if apiConfig.Username != "" {
		err := table.Append([]string{"Username", apiConfig.Username})
		if err != nil {
			return fmt.Errorf("failed to append username to config table: %w", err)
		}
	}

	if apiConfig.SkipSSLValidation {
		err := table.Append([]string{"Skip SSL", strconv.FormatBool(apiConfig.SkipSSLValidation)})
		if err != nil {
			return fmt.Errorf("failed to append SSL skip setting to config table: %w", err)
		}
	}

	return nil
}

// addOrganizationSpaceRows adds organization and space rows to the table.
func addOrganizationSpaceRows(table *tablewriter.Table, apiConfig *APIConfig) error {
	orgSpaceRows := map[string]string{
		"Organization": apiConfig.Organization,
		"Org GUID":     apiConfig.OrganizationGUID,
		"Space":        apiConfig.Space,
		"Space GUID":   apiConfig.SpaceGUID,
	}

	for label, value := range orgSpaceRows {
		if value != "" {
			err := table.Append([]string{label, value})
			if err != nil {
				return fmt.Errorf("failed to append %s to config table: %w", label, err)
			}
		}
	}

	return nil
}

// addUAAConfigRows adds UAA configuration rows to the table.
func addUAAConfigRows(table *tablewriter.Table, apiConfig *APIConfig) error {
	uaaRows := map[string]string{
		"UAA Endpoint":  apiConfig.UAAEndpoint,
		"UAA Client ID": apiConfig.UAAClientID,
	}

	for label, value := range uaaRows {
		if value != "" {
			err := table.Append([]string{label, value})
			if err != nil {
				return fmt.Errorf("failed to append %s to config table: %w", label, err)
			}
		}
	}

	return nil
}

// addTokenRows adds token-related rows to the table (redacted for security).
func addTokenRows(table *tablewriter.Table, apiConfig *APIConfig) error {
	tokenRows := map[string]string{
		"Token":             apiConfig.Token,
		"Refresh Token":     apiConfig.RefreshToken,
		"UAA Token":         apiConfig.UAAToken,
		"UAA Refresh Token": apiConfig.UAARefreshToken,
	}

	for label, value := range tokenRows {
		if value != "" {
			err := table.Append([]string{label, "[REDACTED]"})
			if err != nil {
				return fmt.Errorf("failed to append %s to config table: %w", label, err)
			}
		}
	}

	return nil
}

// outputConfigUpdateResult outputs configuration update results in the requested format.
func outputConfigUpdateResult(action, key, value, apiDomain string) error {
	result := buildConfigResult(action, key, value, apiDomain)
	output := viper.GetString("output")

	switch output {
	case constants.FormatJSON:
		return outputConfigAsJSON(result)
	case constants.FormatYAML:
		return outputConfigAsYAML(result)
	default:
		return outputConfigAsTable(action, key, value, apiDomain)
	}
}

func buildConfigResult(action, key, value, apiDomain string) map[string]string {
	result := map[string]string{
		"action": action,
		"key":    key,
	}

	if value != "" {
		result["value"] = value
	}

	if apiDomain != "" {
		result["api_domain"] = apiDomain
	}

	return result
}

func outputConfigAsJSON(result map[string]string) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(result)
	if err != nil {
		return fmt.Errorf("failed to encode config result as JSON: %w", err)
	}

	return nil
}

func outputConfigAsYAML(result map[string]string) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(result)
	if err != nil {
		return fmt.Errorf("failed to encode config result as YAML: %w", err)
	}

	return nil
}

func outputConfigAsTable(action, key, value, apiDomain string) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	err := table.Append([]string{"Action", action})
	if err != nil {
		return fmt.Errorf("failed to append action to table: %w", err)
	}

	err = table.Append([]string{"Key", key})
	if err != nil {
		return fmt.Errorf("failed to append key to table: %w", err)
	}

	if value != "" {
		err := table.Append([]string{"Value", value})
		if err != nil {
			return fmt.Errorf("failed to append value to table: %w", err)
		}
	}

	if apiDomain != "" {
		err := table.Append([]string{"API Domain", apiDomain})
		if err != nil {
			return fmt.Errorf("failed to append API domain to table: %w", err)
		}
	}

	err = table.Render()
	if err != nil {
		return fmt.Errorf("failed to render update results table: %w", err)
	}

	return nil
}
