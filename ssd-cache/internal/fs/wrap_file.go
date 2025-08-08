//go:build linux
// +build linux

package fs

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

type WrapAppendFile struct {
	WriteDirectIO        bool
	ReadDirectIO         bool
	wrapped              bool
	blockSize            int
	WriteFd              int      // write file descriptor
	ReadFd               int      // read file descriptor
	MaxFileSize          int64    // max file size in bytes
	FilePunchHoleSize    int64    // file punch hole size in bytes
	PhysicalStartOffset  int64    // physical start offset in bytes
	LogicalCurrentOffset int64    // file current size in bytes
	PhysicalWriteOffset  int64    // file current physical offset in bytes
	WriteFile            *os.File // write file
	ReadFile             *os.File // read file
	Stat                 *Stat    // file statistics
}

func NewWrapAppendFile(config FileConfig) (*WrapAppendFile, error) {
	filename := config.Filename
	maxFileSize := config.MaxFileSize
	filePunchHoleSize := config.FilePunchHoleSize

	writeFd, writeFile, wDirectIO, err := createPreAllocatedWriteFileDescriptor(filename, maxFileSize)
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
	return &WrapAppendFile{
		WriteDirectIO:        wDirectIO,
		ReadDirectIO:         rDirectIO,
		blockSize:            blockSize,
		WriteFd:              writeFd,
		ReadFd:               readFd,
		WriteFile:            writeFile,
		ReadFile:             readFile,
		MaxFileSize:          maxFileSize,
		FilePunchHoleSize:    filePunchHoleSize,
		PhysicalStartOffset:  0,
		LogicalCurrentOffset: 0,
		PhysicalWriteOffset:  0,
		Stat: &Stat{
			WriteCount:         0,
			ReadCount:          0,
			PunchHoleCount:     0,
			CurrentLogicalSize: 0,
		},
	}, nil
}

func (r *WrapAppendFile) Pwrite(buf []byte) (currentPhysicalOffset int64, err error) {
	if r.WriteDirectIO {
		if !isAlignedBuffer(buf, r.blockSize) {
			return 0, ErrBufNoAlign
		}
	}
	n, err := syscall.Pwrite(r.WriteFd, buf, r.PhysicalWriteOffset)
	if err != nil {
		return 0, err
	}
	r.PhysicalWriteOffset += int64(n)
	if r.PhysicalWriteOffset >= r.MaxFileSize {
		r.wrapped = true
		r.PhysicalWriteOffset = r.PhysicalStartOffset
	}
	r.LogicalCurrentOffset += int64(n)
	r.Stat.WriteCount++
	return r.PhysicalWriteOffset, nil
}

func (r *WrapAppendFile) TrimHeadIfNeeded() bool {
	if r.wrapped && r.PhysicalWriteOffset == r.PhysicalStartOffset {
		return true
	}
	return false
}

func (r *WrapAppendFile) Pread(fileOffset int64, buf []byte) (int32, error) {
	if r.ReadDirectIO {
		if !isAlignedOffset(fileOffset, r.blockSize) {
			return 0, ErrOffsetNotAligned
		}
		if !isAlignedBuffer(buf, r.blockSize) {
			return 0, ErrBufNoAlign
		}
	}

	// Validate read window depending on wrap state
	readEnd := fileOffset + int64(len(buf))
	valid := false

	if !r.wrapped {
		// Single valid region: [PhysicalStartOffset, PhysicalWriteOffset)
		valid = fileOffset >= r.PhysicalStartOffset && readEnd <= r.PhysicalWriteOffset
	} else {
		// Two valid regions:
		// 1. [PhysicalStartOffset, MaxFileSize)
		// 2. [0, PhysicalWriteOffset)
		fileOffset = fileOffset % r.MaxFileSize
		readEnd = readEnd % r.MaxFileSize
		if fileOffset >= r.PhysicalStartOffset {
			valid = readEnd <= r.MaxFileSize
		} else {
			valid = readEnd <= r.PhysicalWriteOffset
		}
	}
	if !valid {
		return 0, ErrFileOffsetOutOfRange
	}

	n, err := syscall.Pread(r.ReadFd, buf, fileOffset)
	if err != nil {
		return 0, err
	}
	r.Stat.ReadCount++
	return int32(n), nil
}

func (r *WrapAppendFile) TrimHead() (err error) {
	if r.WriteDirectIO {
		if !isAlignedOffset(r.PhysicalStartOffset, r.blockSize) {
			return ErrOffsetNotAligned
		}
	}
	err = unix.Fallocate(r.WriteFd, FALLOC_FL_PUNCH_HOLE|FALLOC_FL_KEEP_SIZE, r.PhysicalStartOffset, int64(r.FilePunchHoleSize))
	if err != nil {
		return err
	}
	r.PhysicalStartOffset += int64(r.FilePunchHoleSize)
	if r.PhysicalStartOffset >= r.MaxFileSize {
		r.PhysicalStartOffset = 0
	}
	r.Stat.PunchHoleCount++
	return nil
}

func (r *WrapAppendFile) Close() {
	syscall.Close(r.WriteFd)
	syscall.Close(r.ReadFd)
	os.Remove(r.WriteFile.Name())
	os.Remove(r.ReadFile.Name())
}
