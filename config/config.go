package config

import (
	"os"
	"path/filepath"
	"fmt"
	"strconv"
	"strings"
)

type Config struct {
	Token      string
	Parallel   int
	SSH        bool
	GeminiKey  string // kept for future use
	OllamaURL  string
	OllamaModel string
}

func Load() (*Config, error) {

	//find the file at ~/.gitpullrc using os.UserHomeDir
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	filePath := filepath.Join(home, ".gitpullrc")
	lines, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{
				Parallel:    4,
				Token:       os.Getenv("GITHUB_TOKEN"),
				GeminiKey:   os.Getenv("GEMINI_API_KEY"),
				OllamaURL:   "http://localhost:11434",
				OllamaModel: "llama3.2",
			}, nil
		}
		return nil, fmt.Errorf("failed to read .gitpullrc: %w", err)
	}

	//split each line on "=", trims whitespace, maps key -> value
	mapped := make(map[string]string)
	for _, rawLine := range strings.Split(string(lines), "\n") {
		//skips blank lines and lines starting with #
		line := strings.TrimSpace(rawLine)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		mapped[key] = value
	}

	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" {
		geminiKey = mapped["gemini_key"]
	}

	parallel := 4
	if rawParallel, ok := mapped["parallel"]; ok && rawParallel != "" {
		parsedParallel, err := strconv.Atoi(rawParallel)
		if err != nil {
			return nil, fmt.Errorf("invalid parallel value %q: %w", rawParallel, err)
		}
		parallel = parsedParallel
	}

	ollamaURL := mapped["ollama_url"]
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	ollamaModel := mapped["ollama_model"]
	if ollamaModel == "" {
		ollamaModel = "mistral"
	}

	return &Config{
		Token:       mapped["token"],
		Parallel:    parallel,
		SSH:         mapped["ssh"] == "true",
		GeminiKey:   geminiKey,
		OllamaURL:   ollamaURL,
		OllamaModel: ollamaModel,
	}, nil

}