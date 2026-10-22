package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	capihttp "github.com/fivetwenty-io/capi/v3/internal/http"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRetryMax     = 3
	testRetryAttempt = testRetryMax + 1
	testRetryPath    = "/v3/info"
	testCFEnvelope   = `{"errors":[{"code":10015,"title":"CF-ServiceUnavailable",` +
		`"detail":"Service broker is unavailable"}]}`
)

func newRetryTestClient(serverURL string) *capihttp.Client {
	return capihttp.NewClient(serverURL, nil,
		capihttp.WithRetryConfig(testRetryMax, time.Millisecond, 5*time.Millisecond))
}

// statusServer answers every request with status and body, counting attempts.
func statusServer(t *testing.T, status int, body string) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)

		if body != "" {
			writer.Header().Set("Content-Type", "application/json")
		}

		writer.WriteHeader(status)
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server, &attempts
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestClient_RetriesExhaustedOnServerError(t *testing.T) {
	t.Parallel()

	t.Run("503 without body maps to ErrServerError", func(t *testing.T) {
		t.Parallel()

		server, attempts := statusServer(t, http.StatusServiceUnavailable, "")

		resp, err := newRetryTestClient(server.URL).Get(context.Background(), testRetryPath, nil)
		require.Error(t, err)
		require.ErrorIs(t, err, capi.ErrServerError)
		require.NotNil(t, resp)
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		assert.Contains(t, err.Error(), "status 503")
		assert.Equal(t, int32(testRetryAttempt), attempts.Load())
	})

	t.Run("503 with a v3 error envelope keeps the envelope", func(t *testing.T) {
		t.Parallel()

		server, attempts := statusServer(t, http.StatusServiceUnavailable, testCFEnvelope)

		resp, err := newRetryTestClient(server.URL).Get(context.Background(), testRetryPath, nil)
		require.ErrorIs(t, err, capi.ErrServerError)

		var respErr *capi.ResponseError
		require.ErrorAs(t, err, &respErr)
		require.Len(t, respErr.Errors, 1)
		assert.Equal(t, 10015, respErr.Errors[0].Code)
		assert.Equal(t, "CF-ServiceUnavailable", respErr.Errors[0].Title)

		require.NotNil(t, resp)
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		assert.JSONEq(t, testCFEnvelope, string(resp.Body))
		assert.Equal(t, int32(testRetryAttempt), attempts.Load())
	})

	t.Run("PostRaw 503 maps to ErrServerError", func(t *testing.T) {
		t.Parallel()

		server, attempts := statusServer(t, http.StatusServiceUnavailable, testCFEnvelope)

		resp, err := newRetryTestClient(server.URL).PostRaw(context.Background(), testRetryPath,
			[]byte("payload"), "application/octet-stream")
		require.ErrorIs(t, err, capi.ErrServerError)

		var respErr *capi.ResponseError
		require.ErrorAs(t, err, &respErr)
		require.NotNil(t, resp)
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		assert.Equal(t, int32(testRetryAttempt), attempts.Load())
	})

	t.Run("429 maps to ErrRateLimited", func(t *testing.T) {
		t.Parallel()

		server, attempts := statusServer(t, http.StatusTooManyRequests, "")

		resp, err := newRetryTestClient(server.URL).Get(context.Background(), testRetryPath, nil)
		require.ErrorIs(t, err, capi.ErrRateLimited)
		require.NotNil(t, resp)
		assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
		assert.Equal(t, int32(testRetryAttempt), attempts.Load())
	})

	t.Run("500 that recovers on retry succeeds", func(t *testing.T) {
		t.Parallel()

		var attempts atomic.Int32

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			if attempts.Add(1) < 2 {
				writer.WriteHeader(http.StatusInternalServerError)

				return
			}

			_, _ = writer.Write([]byte(`{"ok":true}`))
		}))
		t.Cleanup(server.Close)

		resp, err := newRetryTestClient(server.URL).Get(context.Background(), testRetryPath, nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.JSONEq(t, `{"ok":true}`, string(resp.Body))
		assert.Equal(t, int32(2), attempts.Load())
	})

	t.Run("transport failure is not a server error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.NotFoundHandler())
		serverURL := server.URL
		server.Close()

		resp, err := newRetryTestClient(serverURL).Get(context.Background(), testRetryPath, nil)
		require.Error(t, err)
		assert.Nil(t, resp)
		require.NotErrorIs(t, err, capi.ErrServerError, "transport error must not carry a status sentinel")
		assert.Contains(t, err.Error(), "giving up after 4 attempt(s)")

		var urlErr *url.Error
		assert.ErrorAs(t, err, &urlErr, "transport error keeps the underlying *url.Error")
	})
}

type closeTrackingBody struct {
	*strings.Reader

	closed bool
}

func (b *closeTrackingBody) Close() error {
	b.closed = true

	return nil
}

func TestLastResponseErrorHandler_BodyHandling(t *testing.T) {
	t.Parallel()

	t.Run("hands back the last response with its body open", func(t *testing.T) {
		t.Parallel()

		body := &closeTrackingBody{Reader: strings.NewReader(testCFEnvelope)}
		last := &http.Response{StatusCode: http.StatusServiceUnavailable, Body: body}

		resp, err := capihttp.LastResponseErrorHandlerForTest(last, nil, testRetryAttempt)
		require.NoError(t, err)
		assert.Same(t, last, resp)
		assert.False(t, body.closed, "caller reads and closes the body")
		require.NoError(t, resp.Body.Close())
	})

	t.Run("closes the body when it returns an error", func(t *testing.T) {
		t.Parallel()

		body := &closeTrackingBody{Reader: strings.NewReader(testCFEnvelope)}
		last := &http.Response{StatusCode: http.StatusServiceUnavailable, Body: body}

		//nolint:bodyclose // the handler closes the body itself and returns no response
		resp, err := capihttp.LastResponseErrorHandlerForTest(last, context.Canceled, testRetryAttempt)
		require.ErrorIs(t, err, context.Canceled)
		require.ErrorIs(t, err, capihttp.ErrGivingUp)
		assert.Nil(t, resp)
		assert.True(t, body.closed)
	})
}
