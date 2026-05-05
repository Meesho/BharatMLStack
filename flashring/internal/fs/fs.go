//go:build linux
// +build linux

package fs

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

const (
	O_DIRECT             = 0x4000
	O_WRONLY             = syscall.O_WRONLY
	O_RDONLY             = syscall.O_RDONLY
	O_APPEND             = syscall.O_APPEND
	O_CREAT              = syscall.O_CREAT
	O_DSYNC              = syscall.O_DSYNC
	FALLOC_FL_PUNCH_HOLE = unix.FALLOC_FL_PUNCH_HOLE
	FALLOC_FL_KEEP_SIZE  = unix.FALLOC_FL_KEEP_SIZE
	FILE_MODE            = 0644
	BLOCK_SIZE           = 4096
)

var (
	ErrBufNoAlign           = errors.New("buffer is not aligned to block size")
	ErrFileSizeExceeded     = errors.New("file size exceeded. Please punch hole")
	ErrFileOffsetOutOfRange = errors.New("file offset is out of range")
	ErrOffsetNotAligned     = errors.New("offset is not aligned to block size")
	ErrReadTimeout          = errors.New("read timeout")
)

type Stat struct {
	WriteCount         atomic.Int64
	ReadCount          atomic.Int64
	PunchHoleCount     atomic.Int64
	CurrentLogicalSize int64
}

type FileConfig struct {
	Filename          string
	MaxFileSize       int64
	FilePunchHoleSize int64
	BlockSize         int
}

type File interface {
	Pwrite(buf []byte) (currentPhysicalOffset int64, err error)
	Pread(fileOffset int64, buf []byte) (n int32, err error)
	TrimHead() (err error)
	Close()
}

type Page interface {
	Unmap() error
}

// openWithDirectIO attempts to open a file with O_DIRECT, falling back to
// regular flags if the filesystem doesn't support it.
func openWithDirectIO(filename string, baseFlags int) (int, bool, error) {
	fd, err := syscall.Open(filename, baseFlags|O_DIRECT, FILE_MODE)
	if err == nil {
		return fd, true, nil
	}
	log.Warn().Msgf("DIRECT_IO not supported, falling back to regular flags: %v", err)
	fd, err = syscall.Open(filename, baseFlags, FILE_MODE)
	if err != nil {
		return 0, false, err
	}
	return fd, false, nil
}

func fdToFile(fd int, filename string) (*os.File, error) {
	file := os.NewFile(uintptr(fd), filename)
	if file == nil {
		return nil, fmt.Errorf("failed to create file from fd")
	}
	return file, nil
}

func createAppendOnlyWriteFileDescriptor(filename string) (int, *os.File, bool, error) {
	fd, directIO, err := openWithDirectIO(filename, O_WRONLY|O_CREAT|O_DSYNC)
	if err != nil {
		return 0, nil, false, err
	}
	file, err := fdToFile(fd, filename)
	if err != nil {
		return 0, nil, false, err
	}
	return fd, file, directIO, nil
}

func createPreAllocatedWriteFileDescriptor(filename string, maxFileSize int64) (int, *os.File, bool, error) {
	fd, directIO, err := openWithDirectIO(filename, O_WRONLY|O_CREAT|O_DSYNC)
	if err != nil {
		return 0, nil, false, err
	}

	if err = unix.Fallocate(fd, 0, 0, maxFileSize); err != nil {
		log.Error().Err(err).Msg("Failed to fallocate file")
		syscall.Close(fd)
		return 0, nil, false, err
	}

	file, err := fdToFile(fd, filename)
	if err != nil {
		return 0, nil, false, err
	}
	return fd, file, directIO, nil
}

func createReadFileDescriptor(filename string) (int, *os.File, bool, error) {
	flags := O_DIRECT | O_RDONLY
	fd, err := syscall.Open(filename, flags, 0)
	if err != nil {
		return 0, nil, false, err
	}
	file, err := fdToFile(fd, filename)
	if err != nil {
		return 0, nil, false, err
	}
	return fd, file, true, nil
}

func isAlignedBuffer(buf []byte, alignment int) bool {
	pt := uintptr(alignment)
	if len(buf) == 0 {
		return false
	}
	addr := uintptr(unsafe.Pointer(&buf[0]))
	return addr%pt == 0
}

func isAlignedOffset(offset int64, alignment int) bool {
	return offset%int64(alignment) == 0
}

// AlignRange computes the block-aligned start offset and total aligned size
// for a read spanning [offset, offset+length). Useful for O_DIRECT reads
// where both offset and buffer size must be block-aligned.
func AlignRange(offset int64, length int, blockSize int64) (alignedStart, alignedSize int64) {
	alignedStart = (offset / blockSize) * blockSize
	end := offset + int64(length)
	alignedEnd := ((end + blockSize - 1) / blockSize) * blockSize
	return alignedStart, alignedEnd - alignedStart
}
