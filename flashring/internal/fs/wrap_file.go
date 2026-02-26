//go:build linux
// +build linux

package fs

import (
	"os"
	"syscall"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
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
	WriteRing            *IoUring // optional io_uring ring for batched writes
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
	var startTime = time.Now()
	n, err := syscall.Pwrite(r.WriteFd, buf, r.PhysicalWriteOffset)
	metrics.Timing(metrics.KEY_PWRITE_LATENCY, time.Since(startTime), []string{})
	if err != nil {
		return 0, err
	}

	r.PhysicalWriteOffset += int64(n)
	if r.PhysicalWriteOffset >= r.MaxFileSize {
		r.wrapped = true
		r.PhysicalWriteOffset = r.PhysicalStartOffset
	}
	r.LogicalCurrentOffset += int64(n)

	return r.PhysicalWriteOffset, nil
}

// PwriteBatch writes a large buffer in chunkSize pieces via io_uring.
// Chunks are submitted in sub-batches that fit within the ring's SQ depth,
// so arbitrarily large buffers work regardless of ring size.
// Returns total bytes written and the final PhysicalWriteOffset.
// Requires WriteRing to be set; falls back to sequential Pwrite if nil.
func (r *WrapAppendFile) PwriteBatch(buf []byte, chunkSize int) (totalWritten int, fileOffset int64, err error) {
	if r.WriteRing == nil {
		// Fallback: sequential pwrite
		for written := 0; written < len(buf); written += chunkSize {
			end := written + chunkSize
			if end > len(buf) {
				end = len(buf)
			}
			fileOffset, err = r.Pwrite(buf[written:end])
			if err != nil {
				return written, fileOffset, err
			}
			totalWritten += end - written
		}
		return totalWritten, fileOffset, nil
	}

	if r.WriteDirectIO {
		if !isAlignedBuffer(buf, r.blockSize) {
			return 0, 0, ErrBufNoAlign
		}
	}

	// Maximum SQEs per submission -- capped to ring depth.
	maxPerBatch := int(r.WriteRing.sqEntries)

	for written := 0; written < len(buf); {
		// Build a sub-batch that fits within the ring
		var bufs [][]byte
		var offsets []uint64

		for i := 0; i < maxPerBatch && written < len(buf); i++ {
			end := written + chunkSize
			if end > len(buf) {
				end = len(buf)
			}
			bufs = append(bufs, buf[written:end])
			offsets = append(offsets, uint64(r.PhysicalWriteOffset))

			// Advance write offset, handle ring-buffer wrap
			r.PhysicalWriteOffset += int64(end - written)
			if r.PhysicalWriteOffset >= r.MaxFileSize {
				r.wrapped = true
				r.PhysicalWriteOffset = r.PhysicalStartOffset
			}
			written = end
		}

		results, serr := r.WriteRing.SubmitWriteBatch(r.WriteFd, bufs, offsets)
		if serr != nil {
			return totalWritten, r.PhysicalWriteOffset, serr
		}

		for _, n := range results {
			totalWritten += n
			r.LogicalCurrentOffset += int64(n)
			r.Stat.WriteCount++
		}
	}

	return totalWritten, r.PhysicalWriteOffset, nil
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

	var startTime = time.Now()
	n, err := syscall.Pread(r.ReadFd, buf, fileOffset)
	metrics.Timing(metrics.KEY_PREAD_LATENCY, time.Since(startTime), []string{})
	// flags := unix.RWF_HIPRI // optionally: | unix.RWF_NOWAIT
	// n, err := preadv2(r.ReadFd, buf, fileOffset, flags)
	if err != nil {
		return 0, err
	}
	r.Stat.ReadCount++
	return int32(n), nil
}

// ValidateReadOffset checks the read window and wraps the offset for ring-buffer
// files. Returns the physical file offset to use, or an error.
// Mirrors the validation logic in PreadAsync / Pread so callers that bypass
// PreadAsync (e.g. the batched io_uring path) get identical safety checks.
func (r *WrapAppendFile) ValidateReadOffset(fileOffset int64, bufLen int) (int64, error) {
	if r.ReadDirectIO {
		if !isAlignedOffset(fileOffset, r.blockSize) {
			return 0, ErrOffsetNotAligned
		}
	}

	readEnd := fileOffset + int64(bufLen)
	valid := false

	if !r.wrapped {
		valid = fileOffset >= r.PhysicalStartOffset && readEnd <= r.PhysicalWriteOffset
	} else {
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

	return fileOffset, nil
}

// PreadAsync submits a pread via io_uring and waits for completion.
// Thread-safe: multiple goroutines can call this concurrently on the same IOUringFile.
// Applies the same read-window validation and offset wrapping as Pread so that
// stale index entries (pointing past MaxFileSize) are rejected cheaply without
// hitting the kernel.
func (f *IOUringFile) PreadAsync(fileOffset int64, buf []byte) (int, error) {
	if f.ReadDirectIO {
		if !isAlignedOffset(fileOffset, f.blockSize) {
			return 0, ErrOffsetNotAligned
		}
		if !isAlignedBuffer(buf, f.blockSize) {
			return 0, ErrBufNoAlign
		}
	}

	// Validate read window and wrap offset (mirrors Pread logic exactly)
	readEnd := fileOffset + int64(len(buf))
	valid := false

	if !f.wrapped {
		// Single valid region: [PhysicalStartOffset, PhysicalWriteOffset)
		valid = fileOffset >= f.PhysicalStartOffset && readEnd <= f.PhysicalWriteOffset
	} else {
		// Ring buffer has wrapped -- map the logical offset back into [0, MaxFileSize)
		fileOffset = fileOffset % f.MaxFileSize
		readEnd = readEnd % f.MaxFileSize
		if fileOffset >= f.PhysicalStartOffset {
			valid = readEnd <= f.MaxFileSize
		} else {
			valid = readEnd <= f.PhysicalWriteOffset
		}
	}
	if !valid {
		return 0, ErrFileOffsetOutOfRange
	}

	var startTime = time.Now()
	n, err := f.ring.SubmitRead(f.ReadFd, buf, uint64(fileOffset))
	metrics.Incr(metrics.KEY_PREAD_COUNT, []string{})
	metrics.Timing(metrics.KEY_PREAD_LATENCY, time.Since(startTime), []string{})
	if err != nil {
		return 0, err
	}

	f.Stat.ReadCount++
	return n, nil
}

func (r *WrapAppendFile) TrimHead() (err error) {

	var startTime = time.Now()
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
	metrics.Incr(metrics.KEY_PUNCH_HOLE_COUNT, []string{})
	metrics.Timing(metrics.KEY_TRIM_HEAD_LATENCY, time.Since(startTime), []string{})
	return nil
}

func (r *WrapAppendFile) Close() {
	syscall.Close(r.WriteFd)
	syscall.Close(r.ReadFd)
	os.Remove(r.WriteFile.Name())
	os.Remove(r.ReadFile.Name())
}

func preadv2(fd int, buf []byte, off int64, flags int) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	n, err := unix.Preadv2(fd, [][]byte{buf}, off, flags)
	// Kernel or FS may not support preadv2/flags; fall back
	if err == unix.ENOSYS || err == unix.EOPNOTSUPP || err == unix.EINVAL {
		return unix.Pread(fd, buf, off)
	}
	return n, err
}
