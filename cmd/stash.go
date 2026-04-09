package cmd

import (
	"fmt"
	"strings"

	"github.com/harris-ahmad/gitpull/git"
	"github.com/harris-ahmad/gitpull/ui"
	"github.com/spf13/cobra"
)

var stashCmd = &cobra.Command{
	Use:   "stash",
	Short: "Stash or restore local changes across all repos",
}

var stashPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Stash local changes in all dirty repos",
	RunE: func(cmd *cobra.Command, args []string) error {
		message, _ := cmd.Flags().GetString("message")

		repos, err := repoList()
		if err != nil {
			return err
		}

		colWidth := 20
		for _, r := range repos {
			if len(r.Name) > colWidth {
				colWidth = len(r.Name)
			}
		}
		colWidth += 2

		var stashed, skipped int

		for _, repo := range repos {
			changes, err := git.LocalChanges(repo.Path)
			if err != nil || len(changes) == 0 {
				continue
			}

			if err := git.Stash(repo.Path, message); err != nil {
				fmt.Printf("%s  %-*s  failed to stash: %v\n", ui.Red("✗"), colWidth, repo.Name, err)
				skipped++
				continue
			}

			fmt.Printf("%s  %-*s  stashed (%s)\n", ui.Green("✓"), colWidth, repo.Name, strings.Join(changes, ", "))
			stashed++
		}

		sep := ui.Bold("────────────────────────────────────")
		fmt.Println("\n" + sep)
		fmt.Printf("  %d repos stashed\n", stashed)
		fmt.Printf("  %d errors\n", skipped)
		fmt.Println(sep)

		return nil
	},
}

var stashPopCmd = &cobra.Command{
	Use:   "pop",
	Short: "Restore stashed changes in all repos that have a stash",
	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := repoList()
		if err != nil {
			return err
		}

		colWidth := 20
		for _, r := range repos {
			if len(r.Name) > colWidth {
				colWidth = len(r.Name)
			}
		}
		colWidth += 2

		var popped, skipped int

		for _, repo := range repos {
			has, err := git.HasStash(repo.Path)
			if err != nil || !has {
				continue
			}

			if err := git.StashPop(repo.Path); err != nil {
				fmt.Printf("%s  %-*s  failed to pop: %v\n", ui.Red("✗"), colWidth, repo.Name, err)
				skipped++
				continue
			}

			fmt.Printf("%s  %-*s  restored\n", ui.Green("✓"), colWidth, repo.Name)
			popped++
		}

		sep := ui.Bold("────────────────────────────────────")
		fmt.Println("\n" + sep)
		fmt.Printf("  %d repos restored\n", popped)
		fmt.Printf("  %d errors\n", skipped)
		fmt.Println(sep)

		return nil
	},
}

func init() {
	stashPushCmd.Flags().StringP("message", "m", "", "stash message")
	stashCmd.AddCommand(stashPushCmd)
	stashCmd.AddCommand(stashPopCmd)
	rootCmd.AddCommand(stashCmd)
}
