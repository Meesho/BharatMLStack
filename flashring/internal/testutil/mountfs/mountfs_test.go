//go:build linux

package mountfs

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

// stubT is a minimal testing.TB that records the first Fatalf / Skipf call
// and halts the calling goroutine the way the real *testing.T does
// (via runtime.Goexit). Tests invoke mountfs APIs in a child goroutine and
// wait on a channel so the host test survives the abort.
type stubT struct {
	testing.TB
	fatalMsg     string
	fataled      bool
	skipMsg      string
	skipped      bool
	cleanups     []func()
	helperCalled bool
}

func (s *stubT) Helper() { s.helperCalled = true }
func (s *stubT) Fatalf(format string, args ...any) {
	s.fataled = true
	s.fatalMsg = sprintf(format, args...)
	runtime.Goexit()
}
func (s *stubT) Skipf(format string, args ...any) {
	s.skipped = true
	s.skipMsg = sprintf(format, args...)
	runtime.Goexit()
}
func (s *stubT) Cleanup(f func()) { s.cleanups = append(s.cleanups, f) }

// runInGoroutine invokes fn in a child goroutine so a Fatalf/Skipf from the
// stub's runtime.Goexit doesn't abort the host test. Returns once fn either
// returns or Goexits.
func runInGoroutine(fn func()) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	<-done
}

func sprintf(format string, args ...any) string {
	// Avoid pulling in fmt at the top of the file; use a tiny formatter so
	// the stub stays self-contained.
	var b strings.Builder
	for i := 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) {
			i++
			if len(args) > 0 {
				b.WriteString(toString(args[0]))
				args = args[1:]
			}
			continue
		}
		b.WriteByte(format[i])
	}
	return b.String()
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case error:
		return x.Error()
	default:
		return ""
	}
}

func TestNew_EnvUnset_FatalsWithGuidance(t *testing.T) {
	t.Setenv(envMount, "")

	s := &stubT{}
	runInGoroutine(func() { _ = New(s) })

	if !s.fataled {
		t.Fatal("New did not call Fatalf when FLASHRING_TEST_MOUNT was empty")
	}
	if !strings.Contains(s.fatalMsg, envMount) {
		t.Fatalf("fatal message missing env var name: %q", s.fatalMsg)
	}
	if !s.helperCalled {
		t.Fatal("New did not mark the helper via t.Helper")
	}
}

func TestNew_BadMountPath_FatalsWithMkdirError(t *testing.T) {
	// /nonexistent/path is not creatable; MkdirTemp returns ENOENT.
	t.Setenv(envMount, "/nonexistent/path/no-way-this-exists")

	s := &stubT{}
	runInGoroutine(func() { _ = New(s) })

	if !s.fataled {
		t.Fatal("New did not call Fatalf when MkdirTemp failed")
	}
	if !strings.Contains(s.fatalMsg, "mountfs.New") {
		t.Fatalf("fatal message missing context prefix: %q", s.fatalMsg)
	}
}

func TestNew_HappyPath_ReturnsCreatedDir(t *testing.T) {
	// The harness sets FLASHRING_TEST_MOUNT; use the real *testing.T so
	// Cleanup actually runs.
	root := os.Getenv(envMount)
	if root == "" {
		t.Skip("FLASHRING_TEST_MOUNT not set; harness-dependent test")
	}

	dir := New(t)
	if dir == "" {
		t.Fatal("New returned empty path")
	}
	if !strings.HasPrefix(dir, root) {
		t.Fatalf("dir %q is not under FLASHRING_TEST_MOUNT %q", dir, root)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat %s: %v", dir, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", dir)
	}
}

func TestNew_RegistersCleanup_ThatRemovesDir(t *testing.T) {
	root := os.Getenv(envMount)
	if root == "" {
		t.Skip("FLASHRING_TEST_MOUNT not set; harness-dependent test")
	}

	s := &stubT{TB: t}
	dir := New(s)

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("dir not created: %v", err)
	}
	if len(s.cleanups) != 1 {
		t.Fatalf("len(cleanups) = %d, want 1", len(s.cleanups))
	}

	// Run the cleanup and verify the directory disappears.
	s.cleanups[0]()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("cleanup did not remove dir; Stat err = %v", err)
	}
}

func TestMode_ReadsEnv(t *testing.T) {
	t.Setenv(envMode, ModeReal)
	if got := Mode(); got != ModeReal {
		t.Fatalf("Mode() = %q, want %q", got, ModeReal)
	}

	t.Setenv(envMode, "")
	if got := Mode(); got != "" {
		t.Fatalf("Mode() with empty env = %q, want empty", got)
	}
}

func TestRequireMode_Matches_DoesNotSkip(t *testing.T) {
	t.Setenv(envMode, ModeFast)

	s := &stubT{}
	RequireMode(s, ModeFast)

	if s.skipped {
		t.Fatalf("RequireMode skipped on matching mode: %q", s.skipMsg)
	}
	if !s.helperCalled {
		t.Fatal("RequireMode did not mark the helper via t.Helper")
	}
}

func TestRequireMode_Mismatch_SkipsWithBothModes(t *testing.T) {
	t.Setenv(envMode, ModeFast)

	s := &stubT{}
	runInGoroutine(func() { RequireMode(s, ModeReal) })

	if !s.skipped {
		t.Fatal("RequireMode did not Skipf on mismatched mode")
	}
	if !strings.Contains(s.skipMsg, ModeReal) {
		t.Fatalf("skip message missing wanted mode %q: %q", ModeReal, s.skipMsg)
	}
	if !strings.Contains(s.skipMsg, ModeFast) {
		t.Fatalf("skip message missing current mode %q: %q", ModeFast, s.skipMsg)
	}
}

func TestModeConstants_AreStable(t *testing.T) {
	// Lock the two public constants — production callers compare against
	// these string values, so changing them is a breaking change.
	if ModeFast != "fast" {
		t.Fatalf("ModeFast = %q, want \"fast\"", ModeFast)
	}
	if ModeReal != "real" {
		t.Fatalf("ModeReal = %q, want \"real\"", ModeReal)
	}
}
