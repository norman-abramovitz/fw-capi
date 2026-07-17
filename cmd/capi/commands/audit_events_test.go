package commands_test

import (
	"testing"

	"github.com/fivetwenty-io/capi/v3/cmd/capi/commands"
	"github.com/stretchr/testify/assert"
)

func TestNewAuditEventsCommand(t *testing.T) {
	t.Parallel()

	cmd := commands.NewAuditEventsCommand()
	assert.Equal(t, "audit-events", cmd.Use)
	assert.Equal(t, []string{"audit", "events", "ae"}, cmd.Aliases)
	assert.Equal(t, "Manage audit events", cmd.Short)
	assert.Equal(t, "View audit events for tracking system changes and user actions", cmd.Long)

	// Check subcommands are added
	subcommands := cmd.Commands()
	assert.Len(t, subcommands, 2)

	commandNames := make([]string, 0, len(subcommands))
	for _, subcmd := range subcommands {
		commandNames = append(commandNames, subcmd.Name())
	}

	assert.Contains(t, commandNames, commands.List)
	assert.Contains(t, commandNames, "get")
}

// Note: Tests for unexported functions (newAuditEventsListCommand, newAuditEventsGetCommand)
// are not included since they cannot be accessed from the commands_test package.
// These functions are tested indirectly through the main command.
