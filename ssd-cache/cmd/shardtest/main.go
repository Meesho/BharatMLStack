package main

import (
	"fmt"
	"time"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/filecache"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// import (
// 	"fmt"

// 	"github.com/Meesho/BharatMLStack/ssd-cache/internal"
// 	"github.com/rs/zerolog"
// 	"github.com/rs/zerolog/log"
// )

// func main() {
// 	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
// 	cache := internal.NewCache(1 * 1024 * 1024)
// 	for i := 0; i < 1000000; i++ {
// 		cache.Put(fmt.Sprintf("key%d", i), []byte(fmt.Sprintf("value%d", i)))
// 	}
// 	for i := 0; i < 1000000; i++ {
// 		data := cache.Get(fmt.Sprintf("key%d", i))
// 		if string(data) != fmt.Sprintf("value%d", i) {
// 			log.Error().Msgf("Error: value mismatch for key %d: %s != %s\n", i, data, fmt.Sprintf("value%d", i))
// 			break
// 		}
// 	}
// }

func main() {
	numKeys := 60000000
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	fileCacheConfig := filecache.FileCacheConfig{
		MemtableSize:        1 * 1024 * 1024,
		MaxFileSize:         1 * 1024 * 1024 * 1024,
		BlockSize:           4096,
		Directory:           "/tmp",
		Rounds:              1,
		RbInitial:           numKeys,
		RbMax:               numKeys,
		DeleteAmortizedStep: 10000,
	}
	// rmKeyNotFoundCount := 0
	// badDataCount := 0
	// roundMap := indices.NewRoundMap(fileCacheConfig.Rounds)
	// for i := 0; i < 50000000; i++ {
	// 	roundMap.AddV2(fmt.Sprintf("key%d", i), uint32(i), indices.Hash64(fmt.Sprintf("key%d", i)), indices.Hash10(fmt.Sprintf("key%d", i)))
	// }
	// for i := 0; i < 50000000; i++ {
	// 	idx, found := roundMap.GetV2(indices.Hash64(fmt.Sprintf("key%d", i)), indices.Hash10(fmt.Sprintf("key%d", i)))
	// 	if !found {
	// 		rmKeyNotFoundCount++
	// 	}
	// 	if idx != uint32(i) {
	// 		badDataCount++
	// 	}
	// }
	// log.Info().Msgf("rmKeyNotFoundCount: %d\n", rmKeyNotFoundCount)
	// log.Info().Msgf("badDataCount: %d\n", badDataCount)

	// for i := 0; i < 1000000; i++ {
	// 	roundMap.RemoveV2(indices.Hash64(fmt.Sprintf("key%d", i)), indices.Hash10(fmt.Sprintf("key%d", i)))
	// }

	// rmKeyNotFoundCount = 0
	// badDataCount = 0
	// for i := 0; i < 50000000; i++ {
	// 	idx, found := roundMap.GetV2(indices.Hash64(fmt.Sprintf("key%d", i)), indices.Hash10(fmt.Sprintf("key%d", i)))
	// 	if !found {
	// 		rmKeyNotFoundCount++
	// 	}
	// 	if idx != uint32(i) {
	// 		badDataCount++
	// 	}
	// }

	// log.Info().Msgf("rmKeyNotFoundCount: %d\n", rmKeyNotFoundCount)
	// log.Info().Msgf("badDataCount: %d\n", badDataCount)

	// rmKeyNotFoundCount = 0
	// badDataCount = 0

	// for i := 50000000; i < 51000000; i++ {
	// 	roundMap.AddV2(fmt.Sprintf("key%d", i), uint32(i), indices.Hash64(fmt.Sprintf("key%d", i)), indices.Hash10(fmt.Sprintf("key%d", i)))
	// }

	// for i := 0; i < 51000000; i++ {
	// 	idx, found := roundMap.GetV2(indices.Hash64(fmt.Sprintf("key%d", i)), indices.Hash10(fmt.Sprintf("key%d", i)))
	// 	if !found {
	// 		rmKeyNotFoundCount++
	// 	}
	// 	if idx != uint32(i) {
	// 		badDataCount++
	// 	}
	// }
	// log.Info().Msgf("rmKeyNotFoundCount: %d\n", rmKeyNotFoundCount)
	// log.Info().Msgf("badDataCount: %d\n", badDataCount)

	// rmKeyNotFoundCount = 0
	// badDataCount = 0
	// keyMap := indices.NewKeyIndex(10, 50000000, 50000000, 10)
	// for i := 0; i < 50000000; i++ {
	// 	keyMap.Put(fmt.Sprintf("key%d", i), uint16(i), uint32(i), uint32(i), uint64(time.Now().Unix()+3600))
	// }
	// for i := 0; i < 50000000; i++ {
	// 	_, _, _, _, _, _, idx, found := keyMap.GetMetaV2(fmt.Sprintf("key%d", i))
	// 	if !found {
	// 		rmKeyNotFoundCount++
	// 	}
	// 	if idx != uint32(i) {
	// 		badDataCount++
	// 	}
	// }
	// log.Info().Msgf("rmKeyNotFoundCount: %d\n", rmKeyNotFoundCount)
	// log.Info().Msgf("badDataCount: %d\n", badDataCount)

	fileCache := filecache.NewFileCache(fileCacheConfig)
	putStart := time.Now()
	for i := 0; i < numKeys; i++ {
		//_, _ = fmt.Sprintf("key%d", i), []byte(fmt.Sprintf("value%d", i))
		err := fileCache.PutV2(fmt.Sprintf("key%d", i), []byte(fmt.Sprintf("value%d", i)), uint64(time.Now().Unix()+3600))
		if err != nil {
			log.Error().Msgf("Error: %v\n", err)
			break
		}
	}
	log.Info().Msgf("time taken to put: %v\n", time.Since(putStart))
	log.Info().Msgf("Puts per second: %d\n", int(float64(numKeys)/time.Since(putStart).Seconds()))
	getStart := time.Now()
	notFoundCount := 0
	expiredCount := 0
	for i := 0; i < numKeys; i++ {
		//_ = fmt.Sprintf("key%d", i)
		value, found, expired := fileCache.Get(fmt.Sprintf("key%d", i))
		if !found {
			notFoundCount++
		}
		if expired {
			expiredCount++
		}
		if found && (string(value) != fmt.Sprintf("value%d", i)) {
			log.Error().Msgf("Error: value mismatch for key %d: %s != %s\n", i, value, fmt.Sprintf("value%d", i))
			break
		}
	}
	log.Info().Msgf("notFoundCount: %d\n", notFoundCount)
	log.Info().Msgf("expiredCount: %d\n", expiredCount)
	log.Info().Msgf("time taken to get: %v\n", time.Since(getStart))
	log.Info().Msgf("Gets per second: %d\n", int(float64(numKeys)/time.Since(getStart).Seconds()))

	// Debug: Check ring buffer state
	log.Info().Msgf("Ring buffer debug info:")
	log.Info().Msgf("  rb.nextIndex: %d", fileCache.GetRingBufferNextIndex())
	log.Info().Msgf("  rb.size: %d", fileCache.GetRingBufferSize())
	log.Info().Msgf("  rb.capacity: %d", fileCache.GetRingBufferCapacity())

}
