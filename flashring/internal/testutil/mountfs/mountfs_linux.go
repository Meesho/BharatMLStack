//go:build linux

// Package mountfs provides per-test mount points rooted on the loopback ext4
// filesystem set up by scripts/test-entrypoint.sh.
//
// Tests in flashring need a real Linux block device under their mountPoint
// (O_DIRECT + fallocate(PUNCH_HOLE) requirements). The standard t.TempDir()
// returns a path under /tmp which, inside the test container, is the
// container's overlayfs and rejects O_DIRECT. mountfs.New(t) returns a fresh
// subdirectory under FLASHRING_TEST_MOUNT (the loopback ext4 mount) instead.
package mountfs

import (
	"os"
	"testing"
)

const (
	// ModeFast indicates the loopback image is RAM-backed (/dev/shm). Default.
	ModeFast = "fast"
	// ModeReal indicates the loopback image is host-disk-backed. Used for
	// throughput benchmarks and prod-parity tests.
	ModeReal = "real"

	envMount = "FLASHRING_TEST_MOUNT"
	envMode  = "FLASHRING_TEST_FS_MODE"
)

// New returns a fresh, empty subdirectory inside the loopback mount. The
// directory is removed at test cleanup. Calls t.Fatal if the harness env
// vars are not set, which means the test is being run outside the
// docker-compose harness (e.g., directly via `go test` on a Linux host).
//
// Safe to call from parallel tests — every call returns a unique path.
func New(t testing.TB) string {
	t.Helper()
	root := os.Getenv(envMount)
	if root == "" {
		t.Fatalf("%s is not set; run tests via `docker compose up -d` then `docker compose exec flashring-test go test ./...` (see docs/local-dev.md)", envMount)
	}
	dir, err := os.MkdirTemp(root, "shard-")
	if err != nil {
		t.Fatalf("mountfs.New: MkdirTemp(%q): %v", root, err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

// Mode returns the active filesystem mode ("fast" or "real"). Empty string
// when run outside the harness.
func Mode() string {
	return os.Getenv(envMode)
}

// RequireMode skips the test unless the active mode matches. Use for tests
// that only make sense at scale (e.g., GB-scale FileSize, throughput
// benchmarks) — gating them keeps the default `fast` mode snappy while
// preserving fidelity when the container is started with
// FLASHRING_TEST_FS_MODE=real.
func RequireMode(t testing.TB, want string) {
	t.Helper()
	got := Mode()
	if got != want {
		t.Skipf("requires FLASHRING_TEST_FS_MODE=%s (got %q)", want, got)
	}
}
