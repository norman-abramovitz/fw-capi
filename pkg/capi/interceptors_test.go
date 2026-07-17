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

func TestInterceptorChain_RequestInterceptors(t *testing.T) {
	t.Parallel()

	chain := capi.NewInterceptorChain()
	ctx := context.Background()

	var executionOrder []string

	// Add multiple interceptors
	chain.AddRequestInterceptor(func(ctx context.Context, req *capi.Request) error {
		executionOrder = append(executionOrder, "first")

		return nil
	})

	chain.AddRequestInterceptor(func(ctx context.Context, req *capi.Request) error {
		executionOrder = append(executionOrder, "second")

		return nil
	})

	req := &capi.Request{
		Method: http.MethodGet,
		Path:   testPath,
	}

	err := chain.ExecuteRequestInterceptors(ctx, req)
	require.NoError(t, err)

	assert.Equal(t, []string{"first", "second"}, executionOrder)
}

func TestInterceptorChain_ResponseInterceptors(t *testing.T) {
	t.Parallel()

	chain := capi.NewInterceptorChain()
	ctx := context.Background()

	var executionOrder []string

	// Add multiple interceptors
	chain.AddResponseInterceptor(func(ctx context.Context, req *capi.Request, resp *capi.Response) error {
		executionOrder = append(executionOrder, "first")

		return nil
	})

	chain.AddResponseInterceptor(func(ctx context.Context, req *capi.Request, resp *capi.Response) error {
		executionOrder = append(executionOrder, "second")

		return nil
	})

	req := &capi.Request{
		Method: http.MethodGet,
		Path:   testPath,
	}
	resp := &capi.Response{
		StatusCode: 200,
	}

	err := chain.ExecuteResponseInterceptors(ctx, req, resp)
	require.NoError(t, err)

	assert.Equal(t, []string{"first", "second"}, executionOrder)
}

func TestHeaderInterceptor(t *testing.T) {
	t.Parallel()

	headers := map[string]string{
		"X-Custom-Header": "custom-value",
		"X-Request-ID":    "123456",
	}

	interceptor := capi.HeaderInterceptor(headers)
	ctx := context.Background()
	req := &capi.Request{
		Method: http.MethodGet,
		Path:   testPath,
	}

	err := interceptor(ctx, req)
	require.NoError(t, err)

	assert.Equal(t, "custom-value", req.Headers.Get("X-Custom-Header"))
	assert.Equal(t, "123456", req.Headers.Get("X-Request-ID"))
}

func TestAuthenticationInterceptor(t *testing.T) {
	t.Parallel()

	tokenProvider := func(ctx context.Context) (string, error) {
		return "test-token", nil
	}

	interceptor := capi.AuthenticationInterceptor(tokenProvider)
	ctx := context.Background()
	req := &capi.Request{
		Method: http.MethodGet,
		Path:   testPath,
	}

	err := interceptor(ctx, req)
	require.NoError(t, err)

	assert.Equal(t, "Bearer test-token", req.Headers.Get("Authorization"))
}

func TestTimeoutInterceptor(t *testing.T) {
	t.Parallel()

	interceptor := capi.TimeoutInterceptor(5 * time.Second)
	ctx := context.Background()
	req := &capi.Request{
		Method: "GET",
		Path:   "/test",
	}

	err := interceptor(ctx, req)
	require.NoError(t, err)

	// TimeoutInterceptor currently just returns nil
	// The actual timeout handling is done by the HTTP client
}

func TestMetricsCollector(t *testing.T) {
	t.Parallel()

	collector := capi.NewMetricsCollector()

	var (
		notifiedEndpoint string
		notifiedMetrics  *capi.Metrics
	)

	collector.SetOnChange(func(endpoint string, metrics *capi.Metrics) {
		notifiedEndpoint = endpoint
		notifiedMetrics = metrics
	})

	// Set up interceptors
	requestInterceptor := capi.MetricsRequestInterceptor(collector)
	responseInterceptor := capi.MetricsResponseInterceptor(collector)

	ctx := context.Background()
	req := &capi.Request{
		Method: http.MethodGet,
		Path:   testAppsPath,
	}

	// Execute request interceptor
	err := requestInterceptor(ctx, req)
	require.NoError(t, err)

	// Execute response interceptor with success
	resp := &capi.Response{
		StatusCode: 200,
	}
	err = responseInterceptor(ctx, req, resp)
	require.NoError(t, err)

	// Check metrics — latency >= 0 avoids scheduler-timing dependency.
	assert.Equal(t, "GET /v3/apps", notifiedEndpoint)
	assert.NotNil(t, notifiedMetrics)
	assert.Equal(t, int64(1), notifiedMetrics.TotalRequests)
	assert.Equal(t, int64(0), notifiedMetrics.TotalErrors)
	assert.GreaterOrEqual(t, notifiedMetrics.AverageLatency.Nanoseconds(), int64(0))

	// Execute another request with error
	req2 := &capi.Request{
		Method: http.MethodGet,
		Path:   testAppsPath,
	}

	// Execute request interceptor for the second request
	err = requestInterceptor(ctx, req2)
	require.NoError(t, err)

	resp2 := &capi.Response{
		StatusCode: 500,
	}
	err = responseInterceptor(ctx, req2, resp2)
	require.NoError(t, err)

	// Check updated metrics
	metrics := collector.GetMetrics("GET /v3/apps")
	assert.Equal(t, int64(2), metrics.TotalRequests)
	assert.Equal(t, int64(1), metrics.TotalErrors)
}

func TestCircuitBreaker(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("wall-clock sleep; deterministic companion TestCircuitBreakerTimeout_Deterministic covers logic")
	}

	config := &capi.CircuitBreakerConfig{
		Threshold:        2,
		Timeout:          100 * time.Millisecond,
		SuccessThreshold: 1,
	}
	breaker := capi.NewCircuitBreaker(config)

	requestInterceptor := capi.CircuitBreakerRequestInterceptor(breaker)
	responseInterceptor := capi.CircuitBreakerResponseInterceptor(breaker)

	ctx := context.Background()
	req := &capi.Request{
		Method: "GET",
		Path:   "/test",
	}

	// Circuit should be closed initially
	err := requestInterceptor(ctx, req)
	require.NoError(t, err)

	// Simulate failures
	for range 2 {
		resp := &capi.Response{StatusCode: 500}
		err = responseInterceptor(ctx, req, resp)
		require.NoError(t, err)
	}

	// Circuit should be open now
	err = requestInterceptor(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker is open")

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Circuit should be half-open now
	err = requestInterceptor(ctx, req)
	require.NoError(t, err)

	// Simulate success
	resp := &capi.Response{StatusCode: 200}
	err = responseInterceptor(ctx, req, resp)
	require.NoError(t, err)

	// Circuit should be closed again
	err = requestInterceptor(ctx, req)
	require.NoError(t, err)
}

func TestRetryResponseInterceptor(t *testing.T) {
	t.Parallel()

	config := &capi.RetryConfig{
		MaxRetries:   3,
		RetryDelay:   100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		RetryOnCodes: []int{429, 500, 502, 503, 504},
	}

	interceptor := capi.RetryResponseInterceptor(config)
	ctx := context.Background()
	req := &capi.Request{
		Method: http.MethodGet,
		Path:   testPath,
	}

	// Test retryable status code
	resp := &capi.Response{
		StatusCode: 500,
		Headers:    make(http.Header),
	}

	err := interceptor(ctx, req, resp)
	require.NoError(t, err)
	assert.Equal(t, stringTrue, resp.Headers.Get("X-Should-Retry"))

	// Test non-retryable status code
	resp2 := &capi.Response{
		StatusCode: 404,
		Headers:    make(http.Header),
	}

	err = interceptor(ctx, req, resp2)
	require.NoError(t, err)
	assert.Empty(t, resp2.Headers.Get("X-Should-Retry"))
}
