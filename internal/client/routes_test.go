package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/fivetwenty-io/capi/v3/internal/client"
	internalhttp "github.com/fivetwenty-io/capi/v3/internal/http"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
)

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestRoutesClient_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		request      *capi.RouteCreateRequest
		response     interface{}
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "create route with host",
			expectedPath: "/v3/routes",
			statusCode:   http.StatusCreated,
			request: &capi.RouteCreateRequest{
				Host: StringPtr("api"),
				Path: StringPtr("/v1"),
				Relationships: capi.RouteRelationships{
					Space:  capi.Relationship{Data: &capi.RelationshipData{GUID: "space-guid"}},
					Domain: capi.Relationship{Data: &capi.RelationshipData{GUID: "domain-guid"}},
				},
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						"type": "api",
					},
				},
			},
			response: capi.Route{
				Resource: capi.Resource{
					GUID:      "route-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						"self": capi.Link{
							Href: "https://api.example.org/v3/routes/route-guid",
						},
						"space": capi.Link{
							Href: "https://api.example.org/v3/spaces/space-guid",
						},
						"domain": capi.Link{
							Href: "https://api.example.org/v3/domains/domain-guid",
						},
						"destinations": capi.Link{
							Href: "https://api.example.org/v3/routes/route-guid/destinations",
						},
					},
				},
				Protocol: "http",
				Host:     "api",
				Path:     "/v1",
				URL:      "api.example.com/v1",
				Relationships: capi.RouteRelationships{
					Space:  capi.Relationship{Data: &capi.RelationshipData{GUID: "space-guid"}},
					Domain: capi.Relationship{Data: &capi.RelationshipData{GUID: "domain-guid"}},
				},
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						"type": "api",
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "create route with port",
			expectedPath: "/v3/routes",
			statusCode:   http.StatusCreated,
			request: &capi.RouteCreateRequest{
				Port: intPtr(8080),
				Relationships: capi.RouteRelationships{
					Space:  capi.Relationship{Data: &capi.RelationshipData{GUID: "space-guid"}},
					Domain: capi.Relationship{Data: &capi.RelationshipData{GUID: "domain-guid"}},
				},
			},
			response: capi.Route{
				Resource: capi.Resource{
					GUID:      "route-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Protocol: "tcp",
				Port:     intPtr(8080),
				URL:      "example.com:8080",
				Relationships: capi.RouteRelationships{
					Space:  capi.Relationship{Data: &capi.RelationshipData{GUID: "space-guid"}},
					Domain: capi.Relationship{Data: &capi.RelationshipData{GUID: "domain-guid"}},
				},
			},
			wantErr: false,
		},
		{
			name:         "route already exists",
			expectedPath: "/v3/routes",
			statusCode:   http.StatusUnprocessableEntity,
			request: &capi.RouteCreateRequest{
				Host: StringPtr("existing"),
				Relationships: capi.RouteRelationships{
					Space:  capi.Relationship{Data: &capi.RelationshipData{GUID: "space-guid"}},
					Domain: capi.Relationship{Data: &capi.RelationshipData{GUID: "domain-guid"}},
				},
			},
			response: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"code":   10008,
						"title":  "CF-UnprocessableEntity",
						"detail": "Route already exists",
					},
				},
			},
			wantErr:    true,
			errMessage: "CF-UnprocessableEntity",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, testCase.expectedPath, request.URL.Path)
				assert.Equal(t, "POST", request.Method)

				var requestBody capi.RouteCreateRequest

				err := json.NewDecoder(request.Body).Decode(&requestBody)
				assert.NoError(t, err)

				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(testCase.statusCode)
				_ = json.NewEncoder(writer).Encode(testCase.response)
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			route, err := client.Routes().Create(context.Background(), testCase.request)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMessage)
				assert.Nil(t, route)
			} else {
				require.NoError(t, err)
				require.NotNil(t, route)
				assert.NotEmpty(t, route.GUID)
			}
		})
	}
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestRoutesClient_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		guid         string
		response     interface{}
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "successful get",
			guid:         "test-route-guid",
			expectedPath: "/v3/routes/test-route-guid",
			statusCode:   http.StatusOK,
			response: capi.Route{
				Resource: capi.Resource{
					GUID:      "test-route-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Protocol: "http",
				Host:     "api",
				Path:     "/v1",
				URL:      "api.example.com/v1",
			},
			wantErr: false,
		},
		{
			name:         "route not found",
			guid:         "non-existent-guid",
			expectedPath: "/v3/routes/non-existent-guid",
			statusCode:   http.StatusNotFound,
			response: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"code":   10010,
						"title":  "CF-ResourceNotFound",
						"detail": "Route not found",
					},
				},
			},
			wantErr:    true,
			errMessage: "CF-ResourceNotFound",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, testCase.expectedPath, request.URL.Path)
				assert.Equal(t, "GET", request.Method)
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(testCase.statusCode)
				_ = json.NewEncoder(writer).Encode(testCase.response)
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			route, err := client.Routes().Get(context.Background(), testCase.guid)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMessage)
				assert.Nil(t, route)
			} else {
				require.NoError(t, err)
				require.NotNil(t, route)
				assert.Equal(t, testCase.guid, route.GUID)
			}
		})
	}
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestRoutesClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/routes", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		// Check query parameters if present
		query := request.URL.Query()
		if hosts := query.Get("hosts"); hosts != "" {
			assert.Equal(t, "api,www", hosts)
		}

		if spaceGuids := query.Get("space_guids"); spaceGuids != "" {
			assert.Equal(t, "space-1,space-2", spaceGuids)
		}

		response := capi.ListResponse[capi.Route]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "https://api.example.org/v3/routes?page=1"},
				Last:         capi.Link{Href: "https://api.example.org/v3/routes?page=1"},
				Next:         nil,
				Previous:     nil,
			},
			Resources: []capi.Route{
				{
					Resource: capi.Resource{
						GUID:      "route-1",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Protocol: "http",
					Host:     "api",
					Path:     "/v1",
					URL:      "api.example.com/v1",
				},
				{
					Resource: capi.Resource{
						GUID:      "route-2",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Protocol: "tcp",
					Port:     intPtr(8080),
					URL:      "example.com:8080",
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

	// Test without filters
	result, err := client.Routes().List(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.TotalResults)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "route-1", result.Resources[0].GUID)

	// Test with filters
	params := &capi.QueryParams{
		Filters: map[string][]string{
			"hosts":       {"api", "www"},
			"space_guids": {"space-1", "space-2"},
		},
	}
	result, err = client.Routes().List(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestRoutesClient_Update(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/routes/test-route-guid", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var requestBody capi.RouteUpdateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		response := capi.Route{
			Resource: capi.Resource{
				GUID:      "test-route-guid",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Protocol: "http",
			Host:     "api",
			Path:     "/v1",
			URL:      "api.example.com/v1",
			Metadata: requestBody.Metadata,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	request := &capi.RouteUpdateRequest{
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				"environment": "staging",
			},
			Annotations: map[string]string{
				"note": "Updated route",
			},
		},
	}

	route, err := client.Routes().Update(context.Background(), "test-route-guid", request)
	require.NoError(t, err)
	require.NotNil(t, route)
	assert.Equal(t, "test-route-guid", route.GUID)
	assert.Equal(t, "staging", route.Metadata.Labels["environment"])
}

func TestRoutesClient_Delete(t *testing.T) {
	t.Parallel()

	// CF v3 DELETE /v3/routes/{guid} is async: 202 Accepted, empty body,
	// Location header pointing at /v3/jobs/{jobGuid}. Same contract as
	// AppsClient.Delete. The Operation field on the Job is NOT populated on
	// the delete response — callers must GET /v3/jobs/{guid} to resolve it.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/routes/test-route-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.Header().Set("Location", "/v3/jobs/job-guid")
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	job, err := client.Routes().Delete(context.Background(), "test-route-guid")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, "job-guid", job.GUID)
}

func TestRoutesClient_DeleteMissingLocation(t *testing.T) {
	t.Parallel()

	// 202 without a Location header is a protocol violation; we return an
	// error rather than a Job with an empty GUID the caller could accidentally
	// poll against /v3/jobs/.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	job, err := client.Routes().Delete(context.Background(), "test-route-guid")
	require.Error(t, err)
	assert.Nil(t, job)
}

func TestRoutesClient_ListDestinations(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/routes/test-route-guid/destinations", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		response := capi.RouteDestinations{
			Destinations: []capi.RouteDestination{
				{
					GUID: "dest-1",
					App: capi.RouteDestinationApp{
						GUID: "app-1",
						Process: &capi.Process{
							Resource: capi.Resource{GUID: "process-1"},
							Type:     "web",
						},
					},
					Port:     intPtr(8080),
					Protocol: StringPtr("http1"),
					Weight:   intPtr(100),
				},
				{
					GUID: "dest-2",
					App: capi.RouteDestinationApp{
						GUID: "app-2",
					},
				},
			},
			Links: capi.Links{
				"self": capi.Link{
					Href: "https://api.example.org/v3/routes/test-route-guid/destinations",
				},
				"route": capi.Link{
					Href: "https://api.example.org/v3/routes/test-route-guid",
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

	destinations, err := client.Routes().ListDestinations(context.Background(), "test-route-guid")
	require.NoError(t, err)
	require.NotNil(t, destinations)
	assert.Len(t, destinations.Destinations, 2)
	assert.Equal(t, "dest-1", destinations.Destinations[0].GUID)
	assert.Equal(t, "app-1", destinations.Destinations[0].App.GUID)
}

func TestRoutesClient_InsertDestinations(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/routes/test-route-guid/destinations", request.URL.Path)
		assert.Equal(t, "POST", request.Method)

		var requestBody struct {
			Destinations []capi.RouteDestination `json:"destinations"`
		}

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)
		assert.Len(t, requestBody.Destinations, 1)

		response := capi.RouteDestinations{
			Destinations: []capi.RouteDestination{
				{
					GUID: "dest-1",
					App: capi.RouteDestinationApp{
						GUID: "app-1",
					},
				},
				{
					GUID: "dest-2",
					App: capi.RouteDestinationApp{
						GUID: "app-2",
					},
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

	newDestinations := []capi.RouteDestination{
		{
			App: capi.RouteDestinationApp{
				GUID: "app-2",
			},
		},
	}

	destinations, err := client.Routes().InsertDestinations(context.Background(), "test-route-guid", newDestinations)
	require.NoError(t, err)
	require.NotNil(t, destinations)
	assert.Len(t, destinations.Destinations, 2)
}

func TestRoutesClient_ReplaceDestinations(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/routes/test-route-guid/destinations", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var requestBody struct {
			Destinations []capi.RouteDestination `json:"destinations"`
		}

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)
		assert.Len(t, requestBody.Destinations, 1)

		response := capi.RouteDestinations{
			Destinations: requestBody.Destinations,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	newDestinations := []capi.RouteDestination{
		{
			App: capi.RouteDestinationApp{
				GUID: "app-new",
			},
		},
	}

	destinations, err := client.Routes().ReplaceDestinations(context.Background(), "test-route-guid", newDestinations)
	require.NoError(t, err)
	require.NotNil(t, destinations)
	assert.Len(t, destinations.Destinations, 1)
}

func TestRoutesClient_UpdateDestination(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/routes/test-route-guid/destinations/dest-guid", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var requestBody struct {
			Protocol string `json:"protocol"`
		}

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)
		assert.Equal(t, "http2", requestBody.Protocol)

		response := capi.RouteDestination{
			GUID: "dest-guid",
			App: capi.RouteDestinationApp{
				GUID: "app-1",
			},
			Protocol: StringPtr("http2"),
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	destination, err := client.Routes().UpdateDestination(context.Background(), "test-route-guid", "dest-guid", "http2")
	require.NoError(t, err)
	require.NotNil(t, destination)
	assert.Equal(t, "dest-guid", destination.GUID)
	assert.Equal(t, "http2", *destination.Protocol)
}

func TestRoutesClient_RemoveDestination(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/routes/test-route-guid/destinations/dest-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	err = client.Routes().RemoveDestination(context.Background(), "test-route-guid", "dest-guid")
	require.NoError(t, err)
}

func TestRoutesClient_ShareWithSpace(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/routes/test-route-guid/relationships/shared_spaces", request.URL.Path)
		assert.Equal(t, "POST", request.Method)

		var requestBody struct {
			Data []capi.RelationshipData `json:"data"`
		}

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)
		assert.Len(t, requestBody.Data, 2)

		response := capi.ToManyRelationship{
			Data: []capi.RelationshipData{
				{GUID: "space-1"},
				{GUID: "space-2"},
				{GUID: "space-3"},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	relationship, err := client.Routes().ShareWithSpace(context.Background(), "test-route-guid", []string{"space-1", "space-2"})
	require.NoError(t, err)
	require.NotNil(t, relationship)
	assert.Len(t, relationship.Data, 3)
}

func TestRoutesClient_UnshareFromSpace(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/routes/test-route-guid/relationships/shared_spaces/space-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	err = client.Routes().UnshareFromSpace(context.Background(), "test-route-guid", "space-guid")
	require.NoError(t, err)
}

func TestRoutesClient_TransferOwnership(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/routes/test-route-guid", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var requestBody struct {
			Relationships capi.RouteRelationships `json:"relationships"`
		}

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)
		assert.Equal(t, "new-space-guid", requestBody.Relationships.Space.Data.GUID)

		response := capi.Route{
			Resource: capi.Resource{
				GUID:      "test-route-guid",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Protocol: "http",
			Host:     "api",
			Path:     "/v1",
			URL:      "api.example.com/v1",
			Relationships: capi.RouteRelationships{
				Space:  capi.Relationship{Data: &capi.RelationshipData{GUID: "new-space-guid"}},
				Domain: capi.Relationship{Data: &capi.RelationshipData{GUID: "domain-guid"}},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	route, err := client.Routes().TransferOwnership(context.Background(), "test-route-guid", "new-space-guid")
	require.NoError(t, err)
	require.NotNil(t, route)
	assert.Equal(t, "test-route-guid", route.GUID)
	assert.Equal(t, "new-space-guid", route.Relationships.Space.Data.GUID)
}

func TestRoutesClient_GetWithIncludes(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/routes/route-guid", request.URL.Path)
		assert.Equal(t, "domain,space.organization", request.URL.Query().Get("include"))

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
		  "guid": "route-guid",
		  "included": {
		    "domains": [{"guid": "domain-1", "name": "apps.example.com"}],
		    "spaces": [{"guid": "space-1"}],
		    "organizations": [{"guid": "org-1"}]
		  }
		}`))
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	routes := NewRoutesClient(httpClient)

	route, err := routes.Get(context.Background(), "route-guid",
		capi.RouteIncludeDomain, capi.RouteIncludeSpaceOrganization)
	require.NoError(t, err)
	require.NotNil(t, route.Included)
	assert.Equal(t, "domain-1", route.Included.Domains[0].GUID)
}

func TestRoutesClient_ListDestinationsWithFilters(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/routes/route-guid/destinations", request.URL.Path)
		assert.Equal(t, "d1,d2", request.URL.Query().Get("guids"))
		assert.Equal(t, "a1", request.URL.Query().Get("app_guids"))

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"destinations": []}`))
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	routes := NewRoutesClient(httpClient)

	_, err := routes.ListDestinations(context.Background(), "route-guid",
		capi.WithDestinationGUIDs("d1", "d2"), capi.WithDestinationAppGUIDs("a1"))
	require.NoError(t, err)
}
