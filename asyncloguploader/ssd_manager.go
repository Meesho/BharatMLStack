package asyncloguploader

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Meesho/go-core/metric"
	logger "github.com/rs/zerolog/log"
)

var (
	ErrNoSSDAvailable  = errors.New("no SSD available — all claimed or stale after max wait")
	ErrPODUIDNotSet    = errors.New("POD_UID environment variable not set")
	ErrNoSSDMountPoint = errors.New("no SSD mount points found under mount root")
)

// SSDManager handles the full SSD lifecycle: claim, renewal, and release.
// LoggerManager owns one SSDManager instance.
type SSDManager struct {
	SSDPath    string
	claimPath  string
	cancelFunc context.CancelFunc // cancels renewal goroutine
	wg         sync.WaitGroup     // tracks renewal goroutine
	metricTags []string
}

// NewSSDManager scans available SSDs under cfg.MountRoot, claims one atomically,
// and spawns a renewal goroutine. Blocks until an SSD is claimed or MaxWait is reached.
func NewSSDManager(cfg SSDConfig, metricTags []string) (*SSDManager, error) {
	podUID := os.Getenv("POD_UID")
	if podUID == "" {
		return nil, ErrPODUIDNotSet
	}

	tags := getBaseTags(metricTags)
	deadline := time.Now().Add(cfg.MaxWait)

	for {
		ssdDirs, err := listSSDDirs(cfg.MountRoot)
		if err != nil {
			return nil, fmt.Errorf("scanning mount root %s: %w", cfg.MountRoot, err)
		}
		if len(ssdDirs) == 0 {
			return nil, ErrNoSSDMountPoint
		}

		for _, ssdDir := range ssdDirs {
			claimPath := filepath.Join(ssdDir, ".claim")

			// Attempt exclusive create
			mgr, err := tryClaimSSD(ssdDir, claimPath, podUID, cfg, tags)
			if err == nil {
				metric.Incr(MetricSSDClaimSuccess, tags)
				logger.Info().Str("ssd", ssdDir).Str("pod_uid", podUID).Msg("SSD claimed successfully")
				return mgr, nil
			}

			if !errors.Is(err, os.ErrExist) {
				// Unexpected error (permissions, etc.) — log and try next SSD
				logger.Warn().Err(err).Str("ssd", ssdDir).Msg("failed to claim SSD, trying next")
				continue
			}

			// Claim file exists — check if stale
			info, statErr := os.Stat(claimPath)
			if statErr != nil {
				// Can't stat — skip this SSD
				continue
			}

			if time.Since(info.ModTime()) < cfg.ClaimTTL {
				// Fresh claim by another pod — skip
				continue
			}

			// Stale claim — remove and retry
			logger.Info().Str("ssd", ssdDir).
				Time("claim_mtime", info.ModTime()).
				Msg("removing stale claim")
			if removeErr := os.Remove(claimPath); removeErr != nil {
				logger.Warn().Err(removeErr).Str("path", claimPath).Msg("failed to remove stale claim")
				continue
			}

			// Retry exclusive create after removing stale claim
			mgr, err = tryClaimSSD(ssdDir, claimPath, podUID, cfg, tags)
			if err == nil {
				metric.Incr(MetricSSDClaimSuccess, tags)
				logger.Info().Str("ssd", ssdDir).Str("pod_uid", podUID).Msg("SSD claimed after stale takeover")
				return mgr, nil
			}
			// Another pod won the race — continue to next SSD
		}

		// No SSD claimed — check deadline
		if time.Now().After(deadline) {
			metric.Incr(MetricSSDClaimFailed, tags)
			logger.Error().
				Str("mount_root", cfg.MountRoot).
				Dur("max_wait", cfg.MaxWait).
				Msg("no SSD available after max wait")
			return nil, ErrNoSSDAvailable
		}

		logger.Info().Dur("retry_interval", cfg.RetryInterval).Msg("no SSD available, retrying")
		time.Sleep(cfg.RetryInterval)
	}
}

// tryClaimSSD attempts to exclusively create the claim file and, on success,
// writes the POD_UID and spawns the renewal goroutine.
func tryClaimSSD(ssdDir, claimPath, podUID string, cfg SSDConfig, metricTags []string) (*SSDManager, error) {
	f, err := createClaimExclusive(claimPath)
	if err != nil {
		return nil, err
	}

	// Write POD_UID as claim content
	if _, writeErr := f.WriteString(podUID); writeErr != nil {
		f.Close()
		os.Remove(claimPath)
		return nil, fmt.Errorf("writing POD_UID to claim: %w", writeErr)
	}
	f.Close()

	ctx, cancel := context.WithCancel(context.Background())
	mgr := &SSDManager{
		SSDPath:    ssdDir,
		claimPath:  claimPath,
		cancelFunc: cancel,
		metricTags: metricTags,
	}

	mgr.wg.Add(1)
	go mgr.renewalLoop(ctx, cfg.RenewalInterval)

	return mgr, nil
}

// createClaimExclusive atomically creates a claim file using O_CREATE|O_EXCL.
// Returns os.ErrExist if the file already exists.
func createClaimExclusive(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
}

// renewalLoop periodically touches the claim file's mtime to signal liveness.
func (s *SSDManager) renewalLoop(ctx context.Context, interval time.Duration) {
	defer s.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			if err := os.Chtimes(s.claimPath, now, now); err != nil {
				// Log but don't exit — single missed renewal is absorbed by 6x TTL margin
				logger.Warn().Err(err).Str("path", s.claimPath).Msg("claim renewal failed")
				metric.Incr(MetricSSDRenewalFailed, s.metricTags)
			}
		}
	}
}

// CancelRenewal cancels the renewal goroutine and waits for it to exit.
// Called by LoggerManager.Close() before drain begins.
func (s *SSDManager) CancelRenewal() {
	s.cancelFunc()
	s.wg.Wait()
}

// Release deletes the claim file. Called by LoggerManager.Close() as its
// absolute last action.
func (s *SSDManager) Release() error {
	err := os.Remove(s.claimPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("releasing SSD claim %s: %w", s.claimPath, err)
	}
	metric.Incr(MetricSSDReleased, s.metricTags)
	logger.Info().Str("ssd", s.SSDPath).Msg("SSD claim released")
	return nil
}

// listSSDDirs returns absolute paths of directories matching ssd* under mountRoot.
func listSSDDirs(mountRoot string) ([]string, error) {
	entries, err := os.ReadDir(mountRoot)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "ssd") {
			dirs = append(dirs, filepath.Join(mountRoot, e.Name()))
		}
	}
	return dirs, nil
}

// listOrphanTmpFiles returns filenames of .log.tmp files in ssdPath.
// Called synchronously before any new file writers are created to snapshot
// orphan files from the previous pod.
func listOrphanTmpFiles(ssdPath string) []string {
	entries, err := os.ReadDir(ssdPath)
	if err != nil {
		logger.Warn().Err(err).Str("ssd", ssdPath).Msg("failed to list orphan tmp files")
		return nil
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".log.tmp") {
			files = append(files, e.Name())
		}
	}
	return files
}

// recoverOrphanTmpFiles renames orphan .log.tmp files to .log in the background
// so the Uploader picks them up. Skips files belonging to the current pod.
func recoverOrphanTmpFiles(ssdPath string, orphanFiles []string, metricTags []string) {
	currentHost := getHostname()
	recovered := 0

	for _, name := range orphanFiles {
		// Skip files belonging to the current pod
		if currentHost != "" && strings.Contains(name, "--"+currentHost+"_") {
			continue
		}
		renameTmpToLog(filepath.Join(ssdPath, name), metricTags)
		recovered++
	}

	if recovered > 0 {
		metric.Count(MetricSSDOrphanTmpRecovered, int64(recovered), metricTags)
		logger.Info().Int("count", recovered).Str("ssd", ssdPath).Msg("orphan tmp files recovered")
	}
}
