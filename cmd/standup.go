package cmd

import (
	"fmt"
	"os"

	"github.com/harris-ahmad/gitpull/ai"
	"github.com/harris-ahmad/gitpull/git"
	"github.com/harris-ahmad/gitpull/ui"
	"github.com/spf13/cobra"
)

var standupCmd = &cobra.Command{
	Use:   "standup",
	Short: "Generate a standup summary from your recent commits across all repos",
	RunE: func(cmd *cobra.Command, args []string) error {
		since, _ := cmd.Flags().GetString("since")

		email, err := git.GitUserEmail()
		if err != nil {
			return fmt.Errorf("could not determine git user: %w", err)
		}

		dirs, err := os.ReadDir(".")
		if err != nil {
			return fmt.Errorf("failed to read current directory: %w", err)
		}

		fmt.Printf("%s collecting commits since %s...\n", ui.Cyan("AI"), since)

		repoCommits := make(map[string][]string)
		for _, dir := range dirs {
			if !dir.IsDir() {
				continue
			}
			commits, err := git.MyRecentCommits(dir.Name(), since, email)
			if err != nil || len(commits) == 0 {
				continue
			}
			// cap at 5 commits per repo to avoid overwhelming the model
			if len(commits) > 5 {
				commits = commits[:5]
			}
			repoCommits[dir.Name()] = commits
		}

		if len(repoCommits) == 0 {
			fmt.Printf("No commits found in the last %s under %s.\n", since, email)
			return nil
		}

		// print raw commits first
		fmt.Println()
		for repo, commits := range repoCommits {
			fmt.Printf("%s %s\n", ui.Bold(repo), "")
			for _, c := range commits {
				fmt.Printf("  %s\n", c)
			}
		}

		fmt.Printf("\n%s generating standup...\n", ui.Cyan("AI"))
		standup, err := ai.GenerateStandup(cfg.OllamaURL, cfg.OllamaModel, repoCommits)
		if err != nil {
			return fmt.Errorf("AI error: %w", err)
		}

		sep := ui.Bold("────────────────────────────────────")
		fmt.Println("\n" + sep)
		fmt.Printf("%s\n", standup)
		fmt.Println(sep)

		return nil
	},
}

func init() {
	standupCmd.Flags().String("since", "24 hours ago", "how far back to look for commits")
	rootCmd.AddCommand(standupCmd)
}
