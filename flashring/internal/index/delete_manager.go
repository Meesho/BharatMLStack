package index

import (
	"errors"
	"fmt"

	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
	"github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
	"github.com/rs/zerolog/log"
)

// ErrTrimNeededRetry is returned by ExecuteDeleteIfNeeded after it punches a
// hole at the file head. Callers up the stack (WrapCache.Put) must detect this
// with errors.Is and retry the write once — the retry will hit the
// delete-in-progress branch and complete the amortized cleanup rather than
// re-tripping the boundary. The string form is preserved for existing tests
// that assert on the message.
var ErrTrimNeededRetry = errors.New("trim needed retry this write")

// DeleteManager amortizes the cost of evicting an aged-out memtable's
// entries from the index across many Put calls. Without amortization,
// the Put that filled the ring would pay for evicting every entry in the
// next memtable to be recycled, producing a tail-latency spike. Instead,
// when the underlying file or ring buffer signals it needs to free
// space, DeleteManager kicks off a multi-step delete pass: each Put
// invokes ExecuteDeleteIfNeeded once, deleting deleteAmortizedStep
// entries at a time until the doomed memtable's entries are fully
// drained.
type DeleteManager struct {
	memtableData        map[uint32]int
	toBeDeletedMemId    uint32
	keyIndex            *Index
	wrapFile            *fs.WrapAppendFile
	deleteInProgress    bool
	deleteAmortizedStep int
	deleteCount         int
}

func NewDeleteManager(keyIndex *Index, wrapFile *fs.WrapAppendFile, deleteAmortizedStep int) *DeleteManager {
	return &DeleteManager{
		memtableData:        make(map[uint32]int),
		keyIndex:            keyIndex,
		wrapFile:            wrapFile,
		deleteAmortizedStep: deleteAmortizedStep,
	}
}

func (dm *DeleteManager) IncMemtableKeyCount(memId uint32) {
	dm.memtableData[memId]++
}

// ExecuteDeleteIfNeeded advances the amortized delete state machine for
// the calling Put. Three outcomes:
//   - delete-in-progress: pop one chunk from the ring, advance state.
//     Returns nil so the Put proceeds.
//   - boundary detected: punch a hole at the file head, mark
//     delete-in-progress, and return a "trim needed retry this write"
//     error so the caller retries the Put after the head has been freed.
//   - nothing to do: returns nil.
//
// The "retry" error is load-bearing — it forces the current Put to be
// reattempted after the boundary trim, ensuring no Put writes to a slot
// whose memtable is mid-eviction.
func (dm *DeleteManager) ExecuteDeleteIfNeeded() error {
	if dm.deleteInProgress {
		memtableId, count := dm.keyIndex.Delete(dm.deleteCount)
		if count == -1 {
			return fmt.Errorf("delete failed")
		}
		metrics.Count(metrics.KEY_DELETED_KEY_COUNT, int64(count), nil)
		if memtableId != dm.toBeDeletedMemId {
			dm.memtableData[dm.toBeDeletedMemId] -= count
			log.Debug().Msgf("memtableId: %d, toBeDeletedMemId: %d", memtableId, dm.toBeDeletedMemId)
			if dm.memtableData[dm.toBeDeletedMemId] != 0 {
				return fmt.Errorf("memtableData[dm.toBeDeletedMemId] != 0")
			}
			delete(dm.memtableData, dm.toBeDeletedMemId)
			dm.toBeDeletedMemId = memtableId
			dm.deleteInProgress = false
			dm.deleteCount = 0
			return nil
		}
		dm.memtableData[memtableId] -= count
		return nil
	}

	trimNeeded := dm.wrapFile.TrimHeadIfNeeded()
	nextAddNeedsDelete := dm.keyIndex.GetRB().NextAddNeedsDelete()

	if trimNeeded || nextAddNeedsDelete {
		dm.deleteInProgress = true
		dm.deleteCount = dm.memtableData[dm.toBeDeletedMemId] / dm.deleteAmortizedStep
		if dm.deleteCount == 0 {
			dm.deleteCount = dm.memtableData[dm.toBeDeletedMemId] % dm.deleteAmortizedStep
		}
		memIdAtHead, err := dm.keyIndex.PeekMemIdAtHead()
		if err == nil && memIdAtHead != dm.toBeDeletedMemId {
			err = fmt.Errorf("memIdAtHead: %d, toBeDeletedMemId: %d", memIdAtHead, dm.toBeDeletedMemId)
		}
		if err != nil {
			// Nothing was trimmed. Leaving deleteInProgress set would make
			// every later Put skip this check and never trim again, while
			// the index ring kept overwriting live slots.
			dm.deleteInProgress = false
			dm.deleteCount = 0
			return err
		}

		if err := dm.wrapFile.TrimHead(); err != nil {
			// fallocate(PUNCH_HOLE) failed — the file head did NOT
			// advance, so returning ErrTrimNeededRetry would let the
			// caller retry as though space was freed. Reset the
			// in-progress flag so the next Put re-evaluates cleanly,
			// and surface the underlying error.
			dm.deleteInProgress = false
			dm.deleteCount = 0
			return fmt.Errorf("trim head failed: %w", err)
		}
		return ErrTrimNeededRetry
	}
	return nil
}
