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

// testRoutesPath is the base routes collection path. testV1Path is defined
// in domains_test.go and shared here since both exercise route path values.
const testRoutesPath = "/v3/routes"

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestRoutesClient_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		request      *capi.RouteCreateRequest
		response     any
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "create route with host",
			expectedPath: testRoutesPath,
			statusCode:   http.StatusCreated,
			request: &capi.RouteCreateRequest{
				Host: StringPtr(testAPIHost),
				Path: StringPtr(testV1Path),
				Relationships: capi.RouteRelationships{
					Space:  capi.Relationship{Data: &capi.RelationshipData{GUID: testSpaceGUID}},
					Domain: capi.Relationship{Data: &capi.RelationshipData{GUID: testDomainGUID}},
				},
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						testTypeKey: testAPIHost,
					},
				},
			},
			response: capi.Route{
				Resource: capi.Resource{
					GUID:      testRoutePolicyRoute,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						testSelfKey: capi.Link{
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
				Protocol: testHTTPProtocol,
				Host:     testAPIHost,
				Path:     testV1Path,
				URL:      testAPIExampleV1URL,
				Relationships: capi.RouteRelationships{
					Space:  capi.Relationship{Data: &capi.RelationshipData{GUID: testSpaceGUID}},
					Domain: capi.Relationship{Data: &capi.RelationshipData{GUID: testDomainGUID}},
				},
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						testTypeKey: testAPIHost,
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "create route with port",
			expectedPath: testRoutesPath,
			statusCode:   http.StatusCreated,
			request: &capi.RouteCreateRequest{
				Port: intPtr(8080),
				Relationships: capi.RouteRelationships{
					Space:  capi.Relationship{Data: &capi.RelationshipData{GUID: testSpaceGUID}},
					Domain: capi.Relationship{Data: &capi.RelationshipData{GUID: testDomainGUID}},
				},
			},
			response: capi.Route{
				Resource: capi.Resource{
					GUID:      testRoutePolicyRoute,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Protocol: testTCPProtocol,
				Port:     intPtr(8080),
				URL:      "example.com:8080",
				Relationships: capi.RouteRelationships{
					Space:  capi.Relationship{Data: &capi.RelationshipData{GUID: testSpaceGUID}},
					Domain: capi.Relationship{Data: &capi.RelationshipData{GUID: testDomainGUID}},
				},
			},
			wantErr: false,
		},
		{
			name:         "route already exists",
			expectedPath: testRoutesPath,
			statusCode:   http.StatusUnprocessableEntity,
			request: &capi.RouteCreateRequest{
				Host: StringPtr("existing"),
				Relationships: capi.RouteRelationships{
					Space:  capi.Relationship{Data: &capi.RelationshipData{GUID: testSpaceGUID}},
					Domain: capi.Relationship{Data: &capi.RelationshipData{GUID: testDomainGUID}},
				},
			},
			response: map[string]any{
				testErrorsKey: []map[string]any{
					{
						testCodeKey:   10008,
						testTitleKey:  testUnprocessableTitle,
						testDetailKey: "Route already exists",
					},
				},
			},
			wantErr:    true,
			errMessage: testUnprocessableTitle,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, testCase.expectedPath, request.URL.Path)
				assert.Equal(t, http.MethodPost, request.Method)

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
				require.ErrorContains(t, err, testCase.errMessage)
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
		response     any
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         testSuccessfulGetCase,
			guid:         testRouteGUIDFixture,
			expectedPath: "/v3/routes/test-route-guid",
			statusCode:   http.StatusOK,
			response: capi.Route{
				Resource: capi.Resource{
					GUID:      testRouteGUIDFixture,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Protocol: testHTTPProtocol,
				Host:     testAPIHost,
				Path:     testV1Path,
				URL:      testAPIExampleV1URL,
			},
			wantErr: false,
		},
		{
			name:         "route not found",
			guid:         testNonExistentGUID,
			expectedPath: "/v3/routes/non-existent-guid",
			statusCode:   http.StatusNotFound,
			response: map[string]any{
				testErrorsKey: []map[string]any{
					{
						testCodeKey:   10010,
						testTitleKey:  testNotFoundTitle,
						testDetailKey: "Route not found",
					},
				},
			},
			wantErr:    true,
			errMessage: testNotFoundTitle,
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
				require.ErrorContains(t, err, testCase.errMessage)
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
		assert.Equal(t, testRoutesPath, request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		// Check query parameters if present
		query := request.URL.Query()
		if hosts := query.Get("hosts"); hosts != "" {
			assert.Equal(t, "api,www", hosts)
		}

		if spaceGuids := query.Get(testSpaceGUIDsParam); spaceGuids != "" {
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
					Protocol: testHTTPProtocol,
					Host:     testAPIHost,
					Path:     testV1Path,
					URL:      testAPIExampleV1URL,
				},
				{
					Resource: capi.Resource{
						GUID:      "route-2",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Protocol: testTCPProtocol,
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
			"hosts":             {testAPIHost, "www"},
			testSpaceGUIDsParam: {testSpaceName1, testSpaceName2},
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
				GUID:      testRouteGUIDFixture,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Protocol: testHTTPProtocol,
			Host:     testAPIHost,
			Path:     testV1Path,
			URL:      testAPIExampleV1URL,
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
				testEnvironmentLabelKey: testStagingLabel,
			},
			Annotations: map[string]string{
				testNoteAnnotationKey: "Updated route",
			},
		},
	}

	route, err := client.Routes().Update(context.Background(), testRouteGUIDFixture, request)
	require.NoError(t, err)
	require.NotNil(t, route)
	assert.Equal(t, testRouteGUIDFixture, route.GUID)
	assert.Equal(t, testStagingLabel, route.Metadata.Labels[testEnvironmentLabelKey])
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

		writer.Header().Set("Location", testJobPath)
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	job, err := client.Routes().Delete(context.Background(), testRouteGUIDFixture)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
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

	job, err := client.Routes().Delete(context.Background(), testRouteGUIDFixture)
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
						GUID: testAppGUID1,
						Process: &capi.Process{
							Resource: capi.Resource{GUID: "process-1"},
							Type:     testWebProcessType,
						},
					},
					Port:     intPtr(8080),
					Protocol: StringPtr("http1"),
					Weight:   intPtr(100),
				},
				{
					GUID: "dest-2",
					App: capi.RouteDestinationApp{
						GUID: testAppGUID2,
					},
				},
			},
			Links: capi.Links{
				testSelfKey: capi.Link{
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

	destinations, err := client.Routes().ListDestinations(context.Background(), testRouteGUIDFixture)
	require.NoError(t, err)
	require.NotNil(t, destinations)
	assert.Len(t, destinations.Destinations, 2)
	assert.Equal(t, "dest-1", destinations.Destinations[0].GUID)
	assert.Equal(t, testAppGUID1, destinations.Destinations[0].App.GUID)
}

func TestRoutesClient_InsertDestinations(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/routes/test-route-guid/destinations", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

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
						GUID: testAppGUID1,
					},
				},
				{
					GUID: "dest-2",
					App: capi.RouteDestinationApp{
						GUID: testAppGUID2,
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
				GUID: testAppGUID2,
			},
		},
	}

	destinations, err := client.Routes().InsertDestinations(context.Background(), testRouteGUIDFixture, newDestinations)
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

	destinations, err := client.Routes().ReplaceDestinations(context.Background(), testRouteGUIDFixture, newDestinations)
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
				GUID: testAppGUID1,
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

	destination, err := client.Routes().UpdateDestination(context.Background(), testRouteGUIDFixture, "dest-guid", "http2")
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

	err = client.Routes().RemoveDestination(context.Background(), testRouteGUIDFixture, "dest-guid")
	require.NoError(t, err)
}

func TestRoutesClient_ShareWithSpace(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/routes/test-route-guid/relationships/shared_spaces", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var requestBody struct {
			Data []capi.RelationshipData `json:"data"`
		}

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)
		assert.Len(t, requestBody.Data, 2)

		response := capi.ToManyRelationship{
			Data: []capi.RelationshipData{
				{GUID: testSpaceName1},
				{GUID: testSpaceName2},
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

	relationship, err := client.Routes().ShareWithSpace(context.Background(), testRouteGUIDFixture, []string{testSpaceName1, testSpaceName2})
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

	err = client.Routes().UnshareFromSpace(context.Background(), testRouteGUIDFixture, testSpaceGUID)
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
				GUID:      testRouteGUIDFixture,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Protocol: testHTTPProtocol,
			Host:     testAPIHost,
			Path:     testV1Path,
			URL:      testAPIExampleV1URL,
			Relationships: capi.RouteRelationships{
				Space:  capi.Relationship{Data: &capi.RelationshipData{GUID: "new-space-guid"}},
				Domain: capi.Relationship{Data: &capi.RelationshipData{GUID: testDomainGUID}},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	route, err := client.Routes().TransferOwnership(context.Background(), testRouteGUIDFixture, "new-space-guid")
	require.NoError(t, err)
	require.NotNil(t, route)
	assert.Equal(t, testRouteGUIDFixture, route.GUID)
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

	route, err := routes.Get(context.Background(), testRoutePolicyRoute,
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
		assert.Equal(t, "a1", request.URL.Query().Get(testAppGUIDsParam))

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"destinations": []}`))
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	routes := NewRoutesClient(httpClient)

	_, err := routes.ListDestinations(context.Background(), testRoutePolicyRoute,
		capi.WithDestinationGUIDs("d1", "d2"), capi.WithDestinationAppGUIDs("a1"))
	require.NoError(t, err)
}
