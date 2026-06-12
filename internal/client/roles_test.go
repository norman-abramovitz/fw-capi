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

func TestRolesClient_Create(t *testing.T) {
	t.Parallel()
	RunRoleCreateTest(t, "organization role create", "organization_auditor", "user-guid", "org-guid", "", "organization_auditor",
		capi.RoleRelationships{
			User: capi.Relationship{
				Data: &capi.RelationshipData{GUID: "user-guid"},
			},
			Organization: &capi.Relationship{
				Data: &capi.RelationshipData{GUID: "org-guid"},
			},
		},
	)
}

func TestRolesClient_CreateSpaceRole(t *testing.T) {
	t.Parallel()
	RunRoleCreateTest(t, "space role create", "space_developer", "user-guid", "", "space-guid", "space_developer",
		capi.RoleRelationships{
			User: capi.Relationship{
				Data: &capi.RelationshipData{GUID: "user-guid"},
			},
			Space: &capi.Relationship{
				Data: &capi.RelationshipData{GUID: "space-guid"},
			},
		},
	)
}

func TestRolesClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/roles/role-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		now := time.Now()
		role := capi.Role{
			Resource: capi.Resource{
				GUID:      "role-guid",
				CreatedAt: now,
				UpdatedAt: now,
			},
			Type: "organization_manager",
			Relationships: capi.RoleRelationships{
				User: capi.Relationship{
					Data: &capi.RelationshipData{
						GUID: "user-guid",
					},
				},
				Organization: &capi.Relationship{
					Data: &capi.RelationshipData{
						GUID: "org-guid",
					},
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(role)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	roles := NewRolesClient(httpClient)

	role, err := roles.Get(context.Background(), "role-guid")
	require.NoError(t, err)
	assert.NotNil(t, role)
	assert.Equal(t, "role-guid", role.GUID)
	assert.Equal(t, "organization_manager", role.Type)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestRolesClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/roles", request.URL.Path)
		assert.Equal(t, "GET", request.Method)
		assert.Equal(t, "organization_auditor,organization_manager", request.URL.Query().Get("types"))
		assert.Equal(t, "org-guid", request.URL.Query().Get("organization_guids"))

		now := time.Now()
		response := capi.ListResponse[capi.Role]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "/v3/roles?page=1"},
				Last:         capi.Link{Href: "/v3/roles?page=1"},
			},
			Resources: []capi.Role{
				{
					Resource: capi.Resource{
						GUID:      "role-guid-1",
						CreatedAt: now,
						UpdatedAt: now,
					},
					Type: "organization_auditor",
					Relationships: capi.RoleRelationships{
						User: capi.Relationship{
							Data: &capi.RelationshipData{
								GUID: "user-guid-1",
							},
						},
						Organization: &capi.Relationship{
							Data: &capi.RelationshipData{
								GUID: "org-guid",
							},
						},
					},
				},
				{
					Resource: capi.Resource{
						GUID:      "role-guid-2",
						CreatedAt: now,
						UpdatedAt: now,
					},
					Type: "organization_manager",
					Relationships: capi.RoleRelationships{
						User: capi.Relationship{
							Data: &capi.RelationshipData{
								GUID: "user-guid-2",
							},
						},
						Organization: &capi.Relationship{
							Data: &capi.RelationshipData{
								GUID: "org-guid",
							},
						},
					},
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	roles := NewRolesClient(httpClient)

	params := &capi.QueryParams{
		Filters: map[string][]string{
			"types":              {"organization_auditor", "organization_manager"},
			"organization_guids": {"org-guid"},
		},
	}

	list, err := roles.List(context.Background(), params)
	require.NoError(t, err)
	assert.NotNil(t, list)
	assert.Equal(t, 2, list.Pagination.TotalResults)
	assert.Len(t, list.Resources, 2)
	assert.Equal(t, "role-guid-1", list.Resources[0].GUID)
	assert.Equal(t, "organization_auditor", list.Resources[0].Type)
	assert.Equal(t, "role-guid-2", list.Resources[1].GUID)
	assert.Equal(t, "organization_manager", list.Resources[1].Type)
}

func TestRolesClient_Delete(t *testing.T) {
	t.Parallel()

	// CF v3 DELETE /v3/roles/{guid} returns 202 Accepted with a
	// Location header pointing at /v3/jobs/{guid}. Mirror Apps.Delete
	// behaviour: extract the job GUID from the Location header and
	// return a Job populated with that GUID.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/roles/role-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.Header().Set("Location", "https://api.example.com/v3/jobs/job-guid-123")
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	roles := NewRolesClient(httpClient)

	job, err := roles.Delete(context.Background(), "role-guid")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, "job-guid-123", job.GUID)
}

// TestRolesClient_DeleteMissingLocation pins the failure mode when CF
// returns 202 without a Location header — should error, not return a
// nil-GUID Job that callers might poll for ever.
func TestRolesClient_DeleteMissingLocation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	roles := NewRolesClient(httpClient)

	job, err := roles.Delete(context.Background(), "role-guid")
	require.Error(t, err)
	assert.Nil(t, job)
}

func TestRolesClient_GetNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/roles/role-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	roles := NewRolesClient(httpClient)

	role, err := roles.Get(context.Background(), "role-guid")
	require.Error(t, err)
	assert.Nil(t, role)
}

func TestRolesClient_GetWithIncludes(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/roles/role-guid", request.URL.Path)
		assert.Equal(t, "space,organization", request.URL.Query().Get("include"))

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
		  "guid": "role-guid",
		  "type": "space_developer",
		  "included": {
		    "spaces": [{"guid": "space-1", "name": "dev"}],
		    "organizations": [{"guid": "org-1", "name": "acme"}]
		  }
		}`))
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	roles := NewRolesClient(httpClient)

	role, err := roles.Get(context.Background(), "role-guid",
		capi.RoleIncludeSpace, capi.RoleIncludeOrganization)
	require.NoError(t, err)
	require.NotNil(t, role.Included)
	assert.Equal(t, "space-1", role.Included.Spaces[0].GUID)
	assert.Equal(t, "org-1", role.Included.Organizations[0].GUID)
}

func TestRolesClient_ListWithIncludes(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/roles", request.URL.Path)
		assert.Equal(t, "space,organization", request.URL.Query().Get("include"))
		assert.Equal(t, "user-1", request.URL.Query().Get("user_guids"))

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
		  "pagination": {"total_results": 1, "total_pages": 1},
		  "resources": [{"guid": "role-1", "type": "space_developer"}],
		  "included": {
		    "spaces": [{"guid": "space-1",
		      "relationships": {"organization": {"data": {"guid": "org-1"}}}}],
		    "organizations": [{"guid": "org-1", "name": "acme"}]
		  }
		}`))
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	roles := NewRolesClient(httpClient)

	params := capi.NewQueryParams()
	params.Filters["user_guids"] = []string{"user-1"}

	list, err := roles.List(context.Background(), params,
		capi.RoleIncludeSpace, capi.RoleIncludeOrganization)
	require.NoError(t, err)

	incl, err := capi.RoleIncludedFrom(list)
	require.NoError(t, err)
	require.NotNil(t, incl.Spaces[0].Relationships.Organization.Data)
	assert.Equal(t, "org-1", incl.Spaces[0].Relationships.Organization.Data.GUID)
}
