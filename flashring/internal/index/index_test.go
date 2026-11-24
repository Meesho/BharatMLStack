package index

import (
	"sync"
	"testing"
	"time"
)

func newTestIndex(t *testing.T, max int) *Index {
	t.Helper()
	return NewIndex(16, 1, max, 1, 8, &sync.RWMutex{})
}

func TestNewIndex_InitializesFields(t *testing.T) {
	idx := newTestIndex(t, 4)

	if idx.rm == nil {
		t.Fatal("rm is nil")
	}
	if idx.rb == nil {
		t.Fatal("rb is nil")
	}
	if idx.mc == nil {
		t.Fatal("mc is nil")
	}
	if idx.hashBits != 16 {
		t.Fatalf("hashBits = %d, want 16", idx.hashBits)
	}
	if idx.startAt.IsZero() {
		t.Fatal("startAt is zero, expected the creation time")
	}
}

func TestPut_NewKey_MakesItGettable(t *testing.T) {
	idx := newTestIndex(t, 8)

	idx.Put("alpha", 42, 60, 100, 200)

	length, _, ttl, _, memId, offset, status := idx.Get("alpha")
	if status != StatusOK {
		t.Fatalf("status = %d, want StatusOK", status)
	}
	if length != 42 {
		t.Fatalf("length = %d, want 42", length)
	}
	if memId != 100 || offset != 200 {
		t.Fatalf("memId/offset = %d/%d, want 100/200", memId, offset)
	}
	if ttl == 0 {
		t.Fatal("ttl = 0 immediately after Put with 60 min TTL")
	}
}

func TestGet_MissingKey_ReturnsNotFound(t *testing.T) {
	idx := newTestIndex(t, 8)

	_, _, _, _, _, _, status := idx.Get("absent")
	if status != StatusNotFound {
		t.Fatalf("status = %d, want StatusNotFound", status)
	}
}

func TestGet_ExpiredKey_ReturnsExpired(t *testing.T) {
	idx := newTestIndex(t, 8)

	// TTL 0 means expiryAt - startAt = 0 minutes → already expired at Get time.
	idx.Put("zerottl", 10, 0, 1, 1)

	_, _, _, _, _, _, status := idx.Get("zerottl")
	if status != StatusExpired {
		t.Fatalf("status = %d, want StatusExpired", status)
	}
}

// TestPut_TwoDistinctKeys_Stored verifies that two distinct keys
// (which naturally hash to different slots) are both retrievable.
// The chain-construction path is exercised by
// TestPut_TwoForcedHashCollision_ChainsAndGetsBoth below — keep these
// two tests separate so a failure points clearly at one path or the other.
func TestPut_TwoDistinctKeys_Stored(t *testing.T) {
	idx := newTestIndex(t, 8)

	idx.Put("k1", 1, 60, 1, 1)
	idx.Put("k2", 2, 60, 2, 2)

	if _, _, _, _, _, _, status := idx.Get("k1"); status != StatusOK {
		t.Fatalf("k1 status = %d, want StatusOK", status)
	}
	if _, _, _, _, _, _, status := idx.Get("k2"); status != StatusOK {
		t.Fatalf("k2 status = %d, want StatusOK", status)
	}

	// Sanity: confirm that k1 and k2 truly take distinct rm slots — if they
	// ever collide naturally this test stops covering "two-distinct-slot"
	// behaviour and silently morphs into a hash-chain test.
	hhi1, hlo1 := hash128("k1")
	hhi2, hlo2 := hash128("k2")
	if hhi1 == hhi2 && hlo1 == hlo2 {
		t.Fatal("k1 and k2 hash-collide; this test no longer covers distinct slots")
	}
	if _, ok1 := idx.rm[hlo1]; !ok1 {
		t.Fatal("rm[hlo1] missing after Put(k1)")
	}
	if _, ok2 := idx.rm[hlo2]; !ok2 {
		t.Fatal("rm[hlo2] missing after Put(k2)")
	}
}

func TestPut_TwoForcedHashCollision_ChainsAndGetsBoth(t *testing.T) {
	idx := newTestIndex(t, 8)

	// Step 1: insert one key normally.
	idx.Put("first", 1, 60, 10, 100)
	_, hlo := hash128("first")
	firstIdx := idx.rm[hlo]

	// Step 2: force a collision by manually allocating a second slot, pointing
	// it at the same hlo via the public Put — except we can't pick the hash.
	// Instead, fall back to manipulating the index state directly to drive the
	// chain branch in Put: pre-insert hlo into rm, then call Put on a key that
	// hashes elsewhere — that won't hit the branch.
	// Easier: invoke Put twice using the same key. Same key reuses the same
	// hlo and forces the chain branch on the second call.
	idx.Put("first", 2, 60, 11, 101)

	if got := idx.rm[hlo]; got == firstIdx {
		t.Fatalf("rm[hlo] = %d, want different (chained) index", got)
	}
	// Both entries live in the ring; the head of the chain is the most recent.
	length, _, _, _, memId, offset, status := idx.Get("first")
	if status != StatusOK {
		t.Fatalf("status = %d, want StatusOK", status)
	}
	if length != 2 || memId != 11 || offset != 101 {
		t.Fatalf("got (length=%d memId=%d offset=%d), want (2,11,101)", length, memId, offset)
	}
}

func TestGet_HashMismatch_WalksChainAndReturnsFallbackMatch(t *testing.T) {
	idx := newTestIndex(t, 8)

	// Put two duplicates of the same key — second Put creates a chain.
	idx.Put("dup", 1, 60, 1, 1)
	idx.Put("dup", 2, 60, 2, 2)

	// Manually corrupt the hash hi of the chain head to force a mismatch on
	// the first lookup hop, exercising the hasNext branch in Get.
	_, hlo := hash128("dup")
	headIdx := idx.rm[hlo]
	_, hnp, _ := idx.rb.Get(headIdx)
	hnp[0] = 0xDEADBEEFCAFEBABE // bogus hhi to mismatch

	length, _, _, _, _, _, status := idx.Get("dup")
	// The next link still points at the real previous entry, so status should
	// be StatusOK with length=1 from that entry.
	if status != StatusOK {
		t.Fatalf("status after chain walk = %d, want StatusOK", status)
	}
	if length != 1 {
		t.Fatalf("length after chain walk = %d, want 1", length)
	}
}

func TestGet_ChainEndsInMismatch_ReturnsNotFound(t *testing.T) {
	idx := newTestIndex(t, 8)

	idx.Put("only", 1, 60, 1, 1)
	_, hlo := hash128("only")
	headIdx := idx.rm[hlo]
	_, hnp, _ := idx.rb.Get(headIdx)
	hnp[0] = 0xDEADBEEFCAFEBABE // corrupt hhi; next is already -1

	_, _, _, _, _, _, status := idx.Get("only")
	if status != StatusNotFound {
		t.Fatalf("status = %d, want StatusNotFound", status)
	}
}

func TestDelete_ZeroCount_ReturnsZeroZero(t *testing.T) {
	idx := newTestIndex(t, 8)

	memId, count := idx.Delete(0)
	if memId != 0 || count != 0 {
		t.Fatalf("Delete(0) = (%d, %d), want (0, 0)", memId, count)
	}
}

func TestDelete_MatchingMemId_AdvancesBoundary(t *testing.T) {
	idx := newTestIndex(t, 8)

	// Two entries on memId=1 then one on memId=2 — delete should return
	// memId=2 (the boundary) and count=2.
	idx.Put("a", 1, 60, 1, 1)
	idx.Put("b", 1, 60, 1, 2)
	idx.Put("c", 1, 60, 2, 3)

	memId, count := idx.Delete(2)
	if memId != 2 {
		t.Fatalf("memId = %d, want 2", memId)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
}

func TestDelete_MemIdSkipsForward_ReturnsError(t *testing.T) {
	idx := newTestIndex(t, 8)

	// Place entries on memId=1 and memId=3 (skip 2) — Delete must surface -1
	// because the boundary is non-contiguous.
	idx.Put("a", 1, 60, 1, 1)
	idx.Put("b", 1, 60, 3, 2)

	memId, count := idx.Delete(1)
	if memId != 0 || count != -1 {
		t.Fatalf("Delete with skip = (%d, %d), want (0, -1)", memId, count)
	}
}

func TestDelete_LastIterationSameMemId_ReturnsCount(t *testing.T) {
	idx := newTestIndex(t, 8)

	// 3 entries on memId=1. Delete(2) walks one "continue" iteration then
	// hits the `nextMemId == delMemId && i == count-1` branch on the second
	// iteration, returning (1, 2).
	idx.Put("a", 1, 60, 1, 1)
	idx.Put("b", 1, 60, 1, 2)
	idx.Put("c", 1, 60, 1, 3)

	memId, count := idx.Delete(2)
	if memId != 1 || count != 2 {
		t.Fatalf("Delete = (%d, %d), want (1, 2)", memId, count)
	}
}

func TestDelete_MidChainSameMemId_Continues(t *testing.T) {
	idx := newTestIndex(t, 8)

	// Three entries on memId=1 then one on memId=2 — Delete(3) walks two
	// "continue" iterations before the boundary returns.
	idx.Put("a", 1, 60, 1, 1)
	idx.Put("b", 1, 60, 1, 2)
	idx.Put("c", 1, 60, 1, 3)
	idx.Put("d", 1, 60, 2, 4)

	memId, count := idx.Delete(3)
	if memId != 2 || count != 3 {
		t.Fatalf("Delete = (%d, %d), want (2, 3)", memId, count)
	}
}

func TestDelete_HeadHasPrev_RewritesPrevChain(t *testing.T) {
	idx := newTestIndex(t, 8)

	// Three Puts on the same key build a chain. rb.Delete advances head
	// (oldest first). That oldest entry has its `prev` set to the second
	// entry's idx (chain head walks newest→oldest), so the `else if hasPrev`
	// branch fires.
	idx.Put("k", 1, 60, 1, 1)
	idx.Put("k", 1, 60, 2, 2)
	idx.Put("k", 1, 60, 3, 3)
	_, hlo := hash128("k")

	memId, count := idx.Delete(1)
	// delMemId=1, next.memId=2 → first branch (nextMemId == delMemId+1).
	if memId != 2 || count != 1 {
		t.Fatalf("Delete(1) = (%d, %d), want (2, 1)", memId, count)
	}
	// rm[hlo] still points at the chain head (newest entry, idx=2).
	if got := idx.rm[hlo]; got != 2 {
		t.Fatalf("rm[hlo] = %d, want 2 (chain head preserved)", got)
	}
	// The previously chained entry@1 must have had its `next` rewritten to -1
	// by encodeUpdateNext, severing the link to the deleted entry.
	_, hnp, _ := idx.rb.Get(1)
	if hasNext(hnp) {
		t.Fatal("entry@1.next not severed after head Delete")
	}
}

func TestDeleteKey_PresentKey_RemovesFromMap(t *testing.T) {
	idx := newTestIndex(t, 8)
	idx.Put("k", 1, 60, 1, 1)

	if !idx.DeleteKey("k") {
		t.Fatal("DeleteKey returned false for present key")
	}
	if _, _, _, _, _, _, status := idx.Get("k"); status != StatusNotFound {
		t.Fatalf("Get after DeleteKey status = %d, want StatusNotFound", status)
	}
}

func TestDeleteKey_AbsentKey_ReturnsFalse(t *testing.T) {
	idx := newTestIndex(t, 8)

	if idx.DeleteKey("absent") {
		t.Fatal("DeleteKey returned true for absent key")
	}
}

func TestGetRB_ReturnsRingBuffer(t *testing.T) {
	idx := newTestIndex(t, 8)

	rb := idx.GetRB()
	if rb == nil {
		t.Fatal("GetRB() returned nil")
	}
	if rb != idx.rb {
		t.Fatal("GetRB returned different instance")
	}
}

func TestPeekMemIdAtHead_ReturnsHeadMemId(t *testing.T) {
	idx := newTestIndex(t, 8)
	idx.Put("k", 1, 60, 42, 0)

	memId, err := idx.PeekMemIdAtHead()
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if memId != 42 {
		t.Fatalf("memId = %d, want 42", memId)
	}
}

func TestGenerateLastAccess_ReturnsNonNegative(t *testing.T) {
	idx := newTestIndex(t, 8)

	got := idx.generateLastAccess()
	// generateLastAccess returns uint16, so it's >=0 by type. Just sanity-check
	// the call site for a fresh index produces zero (within the same minute).
	if got > 1 {
		t.Fatalf("generateLastAccess = %d, want 0 or 1 for fresh index", got)
	}
}

func TestIncrFreq_IncreasesProbabilistically(t *testing.T) {
	idx := newTestIndex(t, 8)

	first := idx.incrFreq(1, 12345)
	if first == 0 {
		t.Fatal("incrFreq returned 0 for freq=1 input")
	}
}

func TestHash128_DistinctKeys_ProduceDistinctHashes(t *testing.T) {
	hhi1, hlo1 := hash128("alpha")
	hhi2, hlo2 := hash128("beta")

	if hhi1 == hhi2 && hlo1 == hlo2 {
		t.Fatal("distinct keys produced identical hash pair")
	}
}

func TestIsHashMatch_ExactMatch(t *testing.T) {
	hnp := &HashNextPrev{0xAA, 0xBB, 0}
	if !isHashMatch(0xAA, 0xBB, hnp) {
		t.Fatal("isHashMatch returned false on exact match")
	}
}

func TestIsHashMatch_Mismatch(t *testing.T) {
	hnp := &HashNextPrev{0xAA, 0xBB, 0}
	if isHashMatch(0x11, 0xBB, hnp) {
		t.Fatal("isHashMatch returned true on hhi mismatch")
	}
	if isHashMatch(0xAA, 0x22, hnp) {
		t.Fatal("isHashMatch returned true on hlo mismatch")
	}
}

func TestHasNext_HasPrev_Sentinels(t *testing.T) {
	var entry HashNextPrev
	encodeHashNextPrev(0, 0, -1, -1, &entry)
	if hasNext(&entry) {
		t.Fatal("hasNext returned true for next=-1")
	}
	if hasPrev(&entry) {
		t.Fatal("hasPrev returned true for prev=-1")
	}

	encodeHashNextPrev(0, 0, 5, 7, &entry)
	if !hasNext(&entry) {
		t.Fatal("hasNext returned false for next=7")
	}
	if !hasPrev(&entry) {
		t.Fatal("hasPrev returned false for prev=5")
	}
}

// Expiry must stay correct however long the process has been up. It used
// to be 16-bit minutes since the index was created, so after about 45 days
// every new entry was stored already expired.
func TestExpiry_CorrectAfterLongUptime(t *testing.T) {
	idx := newTestIndex(t, 4)
	idx.startAt = idx.startAt.Add(-100 * 24 * time.Hour) // the index is 100 days old

	idx.Put("k", 10, 60, 5, 0)
	_, _, remaining, _, _, _, status := idx.Get("k")
	if status != StatusOK || remaining != 60 {
		t.Fatalf("100 days up: status = %v, remainingTTL = %d, want StatusOK and 60", status, remaining)
	}

	idx.startAt = idx.startAt.Add(-59 * time.Minute) // 59 minutes later
	if _, _, remaining, _, _, _, status := idx.Get("k"); status != StatusOK || remaining != 1 {
		t.Fatalf("59 minutes later: status = %v, remainingTTL = %d, want StatusOK and 1", status, remaining)
	}
	idx.startAt = idx.startAt.Add(-time.Minute) // one more minute
	if _, _, _, _, _, _, status := idx.Get("k"); status != StatusExpired {
		t.Fatalf("60 minutes later: status = %v, want StatusExpired", status)
	}
}

// An entry written at the last moment its memtable accepts writes, with the
// longest TTL, must still fit the 16-bit field exactly.
func TestExpiry_OldestMemtableAndLongestTTL(t *testing.T) {
	idx := newTestIndex(t, 4)
	idx.Put("first", 10, 1, 3, 0) // memtable 3 takes its first write now
	if idx.MemtableTooOld(3) {
		t.Fatal("MemtableTooOld(3) = true right after its first write")
	}

	idx.startAt = idx.startAt.Add(-MaxMemtableAgeMinutes * time.Minute)
	if idx.MemtableTooOld(3) {
		t.Fatal("MemtableTooOld(3) = true at exactly the age limit")
	}
	idx.Put("late", 10, MaxTTLMinutes, 3, 100)
	if _, _, remaining, _, _, _, status := idx.Get("late"); status != StatusOK || remaining != MaxTTLMinutes {
		t.Fatalf("status = %v, remainingTTL = %d, want StatusOK and %d", status, remaining, MaxTTLMinutes)
	}

	idx.startAt = idx.startAt.Add(-time.Minute)
	if !idx.MemtableTooOld(3) {
		t.Fatal("MemtableTooOld(3) = false one minute past the age limit")
	}
	if idx.MemtableTooOld(4) {
		t.Fatal("MemtableTooOld(4) = true for a memtable id that has taken no write")
	}
}

// When a newer memtable id takes over a birth slot, the older id's file
// slot has been overwritten; its entries must read as not found.
func TestGet_EntryOfRecycledMemtableId_NotFound(t *testing.T) {
	idx := newTestIndex(t, 4) // 8 birth slots
	idx.Put("old", 10, 60, 2, 0)
	idx.Put("new", 10, 60, 10, 0) // 10 % 8 == 2
	if _, _, _, _, _, _, status := idx.Get("old"); status != StatusNotFound {
		t.Fatalf("status = %v for an entry whose memtable id was recycled, want StatusNotFound", status)
	}
	if _, _, _, _, _, _, status := idx.Get("new"); status != StatusOK {
		t.Fatalf("status = %v for the newer entry, want StatusOK", status)
	}
}

// A chain whose links loop back on themselves must end the lookup, not
// spin the request forever.
func TestGet_CyclicChain_ReturnsNotFound(t *testing.T) {
	idx := newTestIndex(t, 4)
	idx.Put("a", 10, 60, 0, 0)
	_, hloA := hash128("a")
	slot := idx.rm[hloA]

	// Point slot's next at itself and route a different key to it.
	encodeUpdateNext(int32(slot), &idx.rb.hashTable[slot])
	_, hloB := hash128("b")
	idx.rm[hloB] = slot

	done := make(chan Status, 1)
	go func() {
		_, _, _, _, _, _, status := idx.Get("b")
		done <- status
	}()
	select {
	case status := <-done:
		if status != StatusNotFound {
			t.Fatalf("status = %v, want StatusNotFound", status)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Get did not return on a cyclic chain")
	}
}

// Index time comes from the monotonic clock, so a backward wall-clock step
// cannot make "now" earlier than a memtable's birth. If it ever were, the
// delta would go negative and wrap into a 45-day TTL.
func TestExpiry_NowBeforeMemtableBirth_DoesNotExtendTTL(t *testing.T) {
	idx := newTestIndex(t, 4)
	idx.Put("first", 10, 60, 3, 0)
	idx.startAt = idx.startAt.Add(3 * time.Minute) // "now" is 3 minutes before the birth

	idx.Put("k", 10, 1, 3, 100)
	idx.startAt = idx.startAt.Add(-13 * time.Minute) // 10 minutes after the birth
	if _, _, remaining, _, _, _, status := idx.Get("k"); status != StatusExpired {
		t.Fatalf("status = %v, remainingTTL = %d for a 1-minute TTL read 10 minutes later, want StatusExpired", status, remaining)
	}
}
