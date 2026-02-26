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

// BatchIoUringReader collects pread requests from multiple goroutines into a
// single channel and submits them as one io_uring batch. This amortizes the
// syscall overhead (1 io_uring_enter instead of N) and lets NVMe process
// multiple commands in parallel (queue depth > 1).
//
// Collection uses non-blocking channel drain: after receiving the first
// request, it drains whatever else is already queued (no timer). Under load
// this provides natural batching; under low load single requests go out
// with zero added latency.
//
// CQEs are dispatched individually as they complete (no head-of-line blocking).
type BatchIoUringReader struct {
	ring     *IoUring
	reqCh    chan *batchReadRequest
	maxBatch int
	window   time.Duration // wait up to this for more requests before submit (0 = drain only)
	closeCh  chan struct{}
	wg       sync.WaitGroup
}

// BatchIoUringConfig configures the batch reader.
type BatchIoUringConfig struct {
	RingDepth uint32        // io_uring SQ/CQ size (default 256)
	MaxBatch  int           // max requests per batch (capped to RingDepth)
	Window    time.Duration // wait up to this for requests to accumulate before submit (e.g. 500*time.Microsecond); 0 = drain only, no wait
	QueueSize int           // channel buffer size (default 1024)
}

// NewBatchIoUringReader creates a batch reader with its own io_uring ring
// and starts the background collection goroutine.
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

	ring, err := NewIoUring(cfg.RingDepth, 0)
	if err != nil {
		return nil, fmt.Errorf("batch io_uring init: %w", err)
	}

	b := &BatchIoUringReader{
		ring:     ring,
		reqCh:    make(chan *batchReadRequest, cfg.QueueSize),
		maxBatch: cfg.MaxBatch,
		window:   cfg.Window,
		closeCh:  make(chan struct{}),
	}
	b.wg.Add(1)
	go b.loop()
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

// Close shuts down the collection goroutine and releases the io_uring ring.
func (b *BatchIoUringReader) Close() {
	close(b.closeCh)
	b.wg.Wait()
	b.ring.Close()
}

// loop is the single background goroutine that collects and submits batches.
//
// Phase 1: block on first request (no timer ticking when idle).
// Phase 2: non-blocking drain of whatever else is already queued.
// Phase 3: submit the batch and dispatch CQEs as they complete.
func (b *BatchIoUringReader) loop() {
	defer b.wg.Done()

	batch := make([]*batchReadRequest, 0, b.maxBatch)

	for {
		// Phase 1: block until the first request arrives
		select {
		case req := <-b.reqCh:
			batch = append(batch, req)
		case <-b.closeCh:
			return
		}

		// Phase 2: drain with optional wait — if window > 0, wait up to window
		// for more requests; otherwise non-blocking drain only.
		var timer *time.Timer
		if b.window > 0 {
			timer = time.NewTimer(b.window)
		}
	drain:
		for len(batch) < b.maxBatch {
			if b.window > 0 {
				select {
				case req := <-b.reqCh:
					batch = append(batch, req)
				case <-timer.C:
					break drain
				case <-b.closeCh:
					if timer != nil {
						timer.Stop()
					}
					return
				}
			} else {
				select {
				case req := <-b.reqCh:
					batch = append(batch, req)
				default:
					break drain
				}
			}
		}
		if timer != nil {
			timer.Stop()
		}

		// Phase 3: submit and dispatch
		b.submitBatch(batch)
		batch = batch[:0]
	}
}

// submitBatch prepares N SQEs, submits them (fire-and-forget), then dispatches
// each CQE individually as it completes. Fast reads are dispatched immediately
// without waiting for slow reads in the same batch (no head-of-line blocking).
func (b *BatchIoUringReader) submitBatch(batch []*batchReadRequest) {
	metrics.Timing(metrics.KEY_IOURING_SIZE, time.Duration(len(batch))*time.Millisecond, []string{})
	n := len(batch)
	if n == 0 {
		return
	}

	b.ring.mu.Lock()

	// Prepare SQEs
	prepared := 0
	for i, req := range batch {
		sqe := b.ring.getSqe()
		if sqe == nil {
			// SQ full -- error the rest
			for j := i; j < n; j++ {
				batch[j].done <- batchReadResult{
					Err: fmt.Errorf("io_uring: SQ full, batch=%d depth=%d", n, b.ring.sqEntries),
				}
			}
			break
		}
		prepRead(sqe, req.fd, req.buf, req.offset)
		sqe.UserData = uint64(i) // index for CQE matching
		prepared++
	}

	if prepared == 0 {
		b.ring.mu.Unlock()
		return
	}

	// Submit SQEs but do NOT wait for completions (waitNr=0).
	// The kernel starts processing I/O immediately; we dispatch each CQE
	// as it arrives below, so fast reads aren't blocked by slow ones.
	_, err := b.ring.submit(0)
	if err != nil {
		b.ring.mu.Unlock()
		for i := 0; i < prepared; i++ {
			batch[i].done <- batchReadResult{Err: fmt.Errorf("io_uring_enter: %w", err)}
		}
		return
	}

	// Dispatch CQEs one-by-one as they complete.
	completed := 0
	for completed < prepared {
		cqe, err := b.ring.waitCqe()
		if err != nil {
			// Catastrophic ring error -- unblock all unsatisfied callers.
			b.ring.mu.Unlock()
			for i := 0; i < n; i++ {
				select {
				case batch[i].done <- batchReadResult{Err: fmt.Errorf("io_uring waitCqe: %w", err)}:
				default: // already sent
				}
			}
			return
		}

		idx := int(cqe.UserData)
		res := cqe.Res
		b.ring.seenCqe()
		completed++

		if idx < 0 || idx >= prepared {
			continue // unexpected UserData; skip
		}

		if res < 0 {
			batch[idx].done <- batchReadResult{
				Err: fmt.Errorf("io_uring pread errno %d (%s), fd=%d off=%d len=%d",
					-res, syscall.Errno(-res), batch[idx].fd, batch[idx].offset, len(batch[idx].buf)),
			}
		} else {
			batch[idx].done <- batchReadResult{N: int(res)}
		}
	}

	b.ring.mu.Unlock()
}

// ParallelBatchIoUringReader distributes pread requests across N independent
// BatchIoUringReader instances (each with its own io_uring ring and goroutine)
// using round-robin. This removes the single-ring serialization bottleneck and
// lets NVMe service requests across multiple hardware queues in parallel.
type ParallelBatchIoUringReader struct {
	readers []*BatchIoUringReader
	next    atomic.Uint64
}

// NewParallelBatchIoUringReader creates numRings independent batch readers.
// Each ring gets its own io_uring instance and background goroutine.
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
