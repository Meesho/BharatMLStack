package dag

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/spaolacci/murmur3"

	"github.com/Meesho/BharatMLStack/dag-topology-executor/pkg/cache"
	"github.com/Meesho/BharatMLStack/dag-topology-executor/pkg/logger"
)

type ComponentGraphRegistry struct {
	Initializer *ComponentInitializer
	Cache       *cache.Cache
	mutex       sync.Mutex
}

func (cr *ComponentGraphRegistry) getDagTopology(dagConfig map[string][]string) (*DagTopology, error) {

	dagConfigHash := getMurmurHash(dagConfig)

	// try to get the dagTopology from the cache
	val, found := cr.Cache.Get(dagConfigHash)
	if found {
		return val.(*DagTopology), nil
	}

	// acquire a lock to prevent multiple goroutines from
	// computing dagTopology at the same time
	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	// try to get the dagTopology from cache again
	// in case another goroutine has already computed it
	val, found = cr.Cache.Get(dagConfigHash)
	if found {
		return val.(*DagTopology), nil
	}

	// compute dagTopology
	dagTopology, err := cr.Initializer.InitializeComponentDag(dagConfig)
	if err != nil {
		logger.Error("Failed to initialize dagToplogy: %v", err)
		return dagTopology, nil
	}
	// add dagTopology to the cache
	cr.Cache.SetWithTTL(dagConfigHash, dagTopology)
	return dagTopology, nil
}

func getMurmurHash(dagConfig map[string][]string) string {

	dagConfigBytes, _ := json.Marshal(dagConfig)
	dagConfigString := strings.ReplaceAll(string(dagConfigBytes), " ", "")

	h128 := murmur3.New128()
	h128.Write([]byte(dagConfigString))
	dagHash128 := h128.Sum(nil)

	var sb strings.Builder
	for _, hash64 := range dagHash128 {
		sb.WriteString(strconv.Itoa(int(hash64)))
	}

	return sb.String()
}
