//nolint:testpackage // Need access to internal types
package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateUsersContextCommand(t *testing.T) {
	t.Parallel()

	cmd := createUsersContextCommand()
	assert.Equal(t, "context", cmd.Use)
	assert.Equal(t, "Display current UAA context", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.Contains(t, cmd.Long, "Show information about the currently active UAA context")
}

func TestCreateUsersTargetCommand(t *testing.T) {
	t.Parallel()

	cmd := createUsersTargetCommand()
	assert.Equal(t, "target <url>", cmd.Use)
	assert.Equal(t, "Set UAA endpoint URL", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.Contains(t, cmd.Long, "Set the URL of the UAA service")
}

func TestCreateUsersInfoCommand(t *testing.T) {
	t.Parallel()

	cmd := createUsersInfoCommand()
	assert.Equal(t, Info, cmd.Use)
	assert.Equal(t, "Display UAA server information", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.Contains(t, cmd.Long, "Show version and configuration information")
}

func TestCreateUsersVersionCommand(t *testing.T) {
	t.Parallel()

	cmd := createUsersVersionCommand()
	assert.Equal(t, Version, cmd.Use)
	assert.Equal(t, "Display UAA server version", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.Contains(t, cmd.Long, "Show the version of the targeted UAA server")
}

// Note: Display functions would require more complex setup to test properly
// as they depend on actual UAA server responses and internal state.
// These tests focus on command structure validation instead.
