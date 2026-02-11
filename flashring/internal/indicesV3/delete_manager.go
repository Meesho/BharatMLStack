package indicesv2

import (
	"errors"
	"fmt"

	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
	"github.com/rs/zerolog/log"
)

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
		toBeDeletedMemId:    0,
		keyIndex:            keyIndex,
		wrapFile:            wrapFile,
		deleteInProgress:    false,
		deleteAmortizedStep: deleteAmortizedStep,
	}
}

func (dm *DeleteManager) IncMemtableKeyCount(memId uint32) {
	dm.memtableData[memId]++
}

// ExecuteDeleteIfNeeded is the legacy inline delete path.
// Retained for use in modes without a background delete worker (e.g. lockless).
func (dm *DeleteManager) ExecuteDeleteIfNeeded() error {
	if dm.deleteInProgress {
		memtableId, count := dm.keyIndex.Delete(dm.deleteCount)
		if count == -1 {
			return fmt.Errorf("delete failed")
		}
		if memtableId != dm.toBeDeletedMemId {
			dm.memtableData[dm.toBeDeletedMemId] = dm.memtableData[dm.toBeDeletedMemId] - count
			log.Debug().Msgf("memtableId: %d, toBeDeletedMemId: %d", memtableId, dm.toBeDeletedMemId)
			if dm.memtableData[dm.toBeDeletedMemId] != 0 {
				return fmt.Errorf("memtableData[dm.toBeDeletedMemId] != 0")
			}
			delete(dm.memtableData, dm.toBeDeletedMemId)
			dm.toBeDeletedMemId = memtableId
			dm.deleteInProgress = false
			dm.deleteCount = 0
			return nil
		} else {
			dm.memtableData[memtableId] -= count
		}
		return nil
	}

	trimNeeded := dm.wrapFile.TrimHeadIfNeeded()
	nextAddNeedsDelete := dm.keyIndex.GetRB().NextAddNeedsDelete()

	if trimNeeded || nextAddNeedsDelete {
		dm.deleteInProgress = true
		dm.deleteCount = int(dm.memtableData[dm.toBeDeletedMemId] / dm.deleteAmortizedStep)
		if dm.deleteCount == 0 {
			dm.deleteCount = int(dm.memtableData[dm.toBeDeletedMemId] % dm.deleteAmortizedStep)
		}
		memIdAtHead, err := dm.keyIndex.PeekMemIdAtHead()
		if err != nil {
			return err
		}
		if memIdAtHead != dm.toBeDeletedMemId {
			return fmt.Errorf("memIdAtHead: %d, toBeDeletedMemId: %d", memIdAtHead, dm.toBeDeletedMemId)
		}

		dm.wrapFile.TrimHead()
		return errors.New("trim needed retry this write")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Background delete worker API
// ---------------------------------------------------------------------------

// NeedsWork returns true if there is pending delete/trim work.
// Safe to call without the shard lock for approximate (racy) checks;
// a false-negative just delays work by one poll cycle.
func (dm *DeleteManager) NeedsWork() bool {
	if dm.deleteInProgress {
		return true
	}
	return dm.wrapFile.TrimHeadIfNeeded() || dm.keyIndex.GetRB().NextAddNeedsDelete()
}

// IsDeleteInProgress returns whether a delete round is currently active.
func (dm *DeleteManager) IsDeleteInProgress() bool {
	return dm.deleteInProgress
}

// InitDeleteRound checks whether a new delete round should start and, if so,
// initialises it (computes delete count, calls TrimHead). If a round is
// already in progress it returns (true, nil) immediately.
// Must be called under the shard lock.
func (dm *DeleteManager) InitDeleteRound() (bool, error) {
	if dm.deleteInProgress {
		return true, nil
	}

	trimNeeded := dm.wrapFile.TrimHeadIfNeeded()
	nextAddNeedsDelete := dm.keyIndex.GetRB().NextAddNeedsDelete()

	if !trimNeeded && !nextAddNeedsDelete {
		return false, nil
	}

	dm.deleteInProgress = true
	dm.deleteCount = int(dm.memtableData[dm.toBeDeletedMemId] / dm.deleteAmortizedStep)
	if dm.deleteCount == 0 {
		dm.deleteCount = int(dm.memtableData[dm.toBeDeletedMemId] % dm.deleteAmortizedStep)
	}

	memIdAtHead, err := dm.keyIndex.PeekMemIdAtHead()
	if err != nil {
		dm.deleteInProgress = false
		return false, err
	}
	if memIdAtHead != dm.toBeDeletedMemId {
		dm.deleteInProgress = false
		return false, fmt.Errorf("memIdAtHead: %d, toBeDeletedMemId: %d", memIdAtHead, dm.toBeDeletedMemId)
	}

	if err := dm.wrapFile.TrimHead(); err != nil {
		dm.deleteInProgress = false
		return false, fmt.Errorf("TrimHead failed: %w", err)
	}

	return true, nil
}

// ExecuteDeleteBatch performs at most maxPerBatch deletes from the ring buffer
// head. Must be called under the shard lock when a delete round is in
// progress. Returns (roundComplete, error).
func (dm *DeleteManager) ExecuteDeleteBatch(maxPerBatch int) (bool, error) {
	if !dm.deleteInProgress {
		return true, nil
	}

	batchSize := dm.deleteCount
	if batchSize > maxPerBatch {
		batchSize = maxPerBatch
	}
	if batchSize == 0 {
		dm.deleteInProgress = false
		dm.deleteCount = 0
		return true, nil
	}

	memtableId, count := dm.keyIndex.Delete(batchSize)
	if count == -1 {
		return false, fmt.Errorf("delete failed")
	}

	if memtableId != dm.toBeDeletedMemId {
		dm.memtableData[dm.toBeDeletedMemId] = dm.memtableData[dm.toBeDeletedMemId] - count
		log.Debug().Msgf("memtableId: %d, toBeDeletedMemId: %d", memtableId, dm.toBeDeletedMemId)
		if dm.memtableData[dm.toBeDeletedMemId] != 0 {
			return false, fmt.Errorf("memtableData[dm.toBeDeletedMemId] != 0")
		}
		delete(dm.memtableData, dm.toBeDeletedMemId)
		dm.toBeDeletedMemId = memtableId
		dm.deleteInProgress = false
		dm.deleteCount = 0
		return true, nil
	}

	dm.memtableData[memtableId] -= count
	return false, nil
}
