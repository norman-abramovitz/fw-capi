package auth_test

import (
	"testing"
	"time"

	"github.com/fivetwenty-io/capi/v3/internal/auth"
	"github.com/stretchr/testify/assert"
)

// Test constants shared across auth_test package files (oauth_test.go and
// token_test.go are both part of the same external test package).
const (
	testAccessToken     = "test-token"
	testTokenTypeBearer = "bearer"
)

func TestToken_Valid(t *testing.T) {
	t.Parallel()

	tests := getTokenValidityTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.token.Valid())
		})
	}
}

func getTokenValidityTestCases() []struct {
	name     string
	token    *auth.Token
	expected bool
} {
	return []struct {
		name     string
		token    *auth.Token
		expected bool
	}{
		{
			name:     "nil token",
			token:    nil,
			expected: false,
		},
		{
			name: "empty access token",
			token: &auth.Token{
				AccessToken: "",
			},
			expected: false,
		},
		{
			name: "valid token without expiry",
			token: &auth.Token{
				AccessToken: testAccessToken,
			},
			expected: true,
		},
		{
			name: "valid token with future expiry",
			token: &auth.Token{
				AccessToken: testAccessToken,
				ExpiresAt:   time.Now().Add(1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "expired token",
			token: &auth.Token{
				AccessToken: testAccessToken,
				ExpiresAt:   time.Now().Add(-1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "token expiring within buffer",
			token: &auth.Token{
				AccessToken: testAccessToken,
				ExpiresAt:   time.Now().Add(15 * time.Second),
			},
			expected: false, // Should be false due to 30 second buffer
		},
		{
			name: "token expiring just outside buffer",
			token: &auth.Token{
				AccessToken: testAccessToken,
				ExpiresAt:   time.Now().Add(35 * time.Second),
			},
			expected: true,
		},
	}
}

func TestTokenStore(t *testing.T) {
	t.Parallel()
	t.Run("new store is empty", testNewStoreEmpty)
	t.Run("set and get token", testSetAndGetToken)
	t.Run("clear token", testClearToken)
	t.Run("concurrent access", testConcurrentTokenAccess)
}

func testNewStoreEmpty(t *testing.T) {
	t.Parallel()

	store := auth.NewTokenStore()
	assert.Nil(t, store.Get())
}

func testSetAndGetToken(t *testing.T) {
	t.Parallel()

	store := auth.NewTokenStore()
	token := &auth.Token{
		AccessToken: testAccessToken,
		TokenType:   testTokenTypeBearer,
	}

	store.Set(token)
	retrieved := store.Get()
	assert.NotNil(t, retrieved)
	assert.Equal(t, token.AccessToken, retrieved.AccessToken)
	assert.Equal(t, token.TokenType, retrieved.TokenType)
}

func testClearToken(t *testing.T) {
	t.Parallel()

	store := auth.NewTokenStore()
	token := &auth.Token{
		AccessToken: testAccessToken,
	}

	store.Set(token)
	assert.NotNil(t, store.Get())

	store.Clear()
	assert.Nil(t, store.Get())
}

func testConcurrentTokenAccess(t *testing.T) {
	t.Parallel()

	store := auth.NewTokenStore()
	done := make(chan bool)

	// Start concurrent goroutines
	startTokenSetters(store, done)
	startTokenGetters(store, done)

	// Wait for all goroutines
	for range 4 {
		<-done
	}

	// Should not panic and should have a token
	finalToken := store.Get()
	assert.NotNil(t, finalToken)
	assert.True(t, finalToken.AccessToken == "token-1" || finalToken.AccessToken == "token-2")
}

func startTokenSetters(store *auth.TokenStore, done chan bool) {
	// Multiple goroutines setting tokens
	go func() {
		for range 100 {
			store.Set(&auth.Token{
				AccessToken: "token-1",
			})
		}

		done <- true
	}()

	go func() {
		for range 100 {
			store.Set(&auth.Token{
				AccessToken: "token-2",
			})
		}

		done <- true
	}()
}

func startTokenGetters(store *auth.TokenStore, done chan bool) {
	// Multiple goroutines getting tokens
	go func() {
		for range 100 {
			_ = store.Get()
		}

		done <- true
	}()

	go func() {
		for range 100 {
			_ = store.Get()
		}

		done <- true
	}()
}
