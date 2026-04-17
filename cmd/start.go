package cmd

import (
	"github.com/spf13/cobra"
	"strings"
	"fmt"
	"github.com/harris-ahmad/gitpull/ui"
	"github.com/harris-ahmad/gitpull/git"
)

var startCmd = &cobra.Command{
	Use: "start <feature-name>",
	Short: "start working on a new feature branch",
	Long: `Ensures you're on the default branch and up to date, then creates a new feature branch.

  Example:
    gitpull start login
    gitpull start user-authentication
    gitpull start fix-bug-123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		featureName := args[0]

		branchName := "feature/" + strings.ToLower(strings.ReplaceAll(featureName, " ", "-"))

		repos, err := repoList()
		if err != nil {
			return err
		}

		if len(repos) > 1 {
			return fmt.Errorf("start command only works inside a single repo, not across multiple repos")
		}

		repo := repos[0]
		fmt.Printf("%s Ensuring you're on the default branch and up to date...\n", ui.Cyan("→"))
		defaultBranch, err := git.EnsureMainAndUpToDate(repo.Path)
		if err != nil {
			return err 
		}
		fmt.Printf("%s On %s and up to date\n", ui.Green("✓"), ui.Bold(defaultBranch))
		fmt.Printf("%s Creating new feature branch %s...\n", ui.Cyan("→"), ui.Bold(branchName))
		if err := git.CreateAndSwitchBranch(repo.Path, branchName); err != nil {
			return err
		}

		fmt.Printf("%s Ready to work on %s\n", ui.Green("✓"), ui.Bold(branchName))
        fmt.Printf("\n%s\n", ui.Bold("When you're done, run: gitpull done"))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}