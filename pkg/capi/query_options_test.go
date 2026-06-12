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

func TestIncludeConstants_Encoding(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		got  url.Values
		want string
	}{
		{"apps", capi.ApplyQueryOptions(nil, []capi.AppGetOption{capi.AppIncludeSpace, capi.AppIncludeSpaceOrganization}), "space,space.organization"},
		{"routes", capi.ApplyQueryOptions(nil, []capi.RouteGetOption{capi.RouteIncludeDomain, capi.RouteIncludeSpace, capi.RouteIncludeSpaceOrganization}), "domain,space,space.organization"},
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
