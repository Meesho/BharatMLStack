//go:build amd64 && avx2
// +build amd64,avx2

package simdmap

// Link‑time swap to SIMD fast‑path.
func init() { match16 = match16_simd }
