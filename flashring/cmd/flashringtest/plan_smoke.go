package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	cachepkg "github.com/Meesho/BharatMLStack/flashring/pkg/cache"
	"github.com/rs/zerolog/log"
)

// planSmoke is the lightweight smoke plan: it drives the same cache surface
// as the Go unit tests (Put / Get / Delete / MGet) but as a runnable binary,
// with no large preallocations and no millions of ops.
//
// Defaults are sized to complete in single-digit seconds on any machine and
// fit in any mountpoint with a few MiB free. Use this as the first thing to
// run after a refactor — if it fails, the heavy load plans are not worth
// trying.
//
//	PLAN=smoke ./flashringtest                          # default 200 keys
//	PLAN=smoke ./flashringtest -keys 500 -mount /tmp/x  # custom
func planSmoke() {
	// Smoke uses fixed, 4KB-aligned config sizes. The shared commonFlags use
	// a float-derived FileSize that's hard to keep aligned at small targets,
	// so this plan registers its own minimal flag set.
	mountDefault := os.Getenv("FLASHRING_TEST_MOUNT")
	if mountDefault == "" {
		mountDefault = os.TempDir() + "/flashring-smoke"
	}
	mountPoint := flag.String("mount", mountDefault, "data directory")
	keyCount := flag.Int("keys", 200, "number of keys to exercise")
	flag.Parse()

	// Phases 3 and 4 index keys[1]..keys[4]; reject runs that can't satisfy them.
	if *keyCount < 5 {
		log.Fatal().Int("keys", *keyCount).Msg("keys must be >= 5")
	}

	if err := os.MkdirAll(*mountPoint, 0o755); err != nil {
		log.Fatal().Err(err).Msgf("create mount %s", *mountPoint)
	}

	const (
		memtableSize = 1 << 20 // 1 MiB
		fileSize     = 4 << 20 // 4 MiB — both 4KB-aligned
	)

	cfg := cachepkg.Config{
		NumShards:             1,
		KeysPerShard:          1024,
		FileSize:              fileSize,
		MemtableSize:          memtableSize,
		ReWriteScoreThreshold: 0.5,
		GridSearchEpsilon:     0.01,
		SampleDuration:        5 * time.Second,
		FreqBands:             []int{10, 100, 1000},
		RecencyBands:          []int{1000, 100, 10},
	}

	pc, err := cachepkg.NewWrapCache(cfg, *mountPoint)
	if err != nil {
		log.Fatal().Err(err).Msg("NewWrapCache")
	}

	n := *keyCount
	keys := make([]string, n)
	values := make([][]byte, n)
	for i := 0; i < n; i++ {
		keys[i] = fmt.Sprintf("smoke-%05d", i)
		values[i] = []byte(fmt.Sprintf("payload-%05d-content", i))
	}

	start := time.Now()

	// Phase 1: Put-then-Get round-trip across all keys.
	for i, k := range keys {
		if err := pc.Put(k, values[i], 24*time.Hour); err != nil {
			log.Fatal().Err(err).Str("key", k).Msg("Put")
		}
	}
	for i, k := range keys {
		val, found, expired := pc.Get(k)
		if !found {
			log.Fatal().Str("key", k).Msg("Get: not found")
		}
		if expired {
			log.Fatal().Str("key", k).Msg("Get: unexpectedly expired")
		}
		if string(val) != string(values[i]) {
			log.Fatal().Str("key", k).Bytes("got", val).Bytes("want", values[i]).Msg("Get: value mismatch")
		}
	}

	// Phase 2: missing-key returns (nil, false, false).
	if val, found, expired := pc.Get("never-set-key"); found || expired || len(val) != 0 {
		log.Fatal().Bool("found", found).Bool("expired", expired).Int("len", len(val)).Msg("Get unknown key: unexpected non-nil/false")
	}

	// Phase 3: Delete unlinks from index; subsequent Get misses.
	// (Overwrite is intentionally not exercised here — see the matching
	// docs/tests/testcases/cache-put-get.md case which is currently skipped
	// pending a flashring contract decision.)
	deleteKey := keys[1]
	if !pc.Delete(deleteKey) {
		log.Fatal().Str("key", deleteKey).Msg("Delete returned false on present key")
	}
	if _, found, _ := pc.Get(deleteKey); found {
		log.Fatal().Str("key", deleteKey).Msg("Get after Delete: still found")
	}
	if pc.Delete(deleteKey) {
		log.Fatal().Str("key", deleteKey).Msg("second Delete: returned true on absent key")
	}

	// Phase 4: MGet batches a mix of present/absent keys, scattering results
	// back to the input positions.
	batch := []string{keys[2], keys[3], "missing-1", keys[4], "missing-2"}
	results := pc.MGet(batch)
	if len(results) != len(batch) {
		log.Fatal().Int("got", len(results)).Int("want", len(batch)).Msg("MGet: wrong result length")
	}
	present := map[string]bool{keys[2]: true, keys[3]: true, keys[4]: true}
	for i, k := range batch {
		if present[k] && !results[i].Found {
			log.Fatal().Str("key", k).Msg("MGet: present key reported missing")
		}
		if !present[k] && results[i].Found {
			log.Fatal().Str("key", k).Msg("MGet: absent key reported found")
		}
	}

	// Phase 5: close the cache before deleting shard files so we never
	// remove files out from under an open WrapCache.
	if err := pc.Close(); err != nil {
		log.Fatal().Err(err).Msg("Close")
	}
	if files, err := os.ReadDir(*mountPoint); err == nil {
		for _, file := range files {
			_ = os.Remove(filepath.Join(*mountPoint, file.Name()))
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("smoke plan PASS — %d keys exercised in %s on %s\n", n, elapsed.Round(time.Millisecond), *mountPoint)
}
