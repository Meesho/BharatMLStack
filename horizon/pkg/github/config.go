package github

import (
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

// GitHubConfig holds all GitHub-related configuration
type GitHubConfig struct {
	// Authentication
	AppID          int64
	InstallationID int64
	PrivateKeyPath string

	// Organization/Repository
	Owner string
	// Note: HelmChartRepo, InfraHelmChartRepo, and ArgoRepo removed - not used
	// All repository operations use REPOSITORY_NAME environment variable

	// Commit Information
	CommitAuthor string
	CommitEmail  string

	// Metrics (for GPU triggers)
	VictoriaMetricsServerAddress string

	// Branch Configuration (per environment)
	BranchConfig map[string]string // Maps environment names to branch names
}

// InitGitHub initializes all GitHub-related functionality with a single configuration
// This consolidates client initialization, configuration, and repository setup
// This is called internally by externalcall.InitGitHubClient
func InitGitHub(config GitHubConfig) error {
	// Initialize GitHub configuration (owner, commit author, email)
	// These can be set even if client credentials are not provided
	InitGitHubConfig(config.Owner, config.CommitAuthor, config.CommitEmail)

	// Note: InitGitHubRepos removed - repository names now come from REPOSITORY_NAME env var
	// Note: InitGitHubBranches removed - branch configuration now comes from BRANCH_NAME env var

	// Initialize VictoriaMetrics server address for GPU metrics
	if config.VictoriaMetricsServerAddress != "" {
		SetVictoriaMetricsServerAddress(config.VictoriaMetricsServerAddress)
	}

	// Initialize GitHub client if credentials are provided
	if config.AppID > 0 && config.InstallationID > 0 && config.PrivateKeyPath != "" {
		if err := InitGitHubClient(config.AppID, config.InstallationID, config.PrivateKeyPath); err != nil {
			log.Warn().Err(err).Msg("Failed to initialize GitHub client - threshold updates will not work")
			return err
		}
		log.Info().Msg("GitHub client and configuration initialized successfully")
	} else {
		log.Info().Msg("GitHub configuration initialized (client credentials not provided - threshold updates will not work)")
	}

	return nil
}

var (
	// Repository and branch for all commits (mandatory, set via InitRepositoryAndBranch)
	// All files (onboarding and threshold updates) will be pushed to this repository and branch
	repositoryName     string = "" // Set via InitRepositoryAndBranch from REPOSITORY_NAME env var
	branchName         string = "" // Set via InitRepositoryAndBranch from BRANCH_NAME env var
	repositoryInitOnce sync.Once
)

// InitRepositoryAndBranch initializes the repository and branch names for all GitHub commits
// This should be called during application startup with values from REPOSITORY_NAME and BRANCH_NAME env vars
// All commits (onboarding and threshold updates) will use these values
func InitRepositoryAndBranch(repo, branch string) {
	repositoryInitOnce.Do(func() {
		if repo != "" {
			repositoryName = repo
			log.Info().Str("repository", repo).Msg("Repository name initialized for all GitHub commits")
		}
		if branch != "" {
			branchName = branch
			log.Info().Str("branch", branch).Msg("Branch name initialized for all GitHub commits")
		}
		if repo == "" || branch == "" {
			log.Warn().Msg("Repository or branch name not provided - threshold updates may fail. Set REPOSITORY_NAME and BRANCH_NAME in .env file")
		}
	})
}

// GetRepositoryForCommits returns the configured repository name for all commits
// This ensures all commits (onboarding and threshold updates) go to the same repository
func GetRepositoryForCommits() string {
	if repositoryName == "" {
		log.Error().Msg("Repository name not initialized - call InitRepositoryAndBranch with REPOSITORY_NAME value during startup")
	}
	return repositoryName
}

// GetBranchForCommits returns the configured branch name for all commits
// This ensures all commits (onboarding and threshold updates) go to the same branch
func GetBranchForCommits() string {
	if branchName == "" {
		log.Error().Msg("Branch name not initialized - call InitRepositoryAndBranch with BRANCH_NAME value during startup")
	}
	return branchName
}

// GetEnvConfig returns environment configuration for a given working environment.
// This function is vendor-agnostic and derives all values from environment variables
// or sensible defaults. No company-specific mappings are used.
func GetEnvConfig(workingEnv string) map[string]string {
	// Extract base environment name (remove common prefixes if present)
	// This is just for deriving defaults, not for transformation
	configEnv := workingEnv
	if strings.HasPrefix(workingEnv, "gcp_") {
		configEnv = strings.TrimPrefix(workingEnv, "gcp_")
	} else if strings.HasPrefix(workingEnv, "aws_") {
		configEnv = strings.TrimPrefix(workingEnv, "aws_")
	}

	// Get branch from environment variable or use default
	branch := GetBranchForCommits()
	if branch == "" {
		branch = "main" // Sensible default
	}

	// Return minimal environment metadata derived from workingEnv
	// All other configuration should come from environment variables or config.yaml
	return map[string]string{
		"config_env":  configEnv,   // Use the environment name itself (or extracted base)
		"cluster_env": configEnv,   // Default to same as config_env
		"dns_env":     configEnv,   // Default to same as config_env
		"env_norm":    configEnv,   // Default to same as config_env
		"helm_path":   "values_v2", // Default helm path (can be overridden via env var if needed)
		"helm_branch": branch,      // Use configured branch from BRANCH_NAME env var
		"argo_branch": branch,      // Use configured branch from BRANCH_NAME env var
	}
}
