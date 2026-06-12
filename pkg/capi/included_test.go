package capi_test

import (
	"encoding/json"
	"testing"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoleIncludedFrom(t *testing.T) {
	t.Parallel()

	payload := `{
	  "pagination": {"total_results": 1, "total_pages": 1},
	  "resources": [{"guid": "role-1", "type": "space_developer"}],
	  "included": {
	    "users": [{"guid": "user-1", "username": "norm"}],
	    "spaces": [{"guid": "space-1", "name": "dev",
	      "relationships": {"organization": {"data": {"guid": "org-1"}}}}],
	    "organizations": [{"guid": "org-1", "name": "acme"}]
	  }
	}`

	var list capi.ListResponse[capi.Role]
	require.NoError(t, json.Unmarshal([]byte(payload), &list))

	incl, err := capi.RoleIncludedFrom(&list)
	require.NoError(t, err)
	assert.Len(t, incl.Users, 1)
	assert.Len(t, incl.Spaces, 1)
	assert.Len(t, incl.Organizations, 1)
	require.NotNil(t, incl.Spaces[0].Relationships.Organization.Data)
	assert.Equal(t, "org-1", incl.Spaces[0].Relationships.Organization.Data.GUID)
}

func TestRoleIncludedFrom_NilSafe(t *testing.T) {
	t.Parallel()

	incl, err := capi.RoleIncludedFrom(&capi.ListResponse[capi.Role]{})
	require.NoError(t, err)
	assert.Empty(t, incl.Users)
	assert.Empty(t, incl.Spaces)
	assert.Empty(t, incl.Organizations)
}

func TestAppIncludedFrom(t *testing.T) {
	t.Parallel()

	payload := `{
	  "resources": [{"guid": "app-1", "name": "web"}],
	  "included": {
	    "spaces": [{"guid": "space-1", "name": "dev"}],
	    "organizations": [{"guid": "org-1", "name": "acme"}]
	  }
	}`

	var list capi.ListResponse[capi.App]
	require.NoError(t, json.Unmarshal([]byte(payload), &list))

	incl, err := capi.AppIncludedFrom(&list)
	require.NoError(t, err)
	assert.Equal(t, "space-1", incl.Spaces[0].GUID)
	assert.Equal(t, "org-1", incl.Organizations[0].GUID)
}

func TestDecodeIncluded_MalformedJSON(t *testing.T) {
	t.Parallel()

	// Build the Included map manually with one malformed entry.
	// Passing malformed JSON through json.Unmarshal would fail before
	// RoleIncludedFrom is called, so inject the raw message directly.
	var list capi.ListResponse[capi.Role]
	list.Included = map[string][]json.RawMessage{
		"users": {
			json.RawMessage(`{"guid":"u1","username":"norm"}`),
			json.RawMessage(`{bad json}`),
		},
	}

	_, err := capi.RoleIncludedFrom(&list)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "users")
}

func TestRoleIncludedFrom_NilList(t *testing.T) {
	t.Parallel()

	incl, err := capi.RoleIncludedFrom(nil)
	require.NoError(t, err)
	assert.Empty(t, incl.Users)
	assert.Empty(t, incl.Spaces)
	assert.Empty(t, incl.Organizations)
}

func TestSpaceIncludedFrom(t *testing.T) {
	t.Parallel()

	payload := `{
	  "resources": [{"guid": "space-1", "name": "dev"}],
	  "included": {
	    "organizations": [{"guid": "org-1", "name": "acme"}],
	    "spaces": [{"guid": "space-2", "name": "staging"}]
	  }
	}`

	var list capi.ListResponse[capi.Space]
	require.NoError(t, json.Unmarshal([]byte(payload), &list))

	incl, err := capi.SpaceIncludedFrom(&list)
	require.NoError(t, err)
	assert.Len(t, incl.Organizations, 1)
	assert.Equal(t, "org-1", incl.Organizations[0].GUID)
	assert.Len(t, incl.Spaces, 1)
	assert.Equal(t, "space-2", incl.Spaces[0].GUID)
}

func TestServiceOfferingIncludedFrom(t *testing.T) {
	t.Parallel()

	payload := `{
	  "resources": [{"guid": "offer-1", "name": "mysql"}],
	  "included": {
	    "service_brokers": [{"guid": "broker-1", "name": "core-broker"}]
	  }
	}`

	var list capi.ListResponse[capi.ServiceOffering]
	require.NoError(t, json.Unmarshal([]byte(payload), &list))

	incl, err := capi.ServiceOfferingIncludedFrom(&list)
	require.NoError(t, err)
	assert.Len(t, incl.ServiceBrokers, 1)
	assert.Equal(t, "broker-1", incl.ServiceBrokers[0].GUID)
}
