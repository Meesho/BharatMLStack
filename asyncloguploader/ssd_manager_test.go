package asyncloguploader

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		// DataDog statsd client spawns background goroutines via go-core/metric init
		goleak.IgnoreTopFunction("github.com/DataDog/datadog-go/v5/statsd.(*sender).sendLoop"),
		goleak.IgnoreTopFunction("github.com/DataDog/datadog-go/v5/statsd.(*aggregator).start.func1"),
		goleak.IgnoreTopFunction("github.com/DataDog/datadog-go/v5/statsd.(*Client).watch"),
		goleak.IgnoreTopFunction("github.com/DataDog/datadog-go/v5/statsd.(*telemetryClient).run.func1"),
	)
}

// helper: create fake SSD directories under a temp mount root
func setupFakeSSDs(t *testing.T, names ...string) string {
	t.Helper()
	mountRoot := t.TempDir()
	for _, name := range names {
		require.NoError(t, os.MkdirAll(filepath.Join(mountRoot, name), 0755))
	}
	return mountRoot
}

// helper: build an SSDConfig pointing at a temp mount root with fast timeouts for tests
func testSSDConfig(mountRoot string) SSDConfig {
	return SSDConfig{
		MountRoot:       mountRoot,
		ClaimTTL:        2 * time.Second,
		RenewalInterval: 100 * time.Millisecond,
		RetryInterval:   100 * time.Millisecond,
		MaxWait:         1 * time.Second,
	}
}

func TestClaimFreeSSD(t *testing.T) {
	mountRoot := setupFakeSSDs(t, "ssd0", "ssd1")
	t.Setenv("POD_UID", "uid-abc-123")

	cfg := testSSDConfig(mountRoot)
	mgr, err := NewSSDManager(cfg, nil)
	require.NoError(t, err)
	defer func() {
		mgr.CancelRenewal()
		mgr.Release()
	}()

	// Assert claim file created
	claimPath := filepath.Join(mgr.SSDPath, ".claim")
	assert.FileExists(t, claimPath)

	// Assert contents equal POD_UID
	content, err := os.ReadFile(claimPath)
	require.NoError(t, err)
	assert.Equal(t, "uid-abc-123", string(content))
}

func TestClaimSkipsFreshClaim(t *testing.T) {
	mountRoot := setupFakeSSDs(t, "ssd0", "ssd1")
	t.Setenv("POD_UID", "uid-new-pod")

	// Pre-create a fresh claim on ssd0 (current mtime)
	claimPath0 := filepath.Join(mountRoot, "ssd0", ".claim")
	require.NoError(t, os.WriteFile(claimPath0, []byte("uid-other-pod"), 0644))

	cfg := testSSDConfig(mountRoot)
	mgr, err := NewSSDManager(cfg, nil)
	require.NoError(t, err)
	defer func() {
		mgr.CancelRenewal()
		mgr.Release()
	}()

	// Should have claimed ssd1, not ssd0
	assert.Equal(t, filepath.Join(mountRoot, "ssd1"), mgr.SSDPath)

	// ssd0's claim should be untouched
	content, err := os.ReadFile(claimPath0)
	require.NoError(t, err)
	assert.Equal(t, "uid-other-pod", string(content))
}

func TestClaimTakesOverStaleClaim(t *testing.T) {
	mountRoot := setupFakeSSDs(t, "ssd0")
	t.Setenv("POD_UID", "uid-new-pod")

	// Pre-create a claim on ssd0 with stale mtime (2 minutes ago)
	claimPath := filepath.Join(mountRoot, "ssd0", ".claim")
	require.NoError(t, os.WriteFile(claimPath, []byte("uid-dead-pod"), 0644))
	staleTime := time.Now().Add(-2 * time.Minute)
	require.NoError(t, os.Chtimes(claimPath, staleTime, staleTime))

	cfg := testSSDConfig(mountRoot)
	mgr, err := NewSSDManager(cfg, nil)
	require.NoError(t, err)
	defer func() {
		mgr.CancelRenewal()
		mgr.Release()
	}()

	// Should have taken over ssd0
	assert.Equal(t, filepath.Join(mountRoot, "ssd0"), mgr.SSDPath)

	// Claim should now contain new POD_UID
	content, err := os.ReadFile(claimPath)
	require.NoError(t, err)
	assert.Equal(t, "uid-new-pod", string(content))
}

func TestClaimRaceTwoPods(t *testing.T) {
	mountRoot := setupFakeSSDs(t, "ssd0", "ssd1")

	cfg := testSSDConfig(mountRoot)

	var (
		mu       sync.Mutex
		managers []*SSDManager
		errs     []error
		wg       sync.WaitGroup
	)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(podID int) {
			defer wg.Done()
			// Each goroutine uses a unique POD_UID via the env override in the config
			// Since t.Setenv is not goroutine-safe, we test the race by having
			// both use the same POD_UID — the point is atomic claim exclusivity
			mgr, err := newSSDManagerWithPodUID(cfg, nil, "uid-race-pod")
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
			} else {
				managers = append(managers, mgr)
			}
		}(i)
	}
	wg.Wait()

	// Both should succeed (2 SSDs available)
	assert.Empty(t, errs, "no errors expected with 2 SSDs and 2 pods")
	assert.Len(t, managers, 2, "both pods should claim an SSD")

	// They should have claimed different SSDs
	if len(managers) == 2 {
		assert.NotEqual(t, managers[0].SSDPath, managers[1].SSDPath,
			"each pod should claim a different SSD")
	}

	// Cleanup
	for _, mgr := range managers {
		mgr.CancelRenewal()
		mgr.Release()
	}
}

func TestNoSSDAvailableFailsFast(t *testing.T) {
	mountRoot := setupFakeSSDs(t, "ssd0", "ssd1")
	t.Setenv("POD_UID", "uid-latecomer")

	// Pre-create fresh claims on all SSDs
	for _, name := range []string{"ssd0", "ssd1"} {
		claimPath := filepath.Join(mountRoot, name, ".claim")
		require.NoError(t, os.WriteFile(claimPath, []byte("uid-holder"), 0644))
	}

	cfg := testSSDConfig(mountRoot)
	cfg.MaxWait = 500 * time.Millisecond
	cfg.RetryInterval = 100 * time.Millisecond

	start := time.Now()
	mgr, err := NewSSDManager(cfg, nil)
	elapsed := time.Since(start)

	assert.Nil(t, mgr)
	assert.ErrorIs(t, err, ErrNoSSDAvailable)
	// Should fail within MaxWait + some tolerance
	assert.Less(t, elapsed, 2*time.Second, "should fail within reasonable time")
}

func TestReleaseDeletesClaimFile(t *testing.T) {
	mountRoot := setupFakeSSDs(t, "ssd0")
	t.Setenv("POD_UID", "uid-release-test")

	cfg := testSSDConfig(mountRoot)
	mgr, err := NewSSDManager(cfg, nil)
	require.NoError(t, err)

	claimPath := filepath.Join(mgr.SSDPath, ".claim")
	assert.FileExists(t, claimPath)

	mgr.CancelRenewal()
	err = mgr.Release()
	require.NoError(t, err)

	// Claim file should be gone
	_, statErr := os.Stat(claimPath)
	assert.True(t, os.IsNotExist(statErr), "claim file should be deleted after Release")
}

func TestRenewalUpdatesMtime(t *testing.T) {
	mountRoot := setupFakeSSDs(t, "ssd0")
	t.Setenv("POD_UID", "uid-renewal-test")

	cfg := testSSDConfig(mountRoot)
	cfg.RenewalInterval = 100 * time.Millisecond

	mgr, err := NewSSDManager(cfg, nil)
	require.NoError(t, err)
	defer func() {
		mgr.CancelRenewal()
		mgr.Release()
	}()

	claimPath := filepath.Join(mgr.SSDPath, ".claim")

	// Record initial mtime
	info1, err := os.Stat(claimPath)
	require.NoError(t, err)

	// Wait for at least one renewal cycle
	time.Sleep(300 * time.Millisecond)

	// mtime should have been updated
	info2, err := os.Stat(claimPath)
	require.NoError(t, err)

	assert.True(t, info2.ModTime().After(info1.ModTime()),
		"mtime should advance after renewal: before=%v after=%v",
		info1.ModTime(), info2.ModTime())
}

func TestPODUIDNotSetFailsFast(t *testing.T) {
	mountRoot := setupFakeSSDs(t, "ssd0")
	t.Setenv("POD_UID", "") // explicitly unset

	cfg := testSSDConfig(mountRoot)
	mgr, err := NewSSDManager(cfg, nil)

	assert.Nil(t, mgr)
	assert.ErrorIs(t, err, ErrPODUIDNotSet)
}

func TestCancelRenewalWaitsForGoroutine(t *testing.T) {
	mountRoot := setupFakeSSDs(t, "ssd0")
	t.Setenv("POD_UID", "uid-cancel-test")

	cfg := testSSDConfig(mountRoot)
	cfg.RenewalInterval = 50 * time.Millisecond

	mgr, err := NewSSDManager(cfg, nil)
	require.NoError(t, err)
	defer mgr.Release()

	claimPath := filepath.Join(mgr.SSDPath, ".claim")

	// CancelRenewal should return quickly
	start := time.Now()
	mgr.CancelRenewal()
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 1*time.Second, "CancelRenewal should return within 1 second")

	// After cancel, mtime should stop advancing
	info1, err := os.Stat(claimPath)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	info2, err := os.Stat(claimPath)
	require.NoError(t, err)

	assert.Equal(t, info1.ModTime(), info2.ModTime(),
		"mtime should not change after CancelRenewal")
}

func TestRecoverOrphanTmpFiles(t *testing.T) {
	ssdPath := t.TempDir()

	// Create orphan .log.tmp files (from a different pod)
	orphanFiles := []string{
		"event1--old-pod_2026-04-20_10-00-00.log.tmp",
		"event2--old-pod_2026-04-20_11-00-00.log.tmp",
	}
	for _, name := range orphanFiles {
		require.NoError(t, os.WriteFile(filepath.Join(ssdPath, name), []byte("data"), 0644))
	}

	// Create a .log file that should NOT be touched
	logFile := "event3--old-pod_2026-04-20_12-00-00.log"
	require.NoError(t, os.WriteFile(filepath.Join(ssdPath, logFile), []byte("uploaded"), 0644))

	// Create a .log.tmp file belonging to the current pod (should be skipped)
	hostname, _ := os.Hostname()
	currentPodTmp := "event4--" + hostname + "_2026-04-23_10-00-00.log.tmp"
	require.NoError(t, os.WriteFile(filepath.Join(ssdPath, currentPodTmp), []byte("active"), 0644))

	// Snapshot and recover
	listed := listOrphanTmpFiles(ssdPath)
	recoverOrphanTmpFiles(ssdPath, listed, nil)

	// Orphan .log.tmp files should be renamed to .log
	for _, name := range orphanFiles {
		logName := name[:len(name)-4] // strip .tmp
		assert.FileExists(t, filepath.Join(ssdPath, logName), "orphan should be renamed to .log")
		_, err := os.Stat(filepath.Join(ssdPath, name))
		assert.True(t, os.IsNotExist(err), "original .tmp should no longer exist")
	}

	// .log file should be untouched
	assert.FileExists(t, filepath.Join(ssdPath, logFile))

	// Current pod's .log.tmp should NOT be renamed
	assert.FileExists(t, filepath.Join(ssdPath, currentPodTmp),
		"current pod's .tmp file should be left alone")
}

func TestNoSSDMountPoints(t *testing.T) {
	// Mount root with no ssd* directories
	mountRoot := t.TempDir()
	t.Setenv("POD_UID", "uid-test")

	cfg := testSSDConfig(mountRoot)
	mgr, err := NewSSDManager(cfg, nil)

	assert.Nil(t, mgr)
	assert.ErrorIs(t, err, ErrNoSSDMountPoint)
}

// newSSDManagerWithPodUID is a test helper that bypasses os.Getenv for POD_UID
// to allow concurrent goroutines to use different pod UIDs.
func newSSDManagerWithPodUID(cfg SSDConfig, metricTags []string, podUID string) (*SSDManager, error) {
	if podUID == "" {
		return nil, ErrPODUIDNotSet
	}

	tags := getBaseTags(metricTags)
	deadline := time.Now().Add(cfg.MaxWait)

	for {
		ssdDirs, err := listSSDDirs(cfg.MountRoot)
		if err != nil {
			return nil, err
		}
		if len(ssdDirs) == 0 {
			return nil, ErrNoSSDMountPoint
		}

		for _, ssdDir := range ssdDirs {
			claimPath := filepath.Join(ssdDir, ".claim")

			mgr, err := tryClaimSSD(ssdDir, claimPath, podUID, cfg, tags)
			if err == nil {
				return mgr, nil
			}

			if !os.IsExist(err) {
				continue
			}

			info, statErr := os.Stat(claimPath)
			if statErr != nil {
				continue
			}
			if time.Since(info.ModTime()) < cfg.ClaimTTL {
				continue
			}

			os.Remove(claimPath)
			mgr, err = tryClaimSSD(ssdDir, claimPath, podUID, cfg, tags)
			if err == nil {
				return mgr, nil
			}
		}

		if time.Now().After(deadline) {
			return nil, ErrNoSSDAvailable
		}
		time.Sleep(cfg.RetryInterval)
	}
}
