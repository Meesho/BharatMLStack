//go:build linux
// +build linux

package fs

import (
	"fmt"
	"sync"
	"syscall"
	"time"
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
// Typical usage: create one global instance shared across all shards.
type BatchIoUringReader struct {
	ring     *IoUring
	reqCh    chan *batchReadRequest
	maxBatch int
	window   time.Duration
	closeCh  chan struct{}
	wg       sync.WaitGroup
}

// BatchIoUringConfig configures the batch reader.
type BatchIoUringConfig struct {
	RingDepth uint32        // io_uring SQ/CQ size (default 256)
	MaxBatch  int           // max requests per batch (capped to RingDepth)
	Window    time.Duration // collection window after first request arrives
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
	if cfg.Window == 0 {
		cfg.Window = time.Millisecond
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

	req := batchReqPool.Get().(*batchReadRequest)
	req.fd = fd
	req.buf = buf
	req.offset = offset

	b.reqCh <- req

	result := <-req.done
	n, err := result.N, result.Err

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
// Phase 2: collect up to maxBatch or until the window expires.
// Phase 3: submit the batch in one io_uring_enter call.
func (b *BatchIoUringReader) loop() {
	defer b.wg.Done()

	batch := make([]*batchReadRequest, 0, b.maxBatch)

	// Pre-allocate and stop the timer so Reset works correctly.
	collectTimer := time.NewTimer(0)
	if !collectTimer.Stop() {
		<-collectTimer.C
	}

	for {
		// Phase 1: wait for first request (idle)
		select {
		case req := <-b.reqCh:
			batch = append(batch, req)
		case <-b.closeCh:
			return
		}

		// Phase 2: collect more within the window
		collectTimer.Reset(b.window)
	collect:
		for len(batch) < b.maxBatch {
			select {
			case req := <-b.reqCh:
				batch = append(batch, req)
			case <-collectTimer.C:
				break collect
			}
		}
		// Drain timer if we exited early (maxBatch reached before timer fired)
		if !collectTimer.Stop() {
			select {
			case <-collectTimer.C:
			default:
			}
		}

		// Phase 3: submit the batch
		b.submitBatch(batch)
		batch = batch[:0]
	}
}

// submitBatch prepares N SQEs, submits them in one io_uring_enter(N, N),
// drains all CQEs, and dispatches results back to callers via done channels.
func (b *BatchIoUringReader) submitBatch(batch []*batchReadRequest) {
	// log.Info().Msgf("submitting batch of %d requests", len(batch))
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

	// Submit all at once; kernel waits for all completions before returning.
	_, err := b.ring.submit(uint32(prepared))
	if err != nil {
		b.ring.mu.Unlock()
		for i := 0; i < prepared; i++ {
			batch[i].done <- batchReadResult{Err: fmt.Errorf("io_uring_enter: %w", err)}
		}
		return
	}

	// Drain CQEs -- order may differ from submission order.
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
