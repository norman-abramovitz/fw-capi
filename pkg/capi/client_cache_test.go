package capi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheInterceptor(t *testing.T) {
	t.Parallel()
	// Create cache manager
	cache := capi.NewMemoryCache(100)
	manager := capi.NewCacheManager(cache, nil)
	policy := capi.DefaultCachingPolicy()

	// Create interceptors
	reqInterceptor, respInterceptor := capi.CacheInterceptor(manager, policy)

	ctx := context.Background()

	// Test GET request caching
	req := &capi.Request{
		Method: http.MethodGet,
		Path:   testAppsPath,
	}

	// First request - should not be cached
	err := reqInterceptor(ctx, req)
	require.NoError(t, err)

	// Simulate response
	resp := &capi.Response{
		StatusCode: 200,
		Headers:    make(http.Header),
		Body:       []byte(`{"resources": []}`),
	}

	// Response interceptor should cache it
	err = respInterceptor(ctx, req, resp)
	require.NoError(t, err)

	// Second request - should be cached
	req2 := &capi.Request{
		Method: http.MethodGet,
		Path:   testAppsPath,
	}

	err = reqInterceptor(ctx, req2)
	require.NoError(t, err)
	// Note: In a real implementation, the cached response would be in context

	// Test POST request - should not be cached
	postReq := &capi.Request{
		Method: http.MethodPost,
		Path:   testAppsPath,
	}

	err = reqInterceptor(ctx, postReq)
	require.NoError(t, err)
}

func TestConditionalRequestInterceptor(t *testing.T) {
	t.Parallel()
	// Create cache manager with an entry that has an ETag
	cache := capi.NewMemoryCache(100)
	manager := capi.NewCacheManager(cache, nil)

	ctx := context.Background()

	// Store an entry with ETag
	cacheKey := manager.GetCacheKey(http.MethodGet, "/v3/apps/123", nil)
	err := manager.SetWithETag(ctx, cacheKey, []byte("data"), testETag, 1*time.Hour)
	require.NoError(t, err)

	// Create interceptor
	interceptor := capi.ConditionalRequestInterceptor(manager)

	// Test GET request
	req := &capi.Request{
		Method:  http.MethodGet,
		Path:    "/v3/apps/123",
		Headers: make(http.Header),
	}

	err = interceptor(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, testETag, req.Headers.Get("If-None-Match"))

	// Test non-GET request
	postReq := &capi.Request{
		Method:  http.MethodPost,
		Path:    testAppsPath,
		Headers: make(http.Header),
	}

	err = interceptor(ctx, postReq)
	require.NoError(t, err)
	assert.Empty(t, postReq.Headers.Get("If-None-Match"))
}

func TestCacheInvalidationInterceptor(t *testing.T) {
	t.Parallel()
	// Create cache manager
	cache := capi.NewMemoryCache(100)
	manager := capi.NewCacheManager(cache, nil)

	ctx := context.Background()

	// Store some cached GET responses
	cacheKey1 := manager.GetCacheKey("GET", "/v3/apps/123", nil)
	err := manager.Set(ctx, cacheKey1, []byte("app data"), 1*time.Hour)
	require.NoError(t, err)

	cacheKey2 := manager.GetCacheKey("GET", "/v3/apps", nil)
	err = manager.Set(ctx, cacheKey2, []byte("apps list"), 1*time.Hour)
	require.NoError(t, err)

	// Create interceptor
	interceptor := capi.CacheInvalidationInterceptor(manager)

	// Test successful mutation
	req := &capi.Request{
		Method: "PUT",
		Path:   "/v3/apps/123",
	}
	resp := &capi.Response{
		StatusCode: 200,
	}

	err = interceptor(ctx, req, resp)
	require.NoError(t, err)
	// In a real implementation, this would invalidate related cache entries

	// Test failed mutation (should not invalidate)
	req2 := &capi.Request{
		Method: "DELETE",
		Path:   "/v3/apps/456",
	}
	resp2 := &capi.Response{
		StatusCode: 404,
	}

	err = interceptor(ctx, req2, resp2)
	require.NoError(t, err)
	// Cache should still have the entries
}

func TestSmartCacheConfig(t *testing.T) {
	t.Parallel()

	config := capi.DefaultSmartCacheConfig()
	assert.True(t, config.EnableSmartInvalidation)
	assert.True(t, config.EnableConditionalRequests)
	assert.True(t, config.EnableMetrics)
	assert.NotEmpty(t, config.ResourceTTLs)
	assert.Equal(t, 10*time.Minute, config.ResourceTTLs["/v3/organizations"])
}

func TestConfigureSmartCache(t *testing.T) {
	t.Parallel()
	// Create components
	chain := capi.NewInterceptorChain()
	cache := capi.NewMemoryCache(100)
	manager := capi.NewCacheManager(cache, nil)
	config := capi.DefaultSmartCacheConfig()

	// Configure smart cache
	capi.ConfigureSmartCache(chain, manager, config)

	// Verify interceptors were added
	ctx := context.Background()
	req := &capi.Request{
		Method: http.MethodGet,
		Path:   testAppsPath,
	}

	// This should not error if interceptors were added correctly
	err := chain.ExecuteRequestInterceptors(ctx, req)
	require.NoError(t, err)
}

func TestCacheWarmer(t *testing.T) {
	t.Parallel()
	// This test is simplified - in production you'd use a proper mock client
	// For now, we'll just test the warmer creation

	// Create cache manager
	cache := capi.NewMemoryCache(100)
	manager := capi.NewCacheManager(cache, nil)

	// Create warmer with nil client (simplified test)
	warmer := capi.NewCacheWarmer(nil, manager)
	assert.NotNil(t, warmer)

	// In a real test, you'd mock the client and verify cache warming
}

func TestCachingPolicy_ShouldCacheExtended(t *testing.T) {
	t.Parallel()

	policy := capi.DefaultCachingPolicy()

	// Test GET request
	assert.True(t, policy.ShouldCache("GET", "/v3/apps", 200))
	assert.True(t, policy.ShouldCache("GET", "/v3/spaces", 200))

	// Test POST request (should not cache by default)
	assert.False(t, policy.ShouldCache("POST", "/v3/apps", 201))

	// Test error response (should not cache by default)
	assert.False(t, policy.ShouldCache("GET", "/v3/apps", 500))

	// Test excluded paths
	assert.False(t, policy.ShouldCache("GET", "/v3/jobs", 200))
	assert.False(t, policy.ShouldCache("GET", "/v3/deployments", 200))

	// Test with included paths
	policy.IncludePaths = []string{"/v3/organizations"}
	assert.True(t, policy.ShouldCache("GET", "/v3/organizations", 200))
	assert.False(t, policy.ShouldCache("GET", "/v3/apps", 200))
}
