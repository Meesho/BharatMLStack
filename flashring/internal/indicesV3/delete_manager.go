package indicesv2

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/flashring/internal/fs"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
)

type DeleteManager struct {
	memtableData        *xsync.Map[uint32, int]
	toBeDeletedMemId    uint32
	keyIndex            *Index
	wrapFile            *fs.WrapAppendFile
	deleteInProgress    bool
	deleteAmortizedStep int
	deleteCount         int
}

func NewDeleteManager(keyIndex *Index, wrapFile *fs.WrapAppendFile, deleteAmortizedStep int) *DeleteManager {
	return &DeleteManager{
		memtableData:        xsync.NewMap[uint32, int](),
		toBeDeletedMemId:    0,
		keyIndex:            keyIndex,
		wrapFile:            wrapFile,
		deleteInProgress:    false,
		deleteAmortizedStep: deleteAmortizedStep,
	}
}

func (dm *DeleteManager) IncMemtableKeyCount(memId uint32) {
	dm.memtableData.Compute(memId, func(oldValue int, loaded bool) (int, xsync.ComputeOp) {
		return oldValue + 1, xsync.UpdateOp
	})
}

func (dm *DeleteManager) ExecuteDeleteIfNeeded() error {
	if dm.deleteInProgress {
		memtableId, count := dm.keyIndex.Delete(dm.deleteCount)
		if count == -1 {
			return fmt.Errorf("delete failed")
		}
		if memtableId != dm.toBeDeletedMemId {
			newVal, _ := dm.memtableData.Compute(dm.toBeDeletedMemId, func(oldValue int, loaded bool) (int, xsync.ComputeOp) {
				return oldValue - count, xsync.UpdateOp
			})
			log.Debug().Msgf("memtableId: %d, toBeDeletedMemId: %d", memtableId, dm.toBeDeletedMemId)
			if newVal != 0 {
				return fmt.Errorf("memtableData[dm.toBeDeletedMemId] != 0")
			}
			dm.memtableData.Delete(dm.toBeDeletedMemId)
			dm.toBeDeletedMemId = memtableId
			dm.deleteInProgress = false
			dm.deleteCount = 0
			return nil
		} else {
			dm.memtableData.Compute(memtableId, func(oldValue int, loaded bool) (int, xsync.ComputeOp) {
				return oldValue - count, xsync.UpdateOp
			})
		}
		return nil
	}

	trimNeeded := dm.wrapFile.TrimHeadIfNeeded()
	nextAddNeedsDelete := dm.keyIndex.GetRB().NextAddNeedsDelete()

	if trimNeeded || nextAddNeedsDelete {
		dm.deleteInProgress = true
		val, _ := dm.memtableData.Load(dm.toBeDeletedMemId)
		dm.deleteCount = val / dm.deleteAmortizedStep
		if dm.deleteCount == 0 {
			dm.deleteCount = val % dm.deleteAmortizedStep
		}
		memIdAtHead, err := dm.keyIndex.PeekMemIdAtHead()
		if err != nil {
			return err
		}
		if memIdAtHead != dm.toBeDeletedMemId {
			return fmt.Errorf("memIdAtHead: %d, toBeDeletedMemId: %d", memIdAtHead, dm.toBeDeletedMemId)
		}

		dm.wrapFile.TrimHead()
		return fmt.Errorf("trim needed or next add needs delete")
	}
	return nil
}
