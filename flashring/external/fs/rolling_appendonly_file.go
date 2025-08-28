//go:build linux
// +build linux

package fs

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
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

func NewRollingAppendFile(config FileConfig) (*RollingAppendFile, error) {
	filename := config.Filename
	maxFileSize := config.MaxFileSize
	filePunchHoleSize := config.FilePunchHoleSize

	writeFd, writeFile, wDirectIO, err := createAppendOnlyWriteFileDescriptor(filename)
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
	n, err := syscall.Pwrite(r.WriteFd, buf, r.CurrentPhysicalOffset)
	if err != nil {
		return 0, err
	}
	r.CurrentPhysicalOffset += int64(n)
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
