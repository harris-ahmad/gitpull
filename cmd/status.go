package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/harris-ahmad/gitpull/git"
	"github.com/harris-ahmad/gitpull/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Quick dirty/clean check for all repos in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		dirs, err := os.ReadDir(".")
		if err != nil {
			return fmt.Errorf("failed to read current directory: %w", err)
		}

		colWidth := 20
		for _, dir := range dirs {
			if dir.IsDir() && len(dir.Name()) > colWidth {
				colWidth = len(dir.Name())
			}
		}
		colWidth += 2

		var clean, dirty, skipped int

		for _, dir := range dirs {
			if !dir.IsDir() {
				continue
			}

			repoPath := dir.Name()
			changes, err := git.LocalChanges(repoPath)
			if err != nil {
				fmt.Printf("%s  %-*s  not a git repo\n", ui.Bold("-"), colWidth, repoPath)
				skipped++
				continue
			}

			if len(changes) == 0 {
				fmt.Printf("%s  %-*s  clean\n", ui.Green("✓"), colWidth, repoPath)
				clean++
			} else {
				fmt.Printf("%s  %-*s  dirty (%s)\n", ui.Yellow("⚠"), colWidth, repoPath, strings.Join(changes, ", "))
				dirty++
			}
		}

		sep := ui.Bold("────────────────────────────────────")
		fmt.Println("\n" + sep)
		fmt.Printf("  %s %d clean\n", ui.Green("✓"), clean)
		fmt.Printf("  %s %d dirty\n", ui.Yellow("⚠"), dirty)
		fmt.Printf("  %s %d not git repos\n", ui.Bold("-"), skipped)
		fmt.Println(sep)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
