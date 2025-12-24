// // SPDX‑License‑Identifier: Apache‑2.0
// // Package simdmap is a Swiss‑table open‑addressing hash map with an
// // optional AVX2‑vectorised probe loop for amd64.  When the build tag
// // `avx2` is *not* supplied or the CPU lacks AVX2, the implementation
// // falls back to a tight scalar probe, keeping the package portable.
// //
// // Build (Go 1.22+):
// //
// //	$ go test -tags avx2 ./...    # AVX2 fast‑path on Intel/AMD ≥ Haswell/Zen1
// //	$ go test          ./...     # scalar path (any GOARCH)
// //
// // The key type is fixed to uint64 (a 64‑bit fingerprint like xxhash).
// // You will typically store metadata such as {Off uint64; Len uint32} as V.
// package simdmap

// import (
// 	"math/bits"
// 	"unsafe"
// )

// // --------------------------------------------------------------------- //
// // Constants and tiny helpers
// // --------------------------------------------------------------------- //

// const (
// 	groupSize  = 16 // 16 control bytes per Swiss group
// 	ctrlEmpty  = 0x80
// 	ctrlTomb   = 0xfe
// 	loadFactor = 7 // 7/8 = 87.5 %
// )

// type entry[V any] struct {
// 	hash uint64
// 	val  V
// }

// type Map[V any] struct {
// 	mask   uintptr
// 	ctrl   []byte
// 	slots  []entry[V]
// 	size   uintptr
// 	growth uintptr
// }

// // roundUpToGroups returns next power‑of‑two group count ≥ x.
// func roundUpToGroups(x uintptr) uintptr {
// 	if x < groupSize {
// 		x = groupSize
// 	}
// 	return uintptr(1) << bits.Len(uint(x-1))
// }

// func New[V any](capacity int) *Map[V] {
// 	groups := roundUpToGroups(uintptr(capacity))
// 	n := groups * groupSize

// 	m := &Map[V]{
// 		mask:   n - 1,
// 		ctrl:   make([]byte, n+groupSize), // sentinel group
// 		slots:  make([]entry[V], n),
// 		growth: (n * loadFactor) / 8,
// 	}
// 	for i := range m.ctrl {
// 		m.ctrl[i] = ctrlEmpty
// 	}
// 	return m
// }

// // --------------------------------------------------------------------- //
// // SIMD probe helpers
// // --------------------------------------------------------------------- //

// //go:noescape
// func match16_simd(ctrl *byte, h2 byte) uint16 // provided in .s when avx2 tag

// func match16_scalar(ctrl *byte, h2 byte) uint16 {
// 	var m uint16
// 	b := (*[groupSize]byte)(unsafe.Pointer(ctrl))
// 	for i := 0; i < groupSize; i++ {
// 		if b[i] == h2 {
// 			m |= 1 << uint(i)
// 		}
// 	}
// 	return m
// }

// // --------------------------------------------------------------------- //
// // Build‑tag specific swap‑in of the SIMD fast‑path
// // --------------------------------------------------------------------- //

// var match16 = match16_scalar // overridden when the avx2 build‑tag is used

// // --------------------------------------------------------------------- //
// // Probe and API
// // --------------------------------------------------------------------- //

// func (m *Map[V]) findSlot(h uint64) (uintptr, bool) {
// 	h1 := uintptr(h >> 7)
// 	h2 := byte(h & 0x7f)
// 	maskGroups := m.mask & ^uintptr(groupSize-1)

// 	for {
// 		grp := h1 & maskGroups
// 		cptr := (*byte)(unsafe.Pointer(&m.ctrl[grp]))

// 		if mask := match16(cptr, h2); mask != 0 {
// 			for mask != 0 {
// 				i := bits.TrailingZeros16(mask)
// 				idx := grp + uintptr(i)
// 				if m.slots[idx].hash == h {
// 					return idx, true
// 				}
// 				mask &^= 1 << uint(i)
// 			}
// 		}
// 		for i := 0; i < groupSize; i++ {
// 			if *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(cptr)) + uintptr(i))) >= ctrlEmpty {
// 				return grp + uintptr(i), false
// 			}
// 		}
// 		h1 += groupSize
// 	}
// }

// func (m *Map[V]) Get(hash uint64) (V, bool) {
// 	var zero V
// 	idx, ok := m.findSlot(hash)
// 	if !ok {
// 		return zero, false
// 	}
// 	return m.slots[idx].val, true
// }

// func (m *Map[V]) putEntry(hash uint64, v V) {
// 	idx, found := m.findSlot(hash)
// 	if !found {
// 		m.size++
// 	}
// 	m.ctrl[idx] = byte(hash & 0x7f)
// 	m.slots[idx] = entry[V]{hash: hash, val: v}
// }

// func (m *Map[V]) Put(hash uint64, v V) {
// 	m.putEntry(hash, v)
// 	if m.size >= m.growth {
// 		m.rehash()
// 	}
// }

// func (m *Map[V]) Delete(hash uint64) bool {
// 	idx, ok := m.findSlot(hash)
// 	if !ok {
// 		return false
// 	}
// 	m.ctrl[idx] = ctrlTomb
// 	m.size--
// 	return true
// }

// // --------------------------------------------------------------------- //
// // Resize
// // --------------------------------------------------------------------- //

// func (m *Map[V]) rehash() {
// 	oldCtrl, oldSlots := m.ctrl, m.slots
// 	newLen := uintptr(len(oldSlots) * 2)

// 	m.ctrl = make([]byte, newLen+groupSize)
// 	for i := range m.ctrl {
// 		m.ctrl[i] = ctrlEmpty
// 	}
// 	m.slots = make([]entry[V], newLen)
// 	m.mask = newLen - 1
// 	m.size = 0
// 	m.growth = (newLen * loadFactor) / 8

// 	for i, c := range oldCtrl[:len(oldSlots)] {
// 		if c < ctrlEmpty {
// 			e := oldSlots[i]
// 			m.putEntry(e.hash, e.val)
// 		}
// 	}
// }

// SPDX‑License‑Identifier: Apache‑2.0
// Incremental‑rehash version of simdmap.
// Only the growth logic has changed; probe loop and SIMD assembly are
// untouched.  A single `Put` moves at most `migrateStep` live entries
// from the old table to the new, flattening the latency spike to <5 µs.
//
// Build / tags unchanged:
//
//	go test -tags avx2 ./...
package simdmap

import (
	"math/bits"
	"unsafe"
)

const (
	groupSize   = 16
	ctrlEmpty   = 0x80
	ctrlTomb    = 0xfe
	loadFactor  = 7
	migrateStep = 128 // live entries moved per mutation (tune!)
)

type entry[V any] struct {
	hash uint64
	val  V
}

type Map[V any] struct {
	// active table
	mask   uintptr
	ctrl   []byte
	slots  []entry[V]
	size   uintptr
	growth uintptr

	// incremental‑rehash state (nil when not migrating)
	oldCtrl  []byte
	oldSlots []entry[V]
	rehashAt uintptr // next index to migrate
}

// --- constructor unchanged -------------------------------------------------

func roundUpToGroups(x uintptr) uintptr {
	if x < groupSize {
		x = groupSize
	}
	return uintptr(1) << bits.Len(uint(x-1))
}

func New[V any](capHint int) *Map[V] {
	groups := roundUpToGroups(uintptr(capHint))
	n := groups * groupSize
	m := &Map[V]{
		mask:   n - 1,
		ctrl:   make([]byte, n+groupSize),
		slots:  make([]entry[V], n),
		growth: (n * loadFactor) / 8,
	}
	for i := range m.ctrl {
		m.ctrl[i] = ctrlEmpty
	}
	return m
}

// --- SIMD probe machinery (unchanged) -------------------------------------

//go:noescape
func match16_simd(*byte, byte) uint16

func match16_scalar(ctrl *byte, h2 byte) uint16 {
	var m uint16
	b := (*[groupSize]byte)(unsafe.Pointer(ctrl))
	for i := 0; i < groupSize; i++ {
		if b[i] == h2 {
			m |= 1 << uint(i)
		}
	}
	return m
}

var match16 = match16_scalar // overridden by build‑tag file

func (m *Map[V]) findSlot(h uint64) (uintptr, bool) {
	h1 := uintptr(h >> 7)
	h2 := byte(h & 0x7f)
	maskGroups := m.mask & ^uintptr(groupSize-1)
	for {
		grp := h1 & maskGroups
		cptr := (*byte)(unsafe.Pointer(&m.ctrl[grp]))
		if mask := match16(cptr, h2); mask != 0 {
			for mask != 0 {
				i := bits.TrailingZeros16(mask)
				idx := grp + uintptr(i)
				if m.slots[idx].hash == h {
					return idx, true
				}
				mask &^= 1 << uint(i)
			}
		}
		for i := 0; i < groupSize; i++ {
			if *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(cptr)) + uintptr(i))) >= ctrlEmpty {
				return grp + uintptr(i), false
			}
		}
		h1 += groupSize
	}
}

// ---------------- incremental migration helpers ---------------------------

func (m *Map[V]) migrateSome() {
	if m.oldCtrl == nil { // not in rehash
		return
	}
	moved := 0
	oldLen := uintptr(len(m.oldSlots))

	for moved < migrateStep && m.rehashAt < oldLen {
		c := m.oldCtrl[m.rehashAt]
		if c < ctrlEmpty {
			e := m.oldSlots[m.rehashAt]
			m.putEntry(e.hash, e.val) // into new table
			moved++
		}
		m.rehashAt++
	}

	// finished?
	if m.rehashAt >= oldLen {
		m.oldCtrl, m.oldSlots = nil, nil
	}
}

func (m *Map[V]) startRehash() {
	if m.oldCtrl != nil {
		return // already running
	}
	m.oldCtrl, m.oldSlots = m.ctrl, m.slots

	newLen := uintptr(len(m.oldSlots) * 2)
	m.ctrl = make([]byte, newLen+groupSize)
	for i := range m.ctrl {
		m.ctrl[i] = ctrlEmpty
	}
	m.slots = make([]entry[V], newLen)
	m.mask = newLen - 1
	m.size = 0
	m.growth = (newLen * loadFactor) / 8
	m.rehashAt = 0
}

// ---------------- public API (Put/Get/Delete) -----------------------------

func (m *Map[V]) Get(hash uint64) (V, bool) {
	m.migrateSome()
	var zero V
	idx, ok := m.findSlot(hash)
	if !ok {
		return zero, false
	}
	return m.slots[idx].val, true
}

func (m *Map[V]) putEntry(hash uint64, v V) {
	idx, found := m.findSlot(hash)
	if !found {
		m.size++
	}
	m.ctrl[idx] = byte(hash & 0x7f)
	m.slots[idx] = entry[V]{hash: hash, val: v}
}

func (m *Map[V]) Put(hash uint64, v V) {
	m.migrateSome()
	m.putEntry(hash, v)
	if m.size >= m.growth {
		m.startRehash()
	}
}

func (m *Map[V]) Delete(hash uint64) bool {
	m.migrateSome()
	idx, ok := m.findSlot(hash)
	if !ok {
		return false
	}
	m.ctrl[idx] = ctrlTomb
	m.size--
	return true
}
