//nolint:testpackage // RunE behavior tests need the unexported newClientFunc seam and command constructors
package commands

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errClientBoom is a sentinel used to assert that handler errors propagate
// unchanged from the injected client.
var errClientBoom = errors.New("boom from client")

// sidecarOriginUser is the CF API sidecar origin value for user-created sidecars.
const sidecarOriginUser = "user"

func TestSidecarsGet_JSONOutput(t *testing.T) { //nolint:paralleltest // serial: swaps process-global os.Stdout, viper, and newClientFunc
	withOutputFormat(t, OutputFormatJSON)

	sidecar := &capi.Sidecar{
		Resource:     capi.Resource{GUID: "sidecar-guid-1"},
		Name:         "my-sidecar",
		Command:      "bundle exec rackup",
		ProcessTypes: []string{"web", "worker"},
		Origin:       sidecarOriginUser,
	}
	recorder := &recordingSidecarsClient{getResult: sidecar}
	withStubClient(t, &fakeClient{sidecars: recorder})

	out, err := runCommand(t, newSidecarsGetCommand(), "sidecar-guid-1")
	require.NoError(t, err)

	// The handler must call Get with the positional argument verbatim.
	assert.Equal(t, "sidecar-guid-1", recorder.getGUID)

	// The JSON body must round-trip back to the same sidecar.
	var got capi.Sidecar

	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "my-sidecar", got.Name)
	assert.Equal(t, "sidecar-guid-1", got.GUID)
	assert.Equal(t, []string{"web", "worker"}, got.ProcessTypes)
}

func TestSidecarsGet_TableOutput(t *testing.T) { //nolint:paralleltest // serial: swaps process-global os.Stdout, viper, and newClientFunc
	withOutputFormat(t, "table")

	recorder := &recordingSidecarsClient{
		getResult: &capi.Sidecar{
			Resource: capi.Resource{GUID: "sidecar-guid-2"},
			Name:     "table-sidecar",
			Command:  "./run",
		},
	}
	withStubClient(t, &fakeClient{sidecars: recorder})

	out, err := runCommand(t, newSidecarsGetCommand(), "sidecar-guid-2")
	require.NoError(t, err)

	assert.Equal(t, "sidecar-guid-2", recorder.getGUID)
	// Table output is human-formatted; assert the key values are present.
	assert.Contains(t, out, "table-sidecar")
	assert.Contains(t, out, "sidecar-guid-2")
}

func TestSidecarsGet_ClientGetErrorPropagates(t *testing.T) { //nolint:paralleltest // serial: swaps process-global os.Stdout, viper, and newClientFunc
	withOutputFormat(t, OutputFormatJSON)

	recorder := &recordingSidecarsClient{getErr: errClientBoom}
	withStubClient(t, &fakeClient{sidecars: recorder})

	_, err := runCommand(t, newSidecarsGetCommand(), "sidecar-guid-3")

	require.Error(t, err)
	require.ErrorIs(t, err, errClientBoom)
	assert.Equal(t, "sidecar-guid-3", recorder.getGUID)
}

func TestSidecarsGet_ClientConstructionErrorPropagates(t *testing.T) { //nolint:paralleltest // serial: swaps process-global os.Stdout, viper, and newClientFunc
	withOutputFormat(t, OutputFormatJSON)
	withClientError(t, errClientBoom)

	_, err := runCommand(t, newSidecarsGetCommand(), "any-guid")

	require.Error(t, err)
	require.ErrorIs(t, err, errClientBoom)
}

func TestSidecarsDelete_ForceCallsDelete(t *testing.T) { //nolint:paralleltest // serial: swaps process-global os.Stdout, viper, and newClientFunc
	withOutputFormat(t, "table")

	recorder := &recordingSidecarsClient{
		getResult: &capi.Sidecar{
			Resource: capi.Resource{GUID: "sidecar-guid-4"},
			Name:     "doomed",
		},
	}
	withStubClient(t, &fakeClient{sidecars: recorder})

	out, err := runCommand(t, newSidecarsDeleteCommand(), "sidecar-guid-4", "--force")
	require.NoError(t, err)

	// With --force the handler skips the prompt, looks up the name, then deletes.
	assert.Equal(t, "sidecar-guid-4", recorder.getGUID)
	assert.Equal(t, "sidecar-guid-4", recorder.deleteGUID)
	assert.Contains(t, out, "Successfully deleted sidecar 'doomed'")
}

func TestSidecarsDelete_DeleteErrorPropagates(t *testing.T) { //nolint:paralleltest // serial: swaps process-global os.Stdout, viper, and newClientFunc
	withOutputFormat(t, "table")

	recorder := &recordingSidecarsClient{
		getResult: &capi.Sidecar{Name: "doomed"},
		deleteErr: errClientBoom,
	}
	withStubClient(t, &fakeClient{sidecars: recorder})

	_, err := runCommand(t, newSidecarsDeleteCommand(), "sidecar-guid-5", "--force")

	require.Error(t, err)
	require.ErrorIs(t, err, errClientBoom)
	assert.Equal(t, "sidecar-guid-5", recorder.deleteGUID)
}
