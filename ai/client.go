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

	return strings.TrimSpace(result.Response), nil
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
