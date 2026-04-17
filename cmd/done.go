package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/harris-ahmad/gitpull/ai"
	"github.com/harris-ahmad/gitpull/git"
	"github.com/harris-ahmad/gitpull/ui"
	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use: "done",
	Short: "Finish your work: commit, push and create a PR",
	Long: `
	Stages all changes, generates an AI commit message, commits, pushes and creates a GitHub PR.
	This is the companion to 'gitpull start'. After you've finished working on your feature branch, run this command.
	`,

	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := repoList()
		if err != nil {
			return err
		}
		if len(repos) > 1 {
			return fmt.Errorf("done command only works inside a single repo, not across multiple repos")
		}

		repo := repos[0]
		currentBranch, err := git.CurrentBranch(repo.Path)
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
		
		if currentBranch == "main" ||
		currentBranch == "master" ||
		currentBranch == "dev" {
			return fmt.Errorf("you're on %s - use 'gitpull start <feature-name>' to start a new feature branch first", currentBranch)
		}

		changes, err := git.LocalChanges(repo.Path)
		if err != nil {
			return fmt.Errorf("failed to check for local changes: %w", err)
		}
		if len(changes)==0 {
			return fmt.Errorf("no changes to commit")
		}

		fmt.Printf("%s Found changes: %s\n", ui.Green("✓"), strings.Join(changes, ", "))
		if !ai.IsAvailable(cfg.OllamaURL) {
			return fmt.Errorf("Ollama is not running. Start it with: ollama serve")
		}

		fmt.Printf("%s Analyzing diff...\n", ui.Cyan("AI"))

		diff, err := git.LocalDiff(repo.Path)
		if err != nil || diff == "" {
			return fmt.Errorf("failed to get diff: %w", err)
		}

		message, err := ai.GenerateCommitMessage(cfg.OllamaURL, cfg.OllamaModel, diff)
		if err != nil {
			return fmt.Errorf("AI error: %w", err)
		}

		sep := ui.Bold("────────────────────────────────────")
		fmt.Printf("\n%s\n", sep)
		fmt.Printf("Suggested commit message:\n\n%s\n", message)
		fmt.Printf("%s\n\n", sep)

		fmt.Print("Accept (y), edit (e), or reject (n)? ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		finalMessage := message

		switch response {
			case "y":
				finalMessage = message
			case "e":
				finalMessage, err = editInEditor(message)
				if err != nil {
					return fmt.Errorf("failed to open editor: %w", err)
				}
			case "n":
				fmt.Println("Commit cancelled.")
				return nil
			default:
				fmt.Println("Invalid response. Commit cancelled.")
				return nil
		}

		fmt.Printf("%s Committing and pushing...\n", ui.Cyan("→"))
		if err := git.AddCommitPush(repo.Path, finalMessage); err != nil {
			return fmt.Errorf("failed to commit/push: %w", err)
		}

		fmt.Printf("%s Committed and pushed to %s\n", ui.Green("✓"), ui.Bold(currentBranch))

		fmt.Printf("%s Creating PR...\n", ui.Cyan("→"))

		prUrl, err := createPR(repo.Path, currentBranch, finalMessage)
		if err != nil {
			return fmt.Errorf("failed to create PR: %w", err)
		}
		fmt.Printf("%s PR created: %s\n", ui.Green("✓"), prUrl)
		
		return nil
	},
}

func editInEditor(initialContent string) (string, error) {
	tmpFile, err := os.CreateTemp("", "gitpull-commit-*.txt")
	if err != nil {
		return "", err
	}

	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(initialContent); err != nil {
		return "", err
	}
	tmpFile.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	edited, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(edited)), nil
}

func createPR(repoPath, branch, commitMessage string) (string, error) {
	lines := strings.Split(commitMessage, "\n")
	title := lines[0]

	body := "Automated PR created by gitpull.\n\n"
	if len(lines) > 1 {
		body += strings.Join(lines[1:], "\n")
	}

	cmd := exec.Command("gh", "pr", "create", "--title", title, "--body", body)
	cmd.Dir = repoPath
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gh pr create failed (is gh CLI installed?): %w", err)
	}

	prURL := strings.TrimSpace(string(output))
	return prURL, nil
}

func init() {
	rootCmd.AddCommand(doneCmd)
}