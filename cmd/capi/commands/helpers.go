package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Common string constants used throughout the commands package.
const (
	NotAvailable = "N/A"

	// Output formats.
	OutputFormatJSON = "json"
	OutputFormatYAML = "yaml"

	// JSON formatting.
	defaultJSONIndent = 2

	// Common values.
	Yes          = "yes"
	True         = "true"
	False        = "false"
	Ready        = "ready"
	UserProvided = "user-provided"
	Unlimited    = "unlimited"
	Unknown      = "unknown"
	Partial      = "partial"
	Compatible   = "compatible"
	Incompatible = "incompatible"
	List         = "list"
	Create       = "create"
	Update       = "update"
	Delete       = "delete"
	Desc         = "desc"
	Descending   = "descending"
	OrgGUID      = "org_guid"
	Masked       = "***"

	// Test constants.
	CFLinuxFS4Stack       = "cflinuxfs4"
	RubyBuildpackFilename = "ruby_buildpack-v1.0.0.zip"
)

// Common static errors used throughout the commands package.
var (
	ErrApplicationNotFound           = errors.New("application not found")
	ErrAPINotFound                   = errors.New("API not found")
	ErrOrganizationNotFound          = errors.New("organization not found")
	ErrDomainNotFound                = errors.New("domain not found")
	ErrSpaceNotFound                 = errors.New("space not found")
	ErrIsolationSegmentNotFound      = errors.New("isolation segment not found")
	ErrOrganizationQuotaNotFound     = errors.New("organization quota not found")
	ErrSpaceQuotaNotFound            = errors.New("space quota not found")
	ErrAPIConfigNotFound             = errors.New("API configuration not found")
	ErrSecurityGroupNotFound         = errors.New("security group not found")
	ErrRouteNotFound                 = errors.New("route not found")
	ErrServiceNotFound               = errors.New("service not found")
	ErrServiceInstanceNotFound       = errors.New("service instance not found")
	ErrServiceOfferingNotFound       = errors.New("service offering not found")
	ErrServicePlanNotFound           = errors.New("service plan not found")
	ErrServiceBrokerNotFound         = errors.New("service broker not found")
	ErrUserNotFound                  = errors.New("user not found")
	ErrDomainNameRequired            = errors.New("domain name is required")
	ErrOrganizationNameRequired      = errors.New("organization name is required")
	ErrIsolationSegmentNameRequired  = errors.New("isolation segment name is required")
	ErrQuotaNameRequired             = errors.New("quota name is required")
	ErrUserGUIDRequired              = errors.New("user GUID is required")
	ErrAtLeastOneOrgRequired         = errors.New("at least one organization must be specified")
	ErrAtLeastOneOrgNameRequired     = errors.New("at least one organization name is required")
	ErrNoSpaceSpecifiedAndTargeted   = errors.New("no space specified and no space targeted")
	ErrInvalidGroupName              = errors.New("invalid group name")
	ErrInvalidEnabledFlag            = errors.New("enabled flag must be 'true' or 'false'")
	ErrInvalidEnvVarFormat           = errors.New("invalid environment variable format")
	ErrInvalidEnvFileFormat          = errors.New("invalid .env format")
	ErrDirectoryTraversalDetected    = errors.New("path contains directory traversal sequences")
	ErrJobCompletedWithErrors        = errors.New("job completed with errors")
	ErrAPIEndpointRequired           = errors.New("API endpoint is required")
	ErrSecurityGroupNameRequired     = errors.New("security group name is required")
	ErrAtLeastOneSpaceRequired       = errors.New("at least one space must be specified")
	ErrMustSpecifyRunningOrStaging   = errors.New("must specify --running or --staging (or both)")
	ErrSpaceNameRequired             = errors.New("space name is required")
	ErrServiceInstanceNameRequired   = errors.New("service instance name is required")
	ErrServicePlanRequiredForManaged = errors.New("service plan is required for managed services")
	ErrSpaceRequired                 = errors.New("space is required (use --space or target a space)")
	ErrVisibilityTypeRequired        = errors.New("visibility type is required (--type)")
	ErrOrganizationRequired          = errors.New("organization is required (use --org)")
	ErrOrganizationMustBeTargeted    = errors.New("organization must be targeted first")
	ErrNoResourcesSpecified          = errors.New("no resources specified. Use --from-file or --resource flags")
	ErrInvalidResourceFormat         = errors.New("invalid format. Expected sha1:size:path:mode")
	ErrFailedToParseResourceFile     = errors.New("failed to parse file as JSON or YAML resource list")
	ErrBindingNotFound               = errors.New("binding not found")
	ErrNoUAAEndpoint                 = errors.New("no UAA endpoint configured")
	ErrNotAuthenticated              = errors.New("not authenticated")
	ErrNotImplemented                = errors.New("not implemented yet")
)

// AppLimitsConfig defines the interface for app limit configurations used by quota commands.
type AppLimitsConfig interface {
	GetTotalMemoryInMB() int
	GetPerProcessMemoryInMB() int
	GetTotalInstances() int
	GetPerAppTasks() int
	GetLogRateLimitInBytesPerSecond() int
}

// AppLimitsBuilder is a generic interface for building app limit structures.
type AppLimitsBuilder[T any] interface {
	Build() *T
	SetTotalMemoryInMB(value *int) AppLimitsBuilder[T]
	SetPerProcessMemoryInMB(value *int) AppLimitsBuilder[T]
	SetTotalInstances(value *int) AppLimitsBuilder[T]
	SetPerAppTasks(value *int) AppLimitsBuilder[T]
	SetLogRateLimitInBytesPerSecond(value *int) AppLimitsBuilder[T]
}

// OrganizationQuotaAppsBuilder implements AppLimitsBuilder for OrganizationQuotaApps.
type OrganizationQuotaAppsBuilder struct {
	apps *capi.OrganizationQuotaApps
}

func (b *OrganizationQuotaAppsBuilder) Build() *capi.OrganizationQuotaApps {
	return b.apps
}

func (b *OrganizationQuotaAppsBuilder) SetTotalMemoryInMB(value *int) AppLimitsBuilder[capi.OrganizationQuotaApps] {
	b.apps.TotalMemoryInMB = value

	return b
}

func (b *OrganizationQuotaAppsBuilder) SetPerProcessMemoryInMB(value *int) AppLimitsBuilder[capi.OrganizationQuotaApps] {
	b.apps.PerProcessMemoryInMB = value

	return b
}

func (b *OrganizationQuotaAppsBuilder) SetTotalInstances(value *int) AppLimitsBuilder[capi.OrganizationQuotaApps] {
	b.apps.TotalInstances = value

	return b
}

func (b *OrganizationQuotaAppsBuilder) SetPerAppTasks(value *int) AppLimitsBuilder[capi.OrganizationQuotaApps] {
	b.apps.PerAppTasks = value

	return b
}

func (b *OrganizationQuotaAppsBuilder) SetLogRateLimitInBytesPerSecond(value *int) AppLimitsBuilder[capi.OrganizationQuotaApps] {
	b.apps.LogRateLimitInBytesPerSecond = value

	return b
}

// SpaceQuotaAppsBuilder implements AppLimitsBuilder for SpaceQuotaApps.
type SpaceQuotaAppsBuilder struct {
	apps *capi.SpaceQuotaApps
}

func (b *SpaceQuotaAppsBuilder) Build() *capi.SpaceQuotaApps {
	return b.apps
}

func (b *SpaceQuotaAppsBuilder) SetTotalMemoryInMB(value *int) AppLimitsBuilder[capi.SpaceQuotaApps] {
	b.apps.TotalMemoryInMB = value

	return b
}

func (b *SpaceQuotaAppsBuilder) SetPerProcessMemoryInMB(value *int) AppLimitsBuilder[capi.SpaceQuotaApps] {
	b.apps.PerProcessMemoryInMB = value

	return b
}

func (b *SpaceQuotaAppsBuilder) SetTotalInstances(value *int) AppLimitsBuilder[capi.SpaceQuotaApps] {
	b.apps.TotalInstances = value

	return b
}

func (b *SpaceQuotaAppsBuilder) SetPerAppTasks(value *int) AppLimitsBuilder[capi.SpaceQuotaApps] {
	b.apps.PerAppTasks = value

	return b
}

func (b *SpaceQuotaAppsBuilder) SetLogRateLimitInBytesPerSecond(value *int) AppLimitsBuilder[capi.SpaceQuotaApps] {
	b.apps.LogRateLimitInBytesPerSecond = value

	return b
}

// buildAppLimitsGeneric creates app limits using the provided builder.
func buildAppLimitsGeneric[T any](cmd *cobra.Command, config AppLimitsConfig, builder AppLimitsBuilder[T]) *T {
	if !cmd.Flags().Changed("total-memory") && !cmd.Flags().Changed("instance-memory") &&
		!cmd.Flags().Changed("instances") && !cmd.Flags().Changed("app-tasks") &&
		!cmd.Flags().Changed("log-rate-limit") {
		return nil
	}

	if cmd.Flags().Changed("total-memory") {
		totalMemory := config.GetTotalMemoryInMB()
		builder.SetTotalMemoryInMB(&totalMemory)
	}

	if cmd.Flags().Changed("instance-memory") {
		instanceMemory := config.GetPerProcessMemoryInMB()
		builder.SetPerProcessMemoryInMB(&instanceMemory)
	}

	if cmd.Flags().Changed("instances") {
		instances := config.GetTotalInstances()
		builder.SetTotalInstances(&instances)
	}

	if cmd.Flags().Changed("app-tasks") {
		appTasks := config.GetPerAppTasks()
		builder.SetPerAppTasks(&appTasks)
	}

	if cmd.Flags().Changed("log-rate-limit") {
		logRateLimit := config.GetLogRateLimitInBytesPerSecond()
		builder.SetLogRateLimitInBytesPerSecond(&logRateLimit)
	}

	return builder.Build()
}

// BuildOrganizationQuotaApps builds OrganizationQuotaApps from command flags.
func BuildOrganizationQuotaApps(cmd *cobra.Command, config AppLimitsConfig) *capi.OrganizationQuotaApps {
	builder := &OrganizationQuotaAppsBuilder{apps: &capi.OrganizationQuotaApps{}}

	return buildAppLimitsGeneric(cmd, config, builder)
}

// BuildSpaceQuotaApps builds SpaceQuotaApps from command flags.
func BuildSpaceQuotaApps(cmd *cobra.Command, config AppLimitsConfig) *capi.SpaceQuotaApps {
	builder := &SpaceQuotaAppsBuilder{apps: &capi.SpaceQuotaApps{}}

	return buildAppLimitsGeneric(cmd, config, builder)
}

// PurgeReseedConfig holds the configuration for purge and reseed operations.
type PurgeReseedConfig struct {
	EntityType       string // e.g., "app usage events", "service usage events"
	EntityTypePlural string // e.g., "applications", "service instances"
	PurgeFunc        func(ctx context.Context, client interface{}) error
}

// createPurgeAndReseedCommand creates a generic purge and reseed command.
func createPurgeAndReseedCommand(config PurgeReseedConfig) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "purge-and-reseed",
		Short: "Purge and reseed " + config.EntityType,
		Long:  fmt.Sprintf("Purge existing %s and reseed with current state", config.EntityType),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				_, _ = fmt.Fprintf(os.Stdout, "This will purge all existing %s and reseed with current state.\n", config.EntityType)
				_, _ = os.Stdout.WriteString("This action cannot be undone. Continue? (y/N): ")

				var response string

				_, _ = fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					_, _ = os.Stdout.WriteString("Cancelled\n")

					return nil
				}
			}

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			_, _ = fmt.Fprintf(os.Stdout, "Purging and reseeding %s...\n", config.EntityType)
			start := time.Now()

			err = config.PurgeFunc(ctx, client)
			if err != nil {
				return fmt.Errorf("failed to purge and reseed %s: %w", config.EntityType, err)
			}

			duration := time.Since(start)
			_, _ = fmt.Fprintf(os.Stdout, "Successfully purged and reseeded %s in %v\n", config.EntityType, duration)
			_, _ = fmt.Fprintf(os.Stdout, "New events will reflect the current state of all %s\n", config.EntityTypePlural)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")

	return cmd
}

// AppListConfig holds the configuration for app list commands (tasks, droplets, builds).
type AppListConfig struct {
	Use           string // e.g., "tasks [APP_NAME_OR_GUID]"
	Short         string // e.g., "List application tasks"
	Long          string // e.g., "List all tasks for an application"
	StateFilter   string // e.g., "task state", "droplet state", "build state"
	SetupParams   func(ctx context.Context, client interface{}, appNameOrGUID string, allPages bool, perPage int, state string) (interface{}, error)
	FetchPages    func(ctx context.Context, client interface{}, params interface{}, allPages bool) (interface{}, interface{}, error)
	OutputResults func(results interface{}, pagination interface{}, allPages bool) error
}

// DeleteConfig holds the configuration for delete commands.
type DeleteConfig struct {
	Use         string // e.g., "delete BUILDPACK_NAME_OR_GUID"
	Short       string // e.g., "Delete a buildpack"
	Long        string // e.g., "Delete a buildpack"
	EntityType  string // e.g., "buildpack", "domain", "security group"
	GetResource func(ctx context.Context, client interface{}, nameOrGUID string) (guid string, name string, err error)
	DeleteFunc  func(ctx context.Context, client interface{}, guid string) (jobGUID *string, err error)
}

// DeleteResourceFunc represents a function that gets a resource by name or GUID.
type DeleteResourceFunc func(ctx context.Context, client interface{}, nameOrGUID string) (guid string, name string, err error)

// CreateOrganizationDeleteResourceFunc creates a GetResource function for organizations.
func CreateOrganizationDeleteResourceFunc() DeleteResourceFunc {
	return func(ctx context.Context, client interface{}, nameOrGUID string) (string, string, error) {
		capiClient, ok := client.(capi.Client)
		if !ok {
			return "", "", constants.ErrClientNotCAPIClient
		}

		orgsClient := capiClient.Organizations()

		// Try to get by GUID first
		org, err := orgsClient.Get(ctx, nameOrGUID)
		if err == nil {
			return org.GUID, org.Name, nil
		}

		// Try by name
		params := capi.NewQueryParams()
		params.WithFilter("names", nameOrGUID)

		orgs, err := orgsClient.List(ctx, params)
		if err != nil {
			return "", "", fmt.Errorf("failed to find organization: %w", err)
		}

		if len(orgs.Resources) == 0 {
			return "", "", fmt.Errorf("organization '%s': %w", nameOrGUID, ErrOrganizationNotFound)
		}

		return orgs.Resources[0].GUID, orgs.Resources[0].Name, nil
	}
}

// CreateSpaceDeleteResourceFunc creates a GetResource function for spaces.
func CreateSpaceDeleteResourceFunc() DeleteResourceFunc {
	return func(ctx context.Context, client interface{}, nameOrGUID string) (string, string, error) {
		capiClient, ok := client.(capi.Client)
		if !ok {
			return "", "", constants.ErrClientNotCAPIClient
		}

		spacesClient := capiClient.Spaces()

		// Try to get by GUID first
		space, err := spacesClient.Get(ctx, nameOrGUID)
		if err == nil {
			return space.GUID, space.Name, nil
		}

		// Try by name
		params := capi.NewQueryParams()
		params.WithFilter("names", nameOrGUID)

		spaces, err := spacesClient.List(ctx, params)
		if err != nil {
			return "", "", fmt.Errorf("failed to find space: %w", err)
		}

		if len(spaces.Resources) == 0 {
			return "", "", fmt.Errorf("space '%s': %w", nameOrGUID, ErrSpaceNotFound)
		}

		return spaces.Resources[0].GUID, spaces.Resources[0].Name, nil
	}
}

// CreateDomainDeleteResourceFunc creates a GetResource function for domains.
func CreateDomainDeleteResourceFunc() DeleteResourceFunc {
	return func(ctx context.Context, client interface{}, nameOrGUID string) (string, string, error) {
		capiClient, ok := client.(capi.Client)
		if !ok {
			return "", "", constants.ErrClientNotCAPIClient
		}

		domainsClient := capiClient.Domains()

		// Try to get by GUID first
		domain, err := domainsClient.Get(ctx, nameOrGUID)
		if err == nil {
			return domain.GUID, domain.Name, nil
		}

		// Try by name
		params := capi.NewQueryParams()
		params.WithFilter("names", nameOrGUID)

		domains, err := domainsClient.List(ctx, params)
		if err != nil {
			return "", "", fmt.Errorf("failed to find domain: %w", err)
		}

		if len(domains.Resources) == 0 {
			return "", "", fmt.Errorf("domain '%s': %w", nameOrGUID, ErrDomainNotFound)
		}

		return domains.Resources[0].GUID, domains.Resources[0].Name, nil
	}
}

// CreateSecurityGroupDeleteResourceFunc creates a GetResource function for security groups.
func CreateSecurityGroupDeleteResourceFunc() DeleteResourceFunc {
	return func(ctx context.Context, client interface{}, nameOrGUID string) (string, string, error) {
		capiClient, ok := client.(capi.Client)
		if !ok {
			return "", "", constants.ErrClientNotCAPIClient
		}

		securityGroupsClient := capiClient.SecurityGroups()

		// Try to get by GUID first
		securityGroup, err := securityGroupsClient.Get(ctx, nameOrGUID)
		if err == nil {
			return securityGroup.GUID, securityGroup.Name, nil
		}

		// Try by name
		params := capi.NewQueryParams()
		params.WithFilter("names", nameOrGUID)

		groups, err := securityGroupsClient.List(ctx, params)
		if err != nil {
			return "", "", fmt.Errorf("failed to find security group: %w", err)
		}

		if len(groups.Resources) == 0 {
			return "", "", fmt.Errorf("security group '%s': %w", nameOrGUID, ErrSecurityGroupNotFound)
		}

		return groups.Resources[0].GUID, groups.Resources[0].Name, nil
	}
}

// UpdateConfig holds the configuration for update commands.
type UpdateConfig struct {
	Use         string // e.g., "update ISOLATION_SEGMENT_NAME_OR_GUID"
	Short       string // e.g., "Update an isolation segment"
	Long        string // e.g., "Update an existing Cloud Foundry isolation segment"
	EntityType  string // e.g., "isolation segment", "space"
	GetResource func(ctx context.Context, client interface{}, nameOrGUID string) (guid string, name string, err error)
	UpdateFunc  func(ctx context.Context, client interface{}, guid, newName string, labels map[string]string) (updatedName string, err error)
}

// createUpdateCommand creates a generic update command.
func createUpdateCommand(config UpdateConfig) *cobra.Command {
	var (
		newName string
		labels  map[string]string
	)

	cmd := &cobra.Command{
		Use:   config.Use,
		Short: config.Short,
		Long:  config.Long,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find resource using the provided function
			resourceGUID, _, err := config.GetResource(ctx, client, nameOrGUID)
			if err != nil {
				return err
			}

			// Update resource using the provided function
			updatedName, err := config.UpdateFunc(ctx, client, resourceGUID, newName, labels)
			if err != nil {
				return fmt.Errorf("failed to update %s: %w", config.EntityType, err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully updated %s '%s'\n", config.EntityType, updatedName)

			return nil
		},
	}

	cmd.Flags().StringVar(&newName, "name", "", fmt.Sprintf("new %s name", config.EntityType))
	cmd.Flags().StringToStringVar(&labels, "labels", nil, "labels to apply (key=value)")

	return cmd
}

// CreateIsolationSegmentUpdateResourceFunc creates a GetResource function for isolation segments.
func CreateIsolationSegmentUpdateResourceFunc() DeleteResourceFunc {
	return func(ctx context.Context, client interface{}, nameOrGUID string) (string, string, error) {
		capiClient, ok := client.(capi.Client)
		if !ok {
			return "", "", constants.ErrClientNotCAPIClient
		}

		segmentsClient := capiClient.IsolationSegments()

		// Try to get by GUID first
		segment, err := segmentsClient.Get(ctx, nameOrGUID)
		if err == nil {
			return segment.GUID, segment.Name, nil
		}

		// Try by name
		params := capi.NewQueryParams()
		params.WithFilter("names", nameOrGUID)

		segments, err := segmentsClient.List(ctx, params)
		if err != nil {
			return "", "", fmt.Errorf("failed to find isolation segment: %w", err)
		}

		if len(segments.Resources) == 0 {
			return "", "", fmt.Errorf("isolation segment '%s': %w", nameOrGUID, ErrIsolationSegmentNotFound)
		}

		return segments.Resources[0].GUID, segments.Resources[0].Name, nil
	}
}

// CreateSpaceUpdateResourceFunc creates a GetResource function for spaces.
func CreateSpaceUpdateResourceFunc() DeleteResourceFunc {
	return func(ctx context.Context, client interface{}, nameOrGUID string) (string, string, error) {
		capiClient, ok := client.(capi.Client)
		if !ok {
			return "", "", constants.ErrClientNotCAPIClient
		}

		spacesClient := capiClient.Spaces()

		// Try to get by GUID first
		space, err := spacesClient.Get(ctx, nameOrGUID)
		if err == nil {
			return space.GUID, space.Name, nil
		}

		// Try by name
		params := capi.NewQueryParams()
		params.WithFilter("names", nameOrGUID)

		spaces, err := spacesClient.List(ctx, params)
		if err != nil {
			return "", "", fmt.Errorf("failed to find space: %w", err)
		}

		if len(spaces.Resources) == 0 {
			return "", "", fmt.Errorf("space '%s': %w", nameOrGUID, ErrSpaceNotFound)
		}

		return spaces.Resources[0].GUID, spaces.Resources[0].Name, nil
	}
}

// CreateIsolationSegmentDeleteResourceFunc creates a GetResource function for isolation segments.
func CreateIsolationSegmentDeleteResourceFunc() DeleteResourceFunc {
	return func(ctx context.Context, client interface{}, nameOrGUID string) (string, string, error) {
		capiClient, ok := client.(capi.Client)
		if !ok {
			return "", "", constants.ErrClientNotCAPIClient
		}

		segmentsClient := capiClient.IsolationSegments()

		// Try to get by GUID first
		segment, err := segmentsClient.Get(ctx, nameOrGUID)
		if err == nil {
			return segment.GUID, segment.Name, nil
		}

		// Try by name
		params := capi.NewQueryParams()
		params.WithFilter("names", nameOrGUID)

		segments, err := segmentsClient.List(ctx, params)
		if err != nil {
			return "", "", fmt.Errorf("failed to find isolation segment: %w", err)
		}

		if len(segments.Resources) == 0 {
			return "", "", fmt.Errorf("isolation segment '%s': %w", nameOrGUID, ErrIsolationSegmentNotFound)
		}

		return segments.Resources[0].GUID, segments.Resources[0].Name, nil
	}
}

// CreateOrganizationQuotaDeleteResourceFunc creates a GetResource function for organization quotas.
func CreateOrganizationQuotaDeleteResourceFunc() DeleteResourceFunc {
	return func(ctx context.Context, client interface{}, nameOrGUID string) (string, string, error) {
		capiClient, ok := client.(capi.Client)
		if !ok {
			return "", "", constants.ErrClientNotCAPIClient
		}

		quotaClient := capiClient.OrganizationQuotas()

		// Try to get by GUID first
		quota, err := quotaClient.Get(ctx, nameOrGUID)
		if err == nil {
			return quota.GUID, quota.Name, nil
		}

		// Try by name
		params := capi.NewQueryParams()
		params.WithFilter("names", nameOrGUID)

		quotas, err := quotaClient.List(ctx, params)
		if err != nil {
			return "", "", fmt.Errorf("failed to find organization quota: %w", err)
		}

		if len(quotas.Resources) == 0 {
			return "", "", fmt.Errorf("organization quota '%s': %w", nameOrGUID, ErrOrganizationQuotaNotFound)
		}

		return quotas.Resources[0].GUID, quotas.Resources[0].Name, nil
	}
}

// CreateSpaceQuotaDeleteResourceFunc creates a GetResource function for space quotas.
func CreateSpaceQuotaDeleteResourceFunc() DeleteResourceFunc {
	return func(ctx context.Context, client interface{}, nameOrGUID string) (string, string, error) {
		capiClient, ok := client.(capi.Client)
		if !ok {
			return "", "", constants.ErrClientNotCAPIClient
		}

		quotaClient := capiClient.SpaceQuotas()

		// Try to get by GUID first
		quota, err := quotaClient.Get(ctx, nameOrGUID)
		if err == nil {
			return quota.GUID, quota.Name, nil
		}

		// Try by name
		params := capi.NewQueryParams()
		params.WithFilter("names", nameOrGUID)

		quotas, err := quotaClient.List(ctx, params)
		if err != nil {
			return "", "", fmt.Errorf("failed to find space quota: %w", err)
		}

		if len(quotas.Resources) == 0 {
			return "", "", fmt.Errorf("space quota '%s': %w", nameOrGUID, ErrSpaceQuotaNotFound)
		}

		return quotas.Resources[0].GUID, quotas.Resources[0].Name, nil
	}
}

// createCommandWrapper creates a wrapper function that modifies the Use field of an existing command.
func createCommandWrapper(originalCommandFunc func() *cobra.Command, newUse string) func() *cobra.Command {
	return func() *cobra.Command {
		cmd := originalCommandFunc()
		cmd.Use = newUse

		return cmd
	}
}

// CommandConfig holds configuration for creating UAA subcommand groups.
type CommandConfig struct {
	Use         string
	Short       string
	Long        string
	Example     string
	SubCommands []SubCommandConfig
}

// SubCommandConfig holds configuration for individual subcommands.
type SubCommandConfig struct {
	Name        string
	CommandFunc interface{}
	Use         string
}

// CreateUAASubCommandGroup creates a UAA subcommand group with the given configuration.
func CreateUAASubCommandGroup(config CommandConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     config.Use,
		Short:   config.Short,
		Long:    config.Long,
		Example: config.Example,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Add subcommands
	for _, subCmd := range config.SubCommands {
		if cmdFunc, ok := subCmd.CommandFunc.(func() *cobra.Command); ok {
			cmd.AddCommand(createCommandWrapper(cmdFunc, subCmd.Use)())
		}
	}

	return cmd
}

// CreateGenericDeleteFunc creates a generic DeleteFunc for the DeleteConfig.
func CreateGenericDeleteFunc(deleteMethod func(ctx context.Context, guid string) (*capi.Job, error)) func(ctx context.Context, client interface{}, guid string) (*string, error) {
	return func(ctx context.Context, client interface{}, guid string) (*string, error) {
		job, err := deleteMethod(ctx, guid)
		if err != nil {
			return nil, err
		}

		if job != nil {
			return &job.GUID, nil
		}

		return nil, nil
	}
}

// createDeleteCommand creates a generic delete command.
func createDeleteCommand(config DeleteConfig) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   config.Use,
		Short: config.Short,
		Long:  config.Long,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrGUID := args[0]

			if !force {
				_, _ = fmt.Fprintf(os.Stdout, "Really delete %s '%s'? (y/N): ", config.EntityType, nameOrGUID)

				var response string

				_, _ = fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					_, _ = os.Stdout.WriteString("Cancelled\n")

					return nil
				}
			}

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find and delete resource using the provided functions
			resourceGUID, resourceName, err := config.GetResource(ctx, client, nameOrGUID)
			if err != nil {
				return err
			}

			jobGUID, err := config.DeleteFunc(ctx, client, resourceGUID)
			if err != nil {
				return fmt.Errorf("failed to delete %s: %w", config.EntityType, err)
			}

			if jobGUID != nil {
				_, _ = fmt.Fprintf(os.Stdout, "Deleting %s '%s'... (job: %s)\n", config.EntityType, resourceName, *jobGUID)
				_, _ = fmt.Fprintf(os.Stdout, "Monitor with: capi jobs get %s\n", *jobGUID)
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "Successfully deleted %s '%s'\n", config.EntityType, resourceName)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "force deletion without confirmation")

	return cmd
}

// createBuildpackDeleteCommand creates a delete command for buildpacks.
func createBuildpackDeleteCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete BUILDPACK_NAME_OR_GUID",
		Short: "Delete a buildpack",
		Long:  "Delete a buildpack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeDeleteCommand("buildpack", args[0], force, cmd,
				func(ctx context.Context, client interface{}, nameOrGUID string) (string, string, error) {
					// Find buildpack - this will need to be updated with proper types
					return "", "", ErrNotImplemented
				},
				func(ctx context.Context, client interface{}, guid string) (*string, error) {
					// Delete buildpack - this will need to be updated with proper types
					return nil, ErrNotImplemented
				},
			)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "force deletion without confirmation")

	return cmd
}

// executeDeleteCommand handles the common delete command logic.
func executeDeleteCommand(
	entityType, nameOrGUID string, force bool, cmd *cobra.Command,
	getResource func(context.Context, interface{}, string) (string, string, error),
	deleteFunc func(context.Context, interface{}, string) (*string, error),
) error {
	if !force {
		_, _ = fmt.Fprintf(os.Stdout, "Really delete %s '%s'? (y/N): ", entityType, nameOrGUID)

		var response string

		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			_, _ = os.Stdout.WriteString("Cancelled\n")

			return nil
		}
	}

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	resourceGUID, resourceName, err := getResource(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	jobGUID, err := deleteFunc(ctx, client, resourceGUID)
	if err != nil {
		return fmt.Errorf("failed to delete %s: %w", entityType, err)
	}

	if jobGUID != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Deleting %s '%s'... (job: %s)\n", entityType, resourceName, *jobGUID)
		_, _ = fmt.Fprintf(os.Stdout, "Monitor with: capi jobs get %s\n", *jobGUID)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "Successfully deleted %s '%s'\n", entityType, resourceName)
	}

	return nil
}

// createFeatureToggleCommand creates enable/disable commands for app features.
func createFeatureToggleCommand(use, short, long string, enabled bool, successMessage string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Args:  cobra.ExactArgs(constants.TwoArgumentsMax),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFeatureToggle(cmd, args, enabled, successMessage)
		},
	}
}

func runFeatureToggle(cmd *cobra.Command, args []string, enabled bool, successMessage string) error {
	appNameOrGUID := args[0]
	featureName := args[1]

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	app, err := findAppForFeatureToggle(ctx, client, appNameOrGUID)
	if err != nil {
		return err
	}

	request := &capi.AppFeatureUpdateRequest{
		Enabled: enabled,
	}

	updatedFeature, err := client.Apps().UpdateFeature(ctx, app.GUID, featureName, request)
	if err != nil {
		action := "enabling"
		if !enabled {
			action = "disabling"
		}

		return fmt.Errorf("%s app feature '%s': %w", action, featureName, err)
	}

	return outputFeatureToggleResult(updatedFeature, successMessage, featureName, app.Name)
}

func findAppForFeatureToggle(ctx context.Context, client capi.Client, appNameOrGUID string) (*capi.App, error) {
	app, err := client.Apps().Get(ctx, appNameOrGUID)
	if err != nil {
		params := capi.NewQueryParams()
		params.WithFilter("names", appNameOrGUID)

		if spaceGUID := viper.GetString("space_guid"); spaceGUID != "" {
			params.WithFilter("space_guids", spaceGUID)
		}

		apps, err := client.Apps().List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to find application: %w", err)
		}

		if len(apps.Resources) == 0 {
			return nil, fmt.Errorf("application '%s': %w", appNameOrGUID, ErrApplicationNotFound)
		}

		app = &apps.Resources[0]
	}

	return app, nil
}

func outputFeatureToggleResult(updatedFeature *capi.AppFeature, successMessage, featureName, appName string) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(updatedFeature)
		if err != nil {
			return fmt.Errorf("encoding feature to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(updatedFeature)
		if err != nil {
			return fmt.Errorf("encoding feature to YAML: %w", err)
		}

		return nil
	default:
		_, _ = fmt.Fprintf(os.Stdout, successMessage, featureName, appName)
	}

	return nil
}

// createSpaceFeatureToggleCommand creates enable/disable commands for space features.
func createSpaceFeatureToggleCommand(use, short, long string, enabled bool, successMessage string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Args:  cobra.ExactArgs(constants.TwoArgumentsMax),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSpaceFeatureToggle(cmd, args, enabled, successMessage)
		},
	}
}

func runSpaceFeatureToggle(cmd *cobra.Command, args []string, enabled bool, successMessage string) error {
	spaceNameOrGUID := args[0]
	featureName := args[1]

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	space, err := findSpaceForFeatureToggle(ctx, client, spaceNameOrGUID)
	if err != nil {
		return err
	}

	updatedFeature, err := client.Spaces().UpdateFeature(ctx, space.GUID, featureName, enabled)
	if err != nil {
		action := "enabling"
		if !enabled {
			action = "disabling"
		}

		return fmt.Errorf("%s space feature '%s': %w", action, featureName, err)
	}

	return outputSpaceFeatureToggleResult(updatedFeature, successMessage, featureName, space.Name)
}

func findSpaceForFeatureToggle(ctx context.Context, client capi.Client, spaceNameOrGUID string) (*capi.Space, error) {
	space, err := client.Spaces().Get(ctx, spaceNameOrGUID)
	if err != nil {
		params := capi.NewQueryParams()
		params.WithFilter("names", spaceNameOrGUID)

		if orgGUID := viper.GetString("organization_guid"); orgGUID != "" {
			params.WithFilter("organization_guids", orgGUID)
		}

		spaces, err := client.Spaces().List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to find space: %w", err)
		}

		if len(spaces.Resources) == 0 {
			return nil, fmt.Errorf("space '%s': %w", spaceNameOrGUID, ErrSpaceNotFound)
		}

		space = &spaces.Resources[0]
	}

	return space, nil
}

func outputSpaceFeatureToggleResult(updatedFeature *capi.SpaceFeature, successMessage, featureName, spaceName string) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(updatedFeature)
		if err != nil {
			return fmt.Errorf("encoding space feature to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(updatedFeature)
		if err != nil {
			return fmt.Errorf("encoding space feature to YAML: %w", err)
		}

		return nil
	default:
		_, _ = fmt.Fprintf(os.Stdout, successMessage, featureName, spaceName)
	}

	return nil
}

// createUserActivationCommand creates activate/deactivate commands for UAA users.
func createUserActivationCommand(use, short, long string, activate bool, successMessage string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config := loadConfig()
			username := args[0]

			if GetEffectiveUAAEndpoint(config) == "" {
				return fmt.Errorf("%w. Use 'capi uaa target <url>' to set one", ErrNoUAAEndpoint)
			}

			// Create UAA client
			uaaClient, err := NewUAAClient(config)
			if err != nil {
				return fmt.Errorf("failed to create UAA client: %w", err)
			}

			if !uaaClient.IsAuthenticated() {
				return fmt.Errorf("%w. Use a token command to authenticate first", ErrNotAuthenticated)
			}

			// Get user to get ID and version
			user, err := uaaClient.Client().GetUserByUsername(username, "", "")
			if err != nil {
				return fmt.Errorf("failed to get user: %w", err)
			}

			// Get version
			version := 0
			if user.Meta != nil {
				version = user.Meta.Version
			}

			// Activate or deactivate user
			if activate {
				err = uaaClient.Client().ActivateUser(user.ID, version)
			} else {
				err = uaaClient.Client().DeactivateUser(user.ID, version)
			}

			if err != nil {
				action := "activate"
				if !activate {
					action = "deactivate"
				}

				return fmt.Errorf("failed to %s user: %w", action, err)
			}

			_, _ = fmt.Fprintf(os.Stdout, successMessage, username)

			return nil
		},
	}
}

// CreateConfig holds the configuration for create commands.
type CreateConfig struct {
	Use        string // e.g., "create"
	Short      string // e.g., "Create an isolation segment"
	Long       string // e.g., "Create a new Cloud Foundry isolation segment"
	EntityType string // e.g., "isolation segment", "organization"
	NameError  error  // Error to return when name is empty
	CreateFunc func(ctx context.Context, client interface{}, name string, labels map[string]string) (guid string, displayName string, err error)
}

// createGenericCreateCommand creates a generic create command.
func createGenericCreateCommand(config CreateConfig) *cobra.Command {
	var (
		name   string
		labels map[string]string
	)

	cmd := &cobra.Command{
		Use:   config.Use,
		Short: config.Short,
		Long:  config.Long,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return config.NameError
			}

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			guid, displayName, err := config.CreateFunc(ctx, client, name, labels)
			if err != nil {
				return fmt.Errorf("failed to create %s: %w", config.EntityType, err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Successfully created %s '%s' with GUID %s\n", config.EntityType, displayName, guid)

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", config.EntityType+" name (required)")
	cmd.Flags().StringToStringVar(&labels, "labels", nil, "labels to apply (key=value)")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

// SecurityGroupListConfig holds the configuration for security group list commands.
type SecurityGroupListConfig struct {
	Use        string // e.g., "running"
	Short      string // e.g., "List running security groups"
	Long       string // e.g., "List all security groups that are globally enabled for running applications"
	FilterKey  string // e.g., "globally_enabled_running"
	NoItemsMsg string // e.g., "No globally enabled running security groups found"
	ListTitle  string // e.g., "Globally enabled running security groups:"
}

// createSecurityGroupListCommand creates a generic security group listing command.
func createSecurityGroupListCommand(config SecurityGroupListConfig) *cobra.Command {
	return &cobra.Command{
		Use:   config.Use,
		Short: config.Short,
		Long:  config.Long,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()
			params := capi.NewQueryParams()
			params.WithFilter(config.FilterKey, "true")

			securityGroups, err := client.SecurityGroups().List(ctx, params)
			if err != nil {
				return fmt.Errorf("failed to list %s security groups: %w", config.Use, err)
			}

			if len(securityGroups.Resources) == 0 {
				_, _ = os.Stdout.WriteString(config.NoItemsMsg + "\n")

				return nil
			}

			_, _ = os.Stdout.WriteString(config.ListTitle + "\n")
			for _, securityGroup := range securityGroups.Resources {
				_, _ = fmt.Fprintf(os.Stdout, "  - %s (%s)\n", securityGroup.Name, securityGroup.GUID)
			}

			return nil
		},
	}
}

// ResourceResolver helps resolve resources by name or GUID.
type ResourceResolver[T any] struct {
	GetByGUID  func(ctx context.Context, guid string) (*T, error)
	ListByName func(ctx context.Context, name, spaceGUID string) ([]T, error)
	GetGUID    func(resource *T) string
}

// ResolveResource resolves a resource by name or GUID with space filtering.
func (r *ResourceResolver[T]) ResolveResource(ctx context.Context, nameOrGUID, spaceGUID string) (*T, error) {
	// Try by GUID first
	resource, err := r.GetByGUID(ctx, nameOrGUID)
	if err == nil {
		return resource, nil
	}

	// Try by name
	resources, err := r.ListByName(ctx, nameOrGUID, spaceGUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find resource: %w", err)
	}

	if len(resources) == 0 {
		return nil, fmt.Errorf("%w: %s", constants.ErrResourceNotFound, nameOrGUID)
	}

	return &resources[0], nil
}

// OutputRenderer handles different output formats.
type OutputRenderer[T any] struct {
	RenderJSON  func(data T) error
	RenderYAML  func(data T) error
	RenderTable func(data T) error
}

// Render outputs data in the specified format.
func (o *OutputRenderer[T]) Render(data T, format string) error {
	switch format {
	case OutputFormatJSON:
		return o.RenderJSON(data)
	case OutputFormatYAML:
		return o.RenderYAML(data)
	default:
		return o.RenderTable(data)
	}
}

// PageFetcher handles pagination for resources.
type PageFetcher[T any] struct {
	FetchPage func(ctx context.Context, params interface{}, page int) ([]T, *capi.Pagination, error)
}

// FetchAllPages retrieves all pages when allPages is true.
func (p *PageFetcher[T]) FetchAllPages(ctx context.Context, params interface{}, allPages bool, initialResults []T, pagination *capi.Pagination) ([]T, error) {
	if !allPages || pagination.TotalPages <= 1 {
		return initialResults, nil
	}

	allResults := make([]T, 0, len(initialResults))
	allResults = append(allResults, initialResults...)

	for page := 2; page <= pagination.TotalPages; page++ {
		moreResults, _, err := p.FetchPage(ctx, params, page)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		allResults = append(allResults, moreResults...)
	}

	return allResults, nil
}

// RoleManager encapsulates role management operations for organizations and spaces.
type RoleManager struct {
	client capi.Client
}

// NewRoleManager creates a new role manager instance.
func NewRoleManager(client capi.Client) *RoleManager {
	return &RoleManager{client: client}
}

// RoleContext defines the context for role operations.
type RoleContext struct {
	ResourceGUID   string
	ResourceType   string // "organization" or "space"
	DefaultRole    string
	ValidRoles     []string
	SuccessMessage string
}

// AddUserRole adds a user to a resource (organization/space) with a specific role.
func (rm *RoleManager) AddUserRole(ctx context.Context, resourceNameOrGUID, userNameOrGUID string, roleCtx RoleContext, role string) error {
	if role == "" {
		role = roleCtx.DefaultRole
	}

	// Resolve resource GUID
	resourceGUID, err := rm.resolveResourceGUID(ctx, resourceNameOrGUID, roleCtx.ResourceType)
	if err != nil {
		return err
	}

	// Resolve user GUID
	userGUID, err := rm.resolveUserGUID(ctx, userNameOrGUID)
	if err != nil {
		return err
	}

	// Create role request
	roleReq := &capi.RoleCreateRequest{
		Type:          role,
		Relationships: rm.buildRoleRelationships(roleCtx.ResourceType, resourceGUID, userGUID),
	}

	_, err = rm.client.Roles().Create(ctx, roleReq)
	if err != nil {
		return fmt.Errorf("failed to add user to %s: %w", roleCtx.ResourceType, err)
	}

	_, _ = fmt.Fprintf(os.Stdout, roleCtx.SuccessMessage, role)

	return nil
}

// RemoveUserRole removes a user's role from a resource (organization/space).
func (rm *RoleManager) RemoveUserRole(ctx context.Context, resourceNameOrGUID, userNameOrGUID string, roleCtx RoleContext, role string) error {
	// Resolve resource GUID
	resourceGUID, err := rm.resolveResourceGUID(ctx, resourceNameOrGUID, roleCtx.ResourceType)
	if err != nil {
		return err
	}

	// Resolve user GUID
	userGUID, err := rm.resolveUserGUID(ctx, userNameOrGUID)
	if err != nil {
		return err
	}

	// Find and delete role(s)
	rolesClient := rm.client.Roles()
	params := capi.NewQueryParams()
	params.WithFilter("user_guids", userGUID)

	if roleCtx.ResourceType == organizationKey {
		params.WithFilter("organization_guids", resourceGUID)
	} else {
		params.WithFilter("space_guids", resourceGUID)
	}

	if role != "" {
		params.WithFilter("types", role)
	}

	roles, err := rolesClient.List(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list user roles: %w", err)
	}

	if len(roles.Resources) == 0 {
		_, _ = os.Stdout.WriteString("No roles found to remove\n")

		return nil
	}

	// Delete each role
	for _, role := range roles.Resources {
		_, err = rolesClient.Delete(ctx, role.GUID)
		if err != nil {
			return fmt.Errorf("failed to remove role '%s': %w", role.Type, err)
		}

		_, _ = fmt.Fprintf(os.Stdout, "Removed role '%s'\n", role.Type)
	}

	return nil
}

// resolveResourceGUID resolves a resource name or GUID to a GUID.
func (rm *RoleManager) resolveResourceGUID(ctx context.Context, nameOrGUID, resourceType string) (string, error) {
	switch resourceType {
	case "organization":
		orgsClient := rm.client.Organizations()

		org, err := orgsClient.Get(ctx, nameOrGUID)
		if err == nil {
			return org.GUID, nil
		}

		params := capi.NewQueryParams()
		params.WithFilter("names", nameOrGUID)

		orgs, err := orgsClient.List(ctx, params)
		if err != nil {
			return "", fmt.Errorf("failed to find organization: %w", err)
		}

		if len(orgs.Resources) == 0 {
			return "", fmt.Errorf("organization '%s': %w", nameOrGUID, ErrOrganizationNotFound)
		}

		return orgs.Resources[0].GUID, nil

	case "space":
		spacesClient := rm.client.Spaces()

		space, err := spacesClient.Get(ctx, nameOrGUID)
		if err == nil {
			return space.GUID, nil
		}

		params := capi.NewQueryParams()
		params.WithFilter("names", nameOrGUID)

		spaces, err := spacesClient.List(ctx, params)
		if err != nil {
			return "", fmt.Errorf("failed to find space: %w", err)
		}

		if len(spaces.Resources) == 0 {
			return "", fmt.Errorf("space '%s': %w", nameOrGUID, ErrSpaceNotFound)
		}

		return spaces.Resources[0].GUID, nil

	default:
		return "", fmt.Errorf("%w: %s", constants.ErrInvalidResourceType, resourceType)
	}
}

// resolveUserGUID resolves a username or GUID to a user GUID.
func (rm *RoleManager) resolveUserGUID(ctx context.Context, userNameOrGUID string) (string, error) {
	usersClient := rm.client.Users()

	user, err := usersClient.Get(ctx, userNameOrGUID)
	if err == nil {
		return user.GUID, nil
	}

	params := capi.NewQueryParams()
	params.WithFilter("usernames", userNameOrGUID)

	users, err := usersClient.List(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to find user: %w", err)
	}

	if len(users.Resources) == 0 {
		return "", fmt.Errorf("user '%s': %w", userNameOrGUID, ErrUserNotFound)
	}

	return users.Resources[0].GUID, nil
}

// buildRoleRelationships builds the role relationships based on resource type.
func (rm *RoleManager) buildRoleRelationships(resourceType, resourceGUID, userGUID string) capi.RoleRelationships {
	relationships := capi.RoleRelationships{
		User: capi.Relationship{
			Data: &capi.RelationshipData{GUID: userGUID},
		},
	}

	if resourceType == "organization" {
		relationships.Organization = &capi.Relationship{
			Data: &capi.RelationshipData{GUID: resourceGUID},
		}
	} else {
		relationships.Space = &capi.Relationship{
			Data: &capi.RelationshipData{GUID: resourceGUID},
		}
	}

	return relationships
}

// CreateRoleCommand creates a generic role management command.
func CreateRoleCommand(operation, resourceType string, roleContext RoleContext) func() *cobra.Command {
	return func() *cobra.Command {
		var role string

		cmd := &cobra.Command{
			Use:   fmt.Sprintf("%s %s_NAME_OR_GUID USERNAME_OR_GUID", operation, strings.ToUpper(resourceType)),
			Short: fmt.Sprintf("%s user %s %s", cases.Title(language.English).String(operation), operation, resourceType),
			Long:  fmt.Sprintf("%s a user %s %s with a specific role", cases.Title(language.English).String(operation), operation, resourceType),
			Args:  cobra.ExactArgs(constants.TwoArgumentsMax),
			RunE: func(cmd *cobra.Command, args []string) error {
				resourceNameOrGUID := args[0]
				userNameOrGUID := args[1]

				client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
				if err != nil {
					return err
				}

				ctx := context.Background()
				roleManager := NewRoleManager(client)

				if operation == "add-user" || operation == "set-role" {
					return roleManager.AddUserRole(ctx, resourceNameOrGUID, userNameOrGUID, roleContext, role)
				} else {
					return roleManager.RemoveUserRole(ctx, resourceNameOrGUID, userNameOrGUID, roleContext, role)
				}
			},
		}

		// Add role flag with appropriate defaults and options
		if operation == "add-user" || operation == "set-role" {
			cmd.Flags().StringVarP(&role, "role", "r", roleContext.DefaultRole, fmt.Sprintf("role to assign (%s)", strings.Join(roleContext.ValidRoles, ", ")))
		} else {
			cmd.Flags().StringVarP(&role, "role", "r", "", "specific role to remove (if not specified, removes all roles)")
		}

		return cmd
	}
}

// UAAGroupMapper handles UAA group mapping operations.
type UAAGroupMapper struct{}

// GroupMappingConfig defines the configuration for group mapping operations.
type GroupMappingConfig struct {
	Operation      string // "map-group" or "unmap-group"
	SuccessMessage string
	RequiredFlags  []string
}

// CreateUAAGroupMappingCommand creates a generic UAA group mapping command.
func CreateUAAGroupMappingCommand(config GroupMappingConfig) func() *cobra.Command {
	return func() *cobra.Command {
		var group, externalGroup, origin string

		cmd := &cobra.Command{
			Use: config.Operation,
			Short: fmt.Sprintf("%s external group %s UAA group",
				map[string]string{"map-group": "Map", "unmap-group": "Unmap"}[config.Operation],
				map[string]string{"map-group": "to", "unmap-group": "from"}[config.Operation]),
			Long: fmt.Sprintf(`%s an external group from an identity provider %s a UAA group/scope.

This %s users from external identity providers to automatically
inherit UAA group memberships based on their external group memberships.`,
				map[string]string{"map-group": "Map", "unmap-group": "Remove a mapping between"}[config.Operation],
				map[string]string{"map-group": "to", "unmap-group": "and"}[config.Operation],
				map[string]string{"map-group": "allows", "unmap-group": "removes the automatic group membership inheritance for"}[config.Operation]),
			RunE: func(cmd *cobra.Command, args []string) error {
				return runUAAGroupMapping(config, group, externalGroup, origin)
			},
		}

		cmd.Flags().StringVar(&group, "group", "", "UAA group name or ID (required)")
		cmd.Flags().StringVar(&externalGroup, "external-group", "", "External group name (required)")
		cmd.Flags().StringVar(&origin, "origin", "", "Identity provider origin (required)")

		return cmd
	}
}

func runUAAGroupMapping(config GroupMappingConfig, group, externalGroup, origin string) error {
	cfg := loadConfig()

	if GetEffectiveUAAEndpoint(cfg) == "" {
		return constants.ErrNoUAAConfigured
	}

	err := validateGroupMappingFlags(group, externalGroup, origin)
	if err != nil {
		return err
	}

	uaaClient, err := NewUAAClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create UAA client: %w", err)
	}

	if !uaaClient.IsAuthenticated() {
		return constants.ErrNotAuthenticated
	}

	groupID, err := resolveGroupID(uaaClient, group)
	if err != nil {
		return err
	}

	err = performGroupMappingOperation(uaaClient, config.Operation, groupID, externalGroup, origin)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, config.SuccessMessage, externalGroup, origin, group)

	return nil
}

func validateGroupMappingFlags(group, externalGroup, origin string) error {
	if group == "" {
		return constants.ErrGroupRequired
	}

	if externalGroup == "" {
		return constants.ErrExternalGroupRequired
	}

	if origin == "" {
		return constants.ErrOriginRequired
	}

	return nil
}

func resolveGroupID(uaaClient *UAAClientWrapper, group string) (string, error) {
	if isUUID(group) {
		return group, nil
	}

	groupObj, err := uaaClient.Client().GetGroupByName(group, "")
	if err != nil {
		return "", fmt.Errorf("failed to find group '%s': %w", group, err)
	}

	return groupObj.ID, nil
}

func performGroupMappingOperation(uaaClient *UAAClientWrapper, operation, groupID, externalGroup, origin string) error {
	var err error
	if operation == "map-group" {
		err = uaaClient.Client().MapGroup(groupID, externalGroup, origin)
	} else {
		err = uaaClient.Client().UnmapGroup(groupID, externalGroup, origin)
	}

	if err != nil {
		return fmt.Errorf("failed to %s group: %w",
			map[string]string{"map-group": "map", "unmap-group": "unmap"}[operation], err)
	}

	return nil
}

// PaginatedFetcher provides optimized pagination for UAA resources.
type PaginatedFetcher[T any] struct {
	cache    bool
	maxPages int
	pageSize int
}

// NewPaginatedFetcher creates a new paginated fetcher.
func NewPaginatedFetcher[T any](cache bool, maxPages, pageSize int) *PaginatedFetcher[T] {
	return &PaginatedFetcher[T]{
		cache:    cache,
		maxPages: maxPages,
		pageSize: pageSize,
	}
}

// FetchAllResources fetches all resources with optimized pagination and caching.
func (pf *PaginatedFetcher[T]) FetchAllResources(
	cacheKeyPrefix string,
	filter, sortBy, attributes string,
	sortOrder interface{},
	listFunc func(filter, sortBy, attributes string, sortOrder interface{}, startIndex, pageSize int) ([]T, interface{}, error),
	cache interface{}, // Cache interface - should have Get and Set methods
) ([]T, error) {
	cacheKey := fmt.Sprintf("%s:%s:%s:%s:%v", cacheKeyPrefix, filter, sortBy, attributes, sortOrder)

	// Check cache if enabled
	if resources, found := pf.getCachedResources(cache, cacheKey); found {
		return resources, nil
	}

	// Fetch all resources with optimized pagination
	var allResources []T

	startIndex := 1

	for range pf.maxPages {
		resources, pagination, err := listFunc(filter, sortBy, attributes, sortOrder, startIndex, pf.pageSize)
		if err != nil {
			return nil, fmt.Errorf("failed to list resources: %w", err)
		}

		allResources = append(allResources, resources...)

		// Check if we have more pages using reflection
		totalResults := int64(0)

		if paginationValue := reflect.ValueOf(pagination); paginationValue.IsValid() {
			if totalField := paginationValue.FieldByName("TotalResults"); totalField.IsValid() {
				totalResults = totalField.Int()
			}
		}

		if totalResults <= int64(startIndex+len(resources)-1) {
			break
		}

		startIndex += len(resources)
	}

	// Cache the result if enabled
	pf.setCachedResources(cache, cacheKey, allResources)

	return allResources, nil
}

// getCachedResources attempts to retrieve cached resources from the cache.
func (pf *PaginatedFetcher[T]) getCachedResources(cache interface{}, cacheKey string) ([]T, bool) {
	if !pf.cache || cache == nil {
		return nil, false
	}

	// Use reflection to call Get method on cache
	cacheValue := reflect.ValueOf(cache).MethodByName("Get")
	if !cacheValue.IsValid() {
		return nil, false
	}

	result := cacheValue.Call([]reflect.Value{reflect.ValueOf(cacheKey)})
	if len(result) <= 1 || !result[1].Bool() { // not found
		return nil, false
	}

	if resources, ok := result[0].Interface().([]T); ok {
		return resources, true
	}

	return nil, false
}

// setCachedResources stores resources in the cache.
func (pf *PaginatedFetcher[T]) setCachedResources(cache interface{}, cacheKey string, resources []T) {
	if !pf.cache || cache == nil {
		return
	}

	cacheValue := reflect.ValueOf(cache).MethodByName("Set")
	if cacheValue.IsValid() {
		cacheValue.Call([]reflect.Value{reflect.ValueOf(cacheKey), reflect.ValueOf(resources)})
	}
}

// StandardJSONRenderer creates a standard JSON encoder.
func StandardJSONRenderer[T any](data T) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(data)
	if err != nil {
		return fmt.Errorf("encoding data to JSON: %w", err)
	}

	return nil
}

// StandardYAMLRenderer creates a standard YAML encoder.
func StandardYAMLRenderer[T any](data T) error {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(defaultJSONIndent)

	err := encoder.Encode(data)
	if err != nil {
		return fmt.Errorf("encoding data to YAML: %w", err)
	}

	return nil
}

// FilterBuilder helps build query parameters with multiple optional filters.
type FilterBuilder struct {
	params *capi.QueryParams
}

// NewFilterBuilder creates a new filter builder with initialized query params.
func NewFilterBuilder() *FilterBuilder {
	return &FilterBuilder{
		params: capi.NewQueryParams(),
	}
}

// SetPerPage sets the page size.
func (fb *FilterBuilder) SetPerPage(perPage int) *FilterBuilder {
	fb.params.PerPage = perPage

	return fb
}

// AddFilterIf adds a filter conditionally based on the value.
func (fb *FilterBuilder) AddFilterIf(key, value string) *FilterBuilder {
	if value != "" {
		fb.params.WithFilter(key, value)
	}

	return fb
}

// Build returns the constructed query parameters.
func (fb *FilterBuilder) Build() *capi.QueryParams {
	return fb.params
}

// PaginationHandler handles common pagination logic.
type PaginationHandler[T any] struct {
	FetchPage func(ctx context.Context, params *capi.QueryParams, page int) ([]T, *capi.Pagination, error)
}

// FetchAllPages handles pagination when allPages is true.
func (ph *PaginationHandler[T]) FetchAllPages(ctx context.Context, params *capi.QueryParams, allPages bool, initialResources []T, initialPagination *capi.Pagination) ([]T, error) {
	if !allPages || initialPagination.TotalPages <= 1 {
		return initialResources, nil
	}

	allResources := make([]T, 0, len(initialResources))
	allResources = append(allResources, initialResources...)

	for page := 2; page <= initialPagination.TotalPages; page++ {
		moreResources, _, err := ph.FetchPage(ctx, params, page)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		allResources = append(allResources, moreResources...)
	}

	return allResources, nil
}

// StandardOutputRenderer handles common JSON/YAML/table output logic.
type StandardOutputRenderer[T any] struct {
	RenderTable func(resources []T, pagination *capi.Pagination, allPages bool) error
}

// Render handles standard output formats.
func (sor *StandardOutputRenderer[T]) Render(resources []T, pagination *capi.Pagination, allPages bool, output string) error {
	switch output {
	case OutputFormatJSON:
		return StandardJSONRenderer(resources)
	case OutputFormatYAML:
		return StandardYAMLRenderer(resources)
	default:
		return sor.RenderTable(resources, pagination, allPages)
	}
}

// ResourceFinder helps resolve resources by name or GUID.
type ResourceFinder[T any] struct {
	GetByGUID   func(ctx context.Context, client capi.Client, guid string) (*T, error)
	ListByName  func(ctx context.Context, client capi.Client, name string) ([]T, error)
	NotFoundErr error
}

// FindResource finds a resource by name or GUID.
func (rf *ResourceFinder[T]) FindResource(ctx context.Context, client capi.Client, nameOrGUID string) (*T, error) {
	// Try by GUID first
	resource, err := rf.GetByGUID(ctx, client, nameOrGUID)
	if err == nil {
		return resource, nil
	}

	// Try by name
	resources, err := rf.ListByName(ctx, client, nameOrGUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find resource: %w", err)
	}

	if len(resources) == 0 {
		return nil, fmt.Errorf("resource '%s': %w", nameOrGUID, rf.NotFoundErr)
	}

	return &resources[0], nil
}

// BooleanFlagParser helps parse string boolean flags to avoid direct boolean comparison complexity.
type BooleanFlagParser struct {
	flagValue string
	flagName  string
	errorType error
}

// NewBooleanFlagParser creates a parser for string boolean flags.
func NewBooleanFlagParser(flagValue, flagName string, errorType error) *BooleanFlagParser {
	return &BooleanFlagParser{
		flagValue: flagValue,
		flagName:  flagName,
		errorType: errorType,
	}
}

// Parse parses the string boolean flag and returns pointer to bool or error.
func (bfp *BooleanFlagParser) Parse() (*bool, error) {
	if bfp.flagValue == "" {
		return nil, nil
	}

	val, err := strconv.ParseBool(bfp.flagValue)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", bfp.errorType, bfp.flagValue)
	}

	return &val, nil
}

// resolveSpaceGUIDWithOrgFilter finds the space GUID from the space name with optional org filtering.
func resolveSpaceGUIDWithOrgFilter(ctx context.Context, client capi.Client, spaceName string) (string, error) {
	spaceParams := capi.NewQueryParams()
	spaceParams.WithFilter("names", spaceName)

	// Add org filter if targeted
	if orgGUID := viper.GetString("organization_guid"); orgGUID != "" {
		spaceParams.WithFilter("organization_guids", orgGUID)
	}

	spaces, err := client.Spaces().List(ctx, spaceParams)
	if err != nil {
		return "", fmt.Errorf("failed to find space: %w", err)
	}

	if len(spaces.Resources) == 0 {
		return "", fmt.Errorf("space '%s': %w", spaceName, ErrSpaceNotFound)
	}

	return spaces.Resources[0].GUID, nil
}

// resolveSpaceGUID resolves a space GUID from either a space name or targeted space.
func resolveSpaceGUID(ctx context.Context, client capi.Client, spaceName string) (string, error) {
	if spaceName != "" {
		return resolveSpaceGUIDWithOrgFilter(ctx, client, spaceName)
	}

	if targetedSpaceGUID := viper.GetString("space_guid"); targetedSpaceGUID != "" {
		return targetedSpaceGUID, nil
	}

	return "", ErrNoSpaceSpecifiedAndTargeted
}

// resolveSpaceGUIDForServices resolves a space GUID specifically for service commands.
func resolveSpaceGUIDForServices(ctx context.Context, client capi.Client, spaceName string) (string, error) {
	if spaceName != "" {
		return resolveSpaceGUIDWithOrgFilter(ctx, client, spaceName)
	}

	if targetedSpaceGUID := viper.GetString("space_guid"); targetedSpaceGUID != "" {
		return targetedSpaceGUID, nil
	}

	return "", ErrSpaceRequired
}
