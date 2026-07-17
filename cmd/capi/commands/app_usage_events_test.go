package commands_test

import (
	"testing"

	"github.com/fivetwenty-io/capi/v3/cmd/capi/commands"
	"github.com/stretchr/testify/assert"
)

func TestNewAppUsageEventsCommand(t *testing.T) {
	t.Parallel()

	cmd := commands.NewAppUsageEventsCommand()
	assert.Equal(t, "app-usage-events", cmd.Use)
	assert.Equal(t, []string{"app-usage", "app-events", "aue"}, cmd.Aliases)
	assert.Equal(t, "Manage application usage events", cmd.Short)
	assert.Equal(t, "View and manage application usage events for monitoring and billing", cmd.Long)

	// Check subcommands are added
	subcommands := cmd.Commands()
	assert.Len(t, subcommands, 3)

	commandNames := make([]string, 0, len(subcommands))
	for _, subcmd := range subcommands {
		commandNames = append(commandNames, subcmd.Name())
	}

	assert.Contains(t, commandNames, commands.List)
	assert.Contains(t, commandNames, "get")
	assert.Contains(t, commandNames, "purge-and-reseed")
}

// Note: Tests for unexported functions (newAppUsageEventsListCommand, etc.)
// are not included since they cannot be accessed from the commands_test package.
// These functions are tested indirectly through the main command.
