package capi_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const serviceUnavailableEnvelope = `{"errors":[{"code":10015,"title":"CF-ServiceUnavailable",` +
	`"detail":"Service broker is unavailable"}]}`

// TestMapHTTPError_CarriesStatusCode verifies that every error MapHTTPError
// returns exposes the HTTP status through *capi.StatusError, whatever the
// body, while the message, the sentinel and the *ResponseError stay as they
// were.
func TestMapHTTPError_CarriesStatusCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		status       int
		body         string
		sentinel     error
		wantEnvelope bool
		wantMessage  string
	}{
		{
			name: "4xx with envelope", status: http.StatusNotFound, body: wellFormedNotFoundEnvelope,
			sentinel: capi.ErrNotFound, wantEnvelope: true,
			wantMessage: "capi: resource not found\nCF-ResourceNotFound: App not found (code: 10010)",
		},
		{
			name: "5xx with envelope", status: http.StatusServiceUnavailable, body: serviceUnavailableEnvelope,
			sentinel: capi.ErrServerError, wantEnvelope: true,
			wantMessage: "capi: server error\nCF-ServiceUnavailable: Service broker is unavailable (code: 10015)",
		},
		{
			name: "5xx with plain body", status: http.StatusBadGateway, body: malformedBody,
			sentinel:    capi.ErrServerError,
			wantMessage: "capi: server error (status 502): " + malformedBody,
		},
		{
			name: "5xx with empty body", status: http.StatusInternalServerError,
			sentinel: capi.ErrServerError, wantMessage: "capi: server error (status 500)",
		},
		{
			name: "unlisted 4xx", status: http.StatusTeapot,
			sentinel: capi.ErrBadRequest, wantMessage: "capi: bad request (status 418)",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := capi.MapHTTPError(testCase.status, []byte(testCase.body))
			require.ErrorIs(t, err, testCase.sentinel)
			assert.Equal(t, testCase.wantMessage, err.Error())

			var statusErr *capi.StatusError
			require.ErrorAs(t, err, &statusErr)
			assert.Equal(t, testCase.status, statusErr.StatusCode)

			var envelope *capi.ResponseError
			assert.Equal(t, testCase.wantEnvelope, errors.As(err, &envelope))
		})
	}
}
