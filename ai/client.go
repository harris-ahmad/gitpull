package ai

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/outputparser"
	"github.com/tmc/langchaingo/prompts"
)

const defaultOllamaURL = "http://localhost:11434"
const defaultModel = "mistral"

// newLLM creates a LangChain Ollama LLM instance.
func newLLM(ollamaURL, model string) (llms.Model, error) {
	if ollamaURL == "" {
		ollamaURL = defaultOllamaURL
	}
	if model == "" {
		model = defaultModel
	}
	return ollama.New(
		ollama.WithModel(model),
		ollama.WithServerURL(ollamaURL),
	)
}

// IsAvailable checks if Ollama is running and reachable.
func IsAvailable(ollamaURL string) bool {
	if ollamaURL == "" {
		ollamaURL = defaultOllamaURL
	}
	resp, err := http.Get(ollamaURL)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ExplainConflict explains why a merge conflict exists and how to resolve it.
func ExplainConflict(ollamaURL, model string, conflictingFiles []string, diff string) (string, error) {
	llm, err := newLLM(ollamaURL, model)
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf(`You are a Git expert helping a developer resolve merge conflicts.

Conflicting files:
%s

Diff between local HEAD and remote branch:
%s

Analyze the diff carefully and do the following:
1. For each conflicting file, identify the exact line numbers or functions where the conflict occurs.
2. Explain what change was made locally vs what the remote introduced.
3. Explain why these two changes conflict.
4. Suggest the most likely correct resolution.

Be specific — reference actual function names, variable names, and line numbers from the diff.
Plain text only, no markdown.`,
		strings.Join(conflictingFiles, "\n"),
		diff,
	)

	return llms.GenerateFromSinglePrompt(context.Background(), llm, prompt)
}

// SummarizeCommits summarizes a list of incoming commit messages before a pull.
func SummarizeCommits(ollamaURL, model string, commits []string) (string, error) {
	llm, err := newLLM(ollamaURL, model)
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf(`You are a Git expert. The following commits are about to be pulled:
%s

Summarize what these changes do in 1-2 sentences. Be concise and specific.
Plain text only, no markdown.`,
		strings.Join(commits, "\n"),
	)

	return llms.GenerateFromSinglePrompt(context.Background(), llm, prompt)
}

// SummarizeLocalChanges explains what the developer was working on based on their local diff.
func SummarizeLocalChanges(ollamaURL, model string, diff string) (string, error) {
	llm, err := newLLM(ollamaURL, model)
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf(`You are a Git expert. A developer has uncommitted local changes.

Diff:
%s

In 1-2 sentences, summarize what the developer was working on.
Be specific — mention files, functions, or concepts involved.
Do not give advice. Plain text only.`,
		diff,
	)

	return llms.GenerateFromSinglePrompt(context.Background(), llm, prompt)
}

// GenerateCommitMessage uses a two-step LangChain pipeline to generate a clean commit message.
func GenerateCommitMessage(ollamaURL, model string, diff string) (string, error) {
	llm, err := newLLM(ollamaURL, model)
	if err != nil {
		return "", err
	}
	ctx := context.Background()

	// step 1: generate commit message
	step1Prompt := fmt.Sprintf(`You are an expert software engineer. Write a git commit message for this diff.
Follow Conventional Commits format: <type>(<scope>): <summary>

Types: feat, fix, chore, refactor, docs, test, style, perf
Rules:
- Output ONLY the commit message, nothing else
- Summary line under 72 characters
- Optional body: max 2 sentences explaining what and why
- No explanations, no commentary, no intro sentences

Diff:
%s`, diff)

	raw, err := llms.GenerateFromSinglePrompt(ctx, llm, step1Prompt)
	if err != nil {
		return "", err
	}

	// step 2: extract just the commit message using LangChain
	step2Prompt := fmt.Sprintf(`Extract only the git commit message from the text below.
Return only the commit message — no explanation, no intro, no commentary.
The message starts with a type: feat, fix, chore, refactor, docs, test, style, or perf.

Text:
%s`, raw)

	cleaned, err := llms.GenerateFromSinglePrompt(ctx, llm, step2Prompt)
	if err != nil {
		return strings.TrimSpace(raw), nil
	}

	return strings.TrimSpace(cleaned), nil
}

// GenerateStandup produces a standup summary from a map of repo → commits.
func GenerateStandup(ollamaURL, model string, repoCommits map[string][]string) (string, error) {
	llm, err := newLLM(ollamaURL, model)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	repos := make([]string, 0, len(repoCommits))
	for repo, commits := range repoCommits {
		repos = append(repos, repo)
		b.WriteString("REPO: " + repo + "\n")
		b.WriteString("COMMITS:\n")
		for _, c := range commits {
			b.WriteString("  - " + c + "\n")
		}
		b.WriteString("\n")
	}

	prompt := fmt.Sprintf(`You are helping a developer write their daily standup.

There are %d repositories: %s

For each repository, the commits made today are listed below:
%s
Write exactly one standup sentence per repository.
Each sentence must start with the repository name followed by a colon.
Summarize what was worked on based only on the commits listed — do not invent work.
Output only the sentences, no introduction, no markdown.`,
		len(repos),
		strings.Join(repos, ", "),
		b.String(),
	)

	out, err := llms.GenerateFromSinglePrompt(context.Background(), llm, prompt)
	if err != nil {
		return "", err
	}
	return stripMarkdown(out), nil
}

// NewConversation creates a conversational chain with memory for gitpull ask.
// Returns a function that takes a question and returns an answer.
func NewConversation(ollamaURL, model, repoContext string) (func(string) (string, error), error) {
	llm, err := newLLM(ollamaURL, model)
	if err != nil {
		return nil, err
	}

	systemPrompt := `You are a Git assistant helping a developer manage their repositories.

Here is the current state of their repositories:
` + repoContext + `

Answer questions based only on this data. Be specific — reference exact repo names,
branch names, and commit messages. Do not make up information not in the data.
Plain text only, no markdown.

Current conversation:
{{.history}}
Human: {{.input}}
AI:`

	prompt := prompts.NewPromptTemplate(systemPrompt, []string{"history", "input"})
	mem := memory.NewConversationBuffer()

	chain := chains.LLMChain{
		Prompt:       prompt,
		LLM:          llm,
		Memory:       mem,
		OutputParser: outputparser.NewSimple(),
		OutputKey:    "text",
	}

	return func(question string) (string, error) {
		out, err := chains.Call(
			context.Background(),
			&chain,
			map[string]any{"input": question},
		)
		if err != nil {
			return "", err
		}
		raw, _ := out["text"].(string)
		return stripMarkdown(strings.TrimSpace(raw)), nil
	}, nil
}

// Answer sends a single question with repo context (non-conversational fallback).
func Answer(ollamaURL, model, repoContext, question string) (string, error) {
	llm, err := newLLM(ollamaURL, model)
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf(`You are a Git assistant helping a developer manage their repositories.

Repository state:
%s

Question: %s

Answer based only on the data above. Be specific. Plain text only.`,
		repoContext, question,
	)

	out, err := llms.GenerateFromSinglePrompt(context.Background(), llm, prompt)
	if err != nil {
		return "", err
	}
	return stripMarkdown(strings.TrimSpace(out)), nil
}

// stripMarkdown removes markdown formatting for clean terminal output.
func stripMarkdown(s string) string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.ReplaceAll(line, "**", "")
		line = strings.ReplaceAll(line, "__", "")
		line = strings.ReplaceAll(line, "*", "")
		line = strings.ReplaceAll(line, "`", "")
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
