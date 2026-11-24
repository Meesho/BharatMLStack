//go:build linux
// +build linux

package iouring

import (
	"bytes"
	"os"
	"sync"
	"testing"
	"time"
)

func TestNewBatchIoUringReader_DefaultsApplied(t *testing.T) {
	r, err := NewBatchIoUringReader(BatchIoUringConfig{})
	if err != nil {
		t.Fatalf("NewBatchIoUringReader: %v", err)
	}
	defer r.Close()

	if r.maxBatch == 0 {
		t.Fatal("maxBatch = 0, want default")
	}
	if cap(r.freeSlots) == 0 {
		t.Fatal("freeSlots capacity = 0")
	}
	if cap(r.reqCh) == 0 {
		t.Fatal("reqCh capacity = 0 (QueueSize default not applied)")
	}
}

func TestNewBatchIoUringReader_SQPoll_FallsBack(t *testing.T) {
	// SQPOLL requires CAP_SYS_NICE on older kernels; the production code
	// retries without the flag on failure. Inside the test container we
	// usually *do* have the cap because of privileged: true, so this mostly
	// exercises the happy path — but the second NewIoUring call still has
	// to be reachable. Skip if even the fallback can't init.
	r, err := NewBatchIoUringReader(BatchIoUringConfig{SQPoll: true})
	if err != nil {
		t.Skipf("io_uring SQPoll init failed even with fallback: %v", err)
	}
	defer r.Close()
	if r.ring == nil {
		t.Fatal("ring nil after SQPoll setup")
	}
}

func TestNewBatchIoUringReader_MaxBatchClampedToInflight(t *testing.T) {
	r, err := NewBatchIoUringReader(BatchIoUringConfig{
		RingDepth:   8,
		MaxBatch:    1024,
		MaxInflight: 4,
	})
	if err != nil {
		t.Fatalf("NewBatchIoUringReader: %v", err)
	}
	defer r.Close()

	if r.maxBatch > 4 {
		t.Fatalf("maxBatch = %d, want <= 4 (clamped)", r.maxBatch)
	}
}

func TestBatchIoUringReader_Submit_ReadsBackData(t *testing.T) {
	fd, data := newTempReadOnlyFile(t, 4096)

	r, err := NewBatchIoUringReader(BatchIoUringConfig{RingDepth: 16, MaxBatch: 4})
	if err != nil {
		t.Fatalf("NewBatchIoUringReader: %v", err)
	}
	defer r.Close()

	buf := alignedBlock(4096, 4096)
	n, err := r.Submit(fd, buf, 0)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if n != 4096 {
		t.Fatalf("Submit n = %d, want 4096", n)
	}
	if buf[0] != data[0] {
		t.Fatalf("data mismatch at byte 0: got %d, want %d", buf[0], data[0])
	}
}

func TestBatchIoUringReader_Submit_EmptyBuffer(t *testing.T) {
	r, err := NewBatchIoUringReader(BatchIoUringConfig{})
	if err != nil {
		t.Fatalf("NewBatchIoUringReader: %v", err)
	}
	defer r.Close()

	n, err := r.Submit(0, []byte{}, 0)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if n != 0 {
		t.Fatalf("n = %d, want 0", n)
	}
}

func TestBatchIoUringReader_SubmitAsync_EmptyBuffer(t *testing.T) {
	r, err := NewBatchIoUringReader(BatchIoUringConfig{})
	if err != nil {
		t.Fatalf("NewBatchIoUringReader: %v", err)
	}
	defer r.Close()

	ch := r.SubmitAsync(0, []byte{}, 0)
	res := <-ch
	if res.Err != nil || res.N != 0 {
		t.Fatalf("SubmitAsync empty buf = %+v, want zero", res)
	}
}

func TestBatchIoUringReader_SubmitAsync_HappyPath(t *testing.T) {
	fd, _ := newTempReadOnlyFile(t, 4096)

	r, err := NewBatchIoUringReader(BatchIoUringConfig{RingDepth: 16, MaxBatch: 4})
	if err != nil {
		t.Fatalf("NewBatchIoUringReader: %v", err)
	}
	defer r.Close()

	buf := alignedBlock(4096, 4096)
	res := <-r.SubmitAsync(fd, buf, 0)
	if res.Err != nil {
		t.Fatalf("SubmitAsync err: %v", res.Err)
	}
	if res.N != 4096 {
		t.Fatalf("res.N = %d, want 4096", res.N)
	}
}

func TestBatchIoUringReader_MultipleReads_BatchPath(t *testing.T) {
	fd, _ := newTempReadOnlyFile(t, 4096*8)

	r, err := NewBatchIoUringReader(BatchIoUringConfig{RingDepth: 32, MaxBatch: 8})
	if err != nil {
		t.Fatalf("NewBatchIoUringReader: %v", err)
	}
	defer r.Close()

	// Fire several Async reads at once to exercise the non-blocking drain.
	chs := make([]<-chan ReadResult, 8)
	for i := 0; i < 8; i++ {
		buf := alignedBlock(4096, 4096)
		chs[i] = r.SubmitAsync(fd, buf, uint64(i*4096))
	}
	for i, ch := range chs {
		res := <-ch
		if res.Err != nil {
			t.Fatalf("read %d err: %v", i, res.Err)
		}
		if res.N != 4096 {
			t.Fatalf("read %d n = %d, want 4096", i, res.N)
		}
	}
}

type shortReadErr struct{ n int }

func (e *shortReadErr) Error() string { return "short read" }

type mismatchErr struct{}

func (mismatchErr) Error() string { return "data mismatch" }

func TestBatchIoUringReader_ConcurrentReads_AllComplete(t *testing.T) {
	fd, data := newTempReadOnlyFile(t, 4096)

	r, err := NewBatchIoUringReader(BatchIoUringConfig{
		RingDepth:   16,
		MaxBatch:    8,
		MaxInflight: 8,
		QueueSize:   64,
	})
	if err != nil {
		t.Fatalf("NewBatchIoUringReader: %v", err)
	}
	defer r.Close()

	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b := alignedBlock(4096, 4096)
			n, err := r.Submit(fd, b, 0)
			if err != nil {
				errs <- err
				return
			}
			if n != 4096 {
				errs <- &shortReadErr{n: n}
				return
			}
			if !bytes.Equal(b, data) {
				errs <- &mismatchErr{}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Errorf("concurrent read error: %v", e)
	}
}

func TestNewParallelBatchIoUringReader_NumRingsDefaultsTo1(t *testing.T) {
	p, err := NewParallelBatchIoUringReader(BatchIoUringConfig{}, 0)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	defer p.Close()
	if len(p.readers) != 1 {
		t.Fatalf("len(readers) = %d, want 1", len(p.readers))
	}
}

func TestParallelBatchIoUringReader_Submit_RoundRobins(t *testing.T) {
	fd, _ := newTempReadOnlyFile(t, 4096*4)

	p, err := NewParallelBatchIoUringReader(BatchIoUringConfig{RingDepth: 16, MaxBatch: 4}, 2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	defer p.Close()

	buf := alignedBlock(4096, 4096)
	for i := 0; i < 4; i++ {
		n, err := p.Submit(fd, buf, 0)
		if err != nil {
			t.Fatalf("Submit %d: %v", i, err)
		}
		if n != 4096 {
			t.Fatalf("Submit %d n = %d, want 4096", i, n)
		}
	}
}

func TestParallelBatchIoUringReader_SubmitAsync(t *testing.T) {
	fd, _ := newTempReadOnlyFile(t, 4096)

	p, err := NewParallelBatchIoUringReader(BatchIoUringConfig{RingDepth: 16, MaxBatch: 4}, 2)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	defer p.Close()

	buf := alignedBlock(4096, 4096)
	res := <-p.SubmitAsync(fd, buf, 0)
	if res.Err != nil {
		t.Fatalf("SubmitAsync err: %v", res.Err)
	}
	if res.N != 4096 {
		t.Fatalf("res.N = %d, want 4096", res.N)
	}
}

func TestNewParallelBatchIoUringReader_MultipleRings_AllUsable(t *testing.T) {
	fd, data := newTempReadOnlyFile(t, 4096)

	p, err := NewParallelBatchIoUringReader(BatchIoUringConfig{
		RingDepth:   8,
		MaxBatch:    4,
		MaxInflight: 4,
	}, 4) // 4 rings to drive the for-range setup loop
	if err != nil {
		t.Fatalf("NewParallelBatchIoUringReader: %v", err)
	}
	defer p.Close()

	if len(p.readers) != 4 {
		t.Fatalf("len(readers) = %d, want 4", len(p.readers))
	}

	// Round-robin Submit through every ring to ensure each is wired up.
	for i := 0; i < 8; i++ {
		buf := alignedBlock(4096, 4096)
		n, err := p.Submit(fd, buf, 0)
		if err != nil {
			t.Fatalf("Submit #%d: %v", i, err)
		}
		if n != 4096 {
			t.Fatalf("Submit #%d returned %d", i, n)
		}
		if !bytes.Equal(buf, data) {
			t.Fatalf("Submit #%d data mismatch", i)
		}
	}
}

// TestBatchIoUringReader_Close_UnblocksPendingSubmitAsync exercises the
// C5+C6 fixes (PR #2377). Two prior bugs both surfaced as pod-hangs at
// shutdown:
//   - C5: sentinel NOP couldn't be submitted when the SQ was full at Close
//     time, leaving completeLoop blocked in waitCqe forever.
//   - C6: requests stored in inflight[] but not yet submitted when closeCh
//     fired never got a response on their done channel.
//
// The regression signal here: if either bug returns, Close hangs on
// wg.Wait OR the pending SubmitAsync's done channel is never signalled.
// Both cases are caught by the outer time.After on this test.
//
// The test issues N in-flight SubmitAsync calls (not just 1) to widen
// the race window — with only 1 request the race between submitLoop
// picking up the req and Close's inflight walk is very tight and
// occasionally observed to be won by neither side leaving the caller
// waiting on done. Batching amortises that: at least one of the N reqs
// will be Stored in inflight[] before Close's walk runs, and the drain
// covers the rest.
func TestBatchIoUringReader_Close_UnblocksPendingSubmitAsync(t *testing.T) {
	r, err := NewBatchIoUringReader(BatchIoUringConfig{RingDepth: 8, MaxBatch: 4, MaxInflight: 4})
	if err != nil {
		t.Skipf("io_uring init: %v", err)
	}

	// Open a small real file so SubmitAsync has a valid fd to work with.
	tmp := t.TempDir() + "/data"
	f, err := os.Create(tmp)
	if err != nil {
		r.Close()
		t.Fatalf("create: %v", err)
	}
	defer f.Close()
	if _, err := f.Write(make([]byte, 4096)); err != nil {
		r.Close()
		t.Fatalf("write: %v", err)
	}
	fd := int(f.Fd())

	// Issue several async reads in flight. Any one of them being signalled
	// via error or completion proves the shutdown-cancel plumbing works.
	// A single-request test occasionally sat in the "req already picked up
	// by submitLoop but not yet Stored" gap where neither Close's inflight
	// walk NOR its reqCh drain saw it. With N requests, at least one is
	// guaranteed to land in a checked bucket.
	const nInFlight = 4
	dones := make([]<-chan ReadResult, nInFlight)
	for i := 0; i < nInFlight; i++ {
		buf := make([]byte, 4096)
		dones[i] = r.SubmitAsync(fd, buf, 0)
	}

	// Close must complete in bounded time even with requests in flight.
	closeReturned := make(chan struct{})
	go func() {
		r.Close()
		close(closeReturned)
	}()

	select {
	case <-closeReturned:
		// Good — Close returned. Now verify all async callers got
		// unblocked (either real completions or shutdown errors).
	case <-time.After(3 * time.Second):
		t.Fatal("Close never returned within 3s — regression of C5 (sentinel-not-submitted deadlock)")
	}

	// Every done channel MUST receive within a bounded window — if any
	// hangs, some shutdown-cancel path missed the request.
	for i, done := range dones {
		select {
		case <-done:
			// OK — this caller was unblocked.
		case <-time.After(1 * time.Second):
			t.Fatalf("SubmitAsync[%d] channel never received — regression of C6 (mid-batch shutdown leak); req is unreachable from inflight[], reqCh, and completeLoop", i)
		}
	}
}

// Submit racing Close must return, with a result or a shutdown error. Close
// drains the request channel only once, so a request queued after that
// used to block its caller forever.
func TestBatchIoUringReader_SubmitRacingClose_NeverStrandsCaller(t *testing.T) {
	f, err := os.Create(t.TempDir() + "/data")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()
	if _, err := f.Write(make([]byte, 4096)); err != nil {
		t.Fatalf("write: %v", err)
	}
	fd := int(f.Fd())

	for iter := 0; iter < 200; iter++ {
		r, err := NewBatchIoUringReader(BatchIoUringConfig{RingDepth: 8, MaxBatch: 4, MaxInflight: 4, QueueSize: 4})
		if err != nil {
			t.Skipf("io_uring init: %v", err)
		}
		var wg sync.WaitGroup
		for g := 0; g < 16; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				buf := make([]byte, 4096)
				for {
					if _, err := r.Submit(fd, buf, 0); err != nil {
						return
					}
				}
			}()
		}
		r.Close()

		returned := make(chan struct{})
		go func() { wg.Wait(); close(returned) }()
		select {
		case <-returned:
		case <-time.After(5 * time.Second):
			t.Fatalf("iteration %d: a Submit caller is still blocked 5s after Close returned", iter)
		}
	}
}
