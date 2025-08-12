package hashring

import (
	"fmt"
	"hash/crc32"
	"slices"
	"sort"
)

const (
	nodeKeyFmt   = "%s:%d"
	nodeReplicas = 100
)

type ConsistentHashRing struct {
	ring       map[uint32]string
	sortedKeys []uint32
}

func NewConsistentHashRing() *ConsistentHashRing {
	return &ConsistentHashRing{
		ring:       make(map[uint32]string),
		sortedKeys: make([]uint32, 0),
	}
}

func (h *ConsistentHashRing) AddNode(node string) {
	for i := range nodeReplicas {
		nodeKey := fmt.Sprintf(nodeKeyFmt, node, i)
		h.ring[h.getHash(nodeKey)] = node
	}
	h.refreshSortedKeys()
}

func (h *ConsistentHashRing) RemoveNode(node string) {
	for i := range nodeReplicas {
		nodeKey := fmt.Sprintf(nodeKeyFmt, node, i)
		delete(h.ring, h.getHash(nodeKey))
	}
	h.refreshSortedKeys()
}

func (h *ConsistentHashRing) GetNodeForKey(key string) string {
	hash := h.getHash(key)
	idx := sort.Search(len(h.sortedKeys), func(i int) bool {
		return h.sortedKeys[i] >= hash
	})
	if idx == len(h.sortedKeys) {
		idx = 0
	}
	return h.ring[h.sortedKeys[idx]]
}

func (h *ConsistentHashRing) refreshSortedKeys() {
	h.sortedKeys = make([]uint32, 0, len(h.ring))
	for hash := range h.ring {
		h.sortedKeys = append(h.sortedKeys, hash)
	}
	slices.Sort(h.sortedKeys)
}

func (h *ConsistentHashRing) getHash(nodeKey string) uint32 {
	return crc32.ChecksumIEEE([]byte(nodeKey))
}

func (h *ConsistentHashRing) GetRingTopology() map[uint32]string {
	ringTopology := make(map[uint32]string)
	for hash, node := range h.ring {
		ringTopology[hash] = node
	}
	return ringTopology
}
