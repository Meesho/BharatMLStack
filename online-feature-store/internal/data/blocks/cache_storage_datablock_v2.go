package blocks

import (
	"bytes"
	"fmt"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/ds"
	"github.com/rs/zerolog/log"
)

type CacheType uint8

const (

	// CacheTypeInMemory represents in-memory cache
	CacheTypeInMemory CacheType = iota
	CacheTypeDistributed
	CacheTypeStorage

	csdbPrefixLen      = 4
	CSDBLayoutVersion1 = 1
)

type CacheStorageDataBlock struct {
	// 8-byte aligned map pointer
	FGIdToDDB map[int]*DeserializedPSDB // offset: 0

	// 24-byte slice (ptr, len, cap)
	serializedCSDB []byte // offset: 8

	// 4-byte fields
	TTL uint32 // offset: 32

	// 1-byte fields
	layoutVersion uint8     // offset: 36
	cacheType     CacheType // offset: 37
	// 2 bytes padding to maintain 4-byte alignment
}

func NewCacheStorageDataBlock(layoutVersion uint8) *CacheStorageDataBlock {
	return &CacheStorageDataBlock{
		layoutVersion: layoutVersion,
		FGIdToDDB:     make(map[int]*DeserializedPSDB),
	}
}

func CreateCSDBForInMemory(data []byte) (*CacheStorageDataBlock, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data to deserialize")
	}
	csdb := &CacheStorageDataBlock{
		FGIdToDDB:      nil,
		serializedCSDB: data,
		cacheType:      CacheTypeInMemory,
	}
	return csdb, nil
}

func CreateCSDBForDistributedCache(data []byte) (*CacheStorageDataBlock, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data to deserialize")
	}
	csdb := &CacheStorageDataBlock{
		FGIdToDDB:      nil,
		serializedCSDB: data,
		cacheType:      CacheTypeDistributed,
	}
	return csdb, nil
}

func CreateCSDBForStorage(data []byte) (*CacheStorageDataBlock, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data to deserialize")
	}
	csdb := &CacheStorageDataBlock{
		FGIdToDDB:      nil,
		serializedCSDB: data,
		cacheType:      CacheTypeStorage,
	}
	return csdb, nil
}

func (csdb *CacheStorageDataBlock) AddFGIdToDDB(fgId int, ddb *DeserializedPSDB) error {
	if fgId < int(system.MinUint16) || fgId > int(system.MaxUint16) {
		return fmt.Errorf("fgId out of range: %d", fgId)
	}
	csdb.FGIdToDDB[fgId] = ddb
	return nil
}

func (csdb *CacheStorageDataBlock) SerializeForInMemory() ([]byte, error) {
	return csdb.serialize(false)
}

func (csdb *CacheStorageDataBlock) SerializeForDistributedCache() ([]byte, error) {
	return csdb.serialize(true)
}

func (csdb *CacheStorageDataBlock) GetSerializedData() []byte {
	return csdb.serializedCSDB
}

func (csdb *CacheStorageDataBlock) serialize(compressed bool) ([]byte, error) {
	if len(csdb.FGIdToDDB) == 0 {
		return nil, fmt.Errorf("no data to serialize")
	}
	var buffer bytes.Buffer
	_, err := buffer.Write([]byte{csdb.layoutVersion})
	if err != nil {
		return nil, err
	}
	for fgId, ddb := range csdb.FGIdToDDB {
		if ddb.NegativeCache {
			fgSerializedData := make([]byte, csdbPrefixLen)
			system.ByteOrder.PutUint16(fgSerializedData[0:2], uint16(fgId))
			system.ByteOrder.PutUint16(fgSerializedData[2:4], 0)
			_, err := buffer.Write(fgSerializedData)
			if err != nil {
				return nil, err
			}
		} else {
			var err error
			switch ddb.LayoutVersion {
			case 1:
				{
					err = handleForPSDBLayout1(fgId, ddb, &buffer, compressed)
				}
			}
			if err != nil {
				return nil, err
			}
		}
	}
	return buffer.Bytes(), nil
}

func (csdb *CacheStorageDataBlock) GetDeserializedPSDBForAllFGIds() (map[int]*DeserializedPSDB, error) {
	if csdb.FGIdToDDB != nil {
		return csdb.FGIdToDDB, nil
	}
	if len(csdb.serializedCSDB) == 0 {
		return nil, fmt.Errorf("no data to deserialize")
	}
	fgIdToDDB := make(map[int]*DeserializedPSDB)
	layoutVersion := csdb.serializedCSDB[0]
	csdb.layoutVersion = layoutVersion
	var fgOffLenMap map[int]uint64
	var foundFGIds []uint16
	switch layoutVersion {
	case 1:
		fgOffLenMap, foundFGIds = getAllFGOffLenMapForCSDBLayout1(csdb)
	default:
		return nil, fmt.Errorf("unsupported layout version: %d", layoutVersion)
	}
	fgIds := ds.NewOrderedSetFromSlice(foundFGIds)
	fgIds.KeyIterator(func(fgId uint16) bool {
		offLen := fgOffLenMap[int(fgId)]
		startOffSet, endOffSet := system.UnpackUint64InUint32(offLen)
		fgData := csdb.serializedCSDB[startOffSet:endOffSet]
		ddb, err := DeserializePSDB(fgData)
		if err == nil && !ddb.Expired {
			fgIdToDDB[int(fgId)] = ddb
		} else {
			fgIdToDDB = nil
			return false
		}
		return true
	})
	csdb.FGIdToDDB = fgIdToDDB
	return fgIdToDDB, nil
}

func (csdb *CacheStorageDataBlock) GetDeserializedPSDBForFGIds(fgIds ds.Set[int]) (map[int]*DeserializedPSDB, error) {
	if csdb.FGIdToDDB != nil {
		log.Debug().Msgf("FGIdToDDB size: %d", len(csdb.FGIdToDDB))
		return csdb.FGIdToDDB, nil
	}
	if len(csdb.serializedCSDB) == 0 {
		return nil, fmt.Errorf("no data to deserialize")
	}
	if fgIds.IsEmpty() {
		return nil, fmt.Errorf("no fgIds to deserialize")
	}
	fgIdToDDB := make(map[int]*DeserializedPSDB)
	layoutVersion := csdb.serializedCSDB[0]
	csdb.layoutVersion = layoutVersion
	var fgOffLenMap map[int]uint64
	var foundFGIds []uint16
	switch layoutVersion {
	case 1:
		fgOffLenMap, foundFGIds = getFGOffLenMapForCSDBLayout1(csdb, fgIds)
	default:
		return nil, fmt.Errorf("unsupported layout version: %d", layoutVersion)
	}
	//Handle partial hit
	if len(foundFGIds) != fgIds.Size() {
		// For distributed cache mode return nil to trigger a cache miss
		// For storage mode, we should return whatever we found and set negative cache DDB for missing FGIds
		if csdb.cacheType != CacheTypeStorage {
			for _, id := range foundFGIds {
				fgIds.Add(int(id))
			}
			// Some FGIds are missing, have to be fetched anyway
			return nil, nil
		}
	}
	// Process found FGIds
	fgIds.KeyIterator(func(fgId int) bool {
		offLen, exists := fgOffLenMap[fgId]
		if csdb.cacheType == CacheTypeStorage && !exists {
			fgIdToDDB[fgId] = NegativeCacheDeserializePSDB()
			return true
		}

		startOffSet, endOffSet := system.UnpackUint64InUint32(offLen)
		if startOffSet == endOffSet {
			fgIdToDDB[fgId] = NegativeCacheDeserializePSDB()
		} else {
			fgData := csdb.serializedCSDB[startOffSet:endOffSet]
			ddb, err := DeserializePSDB(fgData)
			if err == nil && !ddb.Expired {
				fgIdToDDB[fgId] = ddb
			} else {
				//Handle Deserialization error or expired data
				fgIdToDDB = nil
				return false
			}
		}
		return true
	})
	csdb.FGIdToDDB = fgIdToDDB
	return fgIdToDDB, nil
}

func getFGOffLenMapForCSDBLayout1(csdb *CacheStorageDataBlock, fgIds ds.Set[int]) (map[int]uint64, []uint16) {
	fgOffLenMap := make(map[int]uint64)
	foundFGIds := make([]uint16, 0)
	idx := 1 //Skip layout version
	for idx < len(csdb.serializedCSDB)-1 {
		fgId := int(system.ByteOrder.Uint16(csdb.serializedCSDB[idx : idx+2]))
		fgDataLen := int(system.ByteOrder.Uint16(csdb.serializedCSDB[idx+2 : idx+4]))
		if !fgIds.Has(fgId) {
			//Skip if we don't need this FGId
			idx += csdbPrefixLen + fgDataLen
			continue
		}
		foundFGIds = append(foundFGIds, uint16(fgId))
		fgOffLenMap[fgId] = system.BinPackUint32InUint64(uint32(idx+csdbPrefixLen), uint32(idx+csdbPrefixLen+fgDataLen))
		idx += csdbPrefixLen + fgDataLen
	}
	return fgOffLenMap, foundFGIds
}

func getAllFGOffLenMapForCSDBLayout1(csdb *CacheStorageDataBlock) (map[int]uint64, []uint16) {
	fgOffLenMap := make(map[int]uint64)
	foundFGIds := make([]uint16, 0)
	idx := 1 //Skip layout version
	for idx < len(csdb.serializedCSDB)-1 {
		fgId := int(system.ByteOrder.Uint16(csdb.serializedCSDB[idx : idx+2]))
		fgDataLen := int(system.ByteOrder.Uint16(csdb.serializedCSDB[idx+2 : idx+4]))
		foundFGIds = append(foundFGIds, uint16(fgId))
		fgOffLenMap[fgId] = system.BinPackUint32InUint64(uint32(idx+csdbPrefixLen), uint32(idx+csdbPrefixLen+fgDataLen))
		idx += csdbPrefixLen + fgDataLen
	}
	return fgOffLenMap, foundFGIds
}

func handleForPSDBLayout1(fgId int, ddb *DeserializedPSDB, buffer *bytes.Buffer, compressed bool) error {
	if len(ddb.Header) != PSDBLayout1LengthBytes {
		return fmt.Errorf("header length is not 9")
	}
	var fgDataLen uint16
	if compressed {
		fgDataLen = uint16(len(ddb.Header) + len(ddb.CompressedData))
	} else {
		fgDataLen = uint16(len(ddb.Header) + len(ddb.OriginalData))
	}
	fgSerializedData := make([]byte, csdbPrefixLen+fgDataLen)
	system.ByteOrder.PutUint16(fgSerializedData[0:2], uint16(fgId))
	system.ByteOrder.PutUint16(fgSerializedData[2:4], uint16(fgDataLen))
	if compressed {
		copied := copy(fgSerializedData[csdbPrefixLen:], ddb.Header)
		copy(fgSerializedData[csdbPrefixLen+copied:], ddb.CompressedData)
	} else {
		clearCompressionBits(ddb.Header)
		copied := copy(fgSerializedData[csdbPrefixLen:], ddb.Header)
		copy(fgSerializedData[csdbPrefixLen+copied:], ddb.OriginalData)
	}
	_, err := buffer.Write(fgSerializedData)
	return err
}
