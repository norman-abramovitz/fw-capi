//nolint:testpackage // Need access to internal types
package commands

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCreateUsersCreateGroupCommand(t *testing.T) {
	t.Parallel()

	cmd := createUsersCreateGroupCommand()
	assert.Equal(t, "create-group <name>", cmd.Use)
	assert.Equal(t, "Create a group", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.Contains(t, cmd.Long, "Create a new group")

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("description"))
	assert.NotNil(t, cmd.Flags().Lookup("members"))
}

func TestCreateUsersGetGroupCommand(t *testing.T) {
	t.Parallel()

	cmd := createUsersGetGroupCommand()
	assert.Equal(t, "get-group <name>", cmd.Use)
	assert.Equal(t, "Get group details", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.Contains(t, cmd.Long, "Look up a group")
}

// testGenericListCommand tests generic list commands with common flags.
func testGenericListCommand(t *testing.T, cmd *cobra.Command, expectedUse, expectedShort, expectedLongContains string) {
	t.Helper()

	assert.Equal(t, expectedUse, cmd.Use)
	assert.Equal(t, expectedShort, cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.Contains(t, cmd.Long, expectedLongContains)

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("filter"))
	assert.NotNil(t, cmd.Flags().Lookup("sort-by"))
	assert.NotNil(t, cmd.Flags().Lookup("sort-order"))
	assert.NotNil(t, cmd.Flags().Lookup("attributes"))
	assert.NotNil(t, cmd.Flags().Lookup("count"))
	assert.NotNil(t, cmd.Flags().Lookup("start-index"))
	assert.NotNil(t, cmd.Flags().Lookup("all"))
}

func TestCreateUsersListGroupsCommand(t *testing.T) {
	t.Parallel()
	testGenericListCommand(t, createUsersListGroupsCommand(), "list-groups", "List groups", "Search and list groups")
}

func TestCreateUsersAddMemberCommand(t *testing.T) {
	t.Parallel()

	cmd := createUsersAddMemberCommand()
	assert.Equal(t, "add-member <group> <member>", cmd.Use)
	assert.Equal(t, "Add user to group", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.Contains(t, cmd.Long, "Add a user to a group")

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("origin"))
	assert.NotNil(t, cmd.Flags().Lookup("type"))
}

func TestCreateUsersRemoveMemberCommand(t *testing.T) {
	t.Parallel()

	cmd := createUsersRemoveMemberCommand()
	assert.Equal(t, "remove-member <group> <member>", cmd.Use)
	assert.Equal(t, "Remove user from group", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.Contains(t, cmd.Long, "Remove a user from a group")

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("origin"))
	assert.NotNil(t, cmd.Flags().Lookup("type"))
}

func TestCreateUsersMapGroupCommand(t *testing.T) {
	t.Parallel()

	cmd := createUsersMapGroupCommand()
	assert.Equal(t, OperationMapGroup, cmd.Use)
	assert.Equal(t, "Map external group to UAA group", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.Contains(t, cmd.Long, "Map an external group")

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup(flagGroup))
	assert.NotNil(t, cmd.Flags().Lookup("external-group"))
	assert.NotNil(t, cmd.Flags().Lookup("origin"))
}

func TestCreateUsersUnmapGroupCommand(t *testing.T) {
	t.Parallel()

	cmd := createUsersUnmapGroupCommand()
	assert.Equal(t, OperationUnmapGroup, cmd.Use)
	assert.Equal(t, "Unmap external group from UAA group", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.Contains(t, cmd.Long, "Remove a mapping")

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup(flagGroup))
	assert.NotNil(t, cmd.Flags().Lookup("external-group"))
	assert.NotNil(t, cmd.Flags().Lookup("origin"))
}

func TestCreateUsersListGroupMappingsCommand(t *testing.T) {
	t.Parallel()

	cmd := createUsersListGroupMappingsCommand()
	assert.Equal(t, "list-group-mappings", cmd.Use)
	assert.Equal(t, "List group mappings", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.Contains(t, cmd.Long, "List all mappings")

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("origin"))
	assert.NotNil(t, cmd.Flags().Lookup("count"))
	assert.NotNil(t, cmd.Flags().Lookup("start-index"))
}

func TestIsUUID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"550E8400-E29B-41D4-A716-446655440000", true},
		{"not-a-uuid", false},
		{"550e8400-e29b-41d4-a716", false},
		{"", false},
		{"550e8400-e29b-41d4-a716-44665544000g", false}, // invalid hex
		{"550e8400e29b41d4a716446655440000", false},     // no dashes
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			result := isUUID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Note: Display functions and UUID checking would require more complex setup to test properly
// as they depend on actual UAA API responses and internal data structures.
// These tests focus on command structure validation instead.
