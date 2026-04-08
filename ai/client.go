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
const defaultModel = "llama3.2"

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
	prompt := fmt.Sprintf(`You are a Git expert. The following files have merge conflicts:
%s

Here is the relevant diff:
%s

Explain in plain English why these conflicts exist and how to resolve them.
Be concise — 3-5 sentences max. No markdown, no bullet points, just plain text.`,
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

// SuggestActions suggests what the developer should do based on a sync status report.
func SuggestActions(ollamaURL, model string, summary string) (string, error) {
	prompt := fmt.Sprintf(`You are a Git expert helping a developer manage multiple repositories.
Here is the current sync status of their repos:
%s

Give 2-3 short, specific, actionable suggestions for what they should do next.
No markdown, no bullet points, just plain numbered sentences.`,
		summary,
	)
	return generate(ollamaURL, model, prompt)
}
