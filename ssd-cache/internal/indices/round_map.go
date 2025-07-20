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
)

type TestRoundMap struct {
	bitmap *HBM24L4
}

func NewTestRoundMap() *TestRoundMap {
	return &TestRoundMap{
		bitmap: NewHBM24L4WithMatcher(true, Fingureprint28Matcher, CreateMeta, UpdateLastAccess, IncrementFreq, ExtractMeta),
	}
}

func CreateMeta(memTableId uint32, offset uint32, length uint16, fingerprint28 uint32, freq uint32, ttl uint32, lastAccess uint32) (uint64, uint64, uint64) {
	return uint64(memTableId)<<32 | uint64(offset), uint64(length)<<48 | uint64(fingerprint28&_LO_28BIT_IN_32BIT)<<20 | uint64(freq&_LO_20BIT_IN_32BIT), uint64(ttl)<<32 | uint64(lastAccess)
}

func UpdateLastAccess(lastAccess uint32, meta3 uint64) uint64 {
	return (meta3 & 0xFFFFFFFF00000000) | uint64(lastAccess)
}

func IncrementFreq(freq uint32) uint32 {
	return freq //maths.IncrementLogFreqQ4_16(freq)
}

func ExtractMeta(meta1 uint64, meta2 uint64, meta3 uint64) (uint32, uint32, uint16, uint32, uint32, uint32, uint32) {
	return uint32(meta1 >> 32), uint32(meta1), uint16(meta2 >> 48), uint32((meta2 >> 20) & _LO_28BIT_IN_32BIT), uint32(meta2 & _LO_20BIT_IN_32BIT), uint32(meta3 >> 32), uint32(meta3)
}

func Fingureprint28Matcher(srcMeta2 uint64, targetMeta2 uint64) bool {
	return (srcMeta2 >> 20 & _LO_28BIT_IN_32BIT) == (targetMeta2 >> 20 & _LO_28BIT_IN_32BIT)
}

func Hash10(data string) uint64 {
	return uint64(xxh3.HashString(data) & 0x3FF) // mask 10 bits
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

func (rm *RoundMap) Add(key string, idx uint32) bool {
	hash := xxhash.Sum64String(key)

	first12bits := (hash >> 52) & _LO_12BIT_IN_32BIT // Bits 63–52
	next24bits := (hash >> 28) & _LO_24BIT_IN_32BIT  // Bits 51–28
	last28bits := hash & _LO_28BIT_IN_32BIT          // Bits 27–0
	h2 := Hash10(key)

	round := first12bits % uint64(len(rm.bitmaps))
	rm.bitmaps[round].SetV2(uint64(next24bits), uint64(last28bits), h2, idx)
	return true
}

func (rm *RoundMap) AddV2(key string, idx uint32, h64, h10 uint64) bool {
	first12bits := (h64 >> 52) & _LO_12BIT_IN_32BIT // Bits 63–52
	next24bits := (h64 >> 28) & _LO_24BIT_IN_32BIT  // Bits 51–28
	last28bits := h64 & _LO_28BIT_IN_32BIT          // Bits 27–0

	round := first12bits % uint64(len(rm.bitmaps))
	rm.bitmaps[round].SetV2(uint64(next24bits), uint64(last28bits), h10, idx)
	return true
}

func (rm *RoundMap) Get(key string) (uint32, bool) {
	hash := xxhash.Sum64String(key)

	first12bits := (hash >> 52) & _LO_12BIT_IN_32BIT // Bits 63–52
	next24bits := (hash >> 28) & _LO_24BIT_IN_32BIT  // Bits 51–28
	last28bits := hash & _LO_28BIT_IN_32BIT          // Bits 27–0
	h2 := Hash10(key)

	round := first12bits % uint64(len(rm.bitmaps))
	return rm.bitmaps[round].GetV2(uint64(next24bits), uint64(last28bits), h2)
}

func (rm *RoundMap) GetV2(h64, h10 uint64) (uint32, bool) {

	first12bits := (h64 >> 52) & _LO_12BIT_IN_32BIT // Bits 63–52
	next24bits := (h64 >> 28) & _LO_24BIT_IN_32BIT  // Bits 51–28
	last28bits := h64 & _LO_28BIT_IN_32BIT          // Bits 27–0

	round := first12bits % uint64(len(rm.bitmaps))
	return rm.bitmaps[round].GetV2(uint64(next24bits), uint64(last28bits), h10)
}
