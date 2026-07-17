package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/fivetwenty-io/capi/v3/internal/client"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
)

// captureQueryServer returns an httptest server that records the query of the
// last request and replies with an empty list of T.
func captureQueryServer[T any](t *testing.T, captured *url.Values) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*captured = r.URL.Query()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(capi.ListResponse[T]{})
	}))
}

// TestTypedListOptions_ReachQuery proves typed List options are applied to the
// outgoing request query for both the standard impl path (builds) and the
// generic usage-events override path.
//
//nolint:funlen // Test functions can be longer for comprehensive testing
func TestTypedListOptions_ReachQuery(t *testing.T) {
	t.Parallel()

	t.Run("builds states and app_guids", func(t *testing.T) {
		t.Parallel()

		var got url.Values

		server := captureQueryServer[capi.Build](t, &got)
		defer server.Close()

		client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
		require.NoError(t, err)

		_, err = client.Builds().List(context.Background(), nil,
			capi.WithBuildStates(capi.BuildStateStaging, capi.BuildStateStaged),
			capi.WithBuildAppGUIDs(testAppGUID1),
		)
		require.NoError(t, err)
		assert.Equal(t, "STAGING,STAGED", got.Get(testStatesParam))
		assert.Equal(t, testAppGUID1, got.Get(testAppGUIDsParam))
	})

	t.Run("builds options merge with params", func(t *testing.T) {
		t.Parallel()

		var got url.Values

		server := captureQueryServer[capi.Build](t, &got)
		defer server.Close()

		client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
		require.NoError(t, err)

		params := capi.NewQueryParams().WithPerPage(10)

		_, err = client.Builds().List(context.Background(), params,
			capi.WithBuildStates(capi.BuildStateFailed),
		)
		require.NoError(t, err)
		assert.Equal(t, "10", got.Get("per_page"))
		assert.Equal(t, testStateFailed, got.Get(testStatesParam))
	})

	t.Run("app usage events after_guid override path", func(t *testing.T) {
		t.Parallel()

		var got url.Values

		server := captureQueryServer[capi.AppUsageEvent](t, &got)
		defer server.Close()

		client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
		require.NoError(t, err)

		_, err = client.AppUsageEvents().List(context.Background(), nil,
			capi.WithAppUsageEventAfterGUID("event-0"),
		)
		require.NoError(t, err)
		assert.Equal(t, "event-0", got.Get("after_guid"))
	})

	t.Run("service instances filter and include compose over wire", func(t *testing.T) {
		t.Parallel()

		var got url.Values

		server := captureQueryServer[capi.ServiceInstance](t, &got)
		defer server.Close()

		client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
		require.NoError(t, err)

		params := capi.NewQueryParams().WithPerPage(25)

		_, err = client.ServiceInstances().List(context.Background(), params,
			capi.WithServiceInstanceType(capi.ServiceInstanceFilterTypeManaged),
			capi.WithServiceInstanceSpaceGUIDs(testSpaceName1),
			capi.ServiceInstanceIncludeSpace,
		)
		require.NoError(t, err)
		assert.Equal(t, "25", got.Get("per_page"))
		assert.Equal(t, testManagedType, got.Get(testTypeKey))
		assert.Equal(t, testSpaceName1, got.Get(testSpaceGUIDsParam))
		assert.Equal(t, "space", got.Get("include"))
	})

	t.Run("service usage events typed instance types", func(t *testing.T) {
		t.Parallel()

		var got url.Values

		server := captureQueryServer[capi.ServiceUsageEvent](t, &got)
		defer server.Close()

		client, err := New(context.Background(), &capi.Config{APIEndpoint: server.URL})
		require.NoError(t, err)

		_, err = client.ServiceUsageEvents().List(context.Background(), nil,
			capi.WithServiceUsageEventServiceInstanceTypes(capi.ServiceInstanceTypeManaged),
		)
		require.NoError(t, err)
		assert.Equal(t, "managed_service_instance", got.Get("service_instance_types"))
	})
}
