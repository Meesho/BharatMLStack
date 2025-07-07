package rowcache

import (
	"sync"

	"github.com/Meesho/BharatMLStack/ssd-cache/internal/allocator"
)

type Node struct {
	Key   string
	Value []byte
	Prev  *Node
	Next  *Node
}

type NodePool struct {
	pool sync.Pool
}

func NewNodePool() *NodePool {
	return &NodePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &Node{}
			},
		},
	}
}

func (p *NodePool) Get() *Node {
	return p.pool.Get().(*Node)
}

func (p *NodePool) Put(node *Node) {
	node.Key = ""
	node.Value = nil
	node.Prev = nil
	node.Next = nil
	p.pool.Put(node)
}

type LRUCache struct {
	Capacity          int64
	EvictionThreshold float64
	allocator         *allocator.SlabAllocator
	head              *Node
	tail              *Node
	target            *Node
	cache             map[string]*Node
	size              int64
	stats             LRUCacheStats
	nodePool          *NodePool
}

type LRUCacheStats struct {
	HitCount       int64
	MissCount      int64
	SetCount       int64
	EvictCount     int64
	EvictItemCount int64
	Size           int64
	Capacity       int64
}

type LRUCacheConfig struct {
	Capacity          int64
	EvictionThreshold float64
	SlabSizes         []int
}

type Optype int8

func NewLRUCache(config LRUCacheConfig) *LRUCache {
	allocator := allocator.NewSlabAllocator(allocator.SlabConfig{
		SizeClasses: config.SlabSizes,
	})
	cache := &LRUCache{
		Capacity:          config.Capacity,
		EvictionThreshold: config.EvictionThreshold,
		allocator:         allocator,
		head:              nil,
		tail:              nil,
		target:            nil,
		cache:             make(map[string]*Node),
		size:              0,
		stats: LRUCacheStats{
			Capacity: config.Capacity,
		},
		nodePool: NewNodePool(),
	}
	return cache
}

func (c *LRUCache) Get(key string) []byte {
	node, ok := c.cache[key]
	if !ok {
		c.stats.MissCount++
		return nil
	}
	c.moveToHead(node)
	c.stats.HitCount++
	return node.Value
}

func (c *LRUCache) moveToHead(node *Node) {
	if node == c.head {
		return
	}

	// Remove node from its current position
	if node.Prev != nil {
		node.Prev.Next = node.Next
	}
	if node.Next != nil {
		node.Next.Prev = node.Prev
	}

	// Update tail if this node was the tail
	if node == c.tail {
		c.tail = node.Prev
	}

	// Move to head
	node.Prev = nil
	node.Next = c.head
	if c.head != nil {
		c.head.Prev = node
	}
	c.head = node

	// If this was the only/first node, set it as tail too
	if c.tail == nil {
		c.tail = node
	}
}

func (c *LRUCache) Set(key string, value []byte) {
	node, ok := c.cache[key]
	if ok {
		// Update existing node
		if len(value) == len(node.Value) {
			copy(node.Value, value)
		} else if len(value) < len(node.Value) && PrevPowerOfTwo(uint32(len(value))) == PrevPowerOfTwo(uint32(len(node.Value))) {
			copy(node.Value[:len(value)], value)
			node.Value = node.Value[:len(value)]
		} else if len(value) > len(node.Value) && NextPowerOfTwo(uint32(len(value))) == NextPowerOfTwo(uint32(len(node.Value))) {
			copy(node.Value, value)
		} else {
			c.allocator.PutBuffer(node.Value)
			node.Value, _ = c.allocator.GetBuffer(len(value))
			copy(node.Value, value)
		}
		c.moveToHead(node)
		c.stats.SetCount++
		return
	}

	// Check if we need to evict before adding new item
	if c.size >= c.Capacity {
		c.Evict()
	}

	// Create new node
	v, _ := c.allocator.GetBuffer(len(value))
	copy(v, value)
	node = c.nodePool.Get()
	node.Key = key
	node.Value = v
	node.Prev = nil
	node.Next = nil

	// Add to cache
	c.cache[key] = node

	// Handle empty list
	if c.head == nil {
		c.head = node
		c.tail = node
	} else {
		// Add to head
		node.Next = c.head
		c.head.Prev = node
		c.head = node
	}

	c.size++
	c.stats.SetCount++
	c.stats.Size = c.size
}

func (c *LRUCache) Evict() {
	if c.tail == nil || c.size == 0 {
		return
	}

	count := int64(c.EvictionThreshold * float64(c.Capacity))
	if count == 0 {
		count = 1
	}
	c.stats.EvictCount++
	c.stats.EvictItemCount += count
	for count > 0 && c.tail != nil {
		nodeToEvict := c.tail

		// Move tail pointer
		c.tail = c.tail.Prev
		if c.tail != nil {
			c.tail.Next = nil
		} else {
			// List is now empty
			c.head = nil
		}

		// Clean up the evicted node
		c.allocator.PutBuffer(nodeToEvict.Value)
		delete(c.cache, nodeToEvict.Key)
		c.nodePool.Put(nodeToEvict)
		c.size--
		c.stats.Size = c.size
		count--
	}
}

func NextPowerOfTwo(n uint32) uint32 {
	if n == 0 {
		return 1
	}
	n-- // handle exact power of 2
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}

func PrevPowerOfTwo(n uint32) uint32 {
	if n == 0 {
		return 0
	}
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n - (n >> 1)
}

// Stats returns a copy of the current cache statistics
func (c *LRUCache) Stats() LRUCacheStats {
	c.stats.Size = c.size
	c.stats.Capacity = c.Capacity
	return c.stats
}
