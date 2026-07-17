package capi_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCache_SetAndGet(t *testing.T) {
	t.Parallel()

	cache := capi.NewMemoryCache(10)
	ctx := context.Background()

	entry := &capi.CacheEntry{
		Data:      []byte("test data"),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		ETag:      testETag,
	}

	// Set entry
	err := cache.Set(ctx, "key1", entry)
	require.NoError(t, err)

	// Get entry
	retrieved, err := cache.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, entry.Data, retrieved.Data)
	assert.Equal(t, entry.ETag, retrieved.ETag)
}

func TestMemoryCache_GetNonExistent(t *testing.T) {
	t.Parallel()

	cache := capi.NewMemoryCache(10)
	ctx := context.Background()

	_, err := cache.Get(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")
}

func TestMemoryCache_GetExpired(t *testing.T) {
	t.Parallel()

	cache := capi.NewMemoryCache(10)
	ctx := context.Background()

	entry := &capi.CacheEntry{
		Data:      []byte("test data"),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
		ETag:      testETag,
	}

	err := cache.Set(ctx, "key1", entry)
	require.NoError(t, err)

	_, err = cache.Get(ctx, "key1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "entry expired")
}

func TestMemoryCache_Delete(t *testing.T) {
	t.Parallel()

	cache := capi.NewMemoryCache(10)
	ctx := context.Background()

	entry := &capi.CacheEntry{
		Data:      []byte("test data"),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Set and verify
	err := cache.Set(ctx, "key1", entry)
	require.NoError(t, err)
	assert.True(t, cache.Has(ctx, "key1"))

	// Delete
	err = cache.Delete(ctx, "key1")
	require.NoError(t, err)

	// Verify deleted
	assert.False(t, cache.Has(ctx, "key1"))
}

func TestMemoryCache_Clear(t *testing.T) {
	t.Parallel()

	cache := capi.NewMemoryCache(10)
	ctx := context.Background()

	// Add multiple entries
	for i := range 3 {
		entry := &capi.CacheEntry{
			Data:      []byte("test data"),
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		_ = cache.Set(ctx, string(rune('a'+i)), entry)
	}

	// Verify entries exist
	assert.True(t, cache.Has(ctx, "a"))
	assert.True(t, cache.Has(ctx, "b"))
	assert.True(t, cache.Has(ctx, "c"))

	// Clear cache
	err := cache.Clear(ctx)
	require.NoError(t, err)

	// Verify all cleared
	assert.False(t, cache.Has(ctx, "a"))
	assert.False(t, cache.Has(ctx, "b"))
	assert.False(t, cache.Has(ctx, "c"))
}

func TestMemoryCache_MaxSize(t *testing.T) {
	t.Parallel()

	cache := capi.NewMemoryCache(2)
	ctx := context.Background()

	// Add entries up to max size
	for i := range 3 {
		entry := &capi.CacheEntry{
			Data:      []byte("test data"),
			ExpiresAt: time.Now().Add(time.Duration(i+1) * time.Hour),
		}
		_ = cache.Set(ctx, string(rune('a'+i)), entry)
	}

	// The cache should have evicted the soonest-to-expire entry.
	// Since we can't easily check internal state, we verify behavior.
	has := 0

	for i := range 3 {
		if cache.Has(ctx, string(rune('a'+i))) {
			has++
		}
	}

	assert.LessOrEqual(t, has, 2)
}

func TestMemoryCache_Cleanup(t *testing.T) {
	t.Parallel()

	cache := capi.NewMemoryCache(10)
	ctx := context.Background()

	// Add expired and non-expired entries
	expiredEntry := &capi.CacheEntry{
		Data:      []byte("expired"),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	validEntry := &capi.CacheEntry{
		Data:      []byte("valid"),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	_ = cache.Set(ctx, "expired", expiredEntry)
	_ = cache.Set(ctx, "valid", validEntry)

	// Run cleanup
	cache.Cleanup()

	// Valid entry should still exist
	assert.True(t, cache.Has(ctx, "valid"))
	// Expired entry should be gone
	assert.False(t, cache.Has(ctx, "expired"))
}

// TestMemoryCache_CtxCancel verifies all MemoryCache methods respect context cancellation.
func TestMemoryCache_CtxCancel(t *testing.T) {
	t.Parallel()

	cache := capi.NewMemoryCache(10)

	entry := &capi.CacheEntry{
		Data:      []byte("data"),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Pre-populate with live ctx so Get/Delete have something to work with.
	bg := context.Background()
	require.NoError(t, cache.Set(bg, "k", entry))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := cache.Get(ctx, "k")
	require.ErrorIs(t, err, context.Canceled, "Get must return ctx.Err() on cancelled ctx")

	err = cache.Set(ctx, "k2", entry)
	require.ErrorIs(t, err, context.Canceled, "Set must return ctx.Err() on cancelled ctx")

	err = cache.Delete(ctx, "k")
	require.ErrorIs(t, err, context.Canceled, "Delete must return ctx.Err() on cancelled ctx")

	err = cache.Clear(ctx)
	require.ErrorIs(t, err, context.Canceled, "Clear must return ctx.Err() on cancelled ctx")

	assert.False(t, cache.Has(ctx, "k"), "Has must return false on cancelled ctx")
}

func TestCacheManager_GetCacheKey(t *testing.T) {
	t.Parallel()

	manager := capi.NewCacheManager(nil, nil)

	// Test with no params
	key1 := manager.GetCacheKey("GET", "/v3/apps", nil)
	assert.Equal(t, "GET:/v3/apps", key1)

	// Test with params
	params := map[string]string{testQueryParamPage: "1", testQueryParamPerPage: "50"}
	key2 := manager.GetCacheKey("GET", "/v3/apps", params)
	assert.Contains(t, key2, "GET:/v3/apps:")
	assert.Contains(t, key2, testQueryParamPage)
	assert.Contains(t, key2, testQueryParamPerPage)
}

func TestCacheManager_SetAndGet(t *testing.T) {
	t.Parallel()

	cache := capi.NewMemoryCache(10)
	manager := capi.NewCacheManager(cache, nil)
	ctx := context.Background()

	data := []byte("test data")
	key := "test-key"

	// Set data
	err := manager.Set(ctx, key, data, 1*time.Hour)
	require.NoError(t, err)

	// Get data
	retrieved, err := manager.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, data, retrieved)

	// Check stats via accessor methods
	stats := manager.GetStats()
	assert.Equal(t, int64(1), stats.Hits())
	assert.Equal(t, int64(0), stats.Misses())
	assert.Equal(t, int64(1), stats.Sets())
}

func TestCacheManager_SetWithETag(t *testing.T) {
	t.Parallel()

	cache := capi.NewMemoryCache(10)
	manager := capi.NewCacheManager(cache, nil)
	ctx := context.Background()

	data := []byte("test data")
	key := "test-key"
	etag := "abc123"

	// Set data with ETag
	err := manager.SetWithETag(ctx, key, data, etag, 1*time.Hour)
	require.NoError(t, err)

	// Get data
	retrieved, err := manager.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, data, retrieved)
}

func TestCacheManager_Miss(t *testing.T) {
	t.Parallel()

	cache := capi.NewMemoryCache(10)
	manager := capi.NewCacheManager(cache, nil)
	ctx := context.Background()

	// Try to get non-existent key
	_, err := manager.Get(ctx, "nonexistent")
	require.Error(t, err)

	// Check stats via accessor methods
	stats := manager.GetStats()
	assert.Equal(t, int64(0), stats.Hits())
	assert.Equal(t, int64(1), stats.Misses())
}

// TestCacheStats_GetHitRate verifies hit rate calculation via accessor methods.
func TestCacheStats_GetHitRate(t *testing.T) {
	t.Parallel()

	cache := capi.NewMemoryCache(100)
	manager := capi.NewCacheManager(cache, nil)
	ctx := context.Background()

	// Seed 75 hits via Set + Get, then 25 misses.
	data := []byte("v")

	for i := range 75 {
		key := fmt.Sprintf("k%d", i)
		_ = manager.Set(ctx, key, data, time.Hour)
		_, _ = manager.Get(ctx, key) // hit
	}

	for i := 75; i < 100; i++ {
		_, _ = manager.Get(ctx, fmt.Sprintf("missing%d", i)) // miss
	}

	stats := manager.GetStats()
	assert.InDelta(t, 0.75, stats.GetHitRate(), 0.0001)

	// Zero-requests edge case
	emptyManager := capi.NewCacheManager(capi.NewMemoryCache(10), nil)
	assert.InDelta(t, 0.0, emptyManager.GetStats().GetHitRate(), 0.0001)
}

// TestCacheStats_ConcurrentIncrements verifies no data race on stat counters
// when multiple goroutines increment simultaneously.
func TestCacheStats_ConcurrentIncrements(t *testing.T) {
	t.Parallel()

	cache := capi.NewMemoryCache(1000)
	manager := capi.NewCacheManager(cache, nil)
	ctx := context.Background()

	const (
		goroutines = 50
		opsEach    = 20
	)

	var waitGroup sync.WaitGroup

	for goroutineIdx := range goroutines {
		waitGroup.Add(1)

		go func(id int) {
			defer waitGroup.Done()

			for i := range opsEach {
				key := fmt.Sprintf("g%d-k%d", id, i)
				_ = manager.Set(ctx, key, []byte("v"), time.Hour)
				_, _ = manager.Get(ctx, key)
				_, _ = manager.Get(ctx, key+"miss")
			}
		}(goroutineIdx)
	}

	waitGroup.Wait()

	stats := manager.GetStats()
	// Each goroutine does opsEach Sets, opsEach hits, opsEach misses.
	assert.Equal(t, int64(goroutines*opsEach), stats.Sets())
	assert.Equal(t, int64(goroutines*opsEach), stats.Hits())
	assert.Equal(t, int64(goroutines*opsEach), stats.Misses())
}

// TestCacheManager_InvalidateAll verifies InvalidateAll clears all cache entries.
func TestCacheManager_InvalidateAll(t *testing.T) {
	t.Parallel()

	memCache := capi.NewMemoryCache(10)
	manager := capi.NewCacheManager(memCache, nil)
	ctx := context.Background()

	data := []byte("value")
	keys := []string{"a", "b", "c"}

	for _, k := range keys {
		require.NoError(t, manager.Set(ctx, k, data, time.Hour))
	}

	// Verify entries exist
	for _, k := range keys {
		_, err := manager.Get(ctx, k)
		require.NoError(t, err, "key %q should exist before InvalidateAll", k)
	}

	require.NoError(t, manager.InvalidateAll(ctx))

	// All entries must be absent after invalidation
	for _, k := range keys {
		_, err := manager.Get(ctx, k)
		require.Error(t, err, "key %q should be absent after InvalidateAll", k)
	}
}

func TestCachingPolicy_ShouldCache(t *testing.T) {
	t.Parallel()

	policy := capi.DefaultCachingPolicy()

	// Test GET requests (should cache)
	assert.True(t, policy.ShouldCache("GET", "/v3/apps", 200))

	// Test POST requests (should not cache by default)
	assert.False(t, policy.ShouldCache("POST", "/v3/apps", 201))

	// Test error responses (should not cache by default)
	assert.False(t, policy.ShouldCache("GET", "/v3/apps", 404))

	// Test excluded paths
	assert.False(t, policy.ShouldCache("GET", "/v3/jobs", 200))

	// Test with custom policy
	customPolicy := &capi.CachingPolicy{
		CacheGET:     true,
		CachePOST:    true,
		CacheErrors:  true,
		IncludePaths: []string{testAppsPath},
	}

	// Only included paths should be cached
	assert.True(t, customPolicy.ShouldCache("GET", "/v3/apps", 200))
	assert.False(t, customPolicy.ShouldCache("GET", "/v3/spaces", 200))

	// POST should be cached with custom policy
	assert.True(t, customPolicy.ShouldCache("POST", "/v3/apps", 201))

	// Errors should be cached with custom policy
	assert.True(t, customPolicy.ShouldCache("GET", "/v3/apps", 404))
}
