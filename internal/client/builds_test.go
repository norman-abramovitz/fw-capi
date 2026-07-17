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

// testBuildGUID is the fixture GUID for build tests.
const testBuildGUID = "test-build-guid"

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestBuildsClient_Create(t *testing.T) {
	t.Parallel()

	tests := []TestCreateOperation[capi.BuildCreateRequest, capi.Build]{
		{
			Name:         "create build",
			ExpectedPath: "/v3/builds",
			StatusCode:   http.StatusCreated,
			Request: &capi.BuildCreateRequest{
				Package: &capi.BuildPackageRef{
					GUID: testPackageGUID,
				},
				StagingMemoryInMB: intPtr(1024),
				StagingDiskInMB:   intPtr(1024),
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						testEnvLabelKey: testStagingLabel,
					},
				},
			},
			Response: &capi.Build{
				Resource: capi.Resource{
					GUID:      "build-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						testSelfKey: capi.Link{
							Href: "https://api.example.org/v3/builds/build-guid",
						},
						testAppKey: capi.Link{
							Href: testHrefAppLink,
						},
					},
				},
				State:             testStateStaging,
				StagingMemoryInMB: 1024,
				StagingDiskInMB:   1024,
				Package: &capi.BuildPackageRef{
					GUID: testPackageGUID,
				},
				Droplet: nil,
				CreatedBy: &capi.UserRef{
					GUID:  testUserGUID,
					Name:  "bill",
					Email: "bill@example.com",
				},
				Lifecycle: &capi.Lifecycle{
					Type: testBuildpackLifecycle,
					Data: map[string]any{
						"buildpacks": []string{testRubyBuildpackName},
						"stack":      testCFLinuxFS4Stack,
					},
				},
				Relationships: &capi.BuildRelationships{
					App: &capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: testAppGUID,
						},
					},
				},
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						testEnvLabelKey: testStagingLabel,
					},
				},
			},
			WantErr: false,
		},
		{
			Name:         "missing package",
			ExpectedPath: "/v3/builds",
			StatusCode:   http.StatusUnprocessableEntity,
			Request:      &capi.BuildCreateRequest{},
			Response: map[string]any{
				testErrorsKey: []map[string]any{
					{
						testCodeKey:   10008,
						testTitleKey:  testUnprocessableTitle,
						testDetailKey: "The request is semantically invalid: Missing required field 'package'",
					},
				},
			},
			WantErr:    true,
			ErrMessage: testUnprocessableTitle,
		},
	}

	RunCreateTests(t, tests,
		func(c *Client) func(context.Context, *capi.BuildCreateRequest) (*capi.Build, error) {
			return c.Builds().Create
		},
		func(request *http.Request) (*capi.BuildCreateRequest, error) {
			var requestBody capi.BuildCreateRequest

			err := json.NewDecoder(request.Body).Decode(&requestBody)
			if err != nil {
				return &requestBody, fmt.Errorf("failed to decode request body: %w", err)
			}

			return &requestBody, nil
		},
	)
}

func TestBuildsClient_Get(t *testing.T) {
	t.Parallel()

	tests := []TestGetOperation[capi.Build]{
		{
			Name:         testSuccessfulGetCase,
			GUID:         testBuildGUID,
			ExpectedPath: "/v3/builds/test-build-guid",
			StatusCode:   http.StatusOK,
			Response: &capi.Build{
				Resource: capi.Resource{
					GUID:      testBuildGUID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				State:             testStateStaged,
				StagingMemoryInMB: 1024,
				StagingDiskInMB:   1024,
				Package: &capi.BuildPackageRef{
					GUID: testPackageGUID,
				},
				Droplet: &capi.BuildDropletRef{
					GUID: testDropletGUID,
				},
				Lifecycle: &capi.Lifecycle{
					Type: testBuildpackLifecycle,
					Data: map[string]any{},
				},
			},
			WantErr: false,
		},
		{
			Name:         "build not found",
			GUID:         testNonExistentGUID,
			ExpectedPath: "/v3/builds/non-existent-guid",
			StatusCode:   http.StatusNotFound,
			Response: &capi.Build{
				Resource: capi.Resource{
					GUID:      testBuildGUID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				State:             testStateStaged,
				StagingMemoryInMB: 1024,
				StagingDiskInMB:   1024,
			},
			WantErr:    true,
			ErrMessage: testNotFoundTitle,
		},
	}

	RunGetTests(t, tests, func(c *Client) func(context.Context, string) (*capi.Build, error) {
		return c.Builds().Get
	})
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestBuildsClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/builds", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		// Check query parameters if present
		query := request.URL.Query()
		if states := query.Get(testStatesParam); states != "" {
			assert.Equal(t, "STAGING,STAGED", states)
		}

		if packageGuids := query.Get("package_guids"); packageGuids != "" {
			assert.Equal(t, "package-1,package-2", packageGuids)
		}

		response := capi.ListResponse[capi.Build]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "https://api.example.org/v3/builds?page=1"},
				Last:         capi.Link{Href: "https://api.example.org/v3/builds?page=1"},
				Next:         nil,
				Previous:     nil,
			},
			Resources: []capi.Build{
				{
					Resource: capi.Resource{
						GUID:      "build-1",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					State:             testStateStaging,
					StagingMemoryInMB: 1024,
					Package: &capi.BuildPackageRef{
						GUID: testPackageName1,
					},
				},
				{
					Resource: capi.Resource{
						GUID:      "build-2",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					State:             testStateStaged,
					StagingMemoryInMB: 2048,
					Package: &capi.BuildPackageRef{
						GUID: testPackageName2,
					},
					Droplet: &capi.BuildDropletRef{
						GUID: testDropletName2,
					},
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
	result, err := client.Builds().List(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.TotalResults)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "build-1", result.Resources[0].GUID)
	assert.Equal(t, testStateStaging, result.Resources[0].State)

	// Test with filters
	params := &capi.QueryParams{
		Filters: map[string][]string{
			testStatesParam: {testStateStaging, testStateStaged},
			"package_guids": {testPackageName1, testPackageName2},
		},
	}
	result, err = client.Builds().List(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestBuildsClient_ListForApp(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/apps/app-guid/builds", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		response := capi.ListResponse[capi.Build]{
			Pagination: capi.Pagination{
				TotalResults: 1,
				TotalPages:   1,
			},
			Resources: []capi.Build{
				{
					Resource: capi.Resource{
						GUID: "build-for-app",
					},
					State:             testStateStaged,
					StagingMemoryInMB: 1024,
					Relationships: &capi.BuildRelationships{
						App: &capi.Relationship{
							Data: &capi.RelationshipData{
								GUID: testAppGUID,
							},
						},
					},
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

	result, err := client.Builds().ListForApp(context.Background(), testAppGUID, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.Pagination.TotalResults)
	assert.Equal(t, "build-for-app", result.Resources[0].GUID)
}

//nolint:dupl // Acceptable duplication - each test validates different endpoints with different request/response types
func TestBuildsClient_Update(t *testing.T) {
	t.Parallel()

	request := &capi.BuildUpdateRequest{
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				testEnvLabelKey: testProductionLabel,
			},
			Annotations: map[string]string{
				testVersionAnnotationKey: testVersion100,
			},
		},
	}

	response := &capi.Build{
		Resource: capi.Resource{
			GUID:      testBuildGUID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		State: testStateStaged,
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				testEnvLabelKey: testProductionLabel,
			},
			Annotations: map[string]string{
				testVersionAnnotationKey: testVersion100,
			},
		},
	}

	RunStandardUpdateTest(t, "build", testBuildGUID, "/v3/builds/test-build-guid", request, response,
		func(c *Client) func(context.Context, string, *capi.BuildUpdateRequest) (*capi.Build, error) {
			return c.Builds().Update
		})
}
