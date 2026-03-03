//go:build linux
// +build linux

package fs

import (
	"fmt"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Meesho/BharatMLStack/flashring/pkg/metrics"
)

// batchReadResult holds the outcome of a single batched pread.
type batchReadResult struct {
	N   int
	Err error
}

// batchReadRequest is a pread submitted to the batch reader.
type batchReadRequest struct {
	fd     int
	buf    []byte
	offset uint64
	done   chan batchReadResult
}

var batchReqPool = sync.Pool{
	New: func() interface{} {
		return &batchReadRequest{
			done: make(chan batchReadResult, 1),
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

	// In-flight tracking: each SQE gets a slot index as its UserData.
	// The submit goroutine stores the request; the completion goroutine
	// reads it back when the CQE arrives.
	inflight  []atomic.Pointer[batchReadRequest]
	freeSlots chan uint32  // pool of available slot indices
	pending   atomic.Int32 // number of SQEs currently in-flight
}

// BatchIoUringConfig configures the batch reader.
type BatchIoUringConfig struct {
	RingDepth uint32        // io_uring SQ/CQ size (default 256)
	MaxBatch  int           // max requests per batch (capped to RingDepth)
	Window    time.Duration // unused in decoupled mode; kept for API compatibility
	QueueSize int           // channel buffer size (default 1024)
	SQPoll    bool          // use IORING_SETUP_SQPOLL; kernel polls SQ, eliminating submit syscalls under load
}

// NewBatchIoUringReader creates a batch reader with its own io_uring ring
// and starts the submit + completion goroutines.
func NewBatchIoUringReader(cfg BatchIoUringConfig) (*BatchIoUringReader, error) {
	if cfg.RingDepth == 0 {
		cfg.RingDepth = 256
	}
	if cfg.MaxBatch == 0 || cfg.MaxBatch > int(cfg.RingDepth) {
		cfg.MaxBatch = int(cfg.RingDepth)
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

	ringDepth := int(cfg.RingDepth)
	freeSlots := make(chan uint32, ringDepth)
	for i := 0; i < ringDepth; i++ {
		freeSlots <- uint32(i)
	}

	b := &BatchIoUringReader{
		ring:      ring,
		reqCh:     make(chan *batchReadRequest, cfg.QueueSize),
		maxBatch:  cfg.MaxBatch,
		closeCh:   make(chan struct{}),
		inflight:  make([]atomic.Pointer[batchReadRequest], ringDepth),
		freeSlots: freeSlots,
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

	b.reqCh <- req

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

// Close shuts down both goroutines and releases the io_uring ring.
func (b *BatchIoUringReader) Close() {
	close(b.closeCh)

	// Submit a NOP with sentinel UserData to unblock the completion
	// goroutine if it is blocked in waitCqe with no pending I/O.
	b.ring.mu.Lock()
	sqe := b.ring.getSqe()
	if sqe != nil {
		sqe.Opcode = iouringOpNop
		sqe.UserData = sentinelUserData
		b.ring.submit(0)
	}
	b.ring.mu.Unlock()

	b.wg.Wait()
	b.ring.Close()
}

// submitLoop collects requests from reqCh and submits them as io_uring SQEs.
// It never waits for completions — that happens in completeLoop. The ring
// mutex is held only during SQE preparation + io_uring_enter (~1-5μs).
func (b *BatchIoUringReader) submitLoop() {
	defer b.wg.Done()

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
		// in-flight out of 256 slots) this never blocks.
		for i, req := range batch {
			select {
			case slot := <-b.freeSlots:
				slots = append(slots, slot)
				b.inflight[slot].Store(req)
			case <-b.closeCh:
				for j := i; j < len(batch); j++ {
					batch[j].done <- batchReadResult{Err: fmt.Errorf("io_uring: shutting down")}
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
				for j := i; j < len(slots); j++ {
					req := b.inflight[slots[j]].Swap(nil)
					b.freeSlots <- slots[j]
					if req != nil {
						req.done <- batchReadResult{
							Err: fmt.Errorf("io_uring: SQ full, batch=%d depth=%d", len(batch), b.ring.sqEntries),
						}
					}
				}
				break
			}
			prepRead(sqe, batch[i].fd, batch[i].buf, batch[i].offset)
			sqe.UserData = uint64(slot)
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
						req.done <- batchReadResult{Err: fmt.Errorf("io_uring_enter: %w", err)}
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

		if req == nil {
			continue
		}

		if res < 0 {
			req.done <- batchReadResult{
				Err: fmt.Errorf("io_uring pread errno %d (%s), fd=%d off=%d len=%d",
					-res, syscall.Errno(-res), req.fd, req.offset, len(req.buf)),
			}
		} else {
			req.done <- batchReadResult{N: int(res)}
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

// Close shuts down all underlying batch readers.
func (p *ParallelBatchIoUringReader) Close() {
	for _, r := range p.readers {
		r.Close()
	}
}
