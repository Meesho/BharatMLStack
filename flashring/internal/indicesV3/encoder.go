package indicesv2

func encode(key string, length, deltaExptime, lastAccess, freq uint16, memId, offset uint32, entry *Entry) {

	d1 := uint64(length&LENGTH_MASK) << LENGTH_SHIFT
	d1 |= uint64(deltaExptime&DELTA_EXPTIME_MASK) << DELTA_EXPTIME_SHIFT
	d1 |= uint64(lastAccess&LAST_ACCESS_MASK) << LAST_ACCESS_SHIFT
	d1 |= uint64(freq&FREQ_MASK) << FREQ_SHIFT

	ByteOrder.PutUint64(entry[:8], d1)

	d2 := uint64(memId&MEM_ID_MASK) << MEM_ID_SHIFT
	d2 |= uint64(offset&OFFSET_MASK) << OFFSET_SHIFT

	ByteOrder.PutUint64(entry[8:16], d2)
}

func encodeHashNextPrev(hhi, hlo uint64, prev, next int32, entry *HashNextPrev) {
	entry[0] = hhi
	entry[1] = hlo
	entry[2] = uint64(uint32(prev))<<32 | uint64(uint32(next))
}

func encodeUpdatePrev(prev int32, entry *HashNextPrev) {
	next := entry[2] & NEXT_MASK
	entry[2] = uint64(uint32(prev))<<32 | next
}

func encodeUpdateNext(next int32, entry *HashNextPrev) {
	prev := (entry[2] >> 32) & PREV_MASK
	entry[2] = uint64(uint32(prev))<<32 | uint64(uint32(next))
}

func decodeNext(entry *HashNextPrev) int32 {
	return int32(uint32(entry[2] & NEXT_MASK))
}

func decodePrev(entry *HashNextPrev) int32 {
	return int32(uint32(entry[2]>>32) & PREV_MASK)
}

func decodeHashLo(entry *HashNextPrev) uint64 {
	return entry[1]
}

func decode(entry *Entry) (length, deltaExptime, lastAccess, freq uint16, memId, offset uint32) {
	d1 := ByteOrder.Uint64(entry[:8])
	d2 := ByteOrder.Uint64(entry[8:16])

	length = uint16(d1>>LENGTH_SHIFT) & LENGTH_MASK
	deltaExptime = uint16(d1>>DELTA_EXPTIME_SHIFT) & DELTA_EXPTIME_MASK
	lastAccess = uint16(d1>>LAST_ACCESS_SHIFT) & LAST_ACCESS_MASK
	freq = uint16(d1>>FREQ_SHIFT) & FREQ_MASK

	memId = uint32(d2>>MEM_ID_SHIFT) & MEM_ID_MASK
	offset = uint32(d2>>OFFSET_SHIFT) & OFFSET_MASK

	return length, deltaExptime, lastAccess, freq, memId, offset
}

func decodeLastAccessNFreq(entry *Entry) (lastAccess, freq uint16) {
	d1 := ByteOrder.Uint64(entry[:8])
	lastAccess = uint16(d1>>LAST_ACCESS_SHIFT) & LAST_ACCESS_MASK
	freq = uint16(d1>>FREQ_SHIFT) & FREQ_MASK

	return lastAccess, freq
}

func encodeLastAccessNFreq(lastAccess, freq uint16, entry *Entry) {
	d1 := ByteOrder.Uint64(entry[:8])
	d1 |= uint64(lastAccess&LAST_ACCESS_MASK) << LAST_ACCESS_SHIFT
	d1 |= uint64(freq&FREQ_MASK) << FREQ_SHIFT

	ByteOrder.PutUint64(entry[:8], d1)
}

func decodeMemIdOffset(entry *Entry) (memId, offset uint32) {
	d2 := ByteOrder.Uint64(entry[8:16])
	memId = uint32(d2>>MEM_ID_SHIFT) & MEM_ID_MASK
	offset = uint32(d2>>OFFSET_SHIFT) & OFFSET_MASK
	return memId, offset
}
