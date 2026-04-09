package cmd

import (
	"fmt"

	"github.com/harris-ahmad/gitpull/git"
	"github.com/harris-ahmad/gitpull/ui"
	"github.com/spf13/cobra"
)

var branchCmd = &cobra.Command{
	Use:   "branch [name]",
	Short: "List, create, switch, or delete branches across all repos",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switchTo, _ := cmd.Flags().GetString("switch")
		deleteName, _ := cmd.Flags().GetString("delete")

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

		switch {
		case deleteName != "":
			return runBranchOp(repos, colWidth, "delete", deleteName)
		case switchTo != "":
			return runBranchOp(repos, colWidth, "switch", switchTo)
		case len(args) == 1:
			return runBranchOp(repos, colWidth, "create", args[0])
		default:
			return listBranches(repos, colWidth)

		}
	},
}

func listBranches(repos []Repo, colWidth int) error {
	for _, repo := range repos {
		branch, err := git.CurrentBranch(repo.Path)
		if err != nil {
			continue
		}
		fmt.Printf("  %-*s  %s\n", colWidth, repo.Name, ui.Cyan(branch))
	}
	return nil
}

func runBranchOp(repos []Repo, colWidth int, op, name string) error {
	var ok, failed int

	for _, repo := range repos {
		var err error
		switch op {
		case "create":
			err = git.CreateBranch(repo.Path, name)
		case "switch":
			err = git.SwitchBranch(repo.Path, name)
		case "delete":
			err = git.DeleteBranch(repo.Path, name)
		}

		if err != nil {
			fmt.Printf("%s  %-*s  %v\n", ui.Red("✗"), colWidth, repo.Name, err)
			failed++
			continue
		}

		fmt.Printf("%s  %-*s  %s %s\n", ui.Green("✓"), colWidth, repo.Name, op, ui.Cyan(name))
		ok++
	}

	sep := ui.Bold("────────────────────────────────────")
	fmt.Println("\n" + sep)
	fmt.Printf("  %d succeeded\n", ok)
	fmt.Printf("  %d failed\n", failed)
	fmt.Println(sep)

	return nil
}

func init() {
	branchCmd.Flags().StringP("switch", "s", "", "switch to branch across all repos")
	branchCmd.Flags().StringP("delete", "d", "", "delete branch across all repos")
	rootCmd.AddCommand(branchCmd)
}
