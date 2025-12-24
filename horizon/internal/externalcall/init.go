package externalcall

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/pkg/github"
)

func Init(config configs.Configs) {
	// Temporarily commented out - uncomment when configs are available
	// InitPrometheusClient(config.VmselectStartDaysAgo, config.VmselectBaseUrl, config.VmselectApiKey)
	// InitSlackClient(config.SlackWebhookUrl, config.SlackChannel, config.SlackCcTags, config.DefaultModelPath)
	// InitRingmasterClient(config.RingmasterBaseUrl, config.RingmasterMiscSession, config.RingmasterAuthorization, config.RingmasterEnvironment, config.RingmasterApiKey)

	// Note: WorkingEnv is now passed via query string in API requests (not initialized from config)
	// This matches RingMaster's implementation where workingEnv comes from query parameters

	// Build branch configuration map from individual environment variables
	branchConfig := buildBranchConfig(config)

	// Initialize GitHub client - follows same pattern as other clients
	// Note: HelmChartRepo, InfraHelmChartRepo, ArgoRepo removed - not used
	// All repository operations use REPOSITORY_NAME environment variable
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

	// Temporarily commented out - uncomment when configs are available
	// Initialize feature validation client with local online feature store
	// InitFeatureValidationClient()
	// Initialize pricing client - provides both raw data types and RTP format
	// PricingClient.InitPricingClient()
}
