package main

import (
	"flag"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	cachepkg "github.com/Meesho/BharatMLStack/flashring/pkg/cache"
	"github.com/rs/zerolog/log"
)

func planBadger() {
	var flags commonFlags
	flags.register(flag.CommandLine, commonFlags{
		mountPoint:         "/mnt/disks/nvme/badger",
		numShards:          1,
		keysPerShard:       20_000_000,
		memtableMB:         16,
		fileSizeMultiplier: 1,
		readWorkers:        4,
		writeWorkers:       4,
		sampleSecs:         30,
		iterations:         100_000_000,
		logStats:           true,
		memProfile:         "mem.prof",
	})
	flag.Parse()
	teardown := setupProfiling(flags)
	defer teardown()

	cache, err := cachepkg.NewBadger(cachepkg.Config{}, flags.mountPoint)
	if err != nil {
		panic(err)
	}
	defer cache.Close()

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
		if err := cache.Put(key, val, time.Hour); err != nil {
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
					if err := cache.Put(key, val, time.Hour); err != nil {
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
					randomval := normalDistInt(totalKeys)
					key := fmt.Sprintf("key%d", randomval)
					_, found, expired := cache.Get(key)

					if !found {
						missedKeyChanList[randomval%flags.writeWorkers] <- randomval
					}
					if expired {
						panic("key expired")
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
