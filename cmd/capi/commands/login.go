package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/fivetwenty-io/capi/v3/pkg/cfclient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

// NewLoginCommand creates the login command.
func NewLoginCommand() *cobra.Command {
	var (
		apiEndpoint  string
		username     string
		password     string
		clientID     string
		clientSecret string
		ssoCode      string
		ssoPasscode  string
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to Cloud Foundry",
		Long:  "Authenticate with a Cloud Foundry API endpoint",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(loginParams{
				apiEndpoint:  apiEndpoint,
				username:     username,
				password:     password,
				clientID:     clientID,
				clientSecret: clientSecret,
				ssoCode:      ssoCode,
				ssoPasscode:  ssoPasscode,
			})
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&apiEndpoint, "api", "a", "", "API endpoint URL or short name from config")
	cmd.Flags().StringVarP(&username, "username", "u", "", "username for authentication")
	cmd.Flags().StringVarP(&password, "password", "p", "", "password for authentication")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth2 client ID")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth2 client secret")
	cmd.Flags().StringVar(&ssoCode, "sso-code", "", "SSO authorization code")
	cmd.Flags().StringVar(&ssoPasscode, "sso-passcode", "", "SSO one-time passcode")
	cmd.Flags().Bool("skip-ssl-validation", false, "skip SSL certificate validation")

	return cmd
}

// loginParams holds all login parameters.
type loginParams struct {
	apiEndpoint  string
	username     string
	password     string
	clientID     string
	clientSecret string
	ssoCode      string
	ssoPasscode  string
}

// runLogin handles the main login logic.
func runLogin(params loginParams) error {
	apiEndpoint, originalInput, err := resolveAPIEndpoint(params.apiEndpoint)
	if err != nil {
		return err
	}

	client, err := createLoginClient(apiEndpoint, params)
	if err != nil {
		return err
	}

	ctx := context.Background()

	info, rootInfo, err := fetchAPIInfo(ctx, client)
	if err != nil {
		return err
	}

	normalizedEndpoint, err := normalizeEndpoint(apiEndpoint)
	if err != nil {
		return fmt.Errorf("invalid API endpoint: %w", err)
	}

	configKey := determineConfigKey(originalInput, normalizedEndpoint)

	err = saveLoginConfig(configKey, normalizedEndpoint, params.username, client, rootInfo)
	if err != nil {
		return err
	}

	displayLoginSuccess(configKey, normalizedEndpoint, info, client)

	return nil
}

// resolveAPIEndpoint resolves and validates the API endpoint.
func resolveAPIEndpoint(apiEndpoint string) (string, string, error) {
	originalInput := apiEndpoint

	apiEndpoint, err := getAPIEndpointInput(apiEndpoint)
	if err != nil {
		return "", "", err
	}

	if apiEndpoint == "" {
		return "", "", ErrAPIEndpointRequired
	}

	resolvedEndpoint, err := ResolveAPIEndpoint(apiEndpoint)
	if err != nil {
		return "", "", err
	}

	return resolvedEndpoint, originalInput, nil
}

// getAPIEndpointInput gets API endpoint from various sources.
func getAPIEndpointInput(apiEndpoint string) (string, error) {
	if apiEndpoint == "" {
		apiEndpoint = viper.GetString("api")
	}

	if apiEndpoint == "" {
		apiEndpoint = getCurrentAPIFromConfig()
	}

	if apiEndpoint == "" {
		return promptForAPIEndpoint()
	}

	return apiEndpoint, nil
}

// getCurrentAPIFromConfig gets current API from config if available.
func getCurrentAPIFromConfig() string {
	config := loadConfig()
	if config.CurrentAPI != "" {
		if _, exists := config.APIs[config.CurrentAPI]; exists {
			return config.CurrentAPI
		}
	}

	return ""
}

// promptForAPIEndpoint prompts user for API endpoint input.
func promptForAPIEndpoint() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	_, _ = os.Stdout.WriteString("API endpoint (or short name): ")

	apiEndpoint, _ := reader.ReadString('\n')

	return strings.TrimSpace(apiEndpoint), nil
}

// createLoginClient creates a client with appropriate authentication.
func createLoginClient(apiEndpoint string, params loginParams) (capi.Client, error) {
	skipSSL := viper.GetBool("skip_ssl_validation")

	config := &capi.Config{
		APIEndpoint:   apiEndpoint,
		SkipTLSVerify: skipSSL,
	}

	setupAuthentication(config, params)

	ctx := context.Background()

	client, err := cfclient.New(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}

// setupAuthentication configures authentication method in the client config.
func setupAuthentication(config *capi.Config, params loginParams) {
	switch {
	case params.clientID != "" && params.clientSecret != "":
		config.ClientID = params.clientID
		config.ClientSecret = params.clientSecret
	case params.ssoCode != "" || params.ssoPasscode != "":
		config.AccessToken = params.ssoPasscode
	default:
		username, password := getUsernamePassword(params.username, params.password)
		config.Username = username
		config.Password = password
	}
}

// getUsernamePassword gets username and password, prompting if needed.
func getUsernamePassword(username, password string) (string, string) {
	if username == "" {
		username = promptForUsername()
	}

	if password == "" {
		password = promptForPassword()
	}

	return username, password
}

// promptForUsername prompts user for username.
func promptForUsername() string {
	reader := bufio.NewReader(os.Stdin)

	_, _ = os.Stdout.WriteString("Username: ")

	username, _ := reader.ReadString('\n')

	return strings.TrimSpace(username)
}

// promptForPassword prompts user for password securely.
func promptForPassword() string {
	_, _ = os.Stdout.WriteString("Password: ")

	bytePassword, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return ""
	}

	_, _ = os.Stdout.WriteString("\n")

	return string(bytePassword)
}

// fetchAPIInfo fetches API info and root info from the client.
func fetchAPIInfo(ctx context.Context, client capi.Client) (*capi.Info, *capi.RootInfo, error) {
	info, err := client.GetInfo(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to API: %w", err)
	}

	rootInfo, err := client.GetRootInfo(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Warning: could not fetch API endpoints: %v\n", err)
	}

	return info, rootInfo, nil
}

// determineConfigKey determines the configuration key for storing API config.
func determineConfigKey(originalInput, normalizedEndpoint string) string {
	currentConfig := loadConfig()
	if _, exists := currentConfig.APIs[originalInput]; exists {
		return originalInput
	}

	return extractDomainFromEndpoint(normalizedEndpoint)
}

// saveLoginConfig saves the login configuration.
func saveLoginConfig(configKey, normalizedEndpoint, username string, client capi.Client, rootInfo *capi.RootInfo) error {
	configStruct := loadConfig()

	if configStruct.APIs == nil {
		configStruct.APIs = make(map[string]*APIConfig)
	}

	apiConfig := getOrCreateAPIConfig(configStruct, configKey, normalizedEndpoint)
	updateAPIConfig(apiConfig, username, client, rootInfo)
	setCurrentAPIIfNeeded(configStruct, configKey)

	return saveConfigStruct(configStruct)
}

// getOrCreateAPIConfig gets existing API config or creates a new one.
func getOrCreateAPIConfig(configStruct *Config, configKey, normalizedEndpoint string) *APIConfig {
	apiConfig, exists := configStruct.APIs[configKey]
	if !exists {
		apiConfig = &APIConfig{
			Endpoint: normalizedEndpoint,
		}
		configStruct.APIs[configKey] = apiConfig
	}

	return apiConfig
}

// updateAPIConfig updates API configuration with authentication info.
func updateAPIConfig(apiConfig *APIConfig, username string, client capi.Client, rootInfo *capi.RootInfo) {
	apiConfig.Username = username
	apiConfig.SkipSSLValidation = viper.GetBool("skip_ssl_validation")

	updateAPIConfigToken(apiConfig, client)
	updateAPIConfigLinks(apiConfig, rootInfo)
}

// updateAPIConfigToken updates token information in API config.
func updateAPIConfigToken(apiConfig *APIConfig, client capi.Client) {
	if tokenGetter, ok := client.(interface {
		GetToken(ctx context.Context) (string, error)
	}); ok {
		ctx := context.Background()

		token, err := tokenGetter.GetToken(ctx)
		if err == nil && token != "" {
			apiConfig.Token = token
			now := time.Now()
			apiConfig.LastRefreshed = &now
		}
	}
}

// updateAPIConfigLinks updates API links from root info.
func updateAPIConfigLinks(apiConfig *APIConfig, rootInfo *capi.RootInfo) {
	if rootInfo != nil && rootInfo.Links != nil {
		apiConfig.APILinks = make(map[string]string)
		for key, link := range rootInfo.Links {
			apiConfig.APILinks[key] = link.Href
		}
	}
}

// setCurrentAPIIfNeeded sets current API if this is the first one.
func setCurrentAPIIfNeeded(configStruct *Config, configKey string) {
	if configStruct.CurrentAPI == "" || len(configStruct.APIs) == 1 {
		configStruct.CurrentAPI = configKey
	}
}

// displayLoginSuccess displays success message and available organizations.
func displayLoginSuccess(configKey, normalizedEndpoint string, info *capi.Info, client capi.Client) {
	configStruct := loadConfig()
	isFirstAPI := len(configStruct.APIs) == 1

	_, _ = fmt.Fprintf(os.Stdout, "Successfully logged in to %s\n", normalizedEndpoint)

	if isFirstAPI {
		_, _ = fmt.Fprintf(os.Stdout, "API '%s' set as current target\n", configKey)
	}

	_, _ = fmt.Fprintf(os.Stdout, "API version: %d\n", info.Version)

	displayAvailableOrganizations(client)
}

// displayAvailableOrganizations displays available organizations if accessible.
func displayAvailableOrganizations(client capi.Client) {
	if orgsClient, ok := client.(interface {
		Organizations() capi.OrganizationsClient
	}); ok {
		ctx := context.Background()

		orgs, err := orgsClient.Organizations().List(ctx, nil)
		if err == nil && len(orgs.Resources) > 0 {
			_, _ = os.Stdout.WriteString("\nAvailable organizations:\n")

			for _, org := range orgs.Resources {
				_, _ = fmt.Fprintf(os.Stdout, "  - %s\n", org.Name)
			}

			_, _ = os.Stdout.WriteString("\nUse 'capi target -o <org>' to target an organization\n")
		}
	}
}

// NewLogoutCommand creates the logout command.
func NewLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Logout from Cloud Foundry",
		Long:  "Clear authentication credentials and logout from Cloud Foundry",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Clear authentication data
			viper.Set(tokenKey, "")
			viper.Set("refresh_token", "")
			viper.Set("username", "")
			viper.Set("password", "")
			viper.Set(organizationKey, "")
			viper.Set(spaceKey, "")

			err := saveConfig()
			if err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}

			_, _ = os.Stdout.WriteString("Successfully logged out\n")

			return nil
		},
	}
}
