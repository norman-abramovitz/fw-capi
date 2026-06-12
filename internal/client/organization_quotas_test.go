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

func TestOrganizationQuotasClient_Create(t *testing.T) {
	t.Parallel()
	RunQuotaCreateTest(t, "organization quota create", "/v3/organization_quotas", "test-quota", 1024,
		func(name string) *capi.OrganizationQuotaCreateRequest {
			totalMemory := 1024

			return &capi.OrganizationQuotaCreateRequest{
				Name: name,
				Apps: &capi.OrganizationQuotaApps{
					TotalMemoryInMB: &totalMemory,
				},
			}
		},
		func(guid, name string, totalMemory int) *capi.OrganizationQuota {
			return &capi.OrganizationQuota{
				Resource: capi.Resource{
					GUID:      guid,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: name,
				Apps: &capi.OrganizationQuotaApps{
					TotalMemoryInMB: &totalMemory,
				},
			}
		},
		func(c *Client) func(context.Context, *capi.OrganizationQuotaCreateRequest) (*capi.OrganizationQuota, error) {
			return c.OrganizationQuotas().Create
		},
		func(quota *capi.OrganizationQuota) {
			assert.Equal(t, "quota-guid", quota.GUID)
			assert.Equal(t, "test-quota", quota.Name)
			assert.Equal(t, 1024, *quota.Apps.TotalMemoryInMB)
		},
	)
}

func TestOrganizationQuotasClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/organization_quotas/quota-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		totalMemory := 2048
		quota := capi.OrganizationQuota{
			Resource: capi.Resource{
				GUID: "quota-guid",
			},
			Name: "test-quota",
			Apps: &capi.OrganizationQuotaApps{
				TotalMemoryInMB: &totalMemory,
			},
		}

		_ = json.NewEncoder(writer).Encode(quota)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	quota, err := c.OrganizationQuotas().Get(context.Background(), "quota-guid")
	require.NoError(t, err)
	assert.Equal(t, "quota-guid", quota.GUID)
	assert.Equal(t, "test-quota", quota.Name)
	assert.Equal(t, 2048, *quota.Apps.TotalMemoryInMB)
}

// TestOrganizationQuotasClient_Get_AppFields verifies that the real CF v3 JSON field names
// per_process_memory_in_mb and per_app_tasks are correctly unmarshalled into the renamed
// Go struct fields PerProcessMemoryInMB and PerAppTasks. A real CF returns these names;
// the old names (total_instance_memory_in_mb / total_app_tasks) were rejected with 422.
func TestOrganizationQuotasClient_Get_AppFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/organization_quotas/quota-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		// Simulate a real CF response with the correct field names.
		_, _ = writer.Write([]byte(`{
			"guid": "quota-guid",
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z",
			"name": "test-quota",
			"apps": {
				"total_memory_in_mb": 2048,
				"per_process_memory_in_mb": 512,
				"per_app_tasks": 10,
				"total_instances": 5,
				"log_rate_limit_in_bytes_per_second": 1024
			}
		}`))
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	quota, err := c.OrganizationQuotas().Get(context.Background(), "quota-guid")
	require.NoError(t, err)
	require.NotNil(t, quota.Apps)
	assert.Equal(t, 2048, *quota.Apps.TotalMemoryInMB)
	assert.Equal(t, 512, *quota.Apps.PerProcessMemoryInMB)
	assert.Equal(t, 10, *quota.Apps.PerAppTasks)
}

func TestOrganizationQuotasClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/organization_quotas", request.URL.Path)
		assert.Equal(t, "GET", request.Method)
		assert.Equal(t, "1", request.URL.Query().Get("page"))
		assert.Equal(t, "10", request.URL.Query().Get("per_page"))

		totalMemory1 := 1024
		totalMemory2 := 2048
		response := capi.ListResponse[capi.OrganizationQuota]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
			},
			Resources: []capi.OrganizationQuota{
				{
					Resource: capi.Resource{GUID: "quota-1"},
					Name:     "quota-1",
					Apps: &capi.OrganizationQuotaApps{
						TotalMemoryInMB: &totalMemory1,
					},
				},
				{
					Resource: capi.Resource{GUID: "quota-2"},
					Name:     "quota-2",
					Apps: &capi.OrganizationQuotaApps{
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
	result, err := c.OrganizationQuotas().List(context.Background(), params)

	require.NoError(t, err)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "quota-1", result.Resources[0].Name)
	assert.Equal(t, "quota-2", result.Resources[1].Name)
}

//nolint:dupl // Acceptable duplication - each test validates different resource types
func TestOrganizationQuotasClient_Update(t *testing.T) {
	t.Parallel()
	RunNameUpdateTest(t, NameUpdateTestCase[capi.OrganizationQuotaUpdateRequest, capi.OrganizationQuota]{
		ResourceType: "organization quota",
		ResourceGUID: "quota-guid",
		ResourcePath: "/v3/organization_quotas/quota-guid",
		NewName:      "updated-quota",
		CreateRequest: func(name string) *capi.OrganizationQuotaUpdateRequest {
			return &capi.OrganizationQuotaUpdateRequest{Name: &name}
		},
		CreateResponse: func(guid, name string) *capi.OrganizationQuota {
			return &capi.OrganizationQuota{
				Resource: capi.Resource{GUID: guid},
				Name:     name,
			}
		},
		ExtractName:     func(req *capi.OrganizationQuotaUpdateRequest) string { return *req.Name },
		ExtractNameResp: func(resp *capi.OrganizationQuota) string { return resp.Name },
		UpdateFunc: func(c *Client) func(context.Context, string, *capi.OrganizationQuotaUpdateRequest) (*capi.OrganizationQuota, error) {
			return c.OrganizationQuotas().Update
		},
	})
}

// TestOrganizationQuotasClient_Delete verifies that DELETE /v3/organization_quotas/{guid}
// returns a *Job populated from the Location header (CF v3 async contract: 202 + Location).
func TestOrganizationQuotasClient_Delete(t *testing.T) {
	t.Parallel()

	RunJobDeleteTest(t, "organization quota delete", "/v3/organization_quotas/quota-guid", "organization_quota.delete",
		func(httpClient *internalhttp.Client) interface{} {
			return NewOrganizationQuotasClient(httpClient)
		},
		func(client interface{}) (*capi.Job, error) {
			return client.(*OrganizationQuotasClient).Delete(context.Background(), "quota-guid")
		},
	)
}

// TestOrganizationQuotasClient_DeleteMissingLocation pins the failure mode when CF
// returns 202 without a Location header — must error rather than return a nil-GUID Job.
func TestOrganizationQuotasClient_DeleteMissingLocation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/organization_quotas/quota-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	client := NewOrganizationQuotasClient(httpClient)

	job, err := client.Delete(context.Background(), "quota-guid")
	require.Error(t, err)
	assert.Nil(t, job)
}

//nolint:dupl // Acceptable duplication - each test validates different relationship endpoints for different resource types
func TestOrganizationQuotasClient_ApplyToOrganizations(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/organization_quotas/quota-guid/relationships/organizations", request.URL.Path)
		assert.Equal(t, "POST", request.Method)

		var requestBody capi.ToManyRelationship

		_ = json.NewDecoder(request.Body).Decode(&requestBody)
		assert.Len(t, requestBody.Data, 2)
		assert.Equal(t, "org-1", requestBody.Data[0].GUID)
		assert.Equal(t, "org-2", requestBody.Data[1].GUID)

		response := capi.ToManyRelationship{
			Data: requestBody.Data,
		}

		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	rel, err := client.OrganizationQuotas().ApplyToOrganizations(context.Background(), "quota-guid", []string{"org-1", "org-2"})
	require.NoError(t, err)
	assert.Len(t, rel.Data, 2)
	assert.Equal(t, "org-1", rel.Data[0].GUID)
	assert.Equal(t, "org-2", rel.Data[1].GUID)
}
