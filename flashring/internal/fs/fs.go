//go:build linux
// +build linux

package fs

import (
	"errors"
	"fmt"
	"os"
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
)

type Stat struct {
	WriteCount         int64
	ReadCount          int64
	PunchHoleCount     int64
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

func createAppendOnlyWriteFileDescriptor(filename string) (int, *os.File, bool, error) {

	// Open file with DIRECT_IO, WRITE_ONLY, CREAT flags
	flags := O_DIRECT | O_WRONLY | O_CREAT | O_DSYNC
	fd, err := syscall.Open(filename, flags, FILE_MODE)
	if err != nil {
		// If DIRECT_IO is not supported, fall back to regular flags
		log.Warn().Msgf("DIRECT_IO not supported, falling back to regular flags: %v", err)
		flags = O_WRONLY | O_CREAT | O_DSYNC
		fd, err = syscall.Open(filename, flags, FILE_MODE)
		if err != nil {
			return 0, nil, false, err
		}
	}
	file := os.NewFile(uintptr(fd), filename)
	if file == nil {
		return 0, nil, false, fmt.Errorf("failed to create file from fd")
	}

	return fd, file, true, nil
}

func createPreAllocatedWriteFileDescriptor(filename string, maxFileSize int64) (int, *os.File, bool, error) {
	// flags := O_DIRECT | O_WRONLY | O_CREAT | O_DSYNC
	flags := O_DIRECT | O_WRONLY | O_CREAT
	fd, err := syscall.Open(filename, flags, FILE_MODE)
	if err != nil {
		log.Warn().Msgf("DIRECT_IO not supported, falling back to regular flags: %v", err)
		// flags = O_WRONLY | O_CREAT | O_DSYNC
		flags = O_WRONLY | O_CREAT
		fd, err = syscall.Open(filename, flags, FILE_MODE)
		if err != nil {
			return 0, nil, false, err
		}
	}

	// Preallocate file space
	err = unix.Fallocate(fd, 0, 0, maxFileSize)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fallocate file")
		syscall.Close(fd)
		return 0, nil, false, err
	}

	file := os.NewFile(uintptr(fd), filename)
	if file == nil {
		return 0, nil, false, fmt.Errorf("failed to create file from fd")
	}

	return fd, file, true, nil
}

func createReadFileDescriptor(filename string) (int, *os.File, bool, error) {
	flags := O_DIRECT | O_RDONLY
	fd, err := syscall.Open(filename, flags, 0)
	if err != nil {
		return 0, nil, false, err
	}
	file := os.NewFile(uintptr(fd), filename)
	if file == nil {
		return 0, nil, false, fmt.Errorf("failed to create file from fd")
	}

	return fd, file, true, nil
}

// isAligned checks if the buffer is aligned to the block size
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
