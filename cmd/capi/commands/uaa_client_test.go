//nolint:testpackage // Need access to internal types
package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixture token values used to assert UAA-token-over-CF-token precedence.
const (
	testUAATokenValue = "uaa-token"
	testCFTokenValue  = "cf-token"
)

func TestUAAClientWrapper_IsAuthenticated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config *Config
		want   bool
	}{
		{
			name: "with UAA token",
			config: &Config{
				UAAToken: "test-token",
			},
			want: true,
		},
		{
			name: "with CF token",
			config: &Config{
				Token: "test-token",
			},
			want: true,
		},
		{
			name: "with both tokens",
			config: &Config{
				UAAToken: testUAATokenValue,
				Token:    testCFTokenValue,
			},
			want: true,
		},
		{
			name:   "no authentication",
			config: &Config{},
			want:   false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			wrapper := &UAAClientWrapper{
				config: testCase.config,
			}
			assert.Equal(t, testCase.want, wrapper.IsAuthenticated())
		})
	}
}

func TestUAAClientWrapper_GetToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name: "UAA token takes precedence",
			config: &Config{
				UAAToken: testUAATokenValue,
				Token:    testCFTokenValue,
			},
			expected: testUAATokenValue,
		},
		{
			name: "fallback to CF token",
			config: &Config{
				Token: testCFTokenValue,
			},
			expected: testCFTokenValue,
		},
		{
			name:     "no tokens",
			config:   &Config{},
			expected: "",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			wrapper := &UAAClientWrapper{
				config: testCase.config,
			}
			assert.Equal(t, testCase.expected, wrapper.GetToken())
		})
	}
}

func TestUAAClientWrapper_SetToken(t *testing.T) {
	t.Parallel()

	config := &Config{}
	wrapper := &UAAClientWrapper{
		config: config,
	}

	wrapper.SetToken("new-token")
	assert.Equal(t, "new-token", config.UAAToken)
}

func TestUAAClientWrapper_InferUAAEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cfAPIURL    string
		expectedURL string
	}{
		{
			name:        "standard CF API URL",
			cfAPIURL:    "https://api.cf.example.com",
			expectedURL: "https://uaa.cf.example.com",
		},
		{
			name:        "API with subdomain",
			cfAPIURL:    "https://api.system.cf.example.com",
			expectedURL: "https://uaa.system.cf.example.com",
		},
		{
			name:        "localhost development",
			cfAPIURL:    "https://api.bosh-lite.com",
			expectedURL: "https://uaa.bosh-lite.com",
		},
		{
			name:        "empty URL",
			cfAPIURL:    "",
			expectedURL: "",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			wrapper := &UAAClientWrapper{
				config: &Config{
					API: testCase.cfAPIURL,
				},
			}
			url := wrapper.inferUAAEndpointFromAPI(testCase.cfAPIURL)
			assert.Equal(t, testCase.expectedURL, url)
		})
	}
}

func TestUAAClientWrapper_Endpoint(t *testing.T) {
	t.Parallel()

	wrapper := &UAAClientWrapper{
		endpoint: "https://uaa.example.com",
	}
	assert.Equal(t, "https://uaa.example.com", wrapper.Endpoint())
}

func TestNewUAAClient_NoEndpoint(t *testing.T) {
	t.Parallel()

	config := &Config{
		// No UAA endpoint or CF API endpoint
	}

	client, err := NewUAAClient(config)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "no UAA endpoint configured")
}
