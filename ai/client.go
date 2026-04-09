package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultOllamaURL = "http://localhost:11434"
const defaultModel = "mistral"

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
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

// generate sends a prompt to Ollama and returns the response text.
func generate(ollamaURL, model, prompt string) (string, error) {
	if ollamaURL == "" {
		ollamaURL = defaultOllamaURL
	}
	if model == "" {
		model = defaultModel
	}

	body, err := json.Marshal(ollamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(ollamaURL+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("could not reach Ollama at %s — is it running? (ollama serve): %w", ollamaURL, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result ollamaResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("Ollama error: %s", result.Error)
	}

	return stripMarkdown(strings.TrimSpace(result.Response)), nil
}

// stripMarkdown removes common markdown formatting so output renders cleanly in the terminal.
func stripMarkdown(s string) string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		// remove bold/italic markers
		line = strings.ReplaceAll(line, "**", "")
		line = strings.ReplaceAll(line, "__", "")
		line = strings.ReplaceAll(line, "*", "")
		// convert markdown bullets to a clean dash
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "• ") {
			line = "  " + trimmed
		}
		// strip inline code backticks
		line = strings.ReplaceAll(line, "`", "")
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// ExplainConflict explains why a merge conflict exists and how to resolve it.
func ExplainConflict(ollamaURL, model string, conflictingFiles []string, diff string) (string, error) {
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
	return generate(ollamaURL, model, prompt)
}

// SummarizeCommits summarizes a list of incoming commit messages before a pull.
func SummarizeCommits(ollamaURL, model string, commits []string) (string, error) {
	prompt := fmt.Sprintf(`You are a Git expert. The following commits are about to be pulled into a local repository:
%s

Summarize what these changes do in 1-2 sentences. Be concise and specific.
No markdown, no bullet points, just plain text.`,
		strings.Join(commits, "\n"),
	)
	return generate(ollamaURL, model, prompt)
}

// GenerateCommitMessage suggests a commit message based on the diff of local changes.
func GenerateCommitMessage(ollamaURL, model, diff string) (string, error) {
	prompt := fmt.Sprintf(`You are an expert software engineer helping write a git commit message.

Here is the diff of local changes:
%s

Write a single commit message following the Conventional Commits format:
<type>(<optional scope>): <short summary>

<optional body explaining what and why, not how — max 2 sentences>

Types: feat, fix, chore, refactor, docs, test, style, perf
Rules:
- Summary line must be under 72 characters
- Be specific — reference actual functions, files, or behaviour changed
- No markdown, no bullet points, plain text only
- Output only the commit message, nothing else`,
		diff,
	)
	return generate(ollamaURL, model, prompt)
}

// GenerateStandup produces a standup summary from a map of repo → commits.
func GenerateStandup(ollamaURL, model string, repoCommits map[string][]string) (string, error) {
	var b strings.Builder
	for repo, commits := range repoCommits {
		b.WriteString("repo: " + repo + "\n")
		for _, c := range commits {
			b.WriteString("  - " + c + "\n")
		}
	}

	prompt := fmt.Sprintf(`You are helping a software developer write their daily standup update.

Here are the commits they made, grouped by repository:
%s

Rules:
- Exactly one sentence per repository. Never repeat a repository name.
- Start each sentence with the repo name followed by a colon, e.g. "gitpull: I..."
- Be specific — reference actual features or fixes from the commits.
- No introduction, no header, no markdown. Output the sentences directly.`,
		b.String(),
	)
	return generate(ollamaURL, model, prompt)
}

// Answer responds to a natural language question about the user's repos using gathered context.
func Answer(ollamaURL, model, context, question string) (string, error) {
	prompt := fmt.Sprintf(`You are an assistant helping a developer manage their Git repositories.

Here is the current state of their repositories:
%s

The developer asks: %s

Answer based only on the repository data above. Be specific and concise.
Reference exact repo names, branch names, and commit messages from the data.
Do not make up information that isn't in the data. Plain text only.`,
		context,
		question,
	)
	return generate(ollamaURL, model, prompt)
}

// SummarizeLocalChanges explains what the developer was working on based on their local diff.
func SummarizeLocalChanges(ollamaURL, model string, diff string) (string, error) {
	prompt := fmt.Sprintf(`You are a Git expert. A developer has uncommitted local changes in a repository.

Here is the diff of their local changes:
%s

In 1-2 sentences, summarize what the developer was working on based on these changes.
Be specific — mention the files, functions, or concepts involved.
Do not give advice or suggest commands. Just describe what they were doing.
Plain text only.`,
		diff,
	)
	return generate(ollamaURL, model, prompt)
}
