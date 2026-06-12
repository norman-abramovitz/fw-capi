package capi_test

import (
	"net/url"
	"testing"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
)

func TestApplyQueryOptions_NilValuesNoOpts(t *testing.T) {
	t.Parallel()
	assert.Nil(t, capi.ApplyQueryOptions[capi.RoleGetOption](nil, nil))
}

func TestApplyQueryOptions_AllocatesWhenOptsPresent(t *testing.T) {
	t.Parallel()
	v := capi.ApplyQueryOptions(nil, []capi.RoleGetOption{capi.RoleIncludeSpace})
	assert.Equal(t, "space", v.Get("include"))
}

func TestApplyQueryOptions_IncludesJoinAndDedupe(t *testing.T) {
	t.Parallel()
	v := capi.ApplyQueryOptions(nil, []capi.RoleGetOption{
		capi.RoleIncludeSpace, capi.RoleIncludeOrganization, capi.RoleIncludeSpace,
	})
	assert.Equal(t, "space,organization", v.Get("include"))
}

func TestApplyQueryOptions_MergesIntoExistingValues(t *testing.T) {
	t.Parallel()
	v := url.Values{"include": {"user"}, "page": {"2"}}
	v = capi.ApplyQueryOptions(v, []capi.RoleListOption{capi.RoleIncludeSpace})
	assert.Equal(t, "user,space", v.Get("include"))
	assert.Equal(t, "2", v.Get("page"))
}
