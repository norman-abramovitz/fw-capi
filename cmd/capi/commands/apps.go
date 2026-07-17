package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	// Environment variable display.
	defaultEnvVarTruncateLength = 80

	// Command argument counts.
	setEnvExactArgs   = 3
	unsetEnvExactArgs = 2
	scaleRangeArgs    = 2
	sshRangeArgs      = 2

	// Log and event limits.
	defaultLogLines  = 50
	defaultMaxEvents = 50

	// CPU and memory conversion.
	cpuPercentMultiplier = 100
	bytesToMBDivisor     = 1024

	// Display limits.
	commandTruncateLength     = 50
	descriptionTruncateLength = 60

	// healthCheckTypeNone is the CF API discriminator for "no health check".
	healthCheckTypeNone = "none"
)

// validateFilePath validates that a file path is safe to read.
func validateFilePathApps(filePath string) error {
	// Clean the path to resolve any path traversal attempts
	cleanPath := filepath.Clean(filePath)

	// Check for path traversal attempts
	if filepath.IsAbs(filePath) {
		// Allow absolute paths but ensure they're clean
		if cleanPath != filePath {
			return capi.ErrPathTraversalAttempt
		}
	} else {
		// For relative paths, ensure they don't escape the current directory
		if len(cleanPath) > 0 && cleanPath[0] == '.' && len(cleanPath) > 1 && cleanPath[1] == '.' {
			return capi.ErrPathTraversalNotAllowed
		}
	}

	// Check if file exists and is readable
	_, err := os.Stat(cleanPath)
	if err != nil {
		return fmt.Errorf("file not accessible: %w", err)
	}

	return nil
}

// NewAppsCommand creates the apps command group.
func NewAppsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "apps",
		Aliases: []string{"app", "applications"},
		Short:   "Manage applications",
		Long:    "List, create, and manage Cloud Foundry applications",
	}

	cmd.AddCommand(newAppsListCommand())
	cmd.AddCommand(newAppsStartCommand())
	cmd.AddCommand(newAppsStopCommand())
	cmd.AddCommand(newAppsRestartCommand())
	cmd.AddCommand(newAppsScaleCommand())
	cmd.AddCommand(newAppsEnvCommand())
	cmd.AddCommand(newAppsSetEnvCommand())
	cmd.AddCommand(newAppsUnsetEnvCommand())
	cmd.AddCommand(newAppsLogsCommand())
	cmd.AddCommand(newAppsSSHCommand())
	cmd.AddCommand(newAppsProcessesCommand())
	cmd.AddCommand(newAppsManifestCommand())
	cmd.AddCommand(newAppsStatsCommand())
	cmd.AddCommand(newAppsEventsCommand())
	cmd.AddCommand(newAppsHealthCheckCommand())
	cmd.AddCommand(newAppsTasksCommand())
	cmd.AddCommand(newAppsDeploymentsCommand())
	cmd.AddCommand(newAppsPackagesCommand())
	cmd.AddCommand(newAppsDropletsCommand())
	cmd.AddCommand(newAppsBuildsCommand())
	cmd.AddCommand(newAppsFeaturesCommand())

	return cmd
}

func newAppsListCommand() *cobra.Command {
	var spaceName string

	cmd := &cobra.Command{
		Use:   List,
		Short: "List applications",
		Long:  "List all applications the user has access to",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAppsList(cmd, spaceName)
		},
	}

	cmd.Flags().StringVarP(&spaceName, spaceKey, "s", "", "filter by space name")

	return cmd
}

func runAppsList(cmd *cobra.Command, spaceName string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	params, err := buildAppsListParams(spaceName)
	if err != nil {
		return err
	}

	apps, err := client.Apps().List(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list applications: %w", err)
	}

	return outputAppsList(apps.Resources)
}

func buildAppsListParams(spaceName string) (*capi.QueryParams, error) {
	params := capi.NewQueryParams()

	if spaceName != "" {
		return addSpaceFilterByName(params, spaceName)
	}

	if spaceGUID := viper.GetString("space_guid"); spaceGUID != "" {
		params.WithFilter("space_guids", spaceGUID)
	}

	return params, nil
}

func addSpaceFilterByName(params *capi.QueryParams, spaceName string) (*capi.QueryParams, error) {
	ctx := context.Background()

	client, err := CreateClientWithAPI(viper.GetString("api"))
	if err != nil {
		return nil, err
	}

	spaceParams := capi.NewQueryParams()
	spaceParams.WithFilter("names", spaceName)

	if orgGUID := viper.GetString("organization_guid"); orgGUID != "" {
		spaceParams.WithFilter("organization_guids", orgGUID)
	}

	spaces, err := client.Spaces().List(ctx, spaceParams)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}

	if len(spaces.Resources) == 0 {
		return nil, fmt.Errorf("%w: '%s'", capi.ErrSpaceNotFound, spaceName)
	}

	params.WithFilter("space_guids", spaces.Resources[0].GUID)

	return params, nil
}

type appListInfo struct {
	Name       string `json:"name"`
	GUID       string `json:"guid"`
	State      string `json:"state"`
	Lifecycle  string `json:"lifecycle"`
	Buildpacks string `json:"buildpacks"`
	Stack      string `json:"stack"`
	Created    string `json:"created"`
	Updated    string `json:"updated"`
}

func outputAppsList(apps []capi.App) error {
	output := viper.GetString("output")
	switch output {
	case constants.FormatJSON:
		return outputAppsListJSON(apps)
	case constants.FormatYAML:
		return outputAppsListYAML(apps)
	default:
		return outputAppsListTable(apps)
	}
}

func outputAppsListJSON(apps []capi.App) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(apps)
	if err != nil {
		return fmt.Errorf("failed to encode apps as JSON: %w", err)
	}

	return nil
}

func outputAppsListYAML(apps []capi.App) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(apps)
	if err != nil {
		return fmt.Errorf("failed to encode apps as YAML: %w", err)
	}

	return nil
}

func outputAppsListTable(apps []capi.App) error {
	if len(apps) == 0 {
		_, _ = os.Stdout.WriteString("No applications found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "GUID", "State", "Lifecycle", "Buildpacks", "Stack", "Created", "Updated")

	for _, app := range apps {
		info := buildAppListInfo(app)
		_ = table.Append(info.Name, info.GUID, info.State, info.Lifecycle, info.Buildpacks, info.Stack, info.Created, info.Updated)
	}

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render apps table: %w", err)
	}

	return nil
}

func buildAppListInfo(app capi.App) appListInfo {
	return appListInfo{
		Name:       app.Name,
		GUID:       app.GUID,
		State:      app.State,
		Lifecycle:  extractLifecycle(app),
		Buildpacks: extractBuildpacks(app),
		Stack:      extractStack(app),
		Created:    formatTime(app.CreatedAt),
		Updated:    formatTime(app.UpdatedAt),
	}
}

func extractLifecycle(app capi.App) string {
	if app.Lifecycle.Type == "docker" {
		return "docker"
	}

	return "buildpack"
}

func extractBuildpacks(app capi.App) string {
	if app.Lifecycle.Data == nil {
		return ""
	}

	bps, ok := app.Lifecycle.Data["buildpacks"].([]any)
	if !ok {
		return ""
	}

	var bpStrs []string

	for _, bp := range bps {
		if bpStr, ok := bp.(string); ok {
			bpStrs = append(bpStrs, bpStr)
		}
	}

	return strings.Join(bpStrs, ", ")
}

func extractStack(app capi.App) string {
	if app.Lifecycle.Data == nil {
		return ""
	}

	if s, ok := app.Lifecycle.Data["stack"].(string); ok {
		return s
	}

	return ""
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.Format(TimeFormatDisplay)
}

func newAppsStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start APP_NAME_OR_GUID",
		Short: "Start an application",
		Long:  "Start a Cloud Foundry application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find application
			appGUID, appName, err := resolveApp(ctx, client, nameOrGUID)
			if err != nil {
				return err
			}

			// Start application — CF v3 /actions/start returns a job.
			job, err := client.Apps().Start(ctx, appGUID)
			if err != nil {
				return fmt.Errorf("failed to start application: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Queued start of application '%s' (job %s)\n", appName, job.GUID)

			return nil
		},
	}
}

func newAppsStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop APP_NAME_OR_GUID",
		Short: "Stop an application",
		Long:  "Stop a Cloud Foundry application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find application
			appGUID, appName, err := resolveApp(ctx, client, nameOrGUID)
			if err != nil {
				return err
			}

			// Stop application — CF v3 /actions/stop returns a job.
			job, err := client.Apps().Stop(ctx, appGUID)
			if err != nil {
				return fmt.Errorf("failed to stop application: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Queued stop of application '%s' (job %s)\n", appName, job.GUID)

			return nil
		},
	}
}

func newAppsRestartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "restart APP_NAME_OR_GUID",
		Short: "Restart an application",
		Long:  "Restart a Cloud Foundry application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nameOrGUID := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Find application
			appGUID, appName, err := resolveApp(ctx, client, nameOrGUID)
			if err != nil {
				return err
			}

			// Restart application — CF v3 /actions/restart returns a job.
			job, err := client.Apps().Restart(ctx, appGUID)
			if err != nil {
				return fmt.Errorf("failed to restart application: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "Queued restart of application '%s' (job %s)\n", appName, job.GUID)

			return nil
		},
	}
}

// Helper function to resolve app name or GUID.
func resolveApp(ctx context.Context, client capi.Client, nameOrGUID string) (string, string, error) {
	// Try to get by GUID first
	app, err := client.Apps().Get(ctx, nameOrGUID)
	if err == nil {
		return app.GUID, app.Name, nil
	}

	// Try by name
	params := capi.NewQueryParams()
	params.WithFilter("names", nameOrGUID)

	// Add space filter if targeted
	if spaceGUID := viper.GetString("space_guid"); spaceGUID != "" {
		params.WithFilter("space_guids", spaceGUID)
	}

	apps, err := client.Apps().List(ctx, params)
	if err != nil {
		return "", "", fmt.Errorf("failed to find application: %w", err)
	}

	if len(apps.Resources) == 0 {
		return "", "", fmt.Errorf("%w: '%s'", capi.ErrApplicationNotFound, nameOrGUID)
	}

	return apps.Resources[0].GUID, apps.Resources[0].Name, nil
}

func newAppsScaleCommand() *cobra.Command {
	var (
		instances   int
		memory      int
		disk        int
		processType string
	)

	cmd := &cobra.Command{
		Use:   "scale APP_NAME_OR_GUID",
		Short: "Scale an application",
		Long:  "Scale a Cloud Foundry application instances, memory, or disk",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scaleParams := &appsScaleParams{
				nameOrGUID:  args[0],
				instances:   instances,
				memory:      memory,
				disk:        disk,
				processType: processType,
				cmd:         cmd,
			}

			return runAppsScale(scaleParams)
		},
	}

	cmd.Flags().IntVarP(&instances, "instances", "i", 0, "Number of instances")
	cmd.Flags().IntVarP(&memory, "memory", "m", 0, "Memory in MB")
	cmd.Flags().IntVarP(&disk, "disk", "d", 0, "Disk in MB")
	cmd.Flags().StringVarP(&processType, "process", "p", "", "Process type (defaults to first process)")

	return cmd
}

type appsScaleParams struct {
	nameOrGUID  string
	instances   int
	memory      int
	disk        int
	processType string
	cmd         *cobra.Command
}

type scaleInfo struct {
	AppName     string `json:"app_name"     yaml:"app_name"`
	AppGUID     string `json:"app_guid"     yaml:"app_guid"`
	ProcessType string `json:"process_type" yaml:"process_type"`
	Instances   int    `json:"instances"    yaml:"instances"`
	MemoryMB    int    `json:"memory_mb"    yaml:"memory_mb"`
	DiskMB      int    `json:"disk_mb"      yaml:"disk_mb"`
}

func runAppsScale(params *appsScaleParams) error {
	client, err := CreateClientWithAPI(params.cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	appGUID, appName, err := resolveApp(ctx, client, params.nameOrGUID)
	if err != nil {
		return err
	}

	targetProcess, err := findTargetProcess(ctx, client, appGUID, appName, params.processType)
	if err != nil {
		return err
	}

	scaleReq := buildScaleRequest(params)
	if isScaleRequestEmpty(scaleReq) {
		return outputCurrentScale(appName, appGUID, targetProcess)
	}

	// Scale is async (202 + job). We log the queued job and report the
	// current scale targets back to the user; callers that need terminal
	// state should poll Jobs().Get(job.GUID).
	job, err := client.Processes().Scale(ctx, targetProcess.GUID, scaleReq)
	if err != nil {
		return fmt.Errorf("failed to scale application: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Queued scale of application '%s' process '%s' (job %s)\n", appName, targetProcess.Type, job.GUID)

	return outputScaledResult(appName, appGUID, targetProcess, scaleReq)
}

func findTargetProcess(ctx context.Context, client capi.Client, appGUID, appName, processType string) (*capi.Process, error) {
	processParams := capi.NewQueryParams().WithFilter("app_guids", appGUID)

	processes, err := client.Processes().List(ctx, processParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	if len(processes.Resources) == 0 {
		return nil, fmt.Errorf("%w for application '%s'", capi.ErrNoProcessesFound, appName)
	}

	for _, process := range processes.Resources {
		if processType == "" || process.Type == processType {
			return &process, nil
		}
	}

	return nil, fmt.Errorf("%w '%s' for application '%s'", capi.ErrProcessTypeNotFound, processType, appName)
}

func buildScaleRequest(params *appsScaleParams) *capi.ProcessScaleRequest {
	scaleReq := &capi.ProcessScaleRequest{}

	if params.cmd.Flags().Changed("instances") {
		scaleReq.Instances = &params.instances
	}

	if params.cmd.Flags().Changed("memory") {
		scaleReq.MemoryInMB = &params.memory
	}

	if params.cmd.Flags().Changed("disk") {
		scaleReq.DiskInMB = &params.disk
	}

	return scaleReq
}

func isScaleRequestEmpty(scaleReq *capi.ProcessScaleRequest) bool {
	return scaleReq.Instances == nil && scaleReq.MemoryInMB == nil && scaleReq.DiskInMB == nil
}

func outputCurrentScale(appName, appGUID string, process *capi.Process) error {
	info := scaleInfo{
		AppName:     appName,
		AppGUID:     appGUID,
		ProcessType: process.Type,
		Instances:   process.Instances,
		MemoryMB:    process.MemoryInMB,
		DiskMB:      process.DiskInMB,
	}

	return outputScaleInfo(info, fmt.Sprintf("Current scale for application '%s' process '%s':", appName, process.Type))
}

func outputScaledResult(appName, appGUID string, process *capi.Process, scaleReq *capi.ProcessScaleRequest) error {
	// The returned job doesn't carry process fields, so we echo back the
	// requested scale (falling back to current values for fields that
	// weren't in the request). The queued-job line is printed separately
	// by the caller so operators see the async nature of the call.
	info := scaleInfo{
		AppName:     appName,
		AppGUID:     appGUID,
		ProcessType: process.Type,
		Instances:   process.Instances,
		MemoryMB:    process.MemoryInMB,
		DiskMB:      process.DiskInMB,
	}

	if scaleReq.Instances != nil {
		info.Instances = *scaleReq.Instances
	}

	if scaleReq.MemoryInMB != nil {
		info.MemoryMB = *scaleReq.MemoryInMB
	}

	if scaleReq.DiskInMB != nil {
		info.DiskMB = *scaleReq.DiskInMB
	}

	return outputScaleInfo(info, fmt.Sprintf("Requested scale for application '%s' process '%s':", appName, process.Type))
}

func outputScaleInfo(info scaleInfo, tableTitle string) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return outputScaleInfoJSON(info)
	case OutputFormatYAML:
		return outputScaleInfoYAML(info)
	default:
		return outputScaleInfoTable(info, tableTitle)
	}
}

func outputScaleInfoJSON(info scaleInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(info)
	if err != nil {
		return fmt.Errorf("failed to encode scale info as JSON: %w", err)
	}

	return nil
}

func outputScaleInfoYAML(info scaleInfo) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(info)
	if err != nil {
		return fmt.Errorf("failed to encode scale info as YAML: %w", err)
	}

	return nil
}

func outputScaleInfoTable(info scaleInfo, title string) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")
	_ = table.Append("Application", info.AppName)
	_ = table.Append("Process Type", info.ProcessType)
	_ = table.Append("Instances", strconv.Itoa(info.Instances))
	_ = table.Append("Memory", fmt.Sprintf("%d MB", info.MemoryMB))
	_ = table.Append("Disk", fmt.Sprintf("%d MB", info.DiskMB))

	_, _ = fmt.Fprintf(os.Stdout, "%s\n\n", title)

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render scale info table: %w", err)
	}

	return nil
}

// flattenJSON recursively flattens a JSON object into a map with dot-separated keys.
func flattenJSON(obj any, prefix string) map[string]any {
	result := make(map[string]any)

	switch value := obj.(type) {
	case map[string]any:
		flattenMapObject(value, prefix, result)
	case []any:
		flattenArrayObject(value, prefix, result)
	default:
		flattenPrimitiveObject(value, prefix, result)
	}

	return result
}

// flattenMapObject handles flattening of map[string]any objects.
func flattenMapObject(m map[string]any, prefix string, result map[string]any) {
	for key, value := range m {
		fullKey := buildFullKey(key, prefix)
		mergeNestedResults(value, fullKey, result)
	}
}

// flattenArrayObject handles flattening of []any objects.
func flattenArrayObject(arr []any, prefix string, result map[string]any) {
	for i, item := range arr {
		fullKey := fmt.Sprintf("%s[%d]", prefix, i)
		mergeNestedResults(item, fullKey, result)
	}
}

// flattenPrimitiveObject handles flattening of primitive values.
func flattenPrimitiveObject(value any, prefix string, result map[string]any) {
	if prefix != "" {
		result[prefix] = value
	}
}

// buildFullKey constructs the full key for nested objects.
func buildFullKey(key, prefix string) string {
	if prefix != "" {
		return prefix + "." + key
	}

	return key
}

// mergeNestedResults recursively flattens and merges nested results.
func mergeNestedResults(value any, fullKey string, result map[string]any) {
	if nested := flattenJSON(value, fullKey); len(nested) > 0 {
		for k, v := range nested {
			result[k] = v
		}
	} else {
		result[fullKey] = value
	}
}

func newAppsEnvCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "env APP_NAME_OR_GUID",
		Short: "Show application environment variables",
		Long:  "Display all environment variables for a Cloud Foundry application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAppsEnv(cmd, args[0])
		},
	}
}

type envVar struct {
	Name   string `json:"name"   yaml:"name"`
	Value  any    `json:"value"  yaml:"value"`
	Source string `json:"source" yaml:"source"`
}

type appEnvData struct {
	EnvVars         []envVar `json:"environment_variables"`
	VcapServices    any      `json:"vcap_services"`
	VcapApplication any      `json:"vcap_application"`
	AppName         string   `json:"-"`
}

func runAppsEnv(cmd *cobra.Command, nameOrGUID string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	appGUID, appName, err := resolveApp(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	env, err := client.Apps().GetEnv(ctx, appGUID)
	if err != nil {
		return fmt.Errorf("failed to get environment variables: %w", err)
	}

	envData := collectAppEnvData(env, appName)

	return outputAppEnv(envData)
}

func collectAppEnvData(env *capi.AppEnv, appName string) *appEnvData {
	data := &appEnvData{AppName: appName}

	collectUserProvidedEnvVars(env, &data.EnvVars)
	data.VcapServices = collectSystemEnvVars(env, &data.EnvVars)
	collectStagingEnvVars(env, &data.EnvVars)
	collectRunningEnvVars(env, &data.EnvVars)
	data.VcapApplication = collectApplicationEnvVars(env, &data.EnvVars)

	return data
}

func collectUserProvidedEnvVars(env *capi.AppEnv, envVars *[]envVar) {
	for key, value := range env.EnvironmentVariables {
		*envVars = append(*envVars, envVar{
			Name:   key,
			Value:  value,
			Source: "user-provided",
		})
	}
}

func collectSystemEnvVars(env *capi.AppEnv, envVars *[]envVar) any {
	var vcapServices any

	for key, value := range env.SystemEnvJSON {
		if key == "VCAP_SERVICES" {
			vcapServices = value
		} else {
			*envVars = append(*envVars, envVar{
				Name:   key,
				Value:  value,
				Source: "system",
			})
		}
	}

	return vcapServices
}

func collectStagingEnvVars(env *capi.AppEnv, envVars *[]envVar) {
	for key, value := range env.StagingEnvJSON {
		*envVars = append(*envVars, envVar{
			Name:   key,
			Value:  value,
			Source: LifecycleStaging,
		})
	}
}

func collectRunningEnvVars(env *capi.AppEnv, envVars *[]envVar) {
	for key, value := range env.RunningEnvJSON {
		*envVars = append(*envVars, envVar{
			Name:   key,
			Value:  value,
			Source: LifecycleRunning,
		})
	}
}

func collectApplicationEnvVars(env *capi.AppEnv, envVars *[]envVar) any {
	var vcapApplication any

	for key, value := range env.ApplicationEnvJSON {
		if key == "VCAP_APPLICATION" {
			vcapApplication = value
		} else {
			*envVars = append(*envVars, envVar{
				Name:   key,
				Value:  value,
				Source: "application",
			})
		}
	}

	return vcapApplication
}

func outputAppEnv(data *appEnvData) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return outputAppEnvJSON(data)
	case OutputFormatYAML:
		return outputAppEnvYAML(data)
	default:
		return outputAppEnvTable(data)
	}
}

func outputAppEnvJSON(data *appEnvData) error {
	result := map[string]any{
		keyEnvironmentVariables: data.EnvVars,
		"vcap_services":         data.VcapServices,
		"vcap_application":      data.VcapApplication,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(result)
	if err != nil {
		return fmt.Errorf("failed to encode app env as JSON: %w", err)
	}

	return nil
}

func outputAppEnvYAML(data *appEnvData) error {
	result := map[string]any{
		keyEnvironmentVariables: data.EnvVars,
		"vcap_services":         data.VcapServices,
		"vcap_application":      data.VcapApplication,
	}
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(result)
	if err != nil {
		return fmt.Errorf("failed to encode app env as YAML: %w", err)
	}

	return nil
}

func outputAppEnvTable(data *appEnvData) error {
	_, _ = fmt.Fprintf(os.Stdout, "Environment variables for application '%s':\n\n", data.AppName)

	renderEnvVarsTable(data.EnvVars)
	renderVcapServicesTable(data.VcapServices)
	renderVcapApplicationTable(data.VcapApplication)

	if len(data.EnvVars) == 0 && data.VcapServices == nil && data.VcapApplication == nil {
		_, _ = fmt.Fprintf(os.Stdout, "No environment variables found for application '%s'\n", data.AppName)
	}

	return nil
}

func renderEnvVarsTable(envVars []envVar) {
	if len(envVars) == 0 {
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Name", "Value", "Source")

	for _, envVar := range envVars {
		valueStr := truncateValue(fmt.Sprintf("%v", envVar.Value), defaultEnvVarTruncateLength)
		_ = table.Append(envVar.Name, valueStr, envVar.Source)
	}

	_ = table.Render()

	_, _ = os.Stdout.WriteString("\n")
}

func renderVcapServicesTable(vcapServices any) {
	if vcapServices == nil {
		return
	}

	vcapServicesFlattened := flattenJSON(vcapServices, "")
	if len(vcapServicesFlattened) == 0 {
		return
	}

	_, _ = os.Stdout.WriteString("VCAP_SERVICES:\n")
	_, _ = os.Stdout.WriteString("\n")

	servicesTable := tablewriter.NewWriter(os.Stdout)
	servicesTable.Header("Key", "Value")

	for key, value := range vcapServicesFlattened {
		valueStr := fmt.Sprintf("%v", value)
		_ = servicesTable.Append(key, valueStr)
	}

	_ = servicesTable.Render()

	_, _ = os.Stdout.WriteString("\n")
}

func renderVcapApplicationTable(vcapApplication any) {
	if vcapApplication == nil {
		return
	}

	vcapApplicationFlattened := flattenJSON(vcapApplication, "")
	if len(vcapApplicationFlattened) == 0 {
		return
	}

	_, _ = os.Stdout.WriteString("VCAP_APPLICATION:\n")
	_, _ = os.Stdout.WriteString("\n")

	appTable := tablewriter.NewWriter(os.Stdout)
	appTable.Header("Key", "Value")

	for key, value := range vcapApplicationFlattened {
		valueStr := fmt.Sprintf("%v", value)
		_ = appTable.Append(key, valueStr)
	}

	_ = appTable.Render()
}

func truncateValue(value string, maxLength int) string {
	if len(value) > maxLength {
		return value[:maxLength-3] + "..."
	}

	return value
}

func newAppsSetEnvCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set-env APP_NAME_OR_GUID ENV_VAR_NAME ENV_VAR_VALUE",
		Short: "Set an environment variable for an application",
		Long:  "Set a user-provided environment variable for a Cloud Foundry application",
		Args:  cobra.ExactArgs(setEnvExactArgs),
		RunE:  runAppsSetEnv,
	}
}

type envVarResult struct {
	AppName   string `json:"app_name"  yaml:"app_name"`
	AppGUID   string `json:"app_guid"  yaml:"app_guid"`
	Name      string `json:"name"      yaml:"name"`
	Value     string `json:"value"     yaml:"value"`
	Operation string `json:"operation" yaml:"operation"`
}

func runAppsSetEnv(cmd *cobra.Command, args []string) error {
	nameOrGUID := args[0]
	envVarName := args[1]
	envVarValue := args[2]

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	appGUID, appName, err := resolveApp(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	err = setEnvironmentVariable(ctx, client, appGUID, envVarName, envVarValue)
	if err != nil {
		return err
	}

	result := envVarResult{
		AppName:   appName,
		AppGUID:   appGUID,
		Name:      envVarName,
		Value:     envVarValue,
		Operation: "set",
	}

	return outputEnvVarResult(result, appName, envVarName, envVarValue, "set")
}

func setEnvironmentVariable(ctx context.Context, client capi.Client, appGUID, name, value string) error {
	currentEnvVars, err := client.Apps().GetEnvVars(ctx, appGUID)
	if err != nil {
		return fmt.Errorf("failed to get current environment variables: %w", err)
	}

	if currentEnvVars == nil {
		currentEnvVars = make(map[string]any)
	}

	currentEnvVars[name] = value

	_, err = client.Apps().UpdateEnvVars(ctx, appGUID, currentEnvVars)
	if err != nil {
		return fmt.Errorf("failed to set environment variable: %w", err)
	}

	return nil
}

func outputEnvVarResult(result envVarResult, appName, envVarName, envVarValue, operation string) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(result)
		if err != nil {
			return fmt.Errorf("failed to encode env var result as JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(result)
		if err != nil {
			return fmt.Errorf("failed to encode env var result as YAML: %w", err)
		}

		return nil
	default:
		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Property", "Value")
		_ = table.Append("Application", appName)
		_ = table.Append("Operation", operation)
		_ = table.Append("Variable Name", envVarName)
		_ = table.Append("Variable Value", envVarValue)

		_, _ = fmt.Fprintf(os.Stdout, "Successfully %s environment variable for application '%s':\n\n", operation, appName)

		_ = table.Render()

		_, _ = os.Stdout.WriteString("\nNote: You may need to restart the application for the changes to take effect.\n")
	}

	return nil
}

func newAppsUnsetEnvCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unset-env APP_NAME_OR_GUID ENV_VAR_NAME",
		Short: "Unset an environment variable for an application",
		Long:  "Remove a user-provided environment variable from a Cloud Foundry application",
		Args:  cobra.ExactArgs(unsetEnvExactArgs),
		RunE:  runAppsUnsetEnv,
	}
}

func runAppsUnsetEnv(cmd *cobra.Command, args []string) error {
	nameOrGUID := args[0]
	envVarName := args[1]

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	appGUID, appName, err := resolveApp(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	err = unsetEnvironmentVariable(ctx, client, appGUID, envVarName, appName)
	if err != nil {
		return err
	}

	result := envVarResult{
		AppName:   appName,
		AppGUID:   appGUID,
		Name:      envVarName,
		Operation: "unset",
	}

	return outputEnvVarResult(result, appName, envVarName, "", "unset")
}

func unsetEnvironmentVariable(ctx context.Context, client capi.Client, appGUID, name, appName string) error {
	currentEnvVars, err := client.Apps().GetEnvVars(ctx, appGUID)
	if err != nil {
		return fmt.Errorf("failed to get current environment variables: %w", err)
	}

	if currentEnvVars == nil || currentEnvVars[name] == nil {
		return fmt.Errorf("%w '%s' for application '%s'", capi.ErrEnvironmentVariableNotFound, name, appName)
	}

	currentEnvVars[name] = nil

	_, err = client.Apps().UpdateEnvVars(ctx, appGUID, currentEnvVars)
	if err != nil {
		return fmt.Errorf("failed to unset environment variable: %w", err)
	}

	return nil
}

func newAppsLogsCommand() *cobra.Command {
	var (
		follow   bool
		recent   bool
		numLines int
	)

	cmd := &cobra.Command{
		Use:   "logs APP_NAME_OR_GUID",
		Short: "Show application logs",
		Long:  "Display recent logs or stream logs for a Cloud Foundry application",
		Args:  cobra.ExactArgs(1),
		RunE:  runAppsLogs,
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Stream logs continuously")
	cmd.Flags().BoolVarP(&recent, "recent", "r", false, "Show recent logs only")
	cmd.Flags().IntVarP(&numLines, "lines", "n", defaultLogLines, "Number of recent log lines to show")

	return cmd
}

func runAppsLogs(cmd *cobra.Command, args []string) error {
	nameOrGUID := args[0]

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	appGUID, appName, err := resolveApp(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	follow, _ := cmd.Flags().GetBool("follow")
	recent, _ := cmd.Flags().GetBool("recent")
	numLines, _ := cmd.Flags().GetInt("lines")

	if recent || !follow {
		return showRecentLogs(ctx, client, appGUID, appName, numLines)
	}

	if follow {
		return streamLogs(ctx, client, appGUID, appName)
	}

	return nil
}

func showRecentLogs(ctx context.Context, client capi.Client, appGUID, appName string, numLines int) error {
	logs, err := client.Apps().GetRecentLogs(ctx, appGUID, numLines)
	if err != nil {
		return fmt.Errorf("failed to get recent logs: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Recent logs for application '%s':\n\n", appName)

	for _, logMsg := range logs.Messages {
		timestamp := logMsg.Timestamp.Format("2006-01-02T15:04:05.00-0700")
		_, _ = fmt.Fprintf(os.Stdout, "   %s [%s] %s %s\n",
			timestamp, logMsg.SourceType, logMsg.MessageType, logMsg.Message)
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nNote: Logs streaming requires WebSocket/SSE connection to CF API.\n")
	_, _ = fmt.Fprintf(os.Stdout, "Application GUID: %s\n", appGUID)

	return nil
}

func streamLogs(ctx context.Context, client capi.Client, appGUID, appName string) error {
	_, _ = fmt.Fprintf(os.Stdout, "Streaming logs for application '%s'...\n", appName)
	_, _ = os.Stdout.WriteString("Press Ctrl+C to stop streaming.\n")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logChan, err := client.Apps().StreamLogs(ctx, appGUID)
	if err != nil {
		return fmt.Errorf("failed to start log streaming: %w", err)
	}

	for logMsg := range logChan {
		timestamp := logMsg.Timestamp.Format("2006-01-02T15:04:05.00-0700")
		_, _ = fmt.Fprintf(os.Stdout, "   %s [%s] %s %s\n",
			timestamp, logMsg.SourceType, logMsg.MessageType, logMsg.Message)
	}

	_, _ = os.Stdout.WriteString("\nLog streaming stopped.\n")

	return nil
}

func newAppsSSHCommand() *cobra.Command {
	var (
		index       int
		processType string
		command     string
	)

	cmd := &cobra.Command{
		Use:   "ssh APP_NAME_OR_GUID",
		Short: "SSH into an application instance",
		Long:  "Open an SSH connection to a Cloud Foundry application instance",
		Args:  cobra.ExactArgs(1),
		RunE:  runAppsSSH,
	}

	cmd.Flags().IntVarP(&index, "index", "i", 0, "Instance index to connect to")
	cmd.Flags().StringVarP(&processType, "process", "p", "", "Process type (defaults to first process)")
	cmd.Flags().StringVar(&command, "command", "", "Command to run in the SSH session")

	return cmd
}

func runAppsSSH(cmd *cobra.Command, args []string) error {
	nameOrGUID := args[0]

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	appGUID, appName, err := resolveApp(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	index, _ := cmd.Flags().GetInt("index")
	processType, _ := cmd.Flags().GetString("process")
	command, _ := cmd.Flags().GetString("command")

	return establishSSHConnection(ctx, client, appGUID, appName, processType, index, command)
}

func establishSSHConnection(ctx context.Context, client capi.Client, appGUID, appName, processType string, index int, command string) error {
	targetProcess, err := findTargetProcessForSSH(ctx, client, appGUID, appName, processType)
	if err != nil {
		return err
	}

	if index >= targetProcess.Instances {
		return fmt.Errorf("%w: %d is out of range (0-%d) for process '%s'",
			capi.ErrInstanceIndexOutOfRange, index, targetProcess.Instances-1, targetProcess.Type)
	}

	displaySSHInfo(appName, targetProcess, index, command, appGUID)

	return nil
}

func findTargetProcessForSSH(ctx context.Context, client capi.Client, appGUID, appName, processType string) (*capi.Process, error) {
	processParams := capi.NewQueryParams().WithFilter("app_guids", appGUID)

	processes, err := client.Processes().List(ctx, processParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	if len(processes.Resources) == 0 {
		return nil, fmt.Errorf("%w for application '%s'", capi.ErrNoProcessesFound, appName)
	}

	for _, process := range processes.Resources {
		if processType == "" || process.Type == processType {
			return &process, nil
		}
	}

	return nil, fmt.Errorf("%w '%s' for application '%s'", capi.ErrProcessTypeNotFound, processType, appName)
}

func displaySSHInfo(appName string, targetProcess *capi.Process, index int, command, appGUID string) {
	_, _ = fmt.Fprintf(os.Stdout, "Connecting to application '%s' instance %d via SSH...\n", appName, index)
	_, _ = fmt.Fprintf(os.Stdout, "Process: %s/%d\n", targetProcess.Type, index)

	if command != "" {
		_, _ = fmt.Fprintf(os.Stdout, "Command: %s\n", command)
	}

	_, _ = os.Stdout.WriteString("\nNote: SSH connection requires:\n")
	_, _ = os.Stdout.WriteString("1. SSH to be enabled for the space and application\n")
	_, _ = os.Stdout.WriteString("2. Authentication with Diego SSH proxy\n")
	_, _ = os.Stdout.WriteString("3. Network access to Diego cells\n")
	_, _ = fmt.Fprintf(os.Stdout, "4. App GUID: %s\n", appGUID)
	_, _ = fmt.Fprintf(os.Stdout, "5. Process GUID: %s\n", targetProcess.GUID)
}

func newAppsProcessesCommand() *cobra.Command {
	var showStats bool

	cmd := &cobra.Command{
		Use:   "processes APP_NAME_OR_GUID",
		Short: "List application processes",
		Long:  "List all processes for a Cloud Foundry application with their current status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAppsProcesses(cmd, args[0], showStats)
		},
	}

	cmd.Flags().BoolVarP(&showStats, "stats", "s", false, "Show detailed instance statistics")

	return cmd
}

// runAppsProcesses handles the main logic for listing application processes.
func runAppsProcesses(cmd *cobra.Command, nameOrGUID string, showStats bool) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	appGUID, appName, err := resolveApp(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	processes, err := fetchAppProcesses(ctx, client, appGUID)
	if err != nil {
		return err
	}

	if len(processes.Resources) == 0 {
		return outputEmptyProcesses(appName)
	}

	processData := buildProcessData(ctx, client, processes.Resources, showStats)

	return outputProcesses(processes.Resources, processData, appName, showStats)
}

// fetchAppProcesses retrieves all processes for a given application.
func fetchAppProcesses(ctx context.Context, client capi.Client, appGUID string) (*capi.ProcessList, error) {
	processParams := capi.NewQueryParams().WithFilter("app_guids", appGUID)

	processes, err := client.Processes().List(ctx, processParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	return processes, nil
}

// outputEmptyProcesses handles the case when no processes are found.
func outputEmptyProcesses(appName string) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode([]capi.Process{})
		if err != nil {
			return fmt.Errorf("failed to encode empty processes as JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode([]capi.Process{})
		if err != nil {
			return fmt.Errorf("failed to encode empty processes as YAML: %w", err)
		}

		return nil
	default:
		_, _ = fmt.Fprintf(os.Stdout, "No processes found for application '%s'\n", appName)
	}

	return nil
}

// buildProcessData creates structured data for all processes.
func buildProcessData(ctx context.Context, client capi.Client, processes []capi.Process, showStats bool) []map[string]any {
	processData := make([]map[string]any, 0, len(processes))

	for _, process := range processes {
		processInfo := buildProcessInfo(process)

		if showStats {
			stats, err := getProcessStats(ctx, client, process.GUID)
			if err == nil {
				processInfo["instance_stats"] = stats
			}
		}

		processData = append(processData, processInfo)
	}

	return processData
}

// buildProcessInfo creates the basic process information map.
func buildProcessInfo(process capi.Process) map[string]any {
	processInfo := map[string]any{
		"type":      process.Type,
		"guid":      process.GUID,
		"instances": process.Instances,
		"memory_mb": process.MemoryInMB,
		"disk_mb":   process.DiskInMB,
	}

	setLogRateLimit(processInfo, process.LogRateLimitInBytesPerSecond)
	setCommand(processInfo, process.Command)
	setHealthCheck(processInfo, process.HealthCheck)

	return processInfo
}

// setLogRateLimit adds log rate limit information to the process info.
func setLogRateLimit(processInfo map[string]any, logRateLimit *int) {
	if logRateLimit != nil {
		processInfo["log_rate_limit_bytes_per_sec"] = *logRateLimit
	} else {
		processInfo["log_rate_limit_bytes_per_sec"] = -1
	}
}

// setCommand adds command information to the process info.
func setCommand(processInfo map[string]any, command *string) {
	if command != nil {
		processInfo["command"] = *command
	}
}

// setHealthCheck adds health check information to the process info.
func setHealthCheck(processInfo map[string]any, healthCheck *capi.HealthCheck) {
	if healthCheck == nil {
		return
	}

	healthCheckInfo := map[string]any{
		"type": healthCheck.Type,
	}

	if healthCheck.Data != nil {
		if healthCheck.Data.Timeout != nil {
			healthCheckInfo["timeout"] = *healthCheck.Data.Timeout
		}

		if healthCheck.Data.Endpoint != nil {
			healthCheckInfo["endpoint"] = *healthCheck.Data.Endpoint
		}
	}

	processInfo["health_check"] = healthCheckInfo
}

// getProcessStats retrieves and formats process statistics.
func getProcessStats(ctx context.Context, client capi.Client, processGUID string) ([]map[string]any, error) {
	stats, err := client.Processes().GetStats(ctx, processGUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get process stats: %w", err)
	}

	if len(stats.Resources) == 0 {
		return nil, nil
	}

	instanceStats := make([]map[string]any, 0, len(stats.Resources))
	for _, stat := range stats.Resources {
		statInfo := map[string]any{
			"index": stat.Index,
			"state": stat.State,
		}
		if stat.Usage != nil {
			statInfo["usage"] = map[string]any{
				"cpu_percent":  stat.Usage.CPU * cpuPercentMultiplier,
				"memory_bytes": stat.Usage.Mem,
				"disk_bytes":   stat.Usage.Disk,
			}
		}

		instanceStats = append(instanceStats, statInfo)
	}

	return instanceStats, nil
}

// outputProcesses handles the output of process data in the requested format.
func outputProcesses(processes []capi.Process, processData []map[string]any, appName string, showStats bool) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return outputProcessesJSON(processData)
	case OutputFormatYAML:
		return outputProcessesYAML(processData)
	default:
		return outputProcessesTable(processes, appName, showStats)
	}
}

// outputProcessesJSON outputs processes in JSON format.
func outputProcessesJSON(processData []map[string]any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(processData)
	if err != nil {
		return fmt.Errorf("failed to encode processes as JSON: %w", err)
	}

	return nil
}

// outputProcessesYAML outputs processes in YAML format.
func outputProcessesYAML(processData []map[string]any) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(processData)
	if err != nil {
		return fmt.Errorf("failed to encode processes as YAML: %w", err)
	}

	return nil
}

// outputProcessesTable outputs processes in table format.
func outputProcessesTable(processes []capi.Process, appName string, showStats bool) error {
	table := tablewriter.NewWriter(os.Stdout)
	setProcessTableHeaders(table, showStats)

	for _, process := range processes {
		row := buildProcessTableRow(process, showStats)

		interfaceRow := make([]any, len(row))
		for i, v := range row {
			interfaceRow[i] = v
		}

		_ = table.Append(interfaceRow...)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Processes for application '%s':\n\n", appName)

	_ = table.Render()

	return nil
}

// setProcessTableHeaders sets the appropriate headers for the process table.
func setProcessTableHeaders(table *tablewriter.Table, showStats bool) {
	if showStats {
		table.Header("Process", "GUID", "Instances", "Memory", "Disk", "Log Rate Limit", "Command", "Health Check", "Instance Stats")
	} else {
		table.Header("Process", "GUID", "Instances", "Memory", "Disk", "Log Rate Limit", "Command", "Health Check")
	}
}

// buildProcessTableRow creates a table row for a single process.
func buildProcessTableRow(process capi.Process, showStats bool) []string {
	row := []string{
		process.Type,
		process.GUID,
		strconv.Itoa(process.Instances),
		fmt.Sprintf("%d MB", process.MemoryInMB),
		fmt.Sprintf("%d MB", process.DiskInMB),
		formatLogRateLimit(process.LogRateLimitInBytesPerSecond),
		"[PRIVATE DATA HIDDEN IN LISTS]",
		formatHealthCheck(process.HealthCheck),
	}

	if showStats {
		instanceStats := constants.NotAvailable
		// Note: This would need the client to be passed down for full functionality
		// For now, keeping it as N/A to maintain the interface
		row = append(row, instanceStats)
	}

	return row
}

// formatLogRateLimit formats the log rate limit for display.
func formatLogRateLimit(logRateLimit *int) string {
	if logRateLimit != nil {
		return fmt.Sprintf("%d bytes/sec", *logRateLimit)
	}

	return "-1 bytes/sec"
}

// formatHealthCheck formats the health check information for display.
func formatHealthCheck(healthCheck *capi.HealthCheck) string {
	if healthCheck == nil {
		return healthCheckTypeNone
	}

	result := healthCheck.Type
	if healthCheck.Data != nil {
		if healthCheck.Data.Timeout != nil {
			result += fmt.Sprintf(" (timeout: %ds)", *healthCheck.Data.Timeout)
		}

		if healthCheck.Data.Endpoint != nil {
			result += fmt.Sprintf(" (endpoint: %s)", *healthCheck.Data.Endpoint)
		}
	}

	return result
}

func newAppsManifestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Manage application manifests",
		Long:  "View, apply, or diff application manifests",
	}

	cmd.AddCommand(newAppsManifestGetCommand())
	cmd.AddCommand(newAppsManifestApplyCommand())
	cmd.AddCommand(newAppsManifestDiffCommand())

	return cmd
}

func newAppsManifestGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get APP_NAME_OR_GUID",
		Short: "Get application manifest",
		Long:  "Retrieve the current manifest for a Cloud Foundry application",
		Args:  cobra.ExactArgs(1),
		RunE:  runAppsManifestGet,
	}
}

type manifestInfo struct {
	AppName      string `json:"app_name"      yaml:"app_name"`
	AppGUID      string `json:"app_guid"      yaml:"app_guid"`
	ManifestYAML string `json:"manifest_yaml" yaml:"manifest_yaml"`
}

func runAppsManifestGet(cmd *cobra.Command, args []string) error {
	nameOrGUID := args[0]

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	appGUID, appName, err := resolveApp(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	manifest, err := client.Apps().GetManifest(ctx, appGUID)
	if err != nil {
		return fmt.Errorf("failed to get manifest: %w", err)
	}

	info := manifestInfo{
		AppName:      appName,
		AppGUID:      appGUID,
		ManifestYAML: manifest,
	}

	return outputManifestInfo(info)
}

func outputManifestInfo(info manifestInfo) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(info)
		if err != nil {
			return fmt.Errorf("failed to encode manifest info as JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		_, _ = fmt.Fprintf(os.Stdout, "# Application: %s\n", info.AppName)
		_, _ = fmt.Fprintf(os.Stdout, "# GUID: %s\n", info.AppGUID)
		_, _ = os.Stdout.WriteString("# Manifest:\n")

		_, _ = os.Stdout.WriteString(info.ManifestYAML)

		return nil
	default:
		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Property", "Value")
		_ = table.Append("Application Name", info.AppName)
		_ = table.Append("Application GUID", info.AppGUID)

		_, _ = os.Stdout.WriteString("Manifest information:\n\n")

		_ = table.Render()

		_, _ = os.Stdout.WriteString("\nManifest content:\n\n")

		_, _ = os.Stdout.WriteString(info.ManifestYAML)
	}

	return nil
}

func newAppsManifestApplyCommand() *cobra.Command {
	var manifestFile string

	cmd := &cobra.Command{
		Use:   "apply SPACE_NAME_OR_GUID [MANIFEST_FILE]",
		Short: "Apply application manifest",
		Long:  "Apply a manifest to deploy or update applications in a space",
		Args:  cobra.RangeArgs(1, scaleRangeArgs),
		RunE:  runAppsManifestApply,
	}

	cmd.Flags().StringVarP(&manifestFile, "file", "f", "", "path to manifest file (default: manifest.yml)")

	return cmd
}

func runAppsManifestApply(cmd *cobra.Command, args []string) error {
	spaceIdentifier := args[0]

	manifestFile, _ := cmd.Flags().GetString("file")
	manifestPath := determineManifestPath(args, manifestFile)

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	space, err := resolveSpaceForManifest(ctx, client, spaceIdentifier)
	if err != nil {
		return err
	}

	manifestContent, err := readManifestFile(manifestPath)
	if err != nil {
		return err
	}

	job, err := client.Spaces().ApplyManifest(ctx, space.GUID, manifestContent)
	if err != nil {
		return fmt.Errorf("failed to apply manifest: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully applied manifest to space '%s'\n", space.Name)

	if job != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Job GUID: %s\n", job.GUID)
		_, _ = fmt.Fprintf(os.Stdout, "Monitor job status with: capi jobs get %s\n", job.GUID)
	}

	return nil
}

func newAppsManifestDiffCommand() *cobra.Command {
	var manifestFile string

	cmd := &cobra.Command{
		Use:   "diff SPACE_NAME_OR_GUID [MANIFEST_FILE]",
		Short: "Show manifest diff",
		Long:  "Show the differences between the current state and a manifest",
		Args:  cobra.RangeArgs(1, scaleRangeArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			spaceIdentifier := args[0]

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Resolve space
			space, err := resolveSpaceForManifest(ctx, client, spaceIdentifier)
			if err != nil {
				return err
			}

			// Determine manifest path
			manifestPath := determineManifestPath(args, manifestFile)

			// Read and validate manifest
			manifestContent, err := readManifestFile(manifestPath)
			if err != nil {
				return err
			}

			// Create and display diff
			return createAndDisplayManifestDiff(ctx, client, space, manifestContent)
		},
	}

	cmd.Flags().StringVarP(&manifestFile, "file", "f", "", "path to manifest file (default: manifest.yml)")

	return cmd
}

// resolveSpaceForManifest resolves space by name or GUID for manifest operations.
func resolveSpaceForManifest(ctx context.Context, client capi.Client, spaceIdentifier string) (*capi.Space, error) {
	// Try as GUID first
	space, err := client.Spaces().Get(ctx, spaceIdentifier)
	if err == nil {
		return space, nil
	}

	// Try by name with org filtering
	spaceParams := capi.NewQueryParams()
	spaceParams.WithFilter("names", spaceIdentifier)

	if orgGUID := viper.GetString("organization_guid"); orgGUID != "" {
		spaceParams.WithFilter("organization_guids", orgGUID)
	}

	spaces, err := client.Spaces().List(ctx, spaceParams)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}

	if len(spaces.Resources) == 0 {
		return nil, fmt.Errorf("%w: %s", constants.ErrSpaceNotFound, spaceIdentifier)
	}

	return &spaces.Resources[0], nil
}

// determineManifestPath determines the manifest file path from arguments and flags.
func determineManifestPath(args []string, manifestFile string) string {
	if len(args) == scaleRangeArgs {
		return args[1]
	}

	if manifestFile != "" {
		return manifestFile
	}

	return "manifest.yml"
}

// readManifestFile reads and validates a manifest file.
func readManifestFile(manifestPath string) (string, error) {
	// Validate file path to prevent directory traversal
	err := validateFilePathApps(manifestPath)
	if err != nil {
		return "", fmt.Errorf("invalid manifest file: %w", err)
	}

	manifestContent, err := os.ReadFile(filepath.Clean(manifestPath))
	if err != nil {
		return "", fmt.Errorf("failed to read manifest file '%s': %w", manifestPath, err)
	}

	return string(manifestContent), nil
}

// createAndDisplayManifestDiff creates and displays the manifest diff.
func createAndDisplayManifestDiff(ctx context.Context, client capi.Client, space *capi.Space, manifestContent string) error {
	diff, err := client.Spaces().CreateManifestDiff(ctx, space.GUID, manifestContent)
	if err != nil {
		return fmt.Errorf("failed to create manifest diff: %w", err)
	}

	displayManifestDiff(space.Name, diff)

	return nil
}

// displayManifestDiff displays the manifest diff results.
func displayManifestDiff(spaceName string, diff *capi.ManifestDiff) {
	_, _ = fmt.Fprintf(os.Stdout, "Manifest diff for space '%s':\n\n", spaceName)

	if len(diff.Diff) == 0 {
		_, _ = os.Stdout.WriteString("No differences found - manifest matches current state\n")

		return
	}

	for _, entry := range diff.Diff {
		_, _ = fmt.Fprintf(os.Stdout, "%s: %s\n", entry.Op, entry.Path)

		if entry.Was != nil {
			_, _ = fmt.Fprintf(os.Stdout, "  Was: %v\n", entry.Was)
		}

		if entry.Value != nil {
			_, _ = fmt.Fprintf(os.Stdout, "  Now: %v\n", entry.Value)
		}
	}
}

func newAppsStatsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stats APP_NAME_OR_GUID",
		Short: "Show application statistics",
		Long:  "Display resource usage statistics for all instances of a Cloud Foundry application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAppsStats(cmd, args[0])
		},
	}
}

// InstanceStat represents statistics for a single application instance.
type InstanceStat struct {
	ProcessType   string   `json:"process_type"              yaml:"process_type"`
	ProcessGUID   string   `json:"process_guid"              yaml:"process_guid"`
	Instances     int      `json:"instances"                 yaml:"instances"`
	MemoryMB      int      `json:"memory_mb"                 yaml:"memory_mb"`
	DiskMB        int      `json:"disk_mb"                   yaml:"disk_mb"`
	Index         int      `json:"index"                     yaml:"index"`
	State         string   `json:"state"                     yaml:"state"`
	CPUPercent    *float64 `json:"cpu_percent,omitempty"     yaml:"cpu_percent,omitempty"`
	MemoryUsageMB *int     `json:"memory_usage_mb,omitempty" yaml:"memory_usage_mb,omitempty"`
	DiskUsageMB   *int     `json:"disk_usage_mb,omitempty"   yaml:"disk_usage_mb,omitempty"`
	Host          string   `json:"host,omitempty"            yaml:"host,omitempty"`
	UptimeSeconds *int     `json:"uptime_seconds,omitempty"  yaml:"uptime_seconds,omitempty"`
	Ports         []string `json:"ports,omitempty"           yaml:"ports,omitempty"`
}

// runAppsStats handles the main logic for application statistics command.
func runAppsStats(cmd *cobra.Command, nameOrGUID string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	appGUID, appName, err := resolveApp(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	processes, err := fetchAppProcesses(ctx, client, appGUID)
	if err != nil {
		return err
	}

	if len(processes.Resources) == 0 {
		return handleEmptyProcesses(appName)
	}

	allStats := collectAllStatistics(ctx, client, processes.Resources)

	return outputStatistics(allStats, appName)
}

// handleEmptyProcesses handles the case when no processes are found.
func handleEmptyProcesses(appName string) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode([]any{})
		if err != nil {
			return fmt.Errorf("failed to encode empty processes as JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode([]any{})
		if err != nil {
			return fmt.Errorf("failed to encode empty processes as YAML: %w", err)
		}

		return nil
	default:
		_, _ = fmt.Fprintf(os.Stdout, "No processes found for application '%s'\n", appName)
	}

	return nil
}

// collectAllStatistics collects statistics for all processes.
func collectAllStatistics(ctx context.Context, client capi.Client, processes []capi.Process) []InstanceStat {
	allStats := make([]InstanceStat, 0, len(processes))

	for _, process := range processes {
		stats := collectProcessStatistics(ctx, client, process)
		allStats = append(allStats, stats...)
	}

	return allStats
}

// collectProcessStatistics collects statistics for a single process.
func collectProcessStatistics(ctx context.Context, client capi.Client, process capi.Process) []InstanceStat {
	stats, err := client.Processes().GetStats(ctx, process.GUID)
	if err != nil {
		return []InstanceStat{createErrorStat(process, "stats-error")}
	}

	if len(stats.Resources) == 0 {
		return []InstanceStat{createErrorStat(process, "no-stats")}
	}

	return buildInstanceStats(process, stats.Resources)
}

// createErrorStat creates an InstanceStat for error cases.
func createErrorStat(process capi.Process, state string) InstanceStat {
	return InstanceStat{
		ProcessType: process.Type,
		ProcessGUID: process.GUID,
		Instances:   process.Instances,
		MemoryMB:    process.MemoryInMB,
		DiskMB:      process.DiskInMB,
		Index:       -1,
		State:       state,
	}
}

// buildInstanceStats builds InstanceStat objects from process stats.
func buildInstanceStats(process capi.Process, statsResources []capi.ProcessStat) []InstanceStat {
	stats := make([]InstanceStat, 0, len(statsResources))

	for _, stat := range statsResources {
		instanceStat := InstanceStat{
			ProcessType: process.Type,
			ProcessGUID: process.GUID,
			Instances:   process.Instances,
			MemoryMB:    process.MemoryInMB,
			DiskMB:      process.DiskInMB,
			Index:       stat.Index,
			State:       stat.State,
			Host:        stat.Host,
		}

		populateUsageStats(&instanceStat, stat.Usage)
		populateUptimeStats(&instanceStat, int64(stat.Uptime))
		populatePortStats(&instanceStat, stat.InstancePorts)

		stats = append(stats, instanceStat)
	}

	return stats
}

// populateUsageStats populates usage statistics in the InstanceStat.
func populateUsageStats(instanceStat *InstanceStat, usage *capi.ProcessUsage) {
	if usage == nil {
		return
	}

	cpuPercent := usage.CPU * cpuPercentMultiplier
	memUsageMB := int(usage.Mem / (bytesToMBDivisor * bytesToMBDivisor))
	diskUsageMB := int(usage.Disk / (bytesToMBDivisor * bytesToMBDivisor))

	instanceStat.CPUPercent = &cpuPercent
	instanceStat.MemoryUsageMB = &memUsageMB
	instanceStat.DiskUsageMB = &diskUsageMB
}

// populateUptimeStats populates uptime statistics in the InstanceStat.
func populateUptimeStats(instanceStat *InstanceStat, uptime int64) {
	if uptime > 0 {
		uptimeSeconds := int(uptime)
		instanceStat.UptimeSeconds = &uptimeSeconds
	}
}

// populatePortStats populates port statistics in the InstanceStat.
func populatePortStats(instanceStat *InstanceStat, instancePorts []capi.InstancePort) {
	if len(instancePorts) == 0 {
		return
	}

	ports := make([]string, 0, len(instancePorts))
	for _, port := range instancePorts {
		ports = append(ports, fmt.Sprintf("%d->%d", port.External, port.Internal))
	}

	instanceStat.Ports = ports
}

// outputStatistics outputs the statistics in the requested format.
func outputStatistics(allStats []InstanceStat, appName string) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return outputStatisticsJSON(allStats)
	case OutputFormatYAML:
		return outputStatisticsYAML(allStats)
	default:
		return outputStatisticsTable(allStats, appName)
	}
}

// outputStatisticsJSON outputs statistics as JSON.
func outputStatisticsJSON(allStats []InstanceStat) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(allStats)
	if err != nil {
		return fmt.Errorf("failed to encode statistics as JSON: %w", err)
	}

	return nil
}

// outputStatisticsYAML outputs statistics as YAML.
func outputStatisticsYAML(allStats []InstanceStat) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(allStats)
	if err != nil {
		return fmt.Errorf("failed to encode statistics as YAML: %w", err)
	}

	return nil
}

// outputStatisticsTable outputs statistics as a table.
func outputStatisticsTable(allStats []InstanceStat, appName string) error {
	if len(allStats) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No statistics found for application '%s'\n", appName)

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Process", "Index", "State", "CPU%", "Memory", "Disk", "Uptime", "Host")

	for _, stat := range allStats {
		row := buildStatTableRow(stat)

		interfaceRow := make([]any, len(row))
		for i, v := range row {
			interfaceRow[i] = v
		}

		_ = table.Append(interfaceRow...)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Statistics for application '%s':\n\n", appName)

	_ = table.Render()

	return nil
}

// buildStatTableRow builds a table row for a single instance stat.
func buildStatTableRow(stat InstanceStat) []string {
	return []string{
		stat.ProcessType,
		formatIndex(stat.Index),
		stat.State,
		formatCPU(stat.CPUPercent),
		formatMemory(stat.MemoryUsageMB),
		formatDisk(stat.DiskUsageMB),
		formatUptime(stat.UptimeSeconds),
		stat.Host,
	}
}

// formatIndex formats the instance index for display.
func formatIndex(index int) string {
	if index == -1 {
		return NotAvailable
	}

	return strconv.Itoa(index)
}

// formatCPU formats the CPU percentage for display.
func formatCPU(cpuPercent *float64) string {
	if cpuPercent == nil {
		return NotAvailable
	}

	return fmt.Sprintf("%.2f", *cpuPercent)
}

// formatMemory formats the memory usage for display.
func formatMemory(memoryUsageMB *int) string {
	if memoryUsageMB == nil {
		return NotAvailable
	}

	return fmt.Sprintf("%d MB", *memoryUsageMB)
}

// formatDisk formats the disk usage for display.
func formatDisk(diskUsageMB *int) string {
	if diskUsageMB == nil {
		return NotAvailable
	}

	return fmt.Sprintf("%d MB", *diskUsageMB)
}

// formatUptime formats the uptime for display.
func formatUptime(uptimeSeconds *int) string {
	if uptimeSeconds == nil {
		return NotAvailable
	}

	uptime := time.Duration(*uptimeSeconds) * time.Second

	return uptime.String()
}

func newAppsEventsCommand() *cobra.Command {
	var maxEvents int

	cmd := &cobra.Command{
		Use:   "events APP_NAME_OR_GUID",
		Short: "Show application events",
		Long:  "Display recent events for a Cloud Foundry application",
		Args:  cobra.ExactArgs(1),
		RunE:  runAppsEvents,
	}

	cmd.Flags().IntVarP(&maxEvents, "max", "m", defaultMaxEvents, "Maximum number of events to show")

	return cmd
}

type appEvent struct {
	Type        string `json:"type"        yaml:"type"`
	Time        string `json:"time"        yaml:"time"`
	Actor       string `json:"actor"       yaml:"actor"`
	Description string `json:"description" yaml:"description"`
	AppGUID     string `json:"app_guid"    yaml:"app_guid"`
	AppName     string `json:"app_name"    yaml:"app_name"`
}

func runAppsEvents(cmd *cobra.Command, args []string) error {
	nameOrGUID := args[0]

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	appGUID, appName, err := resolveApp(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	maxEvents, _ := cmd.Flags().GetInt("max")
	events := generateSimulatedEvents(appGUID, appName, maxEvents)

	return outputAppEvents(events, appName)
}

func generateSimulatedEvents(appGUID, appName string, maxEvents int) []appEvent {
	events := []appEvent{
		{
			Type:        "app.start",
			Time:        time.Now().Add(-time.Hour).Format(time.RFC3339),
			Actor:       "user@example.com",
			Description: "Application started",
			AppGUID:     appGUID,
			AppName:     appName,
		},
		{
			Type:        "app.scale",
			Time:        time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
			Actor:       "user@example.com",
			Description: "Application scaled to 2 instances",
			AppGUID:     appGUID,
			AppName:     appName,
		},
		{
			Type:        "app.restart",
			Time:        time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
			Actor:       "system",
			Description: "Application restarted",
			AppGUID:     appGUID,
			AppName:     appName,
		},
	}

	if maxEvents > 0 && maxEvents < len(events) {
		events = events[:maxEvents]
	}

	return events
}

func outputAppEvents(events []appEvent, appName string) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(events)
		if err != nil {
			return fmt.Errorf("failed to encode events as JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(events)
		if err != nil {
			return fmt.Errorf("failed to encode events as YAML: %w", err)
		}

		return nil
	default:
		return outputEventsTable(events, appName)
	}
}

func outputEventsTable(events []appEvent, appName string) error {
	if len(events) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No events found for application '%s'\n", appName)

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Type", "Time", "Actor", "Description")

	for _, event := range events {
		eventTime, err := time.Parse(time.RFC3339, event.Time)
		if err == nil {
			event.Time = eventTime.Format(TimeFormatDisplay)
		}

		_ = table.Append(event.Type, event.Time, event.Actor, event.Description)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Recent events for application '%s':\n\n", appName)

	_ = table.Render()

	_, _ = os.Stdout.WriteString("\nNote: Events shown are simulated examples.\n")
	_, _ = os.Stdout.WriteString("Real implementation would query Cloud Foundry audit events API.\n")
	_, _ = fmt.Fprintf(os.Stdout, "Consider using 'cf events %s' from the CF CLI for actual events.\n", appName)

	return nil
}

func newAppsHealthCheckCommand() *cobra.Command {
	var (
		healthCheckType string
		timeout         int
		endpoint        string
		processType     string
	)

	cmd := &cobra.Command{
		Use:   "health-check APP_NAME_OR_GUID",
		Short: "Configure application health check",
		Long:  "View or configure health check settings for a Cloud Foundry application process",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAppsHealthCheck(cmd, args[0], healthCheckType, timeout, endpoint, processType)
		},
	}

	cmd.Flags().StringVar(&healthCheckType, "type", "", "Health check type (port, process, http, none)")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Health check timeout in seconds")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Health check endpoint (for http type)")
	cmd.Flags().StringVarP(&processType, "process", "p", "", "Process type (defaults to first process)")

	return cmd
}

// HealthCheckInfo represents health check configuration information.
type HealthCheckInfo struct {
	ProcessType       string  `json:"process_type"                           yaml:"process_type"`
	ProcessGUID       string  `json:"process_guid"                           yaml:"process_guid"`
	Type              string  `json:"type"                                   yaml:"type"`
	Timeout           *int    `json:"timeout,omitempty"                      yaml:"timeout,omitempty"`
	Endpoint          *string `json:"endpoint,omitempty"                     yaml:"endpoint,omitempty"`
	InvocationTimeout *int    `json:"invocation_timeout,omitempty"           yaml:"invocation_timeout,omitempty"`
	Interval          *int    `json:"interval,omitempty"                     yaml:"interval,omitempty"`
	ReadinessType     *string `json:"readiness_type,omitempty"               yaml:"readiness_type,omitempty"`
	ReadinessEndpoint *string `json:"readiness_endpoint,omitempty"           yaml:"readiness_endpoint,omitempty"`
	ReadinessTimeout  *int    `json:"readiness_invocation_timeout,omitempty" yaml:"readiness_invocation_timeout,omitempty"`
	ReadinessInterval *int    `json:"readiness_interval,omitempty"           yaml:"readiness_interval,omitempty"`
}

// runAppsHealthCheck handles the main logic for health check command.
func runAppsHealthCheck(cmd *cobra.Command, nameOrGUID, healthCheckType string, timeout int, endpoint, processType string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	appGUID, appName, err := resolveApp(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	targetProcess, err := findTargetProcessForHealthCheck(ctx, client, appGUID, appName, processType)
	if err != nil {
		return err
	}

	if healthCheckType == "" {
		return displayHealthCheck(targetProcess, appName)
	}

	return updateHealthCheck(ctx, client, targetProcess, appName, healthCheckType, timeout, endpoint)
}

// findTargetProcessForHealthCheck finds the target process for health check operations.
func findTargetProcessForHealthCheck(ctx context.Context, client capi.Client, appGUID, appName, processType string) (*capi.Process, error) {
	processParams := capi.NewQueryParams().WithFilter("app_guids", appGUID)

	processes, err := client.Processes().List(ctx, processParams)
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	if len(processes.Resources) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No processes found for application '%s'\n", appName)

		return nil, capi.ErrNoProcessesFound
	}

	var targetProcess *capi.Process

	for _, process := range processes.Resources {
		if processType == "" || process.Type == processType {
			targetProcess = &process

			break
		}
	}

	if targetProcess == nil {
		return nil, fmt.Errorf("%w '%s' for application '%s'", capi.ErrProcessTypeNotFound, processType, appName)
	}

	return targetProcess, nil
}

// displayHealthCheck shows the current health check configuration.
func displayHealthCheck(targetProcess *capi.Process, appName string) error {
	healthCheckInfo := buildHealthCheckInfo(targetProcess)

	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return outputHealthCheckJSON(healthCheckInfo)
	case OutputFormatYAML:
		return outputHealthCheckYAML(healthCheckInfo)
	default:
		return outputHealthCheckTable(healthCheckInfo, appName, targetProcess.Type)
	}
}

// buildHealthCheckInfo creates the health check info structure.
func buildHealthCheckInfo(targetProcess *capi.Process) HealthCheckInfo {
	healthCheckInfo := HealthCheckInfo{
		ProcessType: targetProcess.Type,
		ProcessGUID: targetProcess.GUID,
		Type:        healthCheckTypeNone,
	}

	populateMainHealthCheck(&healthCheckInfo, targetProcess.HealthCheck)
	populateReadinessHealthCheck(&healthCheckInfo, targetProcess.ReadinessHealthCheck)

	return healthCheckInfo
}

// populateMainHealthCheck populates the main health check information.
func populateMainHealthCheck(info *HealthCheckInfo, healthCheck *capi.HealthCheck) {
	if healthCheck == nil {
		return
	}

	info.Type = healthCheck.Type
	if healthCheck.Data != nil {
		info.Timeout = healthCheck.Data.Timeout
		info.Endpoint = healthCheck.Data.Endpoint
		info.InvocationTimeout = healthCheck.Data.InvocationTimeout
		info.Interval = healthCheck.Data.Interval
	}
}

// populateReadinessHealthCheck populates the readiness health check information.
func populateReadinessHealthCheck(info *HealthCheckInfo, readinessHealthCheck *capi.ReadinessHealthCheck) {
	if readinessHealthCheck == nil {
		return
	}

	readinessType := readinessHealthCheck.Type

	info.ReadinessType = &readinessType
	if readinessHealthCheck.Data != nil {
		info.ReadinessEndpoint = readinessHealthCheck.Data.Endpoint
		info.ReadinessTimeout = readinessHealthCheck.Data.InvocationTimeout
		info.ReadinessInterval = readinessHealthCheck.Data.Interval
	}
}

// outputHealthCheckJSON outputs health check info as JSON.
func outputHealthCheckJSON(info HealthCheckInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(info)
	if err != nil {
		return fmt.Errorf("failed to encode health check info as JSON: %w", err)
	}

	return nil
}

// outputHealthCheckYAML outputs health check info as YAML.
func outputHealthCheckYAML(info HealthCheckInfo) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(info)
	if err != nil {
		return fmt.Errorf("failed to encode health check info as YAML: %w", err)
	}

	return nil
}

// outputHealthCheckTable outputs health check info as a table.
func outputHealthCheckTable(info HealthCheckInfo, appName, processType string) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	addBasicHealthCheckRows(table, info)
	addReadinessHealthCheckRows(table, info)

	_, _ = fmt.Fprintf(os.Stdout, "Health check configuration for application '%s' process '%s':\n\n", appName, processType)

	_ = table.Render()

	return nil
}

// addBasicHealthCheckRows adds basic health check rows to the table.
func addBasicHealthCheckRows(table *tablewriter.Table, info HealthCheckInfo) {
	_ = table.Append("Process Type", info.ProcessType)
	_ = table.Append("Health Check Type", info.Type)

	if info.Timeout != nil {
		_ = table.Append("Timeout", fmt.Sprintf("%d seconds", *info.Timeout))
	}

	if info.Endpoint != nil {
		_ = table.Append("Endpoint", *info.Endpoint)
	}

	if info.InvocationTimeout != nil {
		_ = table.Append("Invocation Timeout", fmt.Sprintf("%d seconds", *info.InvocationTimeout))
	}

	if info.Interval != nil {
		_ = table.Append("Interval", fmt.Sprintf("%d seconds", *info.Interval))
	}
}

// addReadinessHealthCheckRows adds readiness health check rows to the table.
func addReadinessHealthCheckRows(table *tablewriter.Table, info HealthCheckInfo) {
	if info.ReadinessType == nil {
		return
	}

	_ = table.Append("Readiness Type", *info.ReadinessType)
	if info.ReadinessEndpoint != nil {
		_ = table.Append("Readiness Endpoint", *info.ReadinessEndpoint)
	}

	if info.ReadinessTimeout != nil {
		_ = table.Append("Readiness Timeout", fmt.Sprintf("%d seconds", *info.ReadinessTimeout))
	}

	if info.ReadinessInterval != nil {
		_ = table.Append("Readiness Interval", fmt.Sprintf("%d seconds", *info.ReadinessInterval))
	}
}

// updateHealthCheck updates the health check configuration.
func updateHealthCheck(ctx context.Context, client capi.Client, targetProcess *capi.Process, appName, healthCheckType string, timeout int, endpoint string) error {
	err := validateHealthCheckType(healthCheckType)
	if err != nil {
		return err
	}

	healthCheckInfo := buildHealthCheckConfig(healthCheckType, timeout, endpoint)
	updateReq := &capi.ProcessUpdateRequest{
		HealthCheck: healthCheckInfo,
	}

	updatedProcess, err := client.Processes().Update(ctx, targetProcess.GUID, updateReq)
	if err != nil {
		return fmt.Errorf("failed to update health check: %w", err)
	}

	displayUpdateResult(appName, updatedProcess.Type, healthCheckInfo)

	return nil
}

// validateHealthCheckType validates the health check type.
func validateHealthCheckType(healthCheckType string) error {
	validTypes := []string{"port", "process", "http", healthCheckTypeNone}
	for _, vt := range validTypes {
		if healthCheckType == vt {
			return nil
		}
	}

	return fmt.Errorf("%w '%s'. Valid types: %v", capi.ErrInvalidHealthCheckType, healthCheckType, validTypes)
}

// buildHealthCheckConfig builds the health check configuration.
func buildHealthCheckConfig(healthCheckType string, timeout int, endpoint string) *capi.HealthCheck {
	if healthCheckType == healthCheckTypeNone {
		return nil
	}

	healthCheckInfo := &capi.HealthCheck{
		Type: healthCheckType,
	}

	if timeout > 0 || endpoint != "" {
		healthCheckInfo.Data = &capi.HealthCheckData{}
		if timeout > 0 {
			healthCheckInfo.Data.Timeout = &timeout
		}

		if endpoint != "" && healthCheckType == "http" {
			healthCheckInfo.Data.Endpoint = &endpoint
		}
	}

	return healthCheckInfo
}

// displayUpdateResult displays the result of health check update.
func displayUpdateResult(appName, processType string, healthCheck *capi.HealthCheck) {
	_, _ = fmt.Fprintf(os.Stdout, "Successfully updated health check for application '%s' process '%s'\n", appName, processType)

	if healthCheck != nil {
		_, _ = fmt.Fprintf(os.Stdout, "Health Check Type: %s\n", healthCheck.Type)

		if healthCheck.Data != nil && healthCheck.Data.Timeout != nil {
			_, _ = fmt.Fprintf(os.Stdout, "  Timeout: %d seconds\n", *healthCheck.Data.Timeout)
		}

		if healthCheck.Data != nil && healthCheck.Data.Endpoint != nil {
			_, _ = fmt.Fprintf(os.Stdout, "  Endpoint: %s\n", *healthCheck.Data.Endpoint)
		}
	} else {
		_, _ = os.Stdout.WriteString("Health Check Type: none\n")
	}
}

// appResourceConfig holds configuration for listing app-related resources (tasks, droplets, builds).
type appResourceConfig struct {
	appNameOrGUID string
	allPages      bool
	perPage       int
	state         string
}

// setupAppTasksParams configures query parameters for task listing.
func setupAppResourceParams(ctx context.Context, client capi.Client, config *appResourceConfig) (*capi.QueryParams, error) {
	params := capi.NewQueryParams()

	if config.perPage > 0 {
		params.PerPage = config.perPage
	}

	// Filter by app if specified
	if config.appNameOrGUID != "" {
		appGUID, err := resolveAppGUID(ctx, client, config.appNameOrGUID)
		if err != nil {
			return nil, err
		}

		params.WithFilter("app_guids", appGUID)
	}

	// Filter by state if specified
	if config.state != "" {
		params.WithFilter("states", config.state)
	}

	return params, nil
}

// appResourceCommandConfig defines the parameters for creating app resource listing commands.
type appResourceCommandConfig struct {
	use             string
	short           string
	long            string
	stateFilterDesc string
	setupParams     func(context.Context, capi.Client, *appResourceConfig) (*capi.QueryParams, error)
	fetchPages      func(context.Context, capi.Client, *capi.QueryParams, bool) (any, *capi.Pagination, error)
	outputResults   func(any, *capi.Pagination, bool) error
}

// createAppResourceCommand creates a standardized command for listing app resources.
func createAppResourceCommand(config appResourceCommandConfig) *cobra.Command {
	resourceConfig := &appResourceConfig{}

	cmd := &cobra.Command{
		Use:   config.use,
		Short: config.short,
		Long:  config.long,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				resourceConfig.appNameOrGUID = args[0]
			}

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Setup query parameters
			params, err := config.setupParams(ctx, client, resourceConfig)
			if err != nil {
				return err
			}

			// Fetch resources (all pages if requested)
			resources, pagination, err := config.fetchPages(ctx, client, params, resourceConfig.allPages)
			if err != nil {
				return err
			}

			// Output results in requested format
			return config.outputResults(resources, pagination, resourceConfig.allPages)
		},
	}

	cmd.Flags().BoolVar(&resourceConfig.allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&resourceConfig.perPage, "per-page", 0, "results per page")
	cmd.Flags().StringVar(&resourceConfig.state, "state", "", config.stateFilterDesc)

	return cmd
}

// Wrapper functions to adapt the specific resource functions to the generic interface

func fetchTaskPages(ctx context.Context, client capi.Client, params *capi.QueryParams, allPages bool) (any, *capi.Pagination, error) {
	return fetchAllTaskPages(ctx, client, params, allPages)
}

func fetchDropletPages(ctx context.Context, client capi.Client, params *capi.QueryParams, allPages bool) (any, *capi.Pagination, error) {
	return fetchAllDropletPages(ctx, client, params, allPages)
}

func fetchBuildPages(ctx context.Context, client capi.Client, params *capi.QueryParams, allPages bool) (any, *capi.Pagination, error) {
	return fetchAllBuildPages(ctx, client, params, allPages)
}

func outputTasks(resources any, pagination *capi.Pagination, allPages bool) error {
	tasks, ok := resources.([]capi.Task)
	if !ok {
		return constants.ErrInvalidResourceTypeForTasks
	}

	return outputTaskList(tasks)
}

func outputDroplets(resources any, pagination *capi.Pagination, allPages bool) error {
	droplets, ok := resources.([]capi.Droplet)
	if !ok {
		return constants.ErrInvalidResourceTypeForDroplets
	}

	return outputDropletList(droplets)
}

func outputBuilds(resources any, pagination *capi.Pagination, allPages bool) error {
	builds, ok := resources.([]capi.Build)
	if !ok {
		return constants.ErrInvalidResourceTypeForBuilds
	}

	return outputBuildList(builds)
}

// resolveAppGUID finds app GUID by name or returns GUID if already provided.
func resolveAppGUID(ctx context.Context, client capi.Client, appNameOrGUID string) (string, error) {
	// Try as GUID first
	app, err := client.Apps().Get(ctx, appNameOrGUID)
	if err == nil {
		return app.GUID, nil
	}

	// Try by name in targeted space
	appParams := capi.NewQueryParams()
	appParams.WithFilter("names", appNameOrGUID)

	if spaceGUID := viper.GetString("space_guid"); spaceGUID != "" {
		appParams.WithFilter("space_guids", spaceGUID)
	}

	apps, err := client.Apps().List(ctx, appParams)
	if err != nil {
		return "", fmt.Errorf("failed to find application: %w", err)
	}

	if len(apps.Resources) == 0 {
		return "", fmt.Errorf("%w: '%s'", capi.ErrApplicationNotFound, appNameOrGUID)
	}

	return apps.Resources[0].GUID, nil
}

// fetchAllTaskPages retrieves all pages of tasks if requested.
func fetchAllTaskPages(ctx context.Context, client capi.Client, params *capi.QueryParams, allPages bool) ([]capi.Task, *capi.Pagination, error) {
	tasks, err := client.Tasks().List(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	allTasks := tasks.Resources

	if allPages && tasks.Pagination.TotalPages > 1 {
		for page := 2; page <= tasks.Pagination.TotalPages; page++ {
			params.Page = page

			moreTasks, err := client.Tasks().List(ctx, params)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
			}

			allTasks = append(allTasks, moreTasks.Resources...)
		}
	}

	return allTasks, &tasks.Pagination, nil
}

// outputTaskList renders the task list in the requested format.
func outputTaskList(tasks []capi.Task) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(tasks)
		if err != nil {
			return fmt.Errorf("failed to encode tasks as JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)
		encoder.SetIndent(defaultJSONIndent)

		err := encoder.Encode(tasks)
		if err != nil {
			return fmt.Errorf("failed to encode tasks as JSON: %w", err)
		}

		return nil
	default:
		return renderTaskTable(tasks)
	}
}

// renderTaskTable renders tasks in table format.
func renderTaskTable(tasks []capi.Task) error {
	if len(tasks) == 0 {
		_, _ = os.Stdout.WriteString("No tasks found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("GUID", "Name", "State", "App", "Command", "Created")

	for _, task := range tasks {
		createdAt := ""
		if !task.CreatedAt.IsZero() {
			createdAt = task.CreatedAt.Format(TimeFormatDisplay)
		}

		appName := ""
		if task.Relationships != nil && task.Relationships.App != nil {
			appName = task.Relationships.App.Data.GUID
		}

		command := ""
		if task.Command != "" {
			command = task.Command
			if len(command) > commandTruncateLength {
				command = command[:commandTruncateLength-3] + "..."
			}
		}

		_ = table.Append(task.GUID, task.Name, task.State, appName, command, createdAt)
	}

	_ = table.Render()

	return nil
}

func newAppsTasksCommand() *cobra.Command {
	return createAppResourceCommand(appResourceCommandConfig{
		use:             "tasks [APP_NAME_OR_GUID]",
		short:           "List application tasks",
		long:            "List all tasks for an application",
		stateFilterDesc: "filter by task state",
		setupParams:     setupAppResourceParams,
		fetchPages:      fetchTaskPages,
		outputResults:   outputTasks,
	})
}

func newAppsDeploymentsCommand() *cobra.Command {
	var config appsDeploymentsConfig

	cmd := &cobra.Command{
		Use:   "deployments [APP_NAME_OR_GUID]",
		Short: "List application deployments",
		Long:  "List all deployments for an application",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config.appNameOrGUID = extractAppNameFromArgs(args)

			return runAppsDeployments(cmd, &config)
		},
	}

	setupAppsDeploymentsFlags(cmd, &config)

	return cmd
}

type appsDeploymentsConfig struct {
	appNameOrGUID string
	allPages      bool
	perPage       int
	state         string
}

func extractAppNameFromArgs(args []string) string {
	if len(args) > 0 {
		return args[0]
	}

	return ""
}

func setupAppsDeploymentsFlags(cmd *cobra.Command, config *appsDeploymentsConfig) {
	cmd.Flags().BoolVar(&config.allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&config.perPage, "per-page", 0, "results per page")
	cmd.Flags().StringVar(&config.state, "state", "", "filter by deployment state")
}

func runAppsDeployments(cmd *cobra.Command, config *appsDeploymentsConfig) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()
	params := buildAppsDeploymentsParams(config)

	if config.appNameOrGUID != "" {
		err := addAppFilterToDeployments(ctx, client, params, config.appNameOrGUID)
		if err != nil {
			return err
		}
	}

	deployments, err := fetchAppsDeployments(ctx, client, params, config.allPages)
	if err != nil {
		return err
	}

	return outputAppsDeployments(deployments)
}

func buildAppsDeploymentsParams(config *appsDeploymentsConfig) *capi.QueryParams {
	params := capi.NewQueryParams()

	if config.perPage > 0 {
		params.PerPage = config.perPage
	}

	if config.state != "" {
		params.WithFilter("states", config.state)
	}

	return params
}

func addAppFilterToDeployments(ctx context.Context, client capi.Client, params *capi.QueryParams, appNameOrGUID string) error {
	app, err := client.Apps().Get(ctx, appNameOrGUID)
	if err != nil {
		return tryAppFilterByNameForDeployments(ctx, client, params, appNameOrGUID)
	}

	params.WithFilter("app_guids", app.GUID)

	return nil
}

func tryAppFilterByNameForDeployments(ctx context.Context, client capi.Client, params *capi.QueryParams, appName string) error {
	appParams := capi.NewQueryParams()
	appParams.WithFilter("names", appName)

	if spaceGUID := viper.GetString("space_guid"); spaceGUID != "" {
		appParams.WithFilter("space_guids", spaceGUID)
	}

	apps, err := client.Apps().List(ctx, appParams)
	if err != nil {
		return fmt.Errorf("failed to find application: %w", err)
	}

	if len(apps.Resources) == 0 {
		return fmt.Errorf("%w: '%s'", capi.ErrApplicationNotFound, appName)
	}

	params.WithFilter("app_guids", apps.Resources[0].GUID)

	return nil
}

func fetchAppsDeployments(ctx context.Context, client capi.Client, params *capi.QueryParams, allPages bool) ([]capi.Deployment, error) {
	deployments, err := client.Deployments().List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	allDeployments := deployments.Resources
	if !allPages || deployments.Pagination.TotalPages <= 1 {
		return allDeployments, nil
	}

	return fetchRemainingDeploymentPages(ctx, client, params, deployments, allDeployments)
}

func fetchRemainingDeploymentPages(ctx context.Context, client capi.Client, params *capi.QueryParams, firstPage *capi.ListResponse[capi.Deployment], allDeployments []capi.Deployment) ([]capi.Deployment, error) {
	for page := 2; page <= firstPage.Pagination.TotalPages; page++ {
		params.Page = page

		moreDeployments, err := client.Deployments().List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		allDeployments = append(allDeployments, moreDeployments.Resources...)
	}

	return allDeployments, nil
}

func outputAppsDeployments(deployments []capi.Deployment) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return outputAppsDeploymentsJSON(deployments)
	case OutputFormatYAML:
		return outputAppsDeploymentsYAML(deployments)
	default:
		return outputAppsDeploymentsTable(deployments)
	}
}

func outputAppsDeploymentsJSON(deployments []capi.Deployment) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(deployments)
	if err != nil {
		return fmt.Errorf("failed to encode deployments as JSON: %w", err)
	}

	return nil
}

func outputAppsDeploymentsYAML(deployments []capi.Deployment) error {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(defaultJSONIndent)

	err := encoder.Encode(deployments)
	if err != nil {
		return fmt.Errorf("failed to encode deployments as YAML: %w", err)
	}

	return nil
}

func outputAppsDeploymentsTable(deployments []capi.Deployment) error {
	if len(deployments) == 0 {
		_, _ = os.Stdout.WriteString("No deployments found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("GUID", "Status", "Strategy", "App", "Created", "Updated")

	for _, deployment := range deployments {
		createdAt := formatDeploymentTime(deployment.CreatedAt)
		updatedAt := formatDeploymentTime(deployment.UpdatedAt)
		appName := extractAppGUIDFromDeployment(deployment)

		_ = table.Append(deployment.GUID, deployment.Status.Value, deployment.Strategy, appName, createdAt, updatedAt)
	}

	_ = table.Render()

	return nil
}

func formatDeploymentTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.Format(TimeFormatDisplay)
}

func extractAppGUIDFromDeployment(deployment capi.Deployment) string {
	if deployment.Relationships != nil && deployment.Relationships.App != nil {
		return deployment.Relationships.App.Data.GUID
	}

	return ""
}

func newAppsPackagesCommand() *cobra.Command {
	var (
		appNameOrGUID string
		allPages      bool
		perPage       int
		state         string
	)

	cmd := &cobra.Command{
		Use:   "packages [APP_NAME_OR_GUID]",
		Short: "List application packages",
		Long:  "List all packages for an application",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				appNameOrGUID = args[0]
			}

			client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
			if err != nil {
				return err
			}

			ctx := context.Background()

			// Setup query parameters
			params, err := setupPackagesQueryParams(ctx, client, appNameOrGUID, perPage, state)
			if err != nil {
				return err
			}

			// Fetch packages with pagination
			packages, err := fetchPackagesWithPagination(ctx, client, params, allPages)
			if err != nil {
				return err
			}

			// Output results
			return renderPackagesOutput(packages)
		},
	}

	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", 0, "results per page")
	cmd.Flags().StringVar(&state, "state", "", "filter by package state")

	return cmd
}

// setupPackagesQueryParams creates and configures query parameters for packages.
func setupPackagesQueryParams(ctx context.Context, client capi.Client, appNameOrGUID string, perPage int, state string) (*capi.QueryParams, error) {
	params := capi.NewQueryParams()

	if perPage > 0 {
		params.PerPage = perPage
	}

	// Filter by app if specified
	if appNameOrGUID != "" {
		appGUID, err := resolveAppGUIDForPackages(ctx, client, appNameOrGUID)
		if err != nil {
			return nil, err
		}

		params.WithFilter("app_guids", appGUID)
	}

	// Filter by state if specified
	if state != "" {
		params.WithFilter("states", state)
	}

	return params, nil
}

// resolveAppGUIDForPackages resolves app name or GUID to GUID.
func resolveAppGUIDForPackages(ctx context.Context, client capi.Client, nameOrGUID string) (string, error) {
	// Try by GUID first
	app, err := client.Apps().Get(ctx, nameOrGUID)
	if err == nil {
		return app.GUID, nil
	}

	// Try by name in targeted space
	appParams := capi.NewQueryParams()
	appParams.WithFilter("names", nameOrGUID)

	if spaceGUID := viper.GetString("space_guid"); spaceGUID != "" {
		appParams.WithFilter("space_guids", spaceGUID)
	}

	apps, err := client.Apps().List(ctx, appParams)
	if err != nil {
		return "", fmt.Errorf("failed to find application: %w", err)
	}

	if len(apps.Resources) == 0 {
		return "", fmt.Errorf("%w: %s", constants.ErrApplicationNotFound, nameOrGUID)
	}

	return apps.Resources[0].GUID, nil
}

// fetchPackagesWithPagination fetches packages with optional pagination.
func fetchPackagesWithPagination(ctx context.Context, client capi.Client, params *capi.QueryParams, allPages bool) ([]capi.Package, error) {
	packages, err := client.Packages().List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	// Return first page if not fetching all
	if !allPages || packages.Pagination.TotalPages <= 1 {
		return packages.Resources, nil
	}

	// Fetch remaining pages
	allPackages := make([]capi.Package, 0, len(packages.Resources))
	allPackages = append(allPackages, packages.Resources...)

	for page := 2; page <= packages.Pagination.TotalPages; page++ {
		params.Page = page

		morePackages, err := client.Packages().List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		allPackages = append(allPackages, morePackages.Resources...)
	}

	return allPackages, nil
}

// renderPackagesOutput renders packages in the specified output format.
func renderPackagesOutput(packages []capi.Package) error {
	output := viper.GetString("output")

	switch output {
	case OutputFormatJSON:
		return StandardJSONRenderer(packages)
	case OutputFormatYAML:
		return StandardYAMLRenderer(packages)
	default:
		return renderPackagesTable(packages)
	}
}

// renderPackagesTable renders packages as a table.
func renderPackagesTable(packages []capi.Package) error {
	if len(packages) == 0 {
		_, _ = os.Stdout.WriteString("No packages found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("GUID", "Type", "State", "App", "Created", "Updated")

	for _, pkg := range packages {
		row := buildPackageTableRow(pkg)
		_ = table.Append(row[0], row[1], row[2], row[3], row[4], row[5])
	}

	_ = table.Render()

	return nil
}

// buildPackageTableRow builds a table row for a package.
func buildPackageTableRow(pkg capi.Package) []string {
	createdAt := ""
	if !pkg.CreatedAt.IsZero() {
		createdAt = pkg.CreatedAt.Format(TimeFormatDisplay)
	}

	updatedAt := ""
	if !pkg.UpdatedAt.IsZero() {
		updatedAt = pkg.UpdatedAt.Format(TimeFormatDisplay)
	}

	appName := ""
	if pkg.Relationships != nil && pkg.Relationships.App != nil {
		appName = pkg.Relationships.App.Data.GUID
	}

	return []string{pkg.GUID, pkg.Type, pkg.State, appName, createdAt, updatedAt}
}

// fetchAllDropletPages retrieves all pages of droplets if requested.
func fetchAllDropletPages(ctx context.Context, client capi.Client, params *capi.QueryParams, allPages bool) ([]capi.Droplet, *capi.Pagination, error) {
	droplets, err := client.Droplets().List(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list droplets: %w", err)
	}

	allDroplets := droplets.Resources

	if allPages && droplets.Pagination.TotalPages > 1 {
		for page := 2; page <= droplets.Pagination.TotalPages; page++ {
			params.Page = page

			moreDroplets, err := client.Droplets().List(ctx, params)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
			}

			allDroplets = append(allDroplets, moreDroplets.Resources...)
		}
	}

	return allDroplets, &droplets.Pagination, nil
}

// outputDropletList renders the droplet list in the requested format.
func outputDropletList(droplets []capi.Droplet) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(droplets)
		if err != nil {
			return fmt.Errorf("failed to encode droplets as JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)
		encoder.SetIndent(defaultJSONIndent)

		err := encoder.Encode(droplets)
		if err != nil {
			return fmt.Errorf("failed to encode droplets as YAML: %w", err)
		}

		return nil
	default:
		return renderDropletTable(droplets)
	}
}

// renderDropletTable renders droplets in table format.
func renderDropletTable(droplets []capi.Droplet) error {
	if len(droplets) == 0 {
		_, _ = os.Stdout.WriteString("No droplets found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("GUID", "State", "Stack", "App", "Current", "Created")

	for _, droplet := range droplets {
		createdAt := ""
		if !droplet.CreatedAt.IsZero() {
			createdAt = droplet.CreatedAt.Format(TimeFormatDisplay)
		}

		appName := ""
		if droplet.Relationships != nil && droplet.Relationships.App != nil {
			appName = droplet.Relationships.App.Data.GUID
		}

		stack := ""
		if droplet.Stack != nil && *droplet.Stack != "" {
			stack = *droplet.Stack
		}

		current := ""
		// Note: Current droplet identification would require additional API call

		_ = table.Append(droplet.GUID, droplet.State, stack, appName, current, createdAt)
	}

	_ = table.Render()

	return nil
}

func newAppsDropletsCommand() *cobra.Command {
	return createAppResourceCommand(appResourceCommandConfig{
		use:             "droplets [APP_NAME_OR_GUID]",
		short:           "List application droplets",
		long:            "List all droplets for an application",
		stateFilterDesc: "filter by droplet state",
		setupParams:     setupAppResourceParams,
		fetchPages:      fetchDropletPages,
		outputResults:   outputDroplets,
	})
}

// fetchAllBuildPages retrieves all pages of builds if requested.
func fetchAllBuildPages(ctx context.Context, client capi.Client, params *capi.QueryParams, allPages bool) ([]capi.Build, *capi.Pagination, error) {
	builds, err := client.Builds().List(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list builds: %w", err)
	}

	allBuilds := builds.Resources

	if allPages && builds.Pagination.TotalPages > 1 {
		for page := 2; page <= builds.Pagination.TotalPages; page++ {
			params.Page = page

			moreBuilds, err := client.Builds().List(ctx, params)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
			}

			allBuilds = append(allBuilds, moreBuilds.Resources...)
		}
	}

	return allBuilds, &builds.Pagination, nil
}

// outputBuildList renders the build list in the requested format.
func outputBuildList(builds []capi.Build) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(builds)
		if err != nil {
			return fmt.Errorf("failed to encode builds as JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)
		encoder.SetIndent(defaultJSONIndent)

		err := encoder.Encode(builds)
		if err != nil {
			return fmt.Errorf("failed to encode builds as YAML: %w", err)
		}

		return nil
	default:
		return renderBuildTable(builds)
	}
}

// renderBuildTable renders builds in table format.
func renderBuildTable(builds []capi.Build) error {
	if len(builds) == 0 {
		_, _ = os.Stdout.WriteString("No builds found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("GUID", "State", "App", "Package", "Created", "Updated")

	for _, build := range builds {
		createdAt := ""
		if !build.CreatedAt.IsZero() {
			createdAt = build.CreatedAt.Format(TimeFormatDisplay)
		}

		updatedAt := ""
		if !build.UpdatedAt.IsZero() {
			updatedAt = build.UpdatedAt.Format(TimeFormatDisplay)
		}

		appName := ""
		if build.Relationships != nil && build.Relationships.App != nil {
			appName = build.Relationships.App.Data.GUID
		}

		packageGUID := ""
		if build.Package != nil {
			packageGUID = build.Package.GUID
		}

		_ = table.Append(build.GUID, build.State, appName, packageGUID, createdAt, updatedAt)
	}

	_ = table.Render()

	return nil
}

func newAppsBuildsCommand() *cobra.Command {
	return createAppResourceCommand(appResourceCommandConfig{
		use:             "builds [APP_NAME_OR_GUID]",
		short:           "List application builds",
		long:            "List all builds for an application",
		stateFilterDesc: "filter by build state",
		setupParams:     setupAppResourceParams,
		fetchPages:      fetchBuildPages,
		outputResults:   outputBuilds,
	})
}

func newAppsFeaturesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "features",
		Short: "Manage application features",
		Long:  "Manage application features including listing, getting, enabling, and disabling features",
	}

	cmd.AddCommand(newAppsFeaturesListCommand())
	cmd.AddCommand(newAppsFeaturesGetCommand())
	cmd.AddCommand(newAppsFeaturesEnableCommand())
	cmd.AddCommand(newAppsFeaturesDisableCommand())

	return cmd
}

func newAppsFeaturesListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list APP_NAME_OR_GUID",
		Short: "List application features",
		Long:  "List all features for an application",
		Args:  cobra.ExactArgs(1),
		RunE:  runAppFeaturesList,
	}
}

type featureInfo struct {
	Name        string `json:"name"        yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Enabled     bool   `json:"enabled"     yaml:"enabled"`
	Status      string `json:"status"      yaml:"status"`
}

func runAppFeaturesList(cmd *cobra.Command, args []string) error {
	appNameOrGUID := args[0]

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	app, err := findAppByNameOrGUID(ctx, client, appNameOrGUID)
	if err != nil {
		return err
	}

	features, err := client.Apps().GetFeatures(ctx, app.GUID)
	if err != nil {
		return fmt.Errorf("getting app features: %w", err)
	}

	featureInfos := buildFeatureInfoList(features.Resources)

	return outputFeaturesList(featureInfos, app.Name)
}

func findAppByNameOrGUID(ctx context.Context, client capi.Client, appNameOrGUID string) (*capi.App, error) {
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
			return nil, fmt.Errorf("%w: %s", constants.ErrApplicationNotFound, appNameOrGUID)
		}

		app = &apps.Resources[0]
	}

	return app, nil
}

func buildFeatureInfoList(features []capi.AppFeature) []featureInfo {
	featureInfos := make([]featureInfo, 0, len(features))

	for _, feature := range features {
		status := "disabled"
		if feature.Enabled {
			status = "enabled"
		}

		featureInfos = append(featureInfos, featureInfo{
			Name:        feature.Name,
			Description: feature.Description,
			Enabled:     feature.Enabled,
			Status:      status,
		})
	}

	return featureInfos
}

func outputFeaturesList(featureInfos []featureInfo, appName string) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(featureInfos)
		if err != nil {
			return fmt.Errorf("failed to encode features as JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(featureInfos)
		if err != nil {
			return fmt.Errorf("failed to encode features as YAML: %w", err)
		}

		return nil
	default:
		return outputFeaturesTable(featureInfos, appName)
	}
}

func outputFeaturesTable(featureInfos []featureInfo, appName string) error {
	if len(featureInfos) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "No features found for application %s\n", appName)

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Feature", "Status", "Description")

	for _, feature := range featureInfos {
		description := feature.Description
		if len(description) > descriptionTruncateLength {
			description = description[:descriptionTruncateLength-3] + "..."
		}

		_ = table.Append(feature.Name, feature.Status, description)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Features for application '%s':\n\n", appName)

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render features table: %w", err)
	}

	return nil
}

func runAppsFeaturesGet(cmd *cobra.Command, args []string) error {
	appNameOrGUID := args[0]
	featureName := args[1]

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Find app
	app, err := findAppByNameOrGUID(ctx, client, appNameOrGUID)
	if err != nil {
		return err
	}

	feature, err := client.Apps().GetFeature(ctx, app.GUID, featureName)
	if err != nil {
		return fmt.Errorf("getting app feature '%s': %w", featureName, err)
	}

	return renderAppFeature(feature, app.Name, featureName)
}

func renderAppFeature(feature *capi.AppFeature, appName, featureName string) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(feature)
		if err != nil {
			return fmt.Errorf("failed to encode feature to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(feature)
		if err != nil {
			return fmt.Errorf("failed to encode feature to YAML: %w", err)
		}

		return nil
	default:
		status := "disabled"
		if feature.Enabled {
			status = "enabled"
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Property", "Value")
		_ = table.Append("Name", feature.Name)
		_ = table.Append("Status", status)
		_ = table.Append("Description", feature.Description)

		_, _ = fmt.Fprintf(os.Stdout, "Feature '%s' for application '%s':\n\n", featureName, appName)

		err := table.Render()
		if err != nil {
			return fmt.Errorf("failed to render table: %w", err)
		}

		return nil
	}
}

func newAppsFeaturesGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get APP_NAME_OR_GUID FEATURE_NAME",
		Short: "Get details for a specific application feature",
		Long:  "Get detailed information about a specific application feature",
		Args:  cobra.ExactArgs(unsetEnvExactArgs),
		RunE:  runAppsFeaturesGet,
	}
}

func newAppsFeaturesEnableCommand() *cobra.Command {
	return createFeatureToggleCommand(
		"enable APP_NAME_OR_GUID FEATURE_NAME",
		"Enable a specific application feature",
		"Enable a specific application feature",
		true,
		"✓ Feature '%s' has been enabled for application '%s'\n",
	)
}

func newAppsFeaturesDisableCommand() *cobra.Command {
	return createFeatureToggleCommand(
		"disable APP_NAME_OR_GUID FEATURE_NAME",
		"Disable a specific application feature",
		"Disable a specific application feature",
		false,
		"✓ Feature '%s' has been disabled for application '%s'\n",
	)
}
