package sdk

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A connect must never outlive the request that needs it: a peer that drops
// SYNs (a full accept queue, a vanished pod IP) used to hold a request for the
// kernel's ~1 s SYN retransmit or the 5 s DialTimeout, far past the 100 ms
// request deadline.

// blackholeDial makes every dial behave like one to a peer that never answers:
// it returns only when the dialer gives up, on its Timeout or its ctx.
func blackholeDial(t *testing.T) {
	t.Helper()
	orig := dialTCP
	dialTCP = func(ctx context.Context, d *net.Dialer, addr string) (net.Conn, error) {
		if d.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, d.Timeout)
			defer cancel()
		}
		<-ctx.Done()
		return nil, &net.OpError{Op: "dial", Net: "tcp", Err: ctx.Err()}
	}
	t.Cleanup(func() { dialTCP = orig })
}

func TestConnPool_GetContextDialStopsAtTheRequestDeadline(t *testing.T) {
	blackholeDial(t)
	p := NewConnPool(2) // DialTimeout defaults to 5 s
	defer p.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := p.GetContext(ctx, "10.0.0.1:9091")

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), time.Second, "the dial waited past the request deadline")
}

func TestConnPool_GetWithoutContextKeepsTheDialTimeout(t *testing.T) {
	blackholeDial(t)
	p := NewConnPoolWithConfig(PoolConfig{DialTimeout: 50 * time.Millisecond})
	defer p.Close()

	start := time.Now()
	_, err := p.Get("10.0.0.1:9091")

	require.Error(t, err)
	assert.Less(t, time.Since(start), time.Second)
}

func TestConnPool_GetContextDialsAndReusesAsGetDoes(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			defer c.Close()
		}
	}()
	p := NewConnPool(2)
	defer p.Close()

	conn, err := p.GetContext(context.Background(), ln.Addr().String())
	require.NoError(t, err)
	p.Put(ln.Addr().String(), conn)
	again, err := p.GetContext(context.Background(), ln.Addr().String())
	require.NoError(t, err)
	assert.Same(t, conn, again, "a pooled connection is reused, not redialled")
}

// The request paths hand their deadline to the dial: every public read fails at
// about the request timeout (100 ms by default), not the 5 s DialTimeout.
func TestClient_ReadsDoNotWaitOnADialPastTheRequestTimeout(t *testing.T) {
	blackholeDial(t)
	c := NewDirectClient("10.0.0.1:9091", 2)
	defer c.Close()
	key := make([]byte, keySize)

	reads := map[string]func() error{
		"Get":            func() error { _, err := c.Get(context.Background(), key); return err },
		"BatchGet":       func() error { _, err := c.BatchGet(context.Background(), [][]byte{key}); return err },
		"StringGet":      func() error { _, err := c.StringGet(context.Background(), []byte("k")); return err },
		"StringBatchGet": func() error { _, err := c.StringBatchGet(context.Background(), [][]byte{[]byte("k")}); return err },
	}
	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			err := read()
			require.Error(t, err)
			assert.True(t, errors.Is(err, context.DeadlineExceeded), "err = %v", err)
			assert.Less(t, time.Since(start), time.Second, "%s waited on the dial past its 100 ms timeout", name)
		})
	}
}
