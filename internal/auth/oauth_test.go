package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fivetwenty-io/capi/v3/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOAuth2TokenManager_GetToken(t *testing.T) {
	t.Parallel()
	t.Run("returns existing valid token", testExistingValidToken)
	t.Run("refreshes expired token using refresh token", testRefreshExpiredToken)
	t.Run("uses client credentials when no refresh token", testClientCredentials)
	t.Run("uses password grant", testPasswordGrant)
	t.Run("handles token request error", testTokenRequestError)
	t.Run("no credentials available", testNoCredentials)
}

func testExistingValidToken(t *testing.T) {
	t.Parallel()

	manager := auth.NewOAuth2TokenManager(&auth.OAuth2Config{
		AccessToken: "existing-token",
	})

	token, err := manager.GetToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "existing-token", token)
}

func testRefreshExpiredToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/oauth/token", request.URL.Path)
		assert.Equal(t, "POST", request.Method)

		err := request.ParseForm()
		assert.NoError(t, err)
		assert.Equal(t, "refresh_token", request.Form.Get("grant_type"))
		assert.Equal(t, "old-refresh-token", request.Form.Get("refresh_token"))

		response := auth.Token{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			ExpiresIn:    3600,
			TokenType:    testTokenTypeBearer,
		}
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	manager := auth.NewOAuth2TokenManager(&auth.OAuth2Config{
		TokenURL:     server.URL + "/oauth/token",
		RefreshToken: "old-refresh-token",
	})

	// Set expired token
	manager.SetToken("expired-token", time.Now().Add(-1*time.Hour))

	token, err := manager.GetToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", token)
}

func testClientCredentials(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/oauth/token", request.URL.Path)
		assert.Equal(t, "POST", request.Method)

		// Check basic auth
		username, password, ok := request.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "client-id", username)
		assert.Equal(t, "client-secret", password)

		err := request.ParseForm()
		assert.NoError(t, err)
		assert.Equal(t, "client_credentials", request.Form.Get("grant_type"))

		response := auth.Token{
			AccessToken: "client-token",
			ExpiresIn:   3600,
			TokenType:   testTokenTypeBearer,
		}
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	manager := auth.NewOAuth2TokenManager(&auth.OAuth2Config{
		TokenURL:     server.URL + "/oauth/token",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	})

	token, err := manager.GetToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "client-token", token)
}

func testPasswordGrant(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/oauth/token", request.URL.Path)
		assert.Equal(t, "POST", request.Method)

		err := request.ParseForm()
		assert.NoError(t, err)
		assert.Equal(t, "password", request.Form.Get("grant_type"))
		assert.Equal(t, "testuser", request.Form.Get("username"))
		assert.Equal(t, "testpass", request.Form.Get("password"))

		response := auth.Token{
			AccessToken:  "password-token",
			RefreshToken: "refresh-token",
			ExpiresIn:    3600,
			TokenType:    testTokenTypeBearer,
		}
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	manager := auth.NewOAuth2TokenManager(&auth.OAuth2Config{
		TokenURL: server.URL + "/oauth/token",
		Username: "testuser",
		Password: "testpass",
	})

	token, err := manager.GetToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "password-token", token)
}

func testTokenRequestError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)

		response := map[string]string{
			"error":             "invalid_client",
			"error_description": "Client authentication failed",
		}
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	manager := auth.NewOAuth2TokenManager(&auth.OAuth2Config{
		TokenURL:     server.URL + "/oauth/token",
		ClientID:     "bad-client",
		ClientSecret: "bad-secret",
	})

	token, err := manager.GetToken(context.Background())
	require.ErrorContains(t, err, "invalid_client")
	require.ErrorContains(t, err, "Client authentication failed")
	assert.Empty(t, token)
}

func testNoCredentials(t *testing.T) {
	t.Parallel()

	manager := auth.NewOAuth2TokenManager(&auth.OAuth2Config{
		TokenURL: "http://example.com/oauth/token",
	})

	token, err := manager.GetToken(context.Background())
	require.ErrorIs(t, err, auth.ErrNoValidCredentials)
	assert.Empty(t, token)
}

func TestOAuth2TokenManager_SetToken(t *testing.T) {
	t.Parallel()

	manager := auth.NewOAuth2TokenManager(&auth.OAuth2Config{})

	expiresAt := time.Now().Add(1 * time.Hour)
	manager.SetToken("manual-token", expiresAt)

	token, err := manager.GetToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "manual-token", token)

	// Note: Internal token storage details are not verified since store field is unexported
}

func TestOAuth2TokenManager_RefreshToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		response := auth.Token{
			AccessToken: "refreshed-token",
			ExpiresIn:   3600,
			TokenType:   testTokenTypeBearer,
		}
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	manager := auth.NewOAuth2TokenManager(&auth.OAuth2Config{
		TokenURL:     server.URL + "/oauth/token",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	})

	// Set a valid token
	manager.SetToken("current-token", time.Now().Add(1*time.Hour))

	// Force refresh
	err := manager.RefreshToken(context.Background())
	require.NoError(t, err)

	// Should have new token
	token, err := manager.GetToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "refreshed-token", token)
}

func TestNewUAATokenManager(t *testing.T) {
	t.Parallel()
	t.Run("creates manager with correct token URL", func(t *testing.T) {
		t.Parallel()

		manager := auth.NewUAATokenManager("https://uaa.example.com", "client-id", "client-secret")
		assert.NotNil(t, manager)
		// Note: Internal config details are not verified since config field is unexported
	})

	t.Run("handles trailing slash in UAA URL", func(t *testing.T) {
		t.Parallel()

		manager := auth.NewUAATokenManager("https://uaa.example.com/", "client-id", "client-secret")
		assert.NotNil(t, manager)
		// Note: Internal config details are not verified since config field is unexported
	})
}

func TestNewUAATokenManagerWithPassword(t *testing.T) {
	t.Parallel()

	manager := auth.NewUAATokenManagerWithPassword(
		"https://uaa.example.com",
		"client-id",
		"client-secret",
		"username",
		"password",
	)

	assert.NotNil(t, manager)
	// Note: Internal config details are not verified since config field is unexported
}
