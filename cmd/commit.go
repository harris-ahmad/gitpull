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

var commitCmd = &cobra.Command{
	Use:   "commit <repo>",
	Short: "AI-generated commit message with one-keystroke add → commit → push",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath := args[0]

		if !ai.IsAvailable(cfg.OllamaURL) {
			return fmt.Errorf("Ollama is not running. Start it with: ollama serve\nThen pull a model: ollama pull mistral")
		}

		// check repo exists
		if _, err := os.Stat(repoPath); err != nil {
			return fmt.Errorf("repo %q not found in current directory", repoPath)
		}

		// get local diff
		diff, err := git.LocalDiff(repoPath)
		if err != nil {
			return fmt.Errorf("could not get diff: %w", err)
		}
		if diff == "" {
			fmt.Printf("No local changes in %s.\n", repoPath)
			return nil
		}

		fmt.Printf("%s analyzing diff...\n", ui.Cyan("AI"))
		message, err := ai.GenerateCommitMessage(cfg.OllamaURL, cfg.OllamaModel, diff)
		if err != nil {
			return fmt.Errorf("AI error: %w", err)
		}

		// print suggested message
		sep := ui.Bold("────────────────────────────────────")
		fmt.Println("\n" + sep)
		fmt.Println(ui.Cyan("Suggested commit message:"))
		fmt.Println()
		fmt.Println(message)
		fmt.Println(sep)

		// prompt user
		fmt.Printf("\n%s / %s / %s  ",
			ui.Green("y — commit & push"),
			ui.Yellow("e — edit message"),
			ui.Bold("n — abort"),
		)

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(strings.ToLower(input))

		switch choice {
		case "y":
			fmt.Printf("\n%s adding, committing, pushing...\n", ui.Cyan("→"))
			if err := git.AddCommitPush(repoPath, message); err != nil {
				return err
			}
			fmt.Printf("%s done.\n", ui.Green("✓"))

		case "e":
			// write message to a temp file and open in $EDITOR
			tmpFile, err := os.CreateTemp("", "gitpull-commit-*.txt")
			if err != nil {
				return fmt.Errorf("could not create temp file: %w", err)
			}
			tmpFile.WriteString(message)
			tmpFile.Close()

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vim"
			}
			editorCmd := exec.Command(editor, tmpFile.Name())
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr
			if err := editorCmd.Run(); err != nil {
				return fmt.Errorf("editor error: %w", err)
			}

			edited, err := os.ReadFile(tmpFile.Name())
			os.Remove(tmpFile.Name())
			if err != nil {
				return fmt.Errorf("could not read edited message: %w", err)
			}
			finalMessage := strings.TrimSpace(string(edited))
			if finalMessage == "" {
				fmt.Println("Empty message — aborted.")
				return nil
			}

			fmt.Printf("\n%s adding, committing, pushing...\n", ui.Cyan("→"))
			if err := git.AddCommitPush(repoPath, finalMessage); err != nil {
				return err
			}
			fmt.Printf("%s done.\n", ui.Green("✓"))

		default:
			fmt.Println("Aborted.")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)
}
