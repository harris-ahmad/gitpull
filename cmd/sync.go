package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/harris-ahmad/gitpull/git"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use: "sync",
	Short: "sync all repos in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		report, _ := cmd.Flags().GetBool("report")

		dirs, err := os.ReadDir(".")
		if err != nil {
			return fmt.Errorf("failed to read current directory: %w", err)
		}

		var total, pulled, upToDate, skippedLocal, skippedConflict, skippedOther int

		for _, dir := range dirs {
			if !dir.IsDir() {
				continue
			}

			total++
			repoPath := dir.Name()

			branch, err := git.DefaultBranch(repoPath)
			
			if err != nil {
				fmt.Printf("-  %-20s not a git repo, skipped\n", repoPath)
				skippedOther++
				continue
			}

			if err := git.Fetch(repoPath); err != nil {
				fmt.Printf("-  %-20s failed to fetch: %v\n", repoPath, err)
				skippedOther++
				continue
			}

			changes, err := git.LocalChanges(repoPath)

			if err != nil {
				fmt.Printf("-  %-20s failed to check local changes: %v\n", repoPath, err)
				skippedOther++
				continue
			}
			if len(changes) > 0 {
				fmt.Printf("⚠  %-20s local changes — skipped (%s)\n", repoPath, strings.Join(changes, ", "))
				skippedLocal++
				continue
			}

			ahead, err := git.IsAhead(repoPath, branch)
			if err != nil {
				fmt.Printf("-  %-20s failed to check if ahead: %v\n", repoPath, err)
				skippedOther++
				continue
			}

			if !ahead {
				fmt.Printf("✓  %-20s up to date\n", repoPath)
				upToDate++
				continue
			}

			conflicts, err := git.PredictConflicts(repoPath, branch)
			if err != nil {
				fmt.Printf("-  %-20s failed to predict conflicts: %v\n", repoPath, err)
				skippedOther++
				continue
			}
			if len(conflicts) > 0 {
				fmt.Printf("✗  %-20s conflict predicted — skipped (%s)\n", repoPath, strings.Join(conflicts, ", "))
				skippedConflict++
				continue
			}

			if report {
				fmt.Printf("↓  %-20s safe to pull\n", repoPath)
			} else {
				fmt.Printf("↓  %-20s pulling...", repoPath)
				if err := git.Pull(repoPath, branch); err != nil {
					fmt.Printf(" failed: %v\n", err)
					skippedOther++
					continue
				}
				fmt.Printf(" done\n")
				pulled++
			}
		}

		summary := "\n--------------------------------\n"
		summary += fmt.Sprintf("summary: %d repos checked\n", total)
		summary += fmt.Sprintf("  ✓ %d up to date\n", upToDate)
		summary += fmt.Sprintf("  ⚠ %d local changes\n", skippedLocal)
		summary += fmt.Sprintf("  ✗ %d conflicts predicted\n", skippedConflict)
		summary += fmt.Sprintf("  ↓ %d pulled\n", pulled)
		summary += "--------------------------------\n"
		
		fmt.Println(summary)

		return nil
	},
}

func init() {
	syncCmd.Flags().Bool("report", false, "analyze only, pull nothing")
	rootCmd.AddCommand(syncCmd)
}