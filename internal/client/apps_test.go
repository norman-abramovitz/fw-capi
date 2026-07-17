package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/fivetwenty-io/capi/v3/internal/client"
	internalhttp "github.com/fivetwenty-io/capi/v3/internal/http"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppsClient_Create(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var req capi.AppCreateRequest

		_ = json.NewDecoder(request.Body).Decode(&req)
		assert.Equal(t, testAppNameFixture, req.Name)
		assert.Equal(t, testSpaceGUID, req.Relationships.Space.Data.GUID)

		app := capi.App{
			Resource: capi.Resource{
				GUID:      testAppGUID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name:  req.Name,
			State: testStateStopped,
			Lifecycle: capi.Lifecycle{
				Type: testBuildpackLifecycle,
				Data: map[string]any{},
			},
			Relationships: req.Relationships,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(writer).Encode(app)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	app, err := client.Apps().Create(context.Background(), &capi.AppCreateRequest{
		Name: testAppNameFixture,
		Relationships: capi.AppRelationships{
			Space: capi.Relationship{
				Data: &capi.RelationshipData{GUID: testSpaceGUID},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, testAppGUID, app.GUID)
	assert.Equal(t, testAppNameFixture, app.Name)
	assert.Equal(t, testStateStopped, app.State)
}

func TestAppsClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps/app-guid", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		app := capi.App{
			Resource: capi.Resource{
				GUID: testAppGUID,
			},
			Name:  testAppNameFixture,
			State: testStateStarted,
		}

		_ = json.NewEncoder(writer).Encode(app)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	app, err := client.Apps().Get(context.Background(), testAppGUID)
	require.NoError(t, err)
	assert.Equal(t, testAppGUID, app.GUID)
	assert.Equal(t, testAppNameFixture, app.Name)
	assert.Equal(t, testStateStarted, app.State)
}

func TestAppsClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "1", request.URL.Query().Get("page"))
		assert.Equal(t, "10", request.URL.Query().Get("per_page"))

		response := capi.ListResponse[capi.App]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
			},
			Resources: []capi.App{
				{
					Resource: capi.Resource{GUID: testAppGUID1},
					Name:     testAppGUID1,
					State:    testStateStarted,
				},
				{
					Resource: capi.Resource{GUID: testAppGUID2},
					Name:     testAppGUID2,
					State:    testStateStopped,
				},
			},
		}

		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	params := capi.NewQueryParams().WithPage(1).WithPerPage(10)
	result, err := client.Apps().List(context.Background(), params)

	require.NoError(t, err)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, testAppGUID1, result.Resources[0].Name)
	assert.Equal(t, testAppGUID2, result.Resources[1].Name)
}

func TestAppsClient_Update(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps/app-guid", request.URL.Path)
		assert.Equal(t, http.MethodPatch, request.Method)

		var req capi.AppUpdateRequest

		_ = json.NewDecoder(request.Body).Decode(&req)
		assert.Equal(t, "updated-app", *req.Name)

		app := capi.App{
			Resource: capi.Resource{GUID: testAppGUID},
			Name:     *req.Name,
			State:    testStateStopped,
		}

		_ = json.NewEncoder(writer).Encode(app)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	newName := "updated-app"
	app, err := client.Apps().Update(context.Background(), testAppGUID, &capi.AppUpdateRequest{
		Name: &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "updated-app", app.Name)
}

func TestAppsClient_Delete(t *testing.T) {
	t.Parallel()

	// CF v3 DELETE /v3/apps/{guid} is async: 202 Accepted, empty body,
	// Location header pointing at /v3/jobs/{jobGuid}.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps/app-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.Header().Set("Location", testJobPath)
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	job, err := client.Apps().Delete(context.Background(), testAppGUID)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
}

func TestAppsClient_DeleteMissingLocation(t *testing.T) {
	t.Parallel()

	// Server that responds 202 without the Location header — the capi
	// impl should treat this as a protocol violation rather than returning
	// a Job with an empty GUID the caller could try to poll.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	job, err := client.Apps().Delete(context.Background(), testAppGUID)
	require.Error(t, err)
	assert.Nil(t, job)
}

func TestAppsClient_ActionMethods(t *testing.T) {
	t.Parallel()

	tests := []TestAppActionOperation{
		{
			Name:          "Start",
			Action:        "start",
			ExpectedState: testStateStarted,
			ActionFunc: func(c *Client) func(context.Context, string) (*capi.Job, error) {
				return c.Apps().Start
			},
		},
		{
			Name:          "Stop",
			Action:        "stop",
			ExpectedState: testStateStopped,
			ActionFunc: func(c *Client) func(context.Context, string) (*capi.Job, error) {
				return c.Apps().Stop
			},
		},
		{
			Name:          "Restart",
			Action:        "restart",
			ExpectedState: testStateStarted,
			ActionFunc: func(c *Client) func(context.Context, string) (*capi.Job, error) {
				return c.Apps().Restart
			},
		},
	}

	RunAppActionTests(t, tests)
}

func TestAppsClient_GetEnv(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps/app-guid/env", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		env := capi.AppEnvironment{
			StagingEnvJSON: map[string]any{
				"STAGING_VAR": "staging_value",
			},
			RunningEnvJSON: map[string]any{
				"RUNNING_VAR": "running_value",
			},
			EnvironmentVariables: map[string]any{
				"USER_VAR": "user_value",
			},
			SystemEnvJSON: map[string]any{
				"VCAP_SERVICES": map[string]any{},
			},
			ApplicationEnvJSON: map[string]any{
				"VCAP_APPLICATION": map[string]any{
					"name": testAppNameFixture,
				},
			},
		}

		_ = json.NewEncoder(writer).Encode(env)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	env, err := client.Apps().GetEnv(context.Background(), testAppGUID)
	require.NoError(t, err)
	assert.Equal(t, "user_value", env.EnvironmentVariables["USER_VAR"])
	assert.Equal(t, "staging_value", env.StagingEnvJSON["STAGING_VAR"])
}

func TestAppsClient_GetEnvVars(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps/app-guid/environment_variables", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		response := map[string]any{
			testVarEnvKey: map[string]any{
				"KEY1": testValue1,
				"KEY2": "value2",
			},
		}

		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	vars, err := client.Apps().GetEnvVars(context.Background(), testAppGUID)
	require.NoError(t, err)
	assert.Equal(t, testValue1, vars["KEY1"])
	assert.Equal(t, "value2", vars["KEY2"])
}

func TestAppsClient_UpdateEnvVars(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps/app-guid/environment_variables", request.URL.Path)
		assert.Equal(t, http.MethodPatch, request.Method)

		var req map[string]any

		_ = json.NewDecoder(request.Body).Decode(&req)
		if varMap, ok := req[testVarEnvKey].(map[string]any); ok {
			assert.Equal(t, "new_value", varMap["NEW_KEY"])
		} else {
			t.Errorf("req[\"var\"] is not a map[string]any")
		}

		response := map[string]any{
			testVarEnvKey: map[string]any{
				"NEW_KEY": "new_value",
				"KEY1":    testValue1,
			},
		}

		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	vars, err := client.Apps().UpdateEnvVars(context.Background(), testAppGUID, map[string]any{
		"NEW_KEY": "new_value",
	})
	require.NoError(t, err)
	assert.Equal(t, "new_value", vars["NEW_KEY"])
}

func TestAppsClient_GetCurrentDroplet(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps/app-guid/droplets/current", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		droplet := capi.Droplet{
			Resource: capi.Resource{GUID: testDropletGUID},
		}

		_ = json.NewEncoder(writer).Encode(droplet)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	droplet, err := client.Apps().GetCurrentDroplet(context.Background(), testAppGUID)
	require.NoError(t, err)
	assert.Equal(t, testDropletGUID, droplet.GUID)
}

func TestAppsClient_SetCurrentDroplet(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps/app-guid/relationships/current_droplet", request.URL.Path)
		assert.Equal(t, http.MethodPatch, request.Method)

		var req capi.Relationship

		_ = json.NewDecoder(request.Body).Decode(&req)
		assert.Equal(t, testDropletGUID, req.Data.GUID)

		_ = json.NewEncoder(writer).Encode(req)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	rel, err := client.Apps().SetCurrentDroplet(context.Background(), testAppGUID, testDropletGUID)
	require.NoError(t, err)
	assert.Equal(t, testDropletGUID, rel.Data.GUID)
}

func TestAppsClient_GetSSHEnabled(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps/app-guid/ssh_enabled", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		sshEnabled := capi.AppSSHEnabled{
			Enabled: true,
			Reason:  "",
		}

		_ = json.NewEncoder(writer).Encode(sshEnabled)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	ssh, err := client.Apps().GetSSHEnabled(context.Background(), testAppGUID)
	require.NoError(t, err)
	assert.True(t, ssh.Enabled)
}

func TestAppsClient_GetPermissions(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps/app-guid/permissions", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		permissions := capi.AppPermissions{
			ReadBasicData:     true,
			ReadSensitiveData: false,
		}

		_ = json.NewEncoder(writer).Encode(permissions)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	perms, err := client.Apps().GetPermissions(context.Background(), testAppGUID)
	require.NoError(t, err)
	assert.True(t, perms.ReadBasicData)
	assert.False(t, perms.ReadSensitiveData)
}

func TestAppsClient_ClearBuildpackCache(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps/app-guid/actions/clear_buildpack_cache", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	err = client.Apps().ClearBuildpackCache(context.Background(), testAppGUID)
	require.NoError(t, err)
}

func TestAppsClient_GetManifest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps/app-guid/manifest", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		manifest := `applications:
- name: test-app
  memory: 512M
  instances: 2
  buildpack: nodejs_buildpack`

		writer.Header().Set("Content-Type", "application/x-yaml")
		_, _ = writer.Write([]byte(manifest))
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	manifest, err := client.Apps().GetManifest(context.Background(), testAppGUID)
	require.NoError(t, err)
	assert.Contains(t, manifest, "name: test-app")
	assert.Contains(t, manifest, "memory: 512M")
}

func TestAppsClient_GetWithIncludes(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps/app-guid", request.URL.Path)
		assert.Equal(t, "space,space.organization", request.URL.Query().Get("include"))

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
		  "guid": "app-guid", "name": "web",
		  "included": {
		    "spaces": [{"guid": "space-1", "name": "dev"}],
		    "organizations": [{"guid": "org-1", "name": "acme"}]
		  }
		}`))
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	apps := NewAppsClient(httpClient)

	app, err := apps.Get(context.Background(), testAppGUID,
		capi.AppIncludeSpace, capi.AppIncludeSpaceOrganization)
	require.NoError(t, err)
	require.NotNil(t, app.Included)
	assert.Equal(t, testSpaceName1, app.Included.Spaces[0].GUID)
	assert.Equal(t, testOrgName1, app.Included.Organizations[0].GUID)
}
