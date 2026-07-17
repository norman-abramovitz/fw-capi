package commands_test

import (
	"testing"

	"github.com/fivetwenty-io/capi/v3/cmd/capi/commands"
	"github.com/stretchr/testify/assert"
)

func TestNewSpaceQuotasCommand(t *testing.T) {
	t.Parallel()

	cmd := commands.NewSpaceQuotasCommand()
	assert.Equal(t, "space-quotas", cmd.Use)
	assert.Equal(t, []string{"space-quota", "sq"}, cmd.Aliases)
	assert.Equal(t, "Manage space quotas", cmd.Short)
	assert.Equal(t, "List, create, update, delete, apply, and remove space quotas", cmd.Long)

	// Check subcommands are added
	subcommands := cmd.Commands()
	assert.Len(t, subcommands, 7)

	commandNames := make([]string, 0, len(subcommands))
	for _, subcmd := range subcommands {
		commandNames = append(commandNames, subcmd.Name())
	}

	assert.Contains(t, commandNames, commands.List)
	assert.Contains(t, commandNames, "get")
	assert.Contains(t, commandNames, commands.Create)
	assert.Contains(t, commandNames, "update")
	assert.Contains(t, commandNames, "delete")
	assert.Contains(t, commandNames, "apply")
	assert.Contains(t, commandNames, "remove")
}

// Note: Tests for unexported functions (newSpaceQuotasListCommand, etc.)
// are not included since they cannot be accessed from the commands_test package.
// These functions are tested indirectly through the main command.
