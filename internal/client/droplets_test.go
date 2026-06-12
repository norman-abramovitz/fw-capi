package client_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/fivetwenty-io/capi/v3/internal/client"
	internalhttp "github.com/fivetwenty-io/capi/v3/internal/http"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
)

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestDropletsClient_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		request      *capi.DropletCreateRequest
		response     interface{}
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "create droplet",
			expectedPath: "/v3/droplets",
			statusCode:   http.StatusCreated,
			request: &capi.DropletCreateRequest{
				Relationships: capi.DropletRelationships{
					App: &capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: "app-guid",
						},
					},
				},
				ProcessTypes: map[string]string{
					"web":  "bundle exec rackup config.ru -p $PORT",
					"rake": "bundle exec rake",
				},
			},
			response: capi.Droplet{
				Resource: capi.Resource{
					GUID:      "droplet-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						"self": capi.Link{
							Href: "https://api.example.org/v3/droplets/droplet-guid",
						},
						"package": capi.Link{
							Href: "https://api.example.org/v3/packages/package-guid",
						},
						"app": capi.Link{
							Href: "https://api.example.org/v3/apps/app-guid",
						},
						"download": capi.Link{
							Href: "https://api.example.org/v3/droplets/droplet-guid/download",
						},
					},
				},
				State: "AWAITING_UPLOAD",
				Error: nil,
				Lifecycle: capi.Lifecycle{
					Type: "buildpack",
					Data: map[string]interface{}{},
				},
				ExecutionMetadata: "",
				ProcessTypes: map[string]string{
					"web":  "bundle exec rackup config.ru -p $PORT",
					"rake": "bundle exec rake",
				},
				Metadata: &capi.Metadata{
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				Relationships: &capi.DropletRelationships{
					App: &capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: "app-guid",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "missing app relationship",
			expectedPath: "/v3/droplets",
			statusCode:   http.StatusUnprocessableEntity,
			request: &capi.DropletCreateRequest{
				Relationships: capi.DropletRelationships{},
			},
			response: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"code":   10008,
						"title":  "CF-UnprocessableEntity",
						"detail": "App relationship is required",
					},
				},
			},
			wantErr:    true,
			errMessage: "CF-UnprocessableEntity",
		},
	}

	runCreateTestsForDroplets(t, tests)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestDropletsClient_Get(t *testing.T) {
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
			guid:         "test-droplet-guid",
			expectedPath: "/v3/droplets/test-droplet-guid",
			statusCode:   http.StatusOK,
			response: capi.Droplet{
				Resource: capi.Resource{
					GUID:      "test-droplet-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				State: "STAGED",
				Error: nil,
				Lifecycle: capi.Lifecycle{
					Type: "buildpack",
					Data: map[string]interface{}{},
				},
				ProcessTypes: map[string]string{
					"web": "bundle exec rackup config.ru -p $PORT",
				},
				Checksum: &capi.DropletChecksum{
					Type:  "sha256",
					Value: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				},
				Buildpacks: []capi.DetectedBuildpack{
					{
						Name:          "ruby_buildpack",
						DetectOutput:  "ruby 2.7.2",
						Version:       StringPtr("1.8.0"),
						BuildpackName: StringPtr("ruby"),
					},
				},
				Stack: StringPtr("cflinuxfs4"),
			},
			wantErr: false,
		},
		{
			name:         "droplet not found",
			guid:         "non-existent-guid",
			expectedPath: "/v3/droplets/non-existent-guid",
			statusCode:   http.StatusNotFound,
			response: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"code":   10010,
						"title":  "CF-ResourceNotFound",
						"detail": "Droplet not found",
					},
				},
			},
			wantErr:    true,
			errMessage: "CF-ResourceNotFound",
		},
	}

	runGetTestsForDroplets(t, tests)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestDropletsClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/droplets", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		// Check query parameters if present
		query := request.URL.Query()
		if appGuids := query.Get("app_guids"); appGuids != "" {
			assert.Equal(t, "app-1,app-2", appGuids)
		}

		if states := query.Get("states"); states != "" {
			assert.Equal(t, "STAGED,FAILED", states)
		}

		response := capi.ListResponse[capi.Droplet]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "https://api.example.org/v3/droplets?page=1"},
				Last:         capi.Link{Href: "https://api.example.org/v3/droplets?page=1"},
				Next:         nil,
				Previous:     nil,
			},
			Resources: []capi.Droplet{
				{
					Resource: capi.Resource{
						GUID:      "droplet-1",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					State: "STAGED",
					Lifecycle: capi.Lifecycle{
						Type: "buildpack",
						Data: map[string]interface{}{},
					},
				},
				{
					Resource: capi.Resource{
						GUID:      "droplet-2",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					State: "STAGED",
					Lifecycle: capi.Lifecycle{
						Type: "docker",
						Data: map[string]interface{}{},
					},
					Image: StringPtr("nginx:latest"),
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
	result, err := client.Droplets().List(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.TotalResults)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "droplet-1", result.Resources[0].GUID)
	assert.Equal(t, "buildpack", result.Resources[0].Lifecycle.Type)

	// Test with filters
	params := &capi.QueryParams{
		Filters: map[string][]string{
			"app_guids": {"app-1", "app-2"},
			"states":    {"STAGED", "FAILED"},
		},
	}
	result, err = client.Droplets().List(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestDropletsClient_ListForApp(t *testing.T) {
	t.Parallel()
	RunSimpleListTest(t, "ListForApp", "/v3/apps/app-guid/droplets", 1,
		func(i int) capi.Droplet {
			return capi.Droplet{
				Resource: capi.Resource{GUID: "droplet-for-app"},
				State:    "STAGED",
			}
		},
		func(c *Client) func(context.Context, string, *capi.QueryParams) (*capi.ListResponse[capi.Droplet], error) {
			return c.Droplets().ListForApp
		},
		"app-guid",
		func(resources []capi.Droplet) {
			assert.Equal(t, "droplet-for-app", resources[0].GUID)
		},
	)
}

func TestDropletsClient_ListForPackage(t *testing.T) {
	t.Parallel()
	RunSimpleListTest(t, "ListForPackage", "/v3/packages/package-guid/droplets", 1,
		func(i int) capi.Droplet {
			return capi.Droplet{
				Resource: capi.Resource{GUID: "droplet-for-package"},
				State:    "STAGED",
			}
		},
		func(c *Client) func(context.Context, string, *capi.QueryParams) (*capi.ListResponse[capi.Droplet], error) {
			return c.Droplets().ListForPackage
		},
		"package-guid",
		func(resources []capi.Droplet) {
			assert.Equal(t, "droplet-for-package", resources[0].GUID)
		},
	)
}

//nolint:dupl // Acceptable duplication - each test validates different endpoints with different request/response types
func TestDropletsClient_Update(t *testing.T) {
	t.Parallel()

	request := &capi.DropletUpdateRequest{
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				"env": "production",
			},
			Annotations: map[string]string{
				"version": "1.0.0",
			},
		},
	}

	response := &capi.Droplet{
		Resource: capi.Resource{
			GUID:      "test-droplet-guid",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		State: "STAGED",
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				"env": "production",
			},
			Annotations: map[string]string{
				"version": "1.0.0",
			},
		},
	}

	RunStandardUpdateTest(t, "droplet", "test-droplet-guid", "/v3/droplets/test-droplet-guid", request, response,
		func(c *Client) func(context.Context, string, *capi.DropletUpdateRequest) (*capi.Droplet, error) {
			return c.Droplets().Update
		})
}

// TestDropletsClient_Delete verifies that DELETE /v3/droplets/{guid} returns a
// *Job populated from the Location header (CF v3 async contract: 202 + Location).
func TestDropletsClient_Delete(t *testing.T) {
	t.Parallel()

	RunJobDeleteTest(t, "droplet delete", "/v3/droplets/test-droplet-guid", "droplet.delete",
		func(httpClient *internalhttp.Client) interface{} {
			return NewDropletsClient(httpClient)
		},
		func(client interface{}) (*capi.Job, error) {
			return client.(*DropletsClient).Delete(context.Background(), "test-droplet-guid")
		},
	)
}

// TestDropletsClient_DeleteMissingLocation pins the failure mode when CF returns
// 202 without a Location header — must error rather than return a nil-GUID Job.
func TestDropletsClient_DeleteMissingLocation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/droplets/test-droplet-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	client := NewDropletsClient(httpClient)

	job, err := client.Delete(context.Background(), "test-droplet-guid")
	require.Error(t, err)
	assert.Nil(t, job)
}

func TestDropletsClient_Copy(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		expectedPath := "/v3/droplets"
		if request.URL.RawQuery != "" {
			expectedPath = expectedPath + "?" + request.URL.RawQuery
		}

		assert.Equal(t, "/v3/droplets?source_guid=source-droplet-guid", expectedPath)
		assert.Equal(t, "POST", request.Method)

		var requestBody capi.DropletCopyRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)
		assert.Equal(t, "target-app-guid", requestBody.Relationships.App.Data.GUID)

		response := capi.Droplet{
			Resource: capi.Resource{
				GUID:      "new-droplet-guid",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			State: "COPYING",
			Relationships: &capi.DropletRelationships{
				App: &capi.Relationship{
					Data: &capi.RelationshipData{
						GUID: "target-app-guid",
					},
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	request := &capi.DropletCopyRequest{
		Relationships: capi.DropletRelationships{
			App: &capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: "target-app-guid",
				},
			},
		},
	}

	droplet, err := client.Droplets().Copy(context.Background(), "source-droplet-guid", request)
	require.NoError(t, err)
	require.NotNil(t, droplet)
	assert.Equal(t, "new-droplet-guid", droplet.GUID)
	assert.Equal(t, "COPYING", droplet.State)
	assert.Equal(t, "target-app-guid", droplet.Relationships.App.Data.GUID)
}

// runCreateTestsForDroplets runs droplet create tests.
func runCreateTestsForDroplets(t *testing.T, tests []struct {
	name         string
	request      *capi.DropletCreateRequest
	response     interface{}
	statusCode   int
	expectedPath string
	wantErr      bool
	errMessage   string
}) {
	t.Helper()

	for _, testCase := range tests {
		RunCreateTestWithValidation(t, testCase.name, testCase.expectedPath, testCase.statusCode, testCase.response, testCase.wantErr, testCase.errMessage, func(c *Client) error {
			droplet, err := c.Droplets().Create(context.Background(), testCase.request)
			if err == nil {
				assert.NotEmpty(t, droplet.GUID)
				assert.NotEmpty(t, droplet.State)
			}

			if err != nil {
				return fmt.Errorf("failed to create droplet: %w", err)
			}

			return nil
		})
	}
}

// runGetTestsForDroplets runs droplet get tests.
func runGetTestsForDroplets(t *testing.T, tests []struct {
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
		RunGetTestWithValidation(t, testCase.name, testCase.guid, testCase.expectedPath, testCase.statusCode, testCase.response, testCase.wantErr, testCase.errMessage, func(client *Client, guid string) error {
			droplet, err := client.Droplets().Get(context.Background(), guid)
			if err == nil {
				assert.Equal(t, guid, droplet.GUID)
				assert.Equal(t, "STAGED", droplet.State)
			}

			if err != nil {
				return fmt.Errorf("failed to get droplet: %w", err)
			}

			return nil
		})
	}
}

func TestDropletsClient_Download(t *testing.T) {
	t.Parallel()

	expectedContent := []byte("test droplet content")

	RunDownloadTest(t, "droplet", "test-droplet-guid", "/v3/droplets/test-droplet-guid/download", expectedContent,
		func(c *Client) func(context.Context, string) ([]byte, error) {
			return c.Droplets().Download
		})
}

func TestDropletsClient_Upload(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/droplets/test-droplet-guid/upload", request.URL.Path)
		assert.Equal(t, "POST", request.Method)
		assert.Contains(t, request.Header.Get("Content-Type"), "multipart/form-data")

		// Read the uploaded file
		file, _, err := request.FormFile("bits")
		assert.NoError(t, err)

		defer func() {
			err := file.Close()
			if err != nil {
				t.Logf("Warning: failed to close file: %v", err)
			}
		}()

		uploadedContent, err := io.ReadAll(file)
		assert.NoError(t, err)
		assert.Equal(t, []byte("test droplet content"), uploadedContent)

		response := capi.Droplet{
			Resource: capi.Resource{
				GUID:      "test-droplet-guid",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			State: "PROCESSING_UPLOAD",
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	dropletContent := []byte("test droplet content")
	droplet, err := c.Droplets().Upload(context.Background(), "test-droplet-guid", dropletContent)
	require.NoError(t, err)
	require.NotNil(t, droplet)
	assert.Equal(t, "test-droplet-guid", droplet.GUID)
	assert.Equal(t, "PROCESSING_UPLOAD", droplet.State)
}
