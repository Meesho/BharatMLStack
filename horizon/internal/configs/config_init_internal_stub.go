//go:build !meesho

package configs

// Stub for Meesho - not used in open source builds
func initUsingCacConfig(configHolder ConfigHolder) {
	panic("initUsingCacConfig is not compiled: build with -tags meesho")
}
