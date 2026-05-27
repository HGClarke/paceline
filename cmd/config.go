package cmd

import (
	"fmt"
	"os"

	"github.com/HGClarke/paceline/internal/config"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration",
	RunE:  runConfig,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Options(tablewriter.WithBorders(tw.Border{
		Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off,
	}))
	table.Header([]string{"Key", "Value"})
	table.Append([]string{"units", cfg.Units})
	table.Render()
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]
	switch key {
	case "units":
		if value != "metric" && value != "imperial" {
			return fmt.Errorf(`units must be "metric" or "imperial"`)
		}
		cfg.Units = value
	default:
		return fmt.Errorf("unknown config key %q; valid keys: units", key)
	}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Set %s = %s\n", key, value)
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	switch key {
	case "units":
		fmt.Fprintln(os.Stdout, cfg.Units)
	default:
		return fmt.Errorf("unknown config key %q; valid keys: units", key)
	}
	return nil
}
