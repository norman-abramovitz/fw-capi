package commands

import (
	"github.com/spf13/cobra"
)

// NewUAAUserCommand creates the user sub-command group.
func NewUAAUserCommand() *cobra.Command {
	config := CommandConfig{
		Use:   UserCommandName,
		Short: "Manage UAA users",
		Long: `Manage UAA users with CRUD operations.

This command group provides comprehensive user management capabilities including:
- Creating new users
- Retrieving user information
- Listing users with filtering
- Updating user attributes
- Activating/deactivating users
- Deleting users`,
		Example: `  # Create a new user
  capi uaa user create john.doe --email john@example.com

  # Get user information
  capi uaa user get john.doe

  # List all active users
  capi uaa user list --filter 'active eq true'

  # Update user email
  capi uaa user update john.doe --email newemail@example.com

  # Deactivate a user
  capi uaa user deactivate john.doe`,
		SubCommands: []SubCommandConfig{
			{Name: Create, CommandFunc: createUsersCreateUserCommand, Use: "create <username>"},
			{Name: "get", CommandFunc: createUsersGetUserCommand, Use: "get <username>"},
			{Name: List, CommandFunc: createUsersListUsersCommand, Use: List},
			{Name: "update", CommandFunc: createUsersUpdateUserCommand, Use: "update <username>"},
			{Name: "activate", CommandFunc: createUsersActivateUserCommand, Use: "activate <username>"},
			{Name: "deactivate", CommandFunc: createUsersDeactivateUserCommand, Use: "deactivate <username>"},
			{Name: "delete", CommandFunc: createUsersDeleteUserCommand, Use: "delete <username>"},
		},
	}

	return CreateUAASubCommandGroup(config)
}
