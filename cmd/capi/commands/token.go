package commands

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fivetwenty-io/capi/v3/internal/auth"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewTokenCommand creates the token command group.
func NewTokenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   Token,
		Short: "Manage authentication tokens",
		Long:  "Commands for managing authentication tokens including status and refresh",
	}

	cmd.AddCommand(newTokenStatusCommand())
	cmd.AddCommand(newTokenRefreshCommand())

	return cmd
}

func newTokenStatusCommand() *cobra.Command {
	var (
		apiFlag string
		showAll bool
	)

	cmd := &cobra.Command{
		Use:   Status,
		Short: "Show token status and expiration",
		Long:  "Display information about the current authentication token including expiration time",
		RunE: func(cmd *cobra.Command, args []string) error {
			config := loadConfig()

			// If --all flag is specified, show all APIs
			if showAll {
				if len(config.APIs) == 0 {
					return constants.ErrNoAPIsConfigured
				}

				return displayAllTokenStatus(config)
			}

			// If no API specified, show current API or all APIs
			if apiFlag == "" {
				if len(config.APIs) == 0 {
					return constants.ErrNoAPIsConfigured
				}

				// If there's a current API, show just that one, otherwise show all
				if config.CurrentAPI != "" {
					if apiConfig, exists := config.APIs[config.CurrentAPI]; exists {
						return displayTokenStatus(apiConfig, config.CurrentAPI)
					}
				}

				// Show all APIs
				return displayAllTokenStatus(config)
			}

			// Get specific API config
			apiConfig, err := getAPIConfigByFlag(apiFlag)
			if err != nil {
				return err
			}

			// Find the API domain key by matching the endpoint or using the flag directly
			apiDomain, err := findAPIDomainByConfig(apiConfig, apiFlag)
			if err != nil {
				return err
			}

			return displayTokenStatus(apiConfig, apiDomain)
		},
	}

	cmd.Flags().StringVar(&apiFlag, "api", "", "show token status for specific API")
	cmd.Flags().BoolVar(&showAll, "all", false, "show token status for all configured APIs")

	return cmd
}

func newTokenRefreshCommand() *cobra.Command {
	var apiFlag string

	cmd := &cobra.Command{
		Use:   Refresh,
		Short: "Manually refresh authentication token",
		Long:  "Force refresh the authentication token using the stored refresh token",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get API config based on flag or current API
			apiConfig, err := getAPIConfigByFlag(apiFlag)
			if err != nil {
				return err
			}

			// Check if we have a refresh token
			if apiConfig.RefreshToken == "" {
				return constants.ErrNoRefreshToken
			}

			// Find the API domain key by matching the endpoint or using the flag directly
			apiDomain, err := findAPIDomainByConfig(apiConfig, apiFlag)
			if err != nil {
				return err
			}

			return refreshToken(apiConfig, apiDomain)
		},
	}

	cmd.Flags().StringVar(&apiFlag, "api", "", "refresh token for specific API")

	return cmd
}

func displayTokenStatus(apiConfig *APIConfig, apiDomain string) error {
	output := viper.GetString("output")
	tokenStatus := buildTokenStatusData(apiConfig, apiDomain)

	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(tokenStatus)
		if err != nil {
			return fmt.Errorf("encoding token status to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(tokenStatus)
		if err != nil {
			return fmt.Errorf("failed to encode token status as YAML: %w", err)
		}

		return nil
	default:
		return displayTokenStatusTable(tokenStatus)
	}
}

func displayTokenStatusTable(tokenStatus map[string]any) error {
	_, _ = fmt.Fprintf(os.Stdout, "Token Status for API: %s\n", tokenStatus["api_domain"])
	_, _ = fmt.Fprintf(os.Stdout, "Endpoint: %s\n\n", tokenStatus[keyEndpoint])

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	err := table.Append([]string{"Authenticated", fmt.Sprintf("%v", tokenStatus["authenticated"])})
	if err != nil {
		return fmt.Errorf("failed to append authenticated status: %w", err)
	}

	err = table.Append([]string{"Status", fmt.Sprintf("%v", tokenStatus[keyStatus])})
	if err != nil {
		return fmt.Errorf("failed to append status: %w", err)
	}

	if expiryStatus, ok := tokenStatus["expiry_status"]; ok {
		err := table.Append([]string{"Expiry Status", fmt.Sprintf("%v", expiryStatus)})
		if err != nil {
			return fmt.Errorf("failed to append expiry status: %w", err)
		}
	}

	if expiresAt, ok := tokenStatus["expires_at"]; ok {
		err := table.Append([]string{"Expires At", fmt.Sprintf("%v", expiresAt)})
		if err != nil {
			return fmt.Errorf("failed to append expires at to table: %w", err)
		}
	}

	if timeUntilExpiry, ok := tokenStatus["time_until_expiry"]; ok {
		err := table.Append([]string{"Time Until Expiry", fmt.Sprintf("%v", timeUntilExpiry)})
		if err != nil {
			return fmt.Errorf("failed to append time until expiry to table: %w", err)
		}
	}

	if lastRefreshed, ok := tokenStatus["last_refreshed"]; ok {
		err := table.Append([]string{"Last Refreshed", fmt.Sprintf("%v", lastRefreshed)})
		if err != nil {
			return fmt.Errorf("failed to append last refreshed to table: %w", err)
		}
	}

	if refreshAvailable, ok := tokenStatus["refresh_token_available"]; ok {
		err := table.Append([]string{"Refresh Token Available", fmt.Sprintf("%v", refreshAvailable)})
		if err != nil {
			return fmt.Errorf("failed to append table row: %w", err)
		}
	}

	err = table.Render()
	if err != nil {
		return fmt.Errorf("failed to render token status table: %w", err)
	}

	return nil
}

func displayAllTokenStatus(config *Config) error {
	output := viper.GetString("output")

	if output == OutputFormatJSON || output == OutputFormatYAML {
		// For structured output, show all APIs in one object
		allStatus := make(map[string]any)

		for domain, apiConfig := range config.APIs {
			tokenStatus := buildTokenStatusData(apiConfig, domain)
			allStatus[domain] = tokenStatus
		}

		switch output {
		case OutputFormatJSON:
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")

			err := encoder.Encode(allStatus)
			if err != nil {
				return fmt.Errorf("encoding all token status to JSON: %w", err)
			}

			return nil
		case OutputFormatYAML:
			encoder := yaml.NewEncoder(os.Stdout)

			err := encoder.Encode(allStatus)
			if err != nil {
				return fmt.Errorf("failed to encode all status as YAML: %w", err)
			}

			return nil
		}
	}

	// For table output, show each API separately
	first := true
	for domain, apiConfig := range config.APIs {
		if !first {
			_, _ = os.Stdout.WriteString("\n") // Add spacing between APIs
		}

		first = false

		err := displayTokenStatus(apiConfig, domain)
		if err != nil {
			return err
		}
	}

	return nil
}

func buildTokenStatusData(apiConfig *APIConfig, apiDomain string) map[string]any {
	tokenStatus := map[string]any{
		"api_domain": apiDomain,
		keyEndpoint:  apiConfig.Endpoint,
	}

	if apiConfig.Token == "" {
		tokenStatus[keyStatus] = "No token"
		tokenStatus["authenticated"] = false

		return tokenStatus
	}

	populateTokenInfo(tokenStatus, apiConfig)

	return tokenStatus
}

// populateTokenInfo adds token information to the status map when a token is present.
func populateTokenInfo(tokenStatus map[string]any, apiConfig *APIConfig) {
	tokenStatus[keyStatus] = "Token present"
	tokenStatus["authenticated"] = true

	// Add expiration info if available
	expiresAt := getTokenExpiration(apiConfig)
	if expiresAt != nil {
		addExpirationInfo(tokenStatus, expiresAt)
	} else {
		tokenStatus["expiry_status"] = "Unknown expiration"
	}

	// Add last refresh info if available
	if apiConfig.LastRefreshed != nil {
		tokenStatus["last_refreshed"] = apiConfig.LastRefreshed.Format(time.RFC3339)
	}

	// Check if we have a refresh token
	tokenStatus["refresh_token_available"] = apiConfig.RefreshToken != ""
}

// getTokenExpiration gets the token expiration time from config or JWT.
func getTokenExpiration(apiConfig *APIConfig) *time.Time {
	if apiConfig.TokenExpiresAt != nil {
		return apiConfig.TokenExpiresAt
	}

	// Try to decode from JWT token
	jwtExp, err := decodeJWTExpiration(apiConfig.Token)
	if err == nil {
		return jwtExp
	}

	return nil
}

// addExpirationInfo adds expiration status and timing information.
func addExpirationInfo(tokenStatus map[string]any, expiresAt *time.Time) {
	tokenStatus["expires_at"] = expiresAt.Format(time.RFC3339)

	now := time.Now()
	timeUntilExpiry := expiresAt.Sub(now)

	switch {
	case timeUntilExpiry <= 0:
		tokenStatus["expiry_status"] = "Expired"
	case timeUntilExpiry <= 5*time.Minute:
		tokenStatus["expiry_status"] = "Expires soon"
	default:
		tokenStatus["expiry_status"] = "Valid"
	}

	tokenStatus["time_until_expiry"] = timeUntilExpiry.String()
}

// decodeJWTExpiration extracts expiration time from a JWT token.
func decodeJWTExpiration(token string) (*time.Time, error) {
	// Split the JWT into its parts
	parts := strings.Split(token, ".")
	if len(parts) != constants.TokenPartsCount {
		return nil, constants.ErrInvalidJWTFormat
	}

	// Decode the payload (second part)
	payload := parts[1]

	// Add padding if necessary
	if len(payload)%4 != 0 {
		payload += strings.Repeat("=", constants.Base64PaddingLength-len(payload)%constants.Base64PaddingLength)
	}

	payloadBytes, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	// Parse the JSON payload
	var claims struct {
		Exp int64 `json:"exp"`
	}

	err = json.Unmarshal(payloadBytes, &claims)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	if claims.Exp == 0 {
		return nil, constants.ErrNoExpirationClaim
	}

	expTime := time.Unix(claims.Exp, 0)

	return &expTime, nil
}

func refreshToken(apiConfig *APIConfig, apiDomain string) error {
	_, _ = fmt.Fprintf(os.Stdout, "Refreshing token for API: %s\n", apiDomain)

	// Create OAuth2 config for refresh
	uaaEndpoint := apiConfig.UAAEndpoint
	if uaaEndpoint == "" {
		// Try to discover from API endpoint
		uaaEndpoint = discoverUAAEndpoint(apiConfig.Endpoint)
	}

	oauth2Config := &auth.OAuth2Config{
		TokenURL:     uaaEndpoint + "/oauth/token",
		ClientID:     "cf", // Default CF CLI client ID
		ClientSecret: "",
		RefreshToken: apiConfig.RefreshToken,
	}

	// Create token manager
	tokenManager := auth.NewOAuth2TokenManager(oauth2Config)

	// Force refresh
	ctx := context.Background()

	err := tokenManager.RefreshToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Get the new token
	newToken := tokenManager.GetTokenStore().Get()
	if newToken == nil {
		return constants.ErrFailedRetrieveToken
	}

	// Update config
	config := loadConfig()
	if config.APIs[apiDomain] == nil {
		return constants.ErrAPIConfigNotFound
	}

	config.APIs[apiDomain].Token = newToken.AccessToken
	if newToken.RefreshToken != "" {
		config.APIs[apiDomain].RefreshToken = newToken.RefreshToken
	}

	if !newToken.ExpiresAt.IsZero() {
		config.APIs[apiDomain].TokenExpiresAt = &newToken.ExpiresAt
	}

	now := time.Now()
	config.APIs[apiDomain].LastRefreshed = &now

	// Save config
	err = saveConfigStruct(config)
	if err != nil {
		return fmt.Errorf("failed to save updated token to config: %w", err)
	}

	_, _ = os.Stdout.WriteString("Token refreshed successfully!\n")

	// Show new expiration
	if !newToken.ExpiresAt.IsZero() {
		_, _ = fmt.Fprintf(os.Stdout, "New token expires at: %s\n", newToken.ExpiresAt.Format(time.RFC3339))
		timeUntilExpiry := time.Until(newToken.ExpiresAt)
		_, _ = fmt.Fprintf(os.Stdout, "Time until expiry: %s\n", timeUntilExpiry.String())
	}

	return nil
}

// findAPIDomainByConfig finds the API domain for a given API config and flag.
func findAPIDomainByConfig(apiConfig *APIConfig, apiFlag string) (string, error) {
	config := loadConfig()

	if apiFlag != "" {
		// First try to use the flag directly as domain
		if _, exists := config.APIs[apiFlag]; exists {
			return apiFlag, nil
		}

		// Otherwise find by endpoint match
		for domain, cfg := range config.APIs {
			if cfg.Endpoint == apiConfig.Endpoint {
				return domain, nil
			}
		}

		return "", fmt.Errorf("%w for '%s'", constants.ErrNoDomainForAPI, apiFlag)
	}

	// If no flag provided, use the current API or find by endpoint
	if config.CurrentAPI != "" {
		return config.CurrentAPI, nil
	}

	// Fallback to endpoint-based lookup
	return findAPIDomain(apiConfig)
}
