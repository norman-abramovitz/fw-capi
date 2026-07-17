package commands

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cloudfoundry-community/go-uaa"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

// uaaInfoKeyApp is the "app" key in the map returned by GetServerInfo,
// holding the UAA server's nested application info block.
const uaaInfoKeyApp = "app"

// UAAClientWrapper provides a convenient wrapper around the go-uaa client
// with configuration integration and token management.
type UAAClientWrapper struct {
	client   *uaa.API
	config   *Config
	endpoint string
	skipSSL  bool
	verbose  bool
}

// UAATokenInfo represents token information for display.
type UAATokenInfo struct {
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	JTI          string    `json:"jti,omitempty"`
}

// isDevelopmentEnvironment checks if we're in a development environment.
func isDevelopmentEnvironment() bool {
	devMode := os.Getenv("CAPI_DEV_MODE")

	return devMode == "true" || devMode == "1"
}

// NewUAAClient creates a new UAA client wrapper.
func NewUAAClient(config *Config) (*UAAClientWrapper, error) {
	if config == nil {
		config = loadConfig()
	}

	wrapper := &UAAClientWrapper{
		config:  config,
		verbose: viper.GetBool("verbose"),
		skipSSL: config.SkipSSLValidation,
	}

	// Determine UAA endpoint
	uaaEndpoint, err := wrapper.discoverUAAEndpoint()
	if err != nil {
		return nil, fmt.Errorf("failed to discover UAA endpoint: %w", err)
	}

	wrapper.endpoint = uaaEndpoint

	// Create UAA API client
	client, err := wrapper.createUAAClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create UAA client: %w", err)
	}

	wrapper.client = client

	return wrapper, nil
}

// NewUAAClientWithEndpoint creates a UAA client with a specific endpoint.
func NewUAAClientWithEndpoint(endpoint string, config *Config) (*UAAClientWrapper, error) {
	if config == nil {
		config = loadConfig()
	}

	wrapper := &UAAClientWrapper{
		config:   config,
		endpoint: endpoint,
		verbose:  viper.GetBool("verbose"),
		skipSSL:  config.SkipSSLValidation,
	}

	// Create UAA API client
	client, err := wrapper.createUAAClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create UAA client: %w", err)
	}

	wrapper.client = client

	return wrapper, nil
}

// Client returns the underlying UAA API client.
func (w *UAAClientWrapper) Client() *uaa.API {
	return w.client
}

// Endpoint returns the current UAA endpoint.
func (w *UAAClientWrapper) Endpoint() string {
	return w.endpoint
}

// IsAuthenticated checks if the client has valid authentication.
func (w *UAAClientWrapper) IsAuthenticated() bool {
	// Check legacy tokens first
	if w.config.UAAToken != "" || w.config.Token != "" {
		return true
	}

	// Check current API configuration for tokens
	if w.config.CurrentAPI != "" {
		if apiConfig, exists := w.config.APIs[w.config.CurrentAPI]; exists {
			return apiConfig.UAAToken != "" || apiConfig.Token != ""
		}
	}

	return false
}

// SetToken sets the authentication token for UAA requests.
func (w *UAAClientWrapper) SetToken(token string) {
	// Set in current API configuration if available
	if w.config.CurrentAPI != "" {
		if apiConfig, exists := w.config.APIs[w.config.CurrentAPI]; exists {
			apiConfig.UAAToken = token
			w.config.APIs[w.config.CurrentAPI] = apiConfig
		}
	} else {
		// Fallback to legacy configuration
		w.config.UAAToken = token
	}
	// Note: The go-uaa client handles authentication internally via its configuration
}

// GetToken retrieves the current authentication token.
func (w *UAAClientWrapper) GetToken() string {
	// Check legacy UAA token first
	if w.config.UAAToken != "" {
		return w.config.UAAToken
	}

	// Check current API configuration for UAA token
	if token := w.getTokenFromCurrentAPI(); token != "" {
		return token
	}

	// Fallback to legacy CF token if UAA token is not available
	return w.config.Token
}

// RefreshToken attempts to refresh the current access token.
func (w *UAAClientWrapper) RefreshToken(ctx context.Context) (*UAATokenInfo, error) {
	// Use the client's built-in token refresh capability
	token, err := w.client.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get/refresh token: %w", err)
	}

	// Update stored tokens
	w.config.UAAToken = token.AccessToken
	if token.RefreshToken != "" {
		w.config.UAARefreshToken = token.RefreshToken
	}

	// Convert to our token info structure
	tokenInfo := &UAATokenInfo{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
	}

	if !token.Expiry.IsZero() {
		tokenInfo.ExpiresAt = token.Expiry
		tokenInfo.ExpiresIn = int(time.Until(token.Expiry).Seconds())
	}

	return tokenInfo, nil
}

// TestConnection tests the connection to the UAA endpoint.
func (w *UAAClientWrapper) TestConnection(ctx context.Context) error {
	if w.client == nil {
		return constants.ErrUAAClientNotInit
	}

	// Try to get server info as a connectivity test
	_, err := w.client.GetInfo()
	if err != nil {
		return fmt.Errorf("failed to connect to UAA: %w", err)
	}

	return nil
}

// GetServerInfo retrieves UAA server information.
func (w *UAAClientWrapper) GetServerInfo(ctx context.Context) (map[string]any, error) {
	if w.client == nil {
		return nil, constants.ErrUAAClientNotInit
	}

	info, err := w.client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}

	// Convert to map for easier handling
	result := map[string]any{
		uaaInfoKeyApp: info.App,
		"commit_id":   info.CommitID,
		"timestamp":   info.Timestamp,
		"links":       info.Links,
		"zone_name":   info.ZoneName,
		"entity_id":   info.EntityID,
	}

	return result, nil
}

// SaveConfig saves the current configuration including UAA tokens.
func (w *UAAClientWrapper) SaveConfig() error {
	return saveConfigStruct(w.config)
}

// getTokenFromCurrentAPI retrieves token from current API configuration.
func (w *UAAClientWrapper) getTokenFromCurrentAPI() string {
	if w.config.CurrentAPI == "" {
		return ""
	}

	apiConfig, exists := w.config.APIs[w.config.CurrentAPI]
	if !exists {
		return ""
	}

	if apiConfig.UAAToken != "" {
		return apiConfig.UAAToken
	}

	// Fallback to CF token if UAA token is not available
	if apiConfig.Token != "" {
		return apiConfig.Token
	}

	return ""
}

// discoverUAAEndpoint attempts to discover the UAA endpoint from various sources.
func (w *UAAClientWrapper) discoverUAAEndpoint() (string, error) {
	// 1. Check if UAA endpoint is explicitly configured in legacy config
	if w.config.UAAEndpoint != "" {
		return w.config.UAAEndpoint, nil
	}

	// 1.5. Check if UAA endpoint is configured in the current API
	if w.config.CurrentAPI != "" {
		if apiConfig, exists := w.config.APIs[w.config.CurrentAPI]; exists && apiConfig.UAAEndpoint != "" {
			return apiConfig.UAAEndpoint, nil
		}
	}

	// 2. Try to discover from CF API endpoint
	apiEndpoint := w.getCurrentAPIEndpoint()
	if apiEndpoint != "" {
		uaaEndpoint, err := w.discoverFromCFAPIEndpoint(apiEndpoint)
		if err == nil && uaaEndpoint != "" {
			return uaaEndpoint, nil
		}

		if w.verbose {
			_, _ = fmt.Fprintf(os.Stdout, "Failed to discover UAA from CF API: %v\n", err)
		}
	}

	// 3. Try to infer from CF API endpoint (common patterns)
	if apiEndpoint != "" {
		if inferredEndpoint := w.inferUAAEndpointFromAPI(apiEndpoint); inferredEndpoint != "" {
			return inferredEndpoint, nil
		}
	}

	return "", constants.ErrNoUAAEndpoint
}

// getCurrentAPIEndpoint returns the current API endpoint from config.
func (w *UAAClientWrapper) getCurrentAPIEndpoint() string {
	// Check current API configuration first
	if w.config.CurrentAPI != "" {
		if apiConfig, exists := w.config.APIs[w.config.CurrentAPI]; exists {
			return apiConfig.Endpoint
		}
	}
	// Fallback to legacy API endpoint
	return w.config.API
}

// discoverFromCFAPIEndpoint attempts to discover UAA endpoint from a specific CF API endpoint.
func (w *UAAClientWrapper) discoverFromCFAPIEndpoint(apiEndpoint string) (string, error) {
	if apiEndpoint == "" {
		return "", constants.ErrNoCFAPIEndpoint
	}

	// Create HTTP client with appropriate SSL settings
	var transport *http.Transport

	if w.skipSSL {
		if !isDevelopmentEnvironment() {
			return "", constants.ErrSSLOnlyInDev
		}

		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // #nosec G402 -- Protected by development environment check above
			},
		}
	} else {
		transport = &http.Transport{}
	}

	httpClient := &http.Client{
		Timeout:   constants.QuickOperationTimeout,
		Transport: transport,
	}

	// Get the root info from the CF API
	rootURL := strings.TrimSuffix(apiEndpoint, "/") + "/"

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rootURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get CF API root info: %w", err)
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d", constants.ErrCFAPIRequestFailed, resp.StatusCode)
	}

	var rootInfo struct {
		Links struct {
			UAA struct {
				HREF string `json:"href"`
			} `json:"uaa"`
		} `json:"links"`
	}

	err = json.NewDecoder(resp.Body).Decode(&rootInfo)
	if err != nil {
		return "", fmt.Errorf("failed to decode CF API root info: %w", err)
	}

	if rootInfo.Links.UAA.HREF == "" {
		return "", constants.ErrNoUAAInCFLinks
	}

	return rootInfo.Links.UAA.HREF, nil
}

// inferUAAEndpointFromAPI infers UAA endpoint from CF API endpoint using common patterns.
func (w *UAAClientWrapper) inferUAAEndpointFromAPI(apiEndpoint string) string {
	if apiEndpoint == "" {
		return ""
	}

	// Parse the API endpoint
	parsed, err := url.Parse(apiEndpoint)
	if err != nil {
		return ""
	}

	// Common pattern: replace 'api.' with 'uaa.'
	if strings.HasPrefix(parsed.Host, "api.") {
		uaaHost := strings.Replace(parsed.Host, "api.", "uaa.", 1)

		return fmt.Sprintf("%s://%s", parsed.Scheme, uaaHost)
	}

	return ""
}

// createUAAClient creates the underlying UAA API client.
func (w *UAAClientWrapper) createUAAClient() (*uaa.API, error) {
	if w.endpoint == "" {
		return nil, constants.ErrNoUAASpecified
	}

	// Create HTTP client with appropriate settings
	var transport *http.Transport

	if w.skipSSL {
		if !isDevelopmentEnvironment() {
			return nil, constants.ErrSSLOnlyInDev
		}

		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // #nosec G402 -- Protected by development environment check above
			},
		}
	} else {
		transport = &http.Transport{}
	}

	httpClient := &http.Client{
		Timeout:   constants.UAAClientTimeout,
		Transport: transport,
	}

	// Determine authentication method
	var authOpt uaa.AuthenticationOption

	if token := w.GetToken(); token != "" {
		// Create oauth2.Token from string
		oauthToken := &oauth2.Token{
			AccessToken: token,
		}
		authOpt = uaa.WithToken(oauthToken)
	} else {
		// No authentication for now
		authOpt = uaa.WithNoAuthentication()
	}

	// Create UAA API client
	client, err := uaa.New(
		w.endpoint,
		authOpt,
		uaa.WithClient(httpClient),
		uaa.WithVerbosity(w.verbose),
		uaa.WithSkipSSLValidation(w.skipSSL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create UAA client: %w", err)
	}

	return client, nil
}
