package client_test

import (
	"bytes"
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

// Test constants for buildpack tests.
const (
	testCFLinuxFS4Stack              = "cflinuxfs4"
	testRubyBuildpackFilename        = "ruby_buildpack-v1.0.0.zip"
	testBuildpackStateAwaitingUpload = "AWAITING_UPLOAD"
)

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestBuildpacksClient_Create(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/buildpacks", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var requestBody capi.BuildpackCreateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.Equal(t, testRubyBuildpackName, requestBody.Name)
		assert.NotNil(t, requestBody.Stack)
		assert.Equal(t, testCFLinuxFS4Stack, *requestBody.Stack)
		assert.NotNil(t, requestBody.Position)
		assert.Equal(t, 42, *requestBody.Position)

		now := time.Now()
		buildpack := capi.Buildpack{
			Resource: capi.Resource{
				GUID:      testBuildpackGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:      requestBody.Name,
			State:     testBuildpackStateAwaitingUpload,
			Stack:     requestBody.Stack,
			Position:  *requestBody.Position,
			Lifecycle: testBuildpackLifecycle,
			Enabled:   true,
			Locked:    false,
			Metadata:  requestBody.Metadata,
			Links: capi.Links{
				testSelfKey: capi.Link{
					Href: "https://api.example.org/v3/buildpacks/buildpack-guid",
				},
				"upload": capi.Link{
					Href:   "https://api.example.org/v3/buildpacks/buildpack-guid/upload",
					Method: http.MethodPost,
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(writer).Encode(buildpack)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	buildpacks := c.Buildpacks()

	position := 42
	stack := testCFLinuxFS4Stack
	request := &capi.BuildpackCreateRequest{
		Name:     testRubyBuildpackName,
		Stack:    &stack,
		Position: &position,
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				testEnvLabelKey: testProductionLabel,
			},
		},
	}

	buildpack, err := buildpacks.Create(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, buildpack)
	assert.Equal(t, testBuildpackGUID, buildpack.GUID)
	assert.Equal(t, testRubyBuildpackName, buildpack.Name)
	assert.Equal(t, testBuildpackStateAwaitingUpload, buildpack.State)
	assert.Equal(t, 42, buildpack.Position)
}

func TestBuildpacksClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/buildpacks/buildpack-guid", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		now := time.Now()
		filename := testRubyBuildpackFilename
		stack := testCFLinuxFS4Stack
		buildpack := capi.Buildpack{
			Resource: capi.Resource{
				GUID:      testBuildpackGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:      testRubyBuildpackName,
			State:     testStateReady,
			Filename:  &filename,
			Stack:     &stack,
			Position:  1,
			Lifecycle: testBuildpackLifecycle,
			Enabled:   true,
			Locked:    false,
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(buildpack)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	buildpacks := c.Buildpacks()

	buildpack, err := buildpacks.Get(context.Background(), testBuildpackGUID)
	require.NoError(t, err)
	assert.NotNil(t, buildpack)
	assert.Equal(t, testBuildpackGUID, buildpack.GUID)
	assert.Equal(t, testRubyBuildpackName, buildpack.Name)
	assert.Equal(t, testStateReady, buildpack.State)
	assert.NotNil(t, buildpack.Filename)
	assert.Equal(t, testRubyBuildpackFilename, *buildpack.Filename)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestBuildpacksClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/buildpacks", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, testCFLinuxFS4Stack, request.URL.Query().Get("stacks"))
		assert.Equal(t, "ruby_buildpack,node_buildpack", request.URL.Query().Get(testNamesParam))

		now := time.Now()
		stack1 := testCFLinuxFS4Stack
		stack2 := testCFLinuxFS4Stack
		filename1 := testRubyBuildpackFilename
		filename2 := "node_buildpack-v2.0.0.zip"

		response := capi.ListResponse[capi.Buildpack]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "/v3/buildpacks?page=1"},
				Last:         capi.Link{Href: "/v3/buildpacks?page=1"},
			},
			Resources: []capi.Buildpack{
				{
					Resource: capi.Resource{
						GUID:      "buildpack-guid-1",
						CreatedAt: now,
						UpdatedAt: now,
					},
					Name:      testRubyBuildpackName,
					State:     testStateReady,
					Filename:  &filename1,
					Stack:     &stack1,
					Position:  1,
					Lifecycle: testBuildpackLifecycle,
					Enabled:   true,
					Locked:    false,
				},
				{
					Resource: capi.Resource{
						GUID:      "buildpack-guid-2",
						CreatedAt: now,
						UpdatedAt: now,
					},
					Name:      "node_buildpack",
					State:     testStateReady,
					Filename:  &filename2,
					Stack:     &stack2,
					Position:  2,
					Lifecycle: testBuildpackLifecycle,
					Enabled:   true,
					Locked:    false,
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	buildpacks := c.Buildpacks()

	params := &capi.QueryParams{
		Filters: map[string][]string{
			"stacks":       {testCFLinuxFS4Stack},
			testNamesParam: {testRubyBuildpackName, "node_buildpack"},
		},
	}

	list, err := buildpacks.List(context.Background(), params)
	require.NoError(t, err)
	assert.NotNil(t, list)
	assert.Equal(t, 2, list.Pagination.TotalResults)
	assert.Len(t, list.Resources, 2)
	assert.Equal(t, "buildpack-guid-1", list.Resources[0].GUID)
	assert.Equal(t, testRubyBuildpackName, list.Resources[0].Name)
	assert.Equal(t, "buildpack-guid-2", list.Resources[1].GUID)
	assert.Equal(t, "node_buildpack", list.Resources[1].Name)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestBuildpacksClient_Update(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/buildpacks/buildpack-guid", request.URL.Path)
		assert.Equal(t, http.MethodPatch, request.Method)

		var updateRequest capi.BuildpackUpdateRequest

		err := json.NewDecoder(request.Body).Decode(&updateRequest)
		assert.NoError(t, err)

		assert.NotNil(t, updateRequest.Position)
		assert.Equal(t, 5, *updateRequest.Position)
		assert.NotNil(t, updateRequest.Enabled)
		assert.False(t, *updateRequest.Enabled)

		now := time.Now()
		stack := testCFLinuxFS4Stack
		buildpack := capi.Buildpack{
			Resource: capi.Resource{
				GUID:      testBuildpackGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:      testRubyBuildpackName,
			State:     testStateReady,
			Stack:     &stack,
			Position:  *updateRequest.Position,
			Lifecycle: testBuildpackLifecycle,
			Enabled:   *updateRequest.Enabled,
			Locked:    false,
			Metadata:  updateRequest.Metadata,
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(buildpack)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	buildpacks := c.Buildpacks()

	position := 5
	enabled := false
	request := &capi.BuildpackUpdateRequest{
		Position: &position,
		Enabled:  &enabled,
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				testUpdatedValue: testTrueString,
			},
		},
	}

	buildpack, err := buildpacks.Update(context.Background(), testBuildpackGUID, request)
	require.NoError(t, err)
	assert.NotNil(t, buildpack)
	assert.Equal(t, testBuildpackGUID, buildpack.GUID)
	assert.Equal(t, 5, buildpack.Position)
	assert.False(t, buildpack.Enabled)
}

func TestBuildpacksClient_Delete(t *testing.T) {
	t.Parallel()

	// CF v3 DELETE /v3/buildpacks/{guid} is async: 202 Accepted, empty body,
	// Location header pointing at /v3/jobs/{jobGuid}.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/buildpacks/buildpack-guid", request.URL.Path)
		assert.Equal(t, http.MethodDelete, request.Method)

		writer.Header().Set("Location", testJobPath)
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	buildpacks := c.Buildpacks()

	job, err := buildpacks.Delete(context.Background(), testBuildpackGUID)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestBuildpacksClient_Upload(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/buildpacks/buildpack-guid/upload", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Contains(t, request.Header.Get("Content-Type"), "multipart/form-data")

		// Parse multipart form
		err := request.ParseMultipartForm(10 << 20) //nolint:gosec // G120: test-only handler with a fixed 10 MB cap
		assert.NoError(t, err)

		// Check that bits file is present
		file, header, err := request.FormFile(testBitsType)
		assert.NoError(t, err)

		defer func() {
			err := file.Close()
			if err != nil {
				t.Logf("Warning: failed to close file: %v", err)
			}
		}()

		assert.NotNil(t, header)

		// Read the uploaded content
		uploadedContent, err := io.ReadAll(file)
		assert.NoError(t, err)
		assert.Equal(t, "buildpack content", string(uploadedContent))

		now := time.Now()
		filename := testRubyBuildpackFilename
		stack := testCFLinuxFS4Stack
		buildpack := capi.Buildpack{
			Resource: capi.Resource{
				GUID:      testBuildpackGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:      testRubyBuildpackName,
			State:     testStateReady,
			Filename:  &filename,
			Stack:     &stack,
			Position:  1,
			Lifecycle: testBuildpackLifecycle,
			Enabled:   true,
			Locked:    false,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(writer).Encode(buildpack)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	buildpacks := c.Buildpacks()

	// Create a reader with test content
	content := bytes.NewReader([]byte("buildpack content"))

	buildpack, err := buildpacks.Upload(context.Background(), testBuildpackGUID, content)
	require.NoError(t, err)
	assert.NotNil(t, buildpack)
	assert.Equal(t, testBuildpackGUID, buildpack.GUID)
	assert.Equal(t, testStateReady, buildpack.State)
	assert.NotNil(t, buildpack.Filename)
	assert.Equal(t, testRubyBuildpackFilename, *buildpack.Filename)
}

func TestBuildpacksClient_GetNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/buildpacks/buildpack-guid", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	buildpacks := c.Buildpacks()

	buildpack, err := buildpacks.Get(context.Background(), testBuildpackGUID)
	require.Error(t, err)
	assert.Nil(t, buildpack)
}

func TestBuildpacksClient_CreateWithCNBLifecycle(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/buildpacks", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var createRequest capi.BuildpackCreateRequest

		err := json.NewDecoder(request.Body).Decode(&createRequest)
		assert.NoError(t, err)

		assert.Equal(t, "paketo_buildpack", createRequest.Name)
		assert.NotNil(t, createRequest.Lifecycle)
		assert.Equal(t, "cnb", *createRequest.Lifecycle)

		now := time.Now()
		buildpack := capi.Buildpack{
			Resource: capi.Resource{
				GUID:      testBuildpackGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:      createRequest.Name,
			State:     testBuildpackStateAwaitingUpload,
			Position:  1,
			Lifecycle: *createRequest.Lifecycle,
			Enabled:   true,
			Locked:    false,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(writer).Encode(buildpack)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	buildpacks := c.Buildpacks()

	lifecycle := "cnb"
	request := &capi.BuildpackCreateRequest{
		Name:      "paketo_buildpack",
		Lifecycle: &lifecycle,
	}

	buildpack, err := buildpacks.Create(context.Background(), request)
	require.NoError(t, err)
	assert.NotNil(t, buildpack)
	assert.Equal(t, testBuildpackGUID, buildpack.GUID)
	assert.Equal(t, "paketo_buildpack", buildpack.Name)
	assert.Equal(t, "cnb", buildpack.Lifecycle)
}
