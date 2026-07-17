package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/fivetwenty-io/capi/v3/pkg/capi"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// validateFilePath validates that a file path is safe to read.
func validateFilePathBuildpacks(filePath string) error {
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

// NewBuildpacksCommand creates the buildpacks command group.
func NewBuildpacksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "buildpacks",
		Aliases: []string{"buildpack"},
		Short:   "Manage buildpacks",
		Long:    "List and manage Cloud Foundry buildpacks",
	}

	cmd.AddCommand(newBuildpacksListCommand())
	cmd.AddCommand(newBuildpacksGetCommand())
	cmd.AddCommand(newBuildpacksCreateCommand())
	cmd.AddCommand(newBuildpacksUpdateCommand())
	cmd.AddCommand(newBuildpacksDeleteCommand())
	cmd.AddCommand(newBuildpacksUploadCommand())

	return cmd
}

func newBuildpacksListCommand() *cobra.Command {
	var (
		allPages bool
		perPage  int
		enabled  bool
		stack    string
	)

	cmd := &cobra.Command{
		Use:   List,
		Short: "List buildpacks",
		Long:  "List all buildpacks",
		RunE: func(cmd *cobra.Command, args []string) error {
			filters := &buildpacksListFilters{
				allPages: allPages,
				perPage:  perPage,
				enabled:  enabled,
				stack:    stack,
				cmd:      cmd,
			}

			return runBuildpacksList(filters)
		},
	}

	cmd.Flags().BoolVar(&allPages, "all", false, "fetch all pages")
	cmd.Flags().IntVar(&perPage, "per-page", constants.DefaultPageSize, "results per page")
	cmd.Flags().BoolVar(&enabled, "enabled", false, "filter by enabled buildpacks")
	cmd.Flags().StringVar(&stack, "stack", "", "filter by stack")

	return cmd
}

type buildpacksListFilters struct {
	allPages bool
	perPage  int
	enabled  bool
	stack    string
	cmd      *cobra.Command
}

func runBuildpacksList(filters *buildpacksListFilters) error {
	client, err := CreateClientWithAPI(filters.cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()
	params := buildBuildpacksListParams(filters)

	buildpacks, err := client.Buildpacks().List(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to list buildpacks: %w", err)
	}

	allBuildpacks, err := fetchAllBuildpacksPages(ctx, client, buildpacks, params, filters.allPages)
	if err != nil {
		return err
	}

	return outputBuildpacksList(allBuildpacks, buildpacks, filters.allPages)
}

func buildBuildpacksListParams(filters *buildpacksListFilters) *capi.QueryParams {
	params := capi.NewQueryParams()
	params.PerPage = filters.perPage

	if filters.cmd.Flags().Changed("enabled") {
		if filters.enabled {
			params.WithFilter("enabled", "true")
		} else {
			params.WithFilter("enabled", "false")
		}
	}

	if filters.stack != "" {
		params.WithFilter("stacks", filters.stack)
	}

	return params
}

func fetchAllBuildpacksPages(ctx context.Context, client capi.Client, buildpacks *capi.BuildpacksList, params *capi.QueryParams, allPages bool) ([]capi.Buildpack, error) {
	allBuildpacks := buildpacks.Resources
	if !allPages || buildpacks.Pagination.TotalPages <= 1 {
		return allBuildpacks, nil
	}

	for page := 2; page <= buildpacks.Pagination.TotalPages; page++ {
		params.Page = page

		moreBuildpacks, err := client.Buildpacks().List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
		}

		allBuildpacks = append(allBuildpacks, moreBuildpacks.Resources...)
	}

	return allBuildpacks, nil
}

func outputBuildpacksList(allBuildpacks []capi.Buildpack, buildpacks *capi.BuildpacksList, allPages bool) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return outputBuildpacksListJSON(allBuildpacks)
	case OutputFormatYAML:
		return outputBuildpacksListYAML(allBuildpacks)
	default:
		return outputBuildpacksListTable(allBuildpacks, buildpacks, allPages)
	}
}

func outputBuildpacksListJSON(buildpacks []capi.Buildpack) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(buildpacks)
	if err != nil {
		return fmt.Errorf("failed to encode buildpacks to JSON: %w", err)
	}

	return nil
}

func outputBuildpacksListYAML(buildpacks []capi.Buildpack) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(buildpacks)
	if err != nil {
		return fmt.Errorf("failed to encode buildpacks to YAML: %w", err)
	}

	return nil
}

func outputBuildpacksListTable(allBuildpacks []capi.Buildpack, buildpacks *capi.BuildpacksList, allPages bool) error {
	if len(allBuildpacks) == 0 {
		_, _ = os.Stdout.WriteString("No buildpacks found\n")

		return nil
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Position", "Name", "Stack", "State", "Enabled", "Locked", "Filename")

	for _, bp := range allBuildpacks {
		info := extractBuildpackTableInfo(bp)
		_ = table.Append(strconv.Itoa(bp.Position), bp.Name, info.stack, bp.State, info.enabled, info.locked, info.filename)
	}

	_ = table.Render()

	if !allPages && buildpacks.Pagination.TotalPages > 1 {
		_, _ = fmt.Fprintf(os.Stdout, "\nShowing page 1 of %d. Use --all to fetch all pages.\n", buildpacks.Pagination.TotalPages)
	}

	return nil
}

type buildpackTableInfo struct {
	stack    string
	enabled  string
	locked   string
	filename string
}

func extractBuildpackTableInfo(bp capi.Buildpack) buildpackTableInfo {
	return buildpackTableInfo{
		stack:    formatBuildpackStack(bp.Stack),
		enabled:  formatBuildpackEnabled(bp.Enabled),
		locked:   formatBuildpackLocked(bp.Locked),
		filename: formatBuildpackFilename(bp.Filename),
	}
}

func formatBuildpackStack(stack *string) string {
	if stack != nil {
		return *stack
	}

	return "any"
}

func formatBuildpackEnabled(enabled bool) string {
	if enabled {
		return Yes
	}

	return "no"
}

func formatBuildpackLocked(locked bool) string {
	if locked {
		return Yes
	}

	return "no"
}

func formatBuildpackFilename(filename *string) string {
	if filename != nil {
		return *filename
	}

	return ""
}

func newBuildpacksGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get BUILDPACK_NAME_OR_GUID",
		Short: "Get buildpack details",
		Long:  "Display detailed information about a specific buildpack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuildpacksGet(cmd, args[0])
		},
	}
}

func runBuildpacksGet(cmd *cobra.Command, nameOrGUID string) error {
	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	bp, err := resolveBuildpack(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	return outputBuildpackDetails(bp)
}

func resolveBuildpack(ctx context.Context, client capi.Client, nameOrGUID string) (*capi.Buildpack, error) {
	// Try to get by GUID first
	buildpack, err := client.Buildpacks().Get(ctx, nameOrGUID)
	if err == nil {
		return buildpack, nil
	}

	// If not found by GUID, try by name
	params := capi.NewQueryParams()
	params.WithFilter("names", nameOrGUID)

	buildpacks, err := client.Buildpacks().List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find buildpack: %w", err)
	}

	if len(buildpacks.Resources) == 0 {
		return nil, fmt.Errorf("%w: '%s'", capi.ErrBuildpackNotFound, nameOrGUID)
	}

	return &buildpacks.Resources[0], nil
}

func outputBuildpackDetails(buildpack *capi.Buildpack) error {
	output := viper.GetString("output")
	switch output {
	case OutputFormatJSON:
		return outputBuildpackDetailsJSON(buildpack)
	case OutputFormatYAML:
		return outputBuildpackDetailsYAML(buildpack)
	default:
		return outputBuildpackDetailsTable(buildpack)
	}
}

func outputBuildpackDetailsJSON(buildpack *capi.Buildpack) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(buildpack)
	if err != nil {
		return fmt.Errorf("failed to encode buildpack to JSON: %w", err)
	}

	return nil
}

func outputBuildpackDetailsYAML(buildpack *capi.Buildpack) error {
	encoder := yaml.NewEncoder(os.Stdout)

	err := encoder.Encode(buildpack)
	if err != nil {
		return fmt.Errorf("failed to encode buildpack to YAML: %w", err)
	}

	return nil
}

func outputBuildpackDetailsTable(buildpack *capi.Buildpack) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	addBuildpackBasicInfo(table, buildpack)
	addBuildpackOptionalInfo(table, buildpack)

	_, _ = fmt.Fprintf(os.Stdout, "Buildpack: %s\n\n", buildpack.Name)

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	return nil
}

func addBuildpackBasicInfo(table *tablewriter.Table, buildpack *capi.Buildpack) {
	_ = table.Append("Name", buildpack.Name)
	_ = table.Append("GUID", buildpack.GUID)
	_ = table.Append("Position", strconv.Itoa(buildpack.Position))
	_ = table.Append("State", buildpack.State)
	_ = table.Append("Enabled", strconv.FormatBool(buildpack.Enabled))
	_ = table.Append("Locked", strconv.FormatBool(buildpack.Locked))
	_ = table.Append("Lifecycle", buildpack.Lifecycle)
	_ = table.Append("Created", buildpack.CreatedAt.Format(TimeFormatDisplay))
	_ = table.Append("Updated", buildpack.UpdatedAt.Format(TimeFormatDisplay))
}

func addBuildpackOptionalInfo(table *tablewriter.Table, buildpack *capi.Buildpack) {
	if buildpack.Stack != nil {
		_ = table.Append("Stack", *buildpack.Stack)
	}

	if buildpack.Filename != nil {
		_ = table.Append("Filename", *buildpack.Filename)
	}
}

func newBuildpacksCreateCommand() *cobra.Command {
	var (
		name      string
		stack     string
		position  int
		enabled   bool
		locked    bool
		lifecycle string
	)

	cmd := &cobra.Command{
		Use:   Create,
		Short: "Create a buildpack",
		Long:  "Create a new buildpack",
		RunE:  runBuildpacksCreate,
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "buildpack name (required)")
	cmd.Flags().StringVar(&stack, "stack", "", "stack name")
	cmd.Flags().IntVarP(&position, "position", "p", 0, "buildpack position")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "enable the buildpack")
	cmd.Flags().BoolVar(&locked, "locked", false, "lock the buildpack")
	cmd.Flags().StringVar(&lifecycle, "lifecycle", "", "lifecycle type")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func runBuildpacksCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		return capi.ErrBuildpackNameRequired
	}

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	createReq := buildBuildpackCreateRequest(cmd)
	createReq.Name = name

	buildpack, err := client.Buildpacks().Create(ctx, createReq)
	if err != nil {
		return fmt.Errorf("failed to create buildpack: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully created buildpack '%s'\n", buildpack.Name)
	_, _ = fmt.Fprintf(os.Stdout, "  GUID:     %s\n", buildpack.GUID)
	_, _ = fmt.Fprintf(os.Stdout, "  Position: %d\n", buildpack.Position)

	return nil
}

func buildBuildpackCreateRequest(cmd *cobra.Command) *capi.BuildpackCreateRequest {
	createReq := &capi.BuildpackCreateRequest{}

	if stack, _ := cmd.Flags().GetString("stack"); stack != "" {
		createReq.Stack = &stack
	}

	if cmd.Flags().Changed("position") {
		position, _ := cmd.Flags().GetInt("position")
		createReq.Position = &position
	}

	if cmd.Flags().Changed("enabled") {
		enabled, _ := cmd.Flags().GetBool("enabled")
		createReq.Enabled = &enabled
	}

	if cmd.Flags().Changed("locked") {
		locked, _ := cmd.Flags().GetBool("locked")
		createReq.Locked = &locked
	}

	if lifecycle, _ := cmd.Flags().GetString("lifecycle"); lifecycle != "" {
		createReq.Lifecycle = &lifecycle
	}

	return createReq
}

func newBuildpacksUpdateCommand() *cobra.Command {
	var (
		newName   string
		stack     string
		position  int
		enabled   bool
		locked    bool
		lifecycle string
	)

	cmd := &cobra.Command{
		Use:   "update BUILDPACK_NAME_OR_GUID",
		Short: "Update a buildpack",
		Long:  "Update an existing buildpack",
		Args:  cobra.ExactArgs(1),
		RunE:  runBuildpacksUpdate,
	}

	cmd.Flags().StringVar(&newName, "name", "", "new buildpack name")
	cmd.Flags().StringVar(&stack, "stack", "", "stack name")
	cmd.Flags().IntVarP(&position, "position", "p", 0, "buildpack position")
	cmd.Flags().BoolVar(&enabled, "enabled", false, "enable the buildpack")
	cmd.Flags().BoolVar(&locked, "locked", false, "lock the buildpack")
	cmd.Flags().StringVar(&lifecycle, "lifecycle", "", "lifecycle type")

	return cmd
}

func runBuildpacksUpdate(cmd *cobra.Command, args []string) error {
	nameOrGUID := args[0]

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	buildpack, err := findBuildpackByNameOrGUID(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	updateReq := buildBuildpackUpdateRequest(cmd)

	updatedBP, err := client.Buildpacks().Update(ctx, buildpack.GUID, updateReq)
	if err != nil {
		return fmt.Errorf("failed to update buildpack: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Successfully updated buildpack '%s'\n", updatedBP.Name)

	return nil
}

func findBuildpackByNameOrGUID(ctx context.Context, client capi.Client, nameOrGUID string) (*capi.Buildpack, error) {
	buildpack, err := client.Buildpacks().Get(ctx, nameOrGUID)
	if err != nil {
		params := capi.NewQueryParams()
		params.WithFilter("names", nameOrGUID)

		buildpacks, err := client.Buildpacks().List(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to find buildpack: %w", err)
		}

		if len(buildpacks.Resources) == 0 {
			return nil, fmt.Errorf("%w: '%s'", capi.ErrBuildpackNotFound, nameOrGUID)
		}

		buildpack = &buildpacks.Resources[0]
	}

	return buildpack, nil
}

func buildBuildpackUpdateRequest(cmd *cobra.Command) *capi.BuildpackUpdateRequest {
	updateReq := &capi.BuildpackUpdateRequest{}

	if newName, _ := cmd.Flags().GetString("name"); newName != "" {
		updateReq.Name = &newName
	}

	if stack, _ := cmd.Flags().GetString("stack"); stack != "" {
		updateReq.Stack = &stack
	}

	if cmd.Flags().Changed("position") {
		position, _ := cmd.Flags().GetInt("position")
		updateReq.Position = &position
	}

	if cmd.Flags().Changed("enabled") {
		enabled, _ := cmd.Flags().GetBool("enabled")
		updateReq.Enabled = &enabled
	}

	if cmd.Flags().Changed("locked") {
		locked, _ := cmd.Flags().GetBool("locked")
		updateReq.Locked = &locked
	}

	return updateReq
}

func newBuildpacksDeleteCommand() *cobra.Command {
	return createBuildpackDeleteCommand()
}

func newBuildpacksUploadCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "upload BUILDPACK_NAME_OR_GUID BUILDPACK_FILE",
		Short: "Upload buildpack bits",
		Long:  "Upload a buildpack zip file to an existing buildpack",
		Args:  cobra.ExactArgs(constants.MinimumArgumentCount),
		RunE:  runBuildpacksUpload,
	}
}

func runBuildpacksUpload(cmd *cobra.Command, args []string) error {
	nameOrGUID := args[0]
	buildpackFile := args[1]

	client, err := CreateClientWithAPI(cmd.Flag("api").Value.String())
	if err != nil {
		return err
	}

	ctx := context.Background()

	buildpack, err := findBuildpackByNameOrGUID(ctx, client, nameOrGUID)
	if err != nil {
		return err
	}

	buildpackBits, err := openBuildpackFile(buildpackFile)
	if err != nil {
		return err
	}

	defer func() {
		err := buildpackBits.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close buildpack file: %v\n", err)
		}
	}()

	updatedBP, err := client.Buildpacks().Upload(ctx, buildpack.GUID, buildpackBits)
	if err != nil {
		return fmt.Errorf("failed to upload buildpack: %w", err)
	}

	displayUploadResult(updatedBP)

	return nil
}

func openBuildpackFile(buildpackFile string) (*os.File, error) {
	err := validateFilePathBuildpacks(buildpackFile)
	if err != nil {
		return nil, fmt.Errorf("invalid buildpack file: %w", err)
	}

	buildpackBits, err := os.Open(filepath.Clean(buildpackFile))
	if err != nil {
		return nil, fmt.Errorf("failed to open buildpack file: %w", err)
	}

	return buildpackBits, nil
}

func displayUploadResult(updatedBP *capi.Buildpack) {
	_, _ = fmt.Fprintf(os.Stdout, "Successfully uploaded buildpack bits to '%s'\n", updatedBP.Name)
	_, _ = fmt.Fprintf(os.Stdout, "  State: %s\n", updatedBP.State)

	if updatedBP.Filename != nil {
		_, _ = fmt.Fprintf(os.Stdout, "  Filename: %s\n", *updatedBP.Filename)
	}
}
