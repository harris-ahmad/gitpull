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

// CleanDryRun returns the list of untracked files/dirs that would be removed.
func CleanDryRun(repoPath string) ([]string, error) {
	out, err := runGitCommand(repoPath, "clean", "-nfd")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return []string{}, nil
	}
	var files []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimPrefix(line, "Would remove ")
		files = append(files, strings.TrimSpace(line))
	}
	return files, nil
}

// Clean removes untracked files and directories from the repo.
func Clean(repoPath string) error {
	_, err := runGitCommand(repoPath, "clean", "-fd")
	return err
}

// Stash saves all local changes in the repo with an optional message.
func Stash(repoPath, message string) error {
	args := []string{"stash", "push", "--include-untracked"}
	if message != "" {
		args = append(args, "-m", message)
	}
	_, err := runGitCommand(repoPath, args...)
	return err
}

// StashPop restores the most recent stash in the repo.
func StashPop(repoPath string) error {
	_, err := runGitCommand(repoPath, "stash", "pop")
	return err
}

// HasStash returns true if the repo has at least one stash entry.
func HasStash(repoPath string) (bool, error) {
	out, err := runGitCommand(repoPath, "stash", "list")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

// GitUserEmail returns the user's git email from global config.
func GitUserEmail() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "config", "user.email")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not get git user.email: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// MyRecentCommits returns commits since the given time string (e.g. "24 hours ago").
func MyRecentCommits(repoPath, since string) ([]string, error) {
	out, err := runGitCommand(repoPath,
		"log",
		"--oneline",
		"--since="+since,
	)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return []string{}, nil
	}
	return strings.Split(out, "\n"), nil
}

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
		line = strings.TrimRight(line, "\r")
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

// LocalDiff returns the full diff of uncommitted local changes.
func LocalDiff(repoPath string) (string, error) {
	out, err := runGitCommand(repoPath, "diff", "HEAD")
	if err != nil {
		return "", err
	}
	if len(out) > 4000 {
		out = out[:4000] + "\n... (truncated)"
	}
	return out, nil
}

// ConflictDiff returns the diff between HEAD and origin/<branch> for the given files.
// This is passed to the AI so it has real context to explain the conflict.
func ConflictDiff(repoPath, branch string, files []string) (string, error) {
	args := append([]string{"diff", "HEAD..origin/" + branch, "--"}, files...)
	out, err := runGitCommand(repoPath, args...)
	if err != nil {
		return "", err
	}
	// truncate to 4000 chars to avoid overwhelming the model
	if len(out) > 4000 {
		out = out[:4000] + "\n... (truncated)"
	}
	return out, nil
}

// git -C <path> log HEAD..origin/<branch> --oneline
func IncomingCommits(repoPath, branch string) ([]string, error) {
	out, err := runGitCommand(repoPath, "log", "HEAD..origin/"+branch, "--oneline")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return []string{}, nil
	}
	return strings.Split(out, "\n"), nil
}

// AddCommitPush stages all changes, commits with the given message, and pushes.
func AddCommitPush(repoPath, message string) error {
	if _, err := runGitCommand(repoPath, "add", "-A"); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}
	if _, err := runGitCommand(repoPath, "commit", "-m", message); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}
	if _, err := runGitCommand(repoPath, "push"); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}
	return nil
}

// git -C <path> pull origin <branch>
func Pull(repoPath string, branch string) error {
	_, err := runGitCommand(repoPath, "pull", "origin", branch)
	return err
}

// RepoContext collects a concise summary of a repo's state for AI context.
// Skips repos with no recent activity to keep the prompt small.
func RepoContext(repoPath string) (string, error) {
	var b strings.Builder

	// current branch
	branch, err := runGitCommand(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}

	// local changes
	var localFiles []string
	status, _ := runGitCommand(repoPath, "status", "--porcelain")
	if status != "" {
		for _, line := range strings.Split(status, "\n") {
			line = strings.TrimRight(line, "\r")
			if len(line) >= 3 {
				localFiles = append(localFiles, strings.TrimSpace(line[3:]))
			}
		}
	}

	// commits behind
	behindCount := "0"
	defaultBranch, err := DefaultBranch(repoPath)
	if err == nil {
		behind, _ := runGitCommand(repoPath, "rev-list", "HEAD..origin/"+defaultBranch, "--count")
		behindCount = strings.TrimSpace(behind)
	}

	// commits from the last 7 days only
	log, _ := runGitCommand(repoPath, "log", "--oneline", "--since=7 days ago", "-5")

	// skip repos with zero recent activity
	if len(localFiles) == 0 && behindCount == "0" && log == "" {
		return "", nil
	}

	b.WriteString("repo: " + repoPath + "\n")
	b.WriteString("branch: " + branch + "\n")

	if len(localFiles) > 0 {
		b.WriteString("local changes: " + strings.Join(localFiles, ", ") + "\n")
	} else {
		b.WriteString("local changes: none\n")
	}

	b.WriteString("commits behind: " + behindCount + "\n")

	if log != "" {
		b.WriteString("recent commits:\n")
		for _, line := range strings.Split(log, "\n") {
			b.WriteString("  " + line + "\n")
		}
	}

	return b.String(), nil
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
	
	return strings.TrimRight(stdout.String(), "\n\r"), nil
}