package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/fivetwenty-io/capi/v3/internal/client"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
	internalhttp "github.com/fivetwenty-io/capi/v3/internal/http"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestServiceInstancesClient_Create_Managed(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_instances", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var requestBody capi.ServiceInstanceCreateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.Equal(t, testManagedType, requestBody.Type)
		assert.Equal(t, testMyInstanceName, requestBody.Name)
		assert.Equal(t, testSpaceGUID, requestBody.Relationships.Space.Data.GUID)
		assert.Equal(t, testPlanGUID, requestBody.Relationships.ServicePlan.Data.GUID)

		// Managed instances return a job
		job := capi.Job{
			Resource: capi.Resource{
				GUID: testJobGUID,
			},
			Operation: "service_instance.create",
			State:     testStateProcessing,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Location", testJobPath)
		writer.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(writer).Encode(job)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceInstances := NewServiceInstancesClient(httpClient)

	request := &capi.ServiceInstanceCreateRequest{
		Type: testManagedType,
		Name: testMyInstanceName,
		Parameters: map[string]any{
			testFooKey: testBarValue,
		},
		Tags: []string{testTag1Value, "tag2"},
		Relationships: capi.ServiceInstanceRelationships{
			Space: capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: testSpaceGUID,
				},
			},
			ServicePlan: &capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: testPlanGUID,
				},
			},
		},
	}

	result, err := serviceInstances.Create(context.Background(), request)
	require.NoError(t, err)

	job, ok := result.(*capi.Job)
	require.True(t, ok, "Expected *capi.Job for managed instance")
	assert.Equal(t, testJobGUID, job.GUID)
	assert.Equal(t, "service_instance.create", job.Operation)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestServiceInstancesClient_Create_UserProvided(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_instances", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var requestBody capi.ServiceInstanceCreateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.Equal(t, testUserProvidedType, requestBody.Type)
		assert.Equal(t, "my-ups", requestBody.Name)
		assert.Equal(t, testSpaceGUID, requestBody.Relationships.Space.Data.GUID)

		// User-provided instances return the instance directly
		now := time.Now()
		instance := capi.ServiceInstance{
			Resource: capi.Resource{
				GUID:      testInstanceGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name: "my-ups",
			Type: testUserProvidedType,
			Tags: []string{testTag1Value},
			LastOperation: &capi.ServiceInstanceLastOperation{
				Type:        testCreateOperation,
				State:       testSucceededOperation,
				Description: "Operation succeeded",
				CreatedAt:   &now,
				UpdatedAt:   &now,
			},
			SyslogDrainURL:  requestBody.SyslogDrainURL,
			RouteServiceURL: requestBody.RouteServiceURL,
			Relationships: capi.ServiceInstanceRelationships{
				Space: capi.Relationship{
					Data: &capi.RelationshipData{
						GUID: testSpaceGUID,
					},
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(writer).Encode(instance)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceInstances := NewServiceInstancesClient(httpClient)

	syslogURL := "https://syslog.example.com"
	routeURL := "https://route.example.com"
	request := &capi.ServiceInstanceCreateRequest{
		Type: testUserProvidedType,
		Name: "my-ups",
		Credentials: map[string]any{
			testUsernameKey: testAdminUsername,
			"password":      "secret",
		},
		Tags:            []string{testTag1Value},
		SyslogDrainURL:  &syslogURL,
		RouteServiceURL: &routeURL,
		Relationships: capi.ServiceInstanceRelationships{
			Space: capi.Relationship{
				Data: &capi.RelationshipData{
					GUID: testSpaceGUID,
				},
			},
		},
	}

	result, err := serviceInstances.Create(context.Background(), request)
	require.NoError(t, err)

	instance, ok := result.(*capi.ServiceInstance)
	require.True(t, ok, "Expected *capi.ServiceInstance for user-provided instance")
	assert.Equal(t, testInstanceGUID, instance.GUID)
	assert.Equal(t, "my-ups", instance.Name)
	assert.Equal(t, testUserProvidedType, instance.Type)
}

func TestServiceInstancesClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_instances/instance-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		now := time.Now()
		instance := capi.ServiceInstance{
			Resource: capi.Resource{
				GUID:      testInstanceGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name: testMyInstanceName,
			Type: testManagedType,
			Tags: []string{testDatabaseFixture, "postgresql"},
			MaintenanceInfo: &capi.ServiceInstanceMaintenance{
				Version: testVersion100,
			},
			UpgradeAvailable: false,
			DashboardURL:     StringPtr("https://dashboard.example.com"),
			LastOperation: &capi.ServiceInstanceLastOperation{
				Type:        testCreateOperation,
				State:       testSucceededOperation,
				Description: "Instance created",
				CreatedAt:   &now,
				UpdatedAt:   &now,
			},
			Relationships: capi.ServiceInstanceRelationships{
				Space: capi.Relationship{
					Data: &capi.RelationshipData{
						GUID: testSpaceGUID,
					},
				},
				ServicePlan: &capi.Relationship{
					Data: &capi.RelationshipData{
						GUID: testPlanGUID,
					},
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(instance)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceInstances := NewServiceInstancesClient(httpClient)

	instance, err := serviceInstances.Get(context.Background(), testInstanceGUID)
	require.NoError(t, err)
	assert.NotNil(t, instance)
	assert.Equal(t, testInstanceGUID, instance.GUID)
	assert.Equal(t, testMyInstanceName, instance.Name)
	assert.Equal(t, testManagedType, instance.Type)
	assert.Contains(t, instance.Tags, testDatabaseFixture)
	assert.Contains(t, instance.Tags, "postgresql")
}

//nolint:dupl // Acceptable duplication - each test validates different endpoints with different query params and assertions
func TestServiceInstancesClient_List(t *testing.T) {
	t.Parallel()

	now := time.Now()
	responseData := []capi.ServiceInstance{
		{
			Resource: capi.Resource{
				GUID:      "instance-guid-1",
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name: "my-instance-1",
			Type: testManagedType,
		},
		{
			Resource: capi.Resource{
				GUID:      "instance-guid-2",
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name: "my-instance-2",
			Type: testUserProvidedType,
		},
	}

	RunServiceListTest(t, "service instances list", "/v3/service_instances",
		func(request *http.Request) {
			assert.Equal(t, testSpaceGUID, request.URL.Query().Get(testSpaceGUIDsParam))
			assert.Equal(t, testMyInstanceName, request.URL.Query().Get(testNamesParam))
		},
		responseData,
		func(httpClient *internalhttp.Client) any {
			return NewServiceInstancesClient(httpClient)
		},
		func(client any) (*capi.ListResponse[capi.ServiceInstance], error) {
			params := &capi.QueryParams{
				Filters: map[string][]string{
					testSpaceGUIDsParam: {testSpaceGUID},
					testNamesParam:      {testMyInstanceName},
				},
			}

			serviceInstancesClient, ok := client.(*ServiceInstancesClient)
			if !ok {
				return nil, constants.ErrInvalidClientType
			}

			return serviceInstancesClient.List(context.Background(), params)
		},
		func(resources []capi.ServiceInstance) {
			assert.Equal(t, "my-instance-1", resources[0].Name)
			assert.Equal(t, testManagedType, resources[0].Type)
			assert.Equal(t, "my-instance-2", resources[1].Name)
			assert.Equal(t, testUserProvidedType, resources[1].Type)
		},
	)
}

func TestServiceInstancesClient_Update_Managed(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_instances/instance-guid", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var requestBody capi.ServiceInstanceUpdateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		// Managed instances return a job for updates
		job := capi.Job{
			Resource: capi.Resource{
				GUID: testJobGUID,
			},
			Operation: "service_instance.update",
			State:     testStateProcessing,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Location", testJobPath)
		writer.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(writer).Encode(job)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceInstances := NewServiceInstancesClient(httpClient)

	newName := "updated-instance"
	request := &capi.ServiceInstanceUpdateRequest{
		Name: &newName,
		Parameters: map[string]any{
			"max_connections": 100,
		},
		Tags: []string{testUpdatedValue, "tags"},
	}

	result, err := serviceInstances.Update(context.Background(), testInstanceGUID, request)
	require.NoError(t, err)

	job, ok := result.(*capi.Job)
	require.True(t, ok, "Expected *capi.Job for managed instance update")
	assert.Equal(t, testJobGUID, job.GUID)
	assert.Equal(t, "service_instance.update", job.Operation)
}

func TestServiceInstancesClient_Update_UserProvided(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_instances/instance-guid", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		// Check for user-provided instance by examining the request
		var requestBody map[string]any

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		// User-provided instances return the updated instance directly
		now := time.Now()
		instance := capi.ServiceInstance{
			Resource: capi.Resource{
				GUID:      testInstanceGUID,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name: "updated-ups",
			Type: testUserProvidedType,
			Tags: []string{testUpdatedValue},
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(instance)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceInstances := NewServiceInstancesClient(httpClient)

	newName := "updated-ups"
	request := &capi.ServiceInstanceUpdateRequest{
		Name: &newName,
		Credentials: map[string]any{
			testUsernameKey: "newuser",
		},
		Tags: []string{testUpdatedValue},
	}

	result, err := serviceInstances.Update(context.Background(), testInstanceGUID, request)
	require.NoError(t, err)

	instance, ok := result.(*capi.ServiceInstance)
	require.True(t, ok, "Expected *capi.ServiceInstance for user-provided instance update")
	assert.Equal(t, testInstanceGUID, instance.GUID)
	assert.Equal(t, "updated-ups", instance.Name)
}

func TestServiceInstancesClient_Delete(t *testing.T) {
	t.Parallel()

	// CF v3 DELETE /v3/service_instances/{guid} (without purge) is async:
	// 202 Accepted, empty body, Location header pointing at /v3/jobs/{jobGuid}.
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_instances/instance-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)
		// Without WithPurge, the purge query parameter must NOT be present.
		assert.Empty(t, request.URL.Query().Get("purge"), "purge param must be absent when WithPurge is not passed")

		writer.Header().Set("Location", testJobPath)
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceInstances := NewServiceInstancesClient(httpClient)

	job, err := serviceInstances.Delete(context.Background(), testInstanceGUID)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
}

func TestServiceInstancesClient_DeleteWithPurge(t *testing.T) {
	t.Parallel()

	// CF v3 DELETE /v3/service_instances/{guid}?purge=true is the sync
	// purge path: 204 No Content with no Location header (broker bypass).
	// The client surfaces this as `nil, nil` — callers treat that as
	// "delete completed synchronously, no job polling needed."
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_instances/instance-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)
		assert.Equal(t, testTrueString, request.URL.Query().Get("purge"), "purge param must be 'true' when WithPurge(true) is passed")

		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceInstances := NewServiceInstancesClient(httpClient)

	job, err := serviceInstances.Delete(context.Background(), testInstanceGUID, capi.WithPurge(true))
	require.NoError(t, err)
	assert.Nil(t, job)
}

func TestServiceInstancesClient_DeleteWithPurgeFalse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_instances/instance-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)
		// WithPurge(false) must NOT set the purge query parameter.
		assert.Empty(t, request.URL.Query().Get("purge"), "purge param must be absent when WithPurge(false) is passed")

		writer.Header().Set("Location", testJobPath)
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceInstances := NewServiceInstancesClient(httpClient)

	job, err := serviceInstances.Delete(context.Background(), testInstanceGUID, capi.WithPurge(false))
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
}

func TestServiceInstancesClient_GetParameters(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_instances/instance-guid/parameters", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		// CF returns parameters as a bare top-level object, not wrapped in
		// {"parameters": ...} — assert against that real wire shape.
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"max_connections":100,"enable_ssl":true,"database_name":"mydb"}`))
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceInstances := NewServiceInstancesClient(httpClient)

	params, err := serviceInstances.GetParameters(context.Background(), testInstanceGUID)
	require.NoError(t, err)
	assert.NotNil(t, params)
	assert.InDelta(t, float64(100), params.Parameters["max_connections"], 0)
	assert.Equal(t, true, params.Parameters["enable_ssl"])
	assert.Equal(t, "mydb", params.Parameters["database_name"])
}

func TestServiceInstancesClient_GetCredentials(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_instances/instance-guid/credentials", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		// CF returns credentials as a bare top-level object (the values the
		// UPS was created with), not wrapped in {"credentials": ...}.
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"username":"my-username","password":"super-secret","other":"credential"}`))
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceInstances := NewServiceInstancesClient(httpClient)

	creds, err := serviceInstances.GetCredentials(context.Background(), testInstanceGUID)
	require.NoError(t, err)
	assert.NotNil(t, creds)
	assert.Equal(t, "my-username", creds.Credentials[testUsernameKey])
	assert.Equal(t, "super-secret", creds.Credentials["password"])
	assert.Equal(t, "credential", creds.Credentials["other"])
}

func TestServiceInstancesClient_ListSharedSpaces(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_instances/instance-guid/relationships/shared_spaces", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		relationships := capi.ServiceInstanceSharedSpacesRelationships{
			Data: []capi.Relationship{
				{Data: &capi.RelationshipData{GUID: testSpaceGUID1}},
				{Data: &capi.RelationshipData{GUID: testSpaceGUID2}},
			},
			Links: capi.Links{
				testSelfKey: capi.Link{
					Href: "/v3/service_instances/instance-guid/relationships/shared_spaces",
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(relationships)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceInstances := NewServiceInstancesClient(httpClient)

	sharedSpaces, err := serviceInstances.ListSharedSpaces(context.Background(), testInstanceGUID)
	require.NoError(t, err)
	assert.NotNil(t, sharedSpaces)
	assert.Len(t, sharedSpaces.Data, 2)
	assert.Equal(t, testSpaceGUID1, sharedSpaces.Data[0].Data.GUID)
	assert.Equal(t, testSpaceGUID2, sharedSpaces.Data[1].Data.GUID)
}

func TestServiceInstancesClient_ShareWithSpaces(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_instances/instance-guid/relationships/shared_spaces", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		var requestBody capi.ServiceInstanceShareRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)
		assert.Len(t, requestBody.Data, 2)

		relationships := capi.ServiceInstanceSharedSpacesRelationships{
			Data: []capi.Relationship{
				{Data: &capi.RelationshipData{GUID: testSpaceGUID1}},
				{Data: &capi.RelationshipData{GUID: testSpaceGUID2}},
				{Data: &capi.RelationshipData{GUID: "space-guid-3"}},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(relationships)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceInstances := NewServiceInstancesClient(httpClient)

	request := &capi.ServiceInstanceShareRequest{
		Data: []capi.Relationship{
			{Data: &capi.RelationshipData{GUID: testSpaceGUID2}},
			{Data: &capi.RelationshipData{GUID: "space-guid-3"}},
		},
	}

	sharedSpaces, err := serviceInstances.ShareWithSpaces(context.Background(), testInstanceGUID, request)
	require.NoError(t, err)
	assert.NotNil(t, sharedSpaces)
	assert.Len(t, sharedSpaces.Data, 3)
}

func TestServiceInstancesClient_UnshareFromSpace(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_instances/instance-guid/relationships/shared_spaces/space-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceInstances := NewServiceInstancesClient(httpClient)

	err := serviceInstances.UnshareFromSpace(context.Background(), testInstanceGUID, testSpaceGUID)
	require.NoError(t, err)
}

func TestServiceInstancesClient_GetNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_instances/instance-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceInstances := NewServiceInstancesClient(httpClient)

	instance, err := serviceInstances.Get(context.Background(), testInstanceGUID)
	require.Error(t, err)
	assert.Nil(t, instance)
}

func TestServiceInstancesClient_DeleteWithBindings(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_instances/instance-guid", request.URL.Path)
		assert.Equal(t, "DELETE", request.Method)

		writer.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	serviceInstances := NewServiceInstancesClient(httpClient)

	job, err := serviceInstances.Delete(context.Background(), testInstanceGUID)
	require.Error(t, err)
	assert.Nil(t, job)
}

func TestServiceInstancesClient_GetWithFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "name,guid", request.URL.Query().Get("fields[space.organization]"))
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
		  "guid": "si-guid",
		  "included": {"organizations": [{"guid": "org-1", "name": "acme"}]}
		}`))
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	instances := NewServiceInstancesClient(httpClient)

	instance, err := instances.Get(context.Background(), "si-guid",
		capi.WithServiceInstanceFields(capi.ServiceInstanceFieldsSpaceOrganization, "name", "guid"))
	require.NoError(t, err)
	require.NotNil(t, instance.Included)
	assert.Equal(t, testOrgName1, instance.Included.Organizations[0].GUID)
}
