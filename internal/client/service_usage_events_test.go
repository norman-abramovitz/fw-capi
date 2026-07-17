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

func TestServiceUsageEventsClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_usage_events/event-guid", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		event := capi.ServiceUsageEvent{
			Resource: capi.Resource{
				GUID:      testEventGUID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			State:               "CREATED",
			ServiceInstanceName: "my-db",
			ServiceInstanceGUID: "service-instance-guid",
			ServiceInstanceType: "managed_service_instance",
			ServicePlanName:     "premium",
			ServicePlanGUID:     "plan-guid-2",
			ServiceOfferingName: "postgres",
			ServiceOfferingGUID: testOfferingGUID,
			ServiceBrokerName:   "postgres-broker",
			ServiceBrokerGUID:   testBrokerGUID,
			SpaceName:           testSpaceNameFixture,
			SpaceGUID:           testSpaceGUID,
			OrganizationName:    testOrgNameFixture,
			OrganizationGUID:    testOrgGUID,
		}

		_ = json.NewEncoder(writer).Encode(event)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	event, err := client.ServiceUsageEvents().Get(context.Background(), testEventGUID)
	require.NoError(t, err)
	assert.Equal(t, testEventGUID, event.GUID)
	assert.Equal(t, "CREATED", event.State)
	assert.Equal(t, "my-db", event.ServiceInstanceName)
	assert.Equal(t, "premium", event.ServicePlanName)
	assert.Equal(t, "postgres", event.ServiceOfferingName)
	assert.Equal(t, "postgres-broker", event.ServiceBrokerName)
}

func TestServiceUsageEventsClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_usage_events", request.URL.Path)
		assert.Equal(t, "GET", request.Method)
		assert.Equal(t, "1", request.URL.Query().Get("page"))
		assert.Equal(t, "10", request.URL.Query().Get("per_page"))

		response := capi.ListResponse[capi.ServiceUsageEvent]{
			Pagination: capi.Pagination{
				TotalResults: 2,
				TotalPages:   1,
			},
			Resources: []capi.ServiceUsageEvent{
				{
					Resource:            capi.Resource{GUID: "event-1"},
					State:               "CREATED",
					ServiceInstanceName: "service-1",
					ServiceInstanceGUID: "service-instance-guid-1",
					ServiceInstanceType: "managed_service_instance",
					ServicePlanName:     testBasicAuthType,
					ServicePlanGUID:     "plan-guid-1",
					ServiceOfferingName: "redis",
					ServiceOfferingGUID: "offering-guid-1",
					ServiceBrokerName:   "redis-broker",
					ServiceBrokerGUID:   "broker-guid-1",
					SpaceName:           testSpaceName1,
					SpaceGUID:           testSpaceGUID1,
					OrganizationName:    testOrgName1,
					OrganizationGUID:    testOrgGUID1,
				},
				{
					Resource:            capi.Resource{GUID: "event-2"},
					State:               "DELETED",
					ServiceInstanceName: "service-2",
					ServiceInstanceGUID: "service-instance-guid-2",
					ServiceInstanceType: "user_provided_service_instance",
					ServicePlanName:     "",
					ServicePlanGUID:     "",
					ServiceOfferingName: "",
					ServiceOfferingGUID: "",
					ServiceBrokerName:   "",
					ServiceBrokerGUID:   "",
					SpaceName:           testSpaceName2,
					SpaceGUID:           testSpaceGUID2,
					OrganizationName:    testOrgName2,
					OrganizationGUID:    testOrgGUID2,
				},
			},
		}

		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	params := capi.NewQueryParams().WithPage(1).WithPerPage(10)
	result, err := client.ServiceUsageEvents().List(context.Background(), params)

	require.NoError(t, err)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, "event-1", result.Resources[0].GUID)
	assert.Equal(t, "event-2", result.Resources[1].GUID)
	assert.Equal(t, "CREATED", result.Resources[0].State)
	assert.Equal(t, "DELETED", result.Resources[1].State)
	assert.Equal(t, "managed_service_instance", result.Resources[0].ServiceInstanceType)
	assert.Equal(t, "user_provided_service_instance", result.Resources[1].ServiceInstanceType)
}

func TestServiceUsageEventsClient_PurgeAndReseed(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/service_usage_events/actions/destructively_purge_all_and_reseed", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		writer.WriteHeader(http.StatusAccepted)
		_, _ = writer.Write([]byte("{}"))
	}))
	defer server.Close()

	client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
	require.NoError(t, err)

	err = client.ServiceUsageEvents().PurgeAndReseed(context.Background())
	require.NoError(t, err)
}
