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

func boolPtr(b bool) *bool {
	return &b
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestDomainsClient_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		request      *capi.DomainCreateRequest
		response     interface{}
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "create shared domain",
			expectedPath: "/v3/domains",
			statusCode:   http.StatusCreated,
			request: &capi.DomainCreateRequest{
				Name:     "example.com",
				Internal: boolPtr(false),
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						"environment": "production",
					},
				},
			},
			response: capi.Domain{
				Resource: capi.Resource{
					GUID:      "domain-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						"self": capi.Link{
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
				Name:               "example.com",
				Internal:           false,
				SupportedProtocols: []string{"http", "tcp"},
				Relationships:      capi.DomainRelationships{},
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						"environment": "production",
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "create private domain for organization",
			expectedPath: "/v3/domains",
			statusCode:   http.StatusCreated,
			request: &capi.DomainCreateRequest{
				Name: "apps.example.com",
				Relationships: &capi.DomainRelationships{
					Organization: &capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: "org-guid",
						},
					},
				},
			},
			response: capi.Domain{
				Resource: capi.Resource{
					GUID:      "domain-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name:               "apps.example.com",
				Internal:           false,
				SupportedProtocols: []string{"http"},
				Relationships: capi.DomainRelationships{
					Organization: &capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: "org-guid",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "create internal domain",
			expectedPath: "/v3/domains",
			statusCode:   http.StatusCreated,
			request: &capi.DomainCreateRequest{
				Name:     "apps.internal",
				Internal: boolPtr(true),
			},
			response: capi.Domain{
				Resource: capi.Resource{
					GUID:      "domain-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name:               "apps.internal",
				Internal:           true,
				SupportedProtocols: []string{"http"},
				Relationships:      capi.DomainRelationships{},
			},
			wantErr: false,
		},
		{
			name:         "domain already exists",
			expectedPath: "/v3/domains",
			statusCode:   http.StatusUnprocessableEntity,
			request: &capi.DomainCreateRequest{
				Name: "existing.com",
			},
			response: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"code":   10008,
						"title":  "CF-UnprocessableEntity",
						"detail": "Domain name existing.com is already in use",
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
		response     interface{}
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "successful get",
			guid:         "test-domain-guid",
			expectedPath: "/v3/domains/test-domain-guid",
			statusCode:   http.StatusOK,
			response: capi.Domain{
				Resource: capi.Resource{
					GUID:      "test-domain-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name:               "example.com",
				Internal:           false,
				SupportedProtocols: []string{"http", "tcp"},
				Relationships:      capi.DomainRelationships{},
			},
			wantErr: false,
		},
		{
			name:         "domain not found",
			guid:         "non-existent-guid",
			expectedPath: "/v3/domains/non-existent-guid",
			statusCode:   http.StatusNotFound,
			response: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"code":   10010,
						"title":  "CF-ResourceNotFound",
						"detail": "Domain not found",
					},
				},
			},
			wantErr:    true,
			errMessage: "CF-ResourceNotFound",
		},
	}

	runGetTestsForDomains(t, tests)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestDomainsClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/domains", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		// Check query parameters if present
		query := request.URL.Query()
		if names := query.Get("names"); names != "" {
			assert.Equal(t, "example.com,test.com", names)
		}

		if orgGuids := query.Get("organization_guids"); orgGuids != "" {
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
					Name:               "example.com",
					Internal:           false,
					SupportedProtocols: []string{"http"},
				},
				{
					Resource: capi.Resource{
						GUID:      "domain-2",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name:               "apps.internal",
					Internal:           true,
					SupportedProtocols: []string{"http"},
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
	assert.Equal(t, "example.com", result.Resources[0].Name)

	// Test with filters
	params := &capi.QueryParams{
		Filters: map[string][]string{
			"names":              {"example.com", "test.com"},
			"organization_guids": {"org-1", "org-2"},
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
			Labels: map[string]string{
				"environment": "staging",
			},
			Annotations: map[string]string{
				"note": "Updated domain",
			},
		},
	}

	response := &capi.Domain{
		Resource: capi.Resource{
			GUID:      "test-domain-guid",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name: "example.com",
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				"environment": "staging",
			},
			Annotations: map[string]string{
				"note": "Updated domain",
			},
		},
	}

	RunStandardUpdateTest(t, "domain", "test-domain-guid", "/v3/domains/test-domain-guid", request, response,
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
		assert.Equal(t, "DELETE", request.Method)

		writer.Header().Set("Location", "/v3/jobs/job-guid")
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	job, err := c.Domains().Delete(context.Background(), "test-domain-guid")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, "job-guid", job.GUID)
}

func TestDomainsClient_ShareWithOrganization(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/domains/test-domain-guid/relationships/shared_organizations", request.URL.Path)
		assert.Equal(t, "POST", request.Method)

		var requestBody struct {
			Data []capi.RelationshipData `json:"data"`
		}

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)
		assert.Len(t, requestBody.Data, 2)

		response := capi.ToManyRelationship{
			Data: []capi.RelationshipData{
				{GUID: "org-1"},
				{GUID: "org-2"},
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

	relationship, err := c.Domains().ShareWithOrganization(context.Background(), "test-domain-guid", []string{"org-1", "org-2"})
	require.NoError(t, err)
	require.NotNil(t, relationship)
	assert.Len(t, relationship.Data, 3)
}

func TestDomainsClient_UnshareFromOrganization(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/domains/test-domain-guid/relationships/shared_organizations/org-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	err = c.Domains().UnshareFromOrganization(context.Background(), "test-domain-guid", "org-guid")
	require.NoError(t, err)
}

func TestDomainsClient_CheckRouteReservations(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/domains/test-domain-guid/route_reservations", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		// Check query parameters
		query := request.URL.Query()
		assert.Equal(t, "api", query.Get("host"))
		assert.Equal(t, "/v1", query.Get("path"))

		response := capi.RouteReservation{
			MatchingRoute: &capi.Route{
				Resource: capi.Resource{
					GUID: "route-guid",
				},
				Host: "api",
				Path: "/v1",
				URL:  "api.example.com/v1",
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
		Host: "api",
		Path: "/v1",
	}

	reservation, err := client.Domains().CheckRouteReservations(context.Background(), "test-domain-guid", request)
	require.NoError(t, err)
	require.NotNil(t, reservation)
	assert.NotNil(t, reservation.MatchingRoute)
	assert.Equal(t, "route-guid", reservation.MatchingRoute.GUID)
	assert.Equal(t, "api", reservation.MatchingRoute.Host)
}

// runGetTestsForDomains runs domain get tests.
func runGetTestsForDomains(t *testing.T, tests []struct {
	name         string
	guid         string
	response     interface{}
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
				assert.Equal(t, "example.com", domain.Name)
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
		assert.Equal(t, "/v3/domains", request.URL.Path)
		assert.Equal(t, "POST", request.Method)

		var requestBody map[string]interface{}

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

	domain, err := client.Domains().Get(context.Background(), "domain-guid")
	require.NoError(t, err)
	assert.False(t, domain.EnforceRoutePolicies)
	assert.Empty(t, domain.RoutePoliciesScope)
}
