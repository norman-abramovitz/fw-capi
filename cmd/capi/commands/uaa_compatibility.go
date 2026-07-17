package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry-community/go-uaa"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// UAAVersionInfo contains UAA version and compatibility information.
type UAAVersionInfo struct {
	Version       string              `json:"version"       yaml:"version"`
	Endpoint      string              `json:"endpoint"      yaml:"endpoint"`
	ServerInfo    map[string]any      `json:"server_info"   yaml:"server_info"`
	Features      []string            `json:"features"      yaml:"features"`
	Compatibility CompatibilityStatus `json:"compatibility" yaml:"compatibility"`
	TestedAt      time.Time           `json:"tested_at"     yaml:"tested_at"`
}

// CompatibilityStatus represents the compatibility test results.
type CompatibilityStatus struct {
	Overall         string            `json:"overall"         yaml:"overall"`         // Compatible, Partial, Incompatible
	BasicAuth       string            `json:"basic_auth"      yaml:"basic_auth"`      // Token operations
	UserMgmt        string            `json:"user_mgmt"       yaml:"user_mgmt"`       // User CRUD operations
	GroupMgmt       string            `json:"group_mgmt"      yaml:"group_mgmt"`      // Group management
	ClientMgmt      string            `json:"client_mgmt"     yaml:"client_mgmt"`     // OAuth client management
	FeatureTests    map[string]string `json:"feature_tests"   yaml:"feature_tests"`   // Individual feature test results
	Issues          []string          `json:"issues"          yaml:"issues"`          // Known issues
	Recommendations []string          `json:"recommendations" yaml:"recommendations"` // Recommendations for this version
}

// CFIntegrationInfo contains Cloud Foundry integration details.
type CFIntegrationInfo struct {
	CFAPIVersion    string `json:"cf_api_version"   yaml:"cf_api_version"`
	UAAVersion      string `json:"uaa_version"      yaml:"uaa_version"`
	AuthMethod      string `json:"auth_method"      yaml:"auth_method"`
	ScopesSupported bool   `json:"scopes_supported" yaml:"scopes_supported"`
	TokenFormat     string `json:"token_format"     yaml:"token_format"`
	Compatible      bool   `json:"compatible"       yaml:"compatible"`
}

// createUsersCompatibilityCommand creates the compatibility testing command.
func createUsersCompatibilityCommand() *cobra.Command {
	var (
		testBasic, testAll bool
		saveResults        bool
	)

	cmd := &cobra.Command{
		Use:   "compatibility",
		Short: "Test UAA compatibility and features",
		Long: `Test compatibility with the current UAA endpoint and version.

This command performs a series of tests to verify that the UAA endpoint
supports the required features and operations for full compatibility
with the capi CLI UAA commands.`,
		Example: `  # Test basic compatibility
  capi uaa compatibility

  # Run comprehensive compatibility tests
  capi uaa compatibility --all

  # Save test results to file
  capi uaa compatibility --save-results`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := loadConfig()

			if GetEffectiveUAAEndpoint(config) == "" {
				return fmt.Errorf("%w. Use 'capi uaa target <url>' to set one", ErrNoUAAEndpoint)
			}

			// Create UAA client
			uaaClient, err := NewUAAClient(config)
			if err != nil {
				return fmt.Errorf("failed to create UAA client: %w", err)
			}

			// Run compatibility tests
			versionInfo := runCompatibilityTests(uaaClient, testAll)

			// Save results if requested
			if saveResults {
				err := saveCompatibilityResults(versionInfo)
				if err != nil {
					_, _ = fmt.Fprintf(os.Stdout, "Warning: Failed to save results: %v\n", err)
				} else {
					_, _ = os.Stdout.WriteString("Compatibility results saved to uaa-compatibility-results.json\n")
				}
			}

			// Display results
			return displayCompatibilityResults(versionInfo)
		},
	}

	cmd.Flags().BoolVar(&testBasic, "basic", false, "Run only basic compatibility tests")
	cmd.Flags().BoolVar(&testAll, "all", false, "Run comprehensive compatibility tests")
	cmd.Flags().BoolVar(&saveResults, "save-results", false, "Save test results to file")

	return cmd
}

// createUsersCFIntegrationCommand creates the CF integration testing command.
func createUsersCFIntegrationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cf-integration",
		Short: "Test Cloud Foundry integration",
		Long: `Test integration with Cloud Foundry API and authentication.

This command verifies that the UAA configuration is compatible with
Cloud Foundry and can properly authenticate CF API requests.`,
		Example: `  # Test CF integration
  capi uaa cf-integration

  # Show integration details in JSON
  capi uaa cf-integration --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := loadConfig()

			if GetEffectiveUAAEndpoint(config) == "" {
				return fmt.Errorf("%w. Use 'capi uaa target <url>' to set one", ErrNoUAAEndpoint)
			}

			// Test CF integration
			integrationInfo := testCFIntegration(config)

			// Display results
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				err := encoder.Encode(integrationInfo)
				if err != nil {
					return fmt.Errorf("encoding integration info to JSON: %w", err)
				}

				return nil
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				err := encoder.Encode(integrationInfo)
				if err != nil {
					return fmt.Errorf("encoding integration info to YAML: %w", err)
				}

				return nil
			default:
				return displayCFIntegrationResults(integrationInfo)
			}
		},
	}

	return cmd
}

// runCompatibilityTests performs comprehensive compatibility testing.
func runCompatibilityTests(client *UAAClientWrapper, comprehensive bool) *UAAVersionInfo {
	versionInfo := &UAAVersionInfo{
		Endpoint: client.Endpoint(),
		TestedAt: time.Now(),
		Features: []string{},
		Compatibility: CompatibilityStatus{
			FeatureTests:    make(map[string]string),
			Issues:          []string{},
			Recommendations: []string{},
		},
	}

	_, _ = os.Stdout.WriteString("Running UAA compatibility tests...\n")

	// Test server info and version detection
	_, _ = os.Stdout.WriteString("• Testing server info... ")

	ctx := context.Background()

	serverInfo, err := client.GetServerInfo(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "❌ Failed: %v\n", err)

		versionInfo.Compatibility.Overall = Incompatible
		versionInfo.Compatibility.Issues = append(versionInfo.Compatibility.Issues, "Cannot retrieve server information")

		return versionInfo
	}

	versionInfo.ServerInfo = serverInfo

	_, _ = os.Stdout.WriteString("✅ Success\n")

	// Extract version information
	if app, ok := serverInfo[uaaInfoKeyApp].(map[string]any); ok {
		if version, ok := app["version"].(string); ok {
			versionInfo.Version = version
		}
	}

	// Test basic authentication capabilities
	_, _ = os.Stdout.WriteString("• Testing authentication capabilities... ")

	authResult := testAuthenticationCapabilities(client)

	versionInfo.Compatibility.BasicAuth = authResult
	if authResult == Compatible {
		_, _ = os.Stdout.WriteString("✅ Compatible\n")

		versionInfo.Features = append(versionInfo.Features, "OAuth2 Authentication")
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "⚠️  %s\n", authResult)
	}

	// Test user management capabilities
	if client.IsAuthenticated() {
		runAuthenticatedTests(client, versionInfo, comprehensive)
	} else {
		versionInfo.Compatibility.Issues = append(versionInfo.Compatibility.Issues, "Authentication required for full testing")
	}

	// Determine overall compatibility
	versionInfo.Compatibility.Overall = determineOverallCompatibility(&versionInfo.Compatibility)

	// Add version-specific recommendations
	addVersionRecommendations(versionInfo)

	return versionInfo
}

// runAuthenticatedTests runs all tests that require authentication.
func runAuthenticatedTests(client *UAAClientWrapper, versionInfo *UAAVersionInfo, comprehensive bool) {
	// Test user management capabilities
	log.Print("• Testing user management... ")

	userMgmtResult := testUserManagementCapabilities(client)
	versionInfo.Compatibility.UserMgmt = userMgmtResult

	if userMgmtResult == Compatible {
		_, _ = os.Stdout.WriteString("✅ Compatible\n")

		versionInfo.Features = append(versionInfo.Features, "User Management")
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "⚠️  %s\n", userMgmtResult)
	}

	if comprehensive {
		runComprehensiveTests(client, versionInfo)
	}
}

// runComprehensiveTests runs additional comprehensive tests.
func runComprehensiveTests(client *UAAClientWrapper, versionInfo *UAAVersionInfo) {
	// Test group management capabilities
	log.Print("• Testing group management... ")

	groupMgmtResult := testGroupManagementCapabilities(client)
	versionInfo.Compatibility.GroupMgmt = groupMgmtResult

	if groupMgmtResult == Compatible {
		_, _ = os.Stdout.WriteString("✅ Compatible\n")

		versionInfo.Features = append(versionInfo.Features, "Group Management")
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "⚠️  %s\n", groupMgmtResult)
	}

	// Test client management capabilities
	log.Print("• Testing client management... ")

	clientMgmtResult := testClientManagementCapabilities(client)
	versionInfo.Compatibility.ClientMgmt = clientMgmtResult

	if clientMgmtResult == Compatible {
		_, _ = os.Stdout.WriteString("✅ Compatible\n")

		versionInfo.Features = append(versionInfo.Features, "OAuth Client Management")
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "⚠️  %s\n", clientMgmtResult)
	}
}

// testAuthenticationCapabilities tests OAuth2 authentication features.
func testAuthenticationCapabilities(client *UAAClientWrapper) string {
	// Test token key retrieval (public endpoint)
	_, err := client.Client().TokenKey()
	if err != nil {
		return Incompatible
	}

	// Test token keys retrieval
	_, err = client.Client().TokenKeys()
	if err != nil {
		return Partial
	}

	return Compatible
}

// testUserManagementCapabilities tests user CRUD operations.
func testUserManagementCapabilities(client *UAAClientWrapper) string {
	// Test user listing (requires authentication)
	_, _, err := client.Client().ListUsers("", "", "", uaa.SortAscending, 1, 1)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") || strings.Contains(err.Error(), "forbidden") {
			return Partial
		}

		return Incompatible
	}

	return Compatible
}

// testGroupManagementCapabilities tests group management operations.
func testGroupManagementCapabilities(client *UAAClientWrapper) string {
	// Test group listing
	_, _, err := client.Client().ListGroups("", "", "", uaa.SortAscending, 1, 1)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") || strings.Contains(err.Error(), "forbidden") {
			return Partial
		}

		return Incompatible
	}

	return Compatible
}

// testClientManagementCapabilities tests OAuth client management.
func testClientManagementCapabilities(client *UAAClientWrapper) string {
	// Test client listing
	_, _, err := client.Client().ListClients("", "", uaa.SortAscending, 1, 1)
	if err != nil {
		if strings.Contains(err.Error(), "unauthorized") || strings.Contains(err.Error(), "forbidden") {
			return Partial
		}

		return Incompatible
	}

	return Compatible
}

// determineOverallCompatibility calculates overall compatibility status.
func determineOverallCompatibility(status *CompatibilityStatus) string {
	compatibleCount := 0
	totalCount := 0

	tests := []string{status.BasicAuth, status.UserMgmt, status.GroupMgmt, status.ClientMgmt}
	for _, test := range tests {
		if test != "" {
			totalCount++

			if test == Compatible {
				compatibleCount++
			}
		}
	}

	if totalCount == 0 {
		return Unknown
	}

	ratio := float64(compatibleCount) / float64(totalCount)
	switch {
	case ratio >= constants.CompatibilityThresholdHigh:
		return Compatible
	case ratio >= constants.CompatibilityThresholdMedium:
		return Partial
	default:
		return Incompatible
	}
}

// addVersionRecommendations adds version-specific recommendations.
func addVersionRecommendations(info *UAAVersionInfo) {
	version := info.Version

	// Extract major.minor version
	re := regexp.MustCompile(`(\d+)\.(\d+)`)

	matches := re.FindStringSubmatch(version)
	if len(matches) >= constants.MinimumVersionMatches {
		major, _ := strconv.Atoi(matches[1])
		minor, _ := strconv.Atoi(matches[2])

		if major < constants.MinimumVersionCompatibility {
			info.Compatibility.Issues = append(info.Compatibility.Issues, "UAA version is very old and may have limited functionality")
			info.Compatibility.Recommendations = append(info.Compatibility.Recommendations, "Consider upgrading to UAA 4.x or later")
		} else if major == 4 && minor < 30 {
			info.Compatibility.Recommendations = append(info.Compatibility.Recommendations, "Consider upgrading to UAA 4.30+ for best compatibility")
		}
	}

	// Add general recommendations based on compatibility
	switch info.Compatibility.Overall {
	case Incompatible:
		info.Compatibility.Recommendations = append(info.Compatibility.Recommendations,
			"UAA endpoint may not be compatible with this CLI version",
			"Verify UAA endpoint URL and network connectivity",
			"Check UAA logs for detailed error information")
	case Partial:
		info.Compatibility.Recommendations = append(info.Compatibility.Recommendations,
			"Some features may not work due to insufficient permissions",
			"Ensure your client has appropriate authorities (scim.read, scim.write, etc.)",
			"Contact your UAA administrator for permission adjustments")
	case Compatible:
		info.Compatibility.Recommendations = append(info.Compatibility.Recommendations,
			"UAA endpoint is fully compatible with this CLI",
			"All features should work as expected")
	}
}

// testCFIntegration tests Cloud Foundry integration.
func testCFIntegration(config *Config) *CFIntegrationInfo {
	integration := &CFIntegrationInfo{
		UAAVersion: Unknown,
		Compatible: false,
	}

	// Check if we have CF API endpoint configured
	if config.API != "" {
		integration.CFAPIVersion = "configured"

		// Try to infer UAA endpoint from CF API
		if GetEffectiveUAAEndpoint(config) == "" {
			// Try to infer UAA endpoint
			inferredUAA := inferUAAFromCFAPI(config.API)
			if inferredUAA != "" {
				integration.AuthMethod = "inferred"
			}
		} else {
			integration.AuthMethod = "explicit"
		}
	}

	// Test UAA compatibility
	testUAACompatibility(config, integration)

	return integration
}

// testUAACompatibility tests UAA compatibility and extracts version info.
func testUAACompatibility(config *Config, integration *CFIntegrationInfo) {
	if config.UAAEndpoint == "" {
		return
	}

	uaaClient, err := NewUAAClient(config)
	if err != nil {
		return
	}

	ctx := context.Background()
	extractUAAVersionInfo(uaaClient, ctx, integration)
	checkUAAAuthentication(uaaClient, integration)
}

// extractUAAVersionInfo extracts UAA version information from server info.
func extractUAAVersionInfo(uaaClient *UAAClientWrapper, ctx context.Context, integration *CFIntegrationInfo) {
	serverInfo, err := uaaClient.GetServerInfo(ctx)
	if err != nil {
		return
	}

	app, ok := serverInfo[uaaInfoKeyApp].(map[string]any)
	if !ok {
		return
	}

	if version, ok := app["version"].(string); ok {
		integration.UAAVersion = version
	}
}

// checkUAAAuthentication checks if UAA authentication is working.
func checkUAAAuthentication(uaaClient *UAAClientWrapper, integration *CFIntegrationInfo) {
	if uaaClient.IsAuthenticated() {
		integration.ScopesSupported = true
		integration.TokenFormat = "JWT"
		integration.Compatible = true
	}
}

// inferUAAFromCFAPI attempts to infer UAA endpoint from CF API endpoint.
func inferUAAFromCFAPI(cfEndpoint string) string {
	// Simple heuristic: replace "api." with "uaa." for common CF deployments
	if strings.Contains(cfEndpoint, "api.") {
		return strings.Replace(cfEndpoint, "api.", "uaa.", 1)
	}

	return ""
}

// saveCompatibilityResults saves test results to file.
func saveCompatibilityResults(info *UAAVersionInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal compatibility results to JSON: %w", err)
	}

	err = os.WriteFile("uaa-compatibility-results.json", data, constants.FilePermissionReadWrite)
	if err != nil {
		return fmt.Errorf("failed to write compatibility results file: %w", err)
	}

	return nil
}

// displayCompatibilityResults displays compatibility test results.
func displayCompatibilityResults(info *UAAVersionInfo) error {
	output := viper.GetString("output")

	switch output {
	case OutputFormatJSON:
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(info)
		if err != nil {
			return fmt.Errorf("encoding compatibility info to JSON: %w", err)
		}

		return nil
	case OutputFormatYAML:
		encoder := yaml.NewEncoder(os.Stdout)

		err := encoder.Encode(info)
		if err != nil {
			return fmt.Errorf("encoding compatibility info to YAML: %w", err)
		}

		return nil
	default:
		return displayCompatibilityTable(info)
	}
}

// displayCompatibilityTable displays results in table format.
func displayCompatibilityTable(info *UAAVersionInfo) error {
	// Basic info table
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	overallStatus := formatStatus(info.Compatibility.Overall)
	_ = table.Append("Endpoint", info.Endpoint)
	_ = table.Append("Version", info.Version)
	_ = table.Append("Tested", info.TestedAt.Format(time.RFC3339))
	_ = table.Append("Overall Status", overallStatus)

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	// Feature compatibility table
	err = renderFeatureCompatibilityTable(info)
	if err != nil {
		return err
	}

	// Display supported features
	if len(info.Features) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "\nSupported Features:\n")

		for _, feature := range info.Features {
			_, _ = fmt.Fprintf(os.Stdout, "  ✅ %s\n", feature)
		}
	}

	// Display issues
	if len(info.Compatibility.Issues) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "\nIssues Found:\n")

		for _, issue := range info.Compatibility.Issues {
			_, _ = fmt.Fprintf(os.Stdout, "  ⚠️  %s\n", issue)
		}
	}

	// Display recommendations
	if len(info.Compatibility.Recommendations) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "\nRecommendations:\n")

		for _, rec := range info.Compatibility.Recommendations {
			_, _ = fmt.Fprintf(os.Stdout, "  💡 %s\n", rec)
		}
	}

	return nil
}

// displayCFIntegrationResults displays CF integration test results.
func displayCFIntegrationResults(info *CFIntegrationInfo) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	compatibleStatus := "❌ No"
	if info.Compatible {
		compatibleStatus = "✅ Yes"
	}

	scopeStatus := "❌ No"
	if info.ScopesSupported {
		scopeStatus = "✅ Yes"
	}

	_ = table.Append("CF API Version", info.CFAPIVersion)
	_ = table.Append("UAA Version", info.UAAVersion)
	_ = table.Append("Auth Method", info.AuthMethod)
	_ = table.Append("Scopes Supported", scopeStatus)
	_ = table.Append("Token Format", info.TokenFormat)
	_ = table.Append("Compatible", compatibleStatus)

	err := table.Render()
	if err != nil {
		return fmt.Errorf("failed to render table: %w", err)
	}

	if !info.Compatible {
		_, _ = os.Stdout.WriteString("\nTroubleshooting:\n")
		_, _ = os.Stdout.WriteString("  • Verify CF API endpoint is correct\n")
		_, _ = os.Stdout.WriteString("  • Ensure UAA endpoint is accessible\n")
		_, _ = os.Stdout.WriteString("  • Check authentication credentials\n")
	}

	return nil
}

// hasFeatureCompatibilityData checks if there's any feature compatibility data to display.
func hasFeatureCompatibilityData(compatibility *CompatibilityStatus) bool {
	return compatibility.BasicAuth != "" || compatibility.UserMgmt != "" ||
		compatibility.GroupMgmt != "" || compatibility.ClientMgmt != ""
}

// renderFeatureCompatibilityTable renders the feature compatibility table.
func renderFeatureCompatibilityTable(info *UAAVersionInfo) error {
	if !hasFeatureCompatibilityData(&info.Compatibility) {
		return nil
	}

	_, _ = os.Stdout.WriteString("\n")

	featureTable := tablewriter.NewWriter(os.Stdout)
	featureTable.Header("Feature", "Status")

	addFeatureRowIfExists(featureTable, "Authentication", info.Compatibility.BasicAuth)
	addFeatureRowIfExists(featureTable, "User Management", info.Compatibility.UserMgmt)
	addFeatureRowIfExists(featureTable, "Group Management", info.Compatibility.GroupMgmt)
	addFeatureRowIfExists(featureTable, "Client Management", info.Compatibility.ClientMgmt)

	err := featureTable.Render()
	if err != nil {
		return fmt.Errorf("failed to render feature table: %w", err)
	}

	return nil
}

// addFeatureRowIfExists adds a feature row to the table if the status is not empty.
func addFeatureRowIfExists(table *tablewriter.Table, featureName, status string) {
	if status != "" {
		_ = table.Append(featureName, formatStatus(status))
	}
}

// formatStatus formats compatibility status with appropriate icons.
func formatStatus(status string) string {
	switch status {
	case Compatible:
		return "✅ Compatible"
	case Partial:
		return "⚠️  Partial"
	case Incompatible:
		return "❌ Incompatible"
	default:
		return "❓ Unknown"
	}
}
