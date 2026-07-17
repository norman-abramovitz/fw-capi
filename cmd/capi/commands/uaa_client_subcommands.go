package commands

import (
	"github.com/spf13/cobra"
)

// NewUAAClientCommand creates the client sub-command group.
func NewUAAClientCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   ClientCommandName,
		Short: "Manage UAA OAuth clients",
		Long: `Manage UAA OAuth clients with CRUD operations.

This command group provides comprehensive OAuth client management capabilities including:
- Creating new OAuth clients
- Retrieving client information
- Listing clients
- Updating client configurations
- Managing client secrets
- Deleting clients`,
		Example: `  # Create a new OAuth client
  capi uaa client create my-app --secret mysecret --authorized-grant-types authorization_code

  # Get client information
  capi uaa client get my-app

  # List all clients
  capi uaa client list

  # Update client configuration
  capi uaa client update my-app --redirect-uri https://myapp.com/callback

  # Update client secret
  capi uaa client set-secret my-app --secret newsecret`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Add client management commands with new naming
	cmd.AddCommand(createClientCreateCommand())
	cmd.AddCommand(createClientGetCommand())
	cmd.AddCommand(createClientListCommand())
	cmd.AddCommand(createClientUpdateCommand())
	cmd.AddCommand(createClientSetSecretCommand())
	cmd.AddCommand(createClientDeleteCommand())

	return cmd
}

func createClientCreateCommand() *cobra.Command {
	cmd := createUsersCreateClientCommand()
	cmd.Use = "create <client-id>"

	return cmd
}

func createClientGetCommand() *cobra.Command {
	cmd := createUsersGetClientCommand()
	cmd.Use = "get <client-id>"

	return cmd
}

func createClientListCommand() *cobra.Command {
	cmd := createUsersListClientsCommand()
	cmd.Use = List

	return cmd
}

func createClientUpdateCommand() *cobra.Command {
	cmd := createUsersUpdateClientCommand()
	cmd.Use = "update <client-id>"

	return cmd
}

func createClientSetSecretCommand() *cobra.Command {
	cmd := createUsersSetClientSecretCommand()
	cmd.Use = "set-secret <client-id>"

	return cmd
}

func createClientDeleteCommand() *cobra.Command {
	cmd := createUsersDeleteClientCommand()
	cmd.Use = "delete <client-id>"

	return cmd
}
