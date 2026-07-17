package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/fivetwenty-io/capi/v3/pkg/cfclient"
)

// operationTypeOrg identifies a batch Operation that creates an organization.
const operationTypeOrg = "org"

func main() {
	// Example 1: Advanced client configuration
	_, _ = os.Stdout.WriteString("=== Advanced Client Configuration ===\n")

	advancedConfigExample()

	// Example 2: Concurrent operations
	_, _ = os.Stdout.WriteString("\n=== Concurrent Operations ===\n")

	concurrentOperationsExample()

	// Example 3: Batch operations
	_, _ = os.Stdout.WriteString("\n=== Batch Operations ===\n")

	batchOperationsExample()

	// Example 4: Advanced error handling and retries
	_, _ = os.Stdout.WriteString("\n=== Advanced Error Handling ===\n")

	errorHandlingExample()

	// Example 5: Streaming large datasets
	_, _ = os.Stdout.WriteString("\n=== Streaming Large Datasets ===\n")

	streamingExample()

	// Example 6: Custom interceptors
	_, _ = os.Stdout.WriteString("\n=== Custom Interceptors ===\n")

	interceptorsExample()

	// Example 7: Performance monitoring
	_, _ = os.Stdout.WriteString("\n=== Performance Monitoring ===\n")

	performanceMonitoringExample()
}

func advancedConfigExample() {
	// Create a highly customized client configuration
	config := &capi.Config{
		APIEndpoint:   "https://api.your-cf-domain.com",
		Username:      "your-username",
		Password:      "your-password",
		SkipTLSVerify: false,
		HTTPTimeout:   constants.ExtendedHTTPTimeout,
		UserAgent:     "advanced-example/1.0.0",
		RetryMax:      constants.DefaultRetryMax,
		RetryWaitMin:  1 * time.Second,
		RetryWaitMax:  constants.DefaultRetryWaitMax,
		Debug:         true,
	}

	ctx := context.Background()

	client, err := cfclient.New(ctx, config)
	if err != nil {
		log.Printf("Failed to create advanced client: %v", err)

		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultHTTPTimeout)
	defer cancel()

	// Test the configuration
	info, err := client.GetInfo(ctx)
	if err != nil {
		log.Printf("Failed to get API info: %v", err)

		return
	}

	_, _ = fmt.Fprintf(os.Stdout, "Connected to CF API version %d with advanced configuration\n", info.Version)
}

func concurrentOperationsExample() {
	ctx := context.Background()

	client, err := cfclient.NewWithPassword(ctx,
		"https://api.your-cf-domain.com",
		"your-username",
		"your-password",
	)
	if err != nil {
		log.Printf("Failed to create client: %v", err)

		return
	}

	// Get all organizations concurrently with their spaces
	orgs, err := client.Organizations().List(ctx, nil)
	if err != nil {
		log.Printf("Failed to list organizations: %v", err)

		return
	}

	_, _ = fmt.Fprintf(os.Stdout, "Processing %d organizations concurrently...\n", len(orgs.Resources))

	resultsChan := processOrganizationsConcurrently(client, ctx, orgs.Resources)
	collectAndReportOrgResults(resultsChan)
}

// processOrganizationsConcurrently processes organizations using a worker pool pattern.
func processOrganizationsConcurrently(client capi.Client, ctx context.Context, orgs []capi.Organization) <-chan OrgSpaceResult {
	const numWorkers = 5

	orgChan := make(chan *capi.Organization, len(orgs))
	resultsChan := make(chan OrgSpaceResult, len(orgs))

	// Start workers
	var waitGroup sync.WaitGroup
	for workerIndex := range numWorkers {
		waitGroup.Add(1)

		go func(workerID int) {
			_ = workerIndex // Use the index variable

			defer waitGroup.Done()

			for org := range orgChan {
				result := processOrganizationSpaces(client, ctx, org, workerID)
				resultsChan <- result
			}
		}(workerIndex)
	}

	// Send work to workers
	go func() {
		for _, org := range orgs {
			orgChan <- &org
		}

		close(orgChan)
	}()

	// Wait for workers to complete
	go func() {
		waitGroup.Wait()
		close(resultsChan)
	}()

	return resultsChan
}

// collectAndReportOrgResults collects and reports the results from concurrent organization processing.
func collectAndReportOrgResults(resultsChan <-chan OrgSpaceResult) {
	totalSpaces := 0

	for result := range resultsChan {
		if result.Error != nil {
			_, _ = fmt.Fprintf(os.Stdout, "Error processing org %s: %v\n", result.OrgName, result.Error)
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "Org %s has %d spaces (processed by worker %d)\n",
				result.OrgName, result.SpaceCount, result.WorkerID)
			totalSpaces += result.SpaceCount
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "Total spaces across all organizations: %d\n", totalSpaces)
}

type OrgSpaceResult struct {
	OrgName    string
	SpaceCount int
	WorkerID   int
	Error      error
}

func processOrganizationSpaces(client capi.Client, ctx context.Context, org *capi.Organization, workerID int) OrgSpaceResult {
	params := capi.NewQueryParams().WithFilter("organization_guids", org.GUID)

	spaces, err := client.Spaces().List(ctx, params)
	if err != nil {
		return OrgSpaceResult{
			OrgName:  org.Name,
			WorkerID: workerID,
			Error:    err,
		}
	}

	return OrgSpaceResult{
		OrgName:    org.Name,
		SpaceCount: len(spaces.Resources),
		WorkerID:   workerID,
	}
}

func batchOperationsExample() {
	ctx := context.Background()

	client, err := cfclient.NewWithPassword(ctx,
		"https://api.your-cf-domain.com",
		"your-username",
		"your-password",
	)
	if err != nil {
		log.Printf("Failed to create client: %v", err)

		return
	}

	operations := createBatchOperations(client, ctx)
	results := executeBatchOperations(operations)
	reportBatchResults(results)
}

// Operation represents a batch operation.
type Operation struct {
	Type string
	Name string
	Func func() error
}

// createBatchOperations creates the batch operations to execute.
func createBatchOperations(client capi.Client, ctx context.Context) []Operation {
	return []Operation{
		{
			Type: operationTypeOrg,
			Name: "batch-org-1",
			Func: func() error {
				_, err := client.Organizations().Create(ctx, &capi.OrganizationCreateRequest{
					Name: "batch-org-1",
				})
				if err != nil {
					return fmt.Errorf("failed to create batch-org-1: %w", err)
				}

				return nil
			},
		},
		{
			Type: operationTypeOrg,
			Name: "batch-org-2",
			Func: func() error {
				_, err := client.Organizations().Create(ctx, &capi.OrganizationCreateRequest{
					Name: "batch-org-2",
				})
				if err != nil {
					return fmt.Errorf("failed to create batch-org-2: %w", err)
				}

				return nil
			},
		},
		{
			Type: operationTypeOrg,
			Name: "batch-org-3",
			Func: func() error {
				_, err := client.Organizations().Create(ctx, &capi.OrganizationCreateRequest{
					Name: "batch-org-3",
				})
				if err != nil {
					return fmt.Errorf("failed to create batch-org-3: %w", err)
				}

				return nil
			},
		},
	}
}

// executeBatchOperations executes batch operations with rate limiting.
func executeBatchOperations(operations []Operation) <-chan OperationResult {
	semaphore := make(chan struct{}, constants.DefaultConcurrencyLimit) // Limit to 3 concurrent operations

	var waitGroup sync.WaitGroup

	results := make(chan OperationResult, len(operations))

	for _, operation := range operations {
		waitGroup.Add(1)

		go func(operation Operation) {
			defer waitGroup.Done()

			semaphore <- struct{}{} // Acquire

			defer func() { <-semaphore }() // Release

			start := time.Now()
			err := operation.Func()
			duration := time.Since(start)

			results <- OperationResult{
				Type:     operation.Type,
				Name:     operation.Name,
				Duration: duration,
				Error:    err,
			}
		}(operation)
	}

	// Wait for completion and collect results
	go func() {
		waitGroup.Wait()
		close(results)
	}()

	return results
}

// reportBatchResults reports the results of batch operations.
func reportBatchResults(results <-chan OperationResult) {
	successful := 0
	failed := 0

	for result := range results {
		if result.Error != nil {
			_, _ = fmt.Fprintf(os.Stdout, "❌ %s %s failed in %v: %v\n",
				result.Type, result.Name, result.Duration, result.Error)

			failed++
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "✅ %s %s succeeded in %v\n",
				result.Type, result.Name, result.Duration)

			successful++
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "Batch operations completed: %d successful, %d failed\n", successful, failed)
}

type OperationResult struct {
	Type     string
	Name     string
	Duration time.Duration
	Error    error
}

func errorHandlingExample() {
	ctx := context.Background()

	client, err := cfclient.NewWithPassword(ctx,
		"https://api.your-cf-domain.com",
		"wrong-username", // Intentionally wrong credentials
		"wrong-password",
	)
	if err != nil {
		log.Printf("Failed to create client: %v", err)

		return
	}

	// Demonstrate comprehensive error handling with retries
	err = withExponentialBackoff(func() error {
		_, err := client.Organizations().List(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to list organizations: %w", err)
		}

		return nil
	}, constants.DefaultConcurrencyLimit, time.Second)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Operation failed after retries: %v\n", err)

		// Analyze the error
		analyzeError(err)
	}

	// Demonstrate timeout handling
	shortCtx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err = client.Organizations().List(shortCtx, nil)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Timeout error (expected): %v\n", err)
	}
}

func withExponentialBackoff(operation func() error, maxRetries int, baseDelay time.Duration) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			_, _ = fmt.Fprintf(os.Stdout, "Non-retryable error encountered: %v\n", err)

			return err
		}

		if attempt < maxRetries {
			delay := time.Duration(math.Pow(constants.ExponentialBackoffBase, float64(attempt))) * baseDelay
			_, _ = fmt.Fprintf(os.Stdout, "Attempt %d failed, retrying in %v: %v\n", attempt+1, delay, err)
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", maxRetries+1, lastErr)
}

func isRetryableError(err error) bool {
	capiErr := &capi.ResponseError{}
	if errors.As(err, &capiErr) {
		// Check if any error is retryable (5xx status codes)
		for _, apiErr := range capiErr.Errors {
			if apiErr.Code >= 50000 && apiErr.Code < 60000 {
				return true
			}
		}

		return false
	}

	// Retry network errors
	return true
}

func analyzeError(err error) {
	var responseError *capi.ResponseError
	if !errors.As(err, &responseError) {
		printNonAPIError(err)

		return
	}

	printAPIErrorSummary(responseError)

	if len(responseError.Errors) > 0 {
		printIndividualErrors(responseError.Errors)
		printErrorAdvice(responseError.Errors[0])
	}
}

func printNonAPIError(err error) {
	_, _ = fmt.Fprintf(os.Stdout, "Other Error: %v (type: %T)\n", err, err)
}

func printAPIErrorSummary(e *capi.ResponseError) {
	_, _ = os.Stdout.WriteString("CF API Error Analysis:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  Error count: %d\n", len(e.Errors))
}

func printIndividualErrors(errors []capi.APIError) {
	_, _ = os.Stdout.WriteString("  Individual Errors:\n")

	for _, detail := range errors {
		_, _ = fmt.Fprintf(os.Stdout, "    - Code: %d, Title: %s, Detail: %s\n", detail.Code, detail.Title, detail.Detail)
	}
}

func printErrorAdvice(firstError capi.APIError) {
	advice := getErrorAdvice(firstError.Code)
	if advice != "" {
		_, _ = fmt.Fprintf(os.Stdout, "  💡 %s\n", advice)
	}
}

func getErrorAdvice(errorCode int) string {
	switch {
	case errorCode >= 10000 && errorCode < 11000:
		return "Check your credentials or refresh your token"
	case errorCode >= 10003 && errorCode < 10004:
		return "You may lack the required permissions for this operation"
	case errorCode == constants.CFErrorCodeNotFound:
		return "The requested resource was not found"
	case errorCode >= 10008 && errorCode < 10009:
		return "The request was invalid - check required fields and formats"
	case errorCode >= constants.CFErrorCodeServerError:
		return "Server error - try again later or contact support"
	default:
		return ""
	}
}

func streamingExample() {
	ctx := context.Background()

	client, err := cfclient.NewWithPassword(ctx,
		"https://api.your-cf-domain.com",
		"your-username",
		"your-password",
	)
	if err != nil {
		log.Printf("Failed to create client: %v", err)

		return
	}

	// Stream applications in batches to reduce memory usage
	_, _ = os.Stdout.WriteString("Streaming applications in batches...\n")

	processedCount := 0
	batchSize := 10
	params := capi.NewQueryParams().WithPerPage(batchSize)

	appList, err := client.Apps().List(ctx, params)
	if err == nil {
		// Process each batch
		_, _ = fmt.Fprintf(os.Stdout, "Processing batch of %d applications...\n", len(appList.Resources))

		for range appList.Resources {
			// Simulate processing each application
			processedCount++

			if processedCount%50 == 0 {
				_, _ = fmt.Fprintf(os.Stdout, "Processed %d applications so far...\n", processedCount)
			}

			// Add artificial delay to simulate processing time
			time.Sleep(constants.QuickPollInterval)
		}
	}

	if err != nil {
		log.Printf("Streaming failed: %v", err)

		return
	}

	_, _ = fmt.Fprintf(os.Stdout, "Streaming completed. Total applications processed: %d\n", processedCount)
}

func interceptorsExample() {
	ctx := context.Background()

	client, err := cfclient.NewWithPassword(ctx,
		"https://api.your-cf-domain.com",
		"your-username",
		"your-password",
	)
	if err != nil {
		log.Printf("Failed to create client: %v", err)

		return
	}

	// Note: This is a conceptual example. The actual client implementation
	// would need to support interceptors. Here we show what it might look like.

	_, _ = os.Stdout.WriteString("Demonstrating request/response interceptor concepts...\n")

	// Conceptual request interceptor
	requestInterceptor := func(req *http.Request) {
		_, _ = fmt.Fprintf(os.Stdout, "🚀 Outgoing request: %s %s\n", req.Method, req.URL.Path)
		req.Header.Set("X-Custom-Client", "advanced-example")
	}

	// Conceptual response interceptor
	responseInterceptor := func(resp *http.Response) {
		_, _ = fmt.Fprintf(os.Stdout, "📥 Incoming response: %d %s (took %v)\n",
			resp.StatusCode, resp.Status,
			resp.Header.Get("X-Response-Time"))
	}

	// In a real implementation, you would add these interceptors:
	// client.AddRequestInterceptor(requestInterceptor)
	// client.AddResponseInterceptor(responseInterceptor)

	// Make a request to demonstrate interceptor concepts
	_, err = client.GetInfo(ctx)
	if err != nil {
		log.Printf("Request failed: %v", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Request/response interceptors would capture: %v, %v\n",
		requestInterceptor != nil, responseInterceptor != nil)
}

func performanceMonitoringExample() {
	ctx := context.Background()

	client, err := cfclient.NewWithPassword(ctx,
		"https://api.your-cf-domain.com",
		"your-username",
		"your-password",
	)
	if err != nil {
		log.Printf("Failed to create client: %v", err)

		return
	}

	metrics := &PerformanceMetrics{
		RequestCount: 0,
		TotalTime:    0,
		Errors:       0,
	}

	operations := createMonitoringOperations(client, ctx)
	runPerformanceTests(operations, metrics)
	reportPerformanceMetrics(metrics)
}

// createMonitoringOperations creates the operations to monitor.
func createMonitoringOperations(client capi.Client, ctx context.Context) []struct {
	name string
	fn   func() error
} {
	return []struct {
		name string
		fn   func() error
	}{
		{
			name: "list organizations",
			fn: func() error {
				_, err := client.Organizations().List(ctx, nil)
				if err != nil {
					return fmt.Errorf("failed to list organizations in monitoring: %w", err)
				}

				return nil
			},
		},
		{
			name: "get API info",
			fn: func() error {
				_, err := client.GetInfo(ctx)
				if err != nil {
					return fmt.Errorf("failed to get API info in monitoring: %w", err)
				}

				return nil
			},
		},
		{
			name: "list applications",
			fn: func() error {
				params := capi.NewQueryParams().WithPerPage(constants.DefaultPageSize)

				_, err := client.Apps().List(ctx, params)
				if err != nil {
					return fmt.Errorf("failed to list applications in monitoring: %w", err)
				}

				return nil
			},
		},
	}
}

// runPerformanceTests runs the performance monitoring tests.
func runPerformanceTests(operations []struct {
	name string
	fn   func() error
}, metrics *PerformanceMetrics) {
	_, _ = os.Stdout.WriteString("Running performance monitoring tests...\n")

	for _, operation := range operations {
		start := time.Now()
		err := operation.fn()
		duration := time.Since(start)

		metrics.RequestCount++
		metrics.TotalTime += duration

		if err != nil {
			metrics.Errors++

			_, _ = fmt.Fprintf(os.Stdout, "❌ %s failed in %v: %v\n", operation.name, duration, err)
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "✅ %s completed in %v\n", operation.name, duration)
		}
	}
}

// reportPerformanceMetrics reports the performance metrics summary.
func reportPerformanceMetrics(metrics *PerformanceMetrics) {
	_, _ = os.Stdout.WriteString("\nPerformance Summary:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  Total requests: %d\n", metrics.RequestCount)
	_, _ = fmt.Fprintf(os.Stdout, "  Total time: %v\n", metrics.TotalTime)
	_, _ = fmt.Fprintf(os.Stdout, "  Average time per request: %v\n", metrics.TotalTime/time.Duration(metrics.RequestCount))
	_, _ = fmt.Fprintf(os.Stdout, "  Error rate: %.2f%%\n", float64(metrics.Errors)/float64(metrics.RequestCount)*constants.PercentageMultiplier)
	_, _ = fmt.Fprintf(os.Stdout, "  Requests per second: %.2f\n", float64(metrics.RequestCount)/metrics.TotalTime.Seconds())
}

type PerformanceMetrics struct {
	RequestCount int
	TotalTime    time.Duration
	Errors       int
}
