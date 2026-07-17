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

// Test constants for service broker tests.
const (
	testServiceBrokersPath    = "/v3/service_brokers"
	testServiceBrokerGUIDPath = "/v3/service_brokers/test-broker-guid"
)

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestServiceBrokersClient_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		request      *capi.ServiceBrokerCreateRequest
		response     any
		statusCode   int
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "create global service broker",
			expectedPath: testServiceBrokersPath,
			statusCode:   http.StatusAccepted,
			request: &capi.ServiceBrokerCreateRequest{
				Name: testServiceBrokerName,
				URL:  testServiceBrokerURL,
				Authentication: capi.ServiceBrokerAuthentication{
					Type: testBasicAuthType,
					Credentials: capi.ServiceBrokerAuthenticationCredentials{
						Username: testAdminUsername,
						Password: "secretpassword",
					},
				},
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						testTypeKey: testDevelopmentLabel,
					},
				},
			},
			response: capi.Job{
				Resource: capi.Resource{
					GUID:      testJobGUID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						testSelfKey: capi.Link{
							Href: "https://api.example.org/v3/jobs/job-guid",
						},
					},
				},
				Operation: "service_broker.create",
				State:     testStateProcessing,
			},
			wantErr: false,
		},
		{
			name:         "create space-scoped service broker",
			expectedPath: testServiceBrokersPath,
			statusCode:   http.StatusAccepted,
			request: &capi.ServiceBrokerCreateRequest{
				Name: "space-broker",
				URL:  "https://space-broker.example.com",
				Authentication: capi.ServiceBrokerAuthentication{
					Type: testBasicAuthType,
					Credentials: capi.ServiceBrokerAuthenticationCredentials{
						Username: testUserUsername,
						Password: testPassPassword,
					},
				},
				Relationships: &capi.ServiceBrokerRelationships{
					Space: &capi.Relationship{
						Data: &capi.RelationshipData{
							GUID: testSpaceGUID,
						},
					},
				},
			},
			response: capi.Job{
				Resource: capi.Resource{
					GUID: testJobGUID,
				},
				Operation: "service_broker.create",
				State:     testStateProcessing,
			},
			wantErr: false,
		},
		{
			name:         "service broker already exists",
			expectedPath: testServiceBrokersPath,
			statusCode:   http.StatusUnprocessableEntity,
			request: &capi.ServiceBrokerCreateRequest{
				Name: "existing-broker",
				URL:  "https://existing.example.com",
				Authentication: capi.ServiceBrokerAuthentication{
					Type: testBasicAuthType,
					Credentials: capi.ServiceBrokerAuthenticationCredentials{
						Username: testUserUsername,
						Password: testPassPassword,
					},
				},
			},
			response: map[string]any{
				testErrorsKey: []map[string]any{
					{
						testCodeKey:   10008,
						testTitleKey:  testUnprocessableTitle,
						testDetailKey: "Service broker with name existing-broker already exists",
					},
				},
			},
			wantErr:    true,
			errMessage: testUnprocessableTitle,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, testCase.expectedPath, request.URL.Path)
				assert.Equal(t, http.MethodPost, request.Method)

				var requestBody capi.ServiceBrokerCreateRequest

				err := json.NewDecoder(request.Body).Decode(&requestBody)
				assert.NoError(t, err)

				writer.Header().Set("Content-Type", "application/json")

				if testCase.statusCode == http.StatusAccepted {
					writer.Header().Set("Location", "https://api.example.org/v3/jobs/job-guid")
				}

				writer.WriteHeader(testCase.statusCode)
				_ = json.NewEncoder(writer).Encode(testCase.response)
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			job, err := client.ServiceBrokers().Create(context.Background(), testCase.request)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMessage)
				assert.Nil(t, job)
			} else {
				require.NoError(t, err)
				require.NotNil(t, job)
				assert.NotEmpty(t, job.GUID)
				assert.Equal(t, "service_broker.create", job.Operation)
			}
		})
	}
}

//nolint:dupl,funlen // Acceptable duplication - each test validates different endpoints with different assertions
func TestServiceBrokersClient_Get(t *testing.T) {
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
			guid:         testBrokerGUIDFixture,
			expectedPath: testServiceBrokerGUIDPath,
			statusCode:   http.StatusOK,
			response: capi.ServiceBroker{
				Resource: capi.Resource{
					GUID:      testBrokerGUIDFixture,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Links: capi.Links{
						testSelfKey: capi.Link{
							Href: "https://api.example.org/v3/service_brokers/test-broker-guid",
						},
						"service_offerings": capi.Link{
							Href: "https://api.example.org/v3/service_offerings?service_broker_guids=test-broker-guid",
						},
					},
				},
				Name: testServiceBrokerName,
				URL:  testServiceBrokerURL,
				Relationships: capi.ServiceBrokerRelationships{
					Space: nil,
				},
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						testTypeKey: testDevelopmentLabel,
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "broker not found",
			guid:         testNonExistentGUID,
			expectedPath: "/v3/service_brokers/non-existent-guid",
			statusCode:   http.StatusNotFound,
			response: map[string]any{
				testErrorsKey: []map[string]any{
					{
						testCodeKey:   10010,
						testTitleKey:  testNotFoundTitle,
						testDetailKey: "Service broker not found",
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

			broker, err := client.ServiceBrokers().Get(context.Background(), testCase.guid)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMessage)
				assert.Nil(t, broker)
			} else {
				require.NoError(t, err)
				require.NotNil(t, broker)
				assert.Equal(t, testCase.guid, broker.GUID)
				assert.Equal(t, testServiceBrokerName, broker.Name)
			}
		})
	}
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestServiceBrokersClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testServiceBrokersPath, request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		// Check query parameters if present
		query := request.URL.Query()
		if names := query.Get(testNamesParam); names != "" {
			assert.Equal(t, "broker1,broker2", names)
		}

		if spaceGuids := query.Get(testSpaceGUIDsParam); spaceGuids != "" {
			assert.Equal(t, "space-1,space-2", spaceGuids)
		}

		response := capi.ListResponse[capi.ServiceBroker]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
				First:        capi.Link{Href: "https://api.example.org/v3/service_brokers?page=1"},
				Last:         capi.Link{Href: "https://api.example.org/v3/service_brokers?page=1"},
				Next:         nil,
				Previous:     nil,
			},
			Resources: []capi.ServiceBroker{
				{
					Resource: capi.Resource{
						GUID:      testBrokerName1,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "global-broker",
					URL:  "https://global-broker.example.com",
				},
				{
					Resource: capi.Resource{
						GUID:      testBrokerName2,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					Name: "space-broker",
					URL:  "https://space-broker.example.com",
					Relationships: capi.ServiceBrokerRelationships{
						Space: &capi.Relationship{
							Data: &capi.RelationshipData{
								GUID: testSpaceGUID,
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

	// Test without filters
	result, err := client.ServiceBrokers().List(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, result.Pagination.TotalResults)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, testBrokerName1, result.Resources[0].GUID)
	assert.Equal(t, "global-broker", result.Resources[0].Name)

	// Test with filters
	params := &capi.QueryParams{
		Filters: map[string][]string{
			testNamesParam:      {"broker1", "broker2"},
			testSpaceGUIDsParam: {testSpaceName1, testSpaceName2},
		},
	}
	result, err = client.ServiceBrokers().List(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, result)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestServiceBrokersClient_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		guid         string
		request      *capi.ServiceBrokerUpdateRequest
		response     any
		statusCode   int
		withJob      bool
		expectedPath string
		wantErr      bool
		errMessage   string
	}{
		{
			name:         "update with job (URL change)",
			guid:         testBrokerGUIDFixture,
			expectedPath: testServiceBrokerGUIDPath,
			statusCode:   http.StatusAccepted,
			withJob:      true,
			request: &capi.ServiceBrokerUpdateRequest{
				URL: StringPtr("https://newriter.service-broker.com"),
				Authentication: &capi.ServiceBrokerAuthentication{
					Type: testBasicAuthType,
					Credentials: capi.ServiceBrokerAuthenticationCredentials{
						Username: "newuser",
						Password: "newpass",
					},
				},
			},
			response: capi.Job{
				Resource: capi.Resource{
					GUID: testJobGUID,
				},
				Operation: "service_broker.update",
				State:     testStateProcessing,
			},
			wantErr: false,
		},
		{
			name:         "update without job (metadata only)",
			guid:         testBrokerGUIDFixture,
			expectedPath: testServiceBrokerGUIDPath,
			statusCode:   http.StatusOK,
			withJob:      false,
			request: &capi.ServiceBrokerUpdateRequest{
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						testEnvironmentLabelKey: testProductionLabel,
					},
				},
			},
			response: capi.ServiceBroker{
				Resource: capi.Resource{
					GUID:      testBrokerGUIDFixture,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: testServiceBrokerName,
				URL:  testServiceBrokerURL,
				Metadata: &capi.Metadata{
					Labels: map[string]string{
						testEnvironmentLabelKey: testProductionLabel,
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "update with synchronization in progress",
			guid:         testBrokerGUIDFixture,
			expectedPath: testServiceBrokerGUIDPath,
			statusCode:   http.StatusUnprocessableEntity,
			request: &capi.ServiceBrokerUpdateRequest{
				URL: StringPtr("https://newriter.service-broker.com"),
			},
			response: map[string]any{
				testErrorsKey: []map[string]any{
					{
						testCodeKey:   10008,
						testTitleKey:  testUnprocessableTitle,
						testDetailKey: "Cannot update service broker while synchronization is in progress",
					},
				},
			},
			wantErr:    true,
			errMessage: testUnprocessableTitle,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, testCase.expectedPath, request.URL.Path)
				assert.Equal(t, "PATCH", request.Method)

				var requestBody capi.ServiceBrokerUpdateRequest

				err := json.NewDecoder(request.Body).Decode(&requestBody)
				assert.NoError(t, err)

				writer.Header().Set("Content-Type", "application/json")

				if testCase.withJob && testCase.statusCode == http.StatusAccepted {
					writer.Header().Set("Location", "https://api.example.org/v3/jobs/job-guid")
				}

				writer.WriteHeader(testCase.statusCode)
				_ = json.NewEncoder(writer).Encode(testCase.response)
			}))
			defer server.Close()

			client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
			require.NoError(t, err)

			result, err := client.ServiceBrokers().Update(context.Background(), testCase.guid, testCase.request)

			if testCase.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMessage)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				if testCase.withJob {
					job := result
					assert.Equal(t, testJobGUID, job.GUID)
					assert.Equal(t, "service_broker.update", job.Operation)
				}
			}
		})
	}
}

func TestServiceBrokersClient_Delete(t *testing.T) {
	t.Parallel()

	// CF v3 DELETE /v3/service_brokers/{guid} is async: 202 Accepted, empty
	// body, Location header pointing at /v3/jobs/{jobGuid}.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testServiceBrokerGUIDPath, request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.Header().Set("Location", "https://api.example.org/v3/jobs/job-guid")
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	job, err := client.ServiceBrokers().Delete(context.Background(), testBrokerGUIDFixture)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
}
