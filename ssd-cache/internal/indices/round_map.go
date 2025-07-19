package indices

import (
	"github.com/cespare/xxhash/v2"
)

const (
	_LO_28BIT_IN_32BIT = (1 << 28) - 1
	_LO_20BIT_IN_32BIT = (1 << 20) - 1
	_LO_12BIT_IN_32BIT = (1 << 12) - 1
	_LO_24BIT_IN_32BIT = (1 << 24) - 1
	_LO_28BIT_IN_64BIT = (1 << 28) - 1
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

type RoundMap struct {
	bitmaps []*HBM24L4
}

func NewRoundMap(numRounds int) *RoundMap {
	bitmaps := make([]*HBM24L4, numRounds)
	for i := 0; i < numRounds; i++ {
		bitmaps[i] = NewHBM24L4WithMatcher(true, Fingureprint28Matcher, CreateMeta, UpdateLastAccess, IncrementFreq, ExtractMeta)
	}
	return &RoundMap{
		bitmaps: bitmaps,
	}
}

func (rm *RoundMap) Add(key string, memTableId uint32, offset uint32, length uint16, freq uint32, ttl uint32, lastAccess uint32) bool {
	hash := xxhash.Sum64String(key)

	first12bits := (hash >> 52) & _LO_12BIT_IN_32BIT // Bits 63–52
	next24bits := (hash >> 28) & _LO_24BIT_IN_32BIT  // Bits 51–28
	last28bits := hash & _LO_28BIT_IN_32BIT          // Bits 27–0

	round := first12bits % uint64(len(rm.bitmaps))
	return rm.bitmaps[round].Add(uint32(next24bits), memTableId, offset, length, uint32(last28bits), freq, ttl, lastAccess)
}

func (rm *RoundMap) Get(key string) (bool, uint32, uint32, uint16, uint32, uint32, uint32, uint32) {
	hash := xxhash.Sum64String(key)

	first12bits := (hash >> 52) & _LO_12BIT_IN_32BIT // Bits 63–52
	next24bits := (hash >> 28) & _LO_24BIT_IN_32BIT  // Bits 51–28
	last28bits := hash & _LO_28BIT_IN_32BIT          // Bits 27–0

	round := first12bits % uint64(len(rm.bitmaps))
	found, memTableId, offset, length, fingerprint28, freq, ttl, lastAccess := rm.bitmaps[round].GetMeta(uint32(next24bits))
	if !found || uint32(last28bits) != fingerprint28 {
		return false, 0, 0, 0, 0, 0, 0, 0
	}
	return true, memTableId, offset, length, fingerprint28, freq, ttl, lastAccess
}
