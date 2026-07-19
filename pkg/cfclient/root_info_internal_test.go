package cfclient

// This file lives in package cfclient (not cfclient_test) so it can reach
// the unexported fetchRootInfo helper used for UAA/login discovery from the
// CF API root (/) response.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFetchRootInfo_NullLinks proves the UAA/login discovery used by New()
// tolerates the root (/) response shape Cloud Controller returns since
// capi-release 1.241.0 (cloud_controller_ng PR 5117), where unconfigured
// links are null rather than omitted or an empty-href object,
// and that a root response with no usable UAA or login link produces the
// clear ErrNoUAAOrLoginURL error instead of panicking or silently
// succeeding with an empty token URL.
func TestFetchRootInfo_NullLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		wantURL    string
		wantErr    error
		wantErrMsg string
	}{
		{
			name:    "uaa link present",
			body:    `{"links":{"uaa":{"href":"https://uaa.example.org"},"login":null}}`,
			wantURL: "https://uaa.example.org",
		},
		{
			name:    "uaa null, login present",
			body:    `{"links":{"uaa":null,"login":{"href":"https://login.example.org"}}}`,
			wantURL: "https://login.example.org",
		},
		{
			name:       "both uaa and login null",
			body:       `{"links":{"uaa":null,"login":null}}`,
			wantErr:    capi.ErrNoUAAOrLoginURL,
			wantErrMsg: "no UAA or login URL",
		},
		{
			name:       "uaa and login absent entirely",
			body:       `{"links":{"self":{"href":"https://api.example.org"}}}`,
			wantErr:    capi.ErrNoUAAOrLoginURL,
			wantErrMsg: "no UAA or login URL",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, "/", request.URL.Path)
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(http.StatusOK)
				_, _ = writer.Write([]byte(testCase.body))
			}))
			defer server.Close()

			httpClient, err := createDiscoveryHTTPClient(false, "")
			require.NoError(t, err)

			uaaURL, err := fetchRootInfo(context.Background(), httpClient, server.URL)

			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
				require.ErrorContains(t, err, testCase.wantErrMsg)
				assert.Empty(t, uaaURL)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.wantURL, uaaURL)
		})
	}
}

// TestFetchRootInfo_NonOKStatus proves a non-200 root response produces a
// clear error rather than attempting to decode an error body as links.
func TestFetchRootInfo_NonOKStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = writer.Write([]byte("maintenance"))
	}))
	defer server.Close()

	httpClient, err := createDiscoveryHTTPClient(false, "")
	require.NoError(t, err)

	uaaURL, err := fetchRootInfo(context.Background(), httpClient, server.URL)
	require.ErrorIs(t, err, capi.ErrRootInfoRequestFailed)
	assert.Empty(t, uaaURL)
}
