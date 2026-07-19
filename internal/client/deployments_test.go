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

// Test constants for deployment tests.
const (
	testDeploymentsPath        = "/v3/deployments"
	testDeploymentActiveStatus = "ACTIVE"
	testDeploymentGUID         = "test-deployment-guid"
)

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestDeploymentsClient_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		request      *capi.DeploymentCreateRequest
		response     any
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "create deployment with droplet",
			expectedPath: testDeploymentsPath,
			statusCode:   http.StatusCreated,
			request: &capi.DeploymentCreateRequest{
				Droplet: &capi.DeploymentDropletRef{
					GUID: testDropletGUID,
				},
				Relationships: capi.DeploymentRelationships{
					App: &capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: testAppGUID,
						},
					},
				},
				Metadata: &capi.Metadata{
					Labels: capi.StringMap(map[string]string{
						testVersionAnnotationKey: "v1.0.0",
					}),
				},
			},
			response: &capi.Deployment{
				Resource: capi.Resource{
					GUID:      "deployment-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						testSelfKey: capi.Link{
							Href: "https://api.example.org/v3/deployments/deployment-guid",
						},
						testAppKey: capi.Link{
							Href: testHrefAppLink,
						},
						"cancel": capi.Link{
							Href:   "https://api.example.org/v3/deployments/deployment-guid/actions/cancel",
							Method: http.MethodPost,
						},
					},
				},
				State: testStateDeploying,
				Status: capi.DeploymentStatus{
					Value:  testDeploymentActiveStatus,
					Reason: testStateDeploying,
				},
				Strategy: testRollingStrategy,
				Droplet: &capi.DeploymentDropletRef{
					GUID: testDropletGUID,
				},
				PreviousDroplet: &capi.DeploymentDropletRef{
					GUID: "previous-droplet-guid",
				},
				NewProcesses: []capi.DeploymentProcess{
					{
						GUID: testProcessGUID,
						Type: testWebProcessType,
					},
				},
				Relationships: &capi.DeploymentRelationships{
					App: &capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: testAppGUID,
						},
					},
				},
				Metadata: &capi.Metadata{
					Labels: capi.StringMap(map[string]string{
						testVersionAnnotationKey: "v1.0.0",
					}),
				},
			},
			wantErr: false,
		},
		{
			name:         "create deployment with revision",
			expectedPath: testDeploymentsPath,
			statusCode:   http.StatusCreated,
			request: &capi.DeploymentCreateRequest{
				Revision: &capi.DeploymentRevisionRef{
					GUID:    testRevisionGUID,
					Version: 42,
				},
				Strategy: StringPtr("canary"),
				Options: &capi.DeploymentOptions{
					MaxInFlight: intPtr(2),
					Canary: &capi.DeploymentCanaryOptions{
						Steps: []capi.DeploymentCanaryStep{
							{Instances: 1, WaitTime: 60},
							{Instances: 5, WaitTime: 120},
						},
					},
				},
				Relationships: capi.DeploymentRelationships{
					App: &capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: testAppGUID,
						},
					},
				},
			},
			response: capi.Deployment{
				Resource: capi.Resource{
					GUID:      "deployment-guid",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				State: testStateDeploying,
				Status: capi.DeploymentStatus{
					Value:  testDeploymentActiveStatus,
					Reason: testStateDeploying,
					Canary: &capi.DeploymentCanaryStatus{
						Steps: capi.DeploymentCanarySteps{
							Current: 1,
							Total:   2,
						},
					},
				},
				Strategy: "canary",
				Revision: &capi.DeploymentRevisionRef{
					GUID:    testRevisionGUID,
					Version: 42,
				},
			},
			wantErr: false,
		},
		{
			name:         testMissingAppRelationshipCase,
			expectedPath: testDeploymentsPath,
			statusCode:   http.StatusUnprocessableEntity,
			request: &capi.DeploymentCreateRequest{
				Droplet: &capi.DeploymentDropletRef{
					GUID: testDropletGUID,
				},
				Relationships: capi.DeploymentRelationships{},
			},
			response: map[string]any{
				testErrorsKey: []map[string]any{
					{
						testCodeKey:   10008,
						testTitleKey:  testUnprocessableTitle,
						testDetailKey: testAppRelationshipRequired,
					},
				},
			},
			wantErr:    true,
			errMessage: testUnprocessableTitle,
		},
	}

	runCreateTestsForDeployments(t, tests)
}

func TestDeploymentsClient_Get(t *testing.T) {
	t.Parallel()

	tests := []TestGetOperation[capi.Deployment]{
		{
			Name:         testSuccessfulGetCase,
			GUID:         testDeploymentGUID,
			ExpectedPath: "/v3/deployments/test-deployment-guid",
			StatusCode:   http.StatusOK,
			Response: &capi.Deployment{
				Resource: capi.Resource{
					GUID:      testDeploymentGUID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				State: testStateDeployed,
				Status: capi.DeploymentStatus{
					Value:  testStateFinalized,
					Reason: testStateDeployed,
					Details: &capi.DeploymentStatusDetails{
						LastHealthyAt: timePtr(time.Now()),
					},
				},
				Strategy: testRollingStrategy,
				Droplet: &capi.DeploymentDropletRef{
					GUID: testDropletGUID,
				},
			},
			WantErr: false,
		},
		{
			Name:         "deployment not found",
			GUID:         testNonExistentGUID,
			ExpectedPath: "/v3/deployments/non-existent-guid",
			StatusCode:   http.StatusNotFound,
			Response: &capi.Deployment{
				Resource: capi.Resource{
					GUID:      testDeploymentGUID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				State: testStateDeployed,
			},
			WantErr:    true,
			ErrMessage: testNotFoundTitle,
		},
	}

	RunGetTests(t, tests, func(c *Client) func(context.Context, string) (*capi.Deployment, error) {
		return c.Deployments().Get
	})
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestDeploymentsClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testDeploymentsPath, request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		// Check query parameters if present
		query := request.URL.Query()
		if appGuids := query.Get(testAppGUIDsParam); appGuids != "" {
			assert.Equal(t, "app-1,app-2", appGuids)
		}

		if states := query.Get(testStatesParam); states != "" {
			assert.Equal(t, "DEPLOYING,DEPLOYED", states)
		}

		if statusReasons := query.Get("status_reasons"); statusReasons != "" {
			assert.Equal(t, "DEPLOYING,DEPLOYED", statusReasons)
		}

		if statusValues := query.Get("status_values"); statusValues != "" {
			assert.Equal(t, "ACTIVE,FINALIZED", statusValues)
		}

		response := capi.ListResponse[capi.Deployment]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "https://api.example.org/v3/deployments?page=1"},
				Last:         capi.Link{Href: "https://api.example.org/v3/deployments?page=1"},
				Next:         nil,
				Previous:     nil,
			},
			Resources: []capi.Deployment{
				{
					Resource: capi.Resource{
						GUID:      "deployment-1",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					State: testStateDeploying,
					Status: capi.DeploymentStatus{
						Value:  testDeploymentActiveStatus,
						Reason: testStateDeploying,
					},
					Strategy: testRollingStrategy,
				},
				{
					Resource: capi.Resource{
						GUID:      "deployment-2",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					State: testStateDeployed,
					Status: capi.DeploymentStatus{
						Value:  testStateFinalized,
						Reason: testStateDeployed,
					},
					Strategy: testRollingStrategy,
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
	result, err := client.Deployments().List(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.TotalResults)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "deployment-1", result.Resources[0].GUID)
	assert.Equal(t, testStateDeploying, result.Resources[0].State)

	// Test with filters
	params := &capi.QueryParams{
		Filters: map[string][]string{
			testAppGUIDsParam: {testAppGUID1, testAppGUID2},
			testStatesParam:   {testStateDeploying, testStateDeployed},
			"status_reasons":  {testStateDeploying, testStateDeployed},
			"status_values":   {testDeploymentActiveStatus, testStateFinalized},
		},
	}
	result, err = client.Deployments().List(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
}

//nolint:dupl // Acceptable duplication - each test validates different endpoints with different request/response types
func TestDeploymentsClient_Update(t *testing.T) {
	t.Parallel()

	request := &capi.DeploymentUpdateRequest{
		Metadata: &capi.Metadata{
			Labels: capi.StringMap(map[string]string{
				testVersionAnnotationKey: "v1.0.1",
			}),
			Annotations: capi.StringMap(map[string]string{
				testNoteAnnotationKey: "Updated deployment",
			}),
		},
	}

	response := &capi.Deployment{
		Resource: capi.Resource{
			GUID:      testDeploymentGUID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		State: testStateDeploying,
		Metadata: &capi.Metadata{
			Labels: capi.StringMap(map[string]string{
				testVersionAnnotationKey: "v1.0.1",
			}),
			Annotations: capi.StringMap(map[string]string{
				testNoteAnnotationKey: "Updated deployment",
			}),
		},
	}

	RunStandardUpdateTest(t, "deployment", testDeploymentGUID, "/v3/deployments/test-deployment-guid", request, response,
		func(c *Client) func(context.Context, string, *capi.DeploymentUpdateRequest) (*capi.Deployment, error) {
			return c.Deployments().Update
		})
}

func TestDeploymentsClient_Cancel(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/deployments/test-deployment-guid/actions/cancel", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	err = c.Deployments().Cancel(context.Background(), testDeploymentGUID)
	require.NoError(t, err)
}

func TestDeploymentsClient_Continue(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/deployments/test-deployment-guid/actions/continue", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	err = c.Deployments().Continue(context.Background(), testDeploymentGUID)
	require.NoError(t, err)
}

// runCreateTestsForDeployments runs deployment create tests.
func runCreateTestsForDeployments(t *testing.T, tests []struct {
	name         string
	request      *capi.DeploymentCreateRequest
	response     any
	statusCode   int
	expectedPath string
	wantErr      bool
	errMessage   string
}) {
	t.Helper()

	for _, testCase := range tests {
		RunCreateTestWithValidation(t, testCase.name, testCase.expectedPath, testCase.statusCode, testCase.response, testCase.wantErr, testCase.errMessage, func(c *Client) error {
			deployment, err := c.Deployments().Create(context.Background(), testCase.request)
			if err == nil {
				assert.NotEmpty(t, deployment.GUID)
				assert.NotEmpty(t, deployment.State)
			}

			if err != nil {
				return fmt.Errorf("failed to create deployment: %w", err)
			}

			return nil
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
