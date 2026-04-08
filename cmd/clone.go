package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/harris-ahmad/gitpull/github"
	"github.com/spf13/cobra"
)

func cloneRepo(repo github.Repo) error {
	if _, err := os.Stat(repo.Name); err == nil {
		fmt.Printf("skipped %s (already exists)\n", repo.Name)
		return nil
	}

	fmt.Printf("cloning  %s...\n", repo.Name)
	cmd := exec.Command("git", "clone", "--quiet", repo.CloneURL, repo.Name)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone %s: %w", repo.Name, err)
	}

	fmt.Printf("done     %s\n", repo.Name)
	return nil
}

var cloneCmd = &cobra.Command{
	Use:   "clone <username>",
	Short: "clone all public repos from a GitHub user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token, _ := cmd.Root().PersistentFlags().GetString("token")
		if token == "" {
			token = os.Getenv("GITHUB_TOKEN")
		}

		username := args[0]
		skipForks, _ := cmd.Flags().GetBool("skip-forks")
		skipArchived, _ := cmd.Flags().GetBool("skip-archived")

		fmt.Printf("cloning repos for %s\n", username)

		repos, err := github.ListRepos(username, token)
		if err != nil {
			return fmt.Errorf("failed to list repos: %w", err)
		}

		filteredRepos := []github.Repo{}
		for _, repo := range repos {
			if skipForks && repo.Fork {
				continue
			}
			if skipArchived && repo.Archived {
				continue
			}
			filteredRepos = append(filteredRepos, repo)
		}

		jobs := make(chan github.Repo, len(filteredRepos))
		results := make(chan error, len(filteredRepos))
		for _, repo := range filteredRepos {
			jobs <- repo
		}
		close(jobs)

		var wg sync.WaitGroup
		workers, _ := cmd.Flags().GetInt("parallel")
		for range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for repo := range jobs {
					err := cloneRepo(repo)
					if err != nil {
						results <- err
					} else {
						results <- nil
					}
				}
			}()
		}
		wg.Wait()
		close(results)

		var cloneErrors []error
		for result := range results {
			if result != nil {
				cloneErrors = append(cloneErrors, result)
			}
		}

		if len(cloneErrors) > 0 {
			for _, err := range cloneErrors {
				fmt.Printf("error: %v\n", err)
			}
			return fmt.Errorf("failed to clone some repos")
		}

		fmt.Println("all repos cloned successfully")
		return nil
	},
}

func init() {
	cloneCmd.Flags().IntP("parallel", "p", cfg.Parallel, "number of concurrent clones")
	cloneCmd.Flags().Bool("skip-forks", false, "skip forked repos")
	cloneCmd.Flags().Bool("skip-archived", false, "skip archived repos")
	rootCmd.AddCommand(cloneCmd)
}
