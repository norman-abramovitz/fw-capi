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

func TestRevisionsClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/revisions/revision-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		description := "Test revision"
		revision := capi.Revision{
			Resource: capi.Resource{
				GUID:      testRevisionGUID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Version:     1,
			Deployable:  true,
			Description: &description,
			Droplet: capi.RevisionDropletRef{
				GUID: testDropletGUID,
			},
			Processes: map[string]capi.Process{
				testWebProcessType: {
					Resource: capi.Resource{
						GUID: testProcessGUID,
					},
					Type:       testWebProcessType,
					Instances:  2,
					MemoryInMB: 512,
					DiskInMB:   1024,
				},
			},
		}

		_ = json.NewEncoder(writer).Encode(revision)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	revision, err := client.Revisions().Get(context.Background(), testRevisionGUID)
	require.NoError(t, err)
	assert.Equal(t, testRevisionGUID, revision.GUID)
	assert.Equal(t, 1, revision.Version)
	assert.True(t, revision.Deployable)
	assert.Equal(t, "Test revision", *revision.Description)
	assert.Equal(t, testDropletGUID, revision.Droplet.GUID)
}

func TestRevisionsClient_Update(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/revisions/revision-guid", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var req capi.RevisionUpdateRequest

		_ = json.NewDecoder(request.Body).Decode(&req)
		assert.NotNil(t, req.Metadata)
		assert.Equal(t, testValue1, req.Metadata.Labels["key1"])

		revision := capi.Revision{
			Resource: capi.Resource{GUID: testRevisionGUID},
			Version:  1,
			Metadata: req.Metadata,
		}

		_ = json.NewEncoder(writer).Encode(revision)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	revision, err := client.Revisions().Update(context.Background(), testRevisionGUID, &capi.RevisionUpdateRequest{
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				"key1": testValue1,
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, testRevisionGUID, revision.GUID)
	assert.Equal(t, testValue1, revision.Metadata.Labels["key1"])
}

func TestRevisionsClient_GetEnvironmentVariables(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/revisions/revision-guid/environment_variables", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		envVars := map[string]any{
			"DATABASE_URL": "postgres://localhost/myapp",
			"API_KEY":      "secret-key",
			"DEBUG":        true,
		}

		response := map[string]any{
			testVarEnvKey: envVars,
		}

		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	envVars, err := client.Revisions().GetEnvironmentVariables(context.Background(), testRevisionGUID)
	require.NoError(t, err)
	assert.Equal(t, "postgres://localhost/myapp", envVars["DATABASE_URL"])
	assert.Equal(t, "secret-key", envVars["API_KEY"])
	assert.Equal(t, true, envVars["DEBUG"])
}

func TestRevisionsClient_ListForApp(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps/app-guid/revisions", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		description := "Test revision"
		response := capi.ListResponse[capi.Revision]{
			Resources: []capi.Revision{
				{
					Resource: capi.Resource{
						GUID:      "revision-guid-1",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Version:     1,
					Deployable:  true,
					Description: &description,
					Droplet: capi.RevisionDropletRef{
						GUID: "droplet-guid-1",
					},
					Processes: map[string]capi.Process{
						testWebProcessType: {
							Resource: capi.Resource{
								GUID: "process-guid-1",
							},
							Type:       testWebProcessType,
							Instances:  2,
							MemoryInMB: 512,
							DiskInMB:   1024,
						},
					},
				},
			},
		}

		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	result, err := client.Revisions().ListForApp(context.Background(), testAppGUID, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Resources, 1)
	assert.Equal(t, "revision-guid-1", result.Resources[0].GUID)
	assert.Equal(t, 1, result.Resources[0].Version)
	assert.True(t, result.Resources[0].Deployable)
}

func TestRevisionsClient_GetDeployedForApp(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps/app-guid/revisions/deployed", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		description := "Deployed revision"
		response := capi.ListResponse[capi.Revision]{
			Resources: []capi.Revision{
				{
					Resource: capi.Resource{
						GUID:      "deployed-revision-guid",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Version:     2,
					Deployable:  true,
					Description: &description,
					Droplet: capi.RevisionDropletRef{
						GUID: "deployed-droplet-guid",
					},
					Processes: map[string]capi.Process{
						testWebProcessType: {
							Resource: capi.Resource{
								GUID: "deployed-process-guid",
							},
							Type:       testWebProcessType,
							Instances:  3,
							MemoryInMB: 512,
							DiskInMB:   1024,
						},
					},
				},
			},
		}

		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	result, err := client.Revisions().GetDeployedForApp(context.Background(), testAppGUID)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Resources, 1)
	assert.Equal(t, "deployed-revision-guid", result.Resources[0].GUID)
	assert.Equal(t, 2, result.Resources[0].Version)
	assert.True(t, result.Resources[0].Deployable)
}
