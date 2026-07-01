package sdk

import (
	"context"
	"encoding/binary"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// metricRecord captures a single emitted metric.
type metricRecord struct {
	Name  string
	Tags  []string
	Value interface{} // time.Duration for timing, int64 for count
}

// metricCollector captures all emitted metrics for assertion.
type metricCollector struct {
	mu      sync.Mutex
	records []metricRecord
}

func (mc *metricCollector) timing(name string, value time.Duration, tags []string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.records = append(mc.records, metricRecord{Name: name, Tags: tags, Value: value})
}

func (mc *metricCollector) count(name string, value int64, tags []string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.records = append(mc.records, metricRecord{Name: name, Tags: tags, Value: value})
}

func (mc *metricCollector) findAll(name string) []metricRecord {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	var found []metricRecord
	for _, r := range mc.records {
		if r.Name == name {
			found = append(found, r)
		}
	}
	return found
}

func (mc *metricCollector) reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.records = nil
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// startMetricFakeServer starts a TCP server that responds to single-key lookups
// (opcode 0x01) and string single lookups (opcode 0x03).
func startMetricFakeServer(t *testing.T, data map[string][]byte) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleMetricConn(conn, data)
		}
	}()
	return ln.Addr().String()
}

func handleMetricConn(conn net.Conn, data map[string][]byte) {
	defer conn.Close()
	buf := make([]byte, 4096)
	for {
		// Read opcode
		n, err := conn.Read(buf)
		if err != nil || n == 0 {
			return
		}
		op := buf[0]
		switch op {
		case opSingle: // 0x01 — 12-byte key
			if n < 1+keySize {
				return
			}
			key := string(buf[1 : 1+keySize])
			val, ok := data[key]
			if !ok {
				conn.Write([]byte{0}) // not found
			} else {
				resp := make([]byte, 1+4+len(val))
				resp[0] = 1
				binary.BigEndian.PutUint32(resp[1:5], uint32(len(val)))
				copy(resp[5:], val)
				conn.Write(resp)
			}
		case opStringSingle: // 0x03 — variable-length key
			if n < 3 {
				return
			}
			keyLen := int(binary.BigEndian.Uint16(buf[1:3]))
			if n < 3+keyLen {
				return
			}
			key := string(buf[3 : 3+keyLen])
			val, ok := data[key]
			if !ok {
				conn.Write([]byte{0})
			} else {
				resp := make([]byte, 1+4+len(val))
				resp[0] = 1
				binary.BigEndian.PutUint32(resp[1:5], uint32(len(val)))
				copy(resp[5:], val)
				conn.Write(resp)
			}
		default:
			return
		}
	}
}

func TestMetrics_NilCallbacksSafe(t *testing.T) {
	// Verify no panic when Timing/Count are nil.
	c := NewDirectClient("127.0.0.1:1", 4)
	defer c.Close()

	c.emitTiming(MetricRequestLatency, time.Millisecond, []string{"op:single"})
	c.emitCount(MetricRequestCount, 1, []string{"op:single"})
	// No panic = pass.
}

func TestMetrics_Get_Hit(t *testing.T) {
	k := make([]byte, keySize)
	copy(k, "hello-metric")
	data := map[string][]byte{string(k): []byte("world")}
	addr := startMetricFakeServer(t, data)

	mc := &metricCollector{}
	c := NewDirectClient(addr, 4)
	c.config.Timing = mc.timing
	c.config.Count = mc.count
	c.config.Tenant = "test-tenant"
	c.config.Store = "test-store"
	defer c.Close()

	val, err := c.Get(context.Background(), k)
	require.NoError(t, err)
	assert.Equal(t, []byte("world"), val)

	// Check request latency emitted
	latencies := mc.findAll(MetricRequestLatency)
	require.Len(t, latencies, 1)
	assert.True(t, hasTag(latencies[0].Tags, "op:single"))
	assert.True(t, hasTag(latencies[0].Tags, "status:hit"))
	assert.True(t, hasTag(latencies[0].Tags, "tenant:test-tenant"))
	assert.True(t, hasTag(latencies[0].Tags, "store:test-store"))

	// Check request count emitted
	counts := mc.findAll(MetricRequestCount)
	require.Len(t, counts, 1)
	assert.Equal(t, int64(1), counts[0].Value)
	assert.True(t, hasTag(counts[0].Tags, "status:hit"))
}

func TestMetrics_Get_Miss(t *testing.T) {
	addr := startMetricFakeServer(t, map[string][]byte{})

	mc := &metricCollector{}
	c := NewDirectClient(addr, 4)
	c.config.Timing = mc.timing
	c.config.Count = mc.count
	defer c.Close()

	_, err := c.Get(context.Background(), make([]byte, keySize))
	assert.ErrorIs(t, err, ErrKeyNotFound)

	counts := mc.findAll(MetricRequestCount)
	require.Len(t, counts, 1)
	assert.True(t, hasTag(counts[0].Tags, "status:miss"))
}

func TestMetrics_Get_Error(t *testing.T) {
	// Point at a dead address → pool dial error → status:error
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	deadAddr := ln.Addr().String()
	ln.Close()

	mc := &metricCollector{}
	c := NewDirectClient(deadAddr, 4)
	c.config.Timing = mc.timing
	c.config.Count = mc.count
	defer c.Close()

	_, err := c.Get(context.Background(), make([]byte, keySize))
	assert.Error(t, err)

	counts := mc.findAll(MetricRequestCount)
	require.Len(t, counts, 1)
	assert.True(t, hasTag(counts[0].Tags, "status:error"))
}

func TestMetrics_StringGet_Hit(t *testing.T) {
	key := "entity:123"
	data := map[string][]byte{key: []byte("value")}
	addr := startMetricFakeServer(t, data)

	mc := &metricCollector{}
	c := NewDirectClient(addr, 4)
	c.config.Timing = mc.timing
	c.config.Count = mc.count
	defer c.Close()

	val, err := c.StringGet(context.Background(), []byte(key))
	require.NoError(t, err)
	assert.Equal(t, []byte("value"), val)

	counts := mc.findAll(MetricRequestCount)
	require.Len(t, counts, 1)
	assert.True(t, hasTag(counts[0].Tags, "op:string_single"))
	assert.True(t, hasTag(counts[0].Tags, "status:hit"))
}

func TestMetrics_Pool_DialEmitted(t *testing.T) {
	k := make([]byte, keySize)
	copy(k, "pool-metric")
	data := map[string][]byte{string(k): []byte("v")}
	addr := startMetricFakeServer(t, data)

	mc := &metricCollector{}
	c := NewDirectClient(addr, 4)
	c.config.Timing = mc.timing
	c.config.Count = mc.count
	c.config.Tenant = "t"
	c.config.Store = "s"
	c.pool.SetMetrics(mc.timing, mc.count, []string{"tenant:t", "store:s"})
	defer c.Close()

	// First Get → pool miss → dial
	_, err := c.Get(context.Background(), k)
	require.NoError(t, err)

	poolGets := mc.findAll(MetricPoolGet)
	require.NotEmpty(t, poolGets)
	assert.True(t, hasTag(poolGets[0].Tags, "result:dial"))

	// Dial latency should be emitted
	dialLats := mc.findAll(MetricPoolDialLatency)
	require.NotEmpty(t, dialLats)
}

func TestMetrics_Pool_HitAfterReturn(t *testing.T) {
	k := make([]byte, keySize)
	copy(k, "pool-reuse")
	data := map[string][]byte{string(k): []byte("v")}
	addr := startMetricFakeServer(t, data)

	mc := &metricCollector{}
	c := NewDirectClient(addr, 4)
	c.config.Timing = mc.timing
	c.config.Count = mc.count
	c.pool.SetMetrics(mc.timing, mc.count, []string{"tenant:t", "store:s"})
	defer c.Close()

	// First Get → dial
	_, err := c.Get(context.Background(), k)
	require.NoError(t, err)

	mc.reset()

	// Second Get → should reuse pooled connection
	_, err = c.Get(context.Background(), k)
	require.NoError(t, err)

	poolGets := mc.findAll(MetricPoolGet)
	require.NotEmpty(t, poolGets)
	assert.True(t, hasTag(poolGets[0].Tags, "result:hit"))
}

func TestMetrics_TagsFormat(t *testing.T) {
	mc := &metricCollector{}
	c := &Client{
		config: Config{
			Tenant: "recsys",
			Store:  "catalog",
			Timing: mc.timing,
			Count:  mc.count,
		},
	}

	tags := c.opTags("single", "hit")
	assert.Equal(t, []string{
		"tenant:recsys",
		"store:catalog",
		"op:single",
		"status:hit",
	}, tags)

	base := c.baseTags()
	assert.Equal(t, []string{
		"tenant:recsys",
		"store:catalog",
	}, base)
}
