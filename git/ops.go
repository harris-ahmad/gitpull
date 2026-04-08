package git

import (
	"fmt"
	"os/exec"
	"context"
	"time"
	"bytes"
	"strings"
	"strconv"
)

func DefaultBranch(repoPath string) (string, error) {
	out, err := runGitCommand(repoPath, "rev-parse", "--abbrev-ref", "origin/HEAD")
	if err != nil {
		return "", err
	}

	branch := strings.TrimPrefix(out, "origin/")
	if branch == out {
		return "", fmt.Errorf("unexpected output from rev-parse: %s", out)
	}
	return branch, nil
}

//git -c <path> fetch origin
func Fetch(repoPath string) error {
	_, err := runGitCommand(repoPath, "fetch", "origin")
	return err
}

//git -c <path> status --porcelain
func LocalChanges(repoPath string) ([]string, error) {
	out, err := runGitCommand(repoPath, "status", "--porcelain")
	if err != nil {
		return nil, err
	}

	if out == "" {
		return []string{}, nil
	}

	changes := []string{}
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 3 {
			continue
		}
		changes = append(changes, strings.TrimSpace(line[3:]))
	}
	return changes, nil
}

//git -c <path> rev-list HEAD..origin/<branch> --count
func IsAhead(repoPath string, branch string) (bool, error) {
	out, err := runGitCommand(repoPath, "rev-list", "HEAD..origin/"+branch, "--count")
	if err != nil {
		return false, err
	}

	count, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

//git -C <path> merge-base HEAD origin/<branch>
//git -C <path> merge-tree <hash> HEAD origin/<branch>
func PredictConflicts(repoPath string, branch string) ([]string, error) {
	baseHash, err := runGitCommand(repoPath, "merge-base", "HEAD", "origin/"+branch)
	if err != nil {
		return nil, err
	}

	out, err := runGitCommand(repoPath, "merge-tree", baseHash, "HEAD", "origin/"+branch)
	if err != nil {
		return nil, err
	}

	conflicts := []string{}

	//split by whitespace and take the last token. 
	//lines containing the word "conflict" are conflicts.
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "conflict") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				conflicts = append(conflicts, fields[len(fields)-1])
			}
		}
	}
	return conflicts, nil
}

// git -C <path> pull origin <branch>
func Pull(repoPath string, branch string) error {
	_, err := runGitCommand(repoPath, "pull", "origin", branch)
	return err
}

func runGitCommand(dir string, args ...string) (string, error) {
	fullArgs := append([]string{"-C", dir}, args...)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", fullArgs...)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}
	
	return strings.TrimSpace(stdout.String()), nil
}