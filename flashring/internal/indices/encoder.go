package indices

/*
-----------
uint64
-----------
round 4 bits
route 24 bits
slice pos 14 bits
exp in minutes 22 bits
---------
uint64
---------
length 16 bits
access 24 bits
freq 24 bits
-------------
uint64
-------------
memId 32 bits
offset 32 bits
*/
func encode(length uint16, memId, offset, lastAccess, freq uint32, exptime uint64, round, route, slicePos int, entry *Entry) {

	d1 := uint64(round&ROUND_MASK) << 60
	d1 |= uint64(route&ROUTE_MASK) << 36
	d1 |= uint64(slicePos&SLICE_POS_MASK) << 22
	d1 |= uint64(exptime & EXPTIME_MASK)

	d2 := uint64(length&LENGTH_MASK) << 48
	d2 |= uint64(lastAccess&LAST_ACCESS_MASK) << 24
	d2 |= uint64(freq & FREQ_MASK)

	d3 := uint64(memId&MEM_ID_MASK) << 32
	d3 |= uint64(offset & OFFSET_MASK)

	ByteOrder.PutUint64(entry[:8], d1)
	ByteOrder.PutUint64(entry[8:16], d2)
	ByteOrder.PutUint64(entry[16:24], d3)
}

func encodeD2(length uint16, lastAccess, freq uint32, entry *Entry) {
	d2 := uint64(length&LENGTH_MASK) << 48
	d2 |= uint64(lastAccess&LAST_ACCESS_MASK) << 24
	d2 |= uint64(freq & FREQ_MASK)
	ByteOrder.PutUint64(entry[8:16], d2)
}

func extract(entry *Entry) (length uint16, memId, offset, lastAccess, freq uint32, exptime uint64, round, route, slicePos int) {
	d1 := ByteOrder.Uint64(entry[:8])
	d2 := ByteOrder.Uint64(entry[8:16])
	d3 := ByteOrder.Uint64(entry[16:24])

	round = int(d1>>60) & ROUND_MASK
	route = int(d1>>36) & ROUTE_MASK
	slicePos = int(d1>>22) & SLICE_POS_MASK
	exptime = d1 & EXPTIME_MASK

	length = uint16(d2>>48) & LENGTH_MASK
	lastAccess = uint32(d2>>24) & LAST_ACCESS_MASK
	freq = uint32(d2) & FREQ_MASK

	memId = uint32(d3>>32) & MEM_ID_MASK
	offset = uint32(d3) & OFFSET_MASK
	return
}

func extractD1(entry *Entry) (round, route, slicePos int) {
	d1 := ByteOrder.Uint64(entry[:8])
	round = int(d1>>60) & ROUND_MASK
	route = int(d1>>36) & ROUTE_MASK
	slicePos = int(d1>>22) & SLICE_POS_MASK
	return
}

func extractD3(entry *Entry) (memId, offset uint32) {
	d3 := ByteOrder.Uint64(entry[16:24])
	memId = uint32(d3>>32) & MEM_ID_MASK
	offset = uint32(d3) & OFFSET_MASK
	return
}

func extractMemId(entry *Entry) (memId uint32) {
	return ByteOrder.Uint32(entry[8:12])
}
