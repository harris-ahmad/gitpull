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

var cfg = mustLoadConfig()

func mustLoadConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load config: %v\n", err)
		return &config.Config{Parallel: 4}
	}
	return cfg
}

func init() {
	rootCmd.PersistentFlags().String("token", cfg.Token, "GitHub API token")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}