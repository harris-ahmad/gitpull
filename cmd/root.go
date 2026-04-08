package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "gitpull",
	Short: "Clone and sync GitHub repos with conflict prediction",
}

func init() {
	rootCmd.PersistentFlags().String("token", "", "GitHub API token")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}