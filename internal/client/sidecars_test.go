package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/fivetwenty-io/capi/v3/internal/client"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSidecarsClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/sidecars/sidecar-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		memoryInMB := 128
		sidecar := capi.Sidecar{
			Resource: capi.Resource{
				GUID:      "sidecar-guid",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name:         "test-sidecar",
			Command:      "echo hello",
			ProcessTypes: []string{testWebProcessType, testWorkerProcessType},
			MemoryInMB:   &memoryInMB,
			Origin:       testUserUsername,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(sidecar)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	sidecar, err := client.Sidecars().Get(context.Background(), "sidecar-guid")
	require.NoError(t, err)
	assert.Equal(t, "sidecar-guid", sidecar.GUID)
	assert.Equal(t, "test-sidecar", sidecar.Name)
	assert.Equal(t, "echo hello", sidecar.Command)
	assert.Equal(t, []string{testWebProcessType, testWorkerProcessType}, sidecar.ProcessTypes)
	assert.Equal(t, 128, *sidecar.MemoryInMB)
}

func TestSidecarsClient_Update(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/sidecars/sidecar-guid", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var req capi.SidecarUpdateRequest

		_ = json.NewDecoder(request.Body).Decode(&req)
		assert.Equal(t, "updated-sidecar", *req.Name)
		assert.Equal(t, "echo updated", *req.Command)

		sidecar := capi.Sidecar{
			Resource: capi.Resource{GUID: "sidecar-guid"},
			Name:     *req.Name,
			Command:  *req.Command,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(sidecar)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	newName := "updated-sidecar"
	newCommand := "echo updated"
	sidecar, err := client.Sidecars().Update(context.Background(), "sidecar-guid", &capi.SidecarUpdateRequest{
		Name:    &newName,
		Command: &newCommand,
	})

	require.NoError(t, err)
	assert.Equal(t, "updated-sidecar", sidecar.Name)
	assert.Equal(t, "echo updated", sidecar.Command)
}

func TestSidecarsClient_Delete(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/sidecars/sidecar-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	err = client.Sidecars().Delete(context.Background(), "sidecar-guid")
	require.NoError(t, err)
}

func TestSidecarsClient_ListForProcess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/processes/process-guid/sidecars", request.URL.Path)
		assert.Equal(t, "GET", request.Method)
		assert.Equal(t, "1", request.URL.Query().Get("page"))
		assert.Equal(t, "10", request.URL.Query().Get("per_page"))

		memory1 := 64
		memory2 := 128
		response := capi.ListResponse[capi.Sidecar]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
			},
			Resources: []capi.Sidecar{
				{
					Resource:     capi.Resource{GUID: "sidecar-1"},
					Name:         "sidecar-1",
					Command:      "echo hello1",
					ProcessTypes: []string{testWebProcessType},
					MemoryInMB:   &memory1,
					Origin:       testUserUsername,
				},
				{
					Resource:     capi.Resource{GUID: "sidecar-2"},
					Name:         "sidecar-2",
					Command:      "echo hello2",
					ProcessTypes: []string{testWorkerProcessType},
					MemoryInMB:   &memory2,
					Origin:       testUserUsername,
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	params := capi.NewQueryParams().WithPage(1).WithPerPage(10)
	result, err := client.Sidecars().ListForProcess(context.Background(), testProcessGUID, params)

	require.NoError(t, err)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "sidecar-1", result.Resources[0].Name)
	assert.Equal(t, "sidecar-2", result.Resources[1].Name)
	assert.Equal(t, "echo hello1", result.Resources[0].Command)
	assert.Equal(t, "echo hello2", result.Resources[1].Command)
}
