package capi_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const wellFormedNotFoundEnvelope = `{
  "errors": [
    {
      "code": 10010,
      "title": "CF-ResourceNotFound",
      "detail": "App not found"
    }
  ]
}`

const wellFormedMultiErrorEnvelope = `{
  "errors": [
    {
      "code": 10008,
      "title": "CF-UnprocessableEntity",
      "detail": "name must be unique"
    },
    {
      "code": 10008,
      "title": "CF-UnprocessableEntity",
      "detail": "stack must be present"
    }
  ]
}`

const malformedBody = `<html><body>502 Bad Gateway</body></html>`

// TestMapHTTPError_BelowThresholdReturnsNil verifies that success and
// informational / redirect statuses do NOT produce an error value. This is
// the contract the internal HTTP client relies on to short-circuit the
// error path for 2xx/3xx responses.
func TestMapHTTPError_BelowThresholdReturnsNil(t *testing.T) {
	t.Parallel()

	successStatuses := []int{
		0, // pseudo-value for "no response yet"; MapHTTPError treats it as <400
		http.StatusOK,
		http.StatusCreated,
		http.StatusAccepted,
		http.StatusNoContent,
		http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusNotModified,
		399,
	}

	for _, status := range successStatuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			err := capi.MapHTTPError(status, []byte(wellFormedNotFoundEnvelope))
			assert.NoError(t, err, "status %d should produce nil error", status)
		})
	}
}

// sentinelCase describes a (status, sentinel) pair the mapper must honor.
type sentinelCase struct {
	name     string
	status   int
	sentinel error
}

func allSentinelCases() []sentinelCase {
	return []sentinelCase{
		{name: "404 -> ErrNotFound", status: http.StatusNotFound, sentinel: capi.ErrNotFound},
		{name: "401 -> ErrUnauthorized", status: http.StatusUnauthorized, sentinel: capi.ErrUnauthorized},
		{name: "403 -> ErrForbidden", status: http.StatusForbidden, sentinel: capi.ErrForbidden},
		{name: "409 -> ErrConflict", status: http.StatusConflict, sentinel: capi.ErrConflict},
		{name: "422 -> ErrUnprocessable", status: http.StatusUnprocessableEntity, sentinel: capi.ErrUnprocessable},
		{name: "429 -> ErrRateLimited", status: http.StatusTooManyRequests, sentinel: capi.ErrRateLimited},
		{name: "500 -> ErrServerError", status: http.StatusInternalServerError, sentinel: capi.ErrServerError},
		{name: "502 -> ErrServerError", status: http.StatusBadGateway, sentinel: capi.ErrServerError},
		{name: "503 -> ErrServerError", status: http.StatusServiceUnavailable, sentinel: capi.ErrServerError},
		{name: "504 -> ErrServerError", status: http.StatusGatewayTimeout, sentinel: capi.ErrServerError},
	}
}

// TestMapHTTPError_WellFormedBodyWrapsSentinel verifies that when the body
// is a valid CF v3 error envelope, the returned error unwraps to the
// correct sentinel via errors.Is AND unwraps to *ResponseError via
// errors.As, so existing callers that inspect APIError.Code / Title /
// Detail continue to work.
func TestMapHTTPError_WellFormedBodyWrapsSentinel(t *testing.T) {
	t.Parallel()

	for _, testCase := range allSentinelCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := capi.MapHTTPError(testCase.status, []byte(wellFormedNotFoundEnvelope))
			require.Error(t, err)

			require.ErrorIs(
				t,
				err, testCase.sentinel,
				"errors.Is(err, %v) must be true for status %d",
				testCase.sentinel, testCase.status,
			)

			var envelope *capi.ResponseError
			require.ErrorAs(t, err, &envelope,
				"errors.As(err, *ResponseError) must be true when body is well-formed")
			require.NotNil(t, envelope)
			require.Len(t, envelope.Errors, 1)
			assert.Equal(t, 10010, envelope.Errors[0].Code)
			assert.Equal(t, "CF-ResourceNotFound", envelope.Errors[0].Title)
			assert.Equal(t, "App not found", envelope.Errors[0].Detail)
		})
	}
}

// TestMapHTTPError_MalformedBodyStillWrapsSentinel verifies that when the
// server returns a body that is NOT a valid CF error envelope (HTML,
// truncated JSON, empty object, etc.), MapHTTPError still wraps the correct
// sentinel so that `errors.Is` checks keep working, and includes the raw
// body in the error message for debuggability.
func TestMapHTTPError_MalformedBodyStillWrapsSentinel(t *testing.T) {
	t.Parallel()

	malformedInputs := []struct {
		name string
		body []byte
	}{
		{name: "html body", body: []byte(malformedBody)},
		{name: "truncated json", body: []byte(`{"errors": [`)},
		{name: "empty errors array", body: []byte(`{"errors": []}`)},
		{name: "wrong shape", body: []byte(`{"message": "nope"}`)},
		{name: "raw string", body: []byte(`"plain string"`)},
	}

	for _, testCase := range allSentinelCases() {
		for _, input := range malformedInputs {
			t.Run(testCase.name+"/"+input.name, func(t *testing.T) {
				t.Parallel()

				err := capi.MapHTTPError(testCase.status, input.body)
				require.Error(t, err)

				require.ErrorIs(
					t,
					err, testCase.sentinel,
					"errors.Is(err, %v) must be true for status %d with malformed body",
					testCase.sentinel, testCase.status,
				)

				// For malformed bodies, MapHTTPError should NOT produce a
				// *ResponseError in the error chain (the body did not
				// parse as one).
				var envelope *capi.ResponseError
				assert.NotErrorAs(
					t,
					err, &envelope,
					"errors.As should not find a *ResponseError for malformed body %q", string(input.body),
				)

				// The raw body should be present in the error message for
				// human debugging.
				assert.Contains(t, err.Error(), string(input.body),
					"error message should include the raw body for debugging")
			})
		}
	}
}

// TestMapHTTPError_EmptyBody verifies MapHTTPError does not panic and still
// wraps the sentinel when given a nil or empty body. The error message
// includes the status code in place of a body.
func TestMapHTTPError_EmptyBody(t *testing.T) {
	t.Parallel()

	for _, testCase := range allSentinelCases() {
		t.Run(testCase.name+"/nil body", func(t *testing.T) {
			t.Parallel()

			err := capi.MapHTTPError(testCase.status, nil)
			require.Error(t, err)
			require.ErrorIs(t, err, testCase.sentinel)
			assert.Contains(t, err.Error(), "status")
		})
		t.Run(testCase.name+"/empty body", func(t *testing.T) {
			t.Parallel()

			err := capi.MapHTTPError(testCase.status, []byte{})
			require.Error(t, err)
			require.ErrorIs(t, err, testCase.sentinel)
			assert.Contains(t, err.Error(), "status")
		})
	}
}

// TestMapHTTPError_MultiErrorEnvelope verifies that a well-formed envelope
// containing multiple APIError entries is fully preserved in the error
// chain and that the returned error still unwraps to the correct sentinel.
func TestMapHTTPError_MultiErrorEnvelope(t *testing.T) {
	t.Parallel()

	err := capi.MapHTTPError(http.StatusUnprocessableEntity, []byte(wellFormedMultiErrorEnvelope))
	require.Error(t, err)

	require.ErrorIs(t, err, capi.ErrUnprocessable)

	var envelope *capi.ResponseError
	require.ErrorAs(t, err, &envelope)
	require.Len(t, envelope.Errors, 2)
	assert.Equal(t, "CF-UnprocessableEntity", envelope.Errors[0].Title)
	assert.Equal(t, "name must be unique", envelope.Errors[0].Detail)
	assert.Equal(t, "stack must be present", envelope.Errors[1].Detail)
}

// TestMapHTTPError_UnknownClient4xxMapsToBadRequest pins the contract that
// a 4xx status code the mapper does not explicitly enumerate (for example
// 400 Bad Request, 405 Method Not Allowed, 418 I'm a Teapot, or 431 Request
// Header Fields Too Large) classifies as ErrBadRequest — a stable sentinel
// with "client error, do not retry" semantics. This is the regression test
// for Wave 6 R6-3 H3: unknown 4xx must not be a non-sentinel fmt.Errorf
// because callers cannot reliably distinguish it with errors.Is. The test
// also verifies the unknown 4xx does NOT accidentally match any of the
// other named sentinels (404/401/403/409/422/429) or the 5xx sentinel.
func TestMapHTTPError_UnknownClient4xxMapsToBadRequest(t *testing.T) {
	t.Parallel()

	unknownStatuses := []int{
		http.StatusBadRequest,                   // 400
		http.StatusMethodNotAllowed,             // 405
		http.StatusNotAcceptable,                // 406
		http.StatusRequestTimeout,               // 408
		http.StatusGone,                         // 410
		http.StatusLengthRequired,               // 411
		http.StatusPreconditionFailed,           // 412
		http.StatusRequestEntityTooLarge,        // 413
		http.StatusRequestURITooLong,            // 414
		http.StatusUnsupportedMediaType,         // 415
		http.StatusRequestedRangeNotSatisfiable, // 416
		http.StatusExpectationFailed,            // 417
		http.StatusTeapot,                       // 418
		http.StatusMisdirectedRequest,           // 421
		http.StatusLocked,                       // 423
		http.StatusFailedDependency,             // 424
		http.StatusTooEarly,                     // 425
		http.StatusUpgradeRequired,              // 426
		http.StatusPreconditionRequired,         // 428
		http.StatusRequestHeaderFieldsTooLarge,  // 431
		http.StatusUnavailableForLegalReasons,   // 451
	}

	nonMatchingSentinels := []error{
		capi.ErrNotFound,
		capi.ErrUnauthorized,
		capi.ErrForbidden,
		capi.ErrConflict,
		capi.ErrUnprocessable,
		capi.ErrRateLimited,
		capi.ErrServerError,
	}

	for _, status := range unknownStatuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			err := capi.MapHTTPError(status, []byte(`{"errors":[]}`))
			require.Error(t, err)

			// Regression pin: unknown 4xx MUST be classified as ErrBadRequest
			// so callers can use errors.Is(err, capi.ErrBadRequest) to
			// detect "client error, do not retry" failures.
			require.ErrorIs(t, err, capi.ErrBadRequest,
				"status %d must unwrap to ErrBadRequest via errors.Is", status)

			for _, s := range nonMatchingSentinels {
				require.NotErrorIs(t, err, s,
					"unknown 4xx status %d must not match sentinel %v", status, s)
			}

			// The error message should still mention the status code for
			// human debuggability.
			assert.Contains(t, err.Error(),
				strconv.Itoa(status),
				"error message must include the raw status code %d", status)
		})
	}
}

// TestMapHTTPError_ErrBadRequestSentinelIdentity verifies ErrBadRequest is
// a distinct sentinel value that does not accidentally unwrap to any other
// sentinel in the package. This is the companion check to
// TestMapHTTPError_SentinelIdentity above and closes the regression surface
// opened by H3 (a non-sentinel fallthrough could be indistinguishable from
// a future sentinel).
func TestMapHTTPError_ErrBadRequestSentinelIdentity(t *testing.T) {
	t.Parallel()

	other := []error{
		capi.ErrNotFound,
		capi.ErrUnauthorized,
		capi.ErrForbidden,
		capi.ErrConflict,
		capi.ErrUnprocessable,
		capi.ErrRateLimited,
		capi.ErrServerError,
	}

	require.Error(t, capi.ErrBadRequest)
	assert.True(t, strings.HasPrefix(capi.ErrBadRequest.Error(), "capi: "))

	// The sentinels are errors.New leaves, so a sentinel-vs-sentinel
	// errors.Is collapses to pointer equality and proves nothing about the
	// unwrap chain. Exercise the real chain instead: a mapped 400 wraps
	// ErrBadRequest, so it must unwrap to ErrBadRequest and to no other
	// sentinel. This is what would catch a regression in MapHTTPError.
	badRequest := capi.MapHTTPError(http.StatusBadRequest, nil)
	require.ErrorIs(t, badRequest, capi.ErrBadRequest)

	for _, s := range other {
		assert.NotErrorIs(t, badRequest, s,
			"a mapped 400 must not unwrap to %v", s)
	}
}

// TestMapHTTPError_ServerErrorSentinelCatchesAll5xx verifies that any 5xx
// status — even one not explicitly listed — maps to ErrServerError.
func TestMapHTTPError_ServerErrorSentinelCatchesAll5xx(t *testing.T) {
	t.Parallel()

	for status := 500; status <= 599; status++ {
		err := capi.MapHTTPError(status, nil)
		require.Error(t, err, "status %d must produce a non-nil error", status)
		assert.ErrorIs(
			t,
			err, capi.ErrServerError,
			"status %d must unwrap to ErrServerError", status,
		)
	}
}

// TestMapHTTPError_SentinelIdentity verifies that each sentinel is a
// distinct value and no sentinel unwraps to another sentinel — this is
// critical for errors.Is disambiguation by callers.
func TestMapHTTPError_SentinelIdentity(t *testing.T) {
	t.Parallel()

	sentinels := map[string]error{
		"ErrNotFound":      capi.ErrNotFound,
		"ErrUnauthorized":  capi.ErrUnauthorized,
		"ErrForbidden":     capi.ErrForbidden,
		"ErrConflict":      capi.ErrConflict,
		"ErrUnprocessable": capi.ErrUnprocessable,
		"ErrRateLimited":   capi.ErrRateLimited,
		"ErrServerError":   capi.ErrServerError,
	}

	for name, s := range sentinels {
		require.Error(t, s, "%s must not be nil", name)
		require.NotEmpty(t, s.Error(), "%s must have a non-empty message", name)
		assert.True(t, strings.HasPrefix(s.Error(), "capi: "),
			"%s message %q must start with 'capi: '", name, s.Error())
	}

	for nameA, sentinelA := range sentinels {
		for nameB, sentinelB := range sentinels {
			if nameA == nameB {
				continue
			}

			assert.NotErrorIs(t, sentinelA, sentinelB,
				"sentinels must be distinct: %s must not unwrap to %s", nameA, nameB)
		}
	}
}

// TestMapHTTPError_IsNotFoundCompatibility verifies that the helper
// IsNotFound (in errors.go) accepts errors produced by MapHTTPError — this
// ensures the sentinel path and the legacy APIError-code path are both
// recognized by the same helper, preserving backward compatibility for
// callers using the helper.
func TestMapHTTPError_IsNotFoundCompatibility(t *testing.T) {
	t.Parallel()

	err := capi.MapHTTPError(http.StatusNotFound, []byte(wellFormedNotFoundEnvelope))
	require.Error(t, err)
	assert.True(t, capi.IsNotFound(err),
		"capi.IsNotFound must return true for errors produced by MapHTTPError")

	err2 := capi.MapHTTPError(http.StatusNotFound, nil)
	require.Error(t, err2)
	assert.True(t, capi.IsNotFound(err2),
		"capi.IsNotFound must work even when body was empty")
}
