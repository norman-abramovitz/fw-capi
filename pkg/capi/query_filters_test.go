package capi_test

import (
	"net/url"
	"testing"

	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
)

// applyOne applies a single typed query option and returns the resulting
// url.Values for assertion.
func applyOne(opt capi.QueryOption) url.Values {
	return capi.ApplyQueryOptions(url.Values{}, []capi.QueryOption{opt})
}

//nolint:funlen,goconst // Test functions can be longer for comprehensive testing; asserts literal wire-format query keys/values, not test fixtures
func TestListFilterOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opt  capi.QueryOption
		key  string
		want string
	}{
		// builds
		{"build guids", capi.WithBuildGUIDs("a", "b"), "guids", "a,b"},
		{"build app_guids", capi.WithBuildAppGUIDs("app1"), "app_guids", "app1"},
		{"build package_guids", capi.WithBuildPackageGUIDs("p1"), "package_guids", "p1"},
		{"build states", capi.WithBuildStates(capi.BuildStateStaged, capi.BuildStateFailed), "states", "STAGED,FAILED"},

		// droplets
		{"droplet guids", capi.WithDropletGUIDs("d1"), "guids", "d1"},
		{"droplet app_guids", capi.WithDropletAppGUIDs("a1"), "app_guids", "a1"},
		{"droplet package_guids", capi.WithDropletPackageGUIDs("p1"), "package_guids", "p1"},
		{"droplet space_guids", capi.WithDropletSpaceGUIDs("s1"), "space_guids", "s1"},
		{"droplet org_guids", capi.WithDropletOrganizationGUIDs("o1"), "organization_guids", "o1"},
		{"droplet states", capi.WithDropletStates(capi.DropletStateStaged), "states", "STAGED"},

		// packages
		{"package guids", capi.WithPackageGUIDs("p1"), "guids", "p1"},
		{"package app_guids", capi.WithPackageAppGUIDs("a1"), "app_guids", "a1"},
		{"package space_guids", capi.WithPackageSpaceGUIDs("s1"), "space_guids", "s1"},
		{"package org_guids", capi.WithPackageOrganizationGUIDs("o1"), "organization_guids", "o1"},
		{"package states", capi.WithPackageStates(capi.PackageStateReady), "states", "READY"},
		{"package types", capi.WithPackageTypes(capi.PackageTypeBits, capi.PackageTypeDocker), "types", "bits,docker"},

		// tasks
		{"task guids", capi.WithTaskGUIDs("t1"), "guids", "t1"},
		{"task app_guids", capi.WithTaskAppGUIDs("a1"), "app_guids", "a1"},
		{"task space_guids", capi.WithTaskSpaceGUIDs("s1"), "space_guids", "s1"},
		{"task org_guids", capi.WithTaskOrganizationGUIDs("o1"), "organization_guids", "o1"},
		{"task names", capi.WithTaskNames("migrate", "seed"), "names", "migrate,seed"},
		{"task states", capi.WithTaskStates(capi.TaskStateRunning, capi.TaskStateCanceling), "states", "RUNNING,CANCELING"},

		// deployments
		{"deployment app_guids", capi.WithDeploymentAppGUIDs("a1"), "app_guids", "a1"},
		{"deployment states", capi.WithDeploymentStates(capi.DeploymentStateDeploying), "states", "DEPLOYING"},
		{"deployment status_values", capi.WithDeploymentStatusValues(capi.DeploymentStatusValueActive), "status_values", "ACTIVE"},
		{"deployment status_reasons", capi.WithDeploymentStatusReasons(capi.DeploymentStatusReasonSuperseded), "status_reasons", "SUPERSEDED"},

		// organizations
		{"org names", capi.WithOrganizationNames("dev", "prod"), "names", "dev,prod"},
		{"org guids", capi.WithOrganizationGUIDs("o1"), "guids", "o1"},

		// domains
		{"domain names", capi.WithDomainNames("example.com"), "names", "example.com"},
		{"domain guids", capi.WithDomainGUIDs("d1"), "guids", "d1"},
		{"domain org_guids", capi.WithDomainOrganizationGUIDs("o1"), "organization_guids", "o1"},

		// organization quotas
		{"org quota names", capi.WithOrganizationQuotaNames("small"), "names", "small"},
		{"org quota guids", capi.WithOrganizationQuotaGUIDs("q1"), "guids", "q1"},
		{"org quota org_guids", capi.WithOrganizationQuotaOrganizationGUIDs("o1"), "organization_guids", "o1"},

		// space quotas
		{"space quota names", capi.WithSpaceQuotaNames("small"), "names", "small"},
		{"space quota guids", capi.WithSpaceQuotaGUIDs("q1"), "guids", "q1"},
		{"space quota org_guids", capi.WithSpaceQuotaOrganizationGUIDs("o1"), "organization_guids", "o1"},
		{"space quota space_guids", capi.WithSpaceQuotaSpaceGUIDs("s1"), "space_guids", "s1"},

		// security groups
		{"secgroup guids", capi.WithSecurityGroupGUIDs("sg1"), "guids", "sg1"},
		{"secgroup names", capi.WithSecurityGroupNames("public"), "names", "public"},
		{"secgroup running_space_guids", capi.WithSecurityGroupRunningSpaceGUIDs("s1"), "running_space_guids", "s1"},
		{"secgroup staging_space_guids", capi.WithSecurityGroupStagingSpaceGUIDs("s1"), "staging_space_guids", "s1"},
		{"secgroup globally_enabled_running", capi.WithSecurityGroupGloballyEnabledRunning(true), "globally_enabled_running", "true"},
		{"secgroup globally_enabled_staging", capi.WithSecurityGroupGloballyEnabledStaging(false), "globally_enabled_staging", "false"},

		// isolation segments
		{"isoseg guids", capi.WithIsolationSegmentGUIDs("i1"), "guids", "i1"},
		{"isoseg names", capi.WithIsolationSegmentNames("seg1"), "names", "seg1"},
		{"isoseg org_guids", capi.WithIsolationSegmentOrganizationGUIDs("o1"), "organization_guids", "o1"},

		// service brokers
		{"broker guids", capi.WithServiceBrokerGUIDs("b1"), "guids", "b1"},
		{"broker names", capi.WithServiceBrokerNames("aws"), "names", "aws"},
		{"broker space_guids", capi.WithServiceBrokerSpaceGUIDs("s1"), "space_guids", "s1"},

		// buildpacks
		{"buildpack guids", capi.WithBuildpackGUIDs("bp1"), "guids", "bp1"},
		{"buildpack names", capi.WithBuildpackNames("ruby"), "names", "ruby"},
		{"buildpack stacks", capi.WithBuildpackStacks("cflinuxfs4"), "stacks", "cflinuxfs4"},
		{"buildpack lifecycle", capi.WithBuildpackLifecycle(capi.BuildpackLifecycleCNB), "lifecycle", "cnb"},

		// stacks
		{"stack guids", capi.WithStackGUIDs("st1"), "guids", "st1"},
		{"stack names", capi.WithStackNames("cflinuxfs4"), "names", "cflinuxfs4"},
		{"stack default", capi.WithStackDefault(true), "default", "true"},

		// users
		{"user guids", capi.WithUserGUIDs("u1"), "guids", "u1"},
		{"user usernames", capi.WithUserUsernames("alice", "bob"), "usernames", "alice,bob"},
		{"user partial_usernames", capi.WithUserPartialUsernames("al"), "partial_usernames", "al"},
		{"user origins", capi.WithUserOrigins("uaa", "ldap"), "origins", "uaa,ldap"},

		// audit events
		{"audit types", capi.WithAuditEventTypes("audit.app.create"), "types", "audit.app.create"},
		{"audit target_guids", capi.WithAuditEventTargetGUIDs("t1"), "target_guids", "t1"},
		{"audit space_guids", capi.WithAuditEventSpaceGUIDs("s1"), "space_guids", "s1"},
		{"audit org_guids", capi.WithAuditEventOrganizationGUIDs("o1"), "organization_guids", "o1"},

		// app usage events
		{"app usage guids", capi.WithAppUsageEventGUIDs("e1"), "guids", "e1"},
		{"app usage after_guid", capi.WithAppUsageEventAfterGUID("e0"), "after_guid", "e0"},

		// service usage events
		{"svc usage guids", capi.WithServiceUsageEventGUIDs("e1"), "guids", "e1"},
		{"svc usage after_guid", capi.WithServiceUsageEventAfterGUID("e0"), "after_guid", "e0"},
		{"svc usage instance types", capi.WithServiceUsageEventServiceInstanceTypes(capi.ServiceInstanceTypeManaged), "service_instance_types", "managed_service_instance"},
		{"svc usage offering guids", capi.WithServiceUsageEventServiceOfferingGUIDs("so1"), "service_offering_guids", "so1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := applyOne(tt.opt)
			assert.Equal(t, tt.want, got.Get(tt.key))
		})
	}
}

// TestListFilterOptions_Compose verifies multiple typed options for one
// resource accumulate into distinct keys.
//
//nolint:goconst // asserts literal wire-format query keys/values, not test fixtures
func TestListFilterOptions_Compose(t *testing.T) {
	t.Parallel()

	opts := []capi.PackageListOption{
		capi.WithPackageAppGUIDs("app1"),
		capi.WithPackageStates(capi.PackageStateReady),
		capi.WithPackageTypes(capi.PackageTypeDocker),
	}

	values := capi.ApplyQueryOptions(url.Values{}, opts)

	assert.Equal(t, "app1", values.Get("app_guids"))
	assert.Equal(t, "READY", values.Get("states"))
	assert.Equal(t, "docker", values.Get("types"))
}
