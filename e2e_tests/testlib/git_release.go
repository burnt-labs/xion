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

	// Handle SSH URL format: git@github.com:owner/repo.git
	if strings.HasPrefix(remoteURL, "git@") {
		// Remove "git@github.com:" prefix and ".git" suffix
		colonIdx := strings.Index(remoteURL, ":")
		if colonIdx == -1 {
			return "", fmt.Errorf("unexpected SSH URL format: %s", remoteURL)
		}
		path := remoteURL[colonIdx+1:]
		path = strings.TrimSuffix(path, ".git")
		return path, nil
	}

	// Handle HTTPS URL format: https://github.com/owner/repo.git
	// Extract the owner and repository name from the remote URL
	parts := strings.Split(remoteURL, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected remote URL format: %s", remoteURL)
	}

	repoName := strings.TrimSuffix(parts[len(parts)-1], ".git")
	ownerName := parts[len(parts)-2]

	return fmt.Sprintf("%s/%s", ownerName, repoName), nil
}

// GHCRRepository is the base GHCR repository for xion images
const GHCRRepository = "ghcr.io/burnt-labs/xion/xion"

// GetLatestReleaseImageComponents returns the repository and version components
// for the latest released xion image from GHCR.
// Returns: [repository, version] e.g. ["ghcr.io/burnt-labs/xion/xion", "25.0.2"]
func GetLatestReleaseImageComponents() ([]string, error) {
	tag, err := GetLatestGithubReleaseCurrentRepo()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}

	// Strip the "v" prefix from the tag if present (GitHub releases use v25.0.2, GHCR uses 25.0.2)
	version := strings.TrimPrefix(tag, "v")

	return []string{GHCRRepository, version}, nil
}

// GetGHCRPackageName creates a ghcr.io package name using the latest release tag
// Deprecated: Use GetLatestReleaseImageComponents instead
func GetGHCRPackageName(repo string) (string, error) {
	tag, err := GetLatestGithubRelease(repo)
	if err != nil {
		return "", fmt.Errorf("failed to get latest release: %w", err)
	}

	// Strip the "v" prefix from the tag if present (GitHub releases use v25.0.2, GHCR uses 25.0.2)
	tag = strings.TrimPrefix(tag, "v")

	return fmt.Sprintf("ghcr.io/%s/xion:%s", repo, tag), nil
}

// GetGHCRPackageNameCurrentRepo creates a ghcr.io package name using the current repo and latest release tag
// Deprecated: Use GetLatestReleaseImageComponents instead
func GetGHCRPackageNameCurrentRepo() (string, error) {
	repo, err := GetRepoInfoFromGit()
	if err != nil {
		return "", fmt.Errorf("failed to get repo info: %w", err)
	}

	return GetGHCRPackageName(repo)
}
