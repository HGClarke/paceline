package cmd

import (
	"fmt"

	"github.com/hollandclarke/paceline/internal/config"
	"github.com/spf13/cobra"
)

var jsonOutput bool
var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "paceline",
	Short: "CLI for analyzing cycling ride data",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return loadCfg()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output as JSON")
}

// loadCfg loads the user config file into the package-level cfg variable.
// It is called from rootCmd.PersistentPreRunE and from any subcommand that
// defines its own PersistentPreRunE (which would otherwise shadow the root one).
func loadCfg() error {
	c, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	cfg = c
	return nil
}
