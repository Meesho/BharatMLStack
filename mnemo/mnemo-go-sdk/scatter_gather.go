package sdk

import (
	"context"
	"sync"
)

// scatterGather groups keys by shard, fans out one batch request per shard in
// parallel, and merges results back into the original key order.
//
// Partial failure is per-shard: a shard that fails (no pod, dial error, batch
// error) sets the Err field on only its own keys; other shards still return
// values. The returned top-level error is the first shard error encountered
// (nil if every shard succeeded).
func scatterGather(ctx context.Context, c *Client, keys [][]byte) ([]Result, error) {
	results := make([]Result, len(keys))

	type entry struct {
		idx int
		key []byte
	}
	groups := make(map[uint32][]entry)
	for i, key := range keys {
		shardID := c.router.ShardFor(key)
		groups[shardID] = append(groups[shardID], entry{idx: i, key: key})
	}

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)

	setErr := func(entries []entry, err error) {
		mu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		mu.Unlock()
		for _, e := range entries {
			results[e.idx] = Result{Key: e.key, Err: err}
		}
	}

	for shardID, entries := range groups {
		shardID, entries := shardID, entries
		wg.Add(1)
		go func() {
			defer wg.Done()

			pod, err := c.router.PodFor(shardID)
			if err != nil {
				setErr(entries, err)
				return
			}

			conn, err := c.pool.Get(pod)
			if err != nil {
				c.router.MarkUnhealthy(pod)
				setErr(entries, err)
				return
			}

			batchKeys := make([][]byte, len(entries))
			for i, e := range entries {
				batchKeys[i] = e.key
			}

			vals, err := conn.BatchLookup(ctx, batchKeys)
			if err != nil {
				_ = conn.Close() // broken connection — don't return to pool
				c.router.MarkUnhealthy(pod)
				setErr(entries, err)
				return
			}
			c.pool.Put(pod, conn)

			for i, e := range entries {
				results[e.idx] = Result{Key: e.key, Value: vals[i]}
			}
		}()
	}

	wg.Wait()
	return results, firstErr
}

// stringScatterGather is the string-key variant of scatterGather. It uses
// StringBatchLookup (opcode 0x04) instead of BatchLookup (opcode 0x02).
func stringScatterGather(ctx context.Context, c *Client, keys [][]byte) ([]Result, error) {
	results := make([]Result, len(keys))

	type entry struct {
		idx int
		key []byte
	}
	groups := make(map[uint32][]entry)
	for i, key := range keys {
		shardID := c.router.ShardFor(key)
		groups[shardID] = append(groups[shardID], entry{idx: i, key: key})
	}

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)

	setErr := func(entries []entry, err error) {
		mu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		mu.Unlock()
		for _, e := range entries {
			results[e.idx] = Result{Key: e.key, Err: err}
		}
	}

	for shardID, entries := range groups {
		shardID, entries := shardID, entries
		wg.Add(1)
		go func() {
			defer wg.Done()

			pod, err := c.router.PodFor(shardID)
			if err != nil {
				setErr(entries, err)
				return
			}

			conn, err := c.pool.Get(pod)
			if err != nil {
				c.router.MarkUnhealthy(pod)
				setErr(entries, err)
				return
			}

			batchKeys := make([][]byte, len(entries))
			for i, e := range entries {
				batchKeys[i] = e.key
			}

			vals, err := conn.StringBatchLookup(ctx, batchKeys)
			if err != nil {
				_ = conn.Close()
				c.router.MarkUnhealthy(pod)
				setErr(entries, err)
				return
			}
			c.pool.Put(pod, conn)

			for i, e := range entries {
				results[e.idx] = Result{Key: e.key, Value: vals[i]}
			}
		}()
	}

	wg.Wait()
	return results, firstErr
}
