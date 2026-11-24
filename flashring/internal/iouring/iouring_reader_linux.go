//go:build linux
// +build linux

package iouring

import (
	"fmt"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
)

// ReadResult holds the outcome of a single io_uring pread.
type ReadResult struct {
	N   int
	Err error
}

// batchReadRequest is a pread submitted to the batch reader.
type batchReadRequest struct {
	fd     int
	buf    []byte
	offset uint64
	done   chan ReadResult
}

var batchReqPool = sync.Pool{
	New: func() interface{} {
		return &batchReadRequest{
			done: make(chan ReadResult, 1),
		}
	},
}

// sentinelUserData is stored in the NOP SQE submitted during Close to unblock
// the completion goroutine.
const sentinelUserData = ^uint64(0)

// BatchIoUringReader collects pread requests and submits them as io_uring
// batches. Submission and completion run in separate goroutines so the submit
// path is never blocked by CQE draining:
//
//	submitLoop:    reqCh → collect batch → prep SQEs → io_uring_enter → loop
//	completeLoop:  waitCqe → dispatch result to caller → loop
//
// This eliminates the head-of-batch queueing delay where new requests had to
// wait for the entire previous batch's CQE drain before being submitted.
type BatchIoUringReader struct {
	ring     *IoUring
	reqCh    chan *batchReadRequest
	maxBatch int
	closeCh  chan struct{}
	wg       sync.WaitGroup

	// closeMu orders enqueue against Close: Close takes the write lock to
	// set closed, so every request is either queued before Close starts
	// (and is answered by Close's drain) or refused.
	closeMu sync.RWMutex
	closed  bool

	// submitLoopDone is closed by submitLoop's defer. Close waits on it
	// BEFORE walking inflight[] a second time and submitting the sentinel —
	// this eliminates the race where sentinel + pending<=0 fires while
	// submitLoop is between Store and pending.Add, leaving the request
	// unreachable after completeLoop returns.
	submitLoopDone chan struct{}

	// In-flight tracking: each SQE gets a slot index as its UserData.
	// The submit goroutine stores the request; the completion goroutine
	// reads it back when the CQE arrives.
	inflight  []atomic.Pointer[batchReadRequest]
	freeSlots chan uint32  // pool of available slot indices
	pending   atomic.Int32 // number of SQEs currently in-flight
}

// BatchIoUringConfig configures the batch reader.
type BatchIoUringConfig struct {
	RingDepth   uint32        // io_uring SQ/CQ size (default 256)
	MaxBatch    int           // max requests per batch (auto-capped to MaxInflight)
	MaxInflight int           // max SQEs in-flight across all batches; controls NVMe queue depth (default RingDepth)
	Window      time.Duration // unused in decoupled mode; kept for API compatibility
	QueueSize   int           // channel buffer size (default 1024)
	SQPoll      bool          // use IORING_SETUP_SQPOLL; kernel polls SQ, eliminating submit syscalls under load
}

// NewBatchIoUringReader creates a batch reader with its own io_uring ring
// and starts the submit + completion goroutines.
func NewBatchIoUringReader(cfg BatchIoUringConfig) (*BatchIoUringReader, error) {
	if cfg.RingDepth == 0 {
		cfg.RingDepth = 256
	}
	ringDepth := int(cfg.RingDepth)

	maxInflight := cfg.MaxInflight
	if maxInflight <= 0 || maxInflight > ringDepth {
		maxInflight = ringDepth
	}
	if cfg.MaxBatch <= 0 || cfg.MaxBatch > maxInflight {
		cfg.MaxBatch = maxInflight
	}
	if cfg.QueueSize == 0 {
		cfg.QueueSize = 1024
	}

	var flags uint32
	if cfg.SQPoll {
		flags = iouringSetupSQPoll
	}

	ring, err := NewIoUring(cfg.RingDepth, flags)
	if err != nil && cfg.SQPoll {
		// SQPOLL may fail without CAP_SYS_NICE on kernels < 5.13; fall back.
		ring, err = NewIoUring(cfg.RingDepth, 0)
	}
	if err != nil {
		return nil, fmt.Errorf("batch io_uring init: %w", err)
	}

	freeSlots := make(chan uint32, maxInflight)
	for i := 0; i < maxInflight; i++ {
		freeSlots <- uint32(i)
	}

	b := &BatchIoUringReader{
		ring:           ring,
		reqCh:          make(chan *batchReadRequest, cfg.QueueSize),
		maxBatch:       cfg.MaxBatch,
		closeCh:        make(chan struct{}),
		submitLoopDone: make(chan struct{}),
		inflight:       make([]atomic.Pointer[batchReadRequest], ringDepth),
		freeSlots:      freeSlots,
	}
	b.wg.Add(2)
	go b.submitLoop()
	go b.completeLoop()
	return b, nil
}

// Submit sends a pread request into the batch channel and blocks until the
// io_uring completion is received. Thread-safe; called from many goroutines.
func (b *BatchIoUringReader) Submit(fd int, buf []byte, offset uint64) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}

	var startTime = time.Now()

	req := batchReqPool.Get().(*batchReadRequest)
	req.fd = fd
	req.buf = buf
	req.offset = offset

	if !b.enqueue(req) {
		req.fd = 0
		req.buf = nil
		req.offset = 0
		batchReqPool.Put(req)
		return 0, fmt.Errorf("io_uring: shutting down")
	}

	result := <-req.done
	n, err := result.N, result.Err
	metrics.Timing(metrics.KEY_PREAD_LATENCY, time.Since(startTime), []string{})
	metrics.Incr(metrics.KEY_PREAD_COUNT, []string{})
	// Reset and return to pool
	req.fd = 0
	req.buf = nil
	req.offset = 0
	batchReqPool.Put(req)

	return n, err
}

// SubmitAsync enqueues a pread request and returns immediately with a channel
// that will receive the result when the io_uring completion arrives.
// Unlike Submit, it does not block the caller. Thread-safe.
// The caller must read exactly once from the returned channel.
func (b *BatchIoUringReader) SubmitAsync(fd int, buf []byte, offset uint64) <-chan ReadResult {
	if len(buf) == 0 {
		ch := make(chan ReadResult, 1)
		ch <- ReadResult{}
		return ch
	}
	req := &batchReadRequest{
		fd:     fd,
		buf:    buf,
		offset: offset,
		done:   make(chan ReadResult, 1),
	}
	if !b.enqueue(req) {
		req.done <- ReadResult{Err: fmt.Errorf("io_uring: shutting down")}
	}
	return req.done
}

// enqueue hands req to submitLoop, or reports false once Close has begun.
// Close drains reqCh only once, so a request queued after that would never
// be answered. A sender blocked here on a full reqCh holds the read lock;
// Close waits for it, and submitLoop is still running to make room.
func (b *BatchIoUringReader) enqueue(req *batchReadRequest) bool {
	b.closeMu.RLock()
	defer b.closeMu.RUnlock()
	if b.closed {
		return false
	}
	b.reqCh <- req
	return true
}

// Close shuts down both goroutines and releases the io_uring ring.
//
// Design note (m-review feedback): the cancel-on-close semantic here
// intentionally does NOT wait for in-flight kernel completions before
// tearing the ring down. For the reader that's obvious — cache reads are
// side-effect free. For the writer (mirrored in batch_writer_linux.go)
// it means writes submitted-to-kernel-but-not-yet-CQE'd may be lost or
// half-written at shutdown. That is deliberate: flashring's on-disk
// files are scratch — WrapCache.NewWrapCache wipes the mount on every
// startup — so writes that hadn't been CQE'd before Close are
// unreachable by definition. Adding a drain would just extend pod
// shutdown time without changing durability semantics. Consumers that
// need persistence must layer it above the cache tier (this is called
// out in the flashring CLAUDE.md).
//
// The old implementation had two deadlock paths at shutdown:
//
//  1. If the SQ was full when Close ran (~256 in-flight reads), getSqe
//     returned nil and the sentinel NOP was never submitted; completeLoop
//     blocked in waitCqe forever after its natural drain and Close hung
//     on wg.Wait.
//  2. If submitLoop had already stored requests into inflight[] but not
//     yet submitted them when closeCh fired, those callers (blocked in
//     Submit / SubmitAsync waiting on req.done) were never notified and
//     hung forever.
//
// The new sequence cancels every in-flight caller first (fixing #2), then
// tries the sentinel; if the sentinel can't be submitted, forcibly closes
// the ring fd so waitCqe's io_uring_enter returns EBADF (fixing #1). The
// mmap regions stay valid until the final ring.Close after wg.Wait.
func (b *BatchIoUringReader) Close() {
	b.closeMu.Lock()
	b.closed = true
	b.closeMu.Unlock()
	close(b.closeCh)

	// STEP 1: wait for submitLoop to exit. All submitLoop exit paths check
	// closeCh, so this happens promptly. After this point, no new Stores
	// into inflight[] and no new pending increments can happen — a
	// guarantee that eliminates the sentinel-vs-Store race where sentinel
	// arrives with pending=0, completeLoop exits, then submitLoop
	// Stores+increments+submits, leaving that request stranded.
	<-b.submitLoopDone

	// STEP 2: cancel every in-flight caller. Runs AFTER submitLoop is
	// dead, so any Store that raced with our earlier closeCh signal is
	// now visible. Covers:
	//   - requests submitLoop Stored (from an in-flight batch that closeCh
	//     interrupted after Store but before we got here)
	//   - requests submitLoop Stored + submitted (kernel CQEs may still
	//     be pending; completeLoop's Swap will return nil for those)
	for i := range b.inflight {
		req := b.inflight[i].Swap(nil)
		if req != nil {
			select {
			case req.done <- ReadResult{Err: fmt.Errorf("io_uring: shutting down")}:
			default:
			}
			b.pending.Add(-1)
		}
	}

	// STEP 3: wake completeLoop via sentinel (clean path) or fd close
	// (fallback if SQ is full). pending is now stable: no more submits
	// will happen and we've decremented every cancelled request.
	b.ring.mu.Lock()
	if sqe := b.ring.getSqe(); sqe != nil {
		sqe.Opcode = iouringOpNop
		sqe.UserData = sentinelUserData
		b.ring.submit(0)
	} else if b.ring.fd >= 0 {
		syscall.Close(b.ring.fd)
		b.ring.fd = -1
	}
	b.ring.mu.Unlock()

	b.wg.Wait()

	// STEP 4: drain requests submitLoop never got to pick up. Safe now —
	// no other receiver on reqCh.
drainReqCh:
	for {
		select {
		case req := <-b.reqCh:
			select {
			case req.done <- ReadResult{Err: fmt.Errorf("io_uring: shutting down")}:
			default:
			}
		default:
			break drainReqCh
		}
	}

	b.ring.Close()
}

// submitLoop collects requests from reqCh and submits them as io_uring SQEs.
// It never waits for completions — that happens in completeLoop. The ring
// mutex is held only during SQE preparation + io_uring_enter (~1-5μs).
func (b *BatchIoUringReader) submitLoop() {
	defer b.wg.Done()
	defer close(b.submitLoopDone)

	batch := make([]*batchReadRequest, 0, b.maxBatch)
	slots := make([]uint32, 0, b.maxBatch)

	for {
		// Block until the first request arrives.
		select {
		case req := <-b.reqCh:
			batch = append(batch, req)
		case <-b.closeCh:
			return
		}

		// Non-blocking drain of whatever else is already queued.
		for len(batch) < b.maxBatch {
			select {
			case req := <-b.reqCh:
				batch = append(batch, req)
			default:
				goto submit
			}
		}

	submit:
		// Acquire a free slot for each request. Under normal load (~30
		// in-flight out of 256 slots) this never blocks. NOTE: we do NOT
		// call inflight[slot].Store(req) here — that happens AFTER
		// prepRead has copied req.fd/buf/offset into the SQE, closing
		// the race where Close cancels this request and Submit
		// resets/pools the batchReadRequest while submitLoop still holds
		// a reference to its fields.
		for range batch {
			select {
			case slot := <-b.freeSlots:
				slots = append(slots, slot)
			case <-b.closeCh:
				// Return the slots we've already acquired — they were
				// never used (Store to inflight[] happens later, under
				// the ring mutex).
				for _, slot := range slots {
					b.freeSlots <- slot
				}
				// Cancel every request in the batch, NOT just j..end.
				// Requests 0..i-1 already got slots but no Store into
				// inflight[] happened (per the reorder), so Close's
				// inflight walk can't see them either — they'd be
				// orphaned. Send explicit shutdown errors here.
				for _, req := range batch {
					select {
					case req.done <- ReadResult{Err: fmt.Errorf("io_uring: shutting down")}:
					default:
					}
				}
				return
			}
		}

		metrics.Timing(metrics.KEY_IOURING_SIZE, time.Duration(len(batch))*time.Millisecond, []string{})

		b.ring.mu.Lock()

		prepared := 0
		for i, slot := range slots {
			sqe := b.ring.getSqe()
			if sqe == nil {
				// Neither the current slot nor the remaining slots have
				// been stored in inflight[] yet (fix for the pooled-req
				// race), so we free them directly and error out the
				// unwrapped batch entries.
				for j := i; j < len(slots); j++ {
					b.freeSlots <- slots[j]
					select {
					case batch[j].done <- ReadResult{
						Err: fmt.Errorf("io_uring: SQ full, batch=%d depth=%d", len(batch), b.ring.sqEntries),
					}:
					default:
					}
				}
				break
			}
			// Read req fields (fd, buf, offset) BEFORE Store — Close
			// cannot cancel a request that isn't in inflight[] yet, so
			// the fields are guaranteed stable for the duration of
			// prepRead.
			prepRead(sqe, batch[i].fd, batch[i].buf, batch[i].offset)
			sqe.UserData = uint64(slot)
			b.inflight[slot].Store(batch[i])
			prepared++
		}

		if prepared > 0 {
			b.pending.Add(int32(prepared))
			_, err := b.ring.submit(0)
			if err != nil {
				b.pending.Add(-int32(prepared))
				for i := 0; i < prepared; i++ {
					req := b.inflight[slots[i]].Swap(nil)
					b.freeSlots <- slots[i]
					if req != nil {
						req.done <- ReadResult{Err: fmt.Errorf("io_uring_enter: %w", err)}
					}
				}
			}
		}

		b.ring.mu.Unlock()

		batch = batch[:0]
		slots = slots[:0]
	}
}

// completeLoop continuously drains CQEs and dispatches results to callers.
// It runs independently of submitLoop — the ring's SQ and CQ are separate
// data structures, so no mutex is needed for CQ access (single consumer).
func (b *BatchIoUringReader) completeLoop() {
	defer b.wg.Done()

	for {
		cqe, err := b.ring.waitCqe()
		if err != nil {
			select {
			case <-b.closeCh:
				if b.pending.Load() <= 0 {
					return
				}
			default:
			}
			continue
		}

		userData := cqe.UserData
		res := cqe.Res
		b.ring.seenCqe()

		// Shutdown NOP — exit once all real I/O has been drained.
		if userData == sentinelUserData {
			if b.pending.Load() <= 0 {
				return
			}
			continue
		}

		slot := uint32(userData)
		b.pending.Add(-1)

		req := b.inflight[slot].Swap(nil)
		b.freeSlots <- slot

		if req != nil {
			if res < 0 {
				req.done <- ReadResult{
					Err: fmt.Errorf("io_uring pread errno %d (%s), fd=%d off=%d len=%d",
						-res, syscall.Errno(-res), req.fd, req.offset, len(req.buf)),
				}
			} else {
				req.done <- ReadResult{N: int(res)}
			}
		}

		// After processing a real CQE, if we're shutting down and
		// pending has drained to zero, exit even if the sentinel CQE
		// was consumed earlier when pending was still > 0. Without
		// this, a kernel that delivers the NOP CQE before the natural
		// read CQE (kernel-scheduler order is not FIFO) leaves
		// completeLoop blocking in the next waitCqe with nothing to
		// wake it.
		if b.pending.Load() <= 0 {
			select {
			case <-b.closeCh:
				return
			default:
			}
		}
	}
}

// ParallelBatchIoUringReader distributes pread requests across N independent
// BatchIoUringReader instances (each with its own io_uring ring and goroutines)
// using round-robin.
type ParallelBatchIoUringReader struct {
	readers []*BatchIoUringReader
	next    atomic.Uint64
}

// NewParallelBatchIoUringReader creates numRings independent batch readers.
// Each ring gets its own io_uring instance and background goroutines.
func NewParallelBatchIoUringReader(cfg BatchIoUringConfig, numRings int) (*ParallelBatchIoUringReader, error) {
	if numRings <= 0 {
		numRings = 1
	}
	readers := make([]*BatchIoUringReader, numRings)
	for i := 0; i < numRings; i++ {
		r, err := NewBatchIoUringReader(cfg)
		if err != nil {
			for j := 0; j < i; j++ {
				readers[j].Close()
			}
			return nil, fmt.Errorf("parallel batch reader ring %d: %w", i, err)
		}
		readers[i] = r
	}
	return &ParallelBatchIoUringReader{readers: readers}, nil
}

// Submit routes the pread to the next ring via round-robin. Thread-safe.
func (p *ParallelBatchIoUringReader) Submit(fd int, buf []byte, offset uint64) (int, error) {
	idx := p.next.Add(1) % uint64(len(p.readers))
	return p.readers[idx].Submit(fd, buf, offset)
}

// SubmitAsync routes the pread to the next ring via round-robin and returns
// immediately with a channel for the result. Thread-safe.
func (p *ParallelBatchIoUringReader) SubmitAsync(fd int, buf []byte, offset uint64) <-chan ReadResult {
	idx := p.next.Add(1) % uint64(len(p.readers))
	return p.readers[idx].SubmitAsync(fd, buf, offset)
}

// Close shuts down all underlying batch readers.
func (p *ParallelBatchIoUringReader) Close() {
	for _, r := range p.readers {
		r.Close()
	}
}
