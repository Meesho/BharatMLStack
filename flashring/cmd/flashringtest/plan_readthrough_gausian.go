package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	cachepkg "github.com/Meesho/BharatMLStack/flashring/pkg/cache"
	"github.com/rs/zerolog/log"
)

func planReadthroughGaussian() {
	var flags commonFlags
	flags.register(flag.CommandLine, commonFlags{
		mountPoint:         "/mnt/disks/nvme/",
		numShards:          50,
		keysPerShard:       6_00_000,
		memtableMB:         2,
		fileSizeMultiplier: 0.25,
		readWorkers:        16,
		writeWorkers:       16,
		sampleSecs:         30,
		iterations:         100_000_000,
		logStats:           true,
		memProfile:         "mem.prof",
	})
	flag.Parse()
	teardown := setupProfiling(flags)
	defer teardown()

	files, err := os.ReadDir(flags.mountPoint)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		os.Remove(filepath.Join(flags.mountPoint, file.Name()))
	}

	cfg := cachepkg.Config{
		NumShards:             flags.numShards,
		KeysPerShard:          flags.keysPerShard,
		FileSize:              flags.fileSizeBytes(),
		MemtableSize:          flags.memtableSizeBytes(),
		ReWriteScoreThreshold: 0.8,
		GridSearchEpsilon:     0.0001,
		SampleDuration:        time.Duration(flags.sampleSecs) * time.Second,
	}

	pc, err := cachepkg.NewWrapCache(cfg, flags.mountPoint)
	if err != nil {
		panic(err)
	}
	defer pc.Close()

	const multiplier = 300
	totalKeys := 10_000_000
	str1kb := "%d" + strings.Repeat("a", 1024)

	var metrics loadMetrics
	stopReporter := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				metrics.printStats("PERIODIC")
			case <-stopReporter:
				return
			}
		}
	}()

	missedKeyChanList := make([]chan int, flags.writeWorkers)
	for i := range missedKeyChanList {
		missedKeyChanList[i] = make(chan int)
	}

	fmt.Println("----------------------------------------------prepopulating keys")
	for k := 0; k < totalKeys; k++ {
		if rand.Intn(100) < 30 {
			continue
		}
		key := fmt.Sprintf("key%d", k)
		val := []byte(fmt.Sprintf(str1kb, k))
		start := time.Now()
		if err := pc.Put(key, val, 60*time.Minute); err != nil {
			log.Error().Err(err).Msgf("error putting key %s", key)
		}
		metrics.prepopulatePutMetrics.record(time.Since(start))
		if k%5_000_000 == 0 {
			fmt.Printf("----------------------------------------------prepopulated %d keys\n", k)
		}
	}

	var wg, writeWg sync.WaitGroup

	if flags.writeWorkers > 0 {
		fmt.Println("----------------------------------------------starting write workers")
		writeWg.Add(flags.writeWorkers)
		for w := 0; w < flags.writeWorkers; w++ {
			go func(workerID int) {
				defer writeWg.Done()
				for mk := range missedKeyChanList[workerID] {
					key := fmt.Sprintf("key%d", mk)
					val := []byte(fmt.Sprintf(str1kb, mk))
					start := time.Now()
					if err := pc.Put(key, val, 60*time.Minute); err != nil {
						log.Error().Err(err).Msgf("error putting key %s", key)
					}
					metrics.putMetrics.record(time.Since(start))
				}
			}(w)
		}
	}

	if flags.readWorkers > 0 {
		fmt.Println("----------------------------------------------reading keys")
		wg.Add(flags.readWorkers)
		for r := 0; r < flags.readWorkers; r++ {
			go func(workerID int) {
				defer wg.Done()
				for k := 0; k < totalKeys*multiplier; k++ {
					randomval := normalDistIntPartitioned(workerID, flags.readWorkers, totalKeys)
					key := fmt.Sprintf("key%d", randomval)
					start := time.Now()
					val, found, expired := pc.Get(key)
					metrics.getMetrics.record(time.Since(start))

					if !found {
						metrics.getMisses.Add(1)
						missedKeyChanList[randomval%flags.writeWorkers] <- randomval
					} else {
						metrics.getHits.Add(1)
					}
					if expired {
						metrics.getExpired.Add(1)
						log.Error().Msgf("key %s expired", key)
					}
					if found && string(val) != fmt.Sprintf(str1kb, randomval) {
						panic("value mismatch")
					}
					if k%50000 == 0 {
						fmt.Printf("----------------------------------------------read %d keys %d readerid\n", k, workerID)
					}
				}
			}(r)
		}
	}

	wg.Wait()
	close(stopReporter)
	metrics.printStats("FINAL")
	log.Info().Msg("done")
}
