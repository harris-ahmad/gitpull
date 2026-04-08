package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use: "sync",
	Short: "sync all repos in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		report, _ := cmd.Flags().GetBool("report")
		fmt.Printf("syncing (report-only: %v)\n", report)
		return nil
	},
}

func init() {
	syncCmd.Flags().Bool("report", false, "analyze only, pull nothing")
	rootCmd.AddCommand(syncCmd)
}