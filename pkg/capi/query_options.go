package capi

import (
	"net/url"
	"strings"
)

// QueryOption is the common behavior shared by all typed query options.
// The apply method is unexported so each resource's option set is closed:
// only values defined in this package satisfy the per-resource interfaces.
type QueryOption interface {
	applyQuery(v url.Values)
}

// ApplyQueryOptions merges typed options into v, allocating v when nil and
// options are present. Include-style options append (comma-joined, deduped);
// scalar options overwrite, so a typed option wins over the same key from
// QueryParams.
func ApplyQueryOptions[O QueryOption](v url.Values, opts []O) url.Values {
	if len(opts) == 0 {
		return v
	}

	if v == nil {
		v = url.Values{}
	}

	for _, o := range opts {
		o.applyQuery(v)
	}

	return v
}

// appendInclude adds value to the comma-joined include parameter,
// skipping values already present.
func appendInclude(v url.Values, value string) {
	current := v.Get("include")
	if current == "" {
		v.Set("include", value)
		return
	}

	for _, existing := range strings.Split(current, ",") {
		if existing == value {
			return
		}
	}

	v.Set("include", current+","+value)
}

// RoleGetOption configures GET /v3/roles/{guid}.
type RoleGetOption interface {
	QueryOption
	roleGet()
}

// RoleListOption configures GET /v3/roles.
type RoleListOption interface {
	QueryOption
	roleList()
}

type roleInclude string

func (roleInclude) roleGet()  {}
func (roleInclude) roleList() {}
func (r roleInclude) applyQuery(v url.Values) { appendInclude(v, string(r)) }

// Valid include values for roles (CF v3 3.222.0).
const (
	RoleIncludeUser         roleInclude = "user"
	RoleIncludeSpace        roleInclude = "space"
	RoleIncludeOrganization roleInclude = "organization"
)
