package indices

import (
	"github.com/cespare/xxhash/v2"
)

const (
	_64_BITS_COUNT = (1 << 18) // 2^24/64 as we are using uint64
)

type FlatBitmap struct {
	bitmap      [_64_BITS_COUNT]uint64
	valueSlice  [_64_BITS_COUNT][]uint64
	chainLength uint64
	count       uint64
}

func NewFlatBitmap() *FlatBitmap {
	return &FlatBitmap{}
}

func (fb *FlatBitmap) Set(key string, idx uint32) {
	hash := xxhash.Sum64String(key)
	//first12bits := (hash >> 52) & _LO_12BIT_IN_32BIT // Bits 63–52
	next24bits := (hash >> 28) & _LO_24BIT_IN_32BIT // Bits 51–28
	last28bits := hash & _LO_28BIT_IN_32BIT         // Bits 27–0
	h2 := Hash10(key)
	pos := int((next24bits >> 6) & 0x3FFFF)
	if fb.bitmap[pos] == 0 {
		fb.valueSlice[pos] = make([]uint64, 64)
		fb.bitmap[pos] |= bitIndex[next24bits&0x3F]
		fb.valueSlice[pos][next24bits&0x3F] = uint64(last28bits<<36) | uint64(h2<<26) | uint64(idx)
	} else if fb.bitmap[pos]&bitIndex[next24bits&0x3F] == 0 {
		fb.bitmap[pos] |= bitIndex[next24bits&0x3F]
		fb.valueSlice[pos][next24bits&0x3F] = uint64(last28bits<<36) | uint64(h2<<26) | uint64(idx)
	} else {
		// First check the initial position for existing key
		bitPos := next24bits & 0x3F
		if fb.valueSlice[pos][bitPos]>>36 == last28bits && (fb.valueSlice[pos][bitPos]>>26)&0x3FF == uint64(h2) {
			fb.valueSlice[pos][bitPos] = uint64(last28bits<<36) | uint64(h2<<26) | uint64(idx)
			return
		}

		// Then check collision list starting from index 64
		i := 64
		for i < len(fb.valueSlice[pos]) {
			if fb.valueSlice[pos][i]>>36 == last28bits && (fb.valueSlice[pos][i]>>26)&0x3FF == uint64(h2) {
				fb.valueSlice[pos][i] = uint64(last28bits<<36) | uint64(h2<<26) | uint64(idx)
				return
			}
			i++
		}
		fb.valueSlice[pos] = append(fb.valueSlice[pos], uint64(last28bits<<36)|uint64(h2<<26)|uint64(idx))
	}
}

func (fb *FlatBitmap) SetV2(next24bits, last28bits, h2 uint64, idx uint32) {
	pos := int((next24bits >> 6) & 0x3FFFF)
	bitPos := next24bits & 0x3F
	if fb.bitmap[pos] == 0 {
		fb.valueSlice[pos] = make([]uint64, 64)
		fb.bitmap[pos] |= bitIndex[bitPos]
		fb.valueSlice[pos][bitPos] = last28bits<<36 | h2<<26 | uint64(idx)
	} else if fb.bitmap[pos]&bitIndex[bitPos] == 0 {
		fb.bitmap[pos] |= bitIndex[bitPos]
		fb.valueSlice[pos][bitPos] = last28bits<<36 | h2<<26 | uint64(idx)
	} else {
		// First check the initial position for existing key
		if fb.valueSlice[pos][bitPos]>>36 == last28bits && (fb.valueSlice[pos][bitPos]>>26)&0x3FF == h2 {
			fb.valueSlice[pos][bitPos] = last28bits<<36 | h2<<26 | uint64(idx)
			return
		}

		// Then check collision list starting from index 64
		i := 64
		for i < len(fb.valueSlice[pos]) {
			if fb.valueSlice[pos][i]>>36 == last28bits && (fb.valueSlice[pos][i]>>26)&0x3FF == h2 {
				fb.valueSlice[pos][i] = last28bits<<36 | h2<<26 | uint64(idx)
				return
			}
			i++
		}
		fb.valueSlice[pos] = append(fb.valueSlice[pos], last28bits<<36|h2<<26|uint64(idx))
	}
}

func (fb *FlatBitmap) Get(key string) (uint32, bool) {
	hash := xxhash.Sum64String(key)
	//first12bits := (hash >> 52) & _LO_12BIT_IN_32BIT // Bits 63–52
	next24bits := (hash >> 28) & _LO_24BIT_IN_32BIT // Bits 51–28
	last28bits := hash & _LO_28BIT_IN_32BIT         // Bits 27–0
	h2 := Hash10(key)
	pos := int((next24bits >> 6) & 0x3FFFF)
	if fb.bitmap[pos] == 0 || fb.bitmap[pos]&bitIndex[next24bits&0x3F] == 0 {
		return 0, false
	}
	if fb.bitmap[pos]&bitIndex[next24bits&0x3F] == bitIndex[next24bits&0x3F] {
		if fb.valueSlice[pos][next24bits&0x3F]>>36 == last28bits && (fb.valueSlice[pos][next24bits&0x3F]>>26)&0x3FF == uint64(h2) {
			return uint32(fb.valueSlice[pos][next24bits&0x3F] & 0x3FFFFFF), true
		}
		i := 64
		for i < len(fb.valueSlice[pos]) {
			if fb.valueSlice[pos][i]>>36 == last28bits && (fb.valueSlice[pos][i]>>26)&0x3FF == uint64(h2) {
				return uint32(fb.valueSlice[pos][i] & 0x3FFFFFF), true
			}
			i++
		}
		return 0, false
	}
	return 0, false
}

func (fb *FlatBitmap) GetV2(next24bits, last28bits, h2 uint64) (uint32, bool) {
	pos := int((next24bits >> 6) & 0x3FFFF)
	bitPos := next24bits & 0x3F
	if fb.bitmap[pos] == 0 || fb.bitmap[pos]&bitIndex[bitPos] == 0 {
		return 0, false
	}
	if fb.bitmap[pos]&bitIndex[bitPos] == bitIndex[bitPos] {
		if fb.valueSlice[pos][bitPos]>>36 == last28bits && (fb.valueSlice[pos][bitPos]>>26)&0x3FF == h2 {
			return uint32(fb.valueSlice[pos][bitPos] & 0x3FFFFFF), true
		}
		i := 64
		for i < len(fb.valueSlice[pos]) {
			if fb.valueSlice[pos][i]>>36 == last28bits && (fb.valueSlice[pos][i]>>26)&0x3FF == h2 {
				return uint32(fb.valueSlice[pos][i] & 0x3FFFFFF), true
			}
			i++
		}
		return 0, false
	}
	return 0, false
}

func (fb *FlatBitmap) Remove(next24bits, last28bits, h2 uint64) (uint32, bool) {
	pos := int((next24bits >> 6) & 0x3FFFF)
	bitPos := next24bits & 0x3F
	if fb.bitmap[pos] == 0 || fb.bitmap[pos]&bitIndex[bitPos] == 0 {
		return 0, false
	}
	if fb.bitmap[pos]&bitIndex[bitPos] == bitIndex[bitPos] {
		if fb.valueSlice[pos][bitPos]>>36 == last28bits && (fb.valueSlice[pos][bitPos]>>26)&0x3FF == h2 {
			fb.bitmap[pos] &= ^bitIndex[bitPos]
			fb.valueSlice[pos][bitPos] = 0
			return 0, true
		}
		i := 64
		for i < len(fb.valueSlice[pos]) {
			if fb.valueSlice[pos][i]>>36 == last28bits && (fb.valueSlice[pos][i]>>26)&0x3FF == h2 {
				fb.bitmap[pos] &= ^bitIndex[bitPos]
				fb.valueSlice[pos][i] = 0
				return 0, true
			}
			i++
		}
	}
	return 0, false
}
