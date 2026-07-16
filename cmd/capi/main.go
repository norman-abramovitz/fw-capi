package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fivetwenty-io/capi/v3/cmd/capi/commands"
	"github.com/fivetwenty-io/capi/v3/internal/constants"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// BuildInfo holds build-time information set via ldflags.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// buildInfo encapsulates build-time variables set via ldflags.
type buildInfo struct {
	version string // Set by ldflags during build
	commit  string // Set by ldflags during build
	date    string // Set by ldflags during build
}

// getBuildInfo returns the build information from ldflags.
func getBuildInfo() BuildInfo {
	info := buildInfo{
		version: getBuildVersion(),
		commit:  getBuildCommit(),
		date:    getBuildDate(),
	}

	return BuildInfo{
		Version: info.version,
		Commit:  info.commit,
		Date:    info.date,
	}
}

// Build-time variables set via ldflags - these must be var for ldflags to work.
//
//nolint:gochecknoglobals // These variables must be global for ldflags to work
var (
	version = "dev"     // Set by ldflags during build
	commit  = "none"    // Set by ldflags during build
	date    = "unknown" // Set by ldflags during build
)

// getBuildVersion returns the build version.
func getBuildVersion() string {
	return version
}

// getBuildCommit returns the build commit.
func getBuildCommit() string {
	return commit
}

// getBuildDate returns the build date.
func getBuildDate() string {
	return date
}

// newRootCommand creates and configures the root command.
func newRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "capi",
		Short: "Cloud Foundry API v3 CLI",
		Long: `A command-line interface for interacting with Cloud Foundry API v3.

This CLI provides comprehensive access to Cloud Foundry resources including
applications, spaces, organizations, services, and more.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	setupGlobalFlags(rootCmd)
	bindFlagsToViper(rootCmd)
	addAllCommands(rootCmd)

	return rootCmd
}

func setupGlobalFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("config", "c", "", "config file (default is $HOME/.capi/config.yml)")
	cmd.PersistentFlags().StringP("api", "a", "", "API endpoint URL")
	cmd.PersistentFlags().StringP("token", "t", "", "authentication token")
	cmd.PersistentFlags().String("output", "table", "output format (table, json, yaml)")
	cmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	cmd.PersistentFlags().Bool("no-color", false, "disable colored output")
	cmd.PersistentFlags().Bool("skip-ssl-validation", false, "skip SSL certificate validation")
}

func bindFlagsToViper(cmd *cobra.Command) {
	_ = viper.BindPFlag("api", cmd.PersistentFlags().Lookup("api"))
	_ = viper.BindPFlag("token", cmd.PersistentFlags().Lookup("token"))
	_ = viper.BindPFlag("output", cmd.PersistentFlags().Lookup("output"))
	_ = viper.BindPFlag("verbose", cmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("no-color", cmd.PersistentFlags().Lookup("no-color"))
	_ = viper.BindPFlag("skip-ssl-validation", cmd.PersistentFlags().Lookup("skip-ssl-validation"))
}

func addAllCommands(cmd *cobra.Command) {
	addCoreCommands(cmd)
	addResourceCommands(cmd)
	addAPIv3Commands(cmd)
}

func addCoreCommands(cmd *cobra.Command) {
	buildInfo := getBuildInfo()
	cmd.AddCommand(commands.NewVersionCommand(buildInfo.Version, buildInfo.Commit, buildInfo.Date))
	cmd.AddCommand(commands.NewLoginCommand())
	cmd.AddCommand(commands.NewLogoutCommand())
	cmd.AddCommand(commands.NewTokenCommand())
	cmd.AddCommand(commands.NewConfigCommand())
	cmd.AddCommand(commands.NewAPIsCommand())
	cmd.AddCommand(commands.NewTargetCommand())
	cmd.AddCommand(commands.NewInfoCommand())
}

func addResourceCommands(cmd *cobra.Command) {
	cmd.AddCommand(commands.NewOrgsCommand())
	cmd.AddCommand(commands.NewSpacesCommand())
	cmd.AddCommand(commands.NewAppsCommand())
	cmd.AddCommand(commands.NewServicesCommand())
	cmd.AddCommand(commands.NewDomainsCommand())
	cmd.AddCommand(commands.NewRoutesCommand())
	cmd.AddCommand(commands.NewRoutePoliciesCommand())
	cmd.AddCommand(commands.NewSecurityGroupsCommand())
	cmd.AddCommand(commands.NewBuildpacksCommand())
	cmd.AddCommand(commands.NewStacksCommand())
	cmd.AddCommand(commands.NewUAACommand())
	cmd.AddCommand(commands.NewRolesCommand())
	cmd.AddCommand(commands.NewJobsCommand())
}

func addAPIv3Commands(cmd *cobra.Command) {
	cmd.AddCommand(commands.NewOrgQuotasCommand())
	cmd.AddCommand(commands.NewSpaceQuotasCommand())
	cmd.AddCommand(commands.NewSidecarsCommand())
	cmd.AddCommand(commands.NewRevisionsCommand())
	cmd.AddCommand(commands.NewEnvVarGroupsCommand())
	cmd.AddCommand(commands.NewAppUsageEventsCommand())
	cmd.AddCommand(commands.NewServiceUsageEventsCommand())
	cmd.AddCommand(commands.NewAuditEventsCommand())
	cmd.AddCommand(commands.NewResourceMatchesCommand())
	cmd.AddCommand(commands.NewIsolationSegmentsCommand())
	cmd.AddCommand(commands.NewFeatureFlagsCommand())
	cmd.AddCommand(commands.NewManifestsCommand())
	cmd.AddCommand(commands.NewAdminCommand())
}

func initConfig() {
	cfgFile := viper.GetString("config")

	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// Create config directory if it doesn't exist
		configDir := filepath.Join(home, ".capi")

		err = os.MkdirAll(configDir, constants.ConfigDirPerm)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
		}

		// Search config in ~/.capi/config.yml
		viper.AddConfigPath(configDir)
		viper.SetConfigType("yml")
		viper.SetConfigName("config")
	}

	// Read in environment variables that match
	viper.SetEnvPrefix("CAPI")
	viper.AutomaticEnv()

	// If a config file is found, read it in
	err := viper.ReadInConfig()
	if err == nil {
		if viper.GetBool("verbose") {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}

func main() {
	cobra.OnInitialize(initConfig)

	rootCmd := newRootCommand()

	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
