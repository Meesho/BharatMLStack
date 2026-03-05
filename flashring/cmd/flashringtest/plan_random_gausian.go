package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	cachepkg "github.com/Meesho/BharatMLStack/flashring/pkg/cache"
	"github.com/rs/zerolog/log"
)

func planRandomGaussian() {
	var flags commonFlags
	flags.register(flag.CommandLine, commonFlags{
		mountPoint:         "/mnt/disks/nvme/",
		numShards:          1,
		keysPerShard:       20_000_000,
		memtableMB:         16,
		fileSizeMultiplier: 40,
		readWorkers:        1,
		writeWorkers:       1,
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

	var wg sync.WaitGroup

	if flags.writeWorkers > 0 {
		fmt.Println("----------------------------------------------writing keys")
		wg.Add(flags.writeWorkers)
		for w := 0; w < flags.writeWorkers; w++ {
			go func(workerID int) {
				defer wg.Done()
				for k := 0; k < totalKeys*multiplier; k++ {
					randomval := normalDistInt(totalKeys)
					key := fmt.Sprintf("key%d", randomval)
					val := []byte(fmt.Sprintf(str1kb, randomval))
					if err := pc.Put(key, val, 60*time.Minute); err != nil {
						panic(err)
					}
					if k%5_000_000 == 0 {
						fmt.Printf("----------------------------------------------wrote %d keys %d writerid\n", k, workerID)
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
					randomval := normalDistInt(totalKeys)
					key := fmt.Sprintf("key%d", randomval)
					val, found, expired := pc.Get(key)
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
