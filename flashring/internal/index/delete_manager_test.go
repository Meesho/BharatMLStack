//go:build linux
// +build linux

package index

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
)

const blockSize = 4096

// newWrapFile builds a real WrapAppendFile backed by a temp file on the
// loopback ext4 mount. Tests for DeleteManager need real punch-hole semantics
// from TrimHead, which only works on a filesystem that supports
// FALLOC_FL_PUNCH_HOLE.
func newWrapFile(t *testing.T) *fs.WrapAppendFile {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "wrap.dat")
	waf, err := fs.NewWrapAppendFile(fs.FileConfig{
		Filename:          filename,
		MaxFileSize:       int64(blockSize) * 8,
		FilePunchHoleSize: int64(blockSize) * 2,
		BlockSize:         blockSize,
	})
	if err != nil {
		t.Fatalf("NewWrapAppendFile: %v", err)
	}
	t.Cleanup(func() {
		waf.Close()
	})
	return waf
}

func newDeleteManagerWithIndex(t *testing.T, amortizedStep int) (*DeleteManager, *Index, *fs.WrapAppendFile) {
	t.Helper()
	waf := newWrapFile(t)
	idx := NewIndex(16, 1, 4, amortizedStep, 8, &sync.RWMutex{})
	dm := NewDeleteManager(idx, waf, amortizedStep)
	return dm, idx, waf
}

func TestNewDeleteManager_InitialFields(t *testing.T) {
	dm, idx, waf := newDeleteManagerWithIndex(t, 4)

	if dm.keyIndex != idx {
		t.Fatal("keyIndex not stored")
	}
	if dm.wrapFile != waf {
		t.Fatal("wrapFile not stored")
	}
	if dm.deleteAmortizedStep != 4 {
		t.Fatalf("deleteAmortizedStep = %d, want 4", dm.deleteAmortizedStep)
	}
	if dm.memtableData == nil {
		t.Fatal("memtableData is nil")
	}
	if dm.deleteInProgress {
		t.Fatal("deleteInProgress should start false")
	}
}

func TestIncMemtableKeyCount_AccumulatesPerMemtable(t *testing.T) {
	dm, _, _ := newDeleteManagerWithIndex(t, 4)

	dm.IncMemtableKeyCount(1)
	dm.IncMemtableKeyCount(1)
	dm.IncMemtableKeyCount(2)

	if got := dm.memtableData[1]; got != 2 {
		t.Fatalf("memtableData[1] = %d, want 2", got)
	}
	if got := dm.memtableData[2]; got != 1 {
		t.Fatalf("memtableData[2] = %d, want 1", got)
	}
}

func TestExecuteDeleteIfNeeded_NoTrimNeeded_NoOp(t *testing.T) {
	dm, _, _ := newDeleteManagerWithIndex(t, 4)

	// wrapFile is freshly created → not wrapped → TrimHeadIfNeeded returns
	// false. RB has no entries → NextAddNeedsDelete also false. Expect nil.
	if err := dm.ExecuteDeleteIfNeeded(); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if dm.deleteInProgress {
		t.Fatal("deleteInProgress flipped without need")
	}
}

func TestExecuteDeleteIfNeeded_NextAddNeedsDelete_StartsDelete(t *testing.T) {
	dm, idx, _ := newDeleteManagerWithIndex(t, 1)

	// All entries on memId=0 so toBeDeletedMemId (default 0) matches.
	idx.Put("a", 1, 60, 0, 0)
	idx.Put("b", 1, 60, 0, blockSize)
	idx.Put("c", 1, 60, 0, blockSize*2)
	idx.Put("d", 1, 60, 0, blockSize*3)
	dm.IncMemtableKeyCount(0)
	dm.IncMemtableKeyCount(0)
	dm.IncMemtableKeyCount(0)
	dm.IncMemtableKeyCount(0)

	err := dm.ExecuteDeleteIfNeeded()
	if err == nil || !strings.Contains(err.Error(), "trim needed retry this write") {
		t.Fatalf("err = %v, want 'trim needed retry this write'", err)
	}
	if !dm.deleteInProgress {
		t.Fatal("deleteInProgress not flipped after first cycle")
	}
}

func TestExecuteDeleteIfNeeded_HeadMemIdMismatch_ReturnsError(t *testing.T) {
	dm, idx, _ := newDeleteManagerWithIndex(t, 1)

	idx.Put("a", 1, 60, 5, 0)
	idx.Put("b", 1, 60, 5, blockSize)
	idx.Put("c", 1, 60, 5, blockSize*2)
	idx.Put("d", 1, 60, 5, blockSize*3)
	dm.IncMemtableKeyCount(5)

	// toBeDeletedMemId defaults to 0, but the actual memId at head is 5 →
	// memIdAtHead != toBeDeletedMemId branch.
	err := dm.ExecuteDeleteIfNeeded()
	if err == nil || !strings.Contains(err.Error(), "memIdAtHead") {
		t.Fatalf("err = %v, want memIdAtHead mismatch error", err)
	}
}

func TestExecuteDeleteIfNeeded_DeleteInProgress_SameMemId_DecrementsAndContinues(t *testing.T) {
	dm, idx, _ := newDeleteManagerWithIndex(t, 2)

	// Three entries on memId=0 (default toBeDeletedMemId).
	idx.Put("a", 1, 60, 0, 0)
	idx.Put("b", 1, 60, 0, blockSize)
	idx.Put("c", 1, 60, 0, blockSize*2)
	dm.memtableData[0] = 3
	dm.deleteInProgress = true
	dm.deleteCount = 2

	// Index.Delete(2) walks two iterations, hits the
	// `nextMemId == delMemId && i == count-1` branch, returns (0, 2).
	// Because memtableId == toBeDeletedMemId, we hit the bottom branch and
	// decrement memtableData[0].
	if err := dm.ExecuteDeleteIfNeeded(); err != nil {
		t.Fatalf("err = %v", err)
	}
	if got := dm.memtableData[0]; got != 1 {
		t.Fatalf("memtableData[0] = %d, want 1", got)
	}
}

func TestExecuteDeleteIfNeeded_DeleteInProgress_BoundaryHit_AdvancesMemId(t *testing.T) {
	dm, idx, _ := newDeleteManagerWithIndex(t, 2)

	idx.Put("a", 1, 60, 0, 0)
	idx.Put("b", 1, 60, 1, blockSize)
	dm.memtableData[0] = 1
	dm.memtableData[1] = 1
	dm.deleteInProgress = true
	dm.deleteCount = 1

	// Delete returns (1, 1) — memtableId (1) differs from toBeDeletedMemId (0).
	// → branch that drains memtableData[0] and advances toBeDeletedMemId.
	if err := dm.ExecuteDeleteIfNeeded(); err != nil {
		t.Fatalf("err = %v", err)
	}
	if _, ok := dm.memtableData[0]; ok {
		t.Fatal("memtableData[0] not pruned after boundary advance")
	}
	if dm.toBeDeletedMemId != 1 {
		t.Fatalf("toBeDeletedMemId = %d, want 1", dm.toBeDeletedMemId)
	}
	if dm.deleteInProgress {
		t.Fatal("deleteInProgress should be cleared after boundary")
	}
	if dm.deleteCount != 0 {
		t.Fatalf("deleteCount = %d, want 0 after boundary", dm.deleteCount)
	}
}

func TestExecuteDeleteIfNeeded_DeleteInProgress_BoundaryHit_ResidualNonZero(t *testing.T) {
	dm, idx, _ := newDeleteManagerWithIndex(t, 2)

	idx.Put("a", 1, 60, 0, 0)
	idx.Put("b", 1, 60, 1, blockSize)
	// Mismatch the accounting on purpose: tell DeleteManager there are 3 keys
	// on memtable 0 even though only 1 will be drained. Triggers the residual
	// check that returns an error.
	dm.memtableData[0] = 3
	dm.memtableData[1] = 1
	dm.deleteInProgress = true
	dm.deleteCount = 1

	err := dm.ExecuteDeleteIfNeeded()
	if err == nil || !strings.Contains(err.Error(), "memtableData[dm.toBeDeletedMemId] != 0") {
		t.Fatalf("err = %v, want 'memtableData[dm.toBeDeletedMemId] != 0'", err)
	}
}

func TestExecuteDeleteIfNeeded_DeleteInProgress_IndexFails_ReturnsError(t *testing.T) {
	dm, idx, _ := newDeleteManagerWithIndex(t, 2)

	// Force Index.Delete to return -1: put a single entry on memId=2, then
	// next.memId is zero (uninit) — mismatches both delMemId and delMemId+1
	// → falls through to the catch-all `return 0, -1`.
	idx.Put("solo", 1, 60, 2, 0)
	dm.memtableData[2] = 1
	dm.deleteInProgress = true
	dm.deleteCount = 1

	err := dm.ExecuteDeleteIfNeeded()
	if err == nil || !strings.Contains(err.Error(), "delete failed") {
		t.Fatalf("err = %v, want 'delete failed'", err)
	}
}

func TestExecuteDeleteIfNeeded_AmortizedStep_ZeroQuotient_UsesRemainder(t *testing.T) {
	dm, idx, _ := newDeleteManagerWithIndex(t, 100)

	// Fill the ring with 4 entries on memId=0. memtableData[0]=4. With
	// amortizedStep=100, the `/` is 0 and the `%` is 4 → deleteCount=4.
	idx.Put("a", 1, 60, 0, 0)
	idx.Put("b", 1, 60, 0, blockSize)
	idx.Put("c", 1, 60, 0, blockSize*2)
	idx.Put("d", 1, 60, 0, blockSize*3)
	dm.memtableData[0] = 4

	// The fifth Put wrap triggers NextAddNeedsDelete, but we need the entry
	// count to be set up — we use 4 entries and ring capacity 4. After 4 Puts,
	// rb.nextIndex wraps back to 0 == head and NextAddNeedsDelete is true.
	err := dm.ExecuteDeleteIfNeeded()
	if err == nil || !strings.Contains(err.Error(), "trim needed retry this write") {
		t.Fatalf("err = %v, want trim-needed error", err)
	}
	if dm.deleteCount != 4 {
		t.Fatalf("deleteCount = %d, want 4 (remainder branch)", dm.deleteCount)
	}
}

func TestExecuteDeleteIfNeeded_PeekHeadFails_ReturnsError(t *testing.T) {
	dm, idx, _ := newDeleteManagerWithIndex(t, 1)

	// Force rb.head past capacity to trigger PeekMemIdAtHead's error branch.
	rb := idx.GetRB()
	rb.head = rb.capacity + 1

	// Make wrapFile.TrimHeadIfNeeded() return true so the outer if-branch fires.
	dm.wrapFile.PhysicalWriteOffset.Store(dm.wrapFile.PhysicalStartOffset.Load())
	// Set wrapped to true by abusing the AdvanceWriteOffset wrap behaviour.
	dm.wrapFile.AdvanceWriteOffset(dm.wrapFile.MaxFileSize)
	// AdvanceWriteOffset sets wrapped=true and PhysicalWriteOffset=0, equal
	// to PhysicalStartOffset (also 0).

	err := dm.ExecuteDeleteIfNeeded()
	if err != ErrGettingHeadEntry {
		t.Fatalf("err = %v, want ErrGettingHeadEntry", err)
	}
}

// TestExecuteDeleteIfNeeded_TrimHeadFailure_ResetsInProgress locks down the
// m-review fix in PR #2377: previously TrimHead's error was swallowed and
// ExecuteDeleteIfNeeded returned ErrTrimNeededRetry as if space had been
// freed. The retry would then proceed against an untrimmed file, risking
// corruption. Fix propagates the underlying error and resets
// deleteInProgress so the next Put re-evaluates cleanly.
//
// We force TrimHead's fallocate to fail with EBADF by closing the write fd
// out from under it.
func TestExecuteDeleteIfNeeded_TrimHeadFailure_ResetsInProgress(t *testing.T) {
	dm, idx, waf := newDeleteManagerWithIndex(t, 1)

	// Drive the RB and file into a state where the boundary check fires
	// (NextAddNeedsDelete → true).
	idx.Put("a", 1, 60, 0, 0)
	idx.Put("b", 1, 60, 0, blockSize)
	idx.Put("c", 1, 60, 0, blockSize*2)
	idx.Put("d", 1, 60, 0, blockSize*3)
	dm.IncMemtableKeyCount(0)
	dm.IncMemtableKeyCount(0)
	dm.IncMemtableKeyCount(0)
	dm.IncMemtableKeyCount(0)

	// An invalid write fd makes unix.Fallocate inside TrimHead return EBADF.
	waf.WriteFd = -1

	err := dm.ExecuteDeleteIfNeeded()
	if err == nil {
		t.Fatal("err = nil, want error propagated from TrimHead failure")
	}
	if strings.Contains(err.Error(), "trim needed retry this write") {
		t.Fatalf("err = %v, want wrapped TrimHead error, NOT the retry sentinel", err)
	}
	if !strings.Contains(err.Error(), "trim head failed") {
		t.Fatalf("err = %v, want error wrapping 'trim head failed'", err)
	}
	if dm.deleteInProgress {
		t.Fatal("deleteInProgress not reset after TrimHead failure — retry-loop invariant broken")
	}
	if dm.deleteCount != 0 {
		t.Fatalf("deleteCount = %d, want 0 after reset", dm.deleteCount)
	}
}
