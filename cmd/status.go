package cmd

import (
	"fmt"
	"strings"

	"github.com/harris-ahmad/gitpull/git"
	"github.com/harris-ahmad/gitpull/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Quick dirty/clean check for all repos in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		repos, err := repoList()
		if err != nil {
			return err
		}

		colWidth := 20
		for _, r := range repos {
			if len(r.Name) > colWidth {
				colWidth = len(r.Name)
			}
		}
		colWidth += 2

		var clean, dirty, skipped int

		for _, repo := range repos {
			changes, err := git.LocalChanges(repo.Path)
			if err != nil {
				fmt.Printf("%s  %-*s  not a git repo\n", ui.Bold("-"), colWidth, repo.Name)
				skipped++
				continue
			}

			if len(changes) == 0 {
				fmt.Printf("%s  %-*s  clean\n", ui.Green("✓"), colWidth, repo.Name)
				clean++
			} else {
				fmt.Printf("%s  %-*s  dirty (%s)\n", ui.Yellow("⚠"), colWidth, repo.Name, strings.Join(changes, ", "))
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
