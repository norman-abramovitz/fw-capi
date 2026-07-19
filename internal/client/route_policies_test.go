package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/fivetwenty-io/capi/v3/internal/client"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
)

// Test constants for route policy tests.
const (
	testRoutePoliciesPath  = "/v3/route_policies"
	testRoutePolicyGUID    = "policy-guid"
	testRoutePolicyRoute   = "route-guid"
	testTeamLabel          = "team"
	testNotFoundTitle      = "CF-ResourceNotFound"
	testUnprocessableTitle = "CF-UnprocessableEntity"

	testNotFoundErrBody      = `{"errors":[{"code":10010,"title":"CF-ResourceNotFound","detail":"Route policy not found"}]}`
	testUnprocessableErrBody = `{"errors":[{"code":10008,"title":"CF-UnprocessableEntity","detail":"Route's domain does not enforce route policies"}]}`
)

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestRoutePoliciesClient_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		request      *capi.RoutePolicyCreateRequest
		response     any
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "create app-source policy",
			expectedPath: testRoutePoliciesPath,
			statusCode:   http.StatusCreated,
			request: &capi.RoutePolicyCreateRequest{
				Source: capi.RoutePolicySourceApp("d76446a1-f429-4444-8797-be2f78b75b08"),
				Relationships: capi.RoutePolicyRelationships{
					Route: capi.Relationship{Data: &capi.RelationshipData{GUID: testRoutePolicyRoute}},
				},
				Metadata: &capi.Metadata{
					Labels: capi.StringMap(map[string]string{testTeamLabel: "frontend"}),
				},
			},
			response: capi.RoutePolicy{
				Resource: capi.Resource{
					GUID:      testRoutePolicyGUID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Source: "cf:app:d76446a1-f429-4444-8797-be2f78b75b08",
				Relationships: capi.RoutePolicyRelationships{
					Route: capi.Relationship{Data: &capi.RelationshipData{GUID: testRoutePolicyRoute}},
					App:   &capi.Relationship{Data: &capi.RelationshipData{GUID: "d76446a1-f429-4444-8797-be2f78b75b08"}},
				},
				Metadata: &capi.Metadata{
					Labels: capi.StringMap(map[string]string{testTeamLabel: "frontend"}),
				},
			},
			wantErr: false,
		},
		{
			name:         "create any-source policy",
			expectedPath: testRoutePoliciesPath,
			statusCode:   http.StatusCreated,
			request: &capi.RoutePolicyCreateRequest{
				Source: capi.RoutePolicySourceAny,
				Relationships: capi.RoutePolicyRelationships{
					Route: capi.Relationship{Data: &capi.RelationshipData{GUID: testRoutePolicyRoute}},
				},
			},
			response: capi.RoutePolicy{
				Resource: capi.Resource{
					GUID:      testRoutePolicyGUID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Source: capi.RoutePolicySourceAny,
				Relationships: capi.RoutePolicyRelationships{
					Route: capi.Relationship{Data: &capi.RelationshipData{GUID: testRoutePolicyRoute}},
				},
			},
			wantErr: false,
		},
		{
			name:         "domain does not enforce route policies",
			expectedPath: testRoutePoliciesPath,
			statusCode:   http.StatusUnprocessableEntity,
			request: &capi.RoutePolicyCreateRequest{
				Source: capi.RoutePolicySourceSpace(testSpaceGUID),
				Relationships: capi.RoutePolicyRelationships{
					Route: capi.Relationship{Data: &capi.RelationshipData{GUID: testRoutePolicyRoute}},
				},
			},
			response:   json.RawMessage(testUnprocessableErrBody),
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

				var requestBody capi.RoutePolicyCreateRequest

				err := json.NewDecoder(request.Body).Decode(&requestBody)
				assert.NoError(t, err)
				assert.Equal(t, testCase.request.Source, requestBody.Source)

				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(testCase.statusCode)
				_ = json.NewEncoder(writer).Encode(testCase.response)
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			policy, err := client.RoutePolicies().Create(context.Background(), testCase.request)

			if testCase.wantErr {
				require.ErrorContains(t, err, testCase.errMessage)
				assert.Nil(t, policy)
			} else {
				require.NoError(t, err)
				require.NotNil(t, policy)
				assert.NotEmpty(t, policy.GUID)
				assert.Equal(t, testCase.request.Source, policy.Source)
			}
		})
	}
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestRoutePoliciesClient_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		guid          string
		opts          []capi.RoutePolicyGetOption
		expectedQuery string
		statusCode    int
		wantErr       bool
		errMessage    string
	}{
		{
			name:       "get route policy",
			guid:       testRoutePolicyGUID,
			statusCode: http.StatusOK,
		},
		{
			name:          "get with include route and source",
			guid:          testRoutePolicyGUID,
			opts:          []capi.RoutePolicyGetOption{capi.RoutePolicyIncludeRoute, capi.RoutePolicyIncludeSource},
			expectedQuery: "include=route,source",
			statusCode:    http.StatusOK,
		},
		{
			name:       "policy not found",
			guid:       "missing-guid",
			statusCode: http.StatusNotFound,
			wantErr:    true,
			errMessage: testNotFoundTitle,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, testRoutePoliciesPath+"/"+testCase.guid, request.URL.Path)
				assert.Equal(t, http.MethodGet, request.Method)

				if testCase.expectedQuery != "" {
					decoded, err := url.QueryUnescape(request.URL.RawQuery)
					assert.NoError(t, err)
					assert.Equal(t, testCase.expectedQuery, decoded)
				}

				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(testCase.statusCode)

				if testCase.wantErr {
					_, _ = writer.Write([]byte(testNotFoundErrBody))

					return
				}

				_ = json.NewEncoder(writer).Encode(capi.RoutePolicy{
					Resource: capi.Resource{GUID: testCase.guid, CreatedAt: time.Now(), UpdatedAt: time.Now()},
					Source:   capi.RoutePolicySourceAny,
					Relationships: capi.RoutePolicyRelationships{
						Route: capi.Relationship{Data: &capi.RelationshipData{GUID: testRoutePolicyRoute}},
					},
				})
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			policy, err := client.RoutePolicies().Get(context.Background(), testCase.guid, testCase.opts...)

			if testCase.wantErr {
				require.ErrorContains(t, err, testCase.errMessage)
				assert.Nil(t, policy)
			} else {
				require.NoError(t, err)
				require.NotNil(t, policy)
				assert.Equal(t, testCase.guid, policy.GUID)
			}
		})
	}
}

// TestRoutePoliciesClient_Get_ReadOnlyRelationshipsAndLinks proves a
// route-policy response with all three read-only relationships present
// (CF v3 3.226.0: app/space/organization are always present, data null
// unless the source references that resource) decodes without error, that
// the non-null relationship yields its GUID while the null ones decode to
// a present relationship object with nil Data, and that links (self,
// route, and the source-specific link) are reachable via Links["..."].Href.
func TestRoutePoliciesClient_Get_ReadOnlyRelationshipsAndLinks(t *testing.T) {
	t.Parallel()

	const appGUID = "d76446a1-f429-4444-8797-be2f78b75b08"

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testRoutePoliciesPath+"/"+testRoutePolicyGUID, request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{
			"guid": "` + testRoutePolicyGUID + `",
			"created_at": "2026-04-21T10:15:30Z",
			"updated_at": "2026-04-21T10:15:30Z",
			"source": "cf:app:` + appGUID + `",
			"relationships": {
				"route": {"data": {"guid": "` + testRoutePolicyRoute + `"}},
				"app": {"data": {"guid": "` + appGUID + `"}},
				"space": {"data": null},
				"organization": {"data": null}
			},
			"links": {
				"self": {"href": "https://api.example.org/v3/route_policies/` + testRoutePolicyGUID + `"},
				"route": {"href": "https://api.example.org/v3/routes/` + testRoutePolicyRoute + `"},
				"app": {"href": "https://api.example.org/v3/apps/` + appGUID + `"}
			}
		}`))
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	policy, err := client.RoutePolicies().Get(context.Background(), testRoutePolicyGUID)
	require.NoError(t, err)
	require.NotNil(t, policy)

	// The non-null relationship (app, matching the source) yields the GUID.
	require.NotNil(t, policy.Relationships.App)
	require.NotNil(t, policy.Relationships.App.Data)
	assert.Equal(t, appGUID, policy.Relationships.App.Data.GUID)

	// The null relationships (space, organization) decode as present
	// objects with nil Data, not as absent fields.
	require.NotNil(t, policy.Relationships.Space)
	assert.Nil(t, policy.Relationships.Space.Data)
	require.NotNil(t, policy.Relationships.Organization)
	assert.Nil(t, policy.Relationships.Organization.Data)

	// Route is the only always-populated relationship.
	require.NotNil(t, policy.Relationships.Route.Data)
	assert.Equal(t, testRoutePolicyRoute, policy.Relationships.Route.Data.GUID)

	// Links: self and route are always present; app is present because the
	// source references an app.
	require.Contains(t, policy.Links, "self")
	require.Contains(t, policy.Links, "route")
	require.Contains(t, policy.Links, "app")
	assert.Equal(t, "https://api.example.org/v3/route_policies/"+testRoutePolicyGUID, policy.Links["self"].Href)
	assert.Equal(t, "https://api.example.org/v3/routes/"+testRoutePolicyRoute, policy.Links["route"].Href)
	assert.Equal(t, "https://api.example.org/v3/apps/"+appGUID, policy.Links["app"].Href)
	assert.NotContains(t, policy.Links, "space")
	assert.NotContains(t, policy.Links, "organization")
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestRoutePoliciesClient_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		params        *capi.QueryParams
		opts          []capi.RoutePolicyListOption
		expectedQuery string
	}{
		{
			name: "list all",
		},
		{
			name: "list with typed filters",
			opts: []capi.RoutePolicyListOption{
				capi.WithRoutePolicyRouteGUIDs(testRoutePolicyRoute),
				capi.WithRoutePolicySources(capi.RoutePolicySourceAny),
			},
			expectedQuery: "route_guids=route-guid&sources=cf:any",
		},
		{
			name: "list with source guid filter and include",
			opts: []capi.RoutePolicyListOption{
				capi.WithRoutePolicySourceGUIDs(testAppGUID),
				capi.RoutePolicyIncludeSource,
			},
			expectedQuery: "include=source&source_guids=app-guid",
		},
		{
			// CF v3 3.226.0 documents source_guids as a comma-separated
			// list of strings; verify multiple GUIDs join correctly.
			name: "list with multiple source guids",
			opts: []capi.RoutePolicyListOption{
				capi.WithRoutePolicySourceGUIDs("a", "b"),
			},
			expectedQuery: "source_guids=a,b",
		},
		{
			name: "list with space guids and guids",
			opts: []capi.RoutePolicyListOption{
				capi.WithRoutePolicyGUIDs("p1", "p2"),
				capi.WithRoutePolicySpaceGUIDs(testSpaceGUID),
			},
			expectedQuery: "guids=p1,p2&space_guids=space-guid",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, testRoutePoliciesPath, request.URL.Path)
				assert.Equal(t, http.MethodGet, request.Method)

				if testCase.expectedQuery != "" {
					decoded, err := url.QueryUnescape(request.URL.RawQuery)
					assert.NoError(t, err)
					assert.Equal(t, testCase.expectedQuery, decoded)
				}

				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(writer).Encode(capi.ListResponse[capi.RoutePolicy]{
					Pagination: capi.Pagination{TotalResults: 1, TotalPages: 1},
					Resources: []capi.RoutePolicy{
						{
							Resource: capi.Resource{GUID: testRoutePolicyGUID, CreatedAt: time.Now(), UpdatedAt: time.Now()},
							Source:   capi.RoutePolicySourceAny,
							Relationships: capi.RoutePolicyRelationships{
								Route: capi.Relationship{Data: &capi.RelationshipData{GUID: testRoutePolicyRoute}},
							},
						},
					},
				})
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			result, err := client.RoutePolicies().List(context.Background(), testCase.params, testCase.opts...)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Resources, 1)
			assert.Equal(t, testRoutePolicyGUID, result.Resources[0].GUID)
		})
	}
}

func TestRoutePoliciesClient_List_IncludedDecoding(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testRoutePoliciesPath, request.URL.Path)

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{
			"pagination": {"total_results": 1, "total_pages": 1},
			"resources": [
				{
					"guid": "policy-guid",
					"created_at": "2026-04-21T10:15:30Z",
					"updated_at": "2026-04-21T10:15:30Z",
					"source": "cf:app:app-guid",
					"relationships": {
						"route": {"data": {"guid": "route-guid"}},
						"app": {"data": {"guid": "app-guid"}},
						"space": {"data": null},
						"organization": {"data": null}
					}
				}
			],
			"included": {
				"routes": [{"guid": "route-guid", "host": "api", "protocol": "http"}],
				"apps": [{"guid": "app-guid", "name": "frontend"}],
				"spaces": [],
				"organizations": []
			}
		}`))
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	result, err := client.RoutePolicies().List(context.Background(), nil,
		capi.RoutePolicyIncludeRoute, capi.RoutePolicyIncludeSource)
	require.NoError(t, err)

	included, err := capi.RoutePolicyIncludedFrom(result)
	require.NoError(t, err)
	require.NotNil(t, included)
	require.Len(t, included.Routes, 1)
	assert.Equal(t, testRoutePolicyRoute, included.Routes[0].GUID)
	require.Len(t, included.Apps, 1)
	assert.Equal(t, "frontend", included.Apps[0].Name)
	assert.Empty(t, included.Spaces)
	assert.Empty(t, included.Organizations)

	// Relationship data for null sources decodes to nil Data.
	policy := result.Resources[0]
	require.NotNil(t, policy.Relationships.App)
	assert.Equal(t, testAppGUID, policy.Relationships.App.Data.GUID)
	require.NotNil(t, policy.Relationships.Space)
	assert.Nil(t, policy.Relationships.Space.Data)
}

func TestRoutePoliciesClient_Update(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testRoutePoliciesPath+"/"+testRoutePolicyGUID, request.URL.Path)
		assert.Equal(t, http.MethodPatch, request.Method)

		var requestBody capi.RoutePolicyUpdateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)
		assert.Equal(t, "backend", capi.StringValue(requestBody.Metadata.Labels[testTeamLabel]))

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(capi.RoutePolicy{
			Resource: capi.Resource{GUID: testRoutePolicyGUID, CreatedAt: time.Now(), UpdatedAt: time.Now()},
			Source:   capi.RoutePolicySourceAny,
			Metadata: &capi.Metadata{Labels: capi.StringMap(map[string]string{testTeamLabel: "backend"})},
			Relationships: capi.RoutePolicyRelationships{
				Route: capi.Relationship{Data: &capi.RelationshipData{GUID: testRoutePolicyRoute}},
			},
		})
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	policy, err := client.RoutePolicies().Update(context.Background(), testRoutePolicyGUID, &capi.RoutePolicyUpdateRequest{
		Metadata: &capi.Metadata{Labels: capi.StringMap(map[string]string{testTeamLabel: "backend"})},
	})
	require.NoError(t, err)
	require.NotNil(t, policy)
	assert.Equal(t, "backend", capi.StringValue(policy.Metadata.Labels[testTeamLabel]))
}

func TestRoutePoliciesClient_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		guid       string
		statusCode int
		wantErr    bool
		errMessage string
	}{
		{
			name:       "delete route policy",
			guid:       testRoutePolicyGUID,
			statusCode: http.StatusNoContent,
		},
		{
			name:       "policy not found",
			guid:       "missing-guid",
			statusCode: http.StatusNotFound,
			wantErr:    true,
			errMessage: testNotFoundTitle,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, testRoutePoliciesPath+"/"+testCase.guid, request.URL.Path)
				assert.Equal(t, http.MethodDelete, request.Method)

				if testCase.wantErr {
					writer.Header().Set("Content-Type", "application/json")
					writer.WriteHeader(testCase.statusCode)
					_, _ = writer.Write([]byte(testNotFoundErrBody))

					return
				}

				writer.WriteHeader(testCase.statusCode)
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			err = client.RoutePolicies().Delete(context.Background(), testCase.guid)

			if testCase.wantErr {
				require.ErrorContains(t, err, testCase.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
