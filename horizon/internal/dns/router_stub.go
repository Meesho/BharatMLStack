//go:build !meesho

package dns

// Init initializes the DNS router (stub implementation for open-source builds)
// This is a no-op for open-source builds - DNS endpoints are not available
// To enable DNS functionality, provide organization-specific implementations
func Init() {
	// DNS router is not available in open-source builds
}
