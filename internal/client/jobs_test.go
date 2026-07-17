package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "github.com/fivetwenty-io/capi/v3/internal/client"
	internalhttp "github.com/fivetwenty-io/capi/v3/internal/http"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testJobPath is the path for the "job-guid" fixture job.
const testJobPath = "/v3/jobs/job-guid"

func TestJobsClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testJobPath, request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		job := capi.Job{
			Resource: capi.Resource{
				GUID:      testJobGUID,
				CreatedAt: time.Now().Add(-time.Minute),
				UpdatedAt: time.Now().Add(-30 * time.Second),
				Links: capi.Links{
					testSelfKey: capi.Link{
						Href: testJobPath,
					},
				},
			},
			Operation: testApplyManifestOperation,
			State:     "COMPLETE",
			Errors:    []capi.APIError{},
			Warnings: []capi.Warning{
				{
					Detail: "Deprecated property detected: buildpack. App manifests must use buildpacks.",
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(job)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	jobs := NewJobsClient(httpClient)

	job, err := jobs.Get(context.Background(), testJobGUID)
	require.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
	assert.Equal(t, testApplyManifestOperation, job.Operation)
	assert.Equal(t, "COMPLETE", job.State)
	assert.Len(t, job.Warnings, 1)
	assert.Equal(t, "Deprecated property detected: buildpack. App manifests must use buildpacks.", job.Warnings[0].Detail)
}

func TestJobsClient_Get_Processing(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testJobPath, request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		job := capi.Job{
			Resource: capi.Resource{
				GUID:      testJobGUID,
				CreatedAt: time.Now().Add(-time.Minute),
				UpdatedAt: time.Now(),
				Links: capi.Links{
					testSelfKey: capi.Link{
						Href: testJobPath,
					},
				},
			},
			Operation: "service_instance.create",
			State:     testStateProcessing,
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(job)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	jobs := NewJobsClient(httpClient)

	job, err := jobs.Get(context.Background(), testJobGUID)
	require.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
	assert.Equal(t, "service_instance.create", job.Operation)
	assert.Equal(t, testStateProcessing, job.State)
	assert.Empty(t, job.Errors)
	assert.Empty(t, job.Warnings)
}

func TestJobsClient_Get_Failed(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testJobPath, request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		job := capi.Job{
			Resource: capi.Resource{
				GUID:      testJobGUID,
				CreatedAt: time.Now().Add(-time.Minute),
				UpdatedAt: time.Now(),
				Links: capi.Links{
					testSelfKey: capi.Link{
						Href: testJobPath,
					},
				},
			},
			Operation: "service_broker.delete",
			State:     testStateFailed,
			Errors: []capi.APIError{
				{
					Detail: "Service broker deletion failed: broker has service instances",
					Title:  "CF-ServiceBrokerNotRemovable",
					Code:   10001,
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(job)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	jobs := NewJobsClient(httpClient)

	job, err := jobs.Get(context.Background(), testJobGUID)
	require.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
	assert.Equal(t, "service_broker.delete", job.Operation)
	assert.Equal(t, testStateFailed, job.State)
	assert.Len(t, job.Errors, 1)
	assert.Equal(t, "Service broker deletion failed: broker has service instances", job.Errors[0].Detail)
	assert.Equal(t, "CF-ServiceBrokerNotRemovable", job.Errors[0].Title)
	assert.Equal(t, 10001, job.Errors[0].Code)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestJobsClient_PollUntilComplete_Success(t *testing.T) {
	t.Parallel()

	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testJobPath, request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		attempts++

		var job capi.Job

		// Simulate job transitioning from PROCESSING to COMPLETE
		if attempts <= 2 {
			job = capi.Job{
				Resource: capi.Resource{
					GUID:      testJobGUID,
					CreatedAt: time.Now().Add(-time.Minute),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						testSelfKey: capi.Link{
							Href: testJobPath,
						},
					},
				},
				Operation: testApplyManifestOperation,
				State:     testStateProcessing,
			}
		} else {
			job = capi.Job{
				Resource: capi.Resource{
					GUID:      testJobGUID,
					CreatedAt: time.Now().Add(-time.Minute),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						testSelfKey: capi.Link{
							Href: testJobPath,
						},
					},
				},
				Operation: testApplyManifestOperation,
				State:     "COMPLETE",
				Warnings: []capi.Warning{
					{
						Detail: "Manifest applied successfully",
					},
				},
			}
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(job)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	jobs := NewJobsClient(httpClient)

	// Test polling functionality

	job, err := jobs.PollUntilComplete(context.Background(), testJobGUID)
	require.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
	assert.Equal(t, testApplyManifestOperation, job.Operation)
	assert.Equal(t, "COMPLETE", job.State)
	assert.Equal(t, 3, attempts)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestJobsClient_PollUntilComplete_Failed(t *testing.T) {
	t.Parallel()

	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testJobPath, request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		attempts++

		var job capi.Job

		// Simulate job transitioning from PROCESSING to FAILED
		if attempts <= 1 {
			job = capi.Job{
				Resource: capi.Resource{
					GUID:      testJobGUID,
					CreatedAt: time.Now().Add(-time.Minute),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						testSelfKey: capi.Link{
							Href: testJobPath,
						},
					},
				},
				Operation: "service_instance.delete",
				State:     testStateProcessing,
			}
		} else {
			job = capi.Job{
				Resource: capi.Resource{
					GUID:      testJobGUID,
					CreatedAt: time.Now().Add(-time.Minute),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						testSelfKey: capi.Link{
							Href: testJobPath,
						},
					},
				},
				Operation: "service_instance.delete",
				State:     testStateFailed,
				Errors: []capi.APIError{
					{
						Detail: "Service instance deletion failed",
						Title:  "CF-ServiceInstanceNotRemovable",
						Code:   10002,
					},
				},
			}
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(job)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	jobs := NewJobsClient(httpClient)

	// Test polling functionality

	job, err := jobs.PollUntilComplete(context.Background(), testJobGUID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrJobFailed)
	assert.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
	assert.Equal(t, "service_instance.delete", job.Operation)
	assert.Equal(t, testStateFailed, job.State)
	assert.Len(t, job.Errors, 1)
}

func TestJobsClient_PollUntilComplete_Timeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testJobPath, request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		// Always return PROCESSING
		job := capi.Job{
			Resource: capi.Resource{
				GUID:      testJobGUID,
				CreatedAt: time.Now().Add(-time.Minute),
				UpdatedAt: time.Now(),
				Links: capi.Links{
					testSelfKey: capi.Link{
						Href: testJobPath,
					},
				},
			},
			Operation: testApplyManifestOperation,
			State:     testStateProcessing,
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(job)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	jobs := NewJobsClient(httpClient)

	// Test polling with timeout

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	job, err := jobs.PollUntilComplete(ctx, testJobGUID)
	require.Error(t, err)
	assert.True(t, err.Error() == "timeout waiting for job to complete: context deadline exceeded" ||
		strings.Contains(err.Error(), "context deadline exceeded"),
		"Expected timeout error, got: %v", err)

	if job != nil {
		assert.Equal(t, testStateProcessing, job.State)
	}
}
