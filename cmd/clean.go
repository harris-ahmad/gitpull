package cmd

import (
	"fmt"
	"os"

	"github.com/harris-ahmad/gitpull/git"
	"github.com/harris-ahmad/gitpull/ui"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove untracked files across all repos (dry-run by default, use --force to delete)",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

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

		var cleaned, skipped, empty int

		for _, dir := range dirs {
			if !dir.IsDir() {
				continue
			}
			repoPath := dir.Name()

			files, err := git.CleanDryRun(repoPath)
			if err != nil {
				continue // not a git repo
			}
			if len(files) == 0 {
				empty++
				continue
			}

			if !force {
				fmt.Printf("%s  %-*s  would remove:\n", ui.Yellow("~"), colWidth, repoPath)
				for _, f := range files {
					fmt.Printf("    %s\n", f)
				}
				continue
			}

			if err := git.Clean(repoPath); err != nil {
				fmt.Printf("%s  %-*s  failed: %v\n", ui.Red("✗"), colWidth, repoPath, err)
				skipped++
				continue
			}

			fmt.Printf("%s  %-*s  removed %d file(s)\n", ui.Green("✓"), colWidth, repoPath, len(files))
			cleaned++
		}

		sep := ui.Bold("────────────────────────────────────")
		fmt.Println("\n" + sep)
		if !force {
			fmt.Println("  dry-run — use --force to actually delete")
		} else {
			fmt.Printf("  %d repos cleaned\n", cleaned)
			fmt.Printf("  %d errors\n", skipped)
		}
		fmt.Printf("  %d repos already clean\n", empty)
		fmt.Println(sep)

		return nil
	},
}

func init() {
	cleanCmd.Flags().BoolP("force", "f", false, "actually delete untracked files (default is dry-run)")
	rootCmd.AddCommand(cleanCmd)
}
