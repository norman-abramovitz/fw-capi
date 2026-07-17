package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fivetwenty-io/capi/v3/internal/auth"
	capihttp "github.com/fivetwenty-io/capi/v3/internal/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Sentinel errors for refresh and token-fetch failure paths in tests.
var (
	errRefreshExplode = errors.New("test: refresh exploded")
	errTokenExplode   = errors.New("test: get-token exploded")
)

// Test constants for token-refresh fixtures. testPathApps, testFieldName,
// and testFieldKey are declared in client_test.go (same http_test package).
const (
	testTokenInitial   = "initial"
	testTokenRefreshed = "refreshed"
)

// refreshingTokenManager is a token manager stub that can be configured to
// fail either the initial GetToken, the RefreshToken call, or the post-
// refresh GetToken call. It tracks how many times each method is called so
// tests can assert the refresh-once-on-401 contract.
type refreshingTokenManager struct {
	initialToken   string
	refreshedToken string

	refreshErr    error
	getTokenErr   error
	postRefreshEr error

	getCalls     int32
	refreshCalls int32
	refreshed    int32 // 1 once RefreshToken has succeeded
}

func (m *refreshingTokenManager) GetToken(_ context.Context) (string, error) {
	atomic.AddInt32(&m.getCalls, 1)

	if atomic.LoadInt32(&m.refreshed) == 1 {
		if m.postRefreshEr != nil {
			return "", m.postRefreshEr
		}

		return m.refreshedToken, nil
	}

	if m.getTokenErr != nil {
		return "", m.getTokenErr
	}

	return m.initialToken, nil
}

func (m *refreshingTokenManager) RefreshToken(_ context.Context) error {
	atomic.AddInt32(&m.refreshCalls, 1)

	if m.refreshErr != nil {
		return m.refreshErr
	}

	atomic.StoreInt32(&m.refreshed, 1)

	return nil
}

func (m *refreshingTokenManager) SetToken(_ string, _ time.Time) {}

// TestAuthRetryTransport_Refresh401 verifies that a 401 response triggers a
// single RefreshToken + replay cycle and that the replay returns the new
// response to the caller.
func TestAuthRetryTransport_Refresh401(t *testing.T) {
	t.Parallel()

	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		n := atomic.AddInt32(&attempts, 1)

		if n == 1 {
			assert.Equal(t, "Bearer initial", request.Header.Get("Authorization"))
			writer.WriteHeader(http.StatusUnauthorized)

			return
		}

		assert.Equal(t, "Bearer refreshed", request.Header.Get("Authorization"))
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	tokenManager := &refreshingTokenManager{
		initialToken:   testTokenInitial,
		refreshedToken: testTokenRefreshed,
	}
	client := capihttp.NewClient(server.URL, tokenManager,
		capihttp.WithRetryConfig(0, time.Millisecond, time.Millisecond))

	resp, err := client.Get(context.Background(), testPathApps, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]bool
	require.NoError(t, json.Unmarshal(resp.Body, &body))
	assert.True(t, body["ok"])

	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts), "server should see exactly two attempts")
	assert.Equal(t, int32(1), atomic.LoadInt32(&tokenManager.refreshCalls), "RefreshToken should be called exactly once")
}

// TestAuthRetryTransport_NonUnauthorizedPassThrough verifies that a non-401
// response does NOT trigger a refresh.
func TestAuthRetryTransport_NonUnauthorizedPassThrough(t *testing.T) {
	t.Parallel()

	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&attempts, 1)

		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	tokenManager := &refreshingTokenManager{initialToken: testTokenInitial, refreshedToken: testTokenRefreshed}
	client := capihttp.NewClient(server.URL, tokenManager,
		capihttp.WithRetryConfig(0, time.Millisecond, time.Millisecond))

	resp, err := client.Get(context.Background(), testPathApps, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
	assert.Equal(t, int32(0), atomic.LoadInt32(&tokenManager.refreshCalls), "no refresh should occur on 200")
}

// TestAuthRetryTransport_RefreshFailureReturnsOriginal401 verifies that a
// RefreshToken failure causes the original 401 to be surfaced to the caller.
func TestAuthRetryTransport_RefreshFailureReturnsOriginal401(t *testing.T) {
	t.Parallel()

	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&attempts, 1)
		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = writer.Write([]byte(`{"errors":[{"code":10002,"title":"CF-NotAuthenticated","detail":"Not authenticated"}]}`))
	}))
	defer server.Close()

	tokenManager := &refreshingTokenManager{
		initialToken: testTokenInitial,
		refreshErr:   errRefreshExplode,
	}
	client := capihttp.NewClient(server.URL, tokenManager,
		capihttp.WithRetryConfig(0, time.Millisecond, time.Millisecond))

	resp, err := client.Get(context.Background(), testPathApps, nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Contains(t, string(resp.Body), "CF-NotAuthenticated")

	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts), "server should see exactly one attempt when refresh fails")
	assert.Equal(t, int32(1), atomic.LoadInt32(&tokenManager.refreshCalls))
}

// TestAuthRetryTransport_PostRefreshGetTokenFailure verifies that a failure
// to fetch the token AFTER a successful refresh also surfaces the original
// 401 unchanged (no retry replay).
func TestAuthRetryTransport_PostRefreshGetTokenFailure(t *testing.T) {
	t.Parallel()

	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&attempts, 1)
		writer.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	tokenManager := &refreshingTokenManager{
		initialToken:  testTokenInitial,
		postRefreshEr: errTokenExplode,
	}
	client := capihttp.NewClient(server.URL, tokenManager,
		capihttp.WithRetryConfig(0, time.Millisecond, time.Millisecond))

	resp, err := client.Get(context.Background(), testPathApps, nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Exactly one server attempt: refresh succeeded but the follow-up
	// GetToken failed before a retry could be issued.
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
	assert.Equal(t, int32(1), atomic.LoadInt32(&tokenManager.refreshCalls))
}

// streamingReadCloser is an io.ReadCloser that, when paired with a request
// via http.NewRequestWithContext, yields a request whose GetBody is nil —
// simulating a non-rewindable streaming body.
type streamingReadCloser struct {
	inner io.Reader
}

//nolint:wrapcheck // test fixture: pass the underlying reader's error through unwrapped
func (s *streamingReadCloser) Read(p []byte) (int, error) { return s.inner.Read(p) }
func (s *streamingReadCloser) Close() error               { return nil }

// TestAuthRetryTransport_StreamingBodyNoRetry verifies that when the
// request body cannot be rewound (GetBody == nil), a 401 is returned to the
// caller WITHOUT a refresh + retry attempt. This path cannot be exercised
// through the high-level capihttp.Client (which always produces rewindable
// *bytes.Reader bodies), so we use the unexported constructor directly via
// the test-only accessor in export_test.go.
func TestAuthRetryTransport_StreamingBodyNoRetry(t *testing.T) {
	t.Parallel()

	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&attempts, 1)

		_, _ = io.Copy(io.Discard, request.Body)

		writer.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	tokenManager := &refreshingTokenManager{initialToken: testTokenInitial, refreshedToken: testTokenRefreshed}

	transport := capihttp.NewAuthRetryTransportForTest(http.DefaultTransport, tokenManager)

	streaming := &streamingReadCloser{inner: bytes.NewReader([]byte("streamed"))}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL+testPathApps, streaming)
	require.NoError(t, err)

	// http.NewRequestWithContext only populates GetBody for body types it
	// recognises (*bytes.Buffer, *bytes.Reader, *strings.Reader). A bare
	// io.ReadCloser yields GetBody == nil — exactly the case we want.
	require.Nil(t, req.GetBody, "test precondition: request must have a non-rewindable body")
	req.Header.Set("Authorization", "Bearer initial")

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	// Exactly one server attempt — the streaming body prevents retry.
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
	assert.Equal(t, int32(0), atomic.LoadInt32(&tokenManager.refreshCalls),
		"streaming body must not trigger RefreshToken because replay is impossible")
}

// TestAuthRetryTransport_NilTokenManager verifies that a transport with a
// nil token manager degrades to a pure pass-through: a 401 is returned
// unchanged with no refresh attempt.
func TestAuthRetryTransport_NilTokenManager(t *testing.T) {
	t.Parallel()

	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&attempts, 1)
		writer.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	transport := capihttp.NewAuthRetryTransportForTest(http.DefaultTransport, nil)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+testPathApps, nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
}

// TestAuthRetryTransport_RewindableBodyRetries verifies that a POST with a
// rewindable body (as produced by capihttp.Client.Post for a JSON body)
// successfully replays after a 401.
func TestAuthRetryTransport_RewindableBodyRetries(t *testing.T) {
	t.Parallel()

	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		attemptCount := atomic.AddInt32(&attempts, 1)

		body, _ := io.ReadAll(request.Body)
		assert.Contains(t, string(body), `"name":"test-app"`,
			"body must be replayed verbatim on retry")

		if attemptCount == 1 {
			writer.WriteHeader(http.StatusUnauthorized)

			return
		}

		assert.Equal(t, "Bearer refreshed", request.Header.Get("Authorization"))
		writer.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	tokenManager := &refreshingTokenManager{
		initialToken:   testTokenInitial,
		refreshedToken: testTokenRefreshed,
	}
	client := capihttp.NewClient(server.URL, tokenManager,
		capihttp.WithRetryConfig(0, time.Millisecond, time.Millisecond))

	resp, err := client.Post(context.Background(), testPathApps, map[string]string{testFieldName: testAppNameValue})
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))
	assert.Equal(t, int32(1), atomic.LoadInt32(&tokenManager.refreshCalls))
}

// serializingTokenManager is a token manager whose RefreshToken publishes
// the next token atomically and whose GetToken returns whatever token is
// currently published. It exists so TestAuthRetryTransport_ConcurrentRefresh
// can observe the exact number of upstream refresh calls and the exact
// tokens returned from GetToken across a concurrent fleet of 401-retries.
type serializingTokenManager struct {
	mu           sync.Mutex
	current      string
	refreshDelay time.Duration

	refreshCalls int32
	getCalls     int32
}

func (m *serializingTokenManager) GetToken(_ context.Context) (string, error) {
	atomic.AddInt32(&m.getCalls, 1)

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.current, nil
}

func (m *serializingTokenManager) RefreshToken(_ context.Context) error {
	atomic.AddInt32(&m.refreshCalls, 1)

	// Simulate upstream latency so concurrent callers pile up while the
	// refresh is in flight. Without this delay the refresh window is too
	// short to race.
	if m.refreshDelay > 0 {
		time.Sleep(m.refreshDelay)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.current = "refreshed-" + m.current

	return nil
}

func (m *serializingTokenManager) SetToken(_ string, _ time.Time) {}

// TestAuthRetryTransport_ConcurrentRefresh is the regression test for Wave
// 6 R6-3 H1: when N goroutines simultaneously receive a 401 they must
// collectively trigger exactly ONE upstream RefreshToken call, not N. The
// N-1 goroutines that lose the refresh race must observe the token published
// by the winner and replay with it without a second upstream refresh.
func TestAuthRetryTransport_ConcurrentRefresh(t *testing.T) {
	t.Parallel()

	const concurrency = 10

	var (
		attemptsTotal     int32
		successfulReplays int32
	)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&attemptsTotal, 1)

		// Any request carrying the INITIAL token gets a 401. Any request
		// carrying the refreshed token succeeds. This lets us assert that
		// every caller eventually replays with the refreshed token.
		auth := request.Header.Get("Authorization")
		if auth == "Bearer initial" {
			writer.WriteHeader(http.StatusUnauthorized)

			return
		}

		if auth == "Bearer refreshed-initial" {
			atomic.AddInt32(&successfulReplays, 1)
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`{"ok":true}`))

			return
		}

		t.Errorf("unexpected Authorization header: %q", auth)
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tokenManager := &serializingTokenManager{
		current:      testTokenInitial,
		refreshDelay: 50 * time.Millisecond,
	}

	client := capihttp.NewClient(server.URL, tokenManager,
		capihttp.WithRetryConfig(0, time.Millisecond, time.Millisecond))

	var waitGroup sync.WaitGroup

	waitGroup.Add(concurrency)

	errs := make([]error, concurrency)
	statuses := make([]int, concurrency)

	for iteration := range concurrency {
		go func() {
			defer waitGroup.Done()

			resp, err := client.Get(context.Background(), testPathApps, nil)
			errs[iteration] = err

			if resp != nil {
				statuses[iteration] = resp.StatusCode
			}
		}()
	}

	waitGroup.Wait()

	for i, err := range errs {
		require.NoError(t, err, "goroutine %d", i)
		assert.Equal(t, http.StatusOK, statuses[i], "goroutine %d", i)
	}

	// The core assertion: exactly one upstream RefreshToken call despite N
	// concurrent 401s.
	assert.Equal(
		t,
		int32(1),
		atomic.LoadInt32(&tokenManager.refreshCalls),
		"RefreshToken must be called exactly once across %d concurrent 401s", concurrency,
	)

	// Every goroutine must have completed a successful replay with the
	// refreshed token. The server therefore sees N initial 401s plus N
	// refreshed-token successes, for a total of 2N attempts.
	assert.Equal(t, int32(concurrency), atomic.LoadInt32(&successfulReplays))
	assert.Equal(t, int32(2*concurrency), atomic.LoadInt32(&attemptsTotal))
}

// TestAuthRetryTransport_SecondUnauthorizedOnReplayReturnsToCaller is the
// regression test for Wave 6 R6-3 H2.2: when the replay itself returns a
// 401 the adapter must hand that second 401 back to the caller verbatim
// without issuing a third attempt (no infinite loop). This is the stated
// contract in authretry.go's package doc ("a second 401 on the retry is
// returned to the caller unchanged").
func TestAuthRetryTransport_SecondUnauthorizedOnReplayReturnsToCaller(t *testing.T) {
	t.Parallel()

	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&attempts, 1)

		// Always 401, regardless of how the Authorization header looks.
		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = writer.Write([]byte(`{"errors":[{"code":10002,"title":"CF-NotAuthenticated","detail":"still bad"}]}`))
	}))
	defer server.Close()

	tokenManager := &refreshingTokenManager{
		initialToken:   testTokenInitial,
		refreshedToken: testTokenRefreshed,
	}

	client := capihttp.NewClient(server.URL, tokenManager,
		capihttp.WithRetryConfig(0, time.Millisecond, time.Millisecond))

	resp, err := client.Get(context.Background(), testPathApps, nil)
	// The client surfaces the 401 as an error via its error mapper, but
	// the important contract here is that there is no infinite loop and
	// exactly TWO server attempts occurred (one original plus one replay).
	require.Error(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Contains(t, string(resp.Body), "CF-NotAuthenticated")

	assert.Equal(
		t,
		int32(2),
		atomic.LoadInt32(&attempts),
		"second 401 on replay must not trigger a third attempt",
	)
	assert.Equal(
		t,
		int32(1),
		atomic.LoadInt32(&tokenManager.refreshCalls),
		"RefreshToken must be called exactly once (no second refresh on second 401)",
	)
}

// TestAuthRetryTransport_RefreshEndpointReturns401 is the regression test
// for Wave 6 R6-3 H2.3: when the token refresh HTTP call itself returns
// 401 (UAA rejects the refresh), the token manager reports an error and
// the adapter must surface the original 401 to the caller without
// attempting any retry loop. This path exercises the real OAuth2 token
// manager against a fake UAA endpoint to cover the interaction between
// authretry and auth.OAuth2TokenManager together.
func TestAuthRetryTransport_RefreshEndpointReturns401(t *testing.T) {
	t.Parallel()

	var (
		uaaCalls int32
		apiCalls int32
	)

	// Fake UAA token endpoint that always 401s the refresh attempt.
	uaa := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&uaaCalls, 1)
		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = writer.Write([]byte(`{"error":"invalid_client","error_description":"client not trusted"}`))
	}))
	defer uaa.Close()

	// Target API that always 401s (so the retry path is exercised but the
	// refresh attempt never succeeds).
	api := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&apiCalls, 1)
		writer.WriteHeader(http.StatusUnauthorized)
	}))
	defer api.Close()

	tokenManager := auth.NewOAuth2TokenManager(&auth.OAuth2Config{
		TokenURL:     uaa.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		AccessToken:  "seed-access-token",
	})

	client := capihttp.NewClient(api.URL, tokenManager,
		capihttp.WithRetryConfig(0, time.Millisecond, time.Millisecond))

	resp, err := client.Get(context.Background(), testPathApps, nil)
	require.Error(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Exactly one API call: the initial 401. The refresh attempt failed
	// (UAA returned 401) so the adapter must NOT replay the API request.
	assert.Equal(
		t,
		int32(1),
		atomic.LoadInt32(&apiCalls),
		"API must see exactly one attempt when UAA refresh fails",
	)
	// Exactly one UAA call: the single refresh attempt that failed.
	// Anything more would indicate an infinite loop.
	assert.Equal(
		t,
		int32(1),
		atomic.LoadInt32(&uaaCalls),
		"UAA must be called at most once; more indicates an infinite loop",
	)
}
