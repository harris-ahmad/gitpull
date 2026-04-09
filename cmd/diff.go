package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/harris-ahmad/gitpull/git"
	"github.com/harris-ahmad/gitpull/ui"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show staged and unstaged changes across all repos",
	RunE: func(cmd *cobra.Command, args []string) error {
		dirs, err := os.ReadDir(".")
		if err != nil {
			return fmt.Errorf("failed to read current directory: %w", err)
		}

		var buf bytes.Buffer
		found := false

		for _, dir := range dirs {
			if !dir.IsDir() {
				continue
			}
			repoPath := dir.Name()

			diff, err := git.Diff(repoPath)
			if err != nil || diff == "" {
				continue
			}

			found = true
			header := fmt.Sprintf("\n%s\n%s\n%s\n",
				ui.Bold("════════════════════════════════════"),
				ui.Bold("  repo: "+repoPath),
				ui.Bold("════════════════════════════════════"),
			)
			buf.WriteString(header)
			buf.WriteString(diff)
			buf.WriteString("\n")
		}

		if !found {
			fmt.Println("No changes across any repos.")
			return nil
		}

		return page(buf.String())
	},
}

// page pipes content through $PAGER or less -R.
func page(content string) error {
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less"
	}

	// split pager into command + args (e.g. "less -R")
	parts := strings.Fields(pager)
	name := parts[0]
	pagerArgs := parts[1:]

	// always pass -R so ANSI colour codes render
	if name == "less" && !contains(pagerArgs, "-R") {
		pagerArgs = append(pagerArgs, "-R")
	}

	c := exec.Command(name, pagerArgs...)
	c.Stdin = strings.NewReader(content)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
