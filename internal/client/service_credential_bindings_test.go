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
		assert.Equal(t, "POST", request.Method)

		var requestBody capi.ServiceCredentialBindingCreateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.Equal(t, "app", requestBody.Type)
		assert.Equal(t, "my-binding", *requestBody.Name)
		assert.Equal(t, "instance-guid", requestBody.Relationships.ServiceInstance.Data.GUID)
		assert.Equal(t, "app-guid", requestBody.Relationships.App.Data.GUID)

		// App bindings may return a job for async operations
		job := capi.Job{
			Resource: capi.Resource{
				GUID: "job-guid",
			},
			Operation: "service_credential_binding.create",
			State:     "PROCESSING",
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Location", "/v3/jobs/job-guid")
		writer.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(writer).Encode(job)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceBindings := NewServiceCredentialBindingsClient(httpClient)

	name := "my-binding"
	request := &capi.ServiceCredentialBindingCreateRequest{
		Type: "app",
		Name: &name,
		Parameters: map[string]interface{}{
			"foo": "bar",
		},
		Relationships: capi.ServiceCredentialBindingRelationships{
			ServiceInstance: capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: "instance-guid",
				},
			},
			App: &capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: "app-guid",
				},
			},
		},
	}

	result, err := serviceBindings.Create(context.Background(), request)
	require.NoError(t, err)

	job, ok := result.(*capi.Job)
	require.True(t, ok, "Expected *capi.Job for app binding")
	assert.Equal(t, "job-guid", job.GUID)
	assert.Equal(t, "service_credential_binding.create", job.Operation)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestServiceCredentialBindingsClient_Create_App_WithStrategy(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_credential_bindings", request.URL.Path)
		assert.Equal(t, "POST", request.Method)

		var requestBody capi.ServiceCredentialBindingCreateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.Equal(t, "app", requestBody.Type)
		assert.Equal(t, "my-binding", *requestBody.Name)
		assert.Equal(t, "instance-guid", requestBody.Relationships.ServiceInstance.Data.GUID)
		assert.Equal(t, "app-guid", requestBody.Relationships.App.Data.GUID)
		assert.NotNil(t, requestBody.Strategy)

		if requestBody.Strategy != nil {
			assert.Equal(t, "multiple", *requestBody.Strategy)
		}

		job := capi.Job{
			Resource: capi.Resource{
				GUID: "job-guid",
			},
			Operation: "service_credential_binding.create",
			State:     "PROCESSING",
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Location", "/v3/jobs/job-guid")
		writer.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(writer).Encode(job)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceBindings := NewServiceCredentialBindingsClient(httpClient)

	strategy := "multiple"
	name := "my-binding"
	request := &capi.ServiceCredentialBindingCreateRequest{
		Type:     "app",
		Name:     &name,
		Strategy: &strategy,
		Relationships: capi.ServiceCredentialBindingRelationships{
			ServiceInstance: capi.Relationship{
				Data: &capi.RelationshipData{GUID: "instance-guid"},
			},
			App: &capi.Relationship{
				Data: &capi.RelationshipData{GUID: "app-guid"},
			},
		},
	}

	result, err := serviceBindings.Create(context.Background(), request)
	require.NoError(t, err)

	job, ok := result.(*capi.Job)
	require.True(t, ok, "Expected *capi.Job for app binding with strategy")
	assert.Equal(t, "job-guid", job.GUID)
	assert.Equal(t, "service_credential_binding.create", job.Operation)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestServiceCredentialBindingsClient_Create_Key(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_credential_bindings", request.URL.Path)
		assert.Equal(t, "POST", request.Method)

		var requestBody capi.ServiceCredentialBindingCreateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.Equal(t, "key", requestBody.Type)
		assert.Equal(t, "my-key", *requestBody.Name)
		assert.Equal(t, "instance-guid", requestBody.Relationships.ServiceInstance.Data.GUID)
		assert.Nil(t, requestBody.Relationships.App)

		// Key bindings usually return the binding directly
		now := time.Now()
		binding := capi.ServiceCredentialBinding{
			Resource: capi.Resource{
				GUID:      "binding-guid",
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name: "my-key",
			Type: "key",
			LastOperation: &capi.ServiceCredentialBindingLastOperation{
				Type:      "create",
				State:     "succeeded",
				CreatedAt: &now,
				UpdatedAt: &now,
			},
			Relationships: capi.ServiceCredentialBindingRelationships{
				ServiceInstance: capi.Relationship{
					Data: &capi.RelationshipData{
						GUID: "instance-guid",
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
		Type: "key",
		Name: &name,
		Relationships: capi.ServiceCredentialBindingRelationships{
			ServiceInstance: capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: "instance-guid",
				},
			},
		},
	}

	result, err := serviceBindings.Create(context.Background(), request)
	require.NoError(t, err)

	binding, ok := result.(*capi.ServiceCredentialBinding)
	require.True(t, ok, "Expected *capi.ServiceCredentialBinding for key binding")
	assert.Equal(t, "binding-guid", binding.GUID)
	assert.Equal(t, "my-key", binding.Name)
	assert.Equal(t, "key", binding.Type)
}

func TestServiceCredentialBindingsClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_credential_bindings/binding-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		now := time.Now()
		binding := capi.ServiceCredentialBinding{
			Resource: capi.Resource{
				GUID:      "binding-guid",
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name: "my-binding",
			Type: "app",
			LastOperation: &capi.ServiceCredentialBindingLastOperation{
				Type:      "create",
				State:     "succeeded",
				CreatedAt: &now,
				UpdatedAt: &now,
			},
			Relationships: capi.ServiceCredentialBindingRelationships{
				ServiceInstance: capi.Relationship{
					Data: &capi.RelationshipData{
						GUID: "instance-guid",
					},
				},
				App: &capi.Relationship{
					Data: &capi.RelationshipData{
						GUID: "app-guid",
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

	binding, err := serviceBindings.Get(context.Background(), "binding-guid")
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "binding-guid", binding.GUID)
	assert.Equal(t, "my-binding", binding.Name)
	assert.Equal(t, "app", binding.Type)
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
			Type: "app",
		},
		{
			Resource: capi.Resource{
				GUID:      "binding-guid-2",
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name: "binding-2",
			Type: "key",
		},
	}

	RunServiceListTest(t, "service credential bindings list", "/v3/service_credential_bindings",
		func(request *http.Request) {
			assert.Equal(t, "instance-guid", request.URL.Query().Get("service_instance_guids"))
			assert.Equal(t, "app-guid", request.URL.Query().Get("app_guids"))
		},
		responseData,
		func(httpClient *internalhttp.Client) interface{} {
			return NewServiceCredentialBindingsClient(httpClient)
		},
		func(client interface{}) (*capi.ListResponse[capi.ServiceCredentialBinding], error) {
			params := &capi.QueryParams{
				Filters: map[string][]string{
					"service_instance_guids": {"instance-guid"},
					"app_guids":              {"app-guid"},
				},
			}

			if serviceClient, ok := client.(*ServiceCredentialBindingsClient); ok {
				return serviceClient.List(context.Background(), params)
			}

			return nil, constants.ErrNotServiceCredentialBindingsClient
		},
		func(resources []capi.ServiceCredentialBinding) {
			assert.Equal(t, "binding-1", resources[0].Name)
			assert.Equal(t, "app", resources[0].Type)
			assert.Equal(t, "binding-2", resources[1].Name)
			assert.Equal(t, "key", resources[1].Type)
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
				GUID:      "binding-guid",
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:     "my-binding",
			Type:     "app",
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
			Labels: map[string]string{
				"env": "production",
			},
			Annotations: map[string]string{
				"owner": "team-a",
			},
		},
	}

	binding, err := serviceBindings.Update(context.Background(), "binding-guid", request)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, "binding-guid", binding.GUID)
}

func TestServiceCredentialBindingsClient_Delete_ManagedAsync(t *testing.T) {
	t.Parallel()

	// Managed service instances: CF deletes asynchronously and returns 202
	// Accepted with an empty body + Location header pointing at /v3/jobs/{guid}.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_credential_bindings/binding-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.Header().Set("Location", "/v3/jobs/job-guid")
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	job, err := client.ServiceCredentialBindings().Delete(context.Background(), "binding-guid")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, "job-guid", job.GUID)
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

	job, err := client.ServiceCredentialBindings().Delete(context.Background(), "binding-guid")
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

	job, err := client.ServiceCredentialBindings().Delete(context.Background(), "binding-guid")
	require.Error(t, err)
	assert.Nil(t, job)
}

func TestServiceCredentialBindingsClient_GetDetails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_credential_bindings/binding-guid/details", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		details := capi.ServiceCredentialBindingDetails{
			Credentials: map[string]interface{}{
				"username": "admin",
				"password": "secret",
				"uri":      "mysql://admin:secret@localhost:3306/mydb",
			},
			SyslogDrainURL: StringPtr("syslog://example.com"),
			VolumeMounts:   []interface{}{},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(details)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceBindings := NewServiceCredentialBindingsClient(httpClient)

	details, err := serviceBindings.GetDetails(context.Background(), "binding-guid")
	require.NoError(t, err)
	assert.NotNil(t, details)
	assert.Equal(t, "admin", details.Credentials["username"])
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
			Parameters: map[string]interface{}{
				"foo":             "bar",
				"max_connections": 10,
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(params)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceBindings := NewServiceCredentialBindingsClient(httpClient)

	params, err := serviceBindings.GetParameters(context.Background(), "binding-guid")
	require.NoError(t, err)
	assert.NotNil(t, params)
	assert.Equal(t, "bar", params.Parameters["foo"])
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

	binding, err := serviceBindings.Get(context.Background(), "binding-guid")
	require.Error(t, err)
	assert.Nil(t, binding)
}

func TestServiceCredentialBindingsClient_CreateForbidden(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_credential_bindings", request.URL.Path)
		assert.Equal(t, "POST", request.Method)

		writer.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceBindings := NewServiceCredentialBindingsClient(httpClient)

	name := "my-binding"
	request := &capi.ServiceCredentialBindingCreateRequest{
		Type: "app",
		Name: &name,
		Relationships: capi.ServiceCredentialBindingRelationships{
			ServiceInstance: capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: "instance-guid",
				},
			},
			App: &capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: "app-guid",
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

	binding, err := bindings.Get(context.Background(), "binding-guid",
		capi.ServiceCredentialBindingIncludeApp, capi.ServiceCredentialBindingIncludeServiceInstance)
	require.NoError(t, err)
	require.NotNil(t, binding.Included)
	assert.Equal(t, "app-1", binding.Included.Apps[0].GUID)
	assert.Equal(t, "si-1", binding.Included.ServiceInstances[0].GUID)
}
