package main

import (
	"math/rand"
	"os"

	_ "net/http/pprof"
)

// normalDistInt returns an integer in [0, max) following a normal distribution
// centered at max/2 with standard deviation = max/6 (so ~99.7% values are in range)
func normalDistInt(max int) int {
	if max <= 0 {
		return 0
	}

	mean := float64(max) / 2.0
	stdDev := float64(max) / 8.0

	for {
		val := rand.NormFloat64()*stdDev + mean

		if val >= 0 && val < float64(max) {
			return int(val)
		}
	}
}

func main() {
	// Flags to parameterize load tests
	//pick plan from the environment variable
	plan := os.Getenv("PLAN")
	if plan == "freecache" {
		planFreecache()
	} else if plan == "readthrough" {
		planReadthroughGaussian()
	} else if plan == "random" {
		planRandomGaussian()
	} else if plan == "readthrough-batched" {
		planReadthroughGaussianBatched()
	} else {
		panic("invalid plan")
	}
}

// func BucketsByWidth(a float64, n int) []float64 {
// 	if n <= 0 {
// 		return []float64{0}
// 	}
// 	b := make([]float64, n+1)
// 	b[0] = 0
// 	if math.Abs(a) < 1e-12 {
// 		// a ~ 0 => uniform
// 		for i := 1; i <= n; i++ {
// 			b[i] = float64(i) / float64(n)
// 		}
// 		return b
// 	}
// 	s := math.Expm1(a) / float64(n) // (e^a - 1)/n (stable)
// 	ia := 1.0 / a
// 	for i := 0; i <= n; i++ {
// 		b[i] = ia * math.Log1p(s*float64(i)) // ln(1 + s*i)
// 	}
// 	return b
// }
