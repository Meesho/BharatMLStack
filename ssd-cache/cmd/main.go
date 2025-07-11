package main

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

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func punchHole(file *os.File, offset, length int64) error {
	return unix.Fallocate(int(file.Fd()),
		unix.FALLOC_FL_PUNCH_HOLE|unix.FALLOC_FL_KEEP_SIZE,
		offset, length)
}

func main() {
	filePath := "/tmp/test.txt" // Change this to your file path

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
		return
	}
	defer file.Close()

	const oneGB = 1 << 30 // 1 GiB
	start := time.Now()

	err = punchHole(file, 0, oneGB)
	if err != nil {
		fmt.Printf("Punch hole failed: %v\n", err)
		return
	}

	duration := time.Since(start)
	fmt.Printf("Successfully punched 1 GiB hole in %v\n", duration)
}
