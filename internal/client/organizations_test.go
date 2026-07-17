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
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrganizationsClient_Create(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/organizations", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var req capi.OrganizationCreateRequest

		_ = json.NewDecoder(request.Body).Decode(&req)
		assert.Equal(t, testOrgNameFixture, req.Name)

		org := capi.Organization{
			Resource: capi.Resource{
				GUID:      testOrgGUID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name:      req.Name,
			Suspended: false,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(writer).Encode(org)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	org, err := c.Organizations().Create(context.Background(), &capi.OrganizationCreateRequest{
		Name: testOrgNameFixture,
	})

	require.NoError(t, err)
	assert.Equal(t, testOrgGUID, org.GUID)
	assert.Equal(t, testOrgNameFixture, org.Name)
}

func TestOrganizationsClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/organizations/org-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		org := capi.Organization{
			Resource: capi.Resource{
				GUID: testOrgGUID,
			},
			Name:      testOrgNameFixture,
			Suspended: false,
		}

		_ = json.NewEncoder(writer).Encode(org)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	org, err := c.Organizations().Get(context.Background(), testOrgGUID)
	require.NoError(t, err)
	assert.Equal(t, testOrgGUID, org.GUID)
	assert.Equal(t, testOrgNameFixture, org.Name)
}

func TestOrganizationsClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/organizations", request.URL.Path)
		assert.Equal(t, "GET", request.Method)
		assert.Equal(t, "2", request.URL.Query().Get("page"))
		assert.Equal(t, "50", request.URL.Query().Get("per_page"))

		response := capi.ListResponse[capi.Organization]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "/v3/organizations?page=1"},
				Last:         capi.Link{Href: "/v3/organizations?page=1"},
			},
			Resources: []capi.Organization{
				{
					Resource: capi.Resource{GUID: testOrgName1},
					Name:     testOrgName1,
				},
				{
					Resource: capi.Resource{GUID: testOrgName2},
					Name:     testOrgName2,
				},
			},
		}

		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	params := capi.NewQueryParams().WithPage(2).WithPerPage(50)
	result, err := c.Organizations().List(context.Background(), params)

	require.NoError(t, err)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, testOrgName1, result.Resources[0].Name)
	assert.Equal(t, testOrgName2, result.Resources[1].Name)
}

//nolint:dupl // Acceptable duplication - each test validates different resource types
func TestOrganizationsClient_Update(t *testing.T) {
	t.Parallel()
	RunNameUpdateTest(t, NameUpdateTestCase[capi.OrganizationUpdateRequest, capi.Organization]{
		ResourceType: testOrganizationType,
		ResourceGUID: testOrgGUID,
		ResourcePath: "/v3/organizations/org-guid",
		NewName:      "updated-org",
		CreateRequest: func(name string) *capi.OrganizationUpdateRequest {
			return &capi.OrganizationUpdateRequest{Name: &name}
		},
		CreateResponse: func(guid, name string) *capi.Organization {
			return &capi.Organization{
				Resource: capi.Resource{GUID: guid},
				Name:     name,
			}
		},
		ExtractName:     func(req *capi.OrganizationUpdateRequest) string { return *req.Name },
		ExtractNameResp: func(resp *capi.Organization) string { return resp.Name },
		UpdateFunc: func(c *Client) func(context.Context, string, *capi.OrganizationUpdateRequest) (*capi.Organization, error) {
			return c.Organizations().Update
		},
	})
}

func TestOrganizationsClient_Delete(t *testing.T) {
	t.Parallel()

	// CF v3 DELETE /v3/organizations/{guid} is async: 202 Accepted, empty
	// body, Location header pointing at /v3/jobs/{jobGuid}.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/organizations/org-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.Header().Set("Location", testJobPath)
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	job, err := c.Organizations().Delete(context.Background(), testOrgGUID)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
}

func TestOrganizationsClient_GetUsageSummary(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/organizations/org-guid/usage_summary", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		summary := capi.OrganizationUsageSummary{}
		summary.UsageSummary.StartedInstances = 5
		summary.UsageSummary.MemoryInMB = 1024

		_ = json.NewEncoder(writer).Encode(summary)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	summary, err := c.Organizations().GetUsageSummary(context.Background(), testOrgGUID)
	require.NoError(t, err)
	assert.Equal(t, 5, summary.UsageSummary.StartedInstances)
	assert.Equal(t, 1024, summary.UsageSummary.MemoryInMB)
}

func TestOrganizationsClient_ListUsers(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/organizations/org-guid/users", request.URL.Path)
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

	result, err := c.Organizations().ListUsers(context.Background(), testOrgGUID, nil)
	require.NoError(t, err)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "user1", result.Resources[0].Username)
	assert.Equal(t, "user2", result.Resources[1].Username)
}

// TestOrganizationsClient_SetDefaultIsolationSegment_Unassign asserts
// the raw wire body for the unassign case: CF only clears an org's
// default segment on an explicit {"data":null}; an empty object {} is
// ignored. Decoding into capi.Relationship can't distinguish the two,
// so this test reads the body verbatim.
func TestOrganizationsClient_SetDefaultIsolationSegment_Unassign(t *testing.T) {
	t.Parallel()

	var rawBody string

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/organizations/org-guid/relationships/default_isolation_segment", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		body, _ := io.ReadAll(request.Body)
		rawBody = string(body)

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":null}`))
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	rel, err := client.Organizations().SetDefaultIsolationSegment(context.Background(), testOrgGUID, "")
	require.NoError(t, err)
	assert.Nil(t, rel.Data)
	assert.JSONEq(t, `{"data":null}`, rawBody)
}
