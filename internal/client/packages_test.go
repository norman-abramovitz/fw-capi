package client_test

import (
	"context"
	"encoding/json"
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
func TestPackagesClient_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		request      *capi.PackageCreateRequest
		response     interface{}
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "create bits package",
			expectedPath: "/v3/packages",
			statusCode:   http.StatusCreated,
			request: &capi.PackageCreateRequest{
				Type: "bits",
				Relationships: capi.PackageRelationships{
					App: &capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: "app-guid",
						},
					},
				},
			},
			response: capi.Package{
				Resource: capi.Resource{
					GUID:      "package-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						"self": capi.Link{
							Href: "https://api.example.org/v3/packages/package-guid",
						},
						"upload": capi.Link{
							Href:   "https://api.example.org/v3/packages/package-guid/upload",
							Method: "POST",
						},
						"download": capi.Link{
							Href:   "https://api.example.org/v3/packages/package-guid/download",
							Method: "GET",
						},
						"app": capi.Link{
							Href: "https://api.example.org/v3/apps/app-guid",
						},
					},
				},
				Type:  "bits",
				State: "AWAITING_UPLOAD",
				Data: &capi.PackageData{
					Checksum: &capi.PackageChecksum{
						Type:  "sha256",
						Value: nil,
					},
					Error: nil,
				},
				Metadata: &capi.Metadata{
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				Relationships: &capi.PackageRelationships{
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
			name:         "create docker package",
			expectedPath: "/v3/packages",
			statusCode:   http.StatusCreated,
			request: &capi.PackageCreateRequest{
				Type: "docker",
				Relationships: capi.PackageRelationships{
					App: &capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: "app-guid",
						},
					},
				},
				Data: &capi.PackageCreateData{
					Image:    StringPtr("nginx:latest"),
					Username: StringPtr("dockeruser"),
					Password: StringPtr("dockerpass"),
				},
			},
			response: capi.Package{
				Resource: capi.Resource{
					GUID:      "docker-package-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Type:  "docker",
				State: "READY",
				Data: &capi.PackageData{
					Image:    StringPtr("nginx:latest"),
					Username: StringPtr("dockeruser"),
					Password: nil, // Password is not returned
				},
			},
			wantErr: false,
		},
		{
			name:         "missing app relationship",
			expectedPath: "/v3/packages",
			statusCode:   http.StatusUnprocessableEntity,
			request: &capi.PackageCreateRequest{
				Type:          "bits",
				Relationships: capi.PackageRelationships{},
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

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, testCase.expectedPath, request.URL.Path)
				assert.Equal(t, "POST", request.Method)

				var requestBody capi.PackageCreateRequest

				err := json.NewDecoder(request.Body).Decode(&requestBody)
				assert.NoError(t, err)

				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(testCase.statusCode)
				_ = json.NewEncoder(writer).Encode(testCase.response)
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			pkg, err := client.Packages().Create(context.Background(), testCase.request)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMessage)
				assert.Nil(t, pkg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pkg)
				assert.NotEmpty(t, pkg.GUID)
				assert.NotEmpty(t, pkg.Type)
			}
		})
	}
}

//nolint:dupl,funlen // Acceptable duplication - each test validates different endpoints with different assertions
func TestPackagesClient_Get(t *testing.T) {
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
			guid:         "test-package-guid",
			expectedPath: "/v3/packages/test-package-guid",
			statusCode:   http.StatusOK,
			response: capi.Package{
				Resource: capi.Resource{
					GUID:      "test-package-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Type:  "bits",
				State: "READY",
				Data: &capi.PackageData{
					Checksum: &capi.PackageChecksum{
						Type:  "sha256",
						Value: StringPtr("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"),
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "package not found",
			guid:         "non-existent-guid",
			expectedPath: "/v3/packages/non-existent-guid",
			statusCode:   http.StatusNotFound,
			response: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"code":   10010,
						"title":  "CF-ResourceNotFound",
						"detail": "Package not found",
					},
				},
			},
			wantErr:    true,
			errMessage: "CF-ResourceNotFound",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, testCase.expectedPath, request.URL.Path)
				assert.Equal(t, "GET", request.Method)
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(testCase.statusCode)
				_ = json.NewEncoder(writer).Encode(testCase.response)
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			pkg, err := client.Packages().Get(context.Background(), testCase.guid)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMessage)
				assert.Nil(t, pkg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pkg)
				assert.Equal(t, testCase.guid, pkg.GUID)
				assert.Equal(t, "READY", pkg.State)
			}
		})
	}
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestPackagesClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/packages", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		// Check query parameters if present
		query := request.URL.Query()
		if appGuids := query.Get("app_guids"); appGuids != "" {
			assert.Equal(t, "app-1,app-2", appGuids)
		}

		if states := query.Get("states"); states != "" {
			assert.Equal(t, "READY,FAILED", states)
		}

		response := capi.ListResponse[capi.Package]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "https://api.example.org/v3/packages?page=1"},
				Last:         capi.Link{Href: "https://api.example.org/v3/packages?page=1"},
				Next:         nil,
				Previous:     nil,
			},
			Resources: []capi.Package{
				{
					Resource: capi.Resource{
						GUID:      "package-1",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Type:  "bits",
					State: "READY",
				},
				{
					Resource: capi.Resource{
						GUID:      "package-2",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Type:  "docker",
					State: "READY",
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
	result, err := client.Packages().List(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.TotalResults)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "package-1", result.Resources[0].GUID)
	assert.Equal(t, "bits", result.Resources[0].Type)

	// Test with filters
	params := &capi.QueryParams{
		Filters: map[string][]string{
			"app_guids": {"app-1", "app-2"},
			"states":    {"READY", "FAILED"},
		},
	}
	result, err = client.Packages().List(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestPackagesClient_Update(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/packages/test-package-guid", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var requestBody capi.PackageUpdateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		response := capi.Package{
			Resource: capi.Resource{
				GUID:      "test-package-guid",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Type:     "bits",
			State:    "READY",
			Metadata: requestBody.Metadata,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	request := &capi.PackageUpdateRequest{
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				"env": "production",
			},
			Annotations: map[string]string{
				"version": "1.0.0",
			},
		},
	}

	pkg, err := client.Packages().Update(context.Background(), "test-package-guid", request)
	require.NoError(t, err)
	require.NotNil(t, pkg)
	assert.Equal(t, "test-package-guid", pkg.GUID)
	assert.Equal(t, "production", pkg.Metadata.Labels["env"])
	assert.Equal(t, "1.0.0", pkg.Metadata.Annotations["version"])
}

// TestPackagesClient_Delete verifies that DELETE /v3/packages/{guid} returns a
// *Job populated from the Location header (CF v3 async contract: 202 + Location).
func TestPackagesClient_Delete(t *testing.T) {
	t.Parallel()

	RunJobDeleteTest(t, "package delete", "/v3/packages/test-package-guid", "package.delete",
		func(httpClient *internalhttp.Client) interface{} {
			return NewPackagesClient(httpClient)
		},
		func(client interface{}) (*capi.Job, error) {
			return client.(*PackagesClient).Delete(context.Background(), "test-package-guid")
		},
	)
}

// TestPackagesClient_DeleteMissingLocation pins the failure mode when CF returns
// 202 without a Location header — must error rather than return a nil-GUID Job.
func TestPackagesClient_DeleteMissingLocation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/packages/test-package-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	client := NewPackagesClient(httpClient)

	job, err := client.Delete(context.Background(), "test-package-guid")
	require.Error(t, err)
	assert.Nil(t, job)
}

func TestPackagesClient_Upload(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/packages/test-package-guid/upload", request.URL.Path)
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
		assert.Equal(t, []byte("test zip content"), uploadedContent)

		response := capi.Package{
			Resource: capi.Resource{
				GUID:      "test-package-guid",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Type:  "bits",
			State: "PROCESSING_UPLOAD",
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	zipContent := []byte("test zip content")
	pkg, err := client.Packages().Upload(context.Background(), "test-package-guid", zipContent)
	require.NoError(t, err)
	require.NotNil(t, pkg)
	assert.Equal(t, "test-package-guid", pkg.GUID)
	assert.Equal(t, "PROCESSING_UPLOAD", pkg.State)
}

func TestPackagesClient_Download(t *testing.T) {
	t.Parallel()

	expectedContent := []byte("test package content")

	RunDownloadTest(t, "package", "test-package-guid", "/v3/packages/test-package-guid/download", expectedContent,
		func(client *Client) func(context.Context, string) ([]byte, error) {
			return client.Packages().Download
		})
}

func TestPackagesClient_Copy(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		expectedPath := "/v3/packages"
		if request.URL.RawQuery != "" {
			expectedPath = expectedPath + "?" + request.URL.RawQuery
		}

		assert.Equal(t, "/v3/packages?source_guid=source-package-guid", expectedPath)
		assert.Equal(t, "POST", request.Method)

		var requestBody capi.PackageCopyRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)
		assert.Equal(t, "target-app-guid", requestBody.Relationships.App.Data.GUID)

		response := capi.Package{
			Resource: capi.Resource{
				GUID:      "new-package-guid",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Type:  "bits",
			State: "COPYING",
			Relationships: &capi.PackageRelationships{
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

	request := &capi.PackageCopyRequest{
		Relationships: capi.PackageRelationships{
			App: &capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: "target-app-guid",
				},
			},
		},
	}

	pkg, err := client.Packages().Copy(context.Background(), "source-package-guid", request)
	require.NoError(t, err)
	require.NotNil(t, pkg)
	assert.Equal(t, "new-package-guid", pkg.GUID)
	assert.Equal(t, "COPYING", pkg.State)
	assert.Equal(t, "target-app-guid", pkg.Relationships.App.Data.GUID)
}
