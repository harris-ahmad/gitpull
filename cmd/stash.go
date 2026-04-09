package cmd

import (
	"fmt"
	"os"
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

		dirs, err := os.ReadDir(".")
		if err != nil {
			return fmt.Errorf("failed to read current directory: %w", err)
		}

		colWidth := 20
		for _, dir := range dirs {
			if dir.IsDir() && len(dir.Name()) > colWidth {
				colWidth = len(dir.Name())
			}
		}
		colWidth += 2

		var stashed, skipped int

		for _, dir := range dirs {
			if !dir.IsDir() {
				continue
			}
			repoPath := dir.Name()

			changes, err := git.LocalChanges(repoPath)
			if err != nil {
				continue // not a git repo
			}
			if len(changes) == 0 {
				continue // nothing to stash
			}

			if err := git.Stash(repoPath, message); err != nil {
				fmt.Printf("%s  %-*s  failed to stash: %v\n", ui.Red("✗"), colWidth, repoPath, err)
				skipped++
				continue
			}

			fmt.Printf("%s  %-*s  stashed (%s)\n", ui.Green("✓"), colWidth, repoPath, strings.Join(changes, ", "))
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
		dirs, err := os.ReadDir(".")
		if err != nil {
			return fmt.Errorf("failed to read current directory: %w", err)
		}

		colWidth := 20
		for _, dir := range dirs {
			if dir.IsDir() && len(dir.Name()) > colWidth {
				colWidth = len(dir.Name())
			}
		}
		colWidth += 2

		var popped, skipped int

		for _, dir := range dirs {
			if !dir.IsDir() {
				continue
			}
			repoPath := dir.Name()

			has, err := git.HasStash(repoPath)
			if err != nil || !has {
				continue
			}

			if err := git.StashPop(repoPath); err != nil {
				fmt.Printf("%s  %-*s  failed to pop: %v\n", ui.Red("✗"), colWidth, repoPath, err)
				skipped++
				continue
			}

			fmt.Printf("%s  %-*s  restored\n", ui.Green("✓"), colWidth, repoPath)
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
