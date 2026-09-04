package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/fivetwenty-io/capi/v3/internal/client"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
	internalhttp "github.com/fivetwenty-io/capi/v3/internal/http"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestServiceCredentialBindingsClient_Create_App(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_credential_bindings", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var requestBody capi.ServiceCredentialBindingCreateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.Equal(t, testAppKey, requestBody.Type)
		assert.Equal(t, testBindingName, *requestBody.Name)
		assert.Equal(t, testInstanceGUID, requestBody.Relationships.ServiceInstance.Data.GUID)
		assert.Equal(t, testAppGUID, requestBody.Relationships.App.Data.GUID)

		// App bindings may return a job for async operations
		job := capi.Job{
			Resource: capi.Resource{
				GUID: testJobGUID,
			},
			Operation: "service_credential_binding.create",
			State:     testStateProcessing,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Location", testJobPath)
		writer.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(writer).Encode(job)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceBindings := NewServiceCredentialBindingsClient(httpClient)

	name := testBindingName
	request := &capi.ServiceCredentialBindingCreateRequest{
		Type: testAppKey,
		Name: &name,
		Parameters: map[string]any{
			testFooKey: testBarValue,
		},
		Relationships: capi.ServiceCredentialBindingRelationships{
			ServiceInstance: capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: testInstanceGUID,
				},
			},
			App: &capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: testAppGUID,
				},
			},
		},
	}

	result, err := serviceBindings.Create(context.Background(), request)
	require.NoError(t, err)

	job, ok := result.(*capi.Job)
	require.True(t, ok, "Expected *capi.Job for app binding")
	assert.Equal(t, testJobGUID, job.GUID)
	assert.Equal(t, "service_credential_binding.create", job.Operation)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestServiceCredentialBindingsClient_Create_App_WithStrategy(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_credential_bindings", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var requestBody capi.ServiceCredentialBindingCreateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.Equal(t, testAppKey, requestBody.Type)
		assert.Equal(t, testBindingName, *requestBody.Name)
		assert.Equal(t, testInstanceGUID, requestBody.Relationships.ServiceInstance.Data.GUID)
		assert.Equal(t, testAppGUID, requestBody.Relationships.App.Data.GUID)
		assert.NotNil(t, requestBody.Strategy)

		if requestBody.Strategy != nil {
			assert.Equal(t, "multiple", *requestBody.Strategy)
		}

		job := capi.Job{
			Resource: capi.Resource{
				GUID: testJobGUID,
			},
			Operation: "service_credential_binding.create",
			State:     testStateProcessing,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Location", testJobPath)
		writer.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(writer).Encode(job)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceBindings := NewServiceCredentialBindingsClient(httpClient)

	strategy := "multiple"
	name := testBindingName
	request := &capi.ServiceCredentialBindingCreateRequest{
		Type:     testAppKey,
		Name:     &name,
		Strategy: &strategy,
		Relationships: capi.ServiceCredentialBindingRelationships{
			ServiceInstance: capi.Relationship{
				Data: &capi.RelationshipData{GUID: testInstanceGUID},
			},
			App: &capi.Relationship{
				Data: &capi.RelationshipData{GUID: testAppGUID},
			},
		},
	}

	result, err := serviceBindings.Create(context.Background(), request)
	require.NoError(t, err)

	job, ok := result.(*capi.Job)
	require.True(t, ok, "Expected *capi.Job for app binding with strategy")
	assert.Equal(t, testJobGUID, job.GUID)
	assert.Equal(t, "service_credential_binding.create", job.Operation)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestServiceCredentialBindingsClient_Create_Key(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_credential_bindings", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var requestBody capi.ServiceCredentialBindingCreateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.Equal(t, testKeyBindingType, requestBody.Type)
		assert.Equal(t, "my-key", *requestBody.Name)
		assert.Equal(t, testInstanceGUID, requestBody.Relationships.ServiceInstance.Data.GUID)
		assert.Nil(t, requestBody.Relationships.App)

		// Key bindings usually return the binding directly
		now := time.Now()
		binding := capi.ServiceCredentialBinding{
			Resource: capi.Resource{
				GUID:      testBindingGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name: "my-key",
			Type: testKeyBindingType,
			LastOperation: &capi.ServiceCredentialBindingLastOperation{
				Type:      testCreateOperation,
				State:     testSucceededOperation,
				CreatedAt: &now,
				UpdatedAt: &now,
			},
			Relationships: capi.ServiceCredentialBindingRelationships{
				ServiceInstance: capi.Relationship{
					Data: &capi.RelationshipData{
						GUID: testInstanceGUID,
					},
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(writer).Encode(binding)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceBindings := NewServiceCredentialBindingsClient(httpClient)

	name := "my-key"
	request := &capi.ServiceCredentialBindingCreateRequest{
		Type: testKeyBindingType,
		Name: &name,
		Relationships: capi.ServiceCredentialBindingRelationships{
			ServiceInstance: capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: testInstanceGUID,
				},
			},
		},
	}

	result, err := serviceBindings.Create(context.Background(), request)
	require.NoError(t, err)

	binding, ok := result.(*capi.ServiceCredentialBinding)
	require.True(t, ok, "Expected *capi.ServiceCredentialBinding for key binding")
	assert.Equal(t, testBindingGUID, binding.GUID)
	assert.Equal(t, "my-key", binding.Name)
	assert.Equal(t, testKeyBindingType, binding.Type)
}

func TestServiceCredentialBindingsClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_credential_bindings/binding-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		now := time.Now()
		binding := capi.ServiceCredentialBinding{
			Resource: capi.Resource{
				GUID:      testBindingGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name: testBindingName,
			Type: testAppKey,
			LastOperation: &capi.ServiceCredentialBindingLastOperation{
				Type:      testCreateOperation,
				State:     testSucceededOperation,
				CreatedAt: &now,
				UpdatedAt: &now,
			},
			Relationships: capi.ServiceCredentialBindingRelationships{
				ServiceInstance: capi.Relationship{
					Data: &capi.RelationshipData{
						GUID: testInstanceGUID,
					},
				},
				App: &capi.Relationship{
					Data: &capi.RelationshipData{
						GUID: testAppGUID,
					},
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(binding)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceBindings := NewServiceCredentialBindingsClient(httpClient)

	binding, err := serviceBindings.Get(context.Background(), testBindingGUID)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, testBindingGUID, binding.GUID)
	assert.Equal(t, testBindingName, binding.Name)
	assert.Equal(t, testAppKey, binding.Type)
}

//nolint:dupl // Acceptable duplication - each test validates different endpoints with different query params and assertions
func TestServiceCredentialBindingsClient_List(t *testing.T) {
	t.Parallel()

	now := time.Now()
	responseData := []capi.ServiceCredentialBinding{
		{
			Resource: capi.Resource{
				GUID:      "binding-guid-1",
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name: "binding-1",
			Type: testAppKey,
		},
		{
			Resource: capi.Resource{
				GUID:      "binding-guid-2",
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name: "binding-2",
			Type: testKeyBindingType,
		},
	}

	RunServiceListTest(t, "service credential bindings list", "/v3/service_credential_bindings",
		func(request *http.Request) {
			assert.Equal(t, testInstanceGUID, request.URL.Query().Get("service_instance_guids"))
			assert.Equal(t, testAppGUID, request.URL.Query().Get(testAppGUIDsParam))
		},
		responseData,
		func(httpClient *internalhttp.Client) any {
			return NewServiceCredentialBindingsClient(httpClient)
		},
		func(client any) (*capi.ListResponse[capi.ServiceCredentialBinding], error) {
			params := &capi.QueryParams{
				Filters: map[string][]string{
					"service_instance_guids": {testInstanceGUID},
					testAppGUIDsParam:        {testAppGUID},
				},
			}

			if serviceClient, ok := client.(*ServiceCredentialBindingsClient); ok {
				return serviceClient.List(context.Background(), params)
			}

			return nil, constants.ErrNotServiceCredentialBindingsClient
		},
		func(resources []capi.ServiceCredentialBinding) {
			assert.Equal(t, "binding-1", resources[0].Name)
			assert.Equal(t, testAppKey, resources[0].Type)
			assert.Equal(t, "binding-2", resources[1].Name)
			assert.Equal(t, testKeyBindingType, resources[1].Type)
		},
	)
}

func TestServiceCredentialBindingsClient_Update(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_credential_bindings/binding-guid", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var requestBody capi.ServiceCredentialBindingUpdateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		now := time.Now()
		binding := capi.ServiceCredentialBinding{
			Resource: capi.Resource{
				GUID:      testBindingGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:     testBindingName,
			Type:     testAppKey,
			Metadata: requestBody.Metadata,
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(binding)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceBindings := NewServiceCredentialBindingsClient(httpClient)

	request := &capi.ServiceCredentialBindingUpdateRequest{
		Metadata: &capi.Metadata{
			Labels: capi.StringMap(map[string]string{
				testEnvLabelKey: testProductionLabel,
			}),
			Annotations: capi.StringMap(map[string]string{
				"owner": "team-a",
			}),
		},
	}

	binding, err := serviceBindings.Update(context.Background(), testBindingGUID, request)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, testBindingGUID, binding.GUID)
}

func TestServiceCredentialBindingsClient_Delete_ManagedAsync(t *testing.T) {
	t.Parallel()

	// Managed service instances: CF deletes asynchronously and returns 202
	// Accepted with an empty body + Location header pointing at /v3/jobs/{guid}.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_credential_bindings/binding-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.Header().Set("Location", testJobPath)
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	job, err := client.ServiceCredentialBindings().Delete(context.Background(), testBindingGUID)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
}

func TestServiceCredentialBindingsClient_Delete_UserProvidedSync(t *testing.T) {
	t.Parallel()

	// User-provided service instances: CF deletes synchronously and returns
	// 204 No Content. The client returns (nil, nil) so callers can treat a
	// nil Job as "no async work pending — already complete."
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	job, err := client.ServiceCredentialBindings().Delete(context.Background(), testBindingGUID)
	require.NoError(t, err)
	assert.Nil(t, job)
}

func TestServiceCredentialBindingsClient_Delete_MissingLocationOn202(t *testing.T) {
	t.Parallel()

	// 202 without Location is a protocol violation; we return an error
	// rather than a Job with an empty GUID.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	job, err := client.ServiceCredentialBindings().Delete(context.Background(), testBindingGUID)
	require.Error(t, err)
	assert.Nil(t, job)
}

func TestServiceCredentialBindingsClient_GetDetails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_credential_bindings/binding-guid/details", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		details := capi.ServiceCredentialBindingDetails{
			Credentials: map[string]any{
				testUsernameKey: testAdminUsername,
				"password":      "secret",
				"uri":           "mysql://admin:secret@localhost:3306/mydb",
			},
			SyslogDrainURL: StringPtr("syslog://example.com"),
			VolumeMounts:   []any{},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(details)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceBindings := NewServiceCredentialBindingsClient(httpClient)

	details, err := serviceBindings.GetDetails(context.Background(), testBindingGUID)
	require.NoError(t, err)
	assert.NotNil(t, details)
	assert.Equal(t, testAdminUsername, details.Credentials[testUsernameKey])
	assert.Equal(t, "secret", details.Credentials["password"])
	assert.NotNil(t, details.SyslogDrainURL)
	assert.Equal(t, "syslog://example.com", *details.SyslogDrainURL)
}

func TestServiceCredentialBindingsClient_GetParameters(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_credential_bindings/binding-guid/parameters", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		params := capi.ServiceCredentialBindingParameters{
			Parameters: map[string]any{
				testFooKey:        testBarValue,
				"max_connections": 10,
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(params)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceBindings := NewServiceCredentialBindingsClient(httpClient)

	params, err := serviceBindings.GetParameters(context.Background(), testBindingGUID)
	require.NoError(t, err)
	assert.NotNil(t, params)
	assert.Equal(t, testBarValue, params.Parameters[testFooKey])
	assert.InDelta(t, float64(10), params.Parameters["max_connections"], 0)
}

func TestServiceCredentialBindingsClient_GetNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_credential_bindings/binding-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceBindings := NewServiceCredentialBindingsClient(httpClient)

	binding, err := serviceBindings.Get(context.Background(), testBindingGUID)
	require.Error(t, err)
	assert.Nil(t, binding)
}

func TestServiceCredentialBindingsClient_CreateForbidden(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_credential_bindings", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		writer.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceBindings := NewServiceCredentialBindingsClient(httpClient)

	name := testBindingName
	request := &capi.ServiceCredentialBindingCreateRequest{
		Type: testAppKey,
		Name: &name,
		Relationships: capi.ServiceCredentialBindingRelationships{
			ServiceInstance: capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: testInstanceGUID,
				},
			},
			App: &capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: testAppGUID,
				},
			},
		},
	}

	result, err := serviceBindings.Create(context.Background(), request)
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestServiceCredentialBindingsClient_GetWithIncludes(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "app,service_instance", request.URL.Query().Get("include"))
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
		  "guid": "binding-guid",
		  "included": {
		    "apps": [{"guid": "app-1"}],
		    "service_instances": [{"guid": "si-1"}]
		  }
		}`))
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	bindings := NewServiceCredentialBindingsClient(httpClient)

	binding, err := bindings.Get(context.Background(), testBindingGUID,
		capi.ServiceCredentialBindingIncludeApp, capi.ServiceCredentialBindingIncludeServiceInstance)
	require.NoError(t, err)
	require.NotNil(t, binding.Included)
	assert.Equal(t, testAppGUID1, binding.Included.Apps[0].GUID)
	assert.Equal(t, "si-1", binding.Included.ServiceInstances[0].GUID)
}
