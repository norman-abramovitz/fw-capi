package client_test

import (
	"context"
	"encoding/json"
	"io"
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

func TestSpacesClient_Create(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/spaces", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var req capi.SpaceCreateRequest

		_ = json.NewDecoder(request.Body).Decode(&req)
		assert.Equal(t, testSpaceNameFixture, req.Name)

		space := capi.Space{
			Resource: capi.Resource{
				GUID:      testSpaceGUID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name: req.Name,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(writer).Encode(space)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	space, err := c.Spaces().Create(context.Background(), &capi.SpaceCreateRequest{
		Name: testSpaceNameFixture,
		Relationships: capi.SpaceRelationships{
			Organization: capi.Relationship{
				Data: &capi.RelationshipData{GUID: testOrgGUID},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, testSpaceGUID, space.GUID)
	assert.Equal(t, testSpaceNameFixture, space.Name)
}

func TestSpacesClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/spaces/space-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		space := capi.Space{
			Resource: capi.Resource{
				GUID: testSpaceGUID,
			},
			Name: testSpaceNameFixture,
		}

		_ = json.NewEncoder(writer).Encode(space)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	space, err := c.Spaces().Get(context.Background(), testSpaceGUID)
	require.NoError(t, err)
	assert.Equal(t, testSpaceGUID, space.GUID)
	assert.Equal(t, testSpaceNameFixture, space.Name)
}

func TestSpacesClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/spaces", request.URL.Path)
		assert.Equal(t, "GET", request.Method)
		assert.Equal(t, "2", request.URL.Query().Get("page"))
		assert.Equal(t, "50", request.URL.Query().Get("per_page"))

		response := capi.ListResponse[capi.Space]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "/v3/spaces?page=1"},
				Last:         capi.Link{Href: "/v3/spaces?page=1"},
			},
			Resources: []capi.Space{
				{
					Resource: capi.Resource{GUID: testSpaceName1},
					Name:     testSpaceName1,
				},
				{
					Resource: capi.Resource{GUID: testSpaceName2},
					Name:     testSpaceName2,
				},
			},
		}

		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	params := capi.NewQueryParams().WithPage(2).WithPerPage(50)
	result, err := c.Spaces().List(context.Background(), params)

	require.NoError(t, err)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, testSpaceName1, result.Resources[0].Name)
	assert.Equal(t, testSpaceName2, result.Resources[1].Name)
}

//nolint:dupl // Acceptable duplication - each test validates different resource types
func TestSpacesClient_Update(t *testing.T) {
	t.Parallel()
	RunNameUpdateTest(t, NameUpdateTestCase[capi.SpaceUpdateRequest, capi.Space]{
		ResourceType: "space",
		ResourceGUID: testSpaceGUID,
		ResourcePath: "/v3/spaces/space-guid",
		NewName:      "updated-space",
		CreateRequest: func(name string) *capi.SpaceUpdateRequest {
			return &capi.SpaceUpdateRequest{Name: &name}
		},
		CreateResponse: func(guid, name string) *capi.Space {
			return &capi.Space{
				Resource: capi.Resource{GUID: guid},
				Name:     name,
			}
		},
		ExtractName:     func(req *capi.SpaceUpdateRequest) string { return *req.Name },
		ExtractNameResp: func(resp *capi.Space) string { return resp.Name },
		UpdateFunc: func(c *Client) func(context.Context, string, *capi.SpaceUpdateRequest) (*capi.Space, error) {
			return c.Spaces().Update
		},
	})
}

func TestSpacesClient_Delete(t *testing.T) {
	t.Parallel()

	// CF v3 DELETE /v3/spaces/{guid} is async: 202 Accepted, empty body,
	// Location header pointing at /v3/jobs/{jobGuid}.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/spaces/space-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.Header().Set("Location", testJobPath)
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	job, err := c.Spaces().Delete(context.Background(), testSpaceGUID)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
}

func TestSpacesClient_GetIsolationSegment(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/spaces/space-guid/relationships/isolation_segment", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		relationship := capi.Relationship{
			Data: &capi.RelationshipData{GUID: "iso-seg-guid"},
		}

		_ = json.NewEncoder(writer).Encode(relationship)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	relationship, err := c.Spaces().GetIsolationSegment(context.Background(), testSpaceGUID)
	require.NoError(t, err)
	assert.NotNil(t, relationship.Data)
	assert.Equal(t, "iso-seg-guid", relationship.Data.GUID)
}

func TestSpacesClient_SetIsolationSegment(t *testing.T) {
	t.Parallel()

	tests := []TestRelationshipOperation{
		{
			Name:         "set isolation segment",
			ResourceGUID: testSpaceGUID,
			TargetGUID:   "new-iso-seg-guid",
			ExpectedPath: "/v3/spaces/space-guid/relationships/isolation_segment",
			RelationshipFunc: func(c *Client) func(context.Context, string, string) (*capi.Relationship, error) {
				return c.Spaces().SetIsolationSegment
			},
		},
	}

	RunRelationshipTests(t, tests)
}

// TestSpacesClient_SetIsolationSegment_Unassign asserts the raw wire
// body for the unassign case: CF only reverts a space to its org's
// default segment on an explicit {"data":null}; an empty object {} is
// ignored. Decoding into capi.Relationship can't distinguish the two,
// so this test reads the body verbatim.
func TestSpacesClient_SetIsolationSegment_Unassign(t *testing.T) {
	t.Parallel()

	var rawBody string

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/spaces/space-guid/relationships/isolation_segment", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		body, _ := io.ReadAll(request.Body)
		rawBody = string(body)

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":null}`))
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	rel, err := client.Spaces().SetIsolationSegment(context.Background(), testSpaceGUID, "")
	require.NoError(t, err)
	assert.Nil(t, rel.Data)
	assert.JSONEq(t, `{"data":null}`, rawBody)
}

func TestSpacesClient_ListUsers(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/spaces/space-guid/users", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		response := capi.ListResponse[capi.User]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
			},
			Resources: []capi.User{
				{
					Resource:         capi.Resource{GUID: testUserName1},
					Username:         "user1",
					PresentationName: "User One",
				},
				{
					Resource:         capi.Resource{GUID: "user-2"},
					Username:         "user2",
					PresentationName: "User Two",
				},
			},
		}

		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	result, err := c.Spaces().ListUsers(context.Background(), testSpaceGUID, nil)
	require.NoError(t, err)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "user1", result.Resources[0].Username)
	assert.Equal(t, "user2", result.Resources[1].Username)
}

func TestSpacesClient_ListManagers(t *testing.T) {
	t.Parallel()
	RunSpaceUserListTest(t, "list managers", "/v3/spaces/space-guid/managers", "manager-1", "manager1",
		func(c *Client) func(context.Context, string, *capi.QueryParams) (*capi.ListResponse[capi.User], error) {
			return c.Spaces().ListManagers
		},
	)
}

func TestSpacesClient_ListDevelopers(t *testing.T) {
	t.Parallel()
	RunSpaceUserListTest(t, "list developers", "/v3/spaces/space-guid/developers", "dev-1", "developer1",
		func(c *Client) func(context.Context, string, *capi.QueryParams) (*capi.ListResponse[capi.User], error) {
			return c.Spaces().ListDevelopers
		},
	)
}

func TestSpacesClient_GetFeature(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/spaces/space-guid/features/ssh", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		feature := capi.SpaceFeature{
			Name:        "ssh",
			Enabled:     true,
			Description: "Enable SSH access to apps",
		}

		_ = json.NewEncoder(writer).Encode(feature)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	feature, err := c.Spaces().GetFeature(context.Background(), testSpaceGUID, "ssh")
	require.NoError(t, err)
	assert.Equal(t, "ssh", feature.Name)
	assert.True(t, feature.Enabled)
}

func TestSpacesClient_UpdateFeature(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/spaces/space-guid/features/ssh", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var req map[string]bool

		_ = json.NewDecoder(request.Body).Decode(&req)
		assert.False(t, req["enabled"])

		feature := capi.SpaceFeature{
			Name:        "ssh",
			Enabled:     false,
			Description: "Enable SSH access to apps",
		}

		_ = json.NewEncoder(writer).Encode(feature)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	feature, err := c.Spaces().UpdateFeature(context.Background(), testSpaceGUID, "ssh", false)
	require.NoError(t, err)
	assert.Equal(t, "ssh", feature.Name)
	assert.False(t, feature.Enabled)
}

func TestSpacesClient_GetQuota(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/spaces/space-guid/quota", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		totalMem := 1024
		totalInstances := 10
		quota := capi.SpaceQuota{
			Resource: capi.Resource{GUID: testQuotaGUID},
			Name:     "test-quota",
			Apps: &capi.AppsQuota{
				TotalMemoryInMB: &totalMem,
				TotalInstances:  &totalInstances,
			},
		}

		_ = json.NewEncoder(writer).Encode(quota)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	quota, err := c.Spaces().GetQuota(context.Background(), testSpaceGUID)
	require.NoError(t, err)
	assert.Equal(t, testQuotaGUID, quota.GUID)
	assert.Equal(t, "test-quota", quota.Name)
	assert.Equal(t, 1024, *quota.Apps.TotalMemoryInMB)
}

func TestSpacesClient_ApplyQuota(t *testing.T) {
	t.Parallel()

	tests := []TestRelationshipOperation{
		{
			Name:         "apply quota",
			ResourceGUID: testSpaceGUID,
			TargetGUID:   testQuotaGUID,
			ExpectedPath: "/v3/spaces/space-guid/relationships/quota",
			RelationshipFunc: func(c *Client) func(context.Context, string, string) (*capi.Relationship, error) {
				return c.Spaces().ApplyQuota
			},
		},
	}

	RunRelationshipTests(t, tests)
}

func TestSpacesClient_RemoveQuota(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/spaces/space-guid/relationships/quota", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var req capi.Relationship

		_ = json.NewDecoder(request.Body).Decode(&req)
		assert.Nil(t, req.Data)

		relationship := capi.Relationship{
			Data: nil,
		}

		_ = json.NewEncoder(writer).Encode(relationship)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	err = c.Spaces().RemoveQuota(context.Background(), testSpaceGUID)
	require.NoError(t, err)
}

func TestSpacesClient_GetWithInclude(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/spaces/space-guid", request.URL.Path)
		assert.Equal(t, testOrganizationType, request.URL.Query().Get("include"))

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
		  "guid": "space-guid", "name": "dev",
		  "included": {"organizations": [{"guid": "org-1", "name": "acme"}]}
		}`))
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	spaces := NewSpacesClient(httpClient)

	space, err := spaces.Get(context.Background(), testSpaceGUID, capi.SpaceIncludeOrganization)
	require.NoError(t, err)
	require.NotNil(t, space.Included)
	assert.Equal(t, testOrgName1, space.Included.Organizations[0].GUID)
}

func TestSpacesClient_SuspendedField(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch request.Method {
		case http.MethodPost:
			assert.Equal(t, "/v3/spaces", request.URL.Path)

			var requestBody map[string]any

			err := json.NewDecoder(request.Body).Decode(&requestBody)
			assert.NoError(t, err)
			assert.Equal(t, true, requestBody["suspended"])

			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write([]byte(`{
			  "guid": "space-guid", "name": "dev", "suspended": true,
			  "relationships": {"organization": {"data": {"guid": "org-guid"}}}
			}`))
		case http.MethodPatch:
			assert.Equal(t, "/v3/spaces/space-guid", request.URL.Path)

			var requestBody map[string]any

			err := json.NewDecoder(request.Body).Decode(&requestBody)
			assert.NoError(t, err)
			assert.Equal(t, false, requestBody["suspended"])

			_, _ = writer.Write([]byte(`{
			  "guid": "space-guid", "name": "dev", "suspended": false,
			  "relationships": {"organization": {"data": {"guid": "org-guid"}}}
			}`))
		}
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	space, err := client.Spaces().Create(context.Background(), &capi.SpaceCreateRequest{
		Name: "dev",
		Relationships: capi.SpaceRelationships{
			Organization: capi.Relationship{Data: &capi.RelationshipData{GUID: testOrgGUID}},
		},
		Suspended: boolPtr(true),
	})
	require.NoError(t, err)
	assert.True(t, space.Suspended)

	space, err = client.Spaces().Update(context.Background(), testSpaceGUID, &capi.SpaceUpdateRequest{
		Suspended: boolPtr(false),
	})
	require.NoError(t, err)
	assert.False(t, space.Suspended)
}
