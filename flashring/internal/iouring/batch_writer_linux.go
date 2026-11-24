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

// WriteResult holds the outcome of a single io_uring pwrite.
type WriteResult struct {
	N   int
	Err error
}

// batchWriteRequest is a pwrite submitted to the batch writer.
type batchWriteRequest struct {
	fd     int
	buf    []byte
	offset uint64
	done   chan WriteResult
}

// BatchIoUringWriter decouples submission from completion for write operations,
// mirroring the BatchIoUringReader pattern. The mutex is held only during SQE
// preparation + io_uring_enter (~1-5μs), not during CQE drain.
//
//	submitLoop:    reqCh → collect batch → prep SQEs → io_uring_enter → loop
//	completeLoop:  waitCqe → dispatch result to caller → loop
type BatchIoUringWriter struct {
	ring     *IoUring
	reqCh    chan *batchWriteRequest
	maxBatch int
	closeCh  chan struct{}

	// closeMu orders SubmitWriteBatch against Close; see BatchIoUringReader.
	closeMu sync.RWMutex
	closed  bool
	wg      sync.WaitGroup

	// Same rationale as BatchIoUringReader.submitLoopDone — Close waits
	// on this before walking inflight[] a second time and submitting the
	// sentinel. See the reader's Close docstring for the race this
	// eliminates.
	submitLoopDone chan struct{}

	inflight  []atomic.Pointer[batchWriteRequest]
	freeSlots chan uint32
	pending   atomic.Int32
}

// NewBatchIoUringWriter creates a decoupled batch writer with its own io_uring
// ring and starts the submit + completion goroutines.
func NewBatchIoUringWriter(cfg BatchIoUringConfig) (*BatchIoUringWriter, error) {
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

	ring, err := NewIoUring(cfg.RingDepth, 0)
	if err != nil {
		return nil, fmt.Errorf("batch io_uring writer init: %w", err)
	}

	freeSlots := make(chan uint32, maxInflight)
	for i := 0; i < maxInflight; i++ {
		freeSlots <- uint32(i)
	}

	b := &BatchIoUringWriter{
		ring:           ring,
		reqCh:          make(chan *batchWriteRequest, cfg.QueueSize),
		maxBatch:       cfg.MaxBatch,
		closeCh:        make(chan struct{}),
		submitLoopDone: make(chan struct{}),
		inflight:       make([]atomic.Pointer[batchWriteRequest], ringDepth),
		freeSlots:      freeSlots,
	}
	b.wg.Add(2)
	go b.submitLoop()
	go b.completeLoop()
	return b, nil
}

// MaxBatchSize returns the ring depth, which is the maximum number of SQEs
// that can be in-flight at once.
func (b *BatchIoUringWriter) MaxBatchSize() int {
	return int(b.ring.sqEntries)
}

// SubmitWriteBatch submits N pwrite operations and waits for all completions.
// Thread-safe. Unlike the old IoUring.SubmitWriteBatch, the ring mutex is NOT
// held during CQE drain — other batches can be submitted concurrently.
func (b *BatchIoUringWriter) SubmitWriteBatch(fd int, bufs [][]byte, offsets []uint64) ([]int, error) {
	n := len(bufs)
	if n == 0 {
		return nil, nil
	}

	startTime := time.Now()

	// Submit all write requests into the channel. The submitLoop will
	// collect them into batches and prep SQEs.
	doneChans := make([]chan WriteResult, n)
	// Hold the read lock across the sends: Close drains reqCh only once, so
	// a request queued after Close has begun would never be answered.
	b.closeMu.RLock()
	if b.closed {
		b.closeMu.RUnlock()
		return make([]int, n), fmt.Errorf("io_uring writer: shutting down")
	}
	for i := 0; i < n; i++ {
		req := &batchWriteRequest{
			fd:     fd,
			buf:    bufs[i],
			offset: offsets[i],
			done:   make(chan WriteResult, 1),
		}
		doneChans[i] = req.done
		b.reqCh <- req
	}
	b.closeMu.RUnlock()

	// Collect all completions.
	results := make([]int, n)
	for i := 0; i < n; i++ {
		res := <-doneChans[i]
		if res.Err != nil {
			return results, res.Err
		}
		results[i] = res.N
		metrics.Timing(metrics.KEY_PWRITE_LATENCY, time.Since(startTime), []string{})
	}

	return results, nil
}

// Close shuts down both goroutines and releases the io_uring ring.
// Mirrors BatchIoUringReader.Close — cancels in-flight callers before
// attempting sentinel or the fd-close fallback. See the reader's Close
// docstring for the two deadlock classes this fixes.
func (b *BatchIoUringWriter) Close() {
	b.closeMu.Lock()
	b.closed = true
	b.closeMu.Unlock()
	close(b.closeCh)

	// Wait for submitLoop to exit before walking inflight — see the
	// reader's Close docstring for the sentinel-vs-Store race this fixes.
	<-b.submitLoopDone

	for i := range b.inflight {
		req := b.inflight[i].Swap(nil)
		if req != nil {
			select {
			case req.done <- WriteResult{Err: fmt.Errorf("io_uring writer: shutting down")}:
			default:
			}
			b.pending.Add(-1)
		}
	}

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

	// Drain any requests still parked in reqCh — see the reader's
	// equivalent block for the rationale.
drainReqCh:
	for {
		select {
		case req := <-b.reqCh:
			select {
			case req.done <- WriteResult{Err: fmt.Errorf("io_uring writer: shutting down")}:
			default:
			}
		default:
			break drainReqCh
		}
	}

	b.ring.Close()
}

// submitLoop collects write requests and submits them as io_uring SQEs.
// Mutex held only during SQE prep + io_uring_enter.
func (b *BatchIoUringWriter) submitLoop() {
	defer b.wg.Done()
	defer close(b.submitLoopDone)

	batch := make([]*batchWriteRequest, 0, b.maxBatch)
	slots := make([]uint32, 0, b.maxBatch)

	for {
		select {
		case req := <-b.reqCh:
			batch = append(batch, req)
		case <-b.closeCh:
			return
		}

		// Non-blocking drain.
		for len(batch) < b.maxBatch {
			select {
			case req := <-b.reqCh:
				batch = append(batch, req)
			default:
				goto submit
			}
		}

	submit:
		// Acquire free slots without storing into inflight[] yet — see
		// the corresponding comment in BatchIoUringReader.submitLoop for
		// the race this avoids.
		for range batch {
			select {
			case slot := <-b.freeSlots:
				slots = append(slots, slot)
			case <-b.closeCh:
				// Return acquired slots and cancel the whole batch —
				// same rationale as BatchIoUringReader.submitLoop's
				// closeCh branch: batch[0..i-1] got slots but no Store
				// into inflight[] means Close's inflight walk can't see
				// them either.
				for _, slot := range slots {
					b.freeSlots <- slot
				}
				for _, req := range batch {
					select {
					case req.done <- WriteResult{Err: fmt.Errorf("io_uring writer: shutting down")}:
					default:
					}
				}
				return
			}
		}

		b.ring.mu.Lock()

		prepared := 0
		for i, slot := range slots {
			sqe := b.ring.getSqe()
			if sqe == nil {
				for j := i; j < len(slots); j++ {
					b.freeSlots <- slots[j]
					select {
					case batch[j].done <- WriteResult{
						Err: fmt.Errorf("io_uring writer: SQ full, batch=%d depth=%d", len(batch), b.ring.sqEntries),
					}:
					default:
					}
				}
				break
			}
			// Read req fields BEFORE Store, so Close can't cancel the
			// request out from under us mid-prepWrite.
			prepWrite(sqe, batch[i].fd, batch[i].buf, batch[i].offset)
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
						req.done <- WriteResult{Err: fmt.Errorf("io_uring_enter: %w", err)}
					}
				}
			}
		}

		b.ring.mu.Unlock()

		batch = batch[:0]
		slots = slots[:0]
	}
}

// completeLoop drains CQEs and dispatches results to callers.
func (b *BatchIoUringWriter) completeLoop() {
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
				req.done <- WriteResult{
					Err: fmt.Errorf("io_uring pwrite errno %d (%s), fd=%d off=%d len=%d",
						-res, syscall.Errno(-res), req.fd, req.offset, len(req.buf)),
				}
			} else {
				req.done <- WriteResult{N: int(res)}
			}
		}

		// Same post-CQE close-check as BatchIoUringReader.completeLoop.
		if b.pending.Load() <= 0 {
			select {
			case <-b.closeCh:
				return
			default:
			}
		}
	}
}
