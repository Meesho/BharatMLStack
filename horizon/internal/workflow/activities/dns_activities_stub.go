//go:build !meesho

package activities

import (
	"github.com/rs/zerolog/log"
)

// CreateCloudDNS creates Cloud DNS records (stub implementation for open-source builds)
// This is a no-op for open-source builds - DNS functionality is not available
func CreateCloudDNS(payload map[string]interface{}, workingEnv string) error {
	appName := getString(payload, "appName")
	log.Warn().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Msg("CreateCloudDNS: DNS functionality is not available in open-source builds. Provide organization-specific implementations to enable DNS operations.")
	// Return nil to allow workflow to continue (DNS is optional)
	return nil
}

// CreateCoreDNS creates Core DNS records (stub implementation for open-source builds)
// This is a no-op for open-source builds - DNS functionality is not available
func CreateCoreDNS(payload map[string]interface{}, workingEnv string) error {
	appName := getString(payload, "appName")
	log.Warn().
		Str("appName", appName).
		Str("workingEnv", workingEnv).
		Msg("CreateCoreDNS: DNS functionality is not available in open-source builds. Provide organization-specific implementations to enable DNS operations.")
	// Return nil to allow workflow to continue (DNS is optional)
	return nil
}
