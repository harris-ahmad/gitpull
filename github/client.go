package github

import (
	"net/http"
	"net/url"
	"fmt"

	"encoding/json"
)


type Repo struct {
	Name string `json:"name"`
	CloneURL string `json:"clone_url"`
	Fork bool `json:"fork"`
	Archived bool `json:"archived"`
	DefaultBranch string `json:"default_branch"`
}

func fetchPage(base string, page int, token string) ([]Repo, error) {
	params := url.Values{}
	params.Set("per_page", "100")
	params.Set("page", fmt.Sprintf("%d", page))

	baseURL := fmt.Sprintf("%s?%s", base, params.Encode())
	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("[GitHub API error]: rate limit reached. set GITHUB_TOKEN for higher limits")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[GitHub API error]: failed to list repos: %s", resp.Status)
	}

	var repos []Repo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}

	return repos, nil
}

func listRepos(base, token string) ([]Repo, error) {
	page := 1
	allRepos := []Repo{}

	for {
		repos, err := fetchPage(base, page, token)
		if err != nil {
			return nil, err
		}

		if len(repos) == 0 {
			break
		}

		allRepos = append(allRepos, repos...)
		page++
	}

	return allRepos, nil
}

func ListOrgRepos(org, token string) ([]Repo, error) {
	return listRepos(fmt.Sprintf("https://api.github.com/orgs/%s/repos", org), token)
}

func ListRepos(username, token string) ([]Repo, error) {
	return listRepos(fmt.Sprintf("https://api.github.com/users/%s/repos", username), token)
}