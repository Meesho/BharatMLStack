package profiling

import (
	"fmt"
	"runtime/debug"

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
	// CICD_VERSION_ID is injected by CI/CD. When it is absent -- a local run, or
	// any deployment outside Meesho's pipeline -- fall back to the VCS revision
	// baked in by the toolchain rather than refusing to profile at all. This
	// mirrors offer-platform-go, whose cloud profiler derives its version the
	// same way, and is why continuous profiling works here with no config.
	version := viper.GetString("CICD_VERSION_ID")
	if version == "" {
		version = buildVersion()
	}
	return profiler.Start(profiler.Config{
		Service:        service,
		ServiceVersion: version,
		ProjectID:      viper.GetString("GCP_PROJECT_ID"),
	})
}

// buildVersion reports the short VCS revision the binary was built from, or
// "unknown" when the build carries no VCS stamp (e.g. `go run`). Mirrors
// offer-platform-go's getBuildVersion.
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && len(s.Value) >= 7 {
			return s.Value[:7]
		}
	}
	return "unknown"
}
