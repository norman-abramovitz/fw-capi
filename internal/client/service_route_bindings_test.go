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

func TestServiceRouteBindingsClient_Create(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_route_bindings", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var requestBody capi.ServiceRouteBindingCreateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.Equal(t, testInstanceGUID, requestBody.Relationships.ServiceInstance.Data.GUID)
		assert.Equal(t, testRoutePolicyRoute, requestBody.Relationships.Route.Data.GUID)

		// Service route bindings may return a job for async operations
		job := capi.Job{
			Resource: capi.Resource{
				GUID: testJobGUID,
			},
			Operation: "service_route_binding.create",
			State:     testStateProcessing,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Location", testJobPath)
		writer.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(writer).Encode(job)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceRouteBindings := NewServiceRouteBindingsClient(httpClient)

	request := &capi.ServiceRouteBindingCreateRequest{
		Parameters: map[string]any{
			"rate_limit": 100,
		},
		Relationships: capi.ServiceRouteBindingRelationships{
			ServiceInstance: capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: testInstanceGUID,
				},
			},
			Route: capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: testRoutePolicyRoute,
				},
			},
		},
	}

	result, err := serviceRouteBindings.Create(context.Background(), request)
	require.NoError(t, err)

	job, ok := result.(*capi.Job)
	require.True(t, ok, "Expected *capi.Job for service route binding")
	assert.Equal(t, testJobGUID, job.GUID)
	assert.Equal(t, "service_route_binding.create", job.Operation)
}

func TestServiceRouteBindingsClient_CreateSync(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_route_bindings", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var requestBody capi.ServiceRouteBindingCreateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		// Some service route bindings may return the binding directly
		now := time.Now()
		binding := capi.ServiceRouteBinding{
			Resource: capi.Resource{
				GUID:      testBindingGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			RouteServiceURL: StringPtr("https://route-service.example.com"),
			LastOperation: &capi.ServiceRouteBindingLastOperation{
				Type:      testCreateOperation,
				State:     testSucceededOperation,
				CreatedAt: &now,
				UpdatedAt: &now,
			},
			Relationships: capi.ServiceRouteBindingRelationships{
				ServiceInstance: capi.Relationship{
					Data: &capi.RelationshipData{
						GUID: testInstanceGUID,
					},
				},
				Route: capi.Relationship{
					Data: &capi.RelationshipData{
						GUID: testRoutePolicyRoute,
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
	serviceRouteBindings := NewServiceRouteBindingsClient(httpClient)

	request := &capi.ServiceRouteBindingCreateRequest{
		Relationships: capi.ServiceRouteBindingRelationships{
			ServiceInstance: capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: testInstanceGUID,
				},
			},
			Route: capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: testRoutePolicyRoute,
				},
			},
		},
	}

	result, err := serviceRouteBindings.Create(context.Background(), request)
	require.NoError(t, err)

	binding, ok := result.(*capi.ServiceRouteBinding)
	require.True(t, ok, "Expected *capi.ServiceRouteBinding for synchronous creation")
	assert.Equal(t, testBindingGUID, binding.GUID)
	assert.NotNil(t, binding.RouteServiceURL)
	assert.Equal(t, "https://route-service.example.com", *binding.RouteServiceURL)
}

func TestServiceRouteBindingsClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_route_bindings/binding-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		now := time.Now()
		binding := capi.ServiceRouteBinding{
			Resource: capi.Resource{
				GUID:      testBindingGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			RouteServiceURL: StringPtr("https://route-service.example.com"),
			LastOperation: &capi.ServiceRouteBindingLastOperation{
				Type:      testCreateOperation,
				State:     testSucceededOperation,
				CreatedAt: &now,
				UpdatedAt: &now,
			},
			Relationships: capi.ServiceRouteBindingRelationships{
				ServiceInstance: capi.Relationship{
					Data: &capi.RelationshipData{
						GUID: testInstanceGUID,
					},
				},
				Route: capi.Relationship{
					Data: &capi.RelationshipData{
						GUID: testRoutePolicyRoute,
					},
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(binding)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceRouteBindings := NewServiceRouteBindingsClient(httpClient)

	binding, err := serviceRouteBindings.Get(context.Background(), testBindingGUID)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, testBindingGUID, binding.GUID)
	assert.NotNil(t, binding.RouteServiceURL)
	assert.Equal(t, "https://route-service.example.com", *binding.RouteServiceURL)
}

func TestServiceRouteBindingsClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_route_bindings", request.URL.Path)
		assert.Equal(t, "GET", request.Method)
		assert.Equal(t, testInstanceGUID, request.URL.Query().Get("service_instance_guids"))
		assert.Equal(t, testRoutePolicyRoute, request.URL.Query().Get("route_guids"))

		now := time.Now()
		response := capi.ListResponse[capi.ServiceRouteBinding]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "/v3/service_route_bindings?page=1"},
				Last:         capi.Link{Href: "/v3/service_route_bindings?page=1"},
			},
			Resources: []capi.ServiceRouteBinding{
				{
					Resource: capi.Resource{
						GUID:      "binding-guid-1",
						CreatedAt: now,
						UpdatedAt: now,
					},
					RouteServiceURL: StringPtr("https://route-service1.example.com"),
				},
				{
					Resource: capi.Resource{
						GUID:      "binding-guid-2",
						CreatedAt: now,
						UpdatedAt: now,
					},
					RouteServiceURL: StringPtr("https://route-service2.example.com"),
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceRouteBindings := NewServiceRouteBindingsClient(httpClient)

	params := &capi.QueryParams{
		Filters: map[string][]string{
			"service_instance_guids": {testInstanceGUID},
			"route_guids":            {testRoutePolicyRoute},
		},
	}

	list, err := serviceRouteBindings.List(context.Background(), params)
	require.NoError(t, err)
	assert.NotNil(t, list)
	assert.Equal(t, 2, list.Pagination.TotalResults)
	assert.Len(t, list.Resources, 2)
	assert.Equal(t, "binding-guid-1", list.Resources[0].GUID)
	assert.Equal(t, "binding-guid-2", list.Resources[1].GUID)
}

func TestServiceRouteBindingsClient_Update(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_route_bindings/binding-guid", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var requestBody capi.ServiceRouteBindingUpdateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		now := time.Now()
		binding := capi.ServiceRouteBinding{
			Resource: capi.Resource{
				GUID:      testBindingGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			RouteServiceURL: StringPtr("https://route-service.example.com"),
			Metadata:        requestBody.Metadata,
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(binding)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceRouteBindings := NewServiceRouteBindingsClient(httpClient)

	request := &capi.ServiceRouteBindingUpdateRequest{
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				testEnvLabelKey: testProductionLabel,
			},
			Annotations: map[string]string{
				"owner": "team-a",
			},
		},
	}

	binding, err := serviceRouteBindings.Update(context.Background(), testBindingGUID, request)
	require.NoError(t, err)
	assert.NotNil(t, binding)
	assert.Equal(t, testBindingGUID, binding.GUID)
}

func TestServiceRouteBindingsClient_Delete(t *testing.T) {
	t.Parallel()
	RunJobDeleteTest(t, "service route binding delete", "/v3/service_route_bindings/binding-guid", "service_route_binding.delete",
		func(httpClient *internalhttp.Client) any {
			return NewServiceRouteBindingsClient(httpClient)
		},
		func(client any) (*capi.Job, error) {
			serviceRouteBindingsClient, ok := client.(*ServiceRouteBindingsClient)
			if !ok {
				return nil, constants.ErrInvalidClientType
			}

			return serviceRouteBindingsClient.Delete(context.Background(), testBindingGUID)
		},
	)
}

func TestServiceRouteBindingsClient_Delete_UserProvidedSync(t *testing.T) {
	t.Parallel()

	// User-provided service instances: CF deletes synchronously and returns
	// 204 No Content. The client returns (nil, nil) so callers can treat a
	// nil Job as "no async work pending — already complete."
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceRouteBindings := NewServiceRouteBindingsClient(httpClient)

	job, err := serviceRouteBindings.Delete(context.Background(), testBindingGUID)
	require.NoError(t, err)
	assert.Nil(t, job)
}

func TestServiceRouteBindingsClient_Delete_ManagedAsync(t *testing.T) {
	t.Parallel()

	// Managed service instances: CF deletes asynchronously and returns 202
	// Accepted with an empty body + Location header pointing at /v3/jobs/{guid}.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_route_bindings/binding-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.Header().Set("Location", testJobPath)
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceRouteBindings := NewServiceRouteBindingsClient(httpClient)

	job, err := serviceRouteBindings.Delete(context.Background(), testBindingGUID)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
}

func TestServiceRouteBindingsClient_Delete_MissingLocationOn202(t *testing.T) {
	t.Parallel()

	// 202 without Location is a protocol violation; we return an error
	// rather than a Job with an empty GUID.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceRouteBindings := NewServiceRouteBindingsClient(httpClient)

	job, err := serviceRouteBindings.Delete(context.Background(), testBindingGUID)
	require.Error(t, err)
	assert.Nil(t, job)
}

func TestServiceRouteBindingsClient_GetParameters(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_route_bindings/binding-guid/parameters", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		params := capi.ServiceRouteBindingParameters{
			Parameters: map[string]any{
				"rate_limit": 100,
				"enabled":    true,
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(params)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceRouteBindings := NewServiceRouteBindingsClient(httpClient)

	params, err := serviceRouteBindings.GetParameters(context.Background(), testBindingGUID)
	require.NoError(t, err)
	assert.NotNil(t, params)
	assert.InDelta(t, float64(100), params.Parameters["rate_limit"], 0.0001)
	assert.Equal(t, true, params.Parameters["enabled"])
}

func TestServiceRouteBindingsClient_GetNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_route_bindings/binding-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceRouteBindings := NewServiceRouteBindingsClient(httpClient)

	binding, err := serviceRouteBindings.Get(context.Background(), testBindingGUID)
	require.Error(t, err)
	assert.Nil(t, binding)
}

func TestServiceRouteBindingsClient_CreateForbidden(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_route_bindings", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		writer.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceRouteBindings := NewServiceRouteBindingsClient(httpClient)

	request := &capi.ServiceRouteBindingCreateRequest{
		Relationships: capi.ServiceRouteBindingRelationships{
			ServiceInstance: capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: testInstanceGUID,
				},
			},
			Route: capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: testRoutePolicyRoute,
				},
			},
		},
	}

	result, err := serviceRouteBindings.Create(context.Background(), request)
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestServiceRouteBindingsClient_GetWithIncludes(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "route,service_instance", request.URL.Query().Get("include"))
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
		  "guid": "binding-guid",
		  "included": {
		    "routes": [{"guid": "route-1"}],
		    "service_instances": [{"guid": "si-1"}]
		  }
		}`))
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	bindings := NewServiceRouteBindingsClient(httpClient)

	binding, err := bindings.Get(context.Background(), testBindingGUID,
		capi.ServiceRouteBindingIncludeRoute, capi.ServiceRouteBindingIncludeServiceInstance)
	require.NoError(t, err)
	require.NotNil(t, binding.Included)
	assert.Equal(t, "route-1", binding.Included.Routes[0].GUID)
}
