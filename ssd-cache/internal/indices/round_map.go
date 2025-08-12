package indices

import (
	"github.com/cespare/xxhash/v2"
	"github.com/zeebo/xxh3"
)

const (
	_LO_28BIT_IN_32BIT = (1 << 28) - 1
	_LO_20BIT_IN_32BIT = (1 << 20) - 1
	_LO_12BIT_IN_32BIT = (1 << 12) - 1
	_LO_24BIT_IN_32BIT = (1 << 24) - 1
	_LO_28BIT_IN_64BIT = (1 << 28) - 1
	_LO_6BIT_IN_32BIT  = (1 << 6) - 1
	_LO_9BIT_IN_32BIT  = (1 << 9) - 1
	_LO_3BIT_IN_32BIT  = (1 << 3) - 1
	_LO_54BIT_IN_64BIT = (1 << 54) - 1
	_LO_34BIT_IN_64BIT = (1 << 34) - 1
)

func Hash34(data string) uint64 {
	return uint64(xxh3.HashString(data) & _LO_34BIT_IN_64BIT) // mask 10 bits
}

func Hash64(data string) uint64 {
	return xxhash.Sum64String(data)
}

type RoundMap struct {
	bitmaps []*FlatBitmap
}

func NewRoundMap(numRounds int) *RoundMap {
	bitmaps := make([]*FlatBitmap, numRounds)
	for i := 0; i < numRounds; i++ {
		bitmaps[i] = NewFlatBitmap()
	}
	return &RoundMap{
		bitmaps: bitmaps,
	}
}

func (rm *RoundMap) Add(key string, idx uint32, h64, h10 uint64) (int, int, int) {
	first12bits, next24bits, last28bits := extractHashSegments(h64) // Bits 27–0

	round := first12bits % uint64(len(rm.bitmaps))
	slicePos := rm.bitmaps[round].Set(uint64(next24bits), uint64(last28bits), h10, idx)
	return int(round), int(next24bits), slicePos
}

func extractHashSegments(h64 uint64) (uint64, uint64, uint64) {
	first12bits := (h64 >> 52) & _LO_12BIT_IN_32BIT // Bits 63–52
	next24bits := (h64 >> 28) & _LO_24BIT_IN_32BIT  // Bits 51–28
	last28bits := h64 & _LO_28BIT_IN_32BIT
	return first12bits, next24bits, last28bits
}

func (rm *RoundMap) Get(h64, h10 uint64) (uint32, int, bool) {
	first12bits, next24bits, last28bits := extractHashSegments(h64) // Bits 27–0

	round := first12bits % uint64(len(rm.bitmaps))
	return rm.bitmaps[round].Get(uint64(next24bits), uint64(last28bits), h10)
}

func (rm *RoundMap) Remove(h64, h10 uint64) (uint32, bool) {

	first12bits, next24bits, last28bits := extractHashSegments(h64) // Bits 27–0

	round := first12bits % uint64(len(rm.bitmaps))
	return rm.bitmaps[round].Remove(uint64(next24bits), uint64(last28bits), h10)
}

func (rm *RoundMap) RemoveV2(round, next24bits, slicePos int) (uint32, bool) {
	return rm.bitmaps[round].RemoveV2(next24bits, slicePos)
}
