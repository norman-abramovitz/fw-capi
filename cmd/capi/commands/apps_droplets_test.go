//nolint:testpackage // uses the RunE harness in rune_harness_test.go, which needs package commands
package commands

import (
	"testing"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAppsDropletsCommand_CurrentFlag verifies that `capi apps droplets
// --current` applies the current=true filter (CF API 3.229.0), matching
// capi.WithDropletCurrent()'s wire format, and that omitting the flag leaves
// the filter unset.
func TestAppsDropletsCommand_CurrentFlag(t *testing.T) { //nolint:paralleltest // serial: swaps process-global os.Stdout, viper, and newClientFunc
	stub := &recordingDropletsClient{
		listResult: &capi.ListResponse[capi.Droplet]{
			Pagination: capi.Pagination{TotalResults: 0, TotalPages: 1},
			Resources:  []capi.Droplet{},
		},
	}

	withStubClient(t, &fakeClient{droplets: stub})
	withOutputFormat(t, "table")

	_, err := runCommand(t, newAppsDropletsCommand(), "--current")
	require.NoError(t, err)
	require.NotNil(t, stub.listParams)
	assert.Equal(t, []string{"true"}, stub.listParams.Filters["current"])
}

// TestAppsDropletsCommand_NoCurrentFlag verifies that the current filter is
// absent when --current is not passed, so existing behavior for callers that
// list every droplet is unchanged.
func TestAppsDropletsCommand_NoCurrentFlag(t *testing.T) { //nolint:paralleltest // serial: swaps process-global os.Stdout, viper, and newClientFunc
	stub := &recordingDropletsClient{
		listResult: &capi.ListResponse[capi.Droplet]{
			Pagination: capi.Pagination{TotalResults: 0, TotalPages: 1},
			Resources:  []capi.Droplet{},
		},
	}

	withStubClient(t, &fakeClient{droplets: stub})
	withOutputFormat(t, "table")

	_, err := runCommand(t, newAppsDropletsCommand())
	require.NoError(t, err)
	require.NotNil(t, stub.listParams)
	assert.NotContains(t, stub.listParams.Filters, "current")
}
