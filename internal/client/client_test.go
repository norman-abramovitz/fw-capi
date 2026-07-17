package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/fivetwenty-io/capi/v3/internal/client"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testVersion100 is a shared semver fixture, referenced from droplets_test.go
// and packages_test.go as well as this file.
const testVersion100 = "1.0.0"

// testAPIExampleComURL is the fixture API endpoint used across TestNew subtests.
const testAPIExampleComURL = "https://api.example.com"

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestNew(t *testing.T) {
	t.Parallel()
	t.Run("requires API endpoint", func(t *testing.T) {
		t.Parallel()

		config := &capi.Config{}
		_, err := New(context.Background(), config)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API endpoint is required")
	})

	t.Run("creates client with access token", func(t *testing.T) {
		t.Parallel()

		config := &capi.Config{
			APIEndpoint: testAPIExampleComURL,
			AccessToken: "test-token",
		}

		client, err := New(context.Background(), config)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("creates client with client credentials", func(t *testing.T) {
		t.Parallel()

		config := &capi.Config{
			APIEndpoint:  testAPIExampleComURL,
			ClientID:     testClientIDFixture,
			ClientSecret: "client-secret",
		}

		client, err := New(context.Background(), config)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("creates client with username/password", func(t *testing.T) {
		t.Parallel()

		config := &capi.Config{
			APIEndpoint: testAPIExampleComURL,
			Username:    testUserUsername,
			Password:    testPassPassword,
		}

		client, err := New(context.Background(), config)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("creates client without authentication", func(t *testing.T) {
		t.Parallel()

		config := &capi.Config{
			APIEndpoint: testAPIExampleComURL,
		}

		client, err := New(context.Background(), config)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("FetchAPILinksOnInit respects cancelled ctx", func(t *testing.T) {
		t.Parallel()

		// Server that blocks until the test ends, ensuring a cancelled ctx
		// causes FetchAPILinks to abort rather than succeed.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Intentionally do not respond — the cancelled ctx must fire first.
			<-r.Context().Done()
			http.Error(w, "cancelled", http.StatusServiceUnavailable)
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately before New is called

		config := &capi.Config{
			APIEndpoint:         server.URL,
			FetchAPILinksOnInit: true,
		}

		// New must still return a usable client; FetchAPILinks error is non-fatal.
		client, err := New(ctx, config)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})
}

func TestClient_GetInfo(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/info", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		info := capi.Info{
			Build:       "1.2.3",
			Name:        "Test CF",
			Version:     3,
			Description: "Test Cloud Foundry",
			CFOnK8s:     false,
			CLIVersion: capi.CLIVersion{
				Minimum:     testVersion100,
				Recommended: "2.0.0",
			},
			RateLimits: capi.RateLimits{
				Enabled:                true,
				GeneralLimit:           2000,
				ResetIntervalInMinutes: 30,
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(info)
	}))
	defer server.Close()

	config := &capi.Config{
		APIEndpoint: server.URL,
	}

	client, err := New(context.Background(), config)
	require.NoError(t, err)

	info, err := client.GetInfo(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, "1.2.3", info.Build)
	assert.Equal(t, "Test CF", info.Name)
	assert.Equal(t, 3, info.Version)
	assert.True(t, info.RateLimits.Enabled)
	assert.Equal(t, 2000, info.RateLimits.GeneralLimit)
	assert.Equal(t, 30, info.RateLimits.ResetIntervalInMinutes)
}

func TestClient_GetRootInfo(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		rootInfo := capi.RootInfo{
			Links: capi.Links{
				testSelfKey: capi.Link{
					Href: request.Host,
				},
				"cloud_controller_v2": capi.Link{
					Href: request.Host + "/v2",
				},
				"cloud_controller_v3": capi.Link{
					Href: request.Host + "/v3",
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(rootInfo)
	}))
	defer server.Close()

	config := &capi.Config{
		APIEndpoint: server.URL,
	}

	client, err := New(context.Background(), config)
	require.NoError(t, err)

	rootInfo, err := client.GetRootInfo(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, rootInfo)
	assert.NotNil(t, rootInfo.Links)
}

func TestClient_GetUsageSummary(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/info/usage_summary", request.URL.Path)
		assert.Equal(t, http.MethodGet, request.Method)

		summary := capi.UsageSummary{
			UsageSummary: capi.UsageSummaryData{
				StartedInstances: 10,
				MemoryInMB:       2048,
			},
			Links: capi.Links{
				testSelfKey: capi.Link{
					Href: request.Host + "/v3/info/usage_summary",
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(summary)
	}))
	defer server.Close()

	config := &capi.Config{
		APIEndpoint: server.URL,
	}

	client, err := New(context.Background(), config)
	require.NoError(t, err)

	summary, err := client.GetUsageSummary(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, 10, summary.UsageSummary.StartedInstances)
	assert.Equal(t, 2048, summary.UsageSummary.MemoryInMB)
}

func TestClient_ClearBuildpackCache(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/admin/actions/clear_buildpack_cache", request.URL.Path)
		assert.Equal(t, http.MethodPost, request.Method)

		job := capi.Job{
			Resource: capi.Resource{
				GUID: testJobGUID,
			},
			Operation: "clear_buildpack_cache",
			State:     testStateProcessing,
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(writer).Encode(job)
	}))
	defer server.Close()

	config := &capi.Config{
		APIEndpoint: server.URL,
	}

	client, err := New(context.Background(), config)
	require.NoError(t, err)

	job, err := client.ClearBuildpackCache(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, testJobGUID, job.GUID)
	assert.Equal(t, "clear_buildpack_cache", job.Operation)
	assert.Equal(t, testStateProcessing, job.State)
}

func TestClient_ResourceAccessors(t *testing.T) {
	t.Parallel()

	config := &capi.Config{
		APIEndpoint: testAPIExampleComURL,
	}

	client, err := New(context.Background(), config)
	require.NoError(t, err)

	// Test that all accessors return their respective clients (or nil for now)
	assert.NotNil(t, client.Apps())                      // Apps client is implemented
	assert.NotNil(t, client.Organizations())             // Organizations client is implemented
	assert.NotNil(t, client.Spaces())                    // Spaces client is implemented
	assert.NotNil(t, client.Domains())                   // Domains client is implemented
	assert.NotNil(t, client.Routes())                    // Routes client is implemented
	assert.NotNil(t, client.ServiceBrokers())            // ServiceBrokers client is implemented
	assert.NotNil(t, client.ServiceOfferings())          // ServiceOfferings client is implemented
	assert.NotNil(t, client.ServicePlans())              // ServicePlans client is implemented
	assert.NotNil(t, client.ServiceInstances())          // ServiceInstances client is implemented
	assert.NotNil(t, client.ServiceCredentialBindings()) // ServiceCredentialBindings client is implemented
	assert.NotNil(t, client.ServiceRouteBindings())      // ServiceRouteBindings client is implemented
	assert.NotNil(t, client.Builds())                    // Builds client is implemented
	assert.NotNil(t, client.Buildpacks())                // Buildpacks client is implemented
	assert.NotNil(t, client.Deployments())               // Deployments client is implemented
	assert.NotNil(t, client.Droplets())                  // Droplets client is implemented
	assert.NotNil(t, client.Packages())                  // Packages client is implemented
	assert.NotNil(t, client.Processes())                 // Processes client is implemented
	assert.NotNil(t, client.Tasks())                     // Tasks client is implemented
	assert.NotNil(t, client.Stacks())                    // Stacks client is implemented
	assert.NotNil(t, client.Roles())                     // Roles client is implemented
	assert.NotNil(t, client.SecurityGroups())            // SecurityGroups client is implemented
	assert.NotNil(t, client.IsolationSegments())         // IsolationSegments client is implemented
	assert.NotNil(t, client.FeatureFlags())              // FeatureFlags client is implemented
	assert.NotNil(t, client.Jobs())                      // Jobs client is implemented
	assert.NotNil(t, client.Users())                     // Users client is implemented
}
