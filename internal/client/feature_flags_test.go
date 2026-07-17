package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/fivetwenty-io/capi/v3/internal/client"
	internalhttp "github.com/fivetwenty-io/capi/v3/internal/http"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFeatureFlagPath is the path for the "my_feature_flag" fixture flag.
const testFeatureFlagPath = "/v3/feature_flags/my_feature_flag"

func TestFeatureFlagsClient_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testFeatureFlagPath, request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		now := time.Now()
		customError := "error message the user sees"
		featureFlag := capi.FeatureFlag{
			Name:               testFeatureFlagNameFixture,
			Enabled:            true,
			UpdatedAt:          &now,
			CustomErrorMessage: &customError,
			Links: capi.Links{
				testSelfKey: capi.Link{
					Href: testFeatureFlagPath,
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(featureFlag)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	featureFlags := NewFeatureFlagsClient(httpClient)

	featureFlag, err := featureFlags.Get(context.Background(), testFeatureFlagNameFixture)
	require.NoError(t, err)
	assert.NotNil(t, featureFlag)
	assert.Equal(t, testFeatureFlagNameFixture, featureFlag.Name)
	assert.True(t, featureFlag.Enabled)
	assert.NotNil(t, featureFlag.UpdatedAt)
	assert.NotNil(t, featureFlag.CustomErrorMessage)
	assert.Equal(t, "error message the user sees", *featureFlag.CustomErrorMessage)
}

func TestFeatureFlagsClient_Get_NotConfigured(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/feature_flags/app_scaling", request.URL.Path)
		assert.Equal(t, "GET", request.Method)

		featureFlag := capi.FeatureFlag{
			Name:               "app_scaling",
			Enabled:            true,
			UpdatedAt:          nil, // Not configured flags have null updated_at
			CustomErrorMessage: nil, // Not configured flags have null custom_error_message
			Links: capi.Links{
				testSelfKey: capi.Link{
					Href: "/v3/feature_flags/app_scaling",
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(featureFlag)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	featureFlags := NewFeatureFlagsClient(httpClient)

	featureFlag, err := featureFlags.Get(context.Background(), "app_scaling")
	require.NoError(t, err)
	assert.NotNil(t, featureFlag)
	assert.Equal(t, "app_scaling", featureFlag.Name)
	assert.True(t, featureFlag.Enabled)
	assert.Nil(t, featureFlag.UpdatedAt)
	assert.Nil(t, featureFlag.CustomErrorMessage)
}

//nolint:funlen // Test functions can be longer for comprehensive testing
func TestFeatureFlagsClient_List(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/feature_flags", request.URL.Path)
		assert.Equal(t, "GET", request.Method)
		assert.Equal(t, "2", request.URL.Query().Get("per_page"))
		assert.Equal(t, "1", request.URL.Query().Get("page"))

		now := time.Now()
		customError := "error message the user sees"
		response := capi.ListResponse[capi.FeatureFlag]{
			Pagination: capi.Pagination{
				TotalResults: 3,
				TotalPages:   2,
				First:        capi.Link{Href: "/v3/feature_flags?page=1&per_page=2"},
				Last:         capi.Link{Href: "/v3/feature_flags?page=2&per_page=2"},
				Next:         &capi.Link{Href: "/v3/feature_flags?page=2&per_page=2"},
			},
			Resources: []capi.FeatureFlag{
				{
					Name:               testFeatureFlagNameFixture,
					Enabled:            true,
					UpdatedAt:          &now,
					CustomErrorMessage: &customError,
					Links: capi.Links{
						testSelfKey: capi.Link{
							Href: testFeatureFlagPath,
						},
					},
				},
				{
					Name:               "my_second_feature_flag",
					Enabled:            false,
					UpdatedAt:          nil,
					CustomErrorMessage: nil,
					Links: capi.Links{
						testSelfKey: capi.Link{
							Href: "/v3/feature_flags/my_second_feature_flag",
						},
					},
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	featureFlags := NewFeatureFlagsClient(httpClient)

	params := &capi.QueryParams{
		Page:    1,
		PerPage: 2,
	}

	list, err := featureFlags.List(context.Background(), params)
	require.NoError(t, err)
	assert.NotNil(t, list)
	assert.Equal(t, 3, list.Pagination.TotalResults)
	assert.Equal(t, 2, list.Pagination.TotalPages)
	assert.Len(t, list.Resources, 2)
	assert.Equal(t, testFeatureFlagNameFixture, list.Resources[0].Name)
	assert.True(t, list.Resources[0].Enabled)
	assert.Equal(t, "my_second_feature_flag", list.Resources[1].Name)
	assert.False(t, list.Resources[1].Enabled)
}

func TestFeatureFlagsClient_Update(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, testFeatureFlagPath, request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var requestBody capi.FeatureFlagUpdateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.False(t, requestBody.Enabled)
		assert.NotNil(t, requestBody.CustomErrorMessage)
		assert.Equal(t, "custom error", *requestBody.CustomErrorMessage)

		now := time.Now()
		featureFlag := capi.FeatureFlag{
			Name:               testFeatureFlagNameFixture,
			Enabled:            requestBody.Enabled,
			UpdatedAt:          &now,
			CustomErrorMessage: requestBody.CustomErrorMessage,
			Links: capi.Links{
				testSelfKey: capi.Link{
					Href: testFeatureFlagPath,
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(featureFlag)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	featureFlags := NewFeatureFlagsClient(httpClient)

	customError := "custom error"
	request := &capi.FeatureFlagUpdateRequest{
		Enabled:            false,
		CustomErrorMessage: &customError,
	}

	featureFlag, err := featureFlags.Update(context.Background(), testFeatureFlagNameFixture, request)
	require.NoError(t, err)
	assert.NotNil(t, featureFlag)
	assert.Equal(t, testFeatureFlagNameFixture, featureFlag.Name)
	assert.False(t, featureFlag.Enabled)
	assert.NotNil(t, featureFlag.UpdatedAt)
	assert.NotNil(t, featureFlag.CustomErrorMessage)
	assert.Equal(t, "custom error", *featureFlag.CustomErrorMessage)
}

func TestFeatureFlagsClient_Update_EnableOnly(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v3/feature_flags/diego_docker", request.URL.Path)
		assert.Equal(t, "PATCH", request.Method)

		var requestBody capi.FeatureFlagUpdateRequest

		err := json.NewDecoder(request.Body).Decode(&requestBody)
		assert.NoError(t, err)

		assert.True(t, requestBody.Enabled)
		assert.Nil(t, requestBody.CustomErrorMessage)

		now := time.Now()
		featureFlag := capi.FeatureFlag{
			Name:               "diego_docker",
			Enabled:            requestBody.Enabled,
			UpdatedAt:          &now,
			CustomErrorMessage: nil,
			Links: capi.Links{
				testSelfKey: capi.Link{
					Href: "/v3/feature_flags/diego_docker",
				},
			},
		}

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(featureFlag)
	}))
	defer server.Close()

	httpClient := internalhttp.NewClient(server.URL, nil)
	featureFlags := NewFeatureFlagsClient(httpClient)

	request := &capi.FeatureFlagUpdateRequest{
		Enabled: true,
	}

	featureFlag, err := featureFlags.Update(context.Background(), "diego_docker", request)
	require.NoError(t, err)
	assert.NotNil(t, featureFlag)
	assert.Equal(t, "diego_docker", featureFlag.Name)
	assert.True(t, featureFlag.Enabled)
	assert.NotNil(t, featureFlag.UpdatedAt)
	assert.Nil(t, featureFlag.CustomErrorMessage)
}
