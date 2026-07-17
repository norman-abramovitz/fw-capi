package commands

import (
	"github.com/spf13/cobra"
)

// NewUAATokenCommand creates the token sub-command group.
func NewUAATokenCommand() *cobra.Command {
	config := CommandConfig{
		Use:   Token,
		Short: "Manage UAA OAuth tokens",
		Long: `Manage UAA OAuth tokens and token operations.

This command group provides comprehensive token management capabilities including:
- Obtaining tokens via various OAuth2 flows
- Refreshing expired tokens
- Retrieving token validation keys
- Token introspection and validation`,
		Example: `  # Get token using client credentials flow
  capi uaa token get-client-credentials --client-id admin --client-secret secret

  # Get token using authorization code flow
  capi uaa token get-authcode --client-id myapp --client-secret secret

  # Get token using password flow
  capi uaa token get-password --client-id myapp --username john.doe

  # Refresh an existing token
  capi uaa token refresh --refresh-token <refresh-token>

  # Get token signing keys
  capi uaa token get-keys`,
		SubCommands: []SubCommandConfig{
			{Name: "authcode", CommandFunc: createUsersGetAuthcodeTokenCommand, Use: "get-authcode"},
			{Name: "client-credentials", CommandFunc: createUsersGetClientCredentialsTokenCommand, Use: "get-client-credentials"},
			{Name: "password", CommandFunc: createUsersGetPasswordTokenCommand, Use: "get-password"},
			{Name: "implicit", CommandFunc: createUsersGetImplicitTokenCommand, Use: "get-implicit"},
			{Name: Refresh, CommandFunc: createUsersRefreshTokenCommand, Use: Refresh},
			{Name: "key", CommandFunc: createUsersGetTokenKeyCommand, Use: "get-key"},
			{Name: "keys", CommandFunc: createUsersGetTokenKeysCommand, Use: "get-keys"},
		},
	}

	return CreateUAASubCommandGroup(config)
}
