package commands

import (
	"github.com/spf13/cobra"
)

// NewUAAGroupCommand creates the group sub-command group.
func NewUAAGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   GroupCommandName,
		Short: "Manage UAA groups",
		Long: `Manage UAA groups and group memberships.

This command group provides comprehensive group management capabilities including:
- Creating new groups
- Retrieving group information
- Listing groups
- Managing group membership
- Group mappings for external identity providers`,
		Example: `  # Create a new group
  capi uaa group create developers --description "Development team"

  # Get group information
  capi uaa group get developers

  # List all groups
  capi uaa group list

  # Add user to group
  capi uaa group add-member developers john.doe

  # Remove user from group
  capi uaa group remove-member developers john.doe`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Add group management commands with new naming
	cmd.AddCommand(createGroupCreateCommand())
	cmd.AddCommand(createGroupGetCommand())
	cmd.AddCommand(createGroupListCommand())
	cmd.AddCommand(createGroupAddMemberCommand())
	cmd.AddCommand(createGroupRemoveMemberCommand())
	cmd.AddCommand(createGroupMapCommand())
	cmd.AddCommand(createGroupUnmapCommand())
	cmd.AddCommand(createGroupListMappingsCommand())

	return cmd
}

func createGroupCreateCommand() *cobra.Command {
	cmd := createUsersCreateGroupCommand()
	cmd.Use = "create <name>"

	return cmd
}

func createGroupGetCommand() *cobra.Command {
	cmd := createUsersGetGroupCommand()
	cmd.Use = "get <name>"

	return cmd
}

func createGroupListCommand() *cobra.Command {
	cmd := createUsersListGroupsCommand()
	cmd.Use = List

	return cmd
}

func createGroupAddMemberCommand() *cobra.Command {
	cmd := createUsersAddMemberCommand()
	cmd.Use = "add-member <group> <member>"

	return cmd
}

func createGroupRemoveMemberCommand() *cobra.Command {
	cmd := createUsersRemoveMemberCommand()
	cmd.Use = "remove-member <group> <member>"

	return cmd
}

func createGroupMapCommand() *cobra.Command {
	cmd := createUsersMapGroupCommand()
	cmd.Use = "map"

	return cmd
}

func createGroupUnmapCommand() *cobra.Command {
	cmd := createUsersUnmapGroupCommand()
	cmd.Use = "unmap"

	return cmd
}

func createGroupListMappingsCommand() *cobra.Command {
	cmd := createUsersListGroupMappingsCommand()
	cmd.Use = "list-mappings"

	return cmd
}
