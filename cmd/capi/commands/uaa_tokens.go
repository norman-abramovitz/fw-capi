package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"syscall"
	"time"

	"github.com/cloudfoundry-community/go-uaa"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

// createUsersGetAuthcodeTokenCommand creates the authorization code token command.
// promptForAuthcodeInputs prompts user for missing authorization code grant inputs.
func promptForAuthcodeInputs(clientID, clientSecret, authCode, redirectURI *string) error {
	if *clientID == "" {
		log.Print("Client ID: ")

		_, _ = fmt.Scanln(clientID)
	}

	if *clientSecret == "" {
		log.Print("Client Secret: ")

		secretBytes, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read client secret: %w", err)
		}

		*clientSecret = string(secretBytes)

		_, _ = os.Stdout.WriteString("\n") // Add newline after password input
	}

	if *authCode == "" {
		log.Print("Authorization Code: ")

		_, _ = fmt.Scanln(authCode)
	}

	if *redirectURI == "" {
		log.Print("Redirect URI: ")

		_, _ = fmt.Scanln(redirectURI)
	}

	return nil
}

// getAuthcodeToken retrieves and stores the authorization code token.
func getAuthcodeToken(config *Config, clientID, clientSecret, authCode, redirectURI string, tokenFormat int) error {
	// Parse redirect URI
	redirectURL, err := url.Parse(redirectURI)
	if err != nil {
		return fmt.Errorf("invalid redirect URI: %w", err)
	}

	// Create UAA client with authorization code authentication
	authOpt := uaa.WithAuthorizationCode(clientID, clientSecret, authCode, uaa.TokenFormat(tokenFormat), redirectURL)

	client, err := uaa.New(config.UAAEndpoint, authOpt)
	if err != nil {
		return fmt.Errorf("failed to create UAA client: %w", err)
	}

	// Get token
	ctx := context.Background()

	token, err := client.Token(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authorization code token: %w", err)
	}

	// Store tokens in config
	config.UAAToken = token.AccessToken
	if token.RefreshToken != "" {
		config.UAARefreshToken = token.RefreshToken
	}

	config.UAAClientID = clientID

	// Save configuration
	err = saveConfigStruct(config)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Warning: Failed to save token to configuration: %v\n", err)
	}

	// Display token information
	return displayTokenInfo(token, "Authorization Code Grant")
}

func createUsersGetAuthcodeTokenCommand() *cobra.Command {
	var (
		clientID, clientSecret, redirectURI, authCode string
		tokenFormat                                   int
	)

	cmd := &cobra.Command{
		Use:     "get-authcode-token",
		Aliases: []string{"authcode", "auth-code"},
		Short:   "Obtain access token using authorization code grant",
		Long: `Obtain an access token using the OAuth2 authorization_code grant type.

This command requires a client ID, client secret, and authorization code that you
obtain from the UAA authorization endpoint. The authorization code is typically
obtained by directing users to the UAA authorization URL in a browser.`,
		Example: `  # Get token with authorization code
  capi uaa get-authcode-token \
    --client-id my-web-app \
    --client-secret app-secret \
    --code AUTHORIZATION_CODE \
    --redirect-uri https://myapp.com/callback`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := loadConfig()

			if GetEffectiveUAAEndpoint(config) == "" {
				return constants.ErrNoUAAConfigured
			}

			// Prompt for required fields if not provided
			err := promptForAuthcodeInputs(&clientID, &clientSecret, &authCode, &redirectURI)
			if err != nil {
				return err
			}

			// Get and store the token
			return getAuthcodeToken(config, clientID, clientSecret, authCode, redirectURI, tokenFormat)
		},
	}

	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth client ID")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth client secret")
	cmd.Flags().StringVar(&redirectURI, "redirect-uri", "", "OAuth redirect URI")
	cmd.Flags().StringVar(&authCode, "auth-code", "", "Authorization code from UAA")
	cmd.Flags().IntVar(&tokenFormat, "token-format", 0, "Token format (0=opaque, 1=JWT)")

	return cmd
}

// createUsersGetClientCredentialsTokenCommand creates the client credentials token command.
func createUsersGetClientCredentialsTokenCommand() *cobra.Command {
	var (
		clientID, clientSecret string
		tokenFormat            int
	)

	cmd := &cobra.Command{
		Use:     "get-client-credentials-token",
		Aliases: []string{"client-creds", "client-credentials", "auth"},
		Short:   "Obtain access token using client credentials grant",
		Long: `Obtain an access token using the OAuth2 client_credentials grant type.

This grant type is used for machine-to-machine authentication where no user
interaction is required. The client authenticates using its own credentials.`,
		Example: `  # Authenticate with client credentials
  capi uaa get-client-credentials-token \
    --client-id admin \
    --client-secret admin-secret

  # Use environment variables
  export UAA_CLIENT_ID=admin
  export UAA_CLIENT_SECRET=admin-secret
  capi uaa get-client-credentials-token`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGetClientCredentialsToken(clientID, clientSecret, tokenFormat)
		},
	}

	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth client ID")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth client secret")
	cmd.Flags().IntVar(&tokenFormat, "token-format", 0, "Token format (0=opaque, 1=JWT)")

	return cmd
}

func runGetClientCredentialsToken(clientID, clientSecret string, tokenFormat int) error {
	config := loadConfig()

	if GetEffectiveUAAEndpoint(config) == "" {
		return constants.ErrNoUAAConfigured
	}

	// Prompt for required fields if not provided
	if clientID == "" {
		log.Print("Client ID: ")

		_, _ = fmt.Scanln(&clientID)
	}

	if clientSecret == "" {
		log.Print("Client Secret: ")

		secretBytes, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read client secret: %w", err)
		}

		clientSecret = string(secretBytes)
		_, _ = os.Stdout.WriteString("\n") // Add newline after password input
	}

	// Create UAA client with client credentials authentication
	authOpt := uaa.WithClientCredentials(clientID, clientSecret, uaa.TokenFormat(tokenFormat))

	client, err := uaa.New(config.UAAEndpoint, authOpt)
	if err != nil {
		return fmt.Errorf("failed to create UAA client: %w", err)
	}

	// Get token
	ctx := context.Background()

	token, err := client.Token(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client credentials token: %w", err)
	}

	// Store tokens in config
	config.UAAToken = token.AccessToken
	if token.RefreshToken != "" {
		config.UAARefreshToken = token.RefreshToken
	}

	config.UAAClientID = clientID

	// Save configuration
	err = saveConfigStruct(config)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Warning: Failed to save token to configuration: %v\n", err)
	}

	// Display token information
	return displayTokenInfo(token, "Client Credentials Grant")
}

// createUsersGetPasswordTokenCommand creates the password token command.
func createUsersGetPasswordTokenCommand() *cobra.Command {
	var (
		clientID, clientSecret, username, password string
		tokenFormat                                int
	)

	cmd := &cobra.Command{
		Use:   "get-password-token",
		Short: "Obtain access token using password grant",
		Long: `Obtain an access token using the OAuth2 password grant type.

This grant type allows exchanging a user's username and password for an access token.
Note: This grant type should only be used by trusted clients as it requires
handling user credentials directly.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGetPasswordToken(clientID, clientSecret, username, password, tokenFormat)
		},
	}

	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth client ID")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth client secret")
	cmd.Flags().StringVar(&username, "username", "", "Username for authentication")
	cmd.Flags().StringVar(&password, "password", "", "Password for authentication")
	cmd.Flags().IntVar(&tokenFormat, "token-format", 0, "Token format (0=opaque, 1=JWT)")

	return cmd
}

func runGetPasswordToken(clientID, clientSecret, username, password string, tokenFormat int) error {
	config := loadConfig()

	if GetEffectiveUAAEndpoint(config) == "" {
		return constants.ErrNoUAAConfigured
	}

	// Prompt for required fields if not provided
	if clientID == "" {
		log.Print("Client ID: ")

		_, _ = fmt.Scanln(&clientID)
	}

	if clientSecret == "" {
		log.Print("Client Secret: ")

		secretBytes, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read client secret: %w", err)
		}

		clientSecret = string(secretBytes)
		_, _ = os.Stdout.WriteString("\n") // Add newline after password input
	}

	if username == "" {
		log.Print("Username: ")

		_, _ = fmt.Scanln(&username)
	}

	if password == "" {
		log.Print("Password: ")

		passwordBytes, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}

		password = string(passwordBytes)
		_, _ = os.Stdout.WriteString("\n") // Add newline after password input
	}

	// Create UAA client with password credentials authentication
	authOpt := uaa.WithPasswordCredentials(clientID, clientSecret, username, password, uaa.TokenFormat(tokenFormat))

	client, err := uaa.New(config.UAAEndpoint, authOpt)
	if err != nil {
		return fmt.Errorf("failed to create UAA client: %w", err)
	}

	// Get token
	ctx := context.Background()

	token, err := client.Token(ctx)
	if err != nil {
		return fmt.Errorf("failed to get password token: %w", err)
	}

	// Store tokens in config
	config.UAAToken = token.AccessToken
	if token.RefreshToken != "" {
		config.UAARefreshToken = token.RefreshToken
	}

	config.UAAClientID = clientID
	config.Username = username

	// Save configuration
	err = saveConfigStruct(config)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Warning: Failed to save token to configuration: %v\n", err)
	}

	// Display token information
	return displayTokenInfo(token, "Password Grant")
}

// createUsersRefreshTokenCommand creates the refresh token command.
func createUsersRefreshTokenCommand() *cobra.Command {
	var (
		clientID, clientSecret, refreshToken string
		tokenFormat                          int
	)

	cmd := &cobra.Command{
		Use:   "refresh-token",
		Short: "Refresh access token",
		Long: `Obtain a new access token using the refresh_token grant type.

This command uses a previously obtained refresh token to get a new access token.
If no refresh token is provided, it will attempt to use the one stored in the
current configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRefreshToken(clientID, clientSecret, refreshToken, tokenFormat)
		},
	}

	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth client ID")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "OAuth client secret")
	cmd.Flags().StringVar(&refreshToken, "refresh-token", "", "Refresh token")
	cmd.Flags().IntVar(&tokenFormat, "token-format", 0, "Token format (0=opaque, 1=JWT)")

	return cmd
}

func runRefreshToken(clientID, clientSecret, refreshToken string, tokenFormat int) error {
	config := loadConfig()

	if GetEffectiveUAAEndpoint(config) == "" {
		return constants.ErrNoUAAConfigured
	}

	// Use stored values if not provided via flags
	if clientID == "" {
		clientID = config.UAAClientID
	}

	if refreshToken == "" {
		refreshToken = config.UAARefreshToken
	}

	// Prompt for required fields if not available
	if clientID == "" {
		log.Print("Client ID: ")

		_, _ = fmt.Scanln(&clientID)
	}

	if clientSecret == "" {
		log.Print("Client Secret: ")

		secretBytes, err := term.ReadPassword(syscall.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read client secret: %w", err)
		}

		clientSecret = string(secretBytes)
		_, _ = os.Stdout.WriteString("\n") // Add newline after password input
	}

	if refreshToken == "" {
		return constants.ErrNoRefreshToken
	}

	// Create UAA client with refresh token authentication
	authOpt := uaa.WithRefreshToken(clientID, clientSecret, refreshToken, uaa.TokenFormat(tokenFormat))

	client, err := uaa.New(config.UAAEndpoint, authOpt)
	if err != nil {
		return fmt.Errorf("failed to create UAA client: %w", err)
	}

	// Get new token
	ctx := context.Background()

	token, err := client.Token(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Store new tokens in config
	config.UAAToken = token.AccessToken
	if token.RefreshToken != "" {
		config.UAARefreshToken = token.RefreshToken
	}

	config.UAAClientID = clientID

	// Save configuration
	err = saveConfigStruct(config)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Warning: Failed to save token to configuration: %v\n", err)
	}

	// Display token information
	return displayTokenInfo(token, "Refresh Token Grant")
}

// createUsersGetTokenKeyCommand creates the get token key command.
// tokenKeyCommandConfig defines the parameters for creating token key commands.
type tokenKeyCommandConfig struct {
	use          string
	short        string
	long         string
	fetchKeys    func(*UAAClientWrapper) (any, error)
	displayTable func(any) error
}

// createTokenKeyCommand creates a standardized command for token key retrieval.
func createTokenKeyCommand(config tokenKeyCommandConfig) *cobra.Command {
	return &cobra.Command{
		Use:   config.use,
		Short: config.short,
		Long:  config.long,
		RunE: func(cmd *cobra.Command, args []string) error {
			uaaConfig := loadConfig()

			if GetEffectiveUAAEndpoint(uaaConfig) == "" {
				return constants.ErrNoUAAConfigured
			}

			// Create UAA client
			uaaClient, err := NewUAAClient(uaaConfig)
			if err != nil {
				return fmt.Errorf("failed to create UAA client: %w", err)
			}

			// Fetch keys
			keys, err := config.fetchKeys(uaaClient)
			if err != nil {
				return err
			}

			// Display keys based on output format
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				err := encoder.Encode(keys)
				if err != nil {
					return fmt.Errorf("encoding keys to JSON: %w", err)
				}

				return nil
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				err := encoder.Encode(keys)
				if err != nil {
					return fmt.Errorf("encoding keys to YAML: %w", err)
				}

				return nil
			default:
				return config.displayTable(keys)
			}
		},
	}
}

func createUsersGetTokenKeyCommand() *cobra.Command {
	return createTokenKeyCommand(tokenKeyCommandConfig{
		use:   "get-token-key",
		short: "View JWT signing key",
		long:  "View the current key used for validating UAA's JWT token signatures",
		fetchKeys: func(uaaClient *UAAClientWrapper) (any, error) {
			key, err := uaaClient.Client().TokenKey()
			if err != nil {
				return nil, fmt.Errorf("failed to get token key: %w", err)
			}

			return key, nil
		},
		displayTable: func(keys any) error {
			key, ok := keys.(*uaa.JWK)
			if !ok {
				return fmt.Errorf("%w, got %T", constants.ErrExpectedJWKPointer, keys)
			}

			return displayTokenKeyTable(key)
		},
	})
}

// createUsersGetTokenKeysCommand creates the get token keys command.
func createUsersGetTokenKeysCommand() *cobra.Command {
	return createTokenKeyCommand(tokenKeyCommandConfig{
		use:   "get-token-keys",
		short: "View all JWT signing keys",
		long:  "View all keys the UAA has used to sign JWT tokens",
		fetchKeys: func(uaaClient *UAAClientWrapper) (any, error) {
			keys, err := uaaClient.Client().TokenKeys()
			if err != nil {
				return nil, fmt.Errorf("failed to get token keys: %w", err)
			}

			return keys, nil
		},
		displayTable: func(keys any) error {
			keySlice, ok := keys.([]uaa.JWK)
			if !ok {
				return fmt.Errorf("%w, got %T", constants.ErrExpectedJWKSlice, keys)
			}

			return displayTokenKeysTable(keySlice)
		},
	})
}

// Helper functions for token display

func displayTokenInfo(token *oauth2.Token, grantType string) error {
	output := viper.GetString("output")

	tokenInfo := map[string]any{
		"grant_type":    grantType,
		"access_token":  token.AccessToken,
		"token_type":    token.TokenType,
		"refresh_token": token.RefreshToken,
	}

	if !token.Expiry.IsZero() {
		tokenInfo["expires_at"] = token.Expiry.Format(time.RFC3339)
		tokenInfo["expires_in"] = int(time.Until(token.Expiry).Seconds())
	}

	// Add extra data if available
	if token.Extra("scope") != nil {
		tokenInfo["scope"] = token.Extra("scope")
	}

	if token.Extra("jti") != nil {
		tokenInfo["jti"] = token.Extra("jti")
	}

	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(tokenInfo)
		if err != nil {
			return fmt.Errorf("encoding token info to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(tokenInfo)
		if err != nil {
			return fmt.Errorf("encoding token info to YAML: %w", err)
		}

		return nil
	default:
		return displayTokenTable(token, grantType)
	}
}

func displayTokenTable(token *oauth2.Token, grantType string) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	_ = table.Append("Grant Type", grantType)
	_ = table.Append("Token Type", token.TokenType)

	if !token.Expiry.IsZero() {
		_ = table.Append("Expires At", token.Expiry.Format(time.RFC3339))
		_ = table.Append("Expires In", fmt.Sprintf("%d seconds", int(time.Until(token.Expiry).Seconds())))
	}

	if scope := token.Extra("scope"); scope != nil {
		_ = table.Append("Scope", fmt.Sprintf("%v", scope))
	}

	if jti := token.Extra("jti"); jti != nil {
		_ = table.Append("JTI", fmt.Sprintf("%v", jti))
	}

	_ = table.Render()

	_, _ = os.Stdout.WriteString("Token stored in configuration\n")

	return nil
}

func displayTokenKeyTable(key *uaa.JWK) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	if key.Kty != "" {
		_ = table.Append("Key Type", key.Kty)
	}

	if key.Kid != "" {
		_ = table.Append("Key ID", key.Kid)
	}

	if key.Alg != "" {
		_ = table.Append("Algorithm", key.Alg)
	}

	if key.Use != "" {
		_ = table.Append("Use", key.Use)
	}

	if key.Value != "" {
		_ = table.Append("Value", key.Value)
	}

	if key.E != "" {
		_ = table.Append("Exponent", key.E)
	}

	if key.N != "" {
		_ = table.Append("Modulus", key.N)
	}

	_ = table.Render()

	return nil
}

func displayTokenKeysTable(keys []uaa.JWK) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Key ID", "Algorithm", "Type", "Use")

	for _, key := range keys {
		_ = table.Append(
			getStringValue(key.Kid),
			getStringValue(key.Alg),
			getStringValue(key.Kty),
			getStringValue(key.Use),
		)
	}

	_ = table.Render()

	return nil
}

func getStringValue(value string) string {
	if value == "" {
		return "(not set)"
	}

	return value
}
