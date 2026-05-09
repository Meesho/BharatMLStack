//go:build !linux
// +build !linux

package ssdio

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type SSDWriter struct {
	dirPath     string
	swapShardId int

	filePath   string
	file       *os.File
	fileOffset int64
	capacity   int64

	uploadChan   chan string
	nextFilePath string
	nextFile     *os.File
}

func NewSSDWriter(capacity int, dirPath string, shardId int) (*SSDWriter, error) {
	capacityAligned := int64(capacity) // no strict alignment needed

	path := fmt.Sprintf("%s/%s/swap_shard_%d.bin",
		dirPath,
		time.Now().Format("2006-01-02-15-04-05"),
		shardId,
	)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	return &SSDWriter{
		dirPath:     dirPath,
		swapShardId: shardId,
		filePath:    path,
		file:        f,
		fileOffset:  0,
		capacity:    capacityAligned,
		uploadChan:  make(chan string, 1000),
	}, nil
}

func (w *SSDWriter) Write(p []byte) (int, error) {
	size := int64(len(p))

	if w.fileOffset+size > w.capacity {
		old := w.filePath
		if err := w.swapFiles(); err != nil {
			return 0, err
		}
		w.uploadChan <- old
	}

	// Pre-create next file at 90%
	if w.nextFile == nil && w.fileOffset+size >= int64(float64(w.capacity)*0.9) {
		if err := w.createNewFile(); err != nil {
			return 0, err
		}
	}

	n, err := w.file.WriteAt(p, w.fileOffset)
	if err != nil {
		return 0, err
	}
	w.fileOffset += int64(n)
	return n, nil
}

func (w *SSDWriter) createNewFile() error {
	path := fmt.Sprintf("%s/%s/swap_shard_%d.bin",
		w.dirPath,
		time.Now().Format("2006-01-02-15-04-05"),
		w.swapShardId,
	)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	w.nextFile = f
	w.nextFilePath = path
	return nil
}

func (w *SSDWriter) swapFiles() error {
	if w.file != nil {
		_ = w.file.Close()
	}

	if w.nextFile == nil || w.nextFilePath == "" {
		return fmt.Errorf("next file is not set")
	}

	w.file = w.nextFile
	w.filePath = w.nextFilePath
	w.fileOffset = 0

	w.nextFile = nil
	w.nextFilePath = ""
	return nil
}
