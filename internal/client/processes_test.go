package client_test

import (
	"context"
	"encoding/json"
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
func TestProcessesClient_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		guid         string
		response     any
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         testSuccessfulGetCase,
			guid:         testProcessGUIDFixture,
			expectedPath: "/v3/processes/test-process-guid",
			statusCode:   http.StatusOK,
			response: capi.Process{
				Resource: capi.Resource{
					GUID:      testProcessGUIDFixture,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						testSelfKey: capi.Link{
							Href: "https://api.example.org/v3/processes/test-process-guid",
						},
						"scale": capi.Link{
							Href:   "https://api.example.org/v3/processes/test-process-guid/actions/scale",
							Method: http.MethodPost,
						},
						testAppKey: capi.Link{
							Href: testHrefAppLink,
						},
					},
				},
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						testEnvironmentLabelKey: testProductionLabel,
					},
					Annotations: map[string]string{
						testNoteAnnotationKey: "web process",
					},
				},
				Type:                         testWebProcessType,
				Command:                      StringPtr("bundle exec rackup"),
				Instances:                    5,
				MemoryInMB:                   256,
				DiskInMB:                     1024,
				LogRateLimitInBytesPerSecond: intPtr(1024),
				HealthCheck: &capi.HealthCheck{
					Type: "port",
					Data: &capi.HealthCheckData{
						Timeout: intPtr(60),
					},
				},
				ReadinessHealthCheck: &capi.ReadinessHealthCheck{
					Type: "process",
					Data: &capi.ReadinessHealthCheckData{
						InvocationTimeout: intPtr(10),
					},
				},
				Relationships: &capi.ProcessRelationships{
					App: &capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: testAppGUID,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "not found",
			guid:         testNonExistentGUID,
			expectedPath: "/v3/processes/non-existent-guid",
			statusCode:   http.StatusNotFound,
			response: map[string]any{
				testErrorsKey: []map[string]any{
					{
						testCodeKey:   10010,
						testTitleKey:  testNotFoundTitle,
						testDetailKey: "Process not found",
					},
				},
			},
			wantErr:    true,
			errMessage: testNotFoundTitle,
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

			process, err := client.Processes().Get(context.Background(), testCase.guid)

			if testCase.wantErr {
				require.Error(t, err)
				require.ErrorContains(t, err, testCase.errMessage)
				assert.Nil(t, process)
			} else {
				require.NoError(t, err)
				require.NotNil(t, process)
				assert.Equal(t, testCase.guid, process.GUID)
				assert.Equal(t, testWebProcessType, process.Type)
				assert.Equal(t, "bundle exec rackup", *process.Command)
				assert.Equal(t, 5, process.Instances)
				assert.Equal(t, 256, process.MemoryInMB)
				assert.Equal(t, 1024, process.DiskInMB)
				assert.NotNil(t, process.HealthCheck)
				assert.Equal(t, "port", process.HealthCheck.Type)
				assert.NotNil(t, process.ReadinessHealthCheck)
				assert.Equal(t, "process", process.ReadinessHealthCheck.Type)
				assert.NotNil(t, process.Relationships)
				assert.Equal(t, testAppGUID, process.Relationships.App.Data.GUID)
			}
		})
	}
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestProcessesClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/processes", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		// Check query parameters if present
		query := request.URL.Query()
		if appGuids := query.Get(testAppGUIDsParam); appGuids != "" {
			assert.Equal(t, "app-1,app-2", appGuids)
		}

		response := capi.ListResponse[capi.Process]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "https://api.example.org/v3/processes?page=1"},
				Last:         capi.Link{Href: "https://api.example.org/v3/processes?page=1"},
				Next:         nil,
				Previous:     nil,
			},
			Resources: []capi.Process{
				{
					Resource: capi.Resource{
						GUID:      "process-1",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Type:       testWebProcessType,
					Command:    StringPtr("bundle exec rackup"),
					Instances:  3,
					MemoryInMB: 256,
					DiskInMB:   512,
				},
				{
					Resource: capi.Resource{
						GUID:      "process-2",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Type:       testWorkerProcessType,
					Command:    StringPtr("bundle exec sidekiq"),
					Instances:  1,
					MemoryInMB: 512,
					DiskInMB:   1024,
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
	result, err := client.Processes().List(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.TotalResults)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "process-1", result.Resources[0].GUID)
	assert.Equal(t, testWebProcessType, result.Resources[0].Type)
	assert.Equal(t, "process-2", result.Resources[1].GUID)
	assert.Equal(t, testWorkerProcessType, result.Resources[1].Type)

	// Test with filters
	params := &capi.QueryParams{
		Filters: map[string][]string{
			testAppGUIDsParam: {testAppGUID1, testAppGUID2},
			"types":           {testWebProcessType},
		},
	}
	result, err = client.Processes().List(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestProcessesClient_Update(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/processes/test-process-guid", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var requestBody capi.ProcessUpdateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		response := capi.Process{
			Resource: capi.Resource{
				GUID:      testProcessGUIDFixture,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Type:       testWebProcessType,
			Command:    requestBody.Command,
			Instances:  3,
			MemoryInMB: 256,
			DiskInMB:   512,
			Metadata:   requestBody.Metadata,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	request := &capi.ProcessUpdateRequest{
		Command: StringPtr("new command"),
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				testEnvLabelKey: testStagingLabel,
			},
		},
	}

	process, err := client.Processes().Update(context.Background(), testProcessGUIDFixture, request)
	require.NoError(t, err)
	require.NotNil(t, process)
	assert.Equal(t, testProcessGUIDFixture, process.GUID)
	assert.Equal(t, "new command", *process.Command)
	assert.Equal(t, testStagingLabel, process.Metadata.Labels[testEnvLabelKey])
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestProcessesClient_Scale(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		guid         string
		request      *capi.ProcessScaleRequest
		expectedPath string
	}{
		{
			name:         "scale instances and resources",
			guid:         testProcessGUIDFixture,
			expectedPath: "/v3/processes/test-process-guid/actions/scale",
			request: &capi.ProcessScaleRequest{
				Instances:  intPtr(10),
				MemoryInMB: intPtr(512),
				DiskInMB:   intPtr(2048),
			},
		},
		{
			name:         "scale with log rate limit",
			guid:         testProcessGUIDFixture,
			expectedPath: "/v3/processes/test-process-guid/actions/scale",
			request: &capi.ProcessScaleRequest{
				Instances:                    intPtr(5),
				LogRateLimitInBytesPerSecond: intPtr(2048),
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, testCase.expectedPath, request.URL.Path)
				assert.Equal(t, http.MethodPost, request.Method)

				var requestBody capi.ProcessScaleRequest

				err := json.NewDecoder(request.Body).Decode(&requestBody)
				assert.NoError(t, err)

				// Scale returns 202 + Location → /v3/jobs/{jobGuid}.
				writer.Header().Set("Location", "/v3/jobs/scale-job-guid")
				writer.WriteHeader(http.StatusAccepted)
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			job, err := client.Processes().Scale(context.Background(), testCase.guid, testCase.request)
			require.NoError(t, err)
			require.NotNil(t, job)
			assert.Equal(t, "scale-job-guid", job.GUID)
		})
	}
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestProcessesClient_GetStats(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/processes/test-process-guid/stats", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		isolationSegment := "default"
		response := capi.ProcessStats{
			Pagination: &capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
			},
			Resources: []capi.ProcessStatsDetail{
				{
					Type:  testWebProcessType,
					Index: 0,
					State: testStateRunning,
					Usage: &capi.ProcessUsage{
						Time:           time.Now().Format(time.RFC3339Nano),
						CPU:            0.15,
						CPUEntitlement: 0.2,
						Mem:            134217728,
						Disk:           268435456,
						LogRate:        100,
					},
					Host: "10.0.0.1",
					InstancePorts: []capi.ProcessInstancePort{
						{
							External:             61001,
							Internal:             8080,
							ExternalTLSProxyPort: 61443,
							InternalTLSProxyPort: 61002,
						},
					},
					Uptime:           3600,
					MemQuota:         268435456,
					DiskQuota:        1073741824,
					FdsQuota:         16384,
					IsolationSegment: &isolationSegment,
				},
				{
					Type:  testWebProcessType,
					Index: 1,
					State: testStateRunning,
					Usage: &capi.ProcessUsage{
						Time:    time.Now().Format(time.RFC3339Nano),
						CPU:     0.12,
						Mem:     100000000,
						Disk:    200000000,
						LogRate: 50,
					},
					Host:      "10.0.0.2",
					Uptime:    3600,
					MemQuota:  268435456,
					DiskQuota: 1073741824,
					FdsQuota:  16384,
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

	stats, err := client.Processes().GetStats(context.Background(), testProcessGUIDFixture)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 2, stats.Pagination.TotalResults)
	assert.Len(t, stats.Resources, 2)

	// Check first instance
	instance0 := stats.Resources[0]
	assert.Equal(t, testWebProcessType, instance0.Type)
	assert.Equal(t, 0, instance0.Index)
	assert.Equal(t, testStateRunning, instance0.State)
	assert.NotNil(t, instance0.Usage)
	assert.InDelta(t, 0.15, instance0.Usage.CPU, 1e-6)
	assert.Equal(t, int64(134217728), instance0.Usage.Mem)
	assert.Equal(t, "10.0.0.1", instance0.Host)
	assert.Len(t, instance0.InstancePorts, 1)
	assert.Equal(t, 61001, instance0.InstancePorts[0].External)
	assert.Equal(t, 8080, instance0.InstancePorts[0].Internal)
	assert.Equal(t, "default", *instance0.IsolationSegment)

	// Check second instance
	instance1 := stats.Resources[1]
	assert.Equal(t, 1, instance1.Index)
	assert.Equal(t, testStateRunning, instance1.State)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestProcessesClient_TerminateInstance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		guid         string
		index        int
		statusCode   int
		expectedPath string
		wantErr      bool
	}{
		{
			name:         "successful terminate",
			guid:         testProcessGUIDFixture,
			index:        0,
			expectedPath: "/v3/processes/test-process-guid/instances/0",
			statusCode:   http.StatusNoContent,
			wantErr:      false,
		},
		{
			name:         "terminate another instance",
			guid:         testProcessGUIDFixture,
			index:        3,
			expectedPath: "/v3/processes/test-process-guid/instances/3",
			statusCode:   http.StatusNoContent,
			wantErr:      false,
		},
		{
			name:         "process not found",
			guid:         testNonExistentGUID,
			index:        0,
			expectedPath: "/v3/processes/non-existent-guid/instances/0",
			statusCode:   http.StatusNotFound,
			wantErr:      true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, testCase.expectedPath, request.URL.Path)
				assert.Equal(t, "DELETE", request.Method)
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(testCase.statusCode)

				if testCase.wantErr {
					response := map[string]any{
						testErrorsKey: []map[string]any{
							{
								testCodeKey:   10010,
								testTitleKey:  testNotFoundTitle,
								testDetailKey: "Resource not found",
							},
						},
					}
					_ = json.NewEncoder(writer).Encode(response)
				}
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			err = client.Processes().TerminateInstance(context.Background(), testCase.guid, testCase.index)

			if testCase.wantErr {
				require.ErrorContains(t, err, testNotFoundTitle)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProcessesClient_GetWithEmbed(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "process_instances", request.URL.Query().Get("embed"))
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"guid": "process-guid", "type": "web"}`))
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	processes := NewProcessesClient(httpClient)

	process, err := processes.Get(context.Background(), testProcessGUID, capi.ProcessEmbedInstances)
	require.NoError(t, err)
	assert.Equal(t, testProcessGUID, process.GUID)
}

// Helper functions.
func intPtr(i int) *int {
	return &i
}
