package capi_test

import (
	"net/url"
	"testing"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
)

// TestParityFilterOptions covers the entity and enum filter constructors added
// to the endpoints that also expose include/fields options.
func TestParityFilterOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opt  capi.QueryOption
		key  string
		want string
	}{
		// apps
		{"app names", capi.WithAppNames("web", "api"), "names", "web,api"},
		{"app guids", capi.WithAppGUIDs("a1"), "guids", "a1"},
		{"app space_guids", capi.WithAppSpaceGUIDs("s1"), "space_guids", "s1"},
		{"app org_guids", capi.WithAppOrganizationGUIDs("o1"), "organization_guids", "o1"},
		{"app stacks", capi.WithAppStacks("cflinuxfs4"), "stacks", "cflinuxfs4"},
		{"app lifecycle_type", capi.WithAppLifecycleType(capi.AppLifecycleTypeDocker), "lifecycle_type", "docker"},

		// routes
		{"route guids", capi.WithRouteGUIDs("r1"), "guids", "r1"},
		{"route hosts", capi.WithRouteHosts("web", "api"), "hosts", "web,api"},
		{"route paths", capi.WithRoutePaths("/a", "/b"), "paths", "/a,/b"},
		{"route ports", capi.WithRoutePorts(8080, 9090), "ports", "8080,9090"},
		{"route domain_guids", capi.WithRouteDomainGUIDs("d1"), "domain_guids", "d1"},
		{"route space_guids", capi.WithRouteSpaceGUIDs("s1"), "space_guids", "s1"},
		{"route org_guids", capi.WithRouteOrganizationGUIDs("o1"), "organization_guids", "o1"},
		{"route service_instance_guids", capi.WithRouteServiceInstanceGUIDs("si1"), "service_instance_guids", "si1"},
		{"route app_guids", capi.WithRouteAppGUIDs("a1"), "app_guids", "a1"},

		// route policies
		{"route policy guids", capi.WithRoutePolicyGUIDs("p1", "p2"), "guids", "p1,p2"},
		{"route policy route_guids", capi.WithRoutePolicyRouteGUIDs("r1"), "route_guids", "r1"},
		{"route policy space_guids", capi.WithRoutePolicySpaceGUIDs("s1"), "space_guids", "s1"},
		{"route policy sources", capi.WithRoutePolicySources("cf:any", "cf:app:a1"), "sources", "cf:any,cf:app:a1"},
		{"route policy source_guids", capi.WithRoutePolicySourceGUIDs("a1"), "source_guids", "a1"},

		// spaces
		{"space names", capi.WithSpaceNames("dev"), "names", "dev"},
		{"space guids", capi.WithSpaceGUIDs("s1"), "guids", "s1"},
		{"space org_guids", capi.WithSpaceOrganizationGUIDs("o1"), "organization_guids", "o1"},

		// roles
		{"role guids", capi.WithRoleGUIDs("r1"), "guids", "r1"},
		{
			"role types",
			capi.WithRoleTypes(capi.RoleTypeSpaceDeveloper, capi.RoleTypeSpaceSupporter),
			"types", "space_developer,space_supporter",
		},
		{"role space_guids", capi.WithRoleSpaceGUIDs("s1"), "space_guids", "s1"},
		{"role org_guids", capi.WithRoleOrganizationGUIDs("o1"), "organization_guids", "o1"},
		{"role user_guids", capi.WithRoleUserGUIDs("u1"), "user_guids", "u1"},

		// service instances
		{"si names", capi.WithServiceInstanceNames("db"), "names", "db"},
		{"si guids", capi.WithServiceInstanceGUIDs("si1"), "guids", "si1"},
		{"si space_guids", capi.WithServiceInstanceSpaceGUIDs("s1"), "space_guids", "s1"},
		{"si org_guids", capi.WithServiceInstanceOrganizationGUIDs("o1"), "organization_guids", "o1"},
		{"si plan_guids", capi.WithServiceInstanceServicePlanGUIDs("sp1"), "service_plan_guids", "sp1"},
		{"si plan_names", capi.WithServiceInstanceServicePlanNames("small"), "service_plan_names", "small"},
		{
			"si type user-provided",
			capi.WithServiceInstanceType(capi.ServiceInstanceFilterTypeUserProvided),
			"type", "user-provided",
		},

		// service plans
		{"plan guids", capi.WithServicePlanGUIDs("sp1"), "guids", "sp1"},
		{"plan names", capi.WithServicePlanNames("small"), "names", "small"},
		{"plan available", capi.WithServicePlanAvailable(true), "available", "true"},
		{"plan broker_catalog_ids", capi.WithServicePlanBrokerCatalogIDs("cat1"), "broker_catalog_ids", "cat1"},
		{"plan broker_guids", capi.WithServicePlanServiceBrokerGUIDs("sb1"), "service_broker_guids", "sb1"},
		{"plan broker_names", capi.WithServicePlanServiceBrokerNames("broker"), "service_broker_names", "broker"},
		{"plan offering_guids", capi.WithServicePlanServiceOfferingGUIDs("so1"), "service_offering_guids", "so1"},
		{"plan offering_names", capi.WithServicePlanServiceOfferingNames("mysql"), "service_offering_names", "mysql"},
		{"plan si_guids", capi.WithServicePlanServiceInstanceGUIDs("si1"), "service_instance_guids", "si1"},
		{"plan space_guids", capi.WithServicePlanSpaceGUIDs("s1"), "space_guids", "s1"},
		{"plan org_guids", capi.WithServicePlanOrganizationGUIDs("o1"), "organization_guids", "o1"},

		// service offerings
		{"offering guids", capi.WithServiceOfferingGUIDs("so1"), "guids", "so1"},
		{"offering names", capi.WithServiceOfferingNames("mysql"), "names", "mysql"},
		{"offering available", capi.WithServiceOfferingAvailable(false), "available", "false"},
		{"offering broker_catalog_ids", capi.WithServiceOfferingBrokerCatalogIDs("cat1"), "broker_catalog_ids", "cat1"},
		{"offering broker_guids", capi.WithServiceOfferingServiceBrokerGUIDs("sb1"), "service_broker_guids", "sb1"},
		{"offering broker_names", capi.WithServiceOfferingServiceBrokerNames("broker"), "service_broker_names", "broker"},
		{"offering space_guids", capi.WithServiceOfferingSpaceGUIDs("s1"), "space_guids", "s1"},
		{"offering org_guids", capi.WithServiceOfferingOrganizationGUIDs("o1"), "organization_guids", "o1"},

		// service credential bindings
		{"scb guids", capi.WithServiceCredentialBindingGUIDs("b1"), "guids", "b1"},
		{"scb names", capi.WithServiceCredentialBindingNames("bind"), "names", "bind"},
		{"scb si_guids", capi.WithServiceCredentialBindingServiceInstanceGUIDs("si1"), "service_instance_guids", "si1"},
		{"scb si_names", capi.WithServiceCredentialBindingServiceInstanceNames("db"), "service_instance_names", "db"},
		{"scb plan_guids", capi.WithServiceCredentialBindingServicePlanGUIDs("sp1"), "service_plan_guids", "sp1"},
		{"scb plan_names", capi.WithServiceCredentialBindingServicePlanNames("small"), "service_plan_names", "small"},
		{"scb offering_guids", capi.WithServiceCredentialBindingServiceOfferingGUIDs("so1"), "service_offering_guids", "so1"},
		{"scb offering_names", capi.WithServiceCredentialBindingServiceOfferingNames("mysql"), "service_offering_names", "mysql"},
		{"scb app_guids", capi.WithServiceCredentialBindingAppGUIDs("a1"), "app_guids", "a1"},
		{"scb app_names", capi.WithServiceCredentialBindingAppNames("web"), "app_names", "web"},
		{"scb type key", capi.WithServiceCredentialBindingType(capi.ServiceCredentialBindingTypeKey), "type", "key"},

		// service route bindings
		{"srb guids", capi.WithServiceRouteBindingGUIDs("rb1"), "guids", "rb1"},
		{"srb si_guids", capi.WithServiceRouteBindingServiceInstanceGUIDs("si1"), "service_instance_guids", "si1"},
		{"srb si_names", capi.WithServiceRouteBindingServiceInstanceNames("db"), "service_instance_names", "db"},
		{"srb route_guids", capi.WithServiceRouteBindingRouteGUIDs("r1"), "route_guids", "r1"},

		// processes
		{"process guids", capi.WithProcessGUIDs("p1"), "guids", "p1"},
		{"process types", capi.WithProcessTypes("web", "worker"), "types", "web,worker"},
		{"process app_guids", capi.WithProcessAppGUIDs("a1"), "app_guids", "a1"},
		{"process space_guids", capi.WithProcessSpaceGUIDs("s1"), "space_guids", "s1"},
		{"process org_guids", capi.WithProcessOrganizationGUIDs("o1"), "organization_guids", "o1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := applyOne(tt.opt)
			assert.Equal(t, tt.want, got.Get(tt.key))
		})
	}
}

// TestParityFilterOptions_ComposeWithInclude proves a filter option and an
// include option for the same resource coexist on a single List call: the
// scalar filter sets its key while the include accumulates separately.
func TestParityFilterOptions_ComposeWithInclude(t *testing.T) {
	t.Parallel()

	t.Run("apps filter and include", func(t *testing.T) {
		t.Parallel()

		got := capi.ApplyQueryOptions(url.Values{}, []capi.AppListOption{
			capi.WithAppNames("web"),
			capi.WithAppSpaceGUIDs("s1"),
			capi.AppIncludeSpace,
		})
		assert.Equal(t, "web", got.Get("names"))
		assert.Equal(t, "s1", got.Get("space_guids"))
		assert.Equal(t, "space", got.Get("include"))
	})

	t.Run("service instances filter and fields", func(t *testing.T) {
		t.Parallel()

		got := capi.ApplyQueryOptions(url.Values{}, []capi.ServiceInstanceListOption{
			capi.WithServiceInstanceType(capi.ServiceInstanceFilterTypeManaged),
			capi.WithServiceInstanceFields(capi.ServiceInstanceFieldsSpace, "name", "guid"),
		})
		assert.Equal(t, "managed", got.Get("type"))
		assert.Equal(t, "name,guid", got.Get("fields[space]"))
	})
}
