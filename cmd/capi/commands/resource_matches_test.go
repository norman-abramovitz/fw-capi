package commands_test

import (
	"testing"

	"github.com/fivetwenty-io/capi/v3/cmd/capi/commands"
	"github.com/stretchr/testify/assert"
)

func TestNewResourceMatchesCommand(t *testing.T) {
	t.Parallel()

	cmd := commands.NewResourceMatchesCommand()
	assert.Equal(t, "resource-matches", cmd.Use)
	assert.Equal(t, []string{"resources", "resource", "rm"}, cmd.Aliases)
	assert.Equal(t, "Manage resource matches", cmd.Short)
	assert.Equal(t, "Create resource matches for optimizing package uploads", cmd.Long)

	// Check subcommands are added
	subcommands := cmd.Commands()
	assert.Len(t, subcommands, 1)

	commandNames := make([]string, 0, len(subcommands))
	for _, subcmd := range subcommands {
		commandNames = append(commandNames, subcmd.Name())
	}

	assert.Contains(t, commandNames, commands.Create)
}

func TestResourceMatchesCreateCommand(t *testing.T) {
	t.Parallel()

	root := commands.NewResourceMatchesCommand()
	cmd := findSubcommand(root, commands.Create)
	assert.Equal(t, commands.Create, cmd.Use)
	assert.Equal(t, "Create resource matches", cmd.Short)
	assert.Equal(t, "Create resource matches to check which resources already exist on the platform", cmd.Long)
	assert.NotNil(t, cmd.RunE)
}
