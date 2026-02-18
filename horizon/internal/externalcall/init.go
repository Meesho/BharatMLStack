package externalcall

import (
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/pkg/github"
	"github.com/spf13/viper"
)

func Init(config configs.Configs) {
	InitPrometheusClient(config.VmselectStartDaysAgo, config.VmselectBaseUrl, config.VmselectApiKey)
	InitSlackClient(config.SlackWebhookUrl, config.SlackChannel, config.SlackCcTags, config.DefaultModelPath)
	// RingMaster client initialization removed - DNS operations now use internal configs with build tags

	branchConfig := buildBranchConfig(config)
	privateKey := viper.GetString("GITHUB_PRIVATE_KEY")
	privateKeyBytes := normalizePEMKey(privateKey)
	InitGitHubClient(
		config.GitHubAppID,
		config.GitHubInstallationID,
		privateKeyBytes,
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

	// Initialize working environment from config
	// This is used by Predator handler and other components that need workingEnv
	workingEnv := viper.GetString("WORKING_ENV")
	if workingEnv != "" {
		InitWorkingEnvironment(workingEnv)
	}

	// Initialize feature validation client with local online feature store
	InitFeatureValidationClient()
	// Initialize pricing client - provides both raw data types and RTP format
	PricingClient.InitPricingClient()
	InitAirflowClient(config.AirflowBaseUrl, config.AirflowUsername, config.AirflowPassword)
	InitPrismV2Client(config.PrismBaseUrl, config.PrismAppUserID)

}

// normalizePEMKey converts env/JSON-friendly key into valid PEM (newlines required).
// Handles two common cases:
//  1. Key with literal "\n" (backslash-n) from JSON/env — replaced with real newlines.
//  2. Key pasted as a single line with spaces — spaces in the base64 body are replaced
//     with newlines while preserving spaces inside the BEGIN/END markers.
func normalizePEMKey(raw string) []byte {
	if raw == "" {
		return nil
	}

	// Step 1: replace literal escaped newlines with real newlines.
	s := strings.ReplaceAll(raw, "\\n", "\n")

	// Step 2: if the string already contains real newlines, assume valid PEM.
	if strings.Contains(s, "\n") {
		return []byte(strings.TrimSpace(s))
	}

	// Step 3: single-line key — split into header / body / footer and
	//         replace spaces only in the base64 body.
	beginIdx := strings.Index(s, "-----")
	if beginIdx < 0 {
		return []byte(s)
	}
	// Locate end of "-----BEGIN ... -----"
	rest := s[beginIdx+5:]
	endOfBegin := strings.Index(rest, "-----")
	if endOfBegin < 0 {
		return []byte(s)
	}
	headerEnd := beginIdx + 5 + endOfBegin + 5

	// Locate "-----END ... -----"
	endMarkerStart := strings.LastIndex(s, "-----END")
	if endMarkerStart < 0 || endMarkerStart <= headerEnd {
		return []byte(s)
	}

	header := s[beginIdx:headerEnd]
	footer := strings.TrimSpace(s[endMarkerStart:])
	body := strings.TrimSpace(s[headerEnd:endMarkerStart])

	// Spaces in the body separate base64-encoded lines — replace with newlines.
	body = strings.ReplaceAll(body, " ", "\n")

	return []byte(header + "\n" + body + "\n" + footer + "\n")
}
