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

func TestIsolationSegmentsClient_Create(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/isolation_segments", request.URL.Path)
		assert.Equal(t, "POST", request.Method)

		var req capi.IsolationSegmentCreateRequest

		err := json.NewDecoder(request.Body).Decode(&req)
		assert.NoError(t, err)

		assert.Equal(t, "my-segment", req.Name)
		assert.NotNil(t, req.Metadata)
		assert.Equal(t, "value1", req.Metadata.Labels["key1"])

		now := time.Now()
		isolationSegment := capi.IsolationSegment{
			Resource: capi.Resource{
				GUID:      "segment-guid",
				CreatedAt: now,
				UpdatedAt: now,
				Links: capi.Links{
					"self": capi.Link{
						Href: "/v3/isolation_segments/segment-guid",
					},
					"organizations": capi.Link{
						Href: "/v3/isolation_segments/segment-guid/organizations",
					},
				},
			},
			Name:     req.Name,
			Metadata: req.Metadata,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(writer).Encode(isolationSegment)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	isolationSegments := NewIsolationSegmentsClient(httpClient)

	request := &capi.IsolationSegmentCreateRequest{
		Name: "my-segment",
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				"key1": "value1",
			},
			Annotations: map[string]string{},
		},
	}

	isolationSegmentResult, err := isolationSegments.Create(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, isolationSegmentResult)
	assert.Equal(t, "segment-guid", isolationSegmentResult.GUID)
	assert.Equal(t, "my-segment", isolationSegmentResult.Name)
	assert.Equal(t, "value1", isolationSegmentResult.Metadata.Labels["key1"])
}

func TestIsolationSegmentsClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/isolation_segments/segment-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		now := time.Now()
		isolationSegment := capi.IsolationSegment{
			Resource: capi.Resource{
				GUID:      "segment-guid",
				CreatedAt: now,
				UpdatedAt: now,
				Links: capi.Links{
					"self": capi.Link{
						Href: "/v3/isolation_segments/segment-guid",
					},
					"organizations": capi.Link{
						Href: "/v3/isolation_segments/segment-guid/organizations",
					},
				},
			},
			Name: "my-segment",
			Metadata: &capi.Metadata{
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(isolationSegment)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	isolationSegments := NewIsolationSegmentsClient(httpClient)

	isolationSegmentResult, err := isolationSegments.Get(context.Background(), "segment-guid")
	require.NoError(t, err)
	assert.NotNil(t, isolationSegmentResult)
	assert.Equal(t, "segment-guid", isolationSegmentResult.GUID)
	assert.Equal(t, "my-segment", isolationSegmentResult.Name)
}

func TestIsolationSegmentsClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/isolation_segments", request.URL.Path)
		assert.Equal(t, "GET", request.Method)
		assert.Equal(t, "segment1,segment2", request.URL.Query().Get("names"))
		assert.Equal(t, "org-guid", request.URL.Query().Get("organization_guids"))

		now := time.Now()
		response := capi.ListResponse[capi.IsolationSegment]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "/v3/isolation_segments?page=1"},
				Last:         capi.Link{Href: "/v3/isolation_segments?page=1"},
			},
			Resources: []capi.IsolationSegment{
				{
					Resource: capi.Resource{
						GUID:      "segment-guid-1",
						CreatedAt: now,
						UpdatedAt: now,
					},
					Name: "segment1",
				},
				{
					Resource: capi.Resource{
						GUID:      "segment-guid-2",
						CreatedAt: now,
						UpdatedAt: now,
					},
					Name: "segment2",
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	isolationSegments := NewIsolationSegmentsClient(httpClient)

	params := &capi.QueryParams{
		Filters: map[string][]string{
			"names":              {"segment1", "segment2"},
			"organization_guids": {"org-guid"},
		},
	}

	list, err := isolationSegments.List(context.Background(), params)
	require.NoError(t, err)
	assert.NotNil(t, list)
	assert.Equal(t, 2, list.Pagination.TotalResults)
	assert.Len(t, list.Resources, 2)
	assert.Equal(t, "segment1", list.Resources[0].Name)
	assert.Equal(t, "segment2", list.Resources[1].Name)
}

func TestIsolationSegmentsClient_Update(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/isolation_segments/segment-guid", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var requestBody capi.IsolationSegmentUpdateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.NotNil(t, requestBody.Name)
		assert.Equal(t, "updated-segment", *requestBody.Name)
		assert.NotNil(t, requestBody.Metadata)
		assert.Equal(t, "value2", requestBody.Metadata.Labels["key2"])

		now := time.Now()
		isolationSegment := capi.IsolationSegment{
			Resource: capi.Resource{
				GUID:      "segment-guid",
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:     *requestBody.Name,
			Metadata: requestBody.Metadata,
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(isolationSegment)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	isolationSegments := NewIsolationSegmentsClient(httpClient)

	name := "updated-segment"
	request := &capi.IsolationSegmentUpdateRequest{
		Name: &name,
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				"key2": "value2",
			},
			Annotations: map[string]string{},
		},
	}

	isolationSegmentResult, err := isolationSegments.Update(context.Background(), "segment-guid", request)
	require.NoError(t, err)
	assert.NotNil(t, isolationSegmentResult)
	assert.Equal(t, "segment-guid", isolationSegmentResult.GUID)
	assert.Equal(t, "updated-segment", isolationSegmentResult.Name)
	assert.Equal(t, "value2", isolationSegmentResult.Metadata.Labels["key2"])
}

func TestIsolationSegmentsClient_Delete(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/isolation_segments/segment-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	isolationSegments := NewIsolationSegmentsClient(httpClient)

	err := isolationSegments.Delete(context.Background(), "segment-guid")
	require.NoError(t, err)
}

func TestIsolationSegmentsClient_EntitleOrganizations(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/isolation_segments/segment-guid/relationships/organizations", request.URL.Path)
		assert.Equal(t, "POST", request.Method)

		var requestBody capi.IsolationSegmentEntitleOrganizationsRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.Len(t, requestBody.Data, 2)
		assert.Equal(t, "org-guid-1", requestBody.Data[0].GUID)
		assert.Equal(t, "org-guid-2", requestBody.Data[1].GUID)

		response := requestBody

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	isolationSegments := NewIsolationSegmentsClient(httpClient)

	relationship, err := isolationSegments.EntitleOrganizations(context.Background(), "segment-guid", []string{"org-guid-1", "org-guid-2"})
	require.NoError(t, err)
	assert.NotNil(t, relationship)
	assert.Len(t, relationship.Data, 2)
	assert.Equal(t, "org-guid-1", relationship.Data[0].GUID)
}

func TestIsolationSegmentsClient_RevokeOrganization(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/isolation_segments/segment-guid/relationships/organizations/org-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	isolationSegments := NewIsolationSegmentsClient(httpClient)

	err := isolationSegments.RevokeOrganization(context.Background(), "segment-guid", "org-guid")
	require.NoError(t, err)
}

func TestIsolationSegmentsClient_ListOrganizations(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/isolation_segments/segment-guid/organizations", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		now := time.Now()
		response := capi.ListResponse[capi.Organization]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "/v3/isolation_segments/segment-guid/organizations?page=1"},
				Last:         capi.Link{Href: "/v3/isolation_segments/segment-guid/organizations?page=1"},
			},
			Resources: []capi.Organization{
				{
					Resource: capi.Resource{
						GUID:      "org-guid-1",
						CreatedAt: now,
						UpdatedAt: now,
					},
					Name:      "org1",
					Suspended: false,
				},
				{
					Resource: capi.Resource{
						GUID:      "org-guid-2",
						CreatedAt: now,
						UpdatedAt: now,
					},
					Name:      "org2",
					Suspended: false,
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	isolationSegments := NewIsolationSegmentsClient(httpClient)

	list, err := isolationSegments.ListOrganizations(context.Background(), "segment-guid", nil)
	require.NoError(t, err)
	assert.NotNil(t, list)
	assert.Equal(t, 2, list.Pagination.TotalResults)
	assert.Len(t, list.Resources, 2)
	assert.Equal(t, "org1", list.Resources[0].Name)
	assert.Equal(t, "org2", list.Resources[1].Name)
}

func TestIsolationSegmentsClient_ListSpaces(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/isolation_segments/segment-guid/relationships/spaces", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		now := time.Now()
		response := capi.ListResponse[capi.Space]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "/v3/isolation_segments/segment-guid/relationships/spaces?page=1"},
				Last:         capi.Link{Href: "/v3/isolation_segments/segment-guid/relationships/spaces?page=1"},
			},
			Resources: []capi.Space{
				{
					Resource: capi.Resource{
						GUID:      "space-guid-1",
						CreatedAt: now,
						UpdatedAt: now,
					},
					Name: "space1",
				},
				{
					Resource: capi.Resource{
						GUID:      "space-guid-2",
						CreatedAt: now,
						UpdatedAt: now,
					},
					Name: "space2",
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	isolationSegments := NewIsolationSegmentsClient(httpClient)

	list, err := isolationSegments.ListSpaces(context.Background(), "segment-guid", nil)
	require.NoError(t, err)
	assert.NotNil(t, list)
	assert.Equal(t, 2, list.Pagination.TotalResults)
	assert.Len(t, list.Resources, 2)
	assert.Equal(t, "space1", list.Resources[0].Name)
	assert.Equal(t, "space2", list.Resources[1].Name)
}

func TestIsolationSegmentsClient_ListOrganizationsWithParams(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/isolation_segments/iso-guid/organizations", request.URL.Path)
		assert.Equal(t, "org-1,org-2", request.URL.Query().Get("guids"))
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"pagination": {"total_results": 0}, "resources": []}`))
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	segments := NewIsolationSegmentsClient(httpClient)

	params := capi.NewQueryParams()
	params.Filters["guids"] = []string{"org-1", "org-2"}

	_, err := segments.ListOrganizations(context.Background(), "iso-guid", params)
	require.NoError(t, err)
}
