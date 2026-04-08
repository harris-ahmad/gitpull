package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use: "clone <username>",
	Short: "clone all public repos from a GitHub user",
	Args: cobra.ExactArgs(1), //to validate exactly one argument is needed
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		parallel, _ := cmd.Flags().GetInt("parallel")
		fmt.Printf("cloning repos for %s with parallelism %d\n", username, parallel)
		return nil
	},
}

func init() {
	cloneCmd.Flags().IntP("parallel", "p", 10, "number of concurrent clones")
	rootCmd.AddCommand(cloneCmd)
}