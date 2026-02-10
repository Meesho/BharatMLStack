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
			//log.Debug().Msgf("memtableData[%d] = %d", memtableId, dm.memtableData[memtableId])
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
