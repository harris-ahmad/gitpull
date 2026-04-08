package cmd

import (
	"fmt"
	"os"
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


		dirs, err := os.ReadDir(".")
		if err != nil {
			return fmt.Errorf("failed to read current directory: %w", err)
		}

		var total, pulled, upToDate, skippedLocal, skippedConflict, skippedOther int

		// track summary lines for AI suggestions at the end
		var summaryLines []string

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
				summaryLines = append(summaryLines, repoPath+": not a git repo")
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
				summaryLines = append(summaryLines, repoPath+": has local changes in "+strings.Join(changes, ", "))
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
				summaryLines = append(summaryLines, repoPath+": up to date")
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
				summaryLines = append(summaryLines, repoPath+": conflict predicted in "+strings.Join(conflicts, ", "))

				// AI: explain the conflict
				if useAI {
					fmt.Printf("    %s analyzing conflict...\n", ui.Cyan("AI"))
					explanation, err := ai.ExplainConflict(cfg.OllamaURL, cfg.OllamaModel, conflicts, "")
					if err != nil {
						fmt.Printf("    %s could not explain conflict: %v\n", ui.Yellow("AI"), err)
					} else {
						fmt.Printf("    %s %s\n", ui.Cyan("AI"), explanation)
					}
				}
				continue
			}

			// safe to pull — optionally summarize incoming commits
			if useAI {
				commits, err := git.IncomingCommits(repoPath, branch)
				if err == nil && len(commits) > 0 {
					summary, err := ai.SummarizeCommits(cfg.OllamaURL, cfg.OllamaModel, commits)
					if err == nil {
						fmt.Printf("    %s incoming: %s\n", ui.Cyan("AI"), summary)
					}
				}
			}

			if report {
				fmt.Printf("%s  %-*s  safe to pull\n", ui.Cyan("↓"), colWidth, repoPath)
				summaryLines = append(summaryLines, repoPath+": safe to pull")
			} else {
				fmt.Printf("%s  %-*s  pulling...", ui.Cyan("↓"), colWidth, repoPath)
				if err := git.Pull(repoPath, branch); err != nil {
					fmt.Printf(" failed: %v\n", err)
					skippedOther++
					continue
				}
				fmt.Printf(" done\n")
				pulled++
				summaryLines = append(summaryLines, repoPath+": pulled successfully")
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

		// AI: suggest actions only if there's something worth acting on
		if useAI && (skippedLocal > 0 || skippedConflict > 0) {
			fmt.Printf("\n%s generating suggestions...\n", ui.Cyan("AI"))
			suggestions, err := ai.SuggestActions(cfg.OllamaURL, cfg.OllamaModel, strings.Join(summaryLines, "\n"))
			if err != nil {
				fmt.Printf("%s could not generate suggestions: %v\n", ui.Yellow("AI"), err)
			} else {
				fmt.Printf("%s\n%s\n", ui.Cyan("AI Suggestions:"), suggestions)
			}
		}

		return nil
	},
}

func init() {
	syncCmd.Flags().Bool("report", false, "analyze only, pull nothing")
	syncCmd.Flags().Bool("ai", false, "enable AI-powered conflict explanation and suggestions (requires GEMINI_API_KEY)")
	rootCmd.AddCommand(syncCmd)
}
