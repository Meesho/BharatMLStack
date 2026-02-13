package externalcall

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/pkg/github"
)

var (
	initGitHubOnce sync.Once
)

// InitGitHubClient initializes the GitHub client and configuration
// This follows the same pattern as other client initializations in this package
// Note: helmChartRepo, infraHelmChartRepo, argoRepo parameters removed - not used
// All repository operations use REPOSITORY_NAME environment variable
func InitGitHubClient(
	appID int64,
	installationID int64,
	privateKey []byte,
	owner string,
	commitAuthor string,
	commitEmail string,
	victoriaMetricsServerAddress string,
	branchConfig map[string]string,
) {
	initGitHubOnce.Do(func() {
		config := github.GitHubConfig{
			AppID:          appID,
			InstallationID: installationID,
			PrivateKey:     privateKey,
			Owner:          owner,
			// HelmChartRepo, InfraHelmChartRepo, ArgoRepo removed - not used
			// All repository operations use REPOSITORY_NAME environment variable
			CommitAuthor:                 commitAuthor,
			CommitEmail:                  commitEmail,
			VictoriaMetricsServerAddress: victoriaMetricsServerAddress,
			BranchConfig:                 branchConfig,
		}
		_ = github.InitGitHub(config) // Errors are logged within the GitHub package
	})
}
