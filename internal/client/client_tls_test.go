package client_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fivetwenty-io/capi/v3/internal/client"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
)

func newSelfSignedCF(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token", "token_type": "bearer", "expires_in": 3600,
		})
	})
	mux.HandleFunc("/v3/organizations", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"pagination": map[string]interface{}{"total_results": 0, "total_pages": 1},
			"resources":  []interface{}{},
		})
	})

	server := httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)

	return server
}

func selfSignedConfig(serverURL string, skipTLS bool) *capi.Config {
	return &capi.Config{
		APIEndpoint:   serverURL,
		ClientID:      "test-client",
		ClientSecret:  "test-secret",
		TokenURL:      serverURL + "/oauth/token",
		SkipTLSVerify: skipTLS,
	}
}

func serverCAPEM(t *testing.T, server *httptest.Server) string {
	t.Helper()

	cert := server.Certificate()
	require.NotNil(t, cert)

	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
}

// unrelatedCAPEM generates a CA that signed nothing the test servers use.
// (httptest.NewTLSServer instances all share one embedded certificate, so
// "another server's cert" would be the same CA.)
func unrelatedCAPEM(t *testing.T) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "unrelated-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func TestCACertPEM_VerifiesAgainstProvidedCA(t *testing.T) {
	server := newSelfSignedCF(t)
	t.Setenv("CAPI_DEV_MODE", "")

	config := selfSignedConfig(server.URL, false)
	config.CACertPEM = serverCAPEM(t, server)

	c, err := client.New(context.Background(), config)
	require.NoError(t, err)

	orgs, err := c.Organizations().List(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, orgs.Resources)
}

func TestCACertPEM_WrongCA_StillFails(t *testing.T) {
	server := newSelfSignedCF(t)
	t.Setenv("CAPI_DEV_MODE", "")

	config := selfSignedConfig(server.URL, false)
	config.CACertPEM = unrelatedCAPEM(t)

	c, err := client.New(context.Background(), config)
	require.NoError(t, err)

	_, err = c.Organizations().List(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "certificate")
}

func TestCACertPEM_TakesPrecedenceOverSkip(t *testing.T) {
	server := newSelfSignedCF(t)
	t.Setenv("CAPI_DEV_MODE", "true")

	// wrong CA plus SkipTLSVerify: the CA path must win, so verification
	// still fails rather than being silently skipped
	config := selfSignedConfig(server.URL, true)
	config.CACertPEM = unrelatedCAPEM(t)

	c, err := client.New(context.Background(), config)
	require.NoError(t, err)

	_, err = c.Organizations().List(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "certificate")
}

func TestSkipTLSVerify_DevMode_SkipsVerification(t *testing.T) {
	server := newSelfSignedCF(t)
	t.Setenv("CAPI_DEV_MODE", "true")

	c, err := client.New(context.Background(), selfSignedConfig(server.URL, true))
	require.NoError(t, err)

	// exercises both the token fetch (auth HTTP client) and the API call
	// (resource HTTP client) against the self-signed server
	orgs, err := c.Organizations().List(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, orgs.Resources)
}

func TestSkipTLSVerify_WithoutDevMode_StillVerifies(t *testing.T) {
	server := newSelfSignedCF(t)
	t.Setenv("CAPI_DEV_MODE", "")

	c, err := client.New(context.Background(), selfSignedConfig(server.URL, true))
	require.NoError(t, err)

	_, err = c.Organizations().List(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "certificate")
}

func TestSkipTLSVerify_False_Verifies(t *testing.T) {
	server := newSelfSignedCF(t)
	t.Setenv("CAPI_DEV_MODE", "true")

	c, err := client.New(context.Background(), selfSignedConfig(server.URL, false))
	require.NoError(t, err)

	_, err = c.Organizations().List(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "certificate")
}
