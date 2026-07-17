package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/fivetwenty-io/capi/v3/internal/client"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestAppUsageEvent() capi.AppUsageEvent {
	previousState := testStateStopped
	previousInstanceCount := 1
	previousMemoryInMB := 256
	buildpackName := "nodejs_buildpack"
	buildpackGUID := testBuildpackGUID
	taskName := testMigrateTaskName
	taskGUID := "task-guid"
	parentAppName := "parent-app"
	parentAppGUID := "parent-app-guid"

	return capi.AppUsageEvent{
		Resource: capi.Resource{
			GUID:      testEventGUID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		State:                         testStateStarted,
		PreviousState:                 &previousState,
		InstanceCount:                 2,
		PreviousInstanceCount:         &previousInstanceCount,
		MemoryInMBPerInstance:         512,
		PreviousMemoryInMBPerInstance: &previousMemoryInMB,
		AppName:                       testAppNameFixture,
		AppGUID:                       testAppGUID,
		SpaceName:                     testSpaceNameFixture,
		SpaceGUID:                     testSpaceGUID,
		OrganizationName:              testOrgNameFixture,
		OrganizationGUID:              testOrgGUID,
		BuildpackName:                 &buildpackName,
		BuildpackGUID:                 &buildpackGUID,
		ProcessType:                   testWebProcessType,
		TaskName:                      &taskName,
		TaskGUID:                      &taskGUID,
		ParentAppName:                 &parentAppName,
		ParentAppGUID:                 &parentAppGUID,
		Package: capi.AppUsageEventPackage{
			State: testStateReady,
		},
	}
}

func TestAppUsageEventsClient_Get(t *testing.T) {
	t.Parallel()

	event := createTestAppUsageEvent()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/app_usage_events/event-guid", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		_ = json.NewEncoder(writer).Encode(event)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	result, err := client.AppUsageEvents().Get(context.Background(), testEventGUID)
	require.NoError(t, err)
	assert.Equal(t, testEventGUID, result.GUID)
	assert.Equal(t, testStateStarted, result.State)
	assert.Equal(t, testStateStopped, *result.PreviousState)
	assert.Equal(t, 2, result.InstanceCount)
	assert.Equal(t, 1, *result.PreviousInstanceCount)
	assert.Equal(t, 512, result.MemoryInMBPerInstance)
	assert.Equal(t, 256, *result.PreviousMemoryInMBPerInstance)
	assert.Equal(t, testAppNameFixture, result.AppName)
	assert.Equal(t, "nodejs_buildpack", *result.BuildpackName)
}

func TestAppUsageEventsClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/app_usage_events", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "1", request.URL.Query().Get("page"))
		assert.Equal(t, "10", request.URL.Query().Get("per_page"))

		response := capi.ListResponse[capi.AppUsageEvent]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
			},
			Resources: []capi.AppUsageEvent{
				{
					Resource:              capi.Resource{GUID: "event-1"},
					State:                 testStateStarted,
					InstanceCount:         1,
					MemoryInMBPerInstance: 256,
					AppName:               testAppGUID1,
					AppGUID:               "app-guid-1",
					SpaceName:             testSpaceName1,
					SpaceGUID:             testSpaceGUID1,
					OrganizationName:      testOrgName1,
					OrganizationGUID:      testOrgGUID1,
					ProcessType:           testWebProcessType,
				},
				{
					Resource:              capi.Resource{GUID: "event-2"},
					State:                 testStateStopped,
					InstanceCount:         0,
					MemoryInMBPerInstance: 512,
					AppName:               testAppGUID2,
					AppGUID:               "app-guid-2",
					SpaceName:             testSpaceName2,
					SpaceGUID:             testSpaceGUID2,
					OrganizationName:      testOrgName2,
					OrganizationGUID:      testOrgGUID2,
					ProcessType:           testWorkerProcessType,
				},
			},
		}

		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	params := capi.NewQueryParams().WithPage(1).WithPerPage(10)
	result, err := client.AppUsageEvents().List(context.Background(), params)

	require.NoError(t, err)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "event-1", result.Resources[0].GUID)
	assert.Equal(t, "event-2", result.Resources[1].GUID)
	assert.Equal(t, testStateStarted, result.Resources[0].State)
	assert.Equal(t, testStateStopped, result.Resources[1].State)
}

func TestAppUsageEventsClient_PurgeAndReseed(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/app_usage_events/actions/destructively_purge_all_and_reseed", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		writer.WriteHeader(http.StatusAccepted)
		_, _ = writer.Write([]byte("{}"))
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	err = client.AppUsageEvents().PurgeAndReseed(context.Background())
	require.NoError(t, err)
}
