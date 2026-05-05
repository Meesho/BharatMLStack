package externalcall

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
)

// buildBranchConfig builds a map of environment names to branch names from config
// Only includes environments where a branch is explicitly configured
func buildBranchConfig(config configs.Configs) map[string]string {
	branchConfig := make(map[string]string)

	// Define environment to config field mappings
	envBranchMappings := []struct {
		env    string
		branch string
	}{
		{"prd", config.GitHubBranchPrd},
		{"gcp_prd", config.GitHubBranchGcpPrd},
		{"int", config.GitHubBranchInt},
		{"gcp_int", config.GitHubBranchGcpInt},
		{"dev", config.GitHubBranchDev},
		{"gcp_dev", config.GitHubBranchGcpDev},
		{"gcp_stg", config.GitHubBranchGcpStg},
		{"ftr", config.GitHubBranchFtr},
		{"gcp_ftr", config.GitHubBranchGcpFtr},
	}

	// Only add non-empty branch configurations
	for _, mapping := range envBranchMappings {
		if mapping.branch != "" {
			branchConfig[mapping.env] = mapping.branch
		}
	}

	return branchConfig
}
