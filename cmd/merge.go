package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/harris-ahmad/gitpull/git"
	"github.com/harris-ahmad/gitpull/ui"
	"github.com/spf13/cobra"
)

var mergeCmd = &cobra.Command{
	Use: "merge",
	Short: "Auto-merge your PR when CI passes",
	Long: `Watches your pull request and automatically merges it when all CI checks pass.

  After running 'gitpull done' to create a PR, use this command to automatically merge it once CI 
  completes.
  It will:
    1. Wait for all CI checks to pass
    2. Merge the PR
    3. Delete the remote branch
    4. Switch back to main and pull
    5. Delete the local feature branch`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := repoList()
		if err != nil {
			return err
		}

		if len(repos) > 1 {
			return fmt.Errorf("multiple repos detected. please run 'gitpull merge' in each repo separately")
		}

		repo := repos[0]

		currentBranch, err := git.CurrentBranch(repo.Path)
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		if currentBranch == "main" || currentBranch == "master" || currentBranch == "dev" {
			return fmt.Errorf("cannot merge main/master/dev branch. please switch to a feature branch")
		}

		fmt.Printf("%s Checking for open PR on %s...\n", ui.Cyan("→"), ui.Bold(currentBranch))

		prUrl, err := getPRURL(repo.Path, currentBranch)
		if err != nil {
			return fmt.Errorf("no open PR found for %s: %w", currentBranch, err)
		}

		fmt.Printf("%s Found PR: %s\n", ui.Green("✓"), prUrl)
		fmt.Printf("%s Waiting for CI checks to complete...\n", ui.Cyan("→"))

		if err := waitForChecks(repo.Path, currentBranch); err != nil {
			return err
		}

		fmt.Printf("%s All checks passed! Merging PR...\n", ui.Green("✓"))

		if err := mergePR(repo.Path); err != nil {
			return fmt.Errorf("failed to merge PR: %w", err)
		}

		fmt.Printf("%s PR merged successfully\n", ui.Green("✓"))
		fmt.Printf("%s Switching back to main and cleaning up...\n", ui.Cyan("→"))

		if err := git.SwitchToDefaultBranch(repo.Path); err != nil {
			return fmt.Errorf("failed to switch to main: %w", err)
		}

		deleteCmd := exec.Command("git", "-C", repo.Path, "branch", "-D", currentBranch)

		if err := deleteCmd.Run(); err != nil {
			fmt.Printf("%s Warning: failesd to delete local branch %s: %v\n", ui.Yellow("⚠"), currentBranch, err)	
		} else {
			fmt.Printf("%s Deleted local branch %s\n", ui.Green("✓"), ui.Bold(currentBranch))
		}

		fmt.Printf("%s Cleaned up successfully\n", ui.Green("✓"))
		return nil
	},
}

func getPRURL(repoPath, branch string) (string, error) {
	cmd := exec.Command("gh", "pr", "view", branch, "--json", "url", "--jq", ".url")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func waitForChecks(repoPath, branch string) error {
	for {
		cmd := exec.Command("gh", "pr", "checks", branch)
		cmd.Dir = repoPath
		out, err := cmd.Output()
		if err != nil { return fmt.Errorf("failed to check CI status: %w", err) }

		status := string(out)

		if strings.Contains(status, "All checks have passed") || !strings.Contains(status, "pending") {
			if strings.Contains(status, "fail") {
				return fmt.Errorf("CI checks failed. please fix the issues and try again")
			}
			return nil
		}

		fmt.Printf("%s CI checks in progress... (this may take a few minutes)\n", ui.Cyan("→"))
		time.Sleep(10 * time.Second)
	}
}

func mergePR(repoPath string) error {
	cmd := exec.Command("gh", "pr", "merge", "--squash", "--delete-branch")
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func init() {
	rootCmd.AddCommand(mergeCmd)
}