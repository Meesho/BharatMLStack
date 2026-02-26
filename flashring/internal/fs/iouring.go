//go:build linux
// +build linux

// Package fs provides a minimal io_uring implementation using raw syscalls.
// No external dependencies beyond golang.org/x/sys/unix are needed.
// Compatible with Go 1.24+ (no go:linkname usage).
package fs

import (
	"fmt"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
	"golang.org/x/sys/unix"
)

// -----------------------------------------------------------------------
// io_uring syscall numbers (amd64)
// -----------------------------------------------------------------------

const (
	sysIOUringSetup    = 425
	sysIOUringEnter    = 426
	sysIOUringRegister = 427
)

// -----------------------------------------------------------------------
// io_uring constants
// -----------------------------------------------------------------------

const (
	// Setup flags
	iouringSetupSQPoll = 1 << 1

	// Enter flags
	iouringEnterGetEvents = 1 << 0
	iouringEnterSQWakeup  = 1 << 1

	// SQ flags (read from kernel-shared memory)
	iouringSQNeedWakeup = 1 << 0

	// Opcodes
	iouringOpNop   = 0
	iouringOpRead  = 22
	iouringOpWrite = 23

	// offsets for mmap
	iouringOffSQRing = 0
	iouringOffCQRing = 0x8000000
	iouringOffSQEs   = 0x10000000
)

// -----------------------------------------------------------------------
// io_uring kernel structures (must match kernel ABI exactly)
// -----------------------------------------------------------------------

// ioUringSqe is the 64-byte submission queue entry.
type ioUringSqe struct {
	Opcode   uint8
	Flags    uint8
	IoPrio   uint16
	Fd       int32
	Off      uint64 // union: off / addr2
	Addr     uint64 // union: addr / splice_off_in
	Len      uint32
	OpFlags  uint32 // union: rw_flags, etc.
	UserData uint64
	BufIndex uint16 // union: buf_index / buf_group
	_        uint16 // personality
	_        int32  // splice_fd_in / file_index
	_        uint64 // addr3
	_        uint64 // __pad2[0]
}

// ioUringCqe is the 16-byte completion queue entry.
type ioUringCqe struct {
	UserData uint64
	Res      int32
	Flags    uint32
}

// ioUringParams is passed to io_uring_setup.
type ioUringParams struct {
	SqEntries    uint32
	CqEntries    uint32
	Flags        uint32
	SqThreadCPU  uint32
	SqThreadIdle uint32
	Features     uint32
	WqFd         uint32
	Resv         [3]uint32
	SqOff        ioUringSqringOffsets
	CqOff        ioUringCqringOffsets
}

type ioUringSqringOffsets struct {
	Head        uint32
	Tail        uint32
	RingMask    uint32
	RingEntries uint32
	Flags       uint32
	Dropped     uint32
	Array       uint32
	Resv1       uint32
	Resv2       uint64
}

type ioUringCqringOffsets struct {
	Head        uint32
	Tail        uint32
	RingMask    uint32
	RingEntries uint32
	Overflow    uint32
	Cqes        uint32
	Flags       uint32
	Resv1       uint32
	Resv2       uint64
}

// -----------------------------------------------------------------------
// IoUring is the main ring handle
// -----------------------------------------------------------------------

// IoUring wraps a single io_uring instance with SQ/CQ ring mappings.
type IoUring struct {
	fd int

	// SQ ring mapped memory
	sqRingPtr  []byte
	sqMask     uint32
	sqEntries  uint32
	sqHead     *uint32 // kernel-updated
	sqTail     *uint32 // user-updated
	sqFlags    *uint32 // kernel-updated (NEED_WAKEUP etc.)
	sqArray    unsafe.Pointer
	sqeTail    uint32 // local tracking of next SQE slot
	sqeHead    uint32 // local tracking of submitted SQEs
	sqesMmap   []byte
	sqesBase   unsafe.Pointer // base pointer to SQE array
	sqRingSz   int
	cqRingSz   int
	sqesSz     int
	singleMmap bool

	// CQ ring mapped memory
	cqRingPtr []byte
	cqMask    uint32
	cqEntries uint32
	cqHead    *uint32 // user-updated
	cqTail    *uint32 // kernel-updated
	cqesBase  unsafe.Pointer

	// Setup flags
	flags uint32

	// Mutex for concurrent SQE submission from multiple goroutines
	mu sync.Mutex

	// Diagnostic counter -- limits debug output to first N failures
	debugCount int
}

// NewIoUring creates a new io_uring instance with the given queue depth.
// flags can be 0 for normal mode.
func NewIoUring(entries uint32, flags uint32) (*IoUring, error) {
	var params ioUringParams
	params.Flags = flags

	fd, _, errno := syscall.Syscall(sysIOUringSetup, uintptr(entries), uintptr(unsafe.Pointer(&params)), 0)
	if errno != 0 {
		return nil, fmt.Errorf("io_uring_setup failed: %w", errno)
	}

	ring := &IoUring{
		fd:    int(fd),
		flags: params.Flags,
	}

	if err := ring.mapRings(&params); err != nil {
		syscall.Close(ring.fd)
		return nil, err
	}

	return ring, nil
}

func (r *IoUring) mapRings(p *ioUringParams) error {
	sqOff := &p.SqOff
	cqOff := &p.CqOff

	// Calculate SQ ring size
	r.sqRingSz = int(sqOff.Array + p.SqEntries*4) // Array + entries*sizeof(uint32)

	// Calculate CQ ring size
	r.cqRingSz = int(cqOff.Cqes + p.CqEntries*uint32(unsafe.Sizeof(ioUringCqe{})))

	// Check if kernel supports single mmap for both rings
	r.singleMmap = (p.Features & 1) != 0 // IORING_FEAT_SINGLE_MMAP = 1
	if r.singleMmap {
		if r.cqRingSz > r.sqRingSz {
			r.sqRingSz = r.cqRingSz
		}
	}

	// Map SQ ring
	var err error
	r.sqRingPtr, err = unix.Mmap(r.fd, iouringOffSQRing, r.sqRingSz,
		unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED|unix.MAP_POPULATE)
	if err != nil {
		return fmt.Errorf("mmap SQ ring: %w", err)
	}

	// Map CQ ring (same or separate mapping)
	if r.singleMmap {
		r.cqRingPtr = r.sqRingPtr
	} else {
		r.cqRingPtr, err = unix.Mmap(r.fd, iouringOffCQRing, r.cqRingSz,
			unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED|unix.MAP_POPULATE)
		if err != nil {
			unix.Munmap(r.sqRingPtr)
			return fmt.Errorf("mmap CQ ring: %w", err)
		}
	}

	// Map SQE array
	r.sqesSz = int(p.SqEntries) * int(unsafe.Sizeof(ioUringSqe{}))
	r.sqesMmap, err = unix.Mmap(r.fd, iouringOffSQEs, r.sqesSz,
		unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED|unix.MAP_POPULATE)
	if err != nil {
		unix.Munmap(r.sqRingPtr)
		if !r.singleMmap {
			unix.Munmap(r.cqRingPtr)
		}
		return fmt.Errorf("mmap SQEs: %w", err)
	}
	r.sqesBase = unsafe.Pointer(&r.sqesMmap[0])

	// Set up SQ ring pointers
	sqBase := unsafe.Pointer(&r.sqRingPtr[0])
	r.sqHead = (*uint32)(unsafe.Add(sqBase, sqOff.Head))
	r.sqTail = (*uint32)(unsafe.Add(sqBase, sqOff.Tail))
	r.sqFlags = (*uint32)(unsafe.Add(sqBase, sqOff.Flags))
	r.sqMask = *(*uint32)(unsafe.Add(sqBase, sqOff.RingMask))
	r.sqEntries = *(*uint32)(unsafe.Add(sqBase, sqOff.RingEntries))
	r.sqArray = unsafe.Add(sqBase, sqOff.Array)

	// Set up CQ ring pointers
	cqBase := unsafe.Pointer(&r.cqRingPtr[0])
	r.cqHead = (*uint32)(unsafe.Add(cqBase, cqOff.Head))
	r.cqTail = (*uint32)(unsafe.Add(cqBase, cqOff.Tail))
	r.cqMask = *(*uint32)(unsafe.Add(cqBase, cqOff.RingMask))
	r.cqEntries = *(*uint32)(unsafe.Add(cqBase, cqOff.RingEntries))
	r.cqesBase = unsafe.Add(cqBase, cqOff.Cqes)

	return nil
}

// Close releases all resources associated with the ring.
func (r *IoUring) Close() {
	unix.Munmap(r.sqesMmap)
	unix.Munmap(r.sqRingPtr)
	if !r.singleMmap {
		unix.Munmap(r.cqRingPtr)
	}
	syscall.Close(r.fd)
}

// -----------------------------------------------------------------------
// SQE helpers
// -----------------------------------------------------------------------

func (r *IoUring) getSqeAt(idx uint32) *ioUringSqe {
	return (*ioUringSqe)(unsafe.Add(r.sqesBase, uintptr(idx)*unsafe.Sizeof(ioUringSqe{})))
}

func (r *IoUring) getCqeAt(idx uint32) *ioUringCqe {
	return (*ioUringCqe)(unsafe.Add(r.cqesBase, uintptr(idx)*unsafe.Sizeof(ioUringCqe{})))
}

func (r *IoUring) sqArrayAt(idx uint32) *uint32 {
	return (*uint32)(unsafe.Add(r.sqArray, uintptr(idx)*4))
}

// getSqe returns the next available SQE, or nil if the SQ is full.
func (r *IoUring) getSqe() *ioUringSqe {
	head := atomic.LoadUint32(r.sqHead)
	next := r.sqeTail + 1
	if next-head > r.sqEntries {
		return nil // SQ full
	}
	sqe := r.getSqeAt(r.sqeTail & r.sqMask)
	r.sqeTail++
	// Zero out the SQE
	*sqe = ioUringSqe{}
	return sqe
}

// flushSq flushes locally queued SQEs into the kernel-visible SQ ring.
func (r *IoUring) flushSq() uint32 {
	tail := *r.sqTail
	toSubmit := r.sqeTail - r.sqeHead
	if toSubmit == 0 {
		return tail - atomic.LoadUint32(r.sqHead)
	}
	for ; toSubmit > 0; toSubmit-- {
		*r.sqArrayAt(tail & r.sqMask) = r.sqeHead & r.sqMask
		tail++
		r.sqeHead++
	}
	atomic.StoreUint32(r.sqTail, tail)
	return tail - atomic.LoadUint32(r.sqHead)
}

// -----------------------------------------------------------------------
// Submission and completion
// -----------------------------------------------------------------------

func ioUringEnter(fd int, toSubmit, minComplete, flags uint32) (int, error) {
	ret, _, errno := syscall.Syscall6(sysIOUringEnter,
		uintptr(fd), uintptr(toSubmit), uintptr(minComplete), uintptr(flags), 0, 0)
	if errno != 0 {
		return int(ret), errno
	}
	return int(ret), nil
}

// submit flushes SQEs and calls io_uring_enter if needed.
// Retries automatically on EINTR (signal interruption).
func (r *IoUring) submit(waitNr uint32) (int, error) {
	submitted := r.flushSq()
	var flags uint32 = 0

	// If not using SQPOLL, we always need to enter
	if r.flags&iouringSetupSQPoll == 0 {
		if waitNr > 0 {
			flags |= iouringEnterGetEvents
		}
		for {
			ret, err := ioUringEnter(r.fd, submitted, waitNr, flags)
			if err == syscall.EINTR {
				continue
			}
			return ret, err
		}
	}

	// SQPOLL: only enter if kernel thread needs wakeup
	if atomic.LoadUint32(r.sqFlags)&iouringSQNeedWakeup != 0 {
		flags |= iouringEnterSQWakeup
	}
	if waitNr > 0 {
		flags |= iouringEnterGetEvents
	}
	if flags != 0 {
		for {
			ret, err := ioUringEnter(r.fd, submitted, waitNr, flags)
			if err == syscall.EINTR {
				continue
			}
			return ret, err
		}
	}
	return int(submitted), nil
}

// waitCqe waits for at least one CQE to be available and returns it.
// The caller MUST call SeenCqe after processing.
func (r *IoUring) waitCqe() (*ioUringCqe, error) {
	for {
		head := atomic.LoadUint32(r.cqHead)
		tail := atomic.LoadUint32(r.cqTail)
		if head != tail {
			cqe := r.getCqeAt(head & r.cqMask)
			return cqe, nil
		}
		// No CQE available, ask the kernel
		_, err := ioUringEnter(r.fd, 0, 1, iouringEnterGetEvents)
		if err != nil {
			if err == syscall.EINTR {
				continue // signal interrupted the syscall; retry
			}
			return nil, err
		}
	}
}

// seenCqe advances the CQ head by 1, releasing the CQE slot.
func (r *IoUring) seenCqe() {
	atomic.StoreUint32(r.cqHead, atomic.LoadUint32(r.cqHead)+1)
}

// -----------------------------------------------------------------------
// PrepRead / PrepWrite helpers
// -----------------------------------------------------------------------

func prepRead(sqe *ioUringSqe, fd int, buf []byte, offset uint64) {
	if len(buf) == 0 {
		sqe.Opcode = iouringOpNop
		return
	}
	sqe.Opcode = iouringOpRead
	sqe.Fd = int32(fd)
	sqe.Addr = uint64(uintptr(unsafe.Pointer(&buf[0])))
	sqe.Len = uint32(len(buf))
	sqe.Off = offset
}

func prepWrite(sqe *ioUringSqe, fd int, buf []byte, offset uint64) {
	if len(buf) == 0 {
		sqe.Opcode = iouringOpNop
		return
	}
	sqe.Opcode = iouringOpWrite
	sqe.Fd = int32(fd)
	sqe.Addr = uint64(uintptr(unsafe.Pointer(&buf[0])))
	sqe.Len = uint32(len(buf))
	sqe.Off = offset
}

// -----------------------------------------------------------------------
// High-level thread-safe API
// -----------------------------------------------------------------------

// SubmitRead submits a pread and waits for completion. Thread-safe.
// Returns bytes read or an error.
func (r *IoUring) SubmitRead(fd int, buf []byte, offset uint64) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}

	r.mu.Lock()

	sqe := r.getSqe()
	if sqe == nil {
		r.mu.Unlock()
		return 0, fmt.Errorf("io_uring: SQ full, no SQE available")
	}
	prepRead(sqe, fd, buf, offset)
	// Tag the SQE so we can verify the CQE belongs to this request
	sqe.UserData = offset

	submitted, err := r.submit(1)
	if err != nil {
		r.mu.Unlock()
		return 0, fmt.Errorf("io_uring_enter failed: %w", err)
	}

	cqe, err := r.waitCqe()
	if err != nil {
		r.mu.Unlock()
		return 0, fmt.Errorf("io_uring wait cqe: %w", err)
	}

	res := cqe.Res
	userData := cqe.UserData
	cqeFlags := cqe.Flags
	r.seenCqe()
	r.mu.Unlock()

	if res < 0 {
		return 0, fmt.Errorf("io_uring pread errno %d (%s), fd=%d off=%d len=%d submitted=%d ud=%d",
			-res, syscall.Errno(-res), fd, offset, len(buf), submitted, userData)
	}

	// Diagnostic: if io_uring returned 0 (EOF) or short read, compare with syscall.Pread
	if r.debugCount < 20 && int(res) != len(buf) {
		r.debugCount++
		pn, perr := syscall.Pread(fd, buf, int64(offset))
		// Also stat the fd to check file size
		var stat syscall.Stat_t
		fstatErr := syscall.Fstat(fd, &stat)
		var fsize int64
		if fstatErr == nil {
			fsize = stat.Size
		}
		fmt.Printf("[io_uring diag] fd=%d off=%d len=%d uring_res=%d uring_ud=%d uring_flags=%d "+
			"submitted=%d pread_n=%d pread_err=%v filesize=%d fstat_err=%v sqeHead=%d sqeTail=%d\n",
			fd, offset, len(buf), res, userData, cqeFlags,
			submitted, pn, perr, fsize, fstatErr, r.sqeHead, r.sqeTail)
	}

	return int(res), nil
}

// SubmitWriteBatch submits N pwrite operations in a single io_uring_enter call
// and waits for all completions. Thread-safe.
// Returns per-chunk bytes written. On error, partial results may be returned.
func (r *IoUring) SubmitWriteBatch(fd int, bufs [][]byte, offsets []uint64) ([]int, error) {
	n := len(bufs)
	if n == 0 {
		return nil, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Prepare all SQEs
	for i := 0; i < n; i++ {
		sqe := r.getSqe()
		if sqe == nil {
			return nil, fmt.Errorf("io_uring: SQ full, need %d slots but ring has %d", n, r.sqEntries)
		}
		prepWrite(sqe, fd, bufs[i], offsets[i])
		sqe.UserData = uint64(i)
	}

	// Submit all at once; kernel waits for all completions
	_, err := r.submit(uint32(n))
	if err != nil {
		return nil, fmt.Errorf("io_uring_enter: %w", err)
	}

	var startTime = time.Now()

	// Drain all CQEs (order may differ from submission)
	results := make([]int, n)
	for i := 0; i < n; i++ {
		cqe, err := r.waitCqe()
		if err != nil {
			return results, fmt.Errorf("io_uring waitCqe: %w", err)
		}
		idx := int(cqe.UserData)
		res := cqe.Res
		r.seenCqe()

		if res < 0 {
			return results, fmt.Errorf("io_uring pwrite errno %d (%s), fd=%d off=%d len=%d",
				-res, syscall.Errno(-res), fd, offsets[idx], len(bufs[idx]))
		}
		if idx >= 0 && idx < n {
			results[idx] = int(res)
		}

		metrics.Timing(metrics.KEY_PWRITE_LATENCY, time.Since(startTime), []string{})
	}

	return results, nil
}

// SubmitWrite submits a pwrite and waits for completion. Thread-safe.
// Returns bytes written or an error.
func (r *IoUring) SubmitWrite(fd int, buf []byte, offset uint64) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}

	r.mu.Lock()

	sqe := r.getSqe()
	if sqe == nil {
		r.mu.Unlock()
		return 0, fmt.Errorf("io_uring: SQ full, no SQE available")
	}
	prepWrite(sqe, fd, buf, offset)

	_, err := r.submit(1)
	if err != nil {
		r.mu.Unlock()
		return 0, fmt.Errorf("io_uring_enter failed: %w", err)
	}

	cqe, err := r.waitCqe()
	if err != nil {
		r.mu.Unlock()
		return 0, fmt.Errorf("io_uring wait cqe: %w", err)
	}

	res := cqe.Res
	r.seenCqe()
	r.mu.Unlock()

	if res < 0 {
		return 0, fmt.Errorf("io_uring pwrite failed: errno %d (%s)", -res, syscall.Errno(-res))
	}
	return int(res), nil
}
