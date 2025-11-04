package testlib

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

// GithubRelease represents the structure of the GitHub API response for releases.
type GithubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
}

// GetLatestGithubRelease fetches the latest release tag from the GitHub API.
func GetLatestGithubRelease(repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var release GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode response body: %w", err)
	}

	return release.TagName, nil
}

func GetLatestGithubReleaseCurrentRepo() (string, error) {
	repo, err := GetRepoInfoFromGit()
	if err != nil {
		return "", fmt.Errorf("failed to get repo info: %w", err)
	}

	return GetLatestGithubRelease(repo)
}

// GetRepoInfoFromGit gets the owner and repository name from the local .git directory
func GetRepoInfoFromGit() (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute git command: %w, output: %s", err, string(output))
	}

	remoteURL := strings.TrimSpace(string(output))

	// Extract the owner and repository name from the remote URL
	parts := strings.Split(remoteURL, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected remote URL format: %s", remoteURL)
	}

	repoName := strings.TrimSuffix(parts[len(parts)-1], ".git")
	ownerName := parts[len(parts)-2]

	return fmt.Sprintf("%s/%s", ownerName, repoName), nil
}

// GetGHCRPackageName creates a ghcr.io package name using the latest release tag
func GetGHCRPackageName(repo string) (string, error) {
	tag, err := GetLatestGithubRelease(repo)
	if err != nil {
		return "", fmt.Errorf("failed to get latest release: %w", err)
	}

	return fmt.Sprintf("ghcr.io/%s/heighliner:%s", repo, tag), nil
}

// GetGHCRPackageNameCurrentRepo creates a ghcr.io package name using the current repo and latest release tag
func GetGHCRPackageNameCurrentRepo() (string, error) {
	repo, err := GetRepoInfoFromGit()
	if err != nil {
		return "", fmt.Errorf("failed to get repo info: %w", err)
	}

	return GetGHCRPackageName(repo)
}
