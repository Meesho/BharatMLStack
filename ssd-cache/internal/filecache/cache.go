package filecache

import (
	"fmt"
	"hash/crc32"
	"time"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/fs"
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/indices"
	"github.com/Meesho/BharatMLStack/ssd-cache/internal/memtables"
	"github.com/rs/zerolog/log"
)

type FileCache struct {
	keyIndex *indices.KeyIndex
	file     *fs.WrapAppendFile
	mm       *memtables.MemtableManager
}

type FileCacheConfig struct {
	Rounds              int
	RbInitial           int
	RbMax               int
	DeleteAmortizedStep int
	MemtableSize        int32
	MaxFileSize         int64
	BlockSize           int
	Directory           string
}

func NewFileCache(config FileCacheConfig) *FileCache {
	filename := fmt.Sprintf("%s/%d.bin", config.Directory, time.Now().UnixNano())
	punchHoleSize := config.MemtableSize
	fsConf := fs.FileConfig{
		Filename:          filename,
		MaxFileSize:       config.MaxFileSize,
		FilePunchHoleSize: int64(punchHoleSize),
		BlockSize:         config.BlockSize,
	}
	file, err := fs.NewWrapAppendFile(fsConf)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to create file")
	}
	memtableManager, err := memtables.NewMemtableManager(file, config.MemtableSize)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to create memtable manager")
	}
	ki := indices.NewKeyIndex(config.Rounds, config.RbInitial, config.RbMax, config.DeleteAmortizedStep)
	return &FileCache{
		keyIndex: ki,
		mm:       memtableManager,
		file:     file,
	}
}

func (fc *FileCache) Put(key string, value []byte, exptime uint64) error {
	size := 4 + len(key) + len(value)
	mt, mtId, _ := fc.mm.GetMemtable()
	buf, offset, length, readyForFlush := mt.GetBuf(uint16(size))
	if readyForFlush {
		fc.mm.Flush()
		mt, mtId, _ = fc.mm.GetMemtable()
		buf, offset, length, _ = mt.GetBuf(uint16(size))
	}
	copy(buf[4:], key)
	copy(buf[4+len(key):], value)
	crc := crc32.ChecksumIEEE(buf[4:])
	indices.ByteOrder.PutUint32(buf[0:4], crc)
	fc.keyIndex.Put(key, length, mtId, uint32(offset), exptime)
	return nil
}
