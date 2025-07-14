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

type RollingAppendFile struct {
	WriteDirectIO         bool
	ReadDirectIO          bool
	blockSize             int
	WriteFd               int      // write file descriptor
	ReadFd                int      // read file descriptor
	MaxFileSize           int64    // max file size in bytes
	FilePunchHoleSize     int64    // file punch hole size in bytes
	LogicalStartOffset    int64    // logical start offset in bytes
	CurrentLogicalOffset  int64    // file current size in bytes
	CurrentPhysicalOffset int64    // file current physical offset in bytes
	WriteFile             *os.File // write file
	ReadFile              *os.File // read file
	Stat                  *Stat    // file statistics
}

type Stat struct {
	WriteCount         int64
	ReadCount          int64
	PunchHoleCount     int64
	CurrentLogicalSize int64
}

type RAFileConfig struct {
	Filename          string
	MaxFileSize       int64
	FilePunchHoleSize int64
	BlockSize         int
}

func createWriteFileDescriptor(filename string) (int, *os.File, bool, error) {

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

func NewRollingAppendFile(config RAFileConfig) (*RollingAppendFile, error) {
	filename := config.Filename
	maxFileSize := config.MaxFileSize
	filePunchHoleSize := config.FilePunchHoleSize

	writeFd, writeFile, wDirectIO, err := createWriteFileDescriptor(filename)
	if err != nil {
		return nil, err
	}
	readFd, readFile, rDirectIO, err := createReadFileDescriptor(filename)
	if err != nil {
		return nil, err
	}
	blockSize := config.BlockSize
	if blockSize == 0 {
		blockSize = BLOCK_SIZE
	}
	return &RollingAppendFile{
		WriteDirectIO:         wDirectIO,
		ReadDirectIO:          rDirectIO,
		blockSize:             blockSize,
		WriteFd:               writeFd,
		ReadFd:                readFd,
		WriteFile:             writeFile,
		ReadFile:              readFile,
		MaxFileSize:           maxFileSize,
		FilePunchHoleSize:     filePunchHoleSize,
		LogicalStartOffset:    0,
		CurrentLogicalOffset:  0,
		CurrentPhysicalOffset: 0,
		Stat: &Stat{
			WriteCount:         0,
			ReadCount:          0,
			PunchHoleCount:     0,
			CurrentLogicalSize: 0,
		},
	}, nil
}

func (r *RollingAppendFile) Pwrite(buf []byte) (currentPhysicalOffset int64, err error) {
	if r.CurrentLogicalOffset+int64(len(buf)) > r.MaxFileSize {
		return 0, ErrFileSizeExceeded
	}
	if r.WriteDirectIO {
		if !isAlignedBuffer(buf, r.blockSize) {
			return 0, ErrBufNoAlign
		}
	}
	syscall.Pwrite(r.WriteFd, buf, r.CurrentPhysicalOffset)
	r.CurrentPhysicalOffset += int64(len(buf))
	r.Stat.WriteCount++
	return r.CurrentPhysicalOffset, nil
}

func (r *RollingAppendFile) Pread(fileOffset int64, buf []byte) (n int32, err error) {
	if fileOffset < r.LogicalStartOffset || fileOffset+int64(len(buf)) > r.CurrentPhysicalOffset {
		return 0, ErrFileOffsetOutOfRange
	}
	if r.ReadDirectIO {
		if !isAlignedOffset(fileOffset, r.blockSize) {
			return 0, ErrOffsetNotAligned
		}
		if !isAlignedBuffer(buf, r.blockSize) {
			return 0, ErrBufNoAlign
		}
	}
	syscall.Pread(r.ReadFd, buf, fileOffset)
	r.Stat.ReadCount++
	return int32(len(buf)), nil
}

func (r *RollingAppendFile) TrimHead() (err error) {
	if r.WriteDirectIO {
		if !isAlignedOffset(r.LogicalStartOffset, r.blockSize) {
			return ErrOffsetNotAligned
		}
	}
	err = unix.Fallocate(r.WriteFd, FALLOC_FL_PUNCH_HOLE|FALLOC_FL_KEEP_SIZE, r.LogicalStartOffset, int64(r.FilePunchHoleSize))
	if err != nil {
		return err
	}
	r.LogicalStartOffset += int64(r.FilePunchHoleSize)
	r.CurrentLogicalOffset -= int64(r.FilePunchHoleSize)
	r.Stat.PunchHoleCount++
	return nil
}

func (r *RollingAppendFile) Close() {
	syscall.Close(r.WriteFd)
	syscall.Close(r.ReadFd)
	os.Remove(r.WriteFile.Name())
	os.Remove(r.ReadFile.Name())
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
