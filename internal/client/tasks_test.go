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
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
)

//nolint:funlen // Test functions can be longer for detailed testing
func TestTasksClient_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		appGUID      string
		request      *capi.TaskCreateRequest
		response     any
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "create task with command",
			appGUID:      testAppGUIDFixture,
			expectedPath: "/v3/apps/test-app-guid/tasks",
			statusCode:   http.StatusAccepted,
			request: &capi.TaskCreateRequest{
				Command:    StringPtr(testMigrateCommand),
				Name:       StringPtr(testMigrateTaskName),
				MemoryInMB: intPtr(512),
				DiskInMB:   intPtr(1024),
			},
			response: capi.Task{
				Resource: capi.Resource{
					GUID:      "task-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						testSelfKey: capi.Link{
							Href: "https://api.example.org/v3/tasks/task-guid",
						},
						testAppKey: capi.Link{
							Href: "https://api.example.org/v3/apps/test-app-guid",
						},
						"cancel": capi.Link{
							Href:   "https://api.example.org/v3/tasks/task-guid/actions/cancel",
							Method: http.MethodPost,
						},
						"droplet": capi.Link{
							Href: "https://api.example.org/v3/droplets/droplet-guid",
						},
					},
				},
				SequenceID:  1,
				Name:        testMigrateTaskName,
				Command:     testMigrateCommand,
				User:        StringPtr("vcap"),
				State:       testStateRunning,
				MemoryInMB:  512,
				DiskInMB:    1024,
				DropletGUID: testDropletGUID,
				Result: &capi.TaskResult{
					FailureReason: nil,
				},
				Metadata: &capi.Metadata{
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				Relationships: &capi.TaskRelationships{
					App: &capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: testAppGUIDFixture,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "create task with template",
			appGUID:      testAppGUIDFixture,
			expectedPath: "/v3/apps/test-app-guid/tasks",
			statusCode:   http.StatusAccepted,
			request: &capi.TaskCreateRequest{
				Template: &capi.TaskTemplate{
					Process: &capi.TaskTemplateProcess{
						GUID: testProcessGUID,
					},
				},
			},
			response: capi.Task{
				Resource: capi.Resource{
					GUID:      "task-guid-2",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				SequenceID:  2,
				Name:        "task",
				Command:     "bundle exec rackup",
				State:       testStatePending,
				MemoryInMB:  256,
				DiskInMB:    512,
				DropletGUID: testDropletGUID,
			},
			wantErr: false,
		},
		{
			name:         "app not found",
			appGUID:      "non-existent-app",
			expectedPath: "/v3/apps/non-existent-app/tasks",
			statusCode:   http.StatusNotFound,
			request: &capi.TaskCreateRequest{
				Command: StringPtr("ls"),
			},
			response: map[string]any{
				testErrorsKey: []map[string]any{
					{
						testCodeKey:   10010,
						testTitleKey:  testNotFoundTitle,
						testDetailKey: "App not found",
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
				assert.Equal(t, http.MethodPost, request.Method)

				var requestBody capi.TaskCreateRequest

				err := json.NewDecoder(request.Body).Decode(&requestBody)
				assert.NoError(t, err)

				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(testCase.statusCode)
				_ = json.NewEncoder(writer).Encode(testCase.response)
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			task, err := client.Tasks().Create(context.Background(), testCase.appGUID, testCase.request)

			if testCase.wantErr {
				require.ErrorContains(t, err, testCase.errMessage)
				assert.Nil(t, task)
			} else {
				require.NoError(t, err)
				require.NotNil(t, task)
				assert.NotEmpty(t, task.GUID)
				assert.NotEmpty(t, task.State)
			}
		})
	}
}

func TestTasksClient_Get(t *testing.T) {
	t.Parallel()

	tests := []TestGetOperation[capi.Task]{
		{
			Name:         testSuccessfulGetCase,
			GUID:         testTaskGUIDFixture,
			ExpectedPath: "/v3/tasks/test-task-guid",
			StatusCode:   http.StatusOK,
			Response: &capi.Task{
				Resource: capi.Resource{
					GUID:      testTaskGUIDFixture,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				SequenceID:                   1,
				Name:                         testMigrateTaskName,
				Command:                      testMigrateCommand,
				User:                         StringPtr("vcap"),
				State:                        "SUCCEEDED",
				MemoryInMB:                   512,
				DiskInMB:                     1024,
				LogRateLimitInBytesPerSecond: intPtr(1024),
				DropletGUID:                  testDropletGUID,
				Result: &capi.TaskResult{
					FailureReason: nil,
				},
			},
			WantErr: false,
		},
		{
			Name:         "task not found",
			GUID:         testNonExistentGUID,
			ExpectedPath: "/v3/tasks/non-existent-guid",
			StatusCode:   http.StatusNotFound,
			Response: &capi.Task{
				Resource: capi.Resource{
					GUID:      testTaskGUIDFixture,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				State: "SUCCEEDED",
			},
			WantErr:    true,
			ErrMessage: testNotFoundTitle,
		},
	}

	RunGetTests(t, tests, func(c *Client) func(context.Context, string) (*capi.Task, error) {
		return c.Tasks().Get
	})
}

//nolint:funlen // Test functions can be longer for detailed testing
func TestTasksClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/tasks", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		// Check query parameters if present
		query := request.URL.Query()
		if appGuids := query.Get(testAppGUIDsParam); appGuids != "" {
			assert.Equal(t, "app-1,app-2", appGuids)
		}

		if states := query.Get(testStatesParam); states != "" {
			assert.Equal(t, "RUNNING,PENDING", states)
		}

		response := capi.ListResponse[capi.Task]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "https://api.example.org/v3/tasks?page=1"},
				Last:         capi.Link{Href: "https://api.example.org/v3/tasks?page=1"},
				Next:         nil,
				Previous:     nil,
			},
			Resources: []capi.Task{
				{
					Resource: capi.Resource{
						GUID:      "task-1",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					SequenceID:  1,
					Name:        testMigrateTaskName,
					Command:     testMigrateCommand,
					State:       testStateRunning,
					MemoryInMB:  512,
					DiskInMB:    1024,
					DropletGUID: "droplet-1",
				},
				{
					Resource: capi.Resource{
						GUID:      "task-2",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					SequenceID:  2,
					Name:        "seed",
					Command:     "rake db:seed",
					State:       testStatePending,
					MemoryInMB:  256,
					DiskInMB:    512,
					DropletGUID: testDropletName2,
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
	result, err := client.Tasks().List(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.TotalResults)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "task-1", result.Resources[0].GUID)
	assert.Equal(t, testStateRunning, result.Resources[0].State)

	// Test with filters
	params := &capi.QueryParams{
		Filters: map[string][]string{
			testAppGUIDsParam: {testAppGUID1, testAppGUID2},
			testStatesParam:   {testStateRunning, testStatePending},
		},
	}
	result, err = client.Tasks().List(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestTasksClient_Update(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/tasks/test-task-guid", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var requestBody capi.TaskUpdateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		response := capi.Task{
			Resource: capi.Resource{
				GUID:      testTaskGUIDFixture,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			SequenceID:  1,
			Name:        testMigrateTaskName,
			Command:     testMigrateCommand,
			State:       testStateRunning,
			MemoryInMB:  512,
			DiskInMB:    1024,
			DropletGUID: testDropletGUID,
			Metadata:    requestBody.Metadata,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	request := &capi.TaskUpdateRequest{
		Metadata: &capi.Metadata{
			Labels: map[string]string{
				testEnvLabelKey: testProductionLabel,
			},
			Annotations: map[string]string{
				testNoteAnnotationKey: "database migration",
			},
		},
	}

	task, err := client.Tasks().Update(context.Background(), testTaskGUIDFixture, request)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, testTaskGUIDFixture, task.GUID)
	assert.Equal(t, testProductionLabel, task.Metadata.Labels[testEnvLabelKey])
	assert.Equal(t, "database migration", task.Metadata.Annotations[testNoteAnnotationKey])
}

//nolint:funlen // Test functions can be longer for detailed testing
func TestTasksClient_Cancel(t *testing.T) {
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
			name:         "successful cancel",
			guid:         testTaskGUIDFixture,
			expectedPath: "/v3/tasks/test-task-guid/actions/cancel",
			statusCode:   http.StatusAccepted,
			response: capi.Task{
				Resource: capi.Resource{
					GUID:      testTaskGUIDFixture,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				SequenceID:  1,
				Name:        testMigrateTaskName,
				Command:     testMigrateCommand,
				State:       "CANCELING",
				MemoryInMB:  512,
				DiskInMB:    1024,
				DropletGUID: testDropletGUID,
			},
			wantErr: false,
		},
		{
			name:         "task already completed",
			guid:         "completed-task-guid",
			expectedPath: "/v3/tasks/completed-task-guid/actions/cancel",
			statusCode:   http.StatusUnprocessableEntity,
			response: map[string]any{
				testErrorsKey: []map[string]any{
					{
						testCodeKey:   10008,
						testTitleKey:  testUnprocessableTitle,
						testDetailKey: "Task has already been completed",
					},
				},
			},
			wantErr:    true,
			errMessage: testUnprocessableTitle,
		},
		{
			name:         "task not found",
			guid:         testNonExistentGUID,
			expectedPath: "/v3/tasks/non-existent-guid/actions/cancel",
			statusCode:   http.StatusNotFound,
			response: map[string]any{
				testErrorsKey: []map[string]any{
					{
						testCodeKey:   10010,
						testTitleKey:  testNotFoundTitle,
						testDetailKey: "Task not found",
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
				assert.Equal(t, http.MethodPost, request.Method)
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(testCase.statusCode)
				_ = json.NewEncoder(writer).Encode(testCase.response)
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			task, err := client.Tasks().Cancel(context.Background(), testCase.guid)

			if testCase.wantErr {
				require.ErrorContains(t, err, testCase.errMessage)
				assert.Nil(t, task)
			} else {
				require.NoError(t, err)
				require.NotNil(t, task)
				assert.Equal(t, testCase.guid, task.GUID)
				assert.Equal(t, "CANCELING", task.State)
			}
		})
	}
}
