package cmd

import (
	"fmt"
	"strings"

	"github.com/harris-ahmad/gitpull/ai"
	"github.com/harris-ahmad/gitpull/git"
	"github.com/harris-ahmad/gitpull/ui"
	"github.com/spf13/cobra"
)

var askCmd = &cobra.Command{
	Use:   "ask <question>",
	Short: "Ask a natural language question about your repos",
	Example: `  gitpull ask "what was I working on yesterday?"
  gitpull ask "which repos have the most activity?"
  gitpull ask "which repos should I focus on today?"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		question := strings.Join(args, " ")

		if !ai.IsAvailable(cfg.OllamaURL) {
			return fmt.Errorf("Ollama is not running. Start it with: ollama serve\nThen pull a model: ollama pull mistral")
		}

		repos, err := repoList()
		if err != nil {
			return err
		}

		fmt.Printf("%s gathering repo context...\n", ui.Cyan("AI"))

		var contextParts []string
		for _, repo := range repos {
			ctx, err := git.RepoContext(repo.Path)
			if err != nil || ctx == "" {
				continue
			}
			contextParts = append(contextParts, ctx)
		}

		if len(contextParts) == 0 {
			return fmt.Errorf("no git repos found in current directory")
		}

		fullContext := strings.Join(contextParts, "\n---\n")

		fmt.Printf("%s thinking...\n", ui.Cyan("AI"))
		answer, err := ai.Answer(cfg.OllamaURL, cfg.OllamaModel, fullContext, question)
		if err != nil {
			return fmt.Errorf("AI error: %w", err)
		}

		fmt.Printf("\n%s\n", answer)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(askCmd)
}
