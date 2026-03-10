package redisq

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
)

type HashQueuePartitioner struct {
	partitionCount int
}

func NewHashQueuePartitioner(partitionCount int) *HashQueuePartitioner {
	if partitionCount <= 0 {
		partitionCount = 1
	}
	return &HashQueuePartitioner{partitionCount: partitionCount}
}

func (p *HashQueuePartitioner) Partition(intent models.WatchIntent) int {
	seed := intent.Operation + "|" + intent.Resource.Namespace + "|" + intent.Resource.LabelSelector + "|" + intent.Resource.Name
	sum := sha256.Sum256([]byte(seed))
	value := binary.BigEndian.Uint32(sum[:4])
	return int(value%uint32(p.partitionCount)) + 1
}
