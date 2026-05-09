//go:build linux
// +build linux

package ssdio

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
)

type SSDWriter struct {
	dirPath        string
	swapShardId    int
	filePath       string
	file           *os.File
	fd             int
	fileOffset     int
	capacity       int64
	uploadChan     chan string
	nextFilePath   string
	nextFile       *os.File
	nextFd         int
	nextFileOffset int
}

const directIOBlockSize = 4096 // typical; tune if your device needs different

// alignUp rounds n up to the next multiple of align (power of 2).
func alignUp(n, align int64) int64 {
	return (n + align - 1) &^ (align - 1)
}

func NewSSDWriter(capacity int, dirPath string, swapShardId int) (*SSDWriter, error) {
	capacityAligned := alignUp(int64(capacity), directIOBlockSize)
	path := fmt.Sprintf("%s/%s/swap_shard_%d.bin", dirPath, time.Now().Format("2006-01-02-15-04-05"), swapShardId)
	os.MkdirAll(filepath.Dir(path), 0o755)
	fd, err := unix.Open(
		path,
		unix.O_CREAT|unix.O_TRUNC|unix.O_WRONLY|unix.O_DIRECT|unix.O_SYNC,
		0o644,
	)
	if err != nil {
		panic(err)
	}
	// Preallocate the file so the extents are ready for direct I/O.
	if err := unix.Fallocate(fd, 0, 0, capacityAligned); err != nil {
		_ = unix.Close(fd)
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("failed to create file descriptor")
	}
	return &SSDWriter{
		dirPath:     dirPath,
		swapShardId: swapShardId,
		filePath:    path,
		file:        file,
		fd:          fd,
		fileOffset:  0,
		capacity:    capacityAligned,
		uploadChan:  make(chan string, 1000),
	}, nil
}

func (w *SSDWriter) Write(p []byte) (int, error) {
	size := len(p)
	if int64(w.fileOffset+size) > w.capacity {
		filePath := w.filePath
		err := w.swapFiles()
		if err != nil {
			return 0, err
		}
		w.uploadChan <- filePath
	}
	if w.fileOffset+size >= int(float64(w.capacity)*0.9) {
		return 0, w.createNewFile()
	}
	n, err := unix.Pwrite(w.fd, p, int64(w.fileOffset))
	if err != nil {
		return 0, err
	}
	w.fileOffset += n
	return n, nil
}

func (w *SSDWriter) createNewFile() error {
	path := fmt.Sprintf("%s/%s/swap_shard_%d.bin", w.dirPath, time.Now().Format("2006-01-02-15-04-05"), w.swapShardId)
	os.MkdirAll(filepath.Dir(path), 0o755)
	fd, err := unix.Open(
		path,
		unix.O_CREAT|unix.O_TRUNC|unix.O_WRONLY|unix.O_DIRECT|unix.O_SYNC,
		0o644,
	)
	if err != nil {
		panic(err)
	}
	// Preallocate the file so the extents are ready for direct I/O.
	if err := unix.Fallocate(fd, 0, 0, w.capacity); err != nil {
		_ = unix.Close(fd)
		return err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return fmt.Errorf("failed to create file descriptor")
	}
	w.nextFile = file
	w.nextFd = fd
	w.nextFileOffset = 0
	w.nextFilePath = path
	return nil
}

func (w *SSDWriter) swapFiles() error {
	unix.Fsync(w.fd)
	_ = unix.Close(w.fd)
	if w.nextFile == nil || w.nextFd == 0 || w.nextFileOffset != 0 || w.nextFilePath == "" {
		return fmt.Errorf("next file is not set")
	}
	w.file = w.nextFile
	w.fd = w.nextFd
	w.fileOffset = w.nextFileOffset
	w.filePath = w.nextFilePath
	w.nextFile = nil
	w.nextFd = 0
	w.nextFileOffset = 0
	w.nextFilePath = ""
	return nil
}
