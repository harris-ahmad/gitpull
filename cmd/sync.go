package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/harris-ahmad/gitpull/git"
	"github.com/harris-ahmad/gitpull/ui"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "sync all repos in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		report, _ := cmd.Flags().GetBool("report")

		dirs, err := os.ReadDir(".")
		if err != nil {
			return fmt.Errorf("failed to read current directory: %w", err)
		}

		var total, pulled, upToDate, skippedLocal, skippedConflict, skippedOther int

		// compute column width from longest repo name
		colWidth := 20
		for _, dir := range dirs {
			if dir.IsDir() && len(dir.Name()) > colWidth {
				colWidth = len(dir.Name())
			}
		}
		colWidth += 2

		for _, dir := range dirs {
			if !dir.IsDir() {
				continue
			}

			total++
			repoPath := dir.Name()

			branch, err := git.DefaultBranch(repoPath)
			if err != nil {
				fmt.Printf("%s  %-*s  not a git repo, skipped\n", ui.Bold("-"), colWidth, repoPath)
				skippedOther++
				continue
			}

			if err := git.Fetch(repoPath); err != nil {
				fmt.Printf("%s  %-*s  failed to fetch: %v\n", ui.Bold("-"), colWidth, repoPath, err)
				skippedOther++
				continue
			}

			changes, err := git.LocalChanges(repoPath)
			if err != nil {
				fmt.Printf("%s  %-*s  failed to check local changes: %v\n", ui.Bold("-"), colWidth, repoPath, err)
				skippedOther++
				continue
			}
			if len(changes) > 0 {
				fmt.Printf("%s  %-*s  local changes — skipped (%s)\n", ui.Yellow("⚠"), colWidth, repoPath, strings.Join(changes, ", "))
				skippedLocal++
				continue
			}

			ahead, err := git.IsAhead(repoPath, branch)
			if err != nil {
				fmt.Printf("%s  %-*s  failed to check if ahead: %v\n", ui.Bold("-"), colWidth, repoPath, err)
				skippedOther++
				continue
			}

			if !ahead {
				fmt.Printf("%s  %-*s  up to date\n", ui.Green("✓"), colWidth, repoPath)
				upToDate++
				continue
			}

			conflicts, err := git.PredictConflicts(repoPath, branch)
			if err != nil {
				fmt.Printf("%s  %-*s  failed to predict conflicts: %v\n", ui.Bold("-"), colWidth, repoPath, err)
				skippedOther++
				continue
			}
			if len(conflicts) > 0 {
				fmt.Printf("%s  %-*s  conflict predicted — skipped (%s)\n", ui.Red("✗"), colWidth, repoPath, strings.Join(conflicts, ", "))
				skippedConflict++
				continue
			}

			if report {
				fmt.Printf("%s  %-*s  safe to pull\n", ui.Cyan("↓"), colWidth, repoPath)
			} else {
				fmt.Printf("%s  %-*s  pulling...", ui.Cyan("↓"), colWidth, repoPath)
				if err := git.Pull(repoPath, branch); err != nil {
					fmt.Printf(" failed: %v\n", err)
					skippedOther++
					continue
				}
				fmt.Printf(" done\n")
				pulled++
			}
		}

		sep := ui.Bold("────────────────────────────────────")
		fmt.Println("\n" + sep)
		fmt.Printf(" %d repos analyzed\n", total)
		fmt.Printf("  %s %d pulled\n", ui.Cyan("↓"), pulled)
		fmt.Printf("  %s %d up to date\n", ui.Green("✓"), upToDate)
		fmt.Printf("  %s %d skipped (local changes)\n", ui.Yellow("⚠"), skippedLocal)
		fmt.Printf("  %s %d skipped (conflict predicted)\n", ui.Red("✗"), skippedConflict)
		fmt.Printf("  %s %d errors or not git repos\n", ui.Bold("-"), skippedOther)
		fmt.Println(sep)

		return nil
	},
}

func init() {
	syncCmd.Flags().Bool("report", false, "analyze only, pull nothing")
	rootCmd.AddCommand(syncCmd)
}
