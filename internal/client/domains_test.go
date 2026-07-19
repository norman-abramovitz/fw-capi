package client_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/fivetwenty-io/capi/v3/internal/client"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
)

// Test constants for domain and route path tests. testV1Path is shared with
// routes_test.go since both exercise route path filtering.
const (
	testDomainsPath        = "/v3/domains"
	testV1Path             = "/v1"
	testExampleComDomain   = "example.com"
	testAppsInternalDomain = "apps.internal"
	testDomainFixtureGUID  = "test-domain-guid"
)

func boolPtr(b bool) *bool {
	return &b
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestDomainsClient_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		request      *capi.DomainCreateRequest
		response     any
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "create shared domain",
			expectedPath: testDomainsPath,
			statusCode:   http.StatusCreated,
			request: &capi.DomainCreateRequest{
				Name:     testExampleComDomain,
				Internal: boolPtr(false),
				Metadata: &capi.Metadata{
					Labels: capi.StringMap(map[string]string{
						testEnvironmentLabelKey: testProductionLabel,
					}),
				},
			},
			response: capi.Domain{
				Resource: capi.Resource{
					GUID:      testDomainGUID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						testSelfKey: capi.Link{
							Href: "https://api.example.org/v3/domains/domain-guid",
						},
						"route_reservations": capi.Link{
							Href: "https://api.example.org/v3/domains/domain-guid/route_reservations",
						},
						"shared_organizations": capi.Link{
							Href: "https://api.example.org/v3/domains/domain-guid/relationships/shared_organizations",
						},
					},
				},
				Name:               testExampleComDomain,
				Internal:           false,
				SupportedProtocols: []string{testHTTPProtocol, testTCPProtocol},
				Relationships:      capi.DomainRelationships{},
				Metadata: &capi.Metadata{
					Labels: capi.StringMap(map[string]string{
						testEnvironmentLabelKey: testProductionLabel,
					}),
				},
			},
			wantErr: false,
		},
		{
			name:         "create private domain for organization",
			expectedPath: testDomainsPath,
			statusCode:   http.StatusCreated,
			request: &capi.DomainCreateRequest{
				Name: "apps.example.com",
				Relationships: &capi.DomainRelationships{
					Organization: &capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: testOrgGUID,
						},
					},
				},
			},
			response: capi.Domain{
				Resource: capi.Resource{
					GUID:      testDomainGUID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name:               "apps.example.com",
				Internal:           false,
				SupportedProtocols: []string{testHTTPProtocol},
				Relationships: capi.DomainRelationships{
					Organization: &capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: testOrgGUID,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "create internal domain",
			expectedPath: testDomainsPath,
			statusCode:   http.StatusCreated,
			request: &capi.DomainCreateRequest{
				Name:     testAppsInternalDomain,
				Internal: boolPtr(true),
			},
			response: capi.Domain{
				Resource: capi.Resource{
					GUID:      testDomainGUID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name:               testAppsInternalDomain,
				Internal:           true,
				SupportedProtocols: []string{testHTTPProtocol},
				Relationships:      capi.DomainRelationships{},
			},
			wantErr: false,
		},
		{
			name:         "domain already exists",
			expectedPath: testDomainsPath,
			statusCode:   http.StatusUnprocessableEntity,
			request: &capi.DomainCreateRequest{
				Name: "existing.com",
			},
			response: map[string]any{
				testErrorsKey: []map[string]any{
					{
						testCodeKey:   10008,
						testTitleKey:  testUnprocessableTitle,
						testDetailKey: "Domain name existing.com is already in use",
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

				var requestBody capi.DomainCreateRequest

				err := json.NewDecoder(request.Body).Decode(&requestBody)
				assert.NoError(t, err)

				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(testCase.statusCode)
				_ = json.NewEncoder(writer).Encode(testCase.response)
			}))
			defer server.Close()

			c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			domain, err := c.Domains().Create(context.Background(), testCase.request)

			if testCase.wantErr {
				require.ErrorContains(t, err, testCase.errMessage)
				assert.Nil(t, domain)
			} else {
				require.NoError(t, err)
				require.NotNil(t, domain)
				assert.NotEmpty(t, domain.GUID)
				assert.Equal(t, testCase.request.Name, domain.Name)
			}
		})
	}
}

func TestDomainsClient_Get(t *testing.T) {
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
			guid:         testDomainFixtureGUID,
			expectedPath: "/v3/domains/test-domain-guid",
			statusCode:   http.StatusOK,
			response: capi.Domain{
				Resource: capi.Resource{
					GUID:      testDomainFixtureGUID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name:               testExampleComDomain,
				Internal:           false,
				SupportedProtocols: []string{testHTTPProtocol, testTCPProtocol},
				Relationships:      capi.DomainRelationships{},
			},
			wantErr: false,
		},
		{
			name:         "domain not found",
			guid:         testNonExistentGUID,
			expectedPath: "/v3/domains/non-existent-guid",
			statusCode:   http.StatusNotFound,
			response: map[string]any{
				testErrorsKey: []map[string]any{
					{
						testCodeKey:   10010,
						testTitleKey:  testNotFoundTitle,
						testDetailKey: "Domain not found",
					},
				},
			},
			wantErr:    true,
			errMessage: testNotFoundTitle,
		},
	}

	runGetTestsForDomains(t, tests)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestDomainsClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testDomainsPath, request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		// Check query parameters if present
		query := request.URL.Query()
		if names := query.Get(testNamesParam); names != "" {
			assert.Equal(t, "example.com,test.com", names)
		}

		if orgGuids := query.Get(testOrgGUIDsParam); orgGuids != "" {
			assert.Equal(t, "org-1,org-2", orgGuids)
		}

		response := capi.ListResponse[capi.Domain]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "https://api.example.org/v3/domains?page=1"},
				Last:         capi.Link{Href: "https://api.example.org/v3/domains?page=1"},
				Next:         nil,
				Previous:     nil,
			},
			Resources: []capi.Domain{
				{
					Resource: capi.Resource{
						GUID:      "domain-1",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name:               testExampleComDomain,
					Internal:           false,
					SupportedProtocols: []string{testHTTPProtocol},
				},
				{
					Resource: capi.Resource{
						GUID:      "domain-2",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name:               testAppsInternalDomain,
					Internal:           true,
					SupportedProtocols: []string{testHTTPProtocol},
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
	result, err := client.Domains().List(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.TotalResults)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "domain-1", result.Resources[0].GUID)
	assert.Equal(t, testExampleComDomain, result.Resources[0].Name)

	// Test with filters
	params := &capi.QueryParams{
		Filters: map[string][]string{
			testNamesParam:    {testExampleComDomain, "test.com"},
			testOrgGUIDsParam: {testOrgName1, testOrgName2},
		},
	}
	result, err = client.Domains().List(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
}

//nolint:dupl // Acceptable duplication - each test validates different endpoints with different request/response types
func TestDomainsClient_Update(t *testing.T) {
	t.Parallel()

	request := &capi.DomainUpdateRequest{
		Metadata: &capi.Metadata{
			Labels: capi.StringMap(map[string]string{
				testEnvironmentLabelKey: testStagingLabel,
			}),
			Annotations: capi.StringMap(map[string]string{
				testNoteAnnotationKey: "Updated domain",
			}),
		},
	}

	response := &capi.Domain{
		Resource: capi.Resource{
			GUID:      testDomainFixtureGUID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name: testExampleComDomain,
		Metadata: &capi.Metadata{
			Labels: capi.StringMap(map[string]string{
				testEnvironmentLabelKey: testStagingLabel,
			}),
			Annotations: capi.StringMap(map[string]string{
				testNoteAnnotationKey: "Updated domain",
			}),
		},
	}

	RunStandardUpdateTest(t, "domain", testDomainFixtureGUID, "/v3/domains/test-domain-guid", request, response,
		func(c *Client) func(context.Context, string, *capi.DomainUpdateRequest) (*capi.Domain, error) {
			return c.Domains().Update
		})
}

func TestDomainsClient_Delete(t *testing.T) {
	t.Parallel()

	// CF v3 DELETE /v3/domains/{guid} is async: 202 Accepted, empty body,
	// Location header pointing at /v3/jobs/{jobGuid}.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/domains/test-domain-guid", request.URL.Path)
		assert.Equal(t, http.MethodDelete, request.Method)

		writer.Header().Set("Location", testJobPath)
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	job, err := c.Domains().Delete(context.Background(), testDomainFixtureGUID)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
}

func TestDomainsClient_ShareWithOrganization(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/domains/test-domain-guid/relationships/shared_organizations", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var requestBody struct {
			Data []capi.RelationshipData `json:"data"`
		}

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)
		assert.Len(t, requestBody.Data, 2)

		response := capi.ToManyRelationship{
			Data: []capi.RelationshipData{
				{GUID: testOrgName1},
				{GUID: testOrgName2},
				{GUID: "org-3"},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	relationship, err := c.Domains().ShareWithOrganization(context.Background(), testDomainFixtureGUID, []string{testOrgName1, testOrgName2})
	require.NoError(t, err)
	require.NotNil(t, relationship)
	assert.Len(t, relationship.Data, 3)
}

func TestDomainsClient_UnshareFromOrganization(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/domains/test-domain-guid/relationships/shared_organizations/org-guid", request.URL.Path)
		assert.Equal(t, http.MethodDelete, request.Method)
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	err = c.Domains().UnshareFromOrganization(context.Background(), testDomainFixtureGUID, testOrgGUID)
	require.NoError(t, err)
}

func TestDomainsClient_CheckRouteReservations(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/domains/test-domain-guid/route_reservations", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		// Check query parameters
		query := request.URL.Query()
		assert.Equal(t, testAPIHost, query.Get("host"))
		assert.Equal(t, testV1Path, query.Get("path"))

		response := capi.RouteReservation{
			MatchingRoute: &capi.Route{
				Resource: capi.Resource{
					GUID: testRoutePolicyRoute,
				},
				Host: testAPIHost,
				Path: testV1Path,
				URL:  testAPIExampleV1URL,
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	request := &capi.RouteReservationRequest{
		Host: testAPIHost,
		Path: testV1Path,
	}

	reservation, err := client.Domains().CheckRouteReservations(context.Background(), testDomainFixtureGUID, request)
	require.NoError(t, err)
	require.NotNil(t, reservation)
	assert.NotNil(t, reservation.MatchingRoute)
	assert.Equal(t, testRoutePolicyRoute, reservation.MatchingRoute.GUID)
	assert.Equal(t, testAPIHost, reservation.MatchingRoute.Host)
}

// runGetTestsForDomains runs domain get tests.
func runGetTestsForDomains(t *testing.T, tests []struct {
	name         string
	guid         string
	response     any
	statusCode   int
	expectedPath string
	wantErr      bool
	errMessage   string
}) {
	t.Helper()

	for _, testCase := range tests {
		RunGetTestWithValidation(t, testCase.name, testCase.guid, testCase.expectedPath, testCase.statusCode, testCase.response, testCase.wantErr, testCase.errMessage, func(c *Client, guid string) error {
			domain, err := c.Domains().Get(context.Background(), guid)
			if err == nil {
				assert.Equal(t, guid, domain.GUID)
				assert.Equal(t, testExampleComDomain, domain.Name)
			}

			if err != nil {
				return fmt.Errorf("failed to get domain: %w", err)
			}

			return nil
		})
	}
}

func TestDomainsClient_Create_IdentityAware(t *testing.T) {
	t.Parallel()

	scope := capi.RoutePoliciesScopeOrg

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testDomainsPath, request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var requestBody map[string]any

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)
		assert.Equal(t, true, requestBody["enforce_route_policies"])
		assert.Equal(t, "org", requestBody["route_policies_scope"])

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{
		  "guid": "domain-guid",
		  "name": "apps.identity",
		  "internal": false,
		  "supported_protocols": ["http"],
		  "enforce_route_policies": true,
		  "route_policies_scope": "org"
		}`))
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	domain, err := client.Domains().Create(context.Background(), &capi.DomainCreateRequest{
		Name:                 "apps.identity",
		EnforceRoutePolicies: boolPtr(true),
		RoutePoliciesScope:   &scope,
	})
	require.NoError(t, err)
	require.NotNil(t, domain)
	assert.True(t, domain.EnforceRoutePolicies)
	assert.Equal(t, capi.RoutePoliciesScopeOrg, domain.RoutePoliciesScope)
}

func TestDomainsClient_Get_OmitsRoutePolicyFieldsByDefault(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
		  "guid": "domain-guid",
		  "name": "example.com",
		  "internal": false,
		  "supported_protocols": ["http"]
		}`))
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	domain, err := client.Domains().Get(context.Background(), testDomainGUID)
	require.NoError(t, err)
	assert.False(t, domain.EnforceRoutePolicies)
	assert.Empty(t, domain.RoutePoliciesScope)
}
