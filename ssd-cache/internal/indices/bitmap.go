package indices

/*
	HBM24L4 is a 24-bit hash table with 4 levels of indirection. HBM stands for Hierarchical Bitmap.
*/

const (
	_24BIT_MASK = 0xFFFFFF
	_0TH_BIT    = 1 << 0
	_1ST_BIT    = 1 << 1
	_2ND_BIT    = 1 << 2
	_3RD_BIT    = 1 << 3
	_4TH_BIT    = 1 << 4
	_5TH_BIT    = 1 << 5
	_6TH_BIT    = 1 << 6
	_7TH_BIT    = 1 << 7
	_8TH_BIT    = 1 << 8
	_9TH_BIT    = 1 << 9
	_10TH_BIT   = 1 << 10
	_11TH_BIT   = 1 << 11
	_12TH_BIT   = 1 << 12
	_13TH_BIT   = 1 << 13
	_14TH_BIT   = 1 << 14
	_15TH_BIT   = 1 << 15
	_16TH_BIT   = 1 << 16
	_17TH_BIT   = 1 << 17
	_18TH_BIT   = 1 << 18
	_19TH_BIT   = 1 << 19
	_20TH_BIT   = 1 << 20
	_21ST_BIT   = 1 << 21
	_22ND_BIT   = 1 << 22
	_23RD_BIT   = 1 << 23
	_24TH_BIT   = 1 << 24
	_25TH_BIT   = 1 << 25
	_26TH_BIT   = 1 << 26
	_27TH_BIT   = 1 << 27
	_28TH_BIT   = 1 << 28
	_29TH_BIT   = 1 << 29
	_30TH_BIT   = 1 << 30
	_31ST_BIT   = 1 << 31
	_32ND_BIT   = 1 << 32
	_33RD_BIT   = 1 << 33
	_34TH_BIT   = 1 << 34
	_35TH_BIT   = 1 << 35
	_36TH_BIT   = 1 << 36
	_37TH_BIT   = 1 << 37
	_38TH_BIT   = 1 << 38
	_39TH_BIT   = 1 << 39
	_40TH_BIT   = 1 << 40
	_41ST_BIT   = 1 << 41
	_42ND_BIT   = 1 << 42
	_43RD_BIT   = 1 << 43
	_44TH_BIT   = 1 << 44
	_45TH_BIT   = 1 << 45
	_46TH_BIT   = 1 << 46
	_47TH_BIT   = 1 << 47
	_48TH_BIT   = 1 << 48
	_49TH_BIT   = 1 << 49
	_50TH_BIT   = 1 << 50
	_51ST_BIT   = 1 << 51
	_52ND_BIT   = 1 << 52
	_53RD_BIT   = 1 << 53
	_54TH_BIT   = 1 << 54
	_55TH_BIT   = 1 << 55
	_56TH_BIT   = 1 << 56
	_57TH_BIT   = 1 << 57
	_58TH_BIT   = 1 << 58
	_59TH_BIT   = 1 << 59
	_60TH_BIT   = 1 << 60
	_61ST_BIT   = 1 << 61
	_62ND_BIT   = 1 << 62
	_63TH_BIT   = 1 << 63
)

var (
	bitIndex = [64]uint64{
		_0TH_BIT, _1ST_BIT, _2ND_BIT, _3RD_BIT, _4TH_BIT, _5TH_BIT, _6TH_BIT, _7TH_BIT,
		_8TH_BIT, _9TH_BIT, _10TH_BIT, _11TH_BIT, _12TH_BIT, _13TH_BIT, _14TH_BIT, _15TH_BIT,
		_16TH_BIT, _17TH_BIT, _18TH_BIT, _19TH_BIT, _20TH_BIT, _21ST_BIT, _22ND_BIT, _23RD_BIT,
		_24TH_BIT, _25TH_BIT, _26TH_BIT, _27TH_BIT, _28TH_BIT, _29TH_BIT, _30TH_BIT, _31ST_BIT,
		_32ND_BIT, _33RD_BIT, _34TH_BIT, _35TH_BIT, _36TH_BIT, _37TH_BIT, _38TH_BIT, _39TH_BIT,
		_40TH_BIT, _41ST_BIT, _42ND_BIT, _43RD_BIT, _44TH_BIT, _45TH_BIT, _46TH_BIT, _47TH_BIT,
		_48TH_BIT, _49TH_BIT, _50TH_BIT, _51ST_BIT, _52ND_BIT, _53RD_BIT, _54TH_BIT, _55TH_BIT,
		_56TH_BIT, _57TH_BIT, _58TH_BIT, _59TH_BIT, _60TH_BIT, _61ST_BIT, _62ND_BIT, _63TH_BIT,
	}
)

// createMeta is a function that creates a meta object from the given parameters.
// Args:
// - memTableId: the memTableId of the object
// - offset: the offset of the object
// - length: the length of the object
// - fingerprint28: the fingerprint28 of the object
// - freq: the freq of the object
// - ttl: the ttl of the object
// - lastAccess: the lastAccess of the object
// Returns:
// - meta1: the meta1 of the object
// - meta2: the meta2 of the object
// - meta3: the meta3 of the object
type createMeta func(uint32, uint32, uint16, uint32, uint32, uint32, uint32) (uint64, uint64, uint64)

// fingerPrintMatcher is a function that matches the fingerprint28 of the object.
// Args:
// - meta2: the meta2 of the object
// - meta2: the meta2 of the object
// Returns:
// - bool: true if the meta2 matches, false otherwise
type fingerPrintMatcher func(uint64, uint64) bool

// updateLastAccess is a function that updates the lastAccess of the object.
// Args:
// - lastAccess: the lastAccess of the object
// - meta3: the meta3 of the object
// Returns:
// - uint64: the updated meta3
type updateLastAccess func(uint32, uint64) uint64

// incrementFreq is a function that increments the freq of the object.
// Args:
// - freq: the freq of the object
// Returns:
// - uint32: the incremented freq
type incrementFreq func(uint32) uint32

// extractMeta is a function to extract original params from all meta fields.
// Args:
// - meta1: the meta1 of the object
// - meta2: the meta2 of the object
// - meta3: the meta3 of the object
// Returns:
// - memTableId: the memTableId of the object
// - offset: the offset of the object
// - length: the length of the object
// - fingerprint28: the fingerprint28 of the object
// - freq: the freq of the object
// - ttl: the ttl of the object
// - lastAccess: the lastAccess of the object
type extractMeta func(uint64, uint64, uint64) (uint32, uint32, uint16, uint32, uint32, uint32, uint32)

// Capacity: 64 * 64 = 4096 entries(2^12) | Space: 64 * 8 = 512 bytes | Inline to avoid pointer indirection
// type Level0 struct {
// 	Leafs [64]uint64
// 	Sum   uint64

// 	meta1 [4096]uint64 //memId << 32 | offset
// 	meta2 [4096]uint64 //length << 48 | figerprint28 << 20 | freq
// 	meta3 [4096]uint64 //ttl << 32 | lastAccess

// }

type Level0 struct {
	Leafs [8]uint64
	Sum   uint8

	meta1 [512]uint64 //memId << 32 | offset
	meta2 [512]uint64 //length << 48 | figerprint28 << 20 | freq
	meta3 [512]uint64 //ttl << 32 | lastAccess

}

// Capacity: 64 * 4096 = 262144 entries(2^18) | Space: 64 * 512 = 32768 bytes
// type Level1 struct {
// 	Nodes [64]*Level0
// 	Sum   uint64
// }

type Level1 struct {
	Nodes [512]*Level0
	Sum   [8]uint64
}

// Capacity: 64 * 262144 =
//
//	entries(2^24) | Space: 64 * 32768 = 2097152 bytes
type Level2 struct {
	Nodes [64]*Level1
	Sum   uint64
}

type HBM24L4 struct {
	root             *Level2
	lazyAlloc        bool
	matcher          func(uint64, uint64) bool
	createMeta       createMeta
	updateLastAccess updateLastAccess
	updateFreq       incrementFreq
	extractMeta      extractMeta
}

func NewHBM24L4(lazyAlloc bool) *HBM24L4 {
	if lazyAlloc {
		return &HBM24L4{
			root:      &Level2{},
			lazyAlloc: lazyAlloc,
		}
	}
	return &HBM24L4{
		root:      generateLevel2(),
		lazyAlloc: lazyAlloc,
	}
}

func NewHBM24L4WithMatcher(lazyAlloc bool, matcher fingerPrintMatcher, createMeta createMeta, updateLastAccess updateLastAccess, updateFreq incrementFreq, extractMeta extractMeta) *HBM24L4 {
	if lazyAlloc {
		return &HBM24L4{
			root:             &Level2{},
			lazyAlloc:        lazyAlloc,
			matcher:          matcher,
			createMeta:       createMeta,
			updateLastAccess: updateLastAccess,
			updateFreq:       updateFreq,
			extractMeta:      extractMeta,
		}
	}
	return &HBM24L4{
		root:             generateLevel2(),
		lazyAlloc:        lazyAlloc,
		matcher:          matcher,
		createMeta:       createMeta,
		updateLastAccess: updateLastAccess,
		updateFreq:       updateFreq,
		extractMeta:      extractMeta,
	}
}

func generateLevel0() *Level0 {
	return &Level0{
		Leafs: [8]uint64{},
		Sum:   0,
	}
}

func generateLevel1() *Level1 {
	nodes := [512]*Level0{}
	for i := range nodes {
		nodes[i] = generateLevel0()
	}
	return &Level1{
		Nodes: nodes,
		Sum:   [8]uint64{},
	}
}

func generateLevel2() *Level2 {
	nodes := [64]*Level1{}
	for i := range nodes {
		nodes[i] = generateLevel1()
	}
	return &Level2{
		Nodes: nodes,
		Sum:   0,
	}
}

// func extractIndices24(hash uint32) (i0, i1, i2, i3 uint8) {
// 	hash &= _24BIT_MASK
// 	i0 = uint8((hash >> 18) & 0x3F)
// 	i1 = uint8((hash >> 12) & 0x3F)
// 	i2 = uint8((hash >> 6) & 0x3F)
// 	i3 = uint8(hash & 0x3F)
// 	return
// }

func extractIndices24(hash uint32) (i0 uint8, i1 uint16, i2 uint8, i3 uint8) {
	hash &= _24BIT_MASK
	i0 = uint8((hash >> 18) & _LO_6BIT_IN_32BIT)
	i1 = uint16((hash >> 9) & _LO_9BIT_IN_32BIT)
	i2 = uint8((hash >> 6) & _LO_3BIT_IN_32BIT)
	i3 = uint8(hash & _LO_6BIT_IN_32BIT)
	return
}

func (hbm *HBM24L4) Put(hash uint32) {
	i0, i1, i2, i3 := extractIndices24(hash)

	l2 := hbm.root

	l1 := l2.Nodes[i0]
	if l1 == nil {
		l1 = &Level1{
			Sum: [8]uint64{},
		}
		l2.Nodes[i0] = l1
	}

	l0 := l1.Nodes[i1]
	if l0 == nil {
		l0 = &Level0{}
		l1.Nodes[i1] = l0
	}

	l0.Leafs[i2] |= bitIndex[i3]

	// Set summary bits
	l0.Sum |= uint8(bitIndex[i2])
	l1.Sum[i1/64] |= bitIndex[i1%64]
	l2.Sum |= bitIndex[i0]
}

// Add is a function to add a new object to the HBM24L4.
// Args:
// - hash: the hash of the object
// - memTableId: the memTableId of the object
// - offset: the offset of the object
// - length: the length of the object
// - fingerprint28: the fingerprint28 of the object
// - freq: the freq of the object
// - ttl: the ttl of the object
// - lastAccess: the lastAccess of the object
// Returns:
// - bool: true if the object is overwritten because of collision on 24bit space, false otherwise
// func (hbm *HBM24L4) Add(hash uint32, memTableId uint32, offset uint32, length uint16, fingerprint28 uint32, freq uint32, ttl uint32, lastAccess uint32) bool {
// 	i0, i1, i2, i3 := extractIndices24(hash)

// 	l2 := hbm.root

// 	l1 := l2.Nodes[i0]
// 	if l1 == nil {
// 		l1 = &Level1{}
// 		l2.Nodes[i0] = l1
// 	}

// 	l0 := l1.Nodes[i1]
// 	if l0 == nil {
// 		l0 = &Level0{}
// 		l1.Nodes[i1] = l0
// 	}

// 	metaOffset := i2*64 + i3
// 	overwritten := false
// 	meta1, meta2, meta3 := hbm.createMeta(memTableId, offset, length, fingerprint28, freq, ttl, lastAccess)
// 	if l0.Leafs[i2]&bitIndex[i3] != 0 {
// 		// Slot is occupied, check if it's the same key (same fingerprint) or a collision
// 		if ok := hbm.matcher(l0.meta2[metaOffset], meta2); !ok {
// 			// Different fingerprint = collision, we're overwriting a different key
// 			overwritten = true
// 		}
// 		// If fingerprints match, we're updating the same key (overwritten = false)
// 	} else {
// 		// Slot is empty, set the bit
// 		l0.Leafs[i2] |= bitIndex[i3]
// 	}

// 	l0.meta1[metaOffset] = meta1
// 	l0.meta2[metaOffset] = meta2
// 	l0.meta3[metaOffset] = meta3

// 	// Set summary bits
// 	l0.Sum |= bitIndex[i2]
// 	l1.Sum |= bitIndex[i1]
// 	l2.Sum |= bitIndex[i0]

// 	return overwritten
// }

func (hbm *HBM24L4) Add(hash uint32, memTableId uint32, offset uint32, length uint16, fingerprint28 uint32, freq uint32, ttl uint32, lastAccess uint32) bool {
	i0, i1, i2, i3 := extractIndices24(hash)

	l2 := hbm.root

	l1 := l2.Nodes[i0]
	if l1 == nil {
		l1 = &Level1{}
		l2.Nodes[i0] = l1
	}

	l0 := l1.Nodes[i1]
	if l0 == nil {
		l0 = &Level0{}
		l1.Nodes[i1] = l0
	}

	metaOffset := i2*8 + i3
	overwritten := false
	meta1, meta2, meta3 := hbm.createMeta(memTableId, offset, length, fingerprint28, freq, ttl, lastAccess)
	if l0.Leafs[i2]&bitIndex[i3] != 0 {
		// Slot is occupied, check if it's the same key (same fingerprint) or a collision
		if ok := hbm.matcher(l0.meta2[metaOffset], meta2); !ok {
			// Different fingerprint = collision, we're overwriting a different key
			overwritten = true
		}
		// If fingerprints match, we're updating the same key (overwritten = false)
	} else {
		// Slot is empty, set the bit
		l0.Leafs[i2] |= bitIndex[i3]
	}

	l0.meta1[metaOffset] = meta1
	l0.meta2[metaOffset] = meta2
	l0.meta3[metaOffset] = meta3

	// Set summary bits
	l0.Sum |= uint8(bitIndex[i2])
	l1.Sum[i1/64] |= bitIndex[i1%64]
	l2.Sum |= bitIndex[i0]

	return overwritten
}

func (hbm *HBM24L4) Get(hash uint32) bool {
	i0, i1, i2, i3 := extractIndices24(hash)

	l2 := hbm.root
	if l2.Sum&bitIndex[i0] == 0 {
		return false
	}

	l1 := l2.Nodes[i0]
	if l1.Sum[i1/64]&bitIndex[i1%64] == 0 {
		return false
	}

	l0 := l1.Nodes[i1]
	if l0.Sum&uint8(bitIndex[i2]) == 0 {
		return false
	}

	return l0.Leafs[i2]&bitIndex[i3] != 0
}

// GetMeta is a function to get the meta object from the given hash.
// Args:
// - hash: the hash of the object
// Returns:
// - bool: true if an object is found with the given hash, false otherwise. User should do fingerprint matching to get the exact object.
// - memTableId: the memTableId of the object
// - offset: the offset of the object
// - length: the length of the object
// - fingerprint28: the fingerprint28 of the object
// - freq: the freq of the object
// - ttl: the ttl of the object
// - lastAccess: the lastAccess of the object
func (hbm *HBM24L4) GetMeta(hash uint32) (bool, uint32, uint32, uint16, uint32, uint32, uint32, uint32) {
	i0, i1, i2, i3 := extractIndices24(hash)

	l2 := hbm.root
	if l2.Sum&bitIndex[i0] == 0 {
		return false, 0, 0, 0, 0, 0, 0, 0
	}

	l1 := l2.Nodes[i0]
	if l1.Sum[i1/64]&bitIndex[i1%64] == 0 {
		return false, 0, 0, 0, 0, 0, 0, 0
	}

	l0 := l1.Nodes[i1]
	if l0.Sum&uint8(bitIndex[i2]) == 0 {
		return false, 0, 0, 0, 0, 0, 0, 0
	}

	if l0.Leafs[i2]&bitIndex[i3] != 0 {
		memTableId, offset, length, fingerprint28, freq, ttl, lastAccess := hbm.extractMeta(l0.meta1[i2*8+i3], l0.meta2[i2*8+i3], l0.meta3[i2*8+i3])
		return true, memTableId, offset, length, fingerprint28, freq, ttl, lastAccess
	}

	return false, 0, 0, 0, 0, 0, 0, 0

}

func (hbm *HBM24L4) Remove(hash uint32) {
	i0, i1, i2, i3 := extractIndices24(hash)

	l2 := hbm.root
	if l2.Sum&bitIndex[i0] == 0 {
		return
	}
	l1 := l2.Nodes[i0]
	if l1.Sum[i1/64]&bitIndex[i1%64] == 0 {
		return
	}
	l0 := l1.Nodes[i1]
	if l0.Sum&uint8(bitIndex[i2]) == 0 {
		return
	}

	l0.Leafs[i2] &= ^bitIndex[i3]

	// Clear l0 summary if that word is now 0
	if l0.Leafs[i2] == 0 {
		l0.Sum &= ^uint8(bitIndex[i2])
	}

	// Clear l1 summary if l0 is now empty
	if l0.Sum == 0 {
		l1.Sum[i1/64] &= ^bitIndex[i1%64]
	}

	// Clear l2 summary if l1 is now empty
	if l1.Sum[i1/64] == 0 {
		l2.Sum &= ^bitIndex[i0]
	}
}
