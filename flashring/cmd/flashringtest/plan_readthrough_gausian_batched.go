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

func planReadthroughGaussianBatched() {
	var flags commonFlags
	flags.register(flag.CommandLine, commonFlags{
		mountPoint:         "/mnt/disks/nvme/",
		numShards:          200,
		keysPerShard:       10_00_00,
		memtableMB:         16,
		fileSizeMultiplier: 10,
		readWorkers:        8,
		writeWorkers:       8,
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
	totalKeys := flags.keysPerShard * flags.numShards
	str1kb := "%d" + strings.Repeat("a", 1024)

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
		if err := pc.Put(key, val, 60*time.Minute); err != nil {
			panic(err)
		}
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
					if err := pc.Put(key, val, 60*time.Minute); err != nil {
						panic(err)
					}
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
					val, found, expired := pc.Get(key)

					if !found {
						missedKeyChanList[randomval%flags.writeWorkers] <- randomval
					}
					if expired {
						panic("key expired")
					}
					if found && string(val) != fmt.Sprintf(str1kb, randomval) {
						panic("value mismatch")
					}
					if k%5_000_000 == 0 {
						fmt.Printf("----------------------------------------------read %d keys %d readerid\n", k, workerID)
					}
				}
			}(r)
		}
	}

	wg.Wait()
	log.Info().Msg("done")
}
