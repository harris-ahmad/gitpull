package cmd

import (
	"fmt"
	"strings"

	"github.com/harris-ahmad/gitpull/ai"
	"github.com/harris-ahmad/gitpull/git"
	"github.com/harris-ahmad/gitpull/ui"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "sync all repos in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		report, _ := cmd.Flags().GetBool("report")
		useAI, _ := cmd.Flags().GetBool("ai")
		cleanup, _ := cmd.Flags().GetBool("cleanup")

		repos, err := repoList()
		if err != nil {
			return err
		}

		var total, pulled, upToDate, skippedConflict, skippedOther int
		var autoStashed, autoPopped, branchesCleanedUp int

		colWidth := 20
		for _, r := range repos {
			if len(r.Name) > colWidth {
				colWidth = len(r.Name)
			}
		}
		colWidth += 2

		for _, repo := range repos {
			total++

			branch, err := git.DefaultBranch(repo.Path)
			if err != nil {
				fmt.Printf("%s  %-*s  not a git repo, skipped\n", ui.Bold("-"), colWidth, repo.Name)
				skippedOther++
				continue
			}

			if err := git.Fetch(repo.Path); err != nil {
				fmt.Printf("%s  %-*s  failed to fetch: %v\n", ui.Bold("-"), colWidth, repo.Name, err)
				skippedOther++
				continue
			}

			ahead, err := git.IsAhead(repo.Path, branch)
			if err != nil {
				fmt.Printf("%s  %-*s  failed to check if ahead: %v\n", ui.Bold("-"), colWidth, repo.Name, err)
				skippedOther++
				continue
			}

			if !ahead {
				fmt.Printf("%s  %-*s  up to date\n", ui.Green("✓"), colWidth, repo.Name)
				upToDate++
				continue
			}

			conflicts, err := git.PredictConflicts(repo.Path, branch)
			if err != nil {
				fmt.Printf("%s  %-*s  failed to predict conflicts: %v\n", ui.Bold("-"), colWidth, repo.Name, err)
				skippedOther++
				continue
			}
			if len(conflicts) > 0 {
				fmt.Printf("%s  %-*s  conflict predicted — skipped (%s)\n", ui.Red("✗"), colWidth, repo.Name, strings.Join(conflicts, ", "))
				skippedConflict++

				if useAI {
					fmt.Printf("    %s analyzing conflict...\n", ui.Cyan("AI"))
					diff, _ := git.ConflictDiff(repo.Path, branch, conflicts)
					explanation, err := ai.ExplainConflict(cfg.OllamaURL, cfg.OllamaModel, conflicts, diff)
					if err != nil {
						fmt.Printf("    %s could not explain conflict: %v\n", ui.Yellow("AI"), err)
					} else {
						fmt.Printf("    %s %s\n", ui.Cyan("AI"), explanation)
					}
				}
				continue
			}

			// Check for local changes AFTER we know there are no conflicts
			changes, err := git.LocalChanges(repo.Path)
			if err != nil {
				fmt.Printf("%s  %-*s  failed to check local changes: %v\n", ui.Bold("-"), colWidth, repo.Name, err)
				skippedOther++
				continue
			}

			// Auto-stash if there are local changes
			stashed := false
			if len(changes) > 0 {
				if report {
					fmt.Printf("%s  %-*s  would auto-stash local changes (%s)\n", ui.Yellow("⚡"), colWidth, repo.Name, strings.Join(changes, ", "))
				} else {
					if err := git.Stash(repo.Path, "gitpull auto-stash"); err != nil {
						fmt.Printf("%s  %-*s  failed to stash: %v\n", ui.Red("✗"), colWidth, repo.Name, err)
						skippedOther++
						continue
					}
					stashed = true
					autoStashed++
				}

				if useAI {
					fmt.Printf("    %s local changes detected (%s)\n", ui.Cyan("AI"), strings.Join(changes, ", "))
					diff, _ := git.LocalDiff(repo.Path)
					if diff != "" {
						summary, err := ai.SummarizeLocalChanges(cfg.OllamaURL, cfg.OllamaModel, diff)
						if err != nil {
							fmt.Printf("    %s could not summarize: %v\n", ui.Yellow("AI"), err)
						} else {
							fmt.Printf("    %s %s\n", ui.Cyan("AI"), summary)
						}
					}
				}
			}

			if useAI {
				commits, err := git.IncomingCommits(repo.Path, branch)
				if err == nil && len(commits) > 0 {
					summary, err := ai.SummarizeCommits(cfg.OllamaURL, cfg.OllamaModel, commits)
					if err == nil {
						fmt.Printf("    %s incoming: %s\n", ui.Cyan("AI"), summary)
					}
				}
			}

			if report {
				fmt.Printf("%s  %-*s  safe to pull\n", ui.Cyan("↓"), colWidth, repo.Name)
				continue
			}

			// Pull
			fmt.Printf("%s  %-*s  pulling...", ui.Cyan("↓"), colWidth, repo.Name)
			pullErr := git.Pull(repo.Path, branch)

			// Auto-pop stash regardless of pull success/failure (to not lose work)
			if stashed {
				if popErr := git.StashPop(repo.Path); popErr != nil {
					fmt.Printf(" %s (stash restore failed: %v)\n", ui.Red("pull ok but stash lost"), popErr)
					skippedOther++
					continue
				}
				autoPopped++
			}

			if pullErr != nil {
				fmt.Printf(" failed: %v\n", pullErr)
				skippedOther++
				continue
			}

			fmt.Printf(" done\n")
			pulled++

			// Cleanup merged branches if --cleanup flag
			if cleanup {
				deleted, err := git.CleanupMergedBranches(repo.Path)
				if err == nil && len(deleted) > 0 {
					fmt.Printf("    %s cleaned up %d merged branch(es): %s\n",
						ui.Green("✓"), len(deleted), strings.Join(deleted, ", "))
					branchesCleanedUp += len(deleted)
				}
			}
		}

		sep := ui.Bold("────────────────────────────────────")
		fmt.Println("\n" + sep)
		fmt.Printf(" %d repos analyzed\n", total)
		fmt.Printf("  %s %d pulled\n", ui.Cyan("↓"), pulled)
		fmt.Printf("  %s %d up to date\n", ui.Green("✓"), upToDate)
		if autoStashed > 0 {
			fmt.Printf("  %s %d auto-stashed and restored\n", ui.Yellow("⚡"), autoStashed)
		}
		if branchesCleanedUp > 0 {
			fmt.Printf("  %s %d merged branches cleaned up\n", ui.Green("✓"), branchesCleanedUp)
		}
		fmt.Printf("  %s %d skipped (conflict predicted)\n", ui.Red("✗"), skippedConflict)
		fmt.Printf("  %s %d errors or not git repos\n", ui.Bold("-"), skippedOther)
		fmt.Println(sep)

		return nil
	},
}

func init() {
	syncCmd.Flags().Bool("report", false, "analyze only, pull nothing")
	syncCmd.Flags().Bool("ai", false, "enable AI-powered analysis (requires ollama)")
	syncCmd.Flags().Bool("cleanup", false, "cleanup merged branches after successful pull")
	rootCmd.AddCommand(syncCmd)
}
