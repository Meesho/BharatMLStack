package indices

import (
	"encoding/binary"
)

const (
	_64_BITS_COUNT = (1 << 18) // 2^24/64 as we are using uint64
)

var bitIndex = [64]uint64{
	SET_BIT_0, SET_BIT_1, SET_BIT_2, SET_BIT_3, SET_BIT_4, SET_BIT_5, SET_BIT_6, SET_BIT_7,
	SET_BIT_8, SET_BIT_9, SET_BIT_10, SET_BIT_11, SET_BIT_12, SET_BIT_13, SET_BIT_14, SET_BIT_15,
	SET_BIT_16, SET_BIT_17, SET_BIT_18, SET_BIT_19, SET_BIT_20, SET_BIT_21, SET_BIT_22, SET_BIT_23,
	SET_BIT_24, SET_BIT_25, SET_BIT_26, SET_BIT_27, SET_BIT_28, SET_BIT_29, SET_BIT_30, SET_BIT_31,
	SET_BIT_32, SET_BIT_33, SET_BIT_34, SET_BIT_35, SET_BIT_36, SET_BIT_37, SET_BIT_38, SET_BIT_39,
	SET_BIT_40, SET_BIT_41, SET_BIT_42, SET_BIT_43, SET_BIT_44, SET_BIT_45, SET_BIT_46, SET_BIT_47,
	SET_BIT_48, SET_BIT_49, SET_BIT_50, SET_BIT_51, SET_BIT_52, SET_BIT_53, SET_BIT_54, SET_BIT_55,
	SET_BIT_56, SET_BIT_57, SET_BIT_58, SET_BIT_59, SET_BIT_60, SET_BIT_61, SET_BIT_62, SET_BIT_63,
}

type FlatBitmap struct {
	bitmap     [_64_BITS_COUNT]uint64
	valueSlice [_64_BITS_COUNT][]Entry12
}

func NewFlatBitmap() *FlatBitmap {
	return &FlatBitmap{}
}

// Entry12 is a packed 12-byte entry: [8-byte tag][4-byte idx] in little-endian.
type Entry12 [12]byte

// buildTag packs last28bits (28 bits) and h2 (up to 34 bits) into a 64-bit tag:
// tag = (last28bits & 0x0FFFFFFF) << 34 | (h2 & ((1<<34)-1))
func buildTag(last28bits, h2 uint64) uint64 {
	const mask28 = 0x0FFFFFFF
	const mask34 = (uint64(1) << 34) - 1
	return ((last28bits & mask28) << 34) | (h2 & mask34)
}

func putEntry(e *Entry12, tag uint64, idx uint32) {
	binary.LittleEndian.PutUint64(e[0:8], tag)
	binary.LittleEndian.PutUint32(e[8:12], idx)
}

func getTag(e *Entry12) uint64 {
	return binary.LittleEndian.Uint64(e[0:8])
}

func getIdx(e *Entry12) uint32 {
	return binary.LittleEndian.Uint32(e[8:12])
}

func zeroEntry(e *Entry12) {
	for i := range e {
		e[i] = 0
	}
}

// FlatBitmapStats contains aggregated statistics for a FlatBitmap instance.
type FlatBitmapStats struct {
	BucketsUsed           uint32 // number of buckets with at least one bit set
	BucketsWithOverflow   uint32 // buckets whose slice length > 64
	TotalEntries          uint64 // total present entries (tag != 0)
	PrimaryEntries        uint64 // entries present in primary region (first 64)
	OverflowEntries       uint64 // entries present in overflow region (index >= 64)
	ReusableOverflowSlots uint64 // zeroed overflow slots available for reuse

	AvgValueSliceLen float64 // average slice length among used buckets
	MaxValueSliceLen int     // maximum slice length among used buckets
	AvgOverflowLen   float64 // average overflow length among buckets that have overflow

	TotalAllocatedBytes uint64 // bytes allocated for value slices (len * 12)
}

// Stats computes aggregated statistics by scanning buckets and their slices.
// This is O(number of buckets + total slice length) and intended for diagnostics.
func (fb *FlatBitmap) Stats() FlatBitmapStats {
	var st FlatBitmapStats
	var sumLen uint64
	var sumOverflowLen uint64

	for pos := 0; pos < _64_BITS_COUNT; pos++ {
		if fb.bitmap[pos] == 0 {
			continue
		}
		st.BucketsUsed++
		sl := fb.valueSlice[pos]
		l := len(sl)
		if l == 0 {
			// Should not normally happen for a used bucket, but guard anyway
			continue
		}
		sumLen += uint64(l)
		if l > st.MaxValueSliceLen {
			st.MaxValueSliceLen = l
		}
		st.TotalAllocatedBytes += uint64(l * 12)

		// Primary region present entries
		primMax := l
		if primMax > 64 {
			primMax = 64
		}
		for i := 0; i < primMax; i++ {
			if getTag(&sl[i]) != 0 {
				st.PrimaryEntries++
				st.TotalEntries++
			}
		}

		// Overflow region stats
		if l > 64 {
			st.BucketsWithOverflow++
			overLen := l - 64
			sumOverflowLen += uint64(overLen)
			for i := 64; i < l; i++ {
				if getTag(&sl[i]) != 0 {
					st.OverflowEntries++
					st.TotalEntries++
				} else {
					st.ReusableOverflowSlots++
				}
			}
		}
	}

	if st.BucketsUsed > 0 {
		st.AvgValueSliceLen = float64(sumLen) / float64(st.BucketsUsed)
	}
	if st.BucketsWithOverflow > 0 {
		st.AvgOverflowLen = float64(sumOverflowLen) / float64(st.BucketsWithOverflow)
	}
	return st
}

func (fb *FlatBitmap) Set(next24bits, last28bits, h34 uint64, idx uint32) int {
	pos := int((next24bits >> 6) & 0x3FFFF)
	bitPos := next24bits & 0x3F
	qTag := buildTag(last28bits, h34)
	if fb.bitmap[pos] == 0 {
		fb.valueSlice[pos] = make([]Entry12, 64)
		fb.bitmap[pos] |= bitIndex[bitPos]
		putEntry(&fb.valueSlice[pos][bitPos], qTag, idx)
		return int(bitPos)
	} else if fb.bitmap[pos]&bitIndex[bitPos] == 0 {
		fb.bitmap[pos] |= bitIndex[bitPos]
		putEntry(&fb.valueSlice[pos][bitPos], qTag, idx)
		return int(bitPos)
	} else {
		// First check the initial position for existing key
		if getTag(&fb.valueSlice[pos][bitPos]) == qTag {
			putEntry(&fb.valueSlice[pos][bitPos], qTag, idx)
			return int(bitPos)
		}

		// Then check collision list starting from index 64
		i := 64
		firstZeroIdx := -1
		for i < len(fb.valueSlice[pos]) {
			if getTag(&fb.valueSlice[pos][i]) == qTag {
				putEntry(&fb.valueSlice[pos][i], qTag, idx)
				return int(i)
			} else if getTag(&fb.valueSlice[pos][i]) == 0 && firstZeroIdx == -1 {
				firstZeroIdx = i
			}
			i++
		}
		if firstZeroIdx != -1 {
			putEntry(&fb.valueSlice[pos][firstZeroIdx], qTag, idx)
			return int(firstZeroIdx)
		} else {
			fb.valueSlice[pos] = append(fb.valueSlice[pos], Entry12{})
			putEntry(&fb.valueSlice[pos][len(fb.valueSlice[pos])-1], qTag, idx)
			return int(len(fb.valueSlice[pos]) - 1)
		}
	}
}

func (fb *FlatBitmap) Get(next24bits, last28bits, h34 uint64) (uint32, int, bool) {
	pos := int((next24bits >> 6) & 0x3FFFF)
	bitPos := next24bits & 0x3F
	if fb.bitmap[pos] == 0 || fb.bitmap[pos]&bitIndex[bitPos] == 0 {
		return 0, -1, false
	}
	if fb.bitmap[pos]&bitIndex[bitPos] == bitIndex[bitPos] {
		qTag := buildTag(last28bits, h34)
		if getTag(&fb.valueSlice[pos][bitPos]) == qTag {
			return getIdx(&fb.valueSlice[pos][bitPos]), int(bitPos), true
		}
		i := 64
		for i < len(fb.valueSlice[pos]) {
			if getTag(&fb.valueSlice[pos][i]) == qTag {
				return getIdx(&fb.valueSlice[pos][i]), int(i), true
			}
			i++
		}
		return 0, -1, false
	}
	return 0, -1, false
}

func (fb *FlatBitmap) Remove(next24bits, last28bits, h34 uint64) (uint32, bool) {
	pos := int((next24bits >> 6) & 0x3FFFF)
	bitPos := next24bits & 0x3F
	if fb.bitmap[pos] == 0 || fb.bitmap[pos]&bitIndex[bitPos] == 0 {
		return 0, false
	}
	if fb.bitmap[pos]&bitIndex[bitPos] == bitIndex[bitPos] {
		qTag := buildTag(last28bits, h34)
		if getTag(&fb.valueSlice[pos][bitPos]) == qTag {
			idx := getIdx(&fb.valueSlice[pos][bitPos])
			zeroEntry(&fb.valueSlice[pos][bitPos])
			return idx, true
		}
		i := 64
		for i < len(fb.valueSlice[pos]) {
			if getTag(&fb.valueSlice[pos][i]) == qTag {
				idx := getIdx(&fb.valueSlice[pos][i])
				zeroEntry(&fb.valueSlice[pos][i])
				return idx, true
			}
			i++
		}
	}
	return 0, false
}

func (fb *FlatBitmap) RemoveV2(next24bits, slicePos int) (uint32, bool) {
	pos := int((next24bits >> 6) & 0x3FFFF)
	bitPos := next24bits & 0x3F
	if fb.bitmap[pos] == 0 || fb.bitmap[pos]&bitIndex[bitPos] == 0 {
		return 0, false
	}
	if fb.bitmap[pos]&bitIndex[bitPos] == bitIndex[bitPos] {
		rbIdx := getIdx(&fb.valueSlice[pos][slicePos])
		zeroEntry(&fb.valueSlice[pos][slicePos])
		return uint32(rbIdx), true
	}
	return 0, false
}
