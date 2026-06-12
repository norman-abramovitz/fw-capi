package capi_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockClient implements capi.Client for testing.
type MockClient struct {
	mock.Mock
}

func (m *MockClient) GetInfo(ctx context.Context) (*capi.Info, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("mock error in GetInfo: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		info, ok := args.Get(0).(*capi.Info)
		if !ok {
			return nil, constants.ErrInvalidTypeAssertion
		}

		return info, fmt.Errorf("mock error in GetInfo: %w", err)
	}

	info, ok := args.Get(0).(*capi.Info)
	if !ok {
		return nil, constants.ErrInvalidTypeAssertion
	}

	return info, nil
}

func (m *MockClient) GetRoot(ctx context.Context) (*capi.RootInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("mock error in GetRoot: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		rootInfo, ok := args.Get(0).(*capi.RootInfo)
		if !ok {
			return nil, constants.ErrInvalidTypeAssertion
		}

		return rootInfo, fmt.Errorf("mock error in GetRoot: %w", err)
	}

	rootInfo, ok := args.Get(0).(*capi.RootInfo)
	if !ok {
		return nil, constants.ErrInvalidTypeAssertion
	}

	return rootInfo, nil
}

func (m *MockClient) GetRootInfo(ctx context.Context) (*capi.RootInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("mock error in GetRootInfo: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		rootInfo, ok := args.Get(0).(*capi.RootInfo)
		if !ok {
			return nil, constants.ErrInvalidTypeAssertion
		}

		return rootInfo, fmt.Errorf("mock error in GetRootInfo: %w", err)
	}

	rootInfo, ok := args.Get(0).(*capi.RootInfo)
	if !ok {
		return nil, constants.ErrInvalidTypeAssertion
	}

	return rootInfo, nil
}

func (m *MockClient) GetUsageSummary(ctx context.Context) (*capi.UsageSummary, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("GetUsageSummary failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		usageSummary, ok := args.Get(0).(*capi.UsageSummary)
		if !ok {
			return nil, constants.ErrInvalidTypeAssertion
		}

		return usageSummary, fmt.Errorf("GetUsageSummary failed: %w", err)
	}

	summary, ok := args.Get(0).(*capi.UsageSummary)
	if !ok {
		return nil, fmt.Errorf("%w", constants.ErrUnexpectedMockReturnType)
	}

	return summary, nil
}

func (m *MockClient) ClearBuildpackCache(ctx context.Context) (*capi.Job, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("ClearBuildpackCache failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		job, ok := args.Get(0).(*capi.Job)
		if !ok {
			return nil, fmt.Errorf("ClearBuildpackCache failed: %w", err)
		}

		return job, fmt.Errorf("ClearBuildpackCache failed: %w", err)
	}

	job, ok := args.Get(0).(*capi.Job)
	if !ok {
		return nil, fmt.Errorf("%w", constants.ErrUnexpectedMockReturnType)
	}

	return job, nil
}

func (m *MockClient) Apps() capi.AppsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, ok := args.Get(0).(capi.AppsClient)
	if !ok {
		return nil
	}

	return client
}

func (m *MockClient) Organizations() capi.OrganizationsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, ok := args.Get(0).(capi.OrganizationsClient)
	if !ok {
		return nil
	}

	return client
}

func (m *MockClient) Spaces() capi.SpacesClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, ok := args.Get(0).(capi.SpacesClient)
	if !ok {
		return nil
	}

	return client
}

func (m *MockClient) Domains() capi.DomainsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.DomainsClient)

	return client
}

func (m *MockClient) Routes() capi.RoutesClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.RoutesClient)

	return client
}

func (m *MockClient) ServiceBrokers() capi.ServiceBrokersClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.ServiceBrokersClient)

	return client
}

func (m *MockClient) ServiceOfferings() capi.ServiceOfferingsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.ServiceOfferingsClient)

	return client
}

func (m *MockClient) ServicePlans() capi.ServicePlansClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.ServicePlansClient)

	return client
}

func (m *MockClient) ServiceInstances() capi.ServiceInstancesClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.ServiceInstancesClient)

	return client
}

func (m *MockClient) ServiceCredentialBindings() capi.ServiceCredentialBindingsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.ServiceCredentialBindingsClient)

	return client
}

func (m *MockClient) ServiceRouteBindings() capi.ServiceRouteBindingsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.ServiceRouteBindingsClient)

	return client
}

func (m *MockClient) Builds() capi.BuildsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.BuildsClient)

	return client
}

func (m *MockClient) Buildpacks() capi.BuildpacksClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.BuildpacksClient)

	return client
}

func (m *MockClient) Deployments() capi.DeploymentsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.DeploymentsClient)

	return client
}

func (m *MockClient) Droplets() capi.DropletsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.DropletsClient)

	return client
}

func (m *MockClient) Packages() capi.PackagesClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.PackagesClient)

	return client
}

func (m *MockClient) Processes() capi.ProcessesClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.ProcessesClient)

	return client
}

func (m *MockClient) Tasks() capi.TasksClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.TasksClient)

	return client
}

func (m *MockClient) Stacks() capi.StacksClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.StacksClient)

	return client
}

func (m *MockClient) Users() capi.UsersClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.UsersClient)

	return client
}

func (m *MockClient) Roles() capi.RolesClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.RolesClient)

	return client
}

func (m *MockClient) SecurityGroups() capi.SecurityGroupsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.SecurityGroupsClient)

	return client
}

func (m *MockClient) IsolationSegments() capi.IsolationSegmentsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.IsolationSegmentsClient)

	return client
}

func (m *MockClient) FeatureFlags() capi.FeatureFlagsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.FeatureFlagsClient)

	return client
}

func (m *MockClient) Jobs() capi.JobsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.JobsClient)

	return client
}

func (m *MockClient) OrganizationQuotas() capi.OrganizationQuotasClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.OrganizationQuotasClient)

	return client
}

func (m *MockClient) SpaceQuotas() capi.SpaceQuotasClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.SpaceQuotasClient)

	return client
}

func (m *MockClient) Sidecars() capi.SidecarsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.SidecarsClient)

	return client
}

func (m *MockClient) Revisions() capi.RevisionsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.RevisionsClient)

	return client
}

func (m *MockClient) EnvironmentVariableGroups() capi.EnvironmentVariableGroupsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.EnvironmentVariableGroupsClient)

	return client
}

func (m *MockClient) AppUsageEvents() capi.AppUsageEventsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.AppUsageEventsClient)

	return client
}

func (m *MockClient) ServiceUsageEvents() capi.ServiceUsageEventsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.ServiceUsageEventsClient)

	return client
}

func (m *MockClient) AuditEvents() capi.AuditEventsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.AuditEventsClient)

	return client
}

func (m *MockClient) ResourceMatches() capi.ResourceMatchesClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.ResourceMatchesClient)

	return client
}

func (m *MockClient) Manifests() capi.ManifestsClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.ManifestsClient)

	return client
}

func (m *MockClient) Routing() capi.RoutingClient {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}

	client, _ := args.Get(0).(capi.RoutingClient)

	return client
}

// MockAppsClient implements capi.AppsClient for testing.
type MockAppsClient struct {
	mock.Mock
}

func (m *MockAppsClient) Create(ctx context.Context, request *capi.AppCreateRequest) (*capi.App, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("mock error: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		result, _ := args.Get(0).(*capi.App)

		return result, fmt.Errorf("mock error: %w", err)
	}

	result, _ := args.Get(0).(*capi.App)

	return result, nil
}

func (m *MockAppsClient) Get(ctx context.Context, guid string, opts ...capi.AppGetOption) (*capi.App, error) {
	args := m.Called(ctx, guid)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("mock error: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		result, _ := args.Get(0).(*capi.App)

		return result, fmt.Errorf("mock error: %w", err)
	}

	result, _ := args.Get(0).(*capi.App)

	return result, nil
}

func (m *MockAppsClient) List(ctx context.Context, params *capi.QueryParams, opts ...capi.AppListOption) (*capi.ListResponse[capi.App], error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, fmt.Errorf("list apps failed: %w", args.Error(1))
	}

	err := args.Error(1)
	if err != nil {
		result, _ := args.Get(0).(*capi.ListResponse[capi.App])

		return result, fmt.Errorf("list apps failed: %w", err)
	}

	result, _ := args.Get(0).(*capi.ListResponse[capi.App])

	return result, nil
}

func (m *MockAppsClient) Update(ctx context.Context, guid string, request *capi.AppUpdateRequest) (*capi.App, error) {
	args := m.Called(ctx, guid, request)
	if args.Get(0) == nil {
		return nil, fmt.Errorf("update app failed: %w", args.Error(1))
	}

	err := args.Error(1)
	if err != nil {
		result, _ := args.Get(0).(*capi.App)

		return result, fmt.Errorf("update app failed: %w", err)
	}

	result, _ := args.Get(0).(*capi.App)

	return result, nil
}

func (m *MockAppsClient) Delete(ctx context.Context, guid string) (*capi.Job, error) {
	args := m.Called(ctx, guid)

	err := args.Error(1)
	if err != nil {
		return nil, fmt.Errorf("delete app failed: %w", err)
	}

	job, _ := args.Get(0).(*capi.Job)

	return job, nil
}

func (m *MockAppsClient) GetCurrentDroplet(ctx context.Context, guid string) (*capi.Droplet, error) {
	args := m.Called(ctx, guid)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("get current droplet failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		result, _ := args.Get(0).(*capi.Droplet)

		return result, fmt.Errorf("get current droplet failed: %w", err)
	}

	result, _ := args.Get(0).(*capi.Droplet)

	return result, nil
}

func (m *MockAppsClient) SetCurrentDroplet(ctx context.Context, guid string, dropletGUID string) (*capi.Relationship, error) {
	args := m.Called(ctx, guid, dropletGUID)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("set current droplet failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		return nil, fmt.Errorf("set current droplet failed: %w", err)
	}

	result, _ := args.Get(0).(*capi.Relationship)

	return result, nil
}

func (m *MockAppsClient) GetEnv(ctx context.Context, guid string) (*capi.AppEnvironment, error) {
	args := m.Called(ctx, guid)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("get app environment failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		return nil, fmt.Errorf("get app environment failed: %w", err)
	}

	result, _ := args.Get(0).(*capi.AppEnvironment)

	return result, nil
}

func (m *MockAppsClient) GetEnvVars(ctx context.Context, guid string) (map[string]interface{}, error) {
	args := m.Called(ctx, guid)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("get env vars failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		return nil, fmt.Errorf("get env vars failed: %w", err)
	}

	result, _ := args.Get(0).(map[string]interface{})

	return result, nil
}

func (m *MockAppsClient) UpdateEnvVars(ctx context.Context, guid string, vars map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(ctx, guid, vars)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("update env vars failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		return nil, fmt.Errorf("update env vars failed: %w", err)
	}

	result, _ := args.Get(0).(map[string]interface{})

	return result, nil
}

func (m *MockAppsClient) GetPermissions(ctx context.Context, guid string) (*capi.AppPermissions, error) {
	args := m.Called(ctx, guid)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("get permissions failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		return nil, fmt.Errorf("get permissions failed: %w", err)
	}

	result, _ := args.Get(0).(*capi.AppPermissions)

	return result, nil
}

func (m *MockAppsClient) GetSSHEnabled(ctx context.Context, guid string) (*capi.AppSSHEnabled, error) {
	args := m.Called(ctx, guid)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("get SSH enabled failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		return nil, fmt.Errorf("get SSH enabled failed: %w", err)
	}

	result, _ := args.Get(0).(*capi.AppSSHEnabled)

	return result, nil
}

func (m *MockAppsClient) Start(ctx context.Context, guid string) (*capi.Job, error) {
	args := m.Called(ctx, guid)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("start app failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		return nil, fmt.Errorf("start app failed: %w", err)
	}

	result, _ := args.Get(0).(*capi.Job)

	return result, nil
}

func (m *MockAppsClient) Stop(ctx context.Context, guid string) (*capi.Job, error) {
	args := m.Called(ctx, guid)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("stop app failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		return nil, fmt.Errorf("stop app failed: %w", err)
	}

	result, _ := args.Get(0).(*capi.Job)

	return result, nil
}

func (m *MockAppsClient) Restart(ctx context.Context, guid string) (*capi.Job, error) {
	args := m.Called(ctx, guid)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("restart app failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		return nil, fmt.Errorf("restart app failed: %w", err)
	}

	result, _ := args.Get(0).(*capi.Job)

	return result, nil
}

func (m *MockAppsClient) ClearBuildpackCache(ctx context.Context, guid string) error {
	args := m.Called(ctx, guid)

	err := args.Error(0)
	if err != nil {
		return fmt.Errorf("clear buildpack cache failed: %w", err)
	}

	return nil
}

func (m *MockAppsClient) GetManifest(ctx context.Context, guid string) (string, error) {
	args := m.Called(ctx, guid)

	err := args.Error(1)
	if err != nil {
		return "", fmt.Errorf("get manifest failed: %w", err)
	}

	return args.String(0), nil
}

func (m *MockAppsClient) GetRecentLogs(ctx context.Context, guid string, lines int) (*capi.AppLogs, error) {
	args := m.Called(ctx, guid, lines)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("get recent logs failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		return nil, fmt.Errorf("get recent logs failed: %w", err)
	}

	result, _ := args.Get(0).(*capi.AppLogs)

	return result, nil
}

func (m *MockAppsClient) StreamLogs(ctx context.Context, guid string) (<-chan capi.LogMessage, error) {
	args := m.Called(ctx, guid)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("stream logs failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		return nil, fmt.Errorf("stream logs failed: %w", err)
	}

	result, _ := args.Get(0).(<-chan capi.LogMessage)

	return result, nil
}

func (m *MockAppsClient) GetFeatures(ctx context.Context, guid string) (*capi.AppFeatures, error) {
	args := m.Called(ctx, guid)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("get features failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		return nil, fmt.Errorf("get features failed: %w", err)
	}

	result, _ := args.Get(0).(*capi.AppFeatures)

	return result, nil
}

func (m *MockAppsClient) GetFeature(ctx context.Context, guid, featureName string) (*capi.AppFeature, error) {
	args := m.Called(ctx, guid, featureName)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("get feature failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		return nil, fmt.Errorf("get feature failed: %w", err)
	}

	result, _ := args.Get(0).(*capi.AppFeature)

	return result, nil
}

func (m *MockAppsClient) UpdateFeature(ctx context.Context, guid, featureName string, request *capi.AppFeatureUpdateRequest) (*capi.AppFeature, error) {
	args := m.Called(ctx, guid, featureName, request)
	if args.Get(0) == nil {
		err := args.Error(1)
		if err != nil {
			return nil, fmt.Errorf("update feature failed: %w", err)
		}

		return nil, nil
	}

	err := args.Error(1)
	if err != nil {
		return nil, fmt.Errorf("update feature failed: %w", err)
	}

	result, _ := args.Get(0).(*capi.AppFeature)

	return result, nil
}

func TestBatchExecutor_Execute(t *testing.T) {
	t.Parallel()

	mockClient := &MockClient{}
	mockApps := &MockAppsClient{}
	mockClient.On("Apps").Return(mockApps)

	executor := capi.NewBatchExecutor(mockClient, 2)
	ctx := context.Background()

	// Set up mock expectations
	app1 := &capi.App{
		Resource: capi.Resource{GUID: "app-1"},
		Name:     "Test App 1",
	}
	app2 := &capi.App{
		Resource: capi.Resource{GUID: "app-2"},
		Name:     "Test App 2",
	}

	mockApps.On("Get", mock.Anything, "app-guid-1").Return(app1, nil)
	mockApps.On("Get", mock.Anything, "app-guid-2").Return(app2, nil)

	operations := []capi.BatchOperation{
		{
			ID:       "op1",
			Type:     "get",
			Resource: "app",
			Data:     "app-guid-1",
		},
		{
			ID:       "op2",
			Type:     "get",
			Resource: "app",
			Data:     "app-guid-2",
		},
	}

	results, err := executor.Execute(ctx, operations)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Check results
	for _, result := range results {
		assert.True(t, result.Success)
		require.NoError(t, result.Error)
		assert.NotNil(t, result.Data)
		assert.Positive(t, result.Duration)
	}

	mockClient.AssertExpectations(t)
	mockApps.AssertExpectations(t)
}

func TestBatchExecutor_WithCallback(t *testing.T) {
	t.Parallel()

	mockClient := &MockClient{}
	mockApps := &MockAppsClient{}
	mockClient.On("Apps").Return(mockApps)

	executor := capi.NewBatchExecutor(mockClient, 1)
	ctx := context.Background()

	app := &capi.App{
		Resource: capi.Resource{GUID: "app-1"},
		Name:     "Test App",
	}
	mockApps.On("Get", mock.Anything, "app-guid").Return(app, nil)

	var (
		callbackCalled bool
		callbackResult *capi.BatchResult
	)

	operation := capi.BatchOperation{
		ID:       "op1",
		Type:     "get",
		Resource: "app",
		Data:     "app-guid",
		Callback: func(result *capi.BatchResult) {
			callbackCalled = true
			callbackResult = result
		},
	}

	_, err := executor.Execute(ctx, []capi.BatchOperation{operation})
	require.NoError(t, err)

	assert.True(t, callbackCalled)
	assert.NotNil(t, callbackResult)
	assert.True(t, callbackResult.Success)
	assert.Equal(t, "op1", callbackResult.ID)

	mockClient.AssertExpectations(t)
	mockApps.AssertExpectations(t)
}

func TestBatchExecutor_WithError(t *testing.T) {
	t.Parallel()

	mockClient := &MockClient{}
	mockApps := &MockAppsClient{}
	mockClient.On("Apps").Return(mockApps)

	executor := capi.NewBatchExecutor(mockClient, 1)
	ctx := context.Background()

	mockApps.On("Get", mock.Anything, "app-guid").Return(nil, capi.ErrAppNotFound)

	operation := capi.BatchOperation{
		ID:       "op1",
		Type:     "get",
		Resource: "app",
		Data:     "app-guid",
	}

	results, err := executor.Execute(ctx, []capi.BatchOperation{operation})
	require.NoError(t, err) // Execute itself shouldn't fail
	assert.Len(t, results, 1)

	result := results[0]
	assert.False(t, result.Success)
	require.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "app not found")

	mockClient.AssertExpectations(t)
	mockApps.AssertExpectations(t)
}

func TestBatchBuilder(t *testing.T) {
	t.Parallel()

	builder := capi.NewBatchBuilder()

	req1 := &capi.AppCreateRequest{
		Name: "app1",
	}
	name := "updated-app"
	req2 := &capi.AppUpdateRequest{
		Name: &name,
	}

	builder.
		AddCreateApp("create-1", req1).
		AddUpdateApp("update-1", "app-guid", req2).
		AddDeleteApp("delete-1", "app-to-delete").
		AddGetApp("get-1", "app-to-get")

	operations := builder.Build()
	assert.Len(t, operations, 4)

	assert.Equal(t, "create-1", operations[0].ID)
	assert.Equal(t, "create", operations[0].Type)
	assert.Equal(t, "app", operations[0].Resource)

	assert.Equal(t, "update-1", operations[1].ID)
	assert.Equal(t, "update", operations[1].Type)

	assert.Equal(t, "delete-1", operations[2].ID)
	assert.Equal(t, "delete", operations[2].Type)

	assert.Equal(t, "get-1", operations[3].ID)
	assert.Equal(t, "get", operations[3].Type)
}

func TestBatchExecutor_Timeout(t *testing.T) {
	t.Parallel()

	mockClient := &MockClient{}
	executor := capi.NewBatchExecutor(mockClient, 1)
	executor.SetTimeout(1 * time.Millisecond)

	// Create an operation that will timeout
	operation := capi.BatchOperation{
		ID:       "op1",
		Type:     "get",
		Resource: "unsupported", // This will cause an error in the executor
		Data:     "test",
	}

	ctx := context.Background()
	results, err := executor.Execute(ctx, []capi.BatchOperation{operation})
	require.NoError(t, err)
	assert.Len(t, results, 1)

	result := results[0]
	assert.False(t, result.Success)
	require.Error(t, result.Error)
}
