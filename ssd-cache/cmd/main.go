package main

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	cache := internal.NewCache(1 * 1024 * 1024)
	for i := 0; i < 1000000; i++ {
		cache.Put(fmt.Sprintf("key%d", i), []byte(fmt.Sprintf("value%d", i)))
	}
	for i := 0; i < 1000000; i++ {
		data := cache.Get(fmt.Sprintf("key%d", i))
		if string(data) != fmt.Sprintf("value%d", i) {
			log.Error().Msgf("Error: value mismatch for key %d: %s != %s\n", i, data, fmt.Sprintf("value%d", i))
			break
		}
	}
}
