package capi_test

// Shared literal fixtures reused across pkg/capi tests. Declaring these
// once satisfies goconst's duplicate-literal check without touching the
// query wire-format parity assertions elsewhere in this package, which
// intentionally keep their literal keys/values (see the //nolint:goconst
// comments on those tests) so a typo in a real CF v3 query parameter name
// still fails the test instead of being silently "fixed" by a shared
// constant.
const (
	testAppName1          = "app1"
	testETag              = "abc123"
	testAppsPath          = "/v3/apps"
	testPath              = "/test"
	testPagedPath         = "/test?page=2"
	testResourceName1     = "Resource 1"
	testResourceName2     = "Resource 2"
	testResourceName3     = "Resource 3"
	testStackName         = "cflinuxfs4"
	testPlanName          = "small"
	testNotFoundTitle     = "CF-ResourceNotFound"
	testNotFoundDetail    = "App not found"
	testBatchOpID1        = "op1"
	testBatchOpTypeGet    = "get"
	testBatchResourceApp  = "app"
	testQueryParamPage    = "page"
	testQueryParamPerPage = "per_page"
	stringTrue            = "true"
	stringFalse           = "false"
)
