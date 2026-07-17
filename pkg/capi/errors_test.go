package capi_test

import (
	"encoding/json"
	"testing"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIError_Error(t *testing.T) {
	t.Parallel()

	err := &capi.APIError{
		Code:   10010,
		Title:  testNotFoundTitle,
		Detail: testNotFoundDetail,
	}

	assert.Equal(t, "CF-ResourceNotFound: App not found (code: 10010)", err.Error())
}

func TestResponseError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response *capi.ResponseError
		expected string
	}{
		{
			name:     "empty errors",
			response: &capi.ResponseError{},
			expected: "unknown error",
		},
		{
			name: "single error",
			response: &capi.ResponseError{
				Errors: []capi.APIError{
					{
						Code:   10010,
						Title:  testNotFoundTitle,
						Detail: testNotFoundDetail,
					},
				},
			},
			expected: "CF-ResourceNotFound: App not found (code: 10010)",
		},
		{
			name: "multiple errors",
			response: &capi.ResponseError{
				Errors: []capi.APIError{
					{
						Code:   10010,
						Title:  testNotFoundTitle,
						Detail: testNotFoundDetail,
					},
					{
						Code:   10008,
						Title:  "CF-UnprocessableEntity",
						Detail: "Invalid request",
					},
				},
			},
			expected: "multiple errors: [{10010 CF-ResourceNotFound App not found} {10008 CF-UnprocessableEntity Invalid request}]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.response.Error())
		})
	}
}

func TestResponseError_FirstError(t *testing.T) {
	t.Parallel()
	t.Run("with errors", func(t *testing.T) {
		t.Parallel()

		response := &capi.ResponseError{
			Errors: []capi.APIError{
				{Code: 10010, Title: testNotFoundTitle, Detail: "Not found"},
				{Code: 10008, Title: "CF-UnprocessableEntity", Detail: "Invalid"},
			},
		}

		first := response.FirstError()
		require.NotNil(t, first)
		assert.Equal(t, 10010, first.Code)
		assert.Equal(t, testNotFoundTitle, first.Title)
	})

	t.Run("without errors", func(t *testing.T) {
		t.Parallel()

		response := &capi.ResponseError{}
		assert.Nil(t, response.FirstError())
	})
}

func TestIsNotFound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "APIError not found",
			err:      &capi.APIError{Code: capi.ErrorCodeNotFound},
			expected: true,
		},
		{
			name:     "APIError resource not found",
			err:      &capi.APIError{Code: capi.ErrorCodeResourceNotFound},
			expected: true,
		},
		{
			name:     "APIError other error",
			err:      &capi.APIError{Code: capi.ErrorCodeNotAuthenticated},
			expected: false,
		},
		{
			name: "capi.ResponseError with not found",
			err: &capi.ResponseError{
				Errors: []capi.APIError{
					{Code: capi.ErrorCodeNotFound},
				},
			},
			expected: true,
		},
		{
			name: "capi.ResponseError without not found",
			err: &capi.ResponseError{
				Errors: []capi.APIError{
					{Code: capi.ErrorCodeNotAuthenticated},
				},
			},
			expected: false,
		},
		{
			name:     "capi.ResponseError empty",
			err:      &capi.ResponseError{},
			expected: false,
		},
		{
			name:     "other error type",
			err:      capi.ErrSomeError,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, capi.IsNotFound(tt.err))
		})
	}
}

// runErrorCodeTests runs standardized tests for error code checking functions.
func runErrorCodeTests(t *testing.T, targetCode int, checkFunc func(error) bool) {
	t.Helper()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "APIError with target code",
			err:      &capi.APIError{Code: targetCode},
			expected: true,
		},
		{
			name:     "APIError other error",
			err:      &capi.APIError{Code: capi.ErrorCodeNotFound},
			expected: targetCode == capi.ErrorCodeNotFound,
		},
		{
			name: "ResponseError with target code",
			err: &capi.ResponseError{
				Errors: []capi.APIError{
					{Code: targetCode},
				},
			},
			expected: true,
		},
		{
			name: "ResponseError without target code",
			err: &capi.ResponseError{
				Errors: []capi.APIError{
					{Code: capi.ErrorCodeNotFound},
				},
			},
			expected: targetCode == capi.ErrorCodeNotFound,
		},
		{
			name:     "other error type",
			err:      capi.ErrSomeError,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, checkFunc(tt.err))
		})
	}
}

func TestIsUnauthorized(t *testing.T) {
	t.Parallel()
	runErrorCodeTests(t, capi.ErrorCodeNotAuthenticated, capi.IsUnauthorized)
}

func TestIsForbidden(t *testing.T) {
	t.Parallel()
	runErrorCodeTests(t, capi.ErrorCodeNotAuthorized, capi.IsForbidden)
}

func TestParseResponseError(t *testing.T) {
	t.Parallel()
	t.Run("valid error response", func(t *testing.T) {
		t.Parallel()

		jsonData := `{
			"errors": [
				{
					"code": 10010,
					"title": "CF-ResourceNotFound",
					"detail": "App not found"
				},
				{
					"code": 10008,
					"title": "CF-UnprocessableEntity",
					"detail": "Invalid request"
				}
			]
		}`

		errResp, err := capi.ParseResponseError([]byte(jsonData))
		require.NoError(t, err)
		require.NotNil(t, errResp)
		assert.Len(t, errResp.Errors, 2)
		assert.Equal(t, 10010, errResp.Errors[0].Code)
		assert.Equal(t, testNotFoundTitle, errResp.Errors[0].Title)
		assert.Equal(t, testNotFoundDetail, errResp.Errors[0].Detail)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()

		jsonData := `{invalid json}`

		errResp, err := capi.ParseResponseError([]byte(jsonData))
		require.Error(t, err)
		assert.Nil(t, errResp)
	})

	t.Run("empty response", func(t *testing.T) {
		t.Parallel()

		jsonData := `{"errors": []}`

		errResp, err := capi.ParseResponseError([]byte(jsonData))
		require.NoError(t, err)
		require.NotNil(t, errResp)
		assert.Empty(t, errResp.Errors)
	})
}

func TestErrorConstants(t *testing.T) {
	t.Parallel()

	// CF-API error templates (keyed by CF error code) used as prototypes
	// when constructing synthetic APIError values.
	assert.Equal(t, 10015, capi.ErrCFServiceUnavailable.Code)
	assert.Equal(t, "CF-ServiceUnavailable", capi.ErrCFServiceUnavailable.Title)

	assert.Equal(t, 10005, capi.ErrCFBadRequest.Code)
	assert.Equal(t, "CF-BadRequest", capi.ErrCFBadRequest.Title)

	assert.Equal(t, 10013, capi.ErrCFTooManyRequests.Code)
	assert.Equal(t, "CF-TooManyRequests", capi.ErrCFTooManyRequests.Title)
}

func TestResponseError_JSONMarshaling(t *testing.T) {
	t.Parallel()

	errResp := &capi.ResponseError{
		Errors: []capi.APIError{
			{
				Code:   10010,
				Title:  testNotFoundTitle,
				Detail: "The app could not be found: test-app",
			},
		},
	}

	data, err := json.Marshal(errResp)
	require.NoError(t, err)

	var decoded capi.ResponseError

	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Errors, 1)
	assert.Equal(t, errResp.Errors[0].Code, decoded.Errors[0].Code)
	assert.Equal(t, errResp.Errors[0].Title, decoded.Errors[0].Title)
	assert.Equal(t, errResp.Errors[0].Detail, decoded.Errors[0].Detail)
}

// TestErrorCodeValues asserts each CF v3 error-code constant carries the
// correct numeric value as defined in cloud_controller_ng errors/v2.yml.
func TestErrorCodeValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		got      int
		expected int
	}{
		{"ErrorCodeNotFound", capi.ErrorCodeNotFound, 10000},
		{"ErrorCodeNotAuthenticated", capi.ErrorCodeNotAuthenticated, 10002},
		{"ErrorCodeNotAuthorized", capi.ErrorCodeNotAuthorized, 10003},
		{"ErrorCodeBadRequest", capi.ErrorCodeBadRequest, 10005},
		{"ErrorCodeUnprocessableEntity", capi.ErrorCodeUnprocessableEntity, 10008},
		{"ErrorCodeResourceNotFound", capi.ErrorCodeResourceNotFound, 10010},
		{"ErrorCodeTooManyRequests", capi.ErrorCodeTooManyRequests, 10013},
		{"ErrorCodeServiceUnavailable", capi.ErrorCodeServiceUnavailable, 10015},
		{"ErrorCodeInvalidRelation", capi.ErrorCodeInvalidRelation, 1002},
		{"ErrorCodeMaintenanceInfo", capi.ErrorCodeMaintenanceInfo, 390006},
		{"ErrorCodeServiceInstanceQuota", capi.ErrorCodeServiceInstanceQuota, 60005},
		{"ErrorCodeAsyncServiceInProgress", capi.ErrorCodeAsyncServiceInProgress, 60016},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.got)
		})
	}
}
