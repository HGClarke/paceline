package cmd

import "github.com/spf13/cobra"

var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:   "paceline",
	Short: "CLI for analyzing cycling ride data",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output as JSON")
}
