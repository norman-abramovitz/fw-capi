package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/cloudfoundry-community/go-uaa"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
)

// CacheEntry represents a cached item with expiry.
type CacheEntry struct {
	Data      any
	ExpiresAt time.Time
}

// IsExpired checks if the cache entry has expired.
func (c *CacheEntry) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// UAACache provides caching functionality for UAA operations.
type UAACache struct {
	cache map[string]*CacheEntry
	mutex sync.RWMutex
	ttl   time.Duration
}

// NewUAACache creates a new cache with specified TTL.
func NewUAACache(ttl time.Duration) *UAACache {
	cache := &UAACache{
		cache: make(map[string]*CacheEntry),
		mutex: sync.RWMutex{},
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves an item from cache.
func (c *UAACache) Get(key string) (any, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.cache[key]
	if !exists || entry.IsExpired() {
		return nil, false
	}

	return entry.Data, true
}

// Set stores an item in cache.
func (c *UAACache) Set(key string, data any) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache[key] = &CacheEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes an item from cache.
func (c *UAACache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.cache, key)
}

// Clear removes all items from cache.
func (c *UAACache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache = make(map[string]*CacheEntry)
}

// cleanup periodically removes expired entries.
func (c *UAACache) cleanup() {
	ticker := time.NewTicker(constants.UAACacheTTL)
	defer ticker.Stop()

	for range ticker.C {
		c.mutex.Lock()

		for key, entry := range c.cache {
			if entry.IsExpired() {
				delete(c.cache, key)
			}
		}

		c.mutex.Unlock()
	}
}

// UAAPerformanceService encapsulates UAA performance functionality.
type UAAPerformanceService struct {
	cache   *UAACache
	metrics *PerformanceMetrics
}

// NewUAAPerformanceService creates a new UAA performance service.
func NewUAAPerformanceService() *UAAPerformanceService {
	return &UAAPerformanceService{
		cache: NewUAACache(constants.UAACacheTimeout),
		metrics: &PerformanceMetrics{
			operations: make(map[string][]time.Duration),
		},
	}
}

// GetDefaultPerformanceService returns a singleton instance for backward compatibility.
func GetDefaultPerformanceService() *UAAPerformanceService {
	return defaultPerformanceServiceSingleton.get()
}

type performanceServiceSingleton struct {
	once     sync.Once
	instance *UAAPerformanceService
}

func (s *performanceServiceSingleton) get() *UAAPerformanceService {
	s.once.Do(func() {
		s.instance = NewUAAPerformanceService()
	})

	return s.instance
}

// Package-level singleton instance
//
//nolint:gochecknoglobals // This needs to be a package-level singleton for proper functionality
var defaultPerformanceServiceSingleton = &performanceServiceSingleton{}

// Batch operation resource type identifiers accepted by BatchOperation.Resource.
const (
	batchResourceUser   = "user"
	batchResourceGroup  = "group"
	batchResourceClient = "client"
)

// BatchOperation represents a batch operation request.
type BatchOperation struct {
	Type     string // Create, Update, Delete
	Resource string // batchResourceUser, batchResourceGroup, batchResourceClient
	Data     any    // Operation data
	ID       string // Optional ID for updates/deletes
}

// BatchResult represents the result of a batch operation.
type BatchResult struct {
	Operation BatchOperation
	Result    any
	Error     error
}

// BatchProcessor handles batch operations efficiently.
type BatchProcessor struct {
	client     *UAAClientWrapper
	maxWorkers int
	batchSize  int
}

// NewBatchProcessor creates a new batch processor.
func NewBatchProcessor(client *UAAClientWrapper) *BatchProcessor {
	return &BatchProcessor{
		client:     client,
		maxWorkers: constants.MaxWorkers,
		batchSize:  constants.StandardPageSize,
	}
}

// ProcessBatch executes multiple operations in parallel.
func (bp *BatchProcessor) ProcessBatch(operations []BatchOperation) []BatchResult {
	results := make([]BatchResult, len(operations))
	jobs := make(chan int, len(operations))

	// Start worker goroutines
	var waitGroup sync.WaitGroup
	for i := 0; i < bp.maxWorkers && i < len(operations); i++ {
		waitGroup.Add(1)

		go bp.worker(jobs, operations, results, &waitGroup)
	}

	// Send jobs
	for i := range operations {
		jobs <- i
	}

	close(jobs)

	// Wait for completion
	waitGroup.Wait()

	return results
}

// worker processes batch operations.
func (bp *BatchProcessor) worker(jobs <-chan int, operations []BatchOperation, results []BatchResult, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	for i := range jobs {
		op := operations[i]
		result, err := bp.executeOperation(op)
		results[i] = BatchResult{
			Operation: op,
			Result:    result,
			Error:     err,
		}
	}
}

// executeOperation executes a single batch operation.
func (bp *BatchProcessor) executeOperation(operation BatchOperation) (any, error) {
	switch operation.Resource {
	case batchResourceUser:
		return bp.executeUserOperation(operation)
	case batchResourceGroup:
		return bp.executeGroupOperation(operation)
	case batchResourceClient:
		return bp.executeClientOperation(operation)
	default:
		return nil, fmt.Errorf("%w: %s", constants.ErrUnsupportedResource, operation.Resource)
	}
}

// executeUserOperation executes user-related batch operations.
func (bp *BatchProcessor) executeUserOperation(operation BatchOperation) (any, error) {
	factory := &BatchOperationFactory{client: bp.client}
	config := factory.CreateUserOperationConfig()

	return bp.executeGenericOperation(operation, config)
}

// executeGroupOperation executes group-related batch operations.
func (bp *BatchProcessor) executeGroupOperation(operation BatchOperation) (any, error) {
	factory := &BatchOperationFactory{client: bp.client}
	config := factory.CreateGroupOperationConfig()

	return bp.executeGenericOperation(operation, config)
}

// executeClientOperation executes client-related batch operations.
func (bp *BatchProcessor) executeClientOperation(operation BatchOperation) (any, error) {
	factory := &BatchOperationFactory{client: bp.client}
	config := factory.CreateClientOperationConfig()

	return bp.executeGenericOperation(operation, config)
}

// BatchOperationConfig defines the configuration for a batch operation type.
// BatchOperationFactory creates batch operation configurations for different entity types.
type BatchOperationFactory struct {
	client *UAAClientWrapper
}

// CreateUserOperationConfig creates a configuration for user operations.
func (f *BatchOperationFactory) CreateUserOperationConfig() BatchOperationConfig {
	return BatchOperationConfig{
		EntityType:     batchResourceUser,
		InvalidDataErr: constants.ErrInvalidUserData,
		IDRequiredErr:  constants.ErrUserIDRequired,
		CreateFunc: func(data any) (any, error) {
			if user, ok := data.(uaa.User); ok {
				return f.client.Client().CreateUser(user)
			}

			return nil, constants.ErrInvalidUserData
		},
		UpdateFunc: func(data any) (any, error) {
			if user, ok := data.(uaa.User); ok {
				return f.client.Client().UpdateUser(user)
			}

			return nil, constants.ErrInvalidUserData
		},
		DeleteFunc: func(id string) (any, error) {
			return f.client.Client().DeleteUser(id)
		},
	}
}

// CreateGroupOperationConfig creates a configuration for group operations.
func (f *BatchOperationFactory) CreateGroupOperationConfig() BatchOperationConfig {
	return BatchOperationConfig{
		EntityType:     batchResourceGroup,
		InvalidDataErr: constants.ErrInvalidGroupData,
		IDRequiredErr:  constants.ErrGroupIDRequired,
		CreateFunc: func(data any) (any, error) {
			if group, ok := data.(uaa.Group); ok {
				return f.client.Client().CreateGroup(group)
			}

			return nil, constants.ErrInvalidGroupData
		},
		UpdateFunc: func(data any) (any, error) {
			if group, ok := data.(uaa.Group); ok {
				return f.client.Client().UpdateGroup(group)
			}

			return nil, constants.ErrInvalidGroupData
		},
		DeleteFunc: func(id string) (any, error) {
			return f.client.Client().DeleteGroup(id)
		},
	}
}

// CreateClientOperationConfig creates a configuration for client operations.
func (f *BatchOperationFactory) CreateClientOperationConfig() BatchOperationConfig {
	return BatchOperationConfig{
		EntityType:     batchResourceClient,
		InvalidDataErr: constants.ErrInvalidClientData,
		IDRequiredErr:  constants.ErrClientIDRequired,
		CreateFunc: func(data any) (any, error) {
			if client, ok := data.(uaa.Client); ok {
				return f.client.Client().CreateClient(client)
			}

			return nil, constants.ErrInvalidClientData
		},
		UpdateFunc: func(data any) (any, error) {
			if client, ok := data.(uaa.Client); ok {
				return f.client.Client().UpdateClient(client)
			}

			return nil, constants.ErrInvalidClientData
		},
		DeleteFunc: func(id string) (any, error) {
			return f.client.Client().DeleteClient(id)
		},
	}
}

type BatchOperationConfig struct {
	EntityType     string
	InvalidDataErr error
	IDRequiredErr  error
	CreateFunc     func(data any) (any, error)
	UpdateFunc     func(data any) (any, error)
	DeleteFunc     func(id string) (any, error)
}

// executeGenericOperation executes a generic batch operation using the provided configuration.
func (bp *BatchProcessor) executeGenericOperation(operation BatchOperation, config BatchOperationConfig) (any, error) {
	switch operation.Type {
	case Create:
		if operation.Data != nil {
			result, err := config.CreateFunc(operation.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s in batch operation: %w", config.EntityType, err)
			}

			return result, nil
		}

		return nil, config.InvalidDataErr

	case Update:
		if operation.Data != nil {
			result, err := config.UpdateFunc(operation.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to update %s in batch operation: %w", config.EntityType, err)
			}

			return result, nil
		}

		return nil, config.InvalidDataErr

	case Delete:
		if operation.ID != "" {
			result, err := config.DeleteFunc(operation.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to delete %s in batch operation: %w", config.EntityType, err)
			}

			return result, nil
		}

		return nil, config.IDRequiredErr

	default:
		return nil, fmt.Errorf("%w: %s", constants.ErrUnsupportedOperation, operation.Type)
	}
}

// CachedUserLookup performs cached user lookups.
func CachedUserLookup(client *UAAClientWrapper, username string) (*uaa.User, error) {
	return GetDefaultPerformanceService().CachedUserLookup(client, username)
}

// CachedUserLookup performs cached user lookups.
func (s *UAAPerformanceService) CachedUserLookup(client *UAAClientWrapper, username string) (*uaa.User, error) {
	cacheKey := "user:" + username

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if user, ok := cached.(*uaa.User); ok {
			return user, nil
		}
	}

	// Fetch from API
	user, err := client.Client().GetUserByUsername(username, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	// Cache the result
	s.cache.Set(cacheKey, user)

	return user, nil
}

// CachedGroupLookup performs cached group lookups.
func CachedGroupLookup(client *UAAClientWrapper, groupName string) (*uaa.Group, error) {
	return GetDefaultPerformanceService().CachedGroupLookup(client, groupName)
}

// CachedGroupLookup performs cached group lookups.
func (s *UAAPerformanceService) CachedGroupLookup(client *UAAClientWrapper, groupName string) (*uaa.Group, error) {
	cacheKey := "group:" + groupName

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if group, ok := cached.(*uaa.Group); ok {
			return group, nil
		}
	}

	// Fetch from API using list groups with filter
	groups, _, err := client.Client().ListGroups(fmt.Sprintf("displayName eq \"%s\"", groupName), "", "", uaa.SortAscending, 1, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("%w: %s", constants.ErrGroupNotFound, groupName)
	}

	group := &groups[0]

	// Cache the result
	s.cache.Set(cacheKey, group)

	return group, nil
}

// CachedClientLookup performs cached client lookups.
func CachedClientLookup(client *UAAClientWrapper, clientID string) (*uaa.Client, error) {
	return GetDefaultPerformanceService().CachedClientLookup(client, clientID)
}

// CachedClientLookup performs cached client lookups.
func (s *UAAPerformanceService) CachedClientLookup(client *UAAClientWrapper, clientID string) (*uaa.Client, error) {
	cacheKey := "client:" + clientID

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if clientObj, ok := cached.(*uaa.Client); ok {
			return clientObj, nil
		}
	}

	// Fetch from API
	clientObj, err := client.Client().GetClient(clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	// Cache the result
	s.cache.Set(cacheKey, clientObj)

	return clientObj, nil
}

// CachedServerInfo performs cached server info lookup.
func CachedServerInfo(client *UAAClientWrapper) (map[string]any, error) {
	return GetDefaultPerformanceService().CachedServerInfo(client)
}

// CachedServerInfo performs cached server info lookup.
func (s *UAAPerformanceService) CachedServerInfo(client *UAAClientWrapper) (map[string]any, error) {
	cacheKey := "server_info"

	// Check cache first
	if cached, found := s.cache.Get(cacheKey); found {
		if serverInfo, ok := cached.(map[string]any); ok {
			return serverInfo, nil
		}
	}

	// Fetch from API
	ctx := context.Background()

	serverInfo, err := client.GetServerInfo(ctx)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.Set(cacheKey, serverInfo)

	return serverInfo, nil
}

// InvalidateUserCache removes user-related cache entries.
func InvalidateUserCache(username string) {
	GetDefaultPerformanceService().InvalidateUserCache(username)
}

// InvalidateUserCache removes user-related cache entries.
func (s *UAAPerformanceService) InvalidateUserCache(username string) {
	s.cache.Delete("user:" + username)
}

// InvalidateGroupCache removes group-related cache entries.
func InvalidateGroupCache(groupName string) {
	GetDefaultPerformanceService().InvalidateGroupCache(groupName)
}

// InvalidateGroupCache removes group-related cache entries.
func (s *UAAPerformanceService) InvalidateGroupCache(groupName string) {
	s.cache.Delete("group:" + groupName)
}

// InvalidateClientCache removes client-related cache entries.
func InvalidateClientCache(clientID string) {
	GetDefaultPerformanceService().InvalidateClientCache(clientID)
}

// InvalidateClientCache removes client-related cache entries.
func (s *UAAPerformanceService) InvalidateClientCache(clientID string) {
	s.cache.Delete("client:" + clientID)
}

// OptimizedPagination handles efficient pagination for large datasets.
type OptimizedPagination struct {
	client     *UAAClientWrapper
	pageSize   int
	maxPages   int
	cache      bool
	cacheStore *UAACache
}

// NewOptimizedPagination creates a new optimized pagination handler.
func NewOptimizedPagination(client *UAAClientWrapper) *OptimizedPagination {
	return &OptimizedPagination{
		client:     client,
		pageSize:   constants.LargePageSize, // Larger page size for efficiency
		maxPages:   constants.MaxPages,      // Prevent infinite loops
		cache:      true,
		cacheStore: GetDefaultPerformanceService().cache,
	}
}

// GetAllUsers efficiently retrieves all users with caching.
func (optPagination *OptimizedPagination) GetAllUsers(filter, sortBy, attributes string, sortOrder uaa.SortOrder) ([]uaa.User, error) {
	return optPagination.getAllUsers(
		fmt.Sprintf("all_users:%s:%s:%s:%s", filter, sortBy, attributes, sortOrder),
		func(startIndex int) ([]uaa.User, any, error) {
			return optPagination.client.Client().ListUsers(filter, sortBy, attributes, sortOrder, startIndex, optPagination.pageSize)
		},
	)
}

// GetAllGroups efficiently retrieves all groups with caching.
func (optPagination *OptimizedPagination) GetAllGroups(filter, sortBy, attributes string, sortOrder uaa.SortOrder) ([]uaa.Group, error) {
	return optPagination.getAllGroups(
		fmt.Sprintf("all_groups:%s:%s:%s:%s", filter, sortBy, attributes, sortOrder),
		func(startIndex int) ([]uaa.Group, any, error) {
			return optPagination.client.Client().ListGroups(filter, sortBy, attributes, sortOrder, startIndex, optPagination.pageSize)
		},
	)
}

// GetAllClients efficiently retrieves all clients with caching.
func (optPagination *OptimizedPagination) GetAllClients(filter, sortBy, attributes string, sortOrder uaa.SortOrder) ([]uaa.Client, error) {
	cacheKey := fmt.Sprintf("all_clients:%s:%s:%s:%s", filter, sortBy, attributes, sortOrder)

	// Check cache if enabled
	if optPagination.cache {
		if cached, found := optPagination.cacheStore.Get(cacheKey); found {
			if clients, ok := cached.([]uaa.Client); ok {
				return clients, nil
			}
		}
	}

	// Fetch all clients with optimized pagination
	var allClients []uaa.Client

	startIndex := 1

	for range optPagination.maxPages {
		clients, pagination, err := optPagination.client.Client().ListClients(filter, sortBy, sortOrder, startIndex, optPagination.pageSize)
		if err != nil {
			return nil, fmt.Errorf("failed to list clients: %w", err)
		}

		allClients = append(allClients, clients...)

		// Check if we have more pages
		if pagination.TotalResults <= startIndex+len(clients)-1 {
			break
		}

		startIndex += len(clients)
	}

	// Cache the result if enabled
	if optPagination.cache {
		optPagination.cacheStore.Set(cacheKey, allClients)
	}

	return allClients, nil
}

// getAllUsers is a generic method to fetch all resources with optimized pagination.
func (optPagination *OptimizedPagination) getAllUsers(
	cacheKey string,
	listFunc func(startIndex int) ([]uaa.User, any, error),
) ([]uaa.User, error) {
	return getAllResourcesGeneric[uaa.User](optPagination, cacheKey, "users", listFunc)
}

// getAllGroups is a helper method to fetch all groups with optimized pagination.
func (optPagination *OptimizedPagination) getAllGroups(
	cacheKey string,
	listFunc func(startIndex int) ([]uaa.Group, any, error),
) ([]uaa.Group, error) {
	return getAllResourcesGeneric[uaa.Group](optPagination, cacheKey, "groups", listFunc)
}

// getAllResourcesGeneric is a generic function for fetching all resources with optimized pagination.
func getAllResourcesGeneric[T any](
	optPagination *OptimizedPagination,
	cacheKey, entityType string,
	listFunc func(startIndex int) ([]T, any, error),
) ([]T, error) {
	// Check cache if enabled
	if optPagination.cache {
		if cached, found := optPagination.cacheStore.Get(cacheKey); found {
			if resources, ok := cached.([]T); ok {
				return resources, nil
			}
		}
	}

	// Fetch all resources with optimized pagination
	var allResources []T

	startIndex := 1

	for range optPagination.maxPages {
		resources, pagination, err := listFunc(startIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to list %s: %w", entityType, err)
		}

		allResources = append(allResources, resources...)

		// Check if we have more pages
		// Use reflection to access TotalResults since pagination is any
		if paginationValue := reflect.ValueOf(pagination); paginationValue.IsValid() {
			if totalField := paginationValue.Elem().FieldByName("TotalResults"); totalField.IsValid() {
				totalResults := int(totalField.Int())
				if totalResults <= startIndex+len(resources)-1 {
					break
				}
			}
		}

		startIndex += len(resources)
	}

	// Cache the result if enabled
	if optPagination.cache {
		optPagination.cacheStore.Set(cacheKey, allResources)
	}

	return allResources, nil
}

// BulkUserImport handles efficient bulk user import from JSON.
func BulkUserImport(client *UAAClientWrapper, usersJSON []byte, parallel bool) ([]BatchResult, error) {
	var users []uaa.User

	err := json.Unmarshal(usersJSON, &users)
	if err != nil {
		return nil, fmt.Errorf("failed to parse users JSON: %w", err)
	}

	// Create batch operations
	operations := make([]BatchOperation, len(users))
	for i, user := range users {
		operations[i] = BatchOperation{
			Type:     Create,
			Resource: batchResourceUser,
			Data:     user,
		}
	}

	if parallel {
		// Use batch processor for parallel execution
		processor := NewBatchProcessor(client)

		return processor.ProcessBatch(operations), nil
	} else {
		// Sequential processing
		results := make([]BatchResult, len(operations))
		for index, operation := range operations {
			user, ok := operation.Data.(uaa.User)
			if !ok {
				results[index] = BatchResult{
					Operation: operation,
					Result:    nil,
					Error:     constants.ErrInvalidDataTypeExpectedUAAUser,
				}

				continue
			}

			result, err := client.Client().CreateUser(user)
			results[index] = BatchResult{
				Operation: operation,
				Result:    result,
				Error:     err,
			}
		}

		return results, nil
	}
}

// PerformanceMetrics tracks operation performance.
type PerformanceMetrics struct {
	mutex           sync.RWMutex
	operations      map[string][]time.Duration
	cacheHits       int64
	cacheMisses     int64
	totalOperations int64
}

// TrackOperation records the duration of an operation.
func (pm *PerformanceMetrics) TrackOperation(operation string, duration time.Duration) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.operations[operation] = append(pm.operations[operation], duration)
	pm.totalOperations++
}

// TrackCacheHit records a cache hit.
func (pm *PerformanceMetrics) TrackCacheHit() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.cacheHits++
}

// TrackCacheMiss records a cache miss.
func (pm *PerformanceMetrics) TrackCacheMiss() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.cacheMisses++
}

// GetMetrics returns current performance metrics.
func (pm *PerformanceMetrics) GetMetrics() map[string]any {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var cacheHitRate float64

	totalCacheOps := pm.cacheHits + pm.cacheMisses
	if totalCacheOps > 0 {
		cacheHitRate = float64(pm.cacheHits) / float64(totalCacheOps) * constants.PercentageMultiplierFloat
	}

	metrics := map[string]any{
		"total_operations": pm.totalOperations,
		"cache_hits":       pm.cacheHits,
		"cache_misses":     pm.cacheMisses,
		"cache_hit_rate":   cacheHitRate,
		"operations":       make(map[string]any),
	}

	// Calculate operation statistics
	for operation, durations := range pm.operations {
		if len(durations) > 0 {
			var total time.Duration

			minDuration := durations[0]
			maxDuration := durations[0]

			for _, duration := range durations {
				total += duration
				if duration < minDuration {
					minDuration = duration
				}

				if duration > maxDuration {
					maxDuration = duration
				}
			}

			avg := total / time.Duration(len(durations))

			if operations, ok := metrics["operations"].(map[string]any); ok {
				operations[operation] = map[string]any{
					"count":   len(durations),
					"average": avg.String(),
					"min":     minDuration.String(),
					"max":     maxDuration.String(),
				}
			}
		}
	}

	return metrics
}

// WithPerformanceTracking wraps a function with performance tracking.
func WithPerformanceTracking(operation string, fn func() error) error {
	return GetDefaultPerformanceService().WithPerformanceTracking(operation, fn)
}

func (s *UAAPerformanceService) WithPerformanceTracking(operation string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	s.metrics.TrackOperation(operation, duration)

	return err
}
