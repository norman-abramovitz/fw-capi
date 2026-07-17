package client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fivetwenty-io/capi/v3/internal/client"
	internalhttp "github.com/fivetwenty-io/capi/v3/internal/http"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
)

// Real CF answers async endpoints with 202 Accepted, an EMPTY body, and the
// job reference in the Location header (per the CF v3 OpenAPI spec). Every
// operation in this table previously json.Unmarshal'ed the empty body and
// failed with "parsing job response: unexpected end of JSON input"
// (cloudfoundry/stratos#5431).
//
//nolint:funlen // one table covering every async operation; splitting it would hide the shared 202/Location contract
func TestAsyncJobOperations_Empty202BodyWithLocationHeader(t *testing.T) {
	t.Parallel()

	const jobGUID = "job-from-location"

	newServer := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Location", "/v3/jobs/"+jobGUID)
			writer.WriteHeader(http.StatusAccepted)
			// Empty body — what real CF sends on async responses.
		}))
	}

	managedSICreate := &capi.ServiceInstanceCreateRequest{
		Type: testManagedType,
		Name: testMyInstanceName,
		Relationships: capi.ServiceInstanceRelationships{
			Space:       capi.Relationship{Data: &capi.RelationshipData{GUID: testSpaceGUID}},
			ServicePlan: &capi.Relationship{Data: &capi.RelationshipData{GUID: testPlanGUID}},
		},
	}

	cases := []struct {
		name string
		call func(t *testing.T, baseURL string) (any, error)
	}{
		{
			name: "service instance create (managed)",
			call: func(_ *testing.T, baseURL string) (any, error) {
				c := client.NewServiceInstancesClient(internalhttp.NewClient(baseURL, nil))

				return c.Create(context.Background(), managedSICreate)
			},
		},
		{
			name: "service instance update (managed)",
			call: func(_ *testing.T, baseURL string) (any, error) {
				c := client.NewServiceInstancesClient(internalhttp.NewClient(baseURL, nil))

				return c.Update(context.Background(), "si-guid", &capi.ServiceInstanceUpdateRequest{
					Parameters: map[string]any{testFooKey: testBarValue},
				})
			},
		},
		{
			name: "service broker create",
			call: func(_ *testing.T, baseURL string) (any, error) {
				c := client.NewServiceBrokersClient(internalhttp.NewClient(baseURL, nil))

				return c.Create(context.Background(), &capi.ServiceBrokerCreateRequest{Name: "broker"})
			},
		},
		{
			name: "service broker update (catalog sync)",
			call: func(_ *testing.T, baseURL string) (any, error) {
				c := client.NewServiceBrokersClient(internalhttp.NewClient(baseURL, nil))
				name := "broker"

				return c.Update(context.Background(), testBrokerGUID, &capi.ServiceBrokerUpdateRequest{Name: &name})
			},
		},
		{
			name: "service credential binding create",
			call: func(_ *testing.T, baseURL string) (any, error) {
				c := client.NewServiceCredentialBindingsClient(internalhttp.NewClient(baseURL, nil))

				return c.Create(context.Background(), &capi.ServiceCredentialBindingCreateRequest{Type: testAppKey})
			},
		},
		{
			name: "service route binding create",
			call: func(_ *testing.T, baseURL string) (any, error) {
				c := client.NewServiceRouteBindingsClient(internalhttp.NewClient(baseURL, nil))

				return c.Create(context.Background(), &capi.ServiceRouteBindingCreateRequest{})
			},
		},
		{
			name: "space apply manifest",
			call: func(_ *testing.T, baseURL string) (any, error) {
				c := client.NewSpacesClient(internalhttp.NewClient(baseURL, nil))

				return c.ApplyManifest(context.Background(), testSpaceGUID, "applications: []")
			},
		},
		{
			name: "space delete unmapped routes",
			call: func(_ *testing.T, baseURL string) (any, error) {
				c := client.NewSpacesClient(internalhttp.NewClient(baseURL, nil))

				return c.DeleteUnmappedRoutes(context.Background(), testSpaceGUID)
			},
		},
		{
			name: "clear buildpack cache",
			call: func(t *testing.T, baseURL string) (any, error) { //nolint:thelper // table-driven operation invoker, not a shared assertion helper
				c, err := client.New(context.Background(), &capi.Config{APIEndpoint: baseURL})
				require.NoError(t, err)

				return c.ClearBuildpackCache(context.Background())
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := newServer()
			defer server.Close()

			out, err := testCase.call(t, server.URL)
			require.NoError(t, err)

			job, ok := out.(*capi.Job)
			require.True(t, ok, "expected *capi.Job, got %T", out)
			require.NotNil(t, job)
			assert.Equal(t, jobGUID, job.GUID)
		})
	}
}

// A 202 body carrying the Job resource (older CF / proxies / emulators) must
// keep working and win over the Location header — the body has full job
// state, the header only the GUID.
func TestAsyncJobOperations_202BodyStillPreferred(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Location", "/v3/jobs/job-from-location")
		writer.WriteHeader(http.StatusAccepted)
		_, _ = writer.Write([]byte(`{"guid":"job-from-body","operation":"service_instance.update","state":"PROCESSING"}`))
	}))
	defer server.Close()

	c := client.NewServiceInstancesClient(internalhttp.NewClient(server.URL, nil))
	out, err := c.Update(context.Background(), "si-guid", &capi.ServiceInstanceUpdateRequest{
		Parameters: map[string]any{testFooKey: testBarValue},
	})
	require.NoError(t, err)

	job, ok := out.(*capi.Job)
	require.True(t, ok, "expected *capi.Job, got %T", out)
	assert.Equal(t, "job-from-body", job.GUID)
	assert.Equal(t, testStateProcessing, job.State)
}

// Neither a parseable body nor a Location header — surface a parse error so
// callers see the contract violation rather than a zero-value Job.
func TestAsyncJobOperations_Empty202NoLocationErrors(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	c := client.NewServiceInstancesClient(internalhttp.NewClient(server.URL, nil))
	_, err := c.Update(context.Background(), "si-guid", &capi.ServiceInstanceUpdateRequest{
		Parameters: map[string]any{testFooKey: testBarValue},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing job response")
}
