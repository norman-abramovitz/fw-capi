package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// validateFilePath validates that a file path is safe to read.
func validateFilePathUsers(filePath string) error {
	// Clean the path to resolve any path traversal attempts
	cleanPath := filepath.Clean(filePath)

	// Check for path traversal attempts
	if filepath.IsAbs(filePath) {
		// Allow absolute paths but ensure they're clean
		if cleanPath != filePath {
			return ErrDirectoryTraversalDetected
		}
	} else {
		// For relative paths, ensure they don't escape the current directory
		if len(cleanPath) > 0 && cleanPath[0] == '.' && len(cleanPath) > 1 && cleanPath[1] == '.' {
			return ErrDirectoryTraversalDetected
		}
	}

	// Check if file exists and is readable
	_, err := os.Stat(cleanPath)
	if err != nil {
		return fmt.Errorf("file not accessible: %w", err)
	}

	return nil
}

// readBatchInputData reads input data from file or stdin.
func readBatchInputData(inputFile string) ([]byte, error) {
	var (
		inputData []byte
		err       error
	)

	if inputFile == "" || inputFile == "-" {
		// Read from stdin
		inputData, err = io.ReadAll(os.Stdin)
	} else {
		// Validate file path before reading
		err = validateFilePathUsers(inputFile)
		if err != nil {
			return nil, fmt.Errorf("invalid input file: %w", err)
		}
		// Read from file
		inputData, err = os.ReadFile(filepath.Clean(inputFile))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	return inputData, nil
}

// validateBatchDryRun validates JSON input without creating users.
func validateBatchDryRun(inputData []byte) error {
	var users []any

	err := json.Unmarshal(inputData, &users)
	if err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Validation successful: %d users ready for import\n", len(users))

	return nil
}

// displayBatchResults displays the results of batch import.
func displayBatchResults(results []BatchResult) {
	successful := 0
	failed := 0

	for _, result := range results {
		if result.Error != nil {
			failed++

			_, _ = fmt.Fprintf(os.Stdout, "❌ Failed to create user: %v\n", result.Error)
		} else {
			successful++
		}
	}

	_, _ = os.Stdout.WriteString("\nBatch import completed:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  ✅ Successful: %d\n", successful)
	_, _ = fmt.Fprintf(os.Stdout, "  ❌ Failed: %d\n", failed)
	_, _ = fmt.Fprintf(os.Stdout, "  📊 Total: %d\n", len(results))
}

// createUsersBatchImportCommand creates the batch import command.
func createUsersBatchImportCommand() *cobra.Command {
	var (
		inputFile string
		parallel  bool
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:   "batch-import",
		Short: "Import users in batch from JSON file",
		Long: `Import multiple users from a JSON file in batch mode.

The JSON file should contain an array of user objects with the same structure
as individual user creation. This command supports both sequential and parallel
processing for improved performance.`,
		Example: `  # Import users from file (sequential)
  capi uaa batch-import --file users.json

  # Import users in parallel for better performance
  capi uaa batch-import --file users.json --parallel

  # Dry run to validate without creating users
  capi uaa batch-import --file users.json --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := loadConfig()

			if GetEffectiveUAAEndpoint(config) == "" {
				return fmt.Errorf("%w. Use 'capi uaa target <url>' to set one", ErrNoUAAEndpoint)
			}

			// Read input data
			inputData, err := readBatchInputData(inputFile)
			if err != nil {
				return err
			}

			if dryRun {
				return validateBatchDryRun(inputData)
			}

			// Create UAA client
			uaaClient, err := NewUAAClient(config)
			if err != nil {
				return fmt.Errorf("failed to create UAA client: %w", err)
			}

			if !uaaClient.IsAuthenticated() {
				return fmt.Errorf("%w. Use a token command to authenticate first", ErrNotAuthenticated)
			}

			// Perform batch import
			results, err := BulkUserImport(uaaClient, inputData, parallel)
			if err != nil {
				return fmt.Errorf("failed to import users: %w", err)
			}

			// Display results
			displayBatchResults(results)

			return nil
		},
	}

	cmd.Flags().StringVar(&inputFile, "file", "", "Input JSON file (use '-' for stdin)")
	cmd.Flags().BoolVar(&parallel, "parallel", false, "Process imports in parallel")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate input without creating users")

	return cmd
}

// createUsersPerformanceCommand creates the performance metrics command.
func createUsersPerformanceCommand() *cobra.Command {
	var reset bool

	cmd := &cobra.Command{
		Use:   "performance",
		Short: "Display performance metrics",
		Long: `Display performance metrics for UAA operations including cache hit rates,
operation timings, and efficiency statistics.`,
		Example: `  # Show current performance metrics
  capi uaa performance

  # Show metrics in JSON format
  capi uaa performance --output json

  # Reset performance counters
  capi uaa performance --reset`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if reset {
				// Reset performance metrics
				GetDefaultPerformanceService().metrics = &PerformanceMetrics{
					operations: make(map[string][]time.Duration),
				}
				GetDefaultPerformanceService().cache.Clear()

				_, _ = os.Stdout.WriteString("Performance metrics and cache cleared\n")

				return nil
			}

			// Get current metrics
			metrics := GetDefaultPerformanceService().metrics.GetMetrics()

			// Display metrics based on output format
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				return encoder.Encode(metrics)
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				return encoder.Encode(metrics)
			default:
				return displayPerformanceMetrics(metrics)
			}
		},
	}

	cmd.Flags().BoolVar(&reset, "reset", false, "Reset performance counters and cache")

	return cmd
}

// createUsersCacheCommand creates the cache management command.
func createUsersCacheCommand() *cobra.Command {
	var (
		clearCache bool
		stats      bool
	)

	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage UAA operation cache",
		Long: `Manage the UAA operation cache for improved performance.

The cache stores frequently accessed data like user lookups, group information,
and server info to reduce API calls and improve response times.`,
		Example: `  # Show cache statistics
  capi uaa cache --stats

  # Show cache statistics in JSON format
  capi uaa cache --stats --output json

  # Clear all cached data
  capi uaa cache --clear`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if clearCache {
				GetDefaultPerformanceService().cache.Clear()

				_, _ = os.Stdout.WriteString("Cache cleared successfully\n")

				return nil
			}

			if stats {
				metrics := GetDefaultPerformanceService().metrics.GetMetrics()

				output := viper.GetString("output")
				switch output {
				case OutputFormatJSON:
					encoder := json.NewEncoder(os.Stdout)
					encoder.SetIndent("", "  ")

					return encoder.Encode(metrics)
				case OutputFormatYAML:
					encoder := yaml.NewEncoder(os.Stdout)

					return encoder.Encode(metrics)
				default:
					return displayCacheStatistics(metrics)
				}
			}

			// Default: show cache status
			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				cacheInfo := map[string]any{
					keyStatus:      Active,
					"ttl":          "10 minutes",
					"auto_cleanup": true,
				}
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				return encoder.Encode(cacheInfo)
			case OutputFormatYAML:
				cacheInfo := map[string]any{
					keyStatus:      Active,
					"ttl":          "10 minutes",
					"auto_cleanup": true,
				}
				encoder := yaml.NewEncoder(os.Stdout)

				return encoder.Encode(cacheInfo)
			default:
				return displayCacheStatus()
			}
		},
	}

	cmd.Flags().BoolVar(&clearCache, "clear", false, "Clear all cached data")
	cmd.Flags().BoolVar(&stats, "stats", false, "Show cache statistics")

	return cmd
}

// Helper function to display performance metrics in table format.
func displayPerformanceMetrics(metrics map[string]any) error {
	_, _ = os.Stdout.WriteString("UAA Performance Metrics:\n")
	_, _ = os.Stdout.WriteString("\n")

	// Create summary table for overall metrics
	summaryTable := tablewriter.NewWriter(os.Stdout)
	summaryTable.Header("Metric", "Value")

	_ = summaryTable.Append("Total Operations", fmt.Sprintf("%v", metrics["total_operations"]))
	_ = summaryTable.Append("Cache Hits", fmt.Sprintf("%v", metrics["cache_hits"]))
	_ = summaryTable.Append("Cache Misses", fmt.Sprintf("%v", metrics["cache_misses"]))

	if cacheHitRate, ok := metrics["cache_hit_rate"].(float64); ok {
		_ = summaryTable.Append("Cache Hit Rate", fmt.Sprintf("%.2f%%", cacheHitRate))
	} else {
		_ = summaryTable.Append("Cache Hit Rate", NotAvailable)
	}

	_ = summaryTable.Render()

	// Create operations statistics table if we have operations data
	if operations, ok := metrics["operations"].(map[string]any); ok && len(operations) > 0 {
		_, _ = os.Stdout.WriteString("\nOperation Statistics:\n")
		_, _ = os.Stdout.WriteString("\n")

		operationsTable := tablewriter.NewWriter(os.Stdout)
		operationsTable.Header("Operation", "Count", "Average", "Min", "Max")

		for operation, stats := range operations {
			if opStats, ok := stats.(map[string]any); ok {
				count := fmt.Sprintf("%v", opStats["count"])
				average := fmt.Sprintf("%v", opStats["average"])
				minVal := fmt.Sprintf("%v", opStats["min"])
				maxVal := fmt.Sprintf("%v", opStats["max"])

				_ = operationsTable.Append(operation, count, average, minVal, maxVal)
			}
		}

		_ = operationsTable.Render()
	}

	return nil
}

// displayCacheStatus displays the cache status in table format.
func displayCacheStatus() error {
	_, _ = os.Stdout.WriteString("UAA Cache Status:\n")
	_, _ = os.Stdout.WriteString("\n")

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Property", "Value")

	_ = table.Append("Status", "Active")
	_ = table.Append("TTL", "10 minutes")
	_ = table.Append("Auto-cleanup", "Enabled")

	_ = table.Render()

	_, _ = os.Stdout.WriteString("\nUse --stats to see detailed statistics\n")
	_, _ = os.Stdout.WriteString("Use --clear to clear cached data\n")

	return nil
}

// displayCacheStatistics displays cache statistics in table format.
func displayCacheStatistics(metrics map[string]any) error {
	_, _ = os.Stdout.WriteString("Cache Statistics:\n")
	_, _ = os.Stdout.WriteString("\n")

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Metric", "Value")

	_ = table.Append("Cache Hits", fmt.Sprintf("%v", metrics["cache_hits"]))
	_ = table.Append("Cache Misses", fmt.Sprintf("%v", metrics["cache_misses"]))

	if cacheHitRate, ok := metrics["cache_hit_rate"].(float64); ok {
		_ = table.Append("Hit Rate", fmt.Sprintf("%.2f%%", cacheHitRate))
	} else {
		_ = table.Append("Hit Rate", NotAvailable)
	}

	_ = table.Append("Total Operations", fmt.Sprintf("%v", metrics["total_operations"]))

	_ = table.Render()

	return nil
}
