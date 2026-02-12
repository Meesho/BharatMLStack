package github

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v53/github"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

var (
	// GitHubOwner is the GitHub organization/owner name (configurable via GITHUB_OWNER env var)
	githubOwner = "Meesho" // Default value, can be overridden via InitGitHubConfig
)

var (
	githubClientOnce   sync.Once
	githubClient       *github.Client
	githubClientErr    error
	githubCommitAuthor string = "horizon-github"     // Default commit author name
	githubCommitEmail  string = "devops@example.com" // Default commit email (should be overridden via GITHUB_COMMIT_EMAIL)
)

// InitGitHubClient initializes the GitHub client using GitHub App authentication
// This should be called during application startup with the GitHub App credentials
func InitGitHubClient(appID int64, installationID int64, privateKey []byte) error {
	githubClientOnce.Do(func() {
		log.Info().
			Int64("appID", appID).
			Int64("installationID", installationID).
			Msg("InitGitHubClient: Initializing GitHub client with App authentication")

		tr := http.DefaultTransport
		itr, err := ghinstallation.New(tr, appID, installationID, privateKey)
		if err != nil {
			log.Error().
				Err(err).
				Int64("appID", appID).
				Int64("installationID", installationID).
				Msg("InitGitHubClient: Failed to create GitHub app transport - check private key file path, permissions, and App ID/Installation ID")
			githubClientErr = err
			return
		}
		githubClient = github.NewClient(&http.Client{Transport: itr})
		log.Info().
			Int64("appID", appID).
			Int64("installationID", installationID).
			Msg("InitGitHubClient: GitHub client initialized successfully")
	})
	if githubClientErr != nil {
		log.Error().
			Err(githubClientErr).
			Msg("InitGitHubClient: Returning initialization error")
	}
	return githubClientErr
}

// InitGitHubConfig initializes GitHub configuration (owner, commit author, email)
// This should be called during application startup to configure GitHub operations
func InitGitHubConfig(owner, commitAuthor, commitEmail string) {
	if owner != "" {
		githubOwner = owner
	}
	if commitAuthor != "" {
		githubCommitAuthor = commitAuthor
	}
	if commitEmail != "" {
		githubCommitEmail = commitEmail
	}
	log.Info().
		Str("owner", githubOwner).
		Str("commitAuthor", githubCommitAuthor).
		Str("commitEmail", githubCommitEmail).
		Msg("GitHub configuration initialized")
}

// GetGitHubOwner returns the configured GitHub owner/organization name
func GetGitHubOwner() string {
	return githubOwner
}

// GetGitHubClient returns the initialized GitHub client
func GetGitHubClient() (*github.Client, error) {
	if githubClient == nil {
		err := errors.New("GitHub client not initialized. Call InitGitHubClient first")
		log.Error().
			Err(err).
			Msg("GetGitHubClient: GitHub client is nil - InitGitHubClient must be called during application startup")
		return nil, err
	}
	return githubClient, nil
}

// GetFile retrieves a file from GitHub repository
func GetFile(ctx context.Context, client *github.Client, repo string, filePath string, branch string) (*github.RepositoryContent, error) {
	log.Info().
		Str("repo", repo).
		Str("filePath", filePath).
		Str("branch", branch).
		Str("owner", githubOwner).
		Msg("GetFile: Retrieving file from GitHub repository")

	fileContent, _, resp, err := client.Repositories.GetContents(ctx, githubOwner, repo, filePath, &github.RepositoryContentGetOptions{Ref: branch})
	if err != nil || resp == nil || resp.StatusCode != 200 {
		statusCode := 0
		statusText := "unknown"
		if resp != nil {
			statusCode = resp.StatusCode
			statusText = resp.Status
		}
		log.Error().
			Err(err).
			Str("repo", repo).
			Str("filePath", filePath).
			Str("branch", branch).
			Str("owner", githubOwner).
			Int("statusCode", statusCode).
			Str("status", statusText).
			Msg("GetFile: Failed to retrieve file from GitHub - file may not exist, branch may be invalid, or repository access denied")
		return nil, fmt.Errorf("unable to find file at path: %s, status: %s, error: %w", filePath, statusText, err)
	}
	log.Info().
		Str("repo", repo).
		Str("filePath", filePath).
		Str("branch", branch).
		Int("statusCode", resp.StatusCode).
		Msg("GetFile: File retrieved successfully from GitHub")
	return fileContent, err
}

// GetLargeFile retrieves a large file from GitHub using blob API if needed
func GetLargeFile(ctx context.Context, client *github.Client, repo string, filePath string, branch string) (*github.RepositoryContent, error) {
	log.Info().Str("repo", repo).Str("filePath", filePath).Str("branch", branch).Msg("Entered GetLargeFile")

	// Get file metadata to access the SHA
	fileContent, _, resp, err := client.Repositories.GetContents(ctx, githubOwner, repo, filePath, &github.RepositoryContentGetOptions{Ref: branch})
	if err != nil || resp.StatusCode != 200 {
		log.Error().Err(err).Str("filePath", filePath).Str("status", resp.Status).Msg("Unable to get file metadata")
		return nil, fmt.Errorf("unable to get file metadata for path: %s, status: %s", filePath, resp.Status)
	}

	// If file is large or fileContent.Content is nil (can happen with large files)
	if fileContent == nil || fileContent.Content == nil || (fileContent.Size != nil && *fileContent.Size > 1024*1024) {
		log.Warn().Int("size", safeInt(fileContent.Size)).Str("sha", fileContent.GetSHA()).Msg("Using Blob API due to large size or missing content")

		blob, _, err := client.Git.GetBlob(ctx, githubOwner, repo, fileContent.GetSHA())
		if err != nil {
			log.Error().Err(err).Str("sha", fileContent.GetSHA()).Msg("Failed to get blob content")
			return nil, fmt.Errorf("failed to get blob for file: %s", filePath)
		}

		// Set blob content and encoding manually
		fileContent.Content = github.String(blob.GetContent())
		fileContent.Encoding = github.String("base64")
	}

	log.Info().Str("filePath", filePath).Msg("Exited GetLargeFile: File content retrieved successfully")
	return fileContent, nil
}

// UpdateFile updates a file in GitHub repository
func UpdateFile(ctx context.Context, client *github.Client, repo string, filePath string, fileSHA *string, commitMsg *string, content []byte, branch *string) (*github.RepositoryContentResponse, error) {
	log.Info().
		Str("repo", repo).
		Str("filePath", filePath).
		Str("branch", safeString(branch)).
		Str("owner", githubOwner).
		Str("fileSHA", safeString(fileSHA)).
		Str("commitMsg", safeString(commitMsg)).
		Int("contentSize", len(content)).
		Str("author", githubCommitAuthor).
		Str("email", githubCommitEmail).
		Msg("UpdateFile: Updating file in GitHub repository")

	date := time.Now()
	authorName := githubCommitAuthor
	authorEmail := githubCommitEmail
	author := &github.CommitAuthor{Date: &github.Timestamp{Time: date}, Name: &authorName, Email: &authorEmail}
	fileOpts := &github.RepositoryContentFileOptions{
		Message:   commitMsg,
		Content:   content,
		SHA:       fileSHA,
		Branch:    branch,
		Author:    author,
		Committer: author,
	}
	updateResponse, resp, err := client.Repositories.UpdateFile(ctx, githubOwner, repo, filePath, fileOpts)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		statusCode := 0
		statusText := "unknown"
		if resp != nil {
			statusCode = resp.StatusCode
			statusText = resp.Status
		}
		log.Error().
			Err(err).
			Str("repo", repo).
			Str("filePath", filePath).
			Str("branch", safeString(branch)).
			Str("owner", githubOwner).
			Str("fileSHA", safeString(fileSHA)).
			Int("statusCode", statusCode).
			Str("status", statusText).
			Int("contentSize", len(content)).
			Msg("UpdateFile: Failed to update file in GitHub - check file SHA, branch existence, repository permissions, and GitHub API rate limits")
		return nil, fmt.Errorf("failed to update file: %s, status: %s, error: %w", filePath, statusText, err)
	}
	commitSHA := ""
	if updateResponse != nil {
		commitSHA = updateResponse.Commit.GetSHA()
	}
	log.Info().
		Str("repo", repo).
		Str("filePath", filePath).
		Str("branch", safeString(branch)).
		Int("statusCode", resp.StatusCode).
		Str("commitSHA", commitSHA).
		Msg("UpdateFile: File updated successfully in GitHub")
	return updateResponse, err
}

// CreateNewFile creates a new file in GitHub repository
func CreateNewFile(ctx context.Context, client *github.Client, repo string, filePath string, commitMsg *string, content []byte, branch *string) (*github.RepositoryContentResponse, error) {
	log.Info().
		Str("repo", repo).
		Str("filePath", filePath).
		Str("branch", safeString(branch)).
		Str("owner", githubOwner).
		Str("commitMsg", safeString(commitMsg)).
		Int("contentSize", len(content)).
		Str("author", githubCommitAuthor).
		Str("email", githubCommitEmail).
		Msg("CreateNewFile: Creating new file in GitHub repository")

	date := time.Now()
	authorName := githubCommitAuthor
	authorEmail := githubCommitEmail
	author := &github.CommitAuthor{Date: &github.Timestamp{Time: date}, Name: &authorName, Email: &authorEmail}
	fileOpts := &github.RepositoryContentFileOptions{
		Message:   commitMsg,
		Content:   content,
		Branch:    branch,
		Author:    author,
		Committer: author,
	}
	createResponse, resp, err := client.Repositories.CreateFile(ctx, githubOwner, repo, filePath, fileOpts)
	if err != nil {
		statusCode := 0
		statusText := "unknown"
		if resp != nil {
			statusCode = resp.StatusCode
			statusText = resp.Status
		}
		log.Error().
			Err(err).
			Str("repo", repo).
			Str("filePath", filePath).
			Str("branch", safeString(branch)).
			Str("owner", githubOwner).
			Int("statusCode", statusCode).
			Str("status", statusText).
			Int("contentSize", len(content)).
			Msg("CreateNewFile: Failed to create new file in GitHub - check branch existence, repository permissions, file path validity, and GitHub API rate limits")
		return createResponse, fmt.Errorf("failed to create new file: %s, status: %s, error: %w", filePath, statusText, err)
	}
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	commitSHA := ""
	if createResponse != nil {
		commitSHA = createResponse.Commit.GetSHA()
	}
	log.Info().
		Str("repo", repo).
		Str("filePath", filePath).
		Str("branch", safeString(branch)).
		Int("statusCode", statusCode).
		Str("commitSHA", commitSHA).
		Msg("CreateNewFile: File created successfully in GitHub")
	return createResponse, err
}

// PushContentToGithub is a helper to push content to GitHub (alias for UpdateFile or CreateNewFile)
func PushContentToGithub(content []byte, repo string, branch string, filePath string) error {
	log.Info().
		Str("repo", repo).
		Str("branch", branch).
		Str("filePath", filePath).
		Int("contentSize", len(content)).
		Msg("PushContentToGithub: Starting push operation - checking if file exists")

	client, err := GetGitHubClient()
	if err != nil {
		log.Error().
			Err(err).
			Str("repo", repo).
			Str("branch", branch).
			Str("filePath", filePath).
			Msg("PushContentToGithub: Failed to get GitHub client")
		return fmt.Errorf("failed to get GitHub client: %w", err)
	}
	ctx := context.TODO()

	// Try to get existing file to check if it exists
	log.Info().
		Str("repo", repo).
		Str("branch", branch).
		Str("filePath", filePath).
		Msg("PushContentToGithub: Checking if file exists in repository")

	existingFile, _, resp, err := client.Repositories.GetContents(ctx, githubOwner, repo, filePath, &github.RepositoryContentGetOptions{Ref: branch})
	if err != nil && resp != nil && resp.StatusCode == 404 {
		// File doesn't exist, create it
		log.Info().
			Str("repo", repo).
			Str("branch", branch).
			Str("filePath", filePath).
			Msg("PushContentToGithub: File does not exist - will create new file")
		commitMsg := fmt.Sprintf("Create %s", filePath)
		_, err = CreateNewFile(ctx, client, repo, filePath, &commitMsg, content, &branch)
		if err != nil {
			log.Error().
				Err(err).
				Str("repo", repo).
				Str("branch", branch).
				Str("filePath", filePath).
				Msg("PushContentToGithub: Failed to create new file")
			return fmt.Errorf("failed to create new file: %w", err)
		}
		log.Info().
			Str("repo", repo).
			Str("branch", branch).
			Str("filePath", filePath).
			Msg("PushContentToGithub: New file created successfully")
		return nil
	} else if existingFile != nil {
		// File exists, update it
		log.Info().
			Str("repo", repo).
			Str("branch", branch).
			Str("filePath", filePath).
			Str("fileSHA", existingFile.GetSHA()).
			Msg("PushContentToGithub: File exists - will update existing file")
		commitMsg := fmt.Sprintf("Update %s", filePath)
		_, err = UpdateFile(ctx, client, repo, filePath, existingFile.SHA, &commitMsg, content, &branch)
		if err != nil {
			log.Error().
				Err(err).
				Str("repo", repo).
				Str("branch", branch).
				Str("filePath", filePath).
				Str("fileSHA", existingFile.GetSHA()).
				Msg("PushContentToGithub: Failed to update existing file")
			return fmt.Errorf("failed to update existing file: %w", err)
		}
		log.Info().
			Str("repo", repo).
			Str("branch", branch).
			Str("filePath", filePath).
			Msg("PushContentToGithub: File updated successfully")
		return nil
	} else {
		// Error checking file existence
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		log.Error().
			Err(err).
			Str("repo", repo).
			Str("branch", branch).
			Str("filePath", filePath).
			Int("statusCode", statusCode).
			Msg("PushContentToGithub: Failed to check if file exists - cannot determine if file should be created or updated")
		return fmt.Errorf("failed to check file existence: %w", err)
	}
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func safeInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// ReadYaml reads a YAML file from GitHub and returns it as a map
func ReadYaml(filePath string, repo string, branch string) (map[string]any, error) {
	log.Info().Str("filePath", filePath).Str("repo", repo).Str("branch", branch).Msg("Entered ReadYaml")
	client, err := GetGitHubClient()
	if err != nil {
		return nil, err
	}
	ctx := context.TODO()
	fileContent, err := GetFile(ctx, client, repo, filePath, branch)
	if err != nil {
		log.Error().Err(err).Msg("ReadYaml: Failed to get file content")
		return nil, err
	}
	fileValue, err := base64.StdEncoding.DecodeString(*fileContent.Content)
	if err != nil {
		log.Error().Err(err).Msg("ReadYaml: Failed to decode file content")
		return nil, err
	}

	var data map[string]any
	err = yaml.Unmarshal(fileValue, &data)
	if err != nil {
		log.Error().Err(err).Msg("ReadYaml: Failed to unmarshal YAML")
		return nil, err
	}
	log.Info().Msg("Exited ReadYaml")
	return data, nil
}
