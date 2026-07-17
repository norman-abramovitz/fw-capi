package capi_test

import (
	"context"
	"testing"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockPaginationClient implements PaginationClient for testing.
type MockPaginationClient struct {
	pages map[int]*capi.ListResponse[TestResource]
}

type TestResource struct {
	ID   string
	Name string
}

func (m *MockPaginationClient) ListWithPath(ctx context.Context, path string, params *capi.QueryParams) (*capi.ListResponse[TestResource], error) {
	page := 1
	if params != nil && params.Page > 0 {
		page = params.Page
	}

	response, ok := m.pages[page]
	if !ok {
		return &capi.ListResponse[TestResource]{
			Pagination: capi.Pagination{
				TotalResults: 0,
				TotalPages:   0,
			},
			Resources: []TestResource{},
		}, nil
	}

	return response, nil
}

func TestPaginationIterator_HasNext(t *testing.T) {
	t.Parallel()

	client := &MockPaginationClient{
		pages: map[int]*capi.ListResponse[TestResource]{
			1: {
				Pagination: capi.Pagination{
					TotalResults: 3,
					TotalPages:   2,
					Next: &capi.Link{
						Href: testPagedPath,
					},
				},
				Resources: []TestResource{
					{ID: "1", Name: testResourceName1},
					{ID: "2", Name: testResourceName2},
				},
			},
			2: {
				Pagination: capi.Pagination{
					TotalResults: 3,
					TotalPages:   2,
					Previous: &capi.Link{
						Href: "/test?page=1",
					},
				},
				Resources: []TestResource{
					{ID: "3", Name: testResourceName3},
				},
			},
		},
	}

	ctx := context.Background()
	iterator := capi.NewPaginationIterator(client, "/test", nil)

	// Should have next before any fetch
	assert.True(t, iterator.HasNext())

	// Fetch first item
	item1, err := iterator.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, "1", item1.ID)

	// Should still have next
	assert.True(t, iterator.HasNext())

	// Fetch second item
	item2, err := iterator.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, "2", item2.ID)

	// Should still have next (page 2)
	assert.True(t, iterator.HasNext())

	// Fetch third item
	item3, err := iterator.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, "3", item3.ID)

	// Should not have next
	assert.False(t, iterator.HasNext())
}

func TestPaginationIterator_All(t *testing.T) {
	t.Parallel()

	client := &MockPaginationClient{
		pages: map[int]*capi.ListResponse[TestResource]{
			1: {
				Pagination: capi.Pagination{
					TotalResults: 3,
					TotalPages:   2,
					Next: &capi.Link{
						Href: "/test?page=2&per_page=2",
					},
				},
				Resources: []TestResource{
					{ID: "1", Name: testResourceName1},
					{ID: "2", Name: testResourceName2},
				},
			},
			2: {
				Pagination: capi.Pagination{
					TotalResults: 3,
					TotalPages:   2,
				},
				Resources: []TestResource{
					{ID: "3", Name: testResourceName3},
				},
			},
		},
	}

	ctx := context.Background()
	iterator := capi.NewPaginationIterator(client, "/test", nil)

	allResources, err := iterator.All(ctx)
	require.NoError(t, err)
	assert.Len(t, allResources, 3)
	assert.Equal(t, "1", allResources[0].ID)
	assert.Equal(t, "2", allResources[1].ID)
	assert.Equal(t, "3", allResources[2].ID)
}

func TestPaginationIterator_ForEach(t *testing.T) {
	t.Parallel()

	client := &MockPaginationClient{
		pages: map[int]*capi.ListResponse[TestResource]{
			1: {
				Pagination: capi.Pagination{
					TotalResults: 2,
					TotalPages:   1,
				},
				Resources: []TestResource{
					{ID: "1", Name: testResourceName1},
					{ID: "2", Name: testResourceName2},
				},
			},
		},
	}

	ctx := context.Background()
	iterator := capi.NewPaginationIterator(client, "/test", nil)

	var collected []string

	err := iterator.ForEach(ctx, func(resource TestResource) error {
		collected = append(collected, resource.ID)

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"1", "2"}, collected)
}

func TestFetchAllPages(t *testing.T) {
	t.Parallel()

	client := &MockPaginationClient{
		pages: map[int]*capi.ListResponse[TestResource]{
			1: {
				Pagination: capi.Pagination{
					TotalResults: 5,
					TotalPages:   3,
					Next: &capi.Link{
						Href: testPagedPath,
					},
				},
				Resources: []TestResource{
					{ID: "1", Name: testResourceName1},
					{ID: "2", Name: testResourceName2},
				},
			},
			2: {
				Pagination: capi.Pagination{
					TotalResults: 5,
					TotalPages:   3,
					Next: &capi.Link{
						Href: "/test?page=3",
					},
				},
				Resources: []TestResource{
					{ID: "3", Name: testResourceName3},
					{ID: "4", Name: "Resource 4"},
				},
			},
			3: {
				Pagination: capi.Pagination{
					TotalResults: 5,
					TotalPages:   3,
				},
				Resources: []TestResource{
					{ID: "5", Name: "Resource 5"},
				},
			},
		},
	}

	ctx := context.Background()

	resources, err := capi.FetchAllPages(ctx, client, "/test", nil, nil)
	require.NoError(t, err)
	assert.Len(t, resources, 5)
}

func TestFetchAllPages_WithMaxPages(t *testing.T) {
	t.Parallel()

	client := &MockPaginationClient{
		pages: map[int]*capi.ListResponse[TestResource]{
			1: {
				Pagination: capi.Pagination{
					TotalResults: 5,
					TotalPages:   3,
					Next: &capi.Link{
						Href: testPagedPath,
					},
				},
				Resources: []TestResource{
					{ID: "1", Name: testResourceName1},
					{ID: "2", Name: testResourceName2},
				},
			},
			2: {
				Pagination: capi.Pagination{
					TotalResults: 5,
					TotalPages:   3,
					Next: &capi.Link{
						Href: "/test?page=3",
					},
				},
				Resources: []TestResource{
					{ID: "3", Name: testResourceName3},
					{ID: "4", Name: "Resource 4"},
				},
			},
			3: {
				Pagination: capi.Pagination{
					TotalResults: 5,
					TotalPages:   3,
				},
				Resources: []TestResource{
					{ID: "5", Name: "Resource 5"},
				},
			},
		},
	}

	options := &capi.PaginationOptions{
		PageSize: 2,
		MaxPages: 2,
	}
	ctx := context.Background()

	resources, err := capi.FetchAllPages(ctx, client, "/test", nil, options)
	require.NoError(t, err)
	assert.Len(t, resources, 4) // Only first 2 pages
}

func TestStreamPages(t *testing.T) {
	t.Parallel()

	client := &MockPaginationClient{
		pages: map[int]*capi.ListResponse[TestResource]{
			1: {
				Pagination: capi.Pagination{
					TotalResults: 3,
					TotalPages:   2,
					Next: &capi.Link{
						Href: testPagedPath,
					},
				},
				Resources: []TestResource{
					{ID: "1", Name: testResourceName1},
					{ID: "2", Name: testResourceName2},
				},
			},
			2: {
				Pagination: capi.Pagination{
					TotalResults: 3,
					TotalPages:   2,
				},
				Resources: []TestResource{
					{ID: "3", Name: testResourceName3},
				},
			},
		},
	}

	ctx := context.Background()

	resultChan := capi.StreamPages(ctx, client, "/test", nil, nil)

	var allResources []TestResource //nolint:prealloc // channel range, size unknown

	pageCount := 0

	for result := range resultChan {
		require.NoError(t, result.Err)
		allResources = append(allResources, result.Items...)
		pageCount++
	}

	assert.Equal(t, 2, pageCount)
	assert.Len(t, allResources, 3)
}
