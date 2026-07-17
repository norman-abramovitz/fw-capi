package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// NewVersionCommand creates the version command.
func NewVersionCommand(version, commit, date string) *cobra.Command {
	return &cobra.Command{
		Use:   Version,
		Short: "Display version information",
		Long:  "Display detailed version information about the CAPI CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			type VersionInfo struct {
				Version string `json:"version" yaml:"version"`
				Commit  string `json:"commit"  yaml:"commit"`
				Built   string `json:"built"   yaml:"built"`
			}

			versionInfo := VersionInfo{
				Version: version,
				Commit:  commit,
				Built:   date,
			}

			output := viper.GetString("output")
			switch output {
			case OutputFormatJSON:
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")

				return encoder.Encode(versionInfo)
			case OutputFormatYAML:
				encoder := yaml.NewEncoder(os.Stdout)

				return encoder.Encode(versionInfo)
			default:
				table := tablewriter.NewWriter(os.Stdout)
				table.Header("Property", "Value")
				_ = table.Append("Version", version)
				_ = table.Append("Commit", commit)
				_ = table.Append("Built", date)

				err := table.Render()
				if err != nil {
					return fmt.Errorf("failed to render table: %w", err)
				}
			}

			return nil
		},
	}
}
