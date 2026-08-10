package profiling

import (
	"fmt"

	"cloud.google.com/go/profiler"
	"github.com/spf13/viper"
)

// startContinuousProfiler boots the Google Cloud Profiler agent. APP_NAME and
// CICD_VERSION_ID are both required so captured profiles tag back to the right
// service and version. An empty GCP_PROJECT_ID lets the SDK fall back to
// GOOGLE_CLOUD_PROJECT or the VM metadata server.
func startContinuousProfiler() error {
	service := viper.GetString("APP_NAME")
	if service == "" {
		return fmt.Errorf("APP_NAME is required for continuous profiler")
	}
	version := viper.GetString("CICD_VERSION_ID")
	if version == "" {
		return fmt.Errorf("CICD_VERSION_ID is required for continuous profiler")
	}
	return profiler.Start(profiler.Config{
		Service:        service,
		ServiceVersion: version,
		ProjectID:      viper.GetString("GCP_PROJECT_ID"),
	})
}
