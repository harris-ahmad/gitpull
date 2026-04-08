package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/harris-ahmad/gitpull/config"
)

var rootCmd = &cobra.Command{
	Use: "gitpull",
	Short: "Clone and sync GitHub repos with conflict prediction",
}

var cfg *config.Config

func init() {
	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load config: %v\n", err)
		cfg = &config.Config{Parallel: 4}
	}
	rootCmd.PersistentFlags().String("token", cfg.Token, "GitHub API token")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}