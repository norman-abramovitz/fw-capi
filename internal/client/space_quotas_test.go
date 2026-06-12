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

func TestSpaceQuotasClient_Create(t *testing.T) {
	t.Parallel()
	RunQuotaCreateTest(t, "space quota create", "/v3/space_quotas", "test-space-quota", 512,
		func(name string) *capi.SpaceQuotaV3CreateRequest {
			totalMemory := 512

			return &capi.SpaceQuotaV3CreateRequest{
				Name: name,
				Apps: &capi.SpaceQuotaApps{
					TotalMemoryInMB: &totalMemory,
				},
				Relationships: capi.SpaceQuotaRelationships{
					Organization: capi.Relationship{
						Data: &capi.RelationshipData{GUID: "org-guid"},
					},
				},
			}
		},
		func(guid, name string, totalMemory int) *capi.SpaceQuotaV3 {
			return &capi.SpaceQuotaV3{
				Resource: capi.Resource{
					GUID:      guid,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: name,
				Apps: &capi.SpaceQuotaApps{
					TotalMemoryInMB: &totalMemory,
				},
			}
		},
		func(c *Client) func(context.Context, *capi.SpaceQuotaV3CreateRequest) (*capi.SpaceQuotaV3, error) {
			return c.SpaceQuotas().Create
		},
		func(quota *capi.SpaceQuotaV3) {
			assert.Equal(t, "quota-guid", quota.GUID)
			assert.Equal(t, "test-space-quota", quota.Name)
			assert.Equal(t, 512, *quota.Apps.TotalMemoryInMB)
		},
	)
}

func TestSpaceQuotasClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/space_quotas/space-quota-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		totalMemory := 1024
		quota := capi.SpaceQuotaV3{
			Resource: capi.Resource{
				GUID: "space-quota-guid",
			},
			Name: "test-space-quota",
			Apps: &capi.SpaceQuotaApps{
				TotalMemoryInMB: &totalMemory,
			},
		}

		_ = json.NewEncoder(writer).Encode(quota)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	quota, err := c.SpaceQuotas().Get(context.Background(), "space-quota-guid")
	require.NoError(t, err)
	assert.Equal(t, "space-quota-guid", quota.GUID)
	assert.Equal(t, "test-space-quota", quota.Name)
	assert.Equal(t, 1024, *quota.Apps.TotalMemoryInMB)
}

// TestSpaceQuotasClient_Get_AppFields verifies that the real CF v3 JSON field names
// per_process_memory_in_mb and per_app_tasks are correctly unmarshalled into the renamed
// Go struct fields PerProcessMemoryInMB and PerAppTasks. A real CF returns these names;
// the old names (total_instance_memory_in_mb / total_app_tasks) were rejected with 422.
func TestSpaceQuotasClient_Get_AppFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/space_quotas/space-quota-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		// Simulate a real CF response with the correct field names.
		_, _ = writer.Write([]byte(`{
			"guid": "space-quota-guid",
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
			"name": "test-space-quota",
			"apps": {
				"total_memory_in_mb": 1024,
				"per_process_memory_in_mb": 256,
				"per_app_tasks": 5,
				"total_instances": 3,
				"log_rate_limit_in_bytes_per_second": 512
			}
		}`))
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	quota, err := c.SpaceQuotas().Get(context.Background(), "space-quota-guid")
	require.NoError(t, err)
	require.NotNil(t, quota.Apps)
	assert.Equal(t, 1024, *quota.Apps.TotalMemoryInMB)
	assert.Equal(t, 256, *quota.Apps.PerProcessMemoryInMB)
	assert.Equal(t, 5, *quota.Apps.PerAppTasks)
}

func TestSpaceQuotasClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/space_quotas", request.URL.Path)
		assert.Equal(t, "GET", request.Method)
		assert.Equal(t, "1", request.URL.Query().Get("page"))
		assert.Equal(t, "10", request.URL.Query().Get("per_page"))

		totalMemory1 := 512
		totalMemory2 := 1024
		response := capi.ListResponse[capi.SpaceQuotaV3]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
			},
			Resources: []capi.SpaceQuotaV3{
				{
					Resource: capi.Resource{GUID: "space-quota-1"},
					Name:     "space-quota-1",
					Apps: &capi.SpaceQuotaApps{
						TotalMemoryInMB: &totalMemory1,
					},
				},
				{
					Resource: capi.Resource{GUID: "space-quota-2"},
					Name:     "space-quota-2",
					Apps: &capi.SpaceQuotaApps{
						TotalMemoryInMB: &totalMemory2,
					},
				},
			},
		}

		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	params := capi.NewQueryParams().WithPage(1).WithPerPage(10)
	result, err := c.SpaceQuotas().List(context.Background(), params)

	require.NoError(t, err)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "space-quota-1", result.Resources[0].Name)
	assert.Equal(t, "space-quota-2", result.Resources[1].Name)
}

//nolint:dupl // Acceptable duplication - each test validates different resource types
func TestSpaceQuotasClient_Update(t *testing.T) {
	t.Parallel()
	RunNameUpdateTest(t, NameUpdateTestCase[capi.SpaceQuotaV3UpdateRequest, capi.SpaceQuotaV3]{
		ResourceType: "space quota",
		ResourceGUID: "space-quota-guid",
		ResourcePath: "/v3/space_quotas/space-quota-guid",
		NewName:      "updated-space-quota",
		CreateRequest: func(name string) *capi.SpaceQuotaV3UpdateRequest {
			return &capi.SpaceQuotaV3UpdateRequest{Name: &name}
		},
		CreateResponse: func(guid, name string) *capi.SpaceQuotaV3 {
			return &capi.SpaceQuotaV3{
				Resource: capi.Resource{GUID: guid},
				Name:     name,
			}
		},
		ExtractName:     func(req *capi.SpaceQuotaV3UpdateRequest) string { return *req.Name },
		ExtractNameResp: func(resp *capi.SpaceQuotaV3) string { return resp.Name },
		UpdateFunc: func(c *Client) func(context.Context, string, *capi.SpaceQuotaV3UpdateRequest) (*capi.SpaceQuotaV3, error) {
			return c.SpaceQuotas().Update
		},
	})
}

// TestSpaceQuotasClient_Delete verifies that DELETE /v3/space_quotas/{guid} returns a
// *Job populated from the Location header (CF v3 async contract: 202 + Location).
func TestSpaceQuotasClient_Delete(t *testing.T) {
	t.Parallel()

	RunJobDeleteTest(t, "space quota delete", "/v3/space_quotas/space-quota-guid", "space_quota.delete",
		func(httpClient *internalhttp.Client) interface{} {
			return NewSpaceQuotasClient(httpClient)
		},
		func(client interface{}) (*capi.Job, error) {
			return client.(*SpaceQuotasClient).Delete(context.Background(), "space-quota-guid")
		},
	)
}

// TestSpaceQuotasClient_DeleteMissingLocation pins the failure mode when CF returns
// 202 without a Location header — must error rather than return a nil-GUID Job.
func TestSpaceQuotasClient_DeleteMissingLocation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/space_quotas/space-quota-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	client := NewSpaceQuotasClient(httpClient)

	job, err := client.Delete(context.Background(), "space-quota-guid")
	require.Error(t, err)
	assert.Nil(t, job)
}

//nolint:dupl // Acceptable duplication - each test validates different relationship endpoints for different resource types
func TestSpaceQuotasClient_ApplyToSpaces(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/space_quotas/space-quota-guid/relationships/spaces", request.URL.Path)
		assert.Equal(t, "POST", request.Method)

		var req capi.ToManyRelationship

		_ = json.NewDecoder(request.Body).Decode(&req)
		assert.Len(t, req.Data, 2)
		assert.Equal(t, "space-1", req.Data[0].GUID)
		assert.Equal(t, "space-2", req.Data[1].GUID)

		response := capi.ToManyRelationship{
			Data: req.Data,
		}

		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	rel, err := c.SpaceQuotas().ApplyToSpaces(context.Background(), "space-quota-guid", []string{"space-1", "space-2"})
	require.NoError(t, err)
	assert.Len(t, rel.Data, 2)
	assert.Equal(t, "space-1", rel.Data[0].GUID)
	assert.Equal(t, "space-2", rel.Data[1].GUID)
}

func TestSpaceQuotasClient_RemoveFromSpace(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/space_quotas/space-quota-guid/relationships/spaces/space-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	err = c.SpaceQuotas().RemoveFromSpace(context.Background(), "space-quota-guid", "space-guid")
	require.NoError(t, err)
}
