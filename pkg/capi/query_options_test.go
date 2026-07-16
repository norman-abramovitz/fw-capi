package capi_test

import (
	"net/url"
	"testing"
	"time"

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

func TestIncludeConstants_Encoding(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		got  url.Values
		want string
	}{
		{"apps", capi.ApplyQueryOptions(nil, []capi.AppGetOption{capi.AppIncludeSpace, capi.AppIncludeSpaceOrganization}), "space,space.organization"},
		{"routes", capi.ApplyQueryOptions(nil, []capi.RouteGetOption{capi.RouteIncludeDomain, capi.RouteIncludeSpace, capi.RouteIncludeSpaceOrganization, capi.RouteIncludeRoutePolicies}), "domain,space,space.organization,route_policies"},
		{"route policies", capi.ApplyQueryOptions(nil, []capi.RoutePolicyGetOption{capi.RoutePolicyIncludeRoute, capi.RoutePolicyIncludeSource}), "route,source"},
		{"spaces", capi.ApplyQueryOptions(nil, []capi.SpaceGetOption{capi.SpaceIncludeOrganization}), "organization"},
		{"scb", capi.ApplyQueryOptions(nil, []capi.ServiceCredentialBindingGetOption{capi.ServiceCredentialBindingIncludeApp, capi.ServiceCredentialBindingIncludeServiceInstance}), "app,service_instance"},
		{"plans", capi.ApplyQueryOptions(nil, []capi.ServicePlanGetOption{capi.ServicePlanIncludeSpaceOrganization, capi.ServicePlanIncludeServiceOffering}), "space.organization,service_offering"},
		{"srb", capi.ApplyQueryOptions(nil, []capi.ServiceRouteBindingGetOption{capi.ServiceRouteBindingIncludeRoute, capi.ServiceRouteBindingIncludeServiceInstance}), "route,service_instance"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, c.got.Get("include"), c.name)
	}
}

func TestProcessEmbed_Encoding(t *testing.T) {
	t.Parallel()

	v := capi.ApplyQueryOptions(nil, []capi.ProcessGetOption{capi.ProcessEmbedInstances})
	assert.Equal(t, "process_instances", v.Get("embed"))
}

func TestFieldsOptions_Encoding(t *testing.T) {
	t.Parallel()

	v := capi.ApplyQueryOptions(nil, []capi.ServiceInstanceGetOption{
		capi.WithServiceInstanceFields(capi.ServiceInstanceFieldsSpaceOrganization, "name", "guid"),
	})
	assert.Equal(t, "name,guid", v.Get("fields[space.organization]"))

	v = capi.ApplyQueryOptions(nil, []capi.ServiceOfferingGetOption{
		capi.WithServiceOfferingFields(capi.ServiceOfferingFieldsServiceBroker, "name", "guid"),
	})
	assert.Equal(t, "name,guid", v.Get("fields[service_broker]"))

	v = capi.ApplyQueryOptions(nil, []capi.ServicePlanGetOption{
		capi.WithServicePlanFields(capi.ServicePlanFieldsServiceOfferingServiceBroker, "name"),
	})
	assert.Equal(t, "name", v.Get("fields[service_offering.service_broker]"))
}

func TestRouteDestinationsOptions_Encoding(t *testing.T) {
	t.Parallel()

	v := capi.ApplyQueryOptions(nil, []capi.RouteDestinationsOption{
		capi.WithDestinationGUIDs("d1", "d2"),
		capi.WithDestinationAppGUIDs("a1"),
	})
	assert.Equal(t, "d1,d2", v.Get("guids"))
	assert.Equal(t, "a1", v.Get("app_guids"))
}

func TestServiceOfferingPurge_Encoding(t *testing.T) {
	t.Parallel()

	v := capi.ApplyQueryOptions(nil, []capi.ServiceOfferingDeleteOption{capi.PurgeServiceOffering})
	assert.Equal(t, "true", v.Get("purge"))
}

func TestFieldsOptions_UsableInListCalls(t *testing.T) {
	t.Parallel()

	v := capi.ApplyQueryOptions(nil, []capi.ServicePlanListOption{
		capi.WithServicePlanFields(capi.ServicePlanFieldsServiceOfferingServiceBroker, "name"),
	})
	assert.Equal(t, "name", v.Get("fields[service_offering.service_broker]"))

	v = capi.ApplyQueryOptions(nil, []capi.ServiceInstanceListOption{
		capi.WithServiceInstanceFields(capi.ServiceInstanceFieldsSpace, "guid"),
	})
	assert.Equal(t, "guid", v.Get("fields[space]"))

	v = capi.ApplyQueryOptions(nil, []capi.ServiceOfferingListOption{
		capi.WithServiceOfferingFields(capi.ServiceOfferingFieldsServiceBroker, "name"),
	})
	assert.Equal(t, "name", v.Get("fields[service_broker]"))
}

func TestApplyQueryOptions_ScalarOverwritesExistingKey(t *testing.T) {
	t.Parallel()

	v := url.Values{"purge": {"false"}}
	v = capi.ApplyQueryOptions(v, []capi.ServiceOfferingDeleteOption{capi.PurgeServiceOffering})
	assert.Equal(t, "true", v.Get("purge"))
	assert.Len(t, v["purge"], 1)
}

func TestApplyQueryOptions_MutatesInPlace(t *testing.T) {
	t.Parallel()

	base := url.Values{"page": {"2"}}
	returned := capi.ApplyQueryOptions(base, []capi.RoleListOption{capi.RoleIncludeSpace})
	// append-style: non-nil input is mutated and aliased by the return
	assert.Equal(t, "space", base.Get("include"))
	assert.Equal(t, "space", returned.Get("include"))
}

func TestServiceInstanceInclude_Constants(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		opts []capi.ServiceInstanceGetOption
		want string
	}{
		{
			"space",
			[]capi.ServiceInstanceGetOption{capi.ServiceInstanceIncludeSpace},
			"space",
		},
		{
			"service_plan",
			[]capi.ServiceInstanceGetOption{capi.ServiceInstanceIncludeServicePlan},
			"service_plan",
		},
		{
			"service_plan.service_offering",
			[]capi.ServiceInstanceGetOption{capi.ServiceInstanceIncludeServicePlanServiceOffering},
			"service_plan.service_offering",
		},
		{
			"service_plan.service_offering.service_broker",
			[]capi.ServiceInstanceGetOption{capi.ServiceInstanceIncludeServicePlanServiceOfferingBroker},
			"service_plan.service_offering.service_broker",
		},
		{
			"multi+dedupe",
			[]capi.ServiceInstanceGetOption{
				capi.ServiceInstanceIncludeSpace,
				capi.ServiceInstanceIncludeServicePlan,
				capi.ServiceInstanceIncludeSpace, // duplicate
			},
			"space,service_plan",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			v := capi.ApplyQueryOptions(nil, c.opts)
			assert.Equal(t, c.want, v.Get("include"))
		})
	}
}

func TestServiceInstanceInclude_SatisfiesListOption(t *testing.T) {
	t.Parallel()

	v := capi.ApplyQueryOptions(nil, []capi.ServiceInstanceListOption{
		capi.ServiceInstanceIncludeSpace,
		capi.ServiceInstanceIncludeServicePlanServiceOffering,
	})
	assert.Equal(t, "space,service_plan.service_offering", v.Get("include"))
}

func TestServiceOfferingInclude_Constants(t *testing.T) {
	t.Parallel()

	// satisfies Get
	v := capi.ApplyQueryOptions(nil, []capi.ServiceOfferingGetOption{
		capi.ServiceOfferingIncludeServiceBroker,
	})
	assert.Equal(t, "service_broker", v.Get("include"))

	// satisfies List
	v = capi.ApplyQueryOptions(nil, []capi.ServiceOfferingListOption{
		capi.ServiceOfferingIncludeServiceBroker,
	})
	assert.Equal(t, "service_broker", v.Get("include"))

	// dedupe
	v = capi.ApplyQueryOptions(nil, []capi.ServiceOfferingGetOption{
		capi.ServiceOfferingIncludeServiceBroker,
		capi.ServiceOfferingIncludeServiceBroker,
	})
	assert.Equal(t, "service_broker", v.Get("include"))
}

func TestWithTimestampFilter_ValidOps(t *testing.T) {
	t.Parallel()

	ts := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
	want := "2024-03-15T12:00:00Z"

	for _, op := range []string{"gt", "gte", "lt", "lte"} {
		op := op
		t.Run(op, func(t *testing.T) {
			t.Parallel()
			v := capi.ApplyQueryOptions(nil, []capi.QueryOption{
				capi.WithTimestampFilter("updated_at", op, ts),
			})
			key := "updated_at[" + op + "]"
			assert.Equal(t, want, v.Get(key), "key=%s", key)
		})
	}
}

func TestWithTimestampFilter_NormalizesToUTC(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("America/New_York")
	assert.NoError(t, err)

	ts := time.Date(2024, 3, 15, 8, 0, 0, 0, loc) // -4h from UTC
	v := capi.ApplyQueryOptions(nil, []capi.QueryOption{
		capi.WithTimestampFilter("created_at", "gte", ts),
	})
	assert.Equal(t, "2024-03-15T12:00:00Z", v.Get("created_at[gte]"))
}

func TestWithTimestampFilter_InvalidOpIsNoop(t *testing.T) {
	t.Parallel()

	ts := time.Now().UTC()

	for _, bad := range []string{"", "eq", "ne", "GT", ">=", "after"} {
		bad := bad
		t.Run(bad, func(t *testing.T) {
			t.Parallel()
			v := capi.ApplyQueryOptions(nil, []capi.QueryOption{
				capi.WithTimestampFilter("created_at", bad, ts),
			})
			// no key must be set — noop leaves url.Values empty
			assert.Empty(t, v, "op=%q should produce no query params", bad)
		})
	}
}
