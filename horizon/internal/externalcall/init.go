package externalcall

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/pkg/github"
)

func Init(config configs.Configs) {
	InitPrometheusClient(config.VmselectStartDaysAgo, config.VmselectBaseUrl, config.VmselectApiKey)
	InitSlackClient(config.SlackWebhookUrl, config.SlackChannel, config.SlackCcTags, config.DefaultModelPath)
	// RingMaster client initialization removed - DNS operations now use internal configs with build tags

	branchConfig := buildBranchConfig(config)

	InitGitHubClient(
		config.GitHubAppID,
		config.GitHubInstallationID,
		config.GitHubPrivateKeyPath,
		config.GitHubOwner,
		config.GitHubCommitAuthor,
		config.GitHubCommitEmail,
		config.VictoriaMetricsServerAddress,
		branchConfig,
	)

	// Initialize repository and branch for all GitHub commits
	// This ensures all commits (onboarding and threshold updates) go to the same repository and branch
	// These values come from REPOSITORY_NAME and BRANCH_NAME environment variables (mandatory)
	github.InitRepositoryAndBranch(config.RepositoryName, config.BranchName)

	// Initialize feature validation client with local online feature store
	InitFeatureValidationClient()
	// Initialize pricing client - provides both raw data types and RTP format
	PricingClient.InitPricingClient()
}
