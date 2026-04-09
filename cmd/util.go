package cmd

import (
	"os"
	"path/filepath"

	"github.com/harris-ahmad/gitpull/git"
)

// Repo holds the git path and the display name for a repository.
type Repo struct {
	Path string // path used for git operations (e.g. "." or "myrepo")
	Name string // name shown in output (e.g. "gitpull" or "myrepo")
}

// repoList returns the repos to operate on.
// If subdirectory repos exist, it returns those.
// If none exist but the current directory is itself a repo, it returns ["."].
func repoList() ([]Repo, error) {
	dirs, err := os.ReadDir(".")
	if err != nil {
		return nil, err
	}

	var repos []Repo
	for _, dir := range dirs {
		if dir.IsDir() && git.IsRepo(dir.Name()) {
			repos = append(repos, Repo{Path: dir.Name(), Name: dir.Name()})
		}
	}

	if len(repos) == 0 && git.IsRepo(".") {
		cwd, err := os.Getwd()
		name := "."
		if err == nil {
			name = filepath.Base(cwd)
		}
		return []Repo{{Path: ".", Name: name}}, nil
	}

	return repos, nil
}
