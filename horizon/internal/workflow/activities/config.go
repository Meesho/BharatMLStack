package activities

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/rs/zerolog/log"
)

var (
	activitiesConfig     *configs.Configs
	activitiesConfigOnce sync.Once
)

// InitActivitiesConfig initializes the activities configuration
// This should be called during application startup
func InitActivitiesConfig(config configs.Configs) {
	activitiesConfigOnce.Do(func() {
		activitiesConfig = &config
		log.Info().Msg("Activities configuration initialized")
	})
}

// GetActivitiesConfig returns the activities configuration
func GetActivitiesConfig() *configs.Configs {
	if activitiesConfig == nil {
		log.Warn().Msg("Activities config not initialized - RepositoryName and BranchName must be set via environment variables")
		return &configs.Configs{}
	}
	return activitiesConfig
}

// GetRepository returns the configured repository name (mandatory)
// Reads from REPOSITORY_NAME environment variable
// All files will be pushed to this repository
// Returns empty string if not set - caller should handle the error
func GetRepository() string {
	config := GetActivitiesConfig()
	repo := config.RepositoryName
	if repo == "" {
		log.Error().
			Str("env_var", "REPOSITORY_NAME").
			Msg("REPOSITORY_NAME is not set - this is a mandatory field. Set it in .env file")
		return ""
	}
	log.Debug().
		Str("repository", repo).
		Msg("Using configured repository name")
	return repo
}

// GetBranch returns the configured branch name (mandatory)
// Reads from BRANCH_NAME environment variable
// All files will be pushed to this branch
// Returns empty string if not set - caller should handle the error
func GetBranch() string {
	config := GetActivitiesConfig()
	branch := config.BranchName
	if branch == "" {
		log.Error().
			Str("env_var", "BRANCH_NAME").
			Msg("BRANCH_NAME is not set - this is a mandatory field. Set it in .env file")
		return ""
	}
	log.Debug().
		Str("branch", branch).
		Msg("Using configured branch name")
	return branch
}

