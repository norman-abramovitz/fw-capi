//nolint:testpackage // needs access to unexported buildDomainCreateRequest and newDomainsCreateCommand
package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
)

// Test constants for domain route policy flag tests.
const (
	testDomainName  = "example.com"
	testFlagName    = "--name"
	testFlagEnforce = "--enforce-route-policies"
	testFlagScope   = "--route-policies-scope"
)

func TestBuildDomainCreateRequestRoutePolicyValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr error
	}{
		{
			name:    "scope without enforce",
			args:    []string{testFlagName, testDomainName, testFlagScope, string(capi.RoutePoliciesScopeOrg)},
			wantErr: ErrScopeRequiresEnforce,
		},
		{
			name:    "enforce without scope",
			args:    []string{testFlagName, testDomainName, testFlagEnforce},
			wantErr: ErrScopeRequiredWithEnforce,
		},
		{
			name:    "enforce with internal",
			args:    []string{testFlagName, testDomainName, testFlagEnforce, testFlagScope, string(capi.RoutePoliciesScopeOrg), "--internal"},
			wantErr: ErrEnforceIncompatibleInternal,
		},
		{
			name:    "enforce with invalid scope",
			args:    []string{testFlagName, testDomainName, testFlagEnforce, testFlagScope, "bogus"},
			wantErr: ErrInvalidRoutePoliciesScope,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cmd := newDomainsCreateCommand()
			require.NoError(t, cmd.ParseFlags(testCase.args))

			createReq, err := buildDomainCreateRequest(context.Background(), nil, cmd)
			require.ErrorIs(t, err, testCase.wantErr)
			assert.Nil(t, createReq)
		})
	}
}

func TestBuildDomainCreateRequestRoutePolicyFields(t *testing.T) {
	t.Parallel()

	cmd := newDomainsCreateCommand()
	require.NoError(t, cmd.ParseFlags([]string{
		testFlagName, "apps.identity", testFlagEnforce, testFlagScope, string(capi.RoutePoliciesScopeOrg),
	}))

	createReq, err := buildDomainCreateRequest(context.Background(), nil, cmd)
	require.NoError(t, err)
	require.NotNil(t, createReq)
	assert.Equal(t, "apps.identity", createReq.Name)
	require.NotNil(t, createReq.EnforceRoutePolicies)
	assert.True(t, *createReq.EnforceRoutePolicies)
	require.NotNil(t, createReq.RoutePoliciesScope)
	assert.Equal(t, capi.RoutePoliciesScopeOrg, *createReq.RoutePoliciesScope)
}
