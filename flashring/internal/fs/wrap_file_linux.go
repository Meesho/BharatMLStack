//go:build linux
// +build linux

package fs

import (
	"fmt"
	"os"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/internal/iouring"
	"github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
	"golang.org/x/sys/unix"
)

// WrapAppendFile is a ring-buffer file: pwrite advances PhysicalWriteOffset
// linearly until MaxFileSize, at which point the offset wraps to 0 and the
// file is reused as a circular log. Reads remain valid only inside the live
// region defined by PhysicalStartOffset / PhysicalWriteOffset (see
// isValidReadRegion). Sized so callers manage GC by punching holes via
// TrimHead.
//
// wrapped, PhysicalStartOffset, LogicalCurrentOffset, and PhysicalWriteOffset
// use sync/atomic types because the async flushConsumer goroutine writes
// them via PwriteBatch while test/read paths (isValidReadRegion, Pread,
// waitFor polling in tests) read them concurrently. Aligned int64 reads on
// x86-64 are atomic in practice, but the Go memory model requires atomic
// access to avoid UB and the race detector correctly flags plain accesses.
type WrapAppendFile struct {
	WriteDirectIO        bool
	ReadDirectIO         bool
	wrapped              atomic.Bool
	blockSize            int
	WriteFd              int                    // write file descriptor
	ReadFd               int                    // read file descriptor
	MaxFileSize          int64                  // max file size in bytes
	FilePunchHoleSize    int64                  // file punch hole size in bytes
	PhysicalStartOffset  atomic.Int64           // physical start offset in bytes
	LogicalCurrentOffset atomic.Int64           // file current size in bytes
	PhysicalWriteOffset  atomic.Int64           // file current physical offset in bytes
	WriteFile            *os.File               // write file
	ReadFile             *os.File               // read file
	Stat                 *Stat                  // file statistics
	WriteRing            *iouring.IoUringWriter // io_uring writer for batched writes
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
	// atomic.Int64 / atomic.Bool fields default to zero — no need to
	// initialise them explicitly.
	return &WrapAppendFile{
		WriteDirectIO:     wDirectIO,
		ReadDirectIO:      rDirectIO,
		blockSize:         blockSize,
		WriteFd:           writeFd,
		ReadFd:            readFd,
		WriteFile:         writeFile,
		ReadFile:          readFile,
		MaxFileSize:       maxFileSize,
		FilePunchHoleSize: filePunchHoleSize,
		Stat:              &Stat{},
	}, nil
}

// Pwrite writes buf at PhysicalWriteOffset and advances the offset. When the
// next offset would meet or exceed MaxFileSize, the offset wraps to 0 and the
// file enters wrapped mode — subsequent reads in the now-overwritten region
// will fail isValidReadRegion. The caller must keep PhysicalStartOffset
// advancing via TrimHead so that live data is not run over by the writer.
// In O_DIRECT mode buf must be block-aligned or returns ErrBufNoAlign.
func (r *WrapAppendFile) Pwrite(buf []byte) (currentPhysicalOffset int64, err error) {
	if r.WriteDirectIO {
		if !isAlignedBuffer(buf, r.blockSize) {
			return 0, ErrBufNoAlign
		}
	}
	var startTime = time.Now()
	n, err := syscall.Pwrite(r.WriteFd, buf, r.PhysicalWriteOffset.Load())
	metrics.Timing(metrics.KEY_PWRITE_LATENCY, time.Since(startTime), []string{})
	if err != nil {
		return 0, err
	}

	newOff := r.PhysicalWriteOffset.Add(int64(n))
	if newOff >= r.MaxFileSize {
		r.wrapped.Store(true)
		r.PhysicalWriteOffset.Store(0)
	}
	r.LogicalCurrentOffset.Add(int64(n))
	r.Stat.WriteCount.Add(1)

	return r.PhysicalWriteOffset.Load(), nil
}

// PwriteBatch writes a large buffer in chunkSize pieces via io_uring.
// Chunks are submitted in sub-batches that fit within the ring's SQ depth,
// so arbitrarily large buffers work regardless of ring size.
// Returns total bytes written and the final PhysicalWriteOffset.
//
// The write pointer always ends len(buf) past where it started, on error
// too. Readers locate a memtable at memId*capacity, so a failed flush must
// still consume its slot: rolling the pointer back would land every later
// memtable one slot early. The failed slot's bytes are undefined and reads
// of it miss on CRC/key mismatch.
func (r *WrapAppendFile) PwriteBatch(buf []byte, chunkSize int) (totalWritten int, fileOffset int64, err error) {
	written := 0
	defer func() {
		if err != nil {
			r.LogicalCurrentOffset.Add(int64(written - totalWritten))
			r.AdvanceWriteOffset(int64(len(buf) - written))
			fileOffset = r.PhysicalWriteOffset.Load()
		}
	}()

	if r.WriteDirectIO {
		if !isAlignedBuffer(buf, r.blockSize) {
			return 0, 0, ErrBufNoAlign
		}
	}

	// Maximum SQEs per submission -- capped to ring depth.
	maxPerBatch := r.WriteRing.MaxBatchSize()

	for written < len(buf) {
		// Build a sub-batch that fits within the ring.
		var bufs [][]byte
		var offsets []uint64

		for i := 0; i < maxPerBatch && written < len(buf); i++ {
			end := written + chunkSize
			if end > len(buf) {
				end = len(buf)
			}
			bufs = append(bufs, buf[written:end])
			offsets = append(offsets, uint64(r.PhysicalWriteOffset.Load()))

			// Advance write offset, handle ring-buffer wrap
			newOff := r.PhysicalWriteOffset.Add(int64(end - written))
			if newOff >= r.MaxFileSize {
				r.wrapped.Store(true)
				r.PhysicalWriteOffset.Store(0)
			}
			written = end
		}

		results, serr := r.WriteRing.SubmitWriteBatch(r.WriteFd, bufs, offsets)
		if serr != nil {
			return totalWritten, r.PhysicalWriteOffset.Load(), serr
		}

		// Detect short writes. Each SQE was submitted with a specific
		// chunk length; a successful CQE reporting fewer bytes means
		// the file was left with an unwritten gap at the tail of that
		// chunk. Retry the residual tail via syscall.Pwrite (blocking,
		// but rare — O_DIRECT usually returns exactly the requested
		// length or errors). Only after MAX_SHORT_WRITE_RETRIES do we
		// give up and fail the batch.
		const maxShortWriteRetries = 3
		for i, n := range results {
			if n != len(bufs[i]) {
				// Retry the missing tail bytes. Compute the offset of
				// the unwritten portion and pwrite until either the
				// full chunk is on disk or retries exhaust.
				residual := bufs[i][n:]
				chunkOffset := int64(offsets[i]) + int64(n)
				retried := n
				success := false
				for attempt := 0; attempt < maxShortWriteRetries && len(residual) > 0; attempt++ {
					wn, werr := syscall.Pwrite(r.WriteFd, residual, chunkOffset)
					if werr != nil {
						break
					}
					if wn <= 0 {
						break
					}
					retried += wn
					chunkOffset += int64(wn)
					residual = residual[wn:]
					if len(residual) == 0 {
						success = true
						break
					}
				}
				if !success {
					return totalWritten, r.PhysicalWriteOffset.Load(),
						fmt.Errorf("io_uring pwrite short write on chunk %d: wrote %d/%d bytes after %d retries",
							i, retried, len(bufs[i]), maxShortWriteRetries)
				}
				// Retry succeeded — count the FULL chunk as written.
				totalWritten += len(bufs[i])
				r.LogicalCurrentOffset.Add(int64(len(bufs[i])))
				r.Stat.WriteCount.Add(1)
				continue
			}
			totalWritten += n
			r.LogicalCurrentOffset.Add(int64(n))
			r.Stat.WriteCount.Add(1)
		}
	}

	return totalWritten, r.PhysicalWriteOffset.Load(), nil
}

// AdvanceWriteOffset moves the write pointer forward by n bytes without
// writing any data. Used to skip over uninitialized regions in staggered
// memtables so that the memId*capacity file layout is preserved.
func (r *WrapAppendFile) AdvanceWriteOffset(n int64) {
	newOff := r.PhysicalWriteOffset.Add(n)
	if newOff >= r.MaxFileSize {
		r.wrapped.Store(true)
		r.PhysicalWriteOffset.Store(0)
	}
	r.LogicalCurrentOffset.Add(n)
}

func (r *WrapAppendFile) TrimHeadIfNeeded() bool {
	if r.wrapped.Load() && r.PhysicalWriteOffset.Load() == r.PhysicalStartOffset.Load() {
		return true
	}
	return false
}

// isValidReadRegion checks whether [physOffset, physEnd) falls entirely within
// the ring's live data.
//
// When wrapped, the writer and start pointers chase each other around
// [0, MaxFileSize). Three sub-cases:
//
//	W == S  →  ring is full (writer just caught up); entire file is valid.
//	W <  S  →  two valid segments: [S, MaxFileSize) and [0, W).
//	W >  S  →  contiguous [S, W); the region [W, MaxFileSize) was punched
//	           by the TrimHead that sent S past W's position.
func (r *WrapAppendFile) isValidReadRegion(physOffset, physEnd int64) bool {
	S := r.PhysicalStartOffset.Load()
	W := r.PhysicalWriteOffset.Load()
	if !r.wrapped.Load() {
		return physOffset >= S && physEnd <= W
	}
	if W == S {
		return physEnd <= r.MaxFileSize
	}
	if W < S {
		return (physOffset >= S && physEnd <= r.MaxFileSize) ||
			(physOffset >= 0 && physEnd <= W)
	}
	return physOffset >= S && physEnd <= W
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

	physOffset := fileOffset
	if r.wrapped.Load() {
		physOffset = fileOffset % r.MaxFileSize
	}
	physEnd := physOffset + int64(len(buf))

	if !r.isValidReadRegion(physOffset, physEnd) {
		return 0, ErrFileOffsetOutOfRange
	}

	var startTime = time.Now()
	n, err := syscall.Pread(r.ReadFd, buf, physOffset)
	metrics.Timing(metrics.KEY_PREAD_LATENCY, time.Since(startTime), []string{})
	if err != nil {
		return 0, err
	}
	r.Stat.ReadCount.Add(1)
	return int32(n), nil
}

// ValidateReadOffset checks the read window and wraps the offset for ring-buffer
// files. Returns the physical file offset to use, or an error.
// Mirrors the validation logic in Pread so callers that use the batched
// io_uring path get identical safety checks.
func (r *WrapAppendFile) ValidateReadOffset(fileOffset int64, bufLen int) (int64, error) {
	if r.ReadDirectIO {
		if !isAlignedOffset(fileOffset, r.blockSize) {
			return 0, ErrOffsetNotAligned
		}
	}

	physOffset := fileOffset
	if r.wrapped.Load() {
		physOffset = fileOffset % r.MaxFileSize
	}
	physEnd := physOffset + int64(bufLen)

	if !r.isValidReadRegion(physOffset, physEnd) {
		return 0, ErrFileOffsetOutOfRange
	}

	return physOffset, nil
}

// TrimHead punches a FilePunchHoleSize-byte hole at PhysicalStartOffset
// using fallocate(FALLOC_FL_PUNCH_HOLE|FALLOC_FL_KEEP_SIZE) — the file's
// logical size is unchanged but the underlying blocks are released back to
// the filesystem. PhysicalStartOffset advances by FilePunchHoleSize and
// wraps to 0 when it crosses MaxFileSize. Required to be called periodically
// in lockstep with Pwrite to keep the live region from being overrun.
func (r *WrapAppendFile) TrimHead() (err error) {
	var startTime = time.Now()
	startOff := r.PhysicalStartOffset.Load()
	if r.WriteDirectIO {
		if !isAlignedOffset(startOff, r.blockSize) {
			return ErrOffsetNotAligned
		}
	}
	err = unix.Fallocate(r.WriteFd, FALLOC_FL_PUNCH_HOLE|FALLOC_FL_KEEP_SIZE, startOff, int64(r.FilePunchHoleSize))
	if err != nil {
		return err
	}
	newStart := r.PhysicalStartOffset.Add(int64(r.FilePunchHoleSize))
	if newStart >= r.MaxFileSize {
		r.PhysicalStartOffset.Store(0)
	}
	r.Stat.PunchHoleCount.Add(1)
	metrics.Incr(metrics.KEY_PUNCH_HOLE_COUNT, []string{})
	metrics.Timing(metrics.KEY_TRIM_HEAD_LATENCY, time.Since(startTime), []string{})
	return nil
}

// Close closes both file descriptors and deletes the underlying file from
// disk. Destructive — all cached data is lost. WrapAppendFile is not
// designed for persistence across process restarts; the on-disk file is
// scratch space that lives for the lifetime of the cache.
func (r *WrapAppendFile) Close() {
	// Close through the *os.File, not the raw fd: the File's finalizer
	// would otherwise close the same number again, possibly after the
	// kernel has handed it to another file.
	r.WriteFile.Close()
	r.ReadFile.Close()
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
