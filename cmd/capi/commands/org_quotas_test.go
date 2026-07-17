package commands_test

import (
	"testing"

	"github.com/fivetwenty-io/capi/v3/cmd/capi/commands"
	"github.com/stretchr/testify/assert"
)

func TestNewOrgQuotasCommand(t *testing.T) {
	t.Parallel()

	cmd := commands.NewOrgQuotasCommand()
	assert.Equal(t, "org-quotas", cmd.Use)
	assert.Equal(t, []string{"organization-quotas", "org-quota", "quotas"}, cmd.Aliases)
	assert.Equal(t, "Manage organization quotas", cmd.Short)
	assert.Equal(t, "List, create, update, delete, and apply organization quotas", cmd.Long)

	// Check subcommands are added
	subcommands := cmd.Commands()
	assert.Len(t, subcommands, 6)

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
}

func TestOrgQuotasListCommand(t *testing.T) {
	t.Parallel()

	root := commands.NewOrgQuotasCommand()
	cmd := findSubcommand(root, commands.List)
	assert.Equal(t, commands.List, cmd.Use)
	assert.Equal(t, "List organization quotas", cmd.Short)
	assert.Equal(t, "List all organization quotas", cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("all"))
	assert.NotNil(t, cmd.Flags().Lookup("per-page"))

	// Check flag defaults
	allFlag := cmd.Flags().Lookup("all")
	assert.Equal(t, "false", allFlag.DefValue)

	perPageFlag := cmd.Flags().Lookup("per-page")
	assert.Equal(t, "50", perPageFlag.DefValue)
}

func TestOrgQuotasGetCommand(t *testing.T) {
	t.Parallel()

	root := commands.NewOrgQuotasCommand()
	cmd := findSubcommand(root, "get")
	assert.Equal(t, "get QUOTA_NAME_OR_GUID", cmd.Use)
	assert.Equal(t, "Get organization quota details", cmd.Short)
	assert.Equal(t, "Display detailed information about a specific organization quota", cmd.Long)
	assert.NotNil(t, cmd.RunE)
	assert.NotNil(t, cmd.Args)
}

func TestOrgQuotasCreateCommand(t *testing.T) {
	t.Parallel()

	root := commands.NewOrgQuotasCommand()
	cmd := findSubcommand(root, commands.Create)
	assert.Equal(t, commands.Create, cmd.Use)
	assert.Equal(t, "Create a new organization quota", cmd.Short)
	assert.Equal(t, "Create a new Cloud Foundry organization quota", cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Check all required flags exist
	flags := []string{
		"name", "total-memory", "instance-memory", "instances", "app-tasks",
		"log-rate-limit", "paid-services", "service-instances", "service-keys",
		"routes", "reserved-ports", "domains",
	}

	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		assert.NotNil(t, flag, "Flag %s should exist", flagName)
	}

	// Check required flag
	nameFlag := cmd.Flags().Lookup("name")
	assert.NotNil(t, nameFlag)
}

func TestOrgQuotasUpdateCommand(t *testing.T) {
	t.Parallel()

	root := commands.NewOrgQuotasCommand()
	cmd := findSubcommand(root, "update")
	assert.Equal(t, "update QUOTA_NAME_OR_GUID", cmd.Use)
	assert.Equal(t, "Update an organization quota", cmd.Short)
	assert.Equal(t, "Update an existing Cloud Foundry organization quota", cmd.Long)
	assert.NotNil(t, cmd.RunE)
	assert.NotNil(t, cmd.Args)

	// Check flags exist (same as create, plus name for updating)
	flags := []string{
		"name", "total-memory", "instance-memory", "instances", "app-tasks",
		"log-rate-limit", "paid-services", "service-instances", "service-keys",
		"routes", "reserved-ports", "domains",
	}

	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		assert.NotNil(t, flag, "Flag %s should exist", flagName)
	}
}

func TestOrgQuotasDeleteCommand(t *testing.T) {
	t.Parallel()

	root := commands.NewOrgQuotasCommand()
	cmd := findSubcommand(root, "delete")
	assert.Equal(t, "delete QUOTA_NAME_OR_GUID", cmd.Use)
	assert.Equal(t, "Delete an organization quota", cmd.Short)
	assert.Equal(t, "Delete a Cloud Foundry organization quota", cmd.Long)
	assert.NotNil(t, cmd.RunE)
	assert.NotNil(t, cmd.Args)

	// Check force flag
	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag)
	assert.Equal(t, "f", forceFlag.Shorthand)
	assert.Equal(t, "false", forceFlag.DefValue)
}

func TestOrgQuotasApplyCommand(t *testing.T) {
	t.Parallel()

	root := commands.NewOrgQuotasCommand()
	cmd := findSubcommand(root, "apply")
	assert.Equal(t, "apply QUOTA_NAME_OR_GUID ORG_NAME_OR_GUID...", cmd.Use)
	assert.Equal(t, "Apply quota to organizations", cmd.Short)
	assert.Equal(t, "Apply an organization quota to one or more organizations", cmd.Long)
	assert.NotNil(t, cmd.RunE)

	assert.NotNil(t, cmd.Args)
}
