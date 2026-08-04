// Package placement implements the shard→pod assignment strategy for OnyxDB.
//
// V1 uses the S1 topology: each shard is a separate K8s StatefulSet.
// Pod names encode the shard they serve: {tenant}-{store}-shard-{shardID}-{replicaIdx}.
package placement

import (
	"fmt"
	"strings"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

// defaultReadServerPort is used when a pod registration omits Port (legacy pods,
// or the K8s pod-per-IP model where every read server listens on 9091).
const defaultReadServerPort = 9091

// DeriveAssignment builds the shard→pod-addresses map from current pod registrations.
// Only pods that have version in their WarmVersions are included.
// Every shard ID from 0 to shardCount-1 is present in the output (empty slice if no warm pods).
func DeriveAssignment(shardCount int, pods map[string]model.PodData, version string) map[string][]string {
	assignment := make(map[string][]string, shardCount)
	for i := 0; i < shardCount; i++ {
		assignment[fmt.Sprintf("%d", i)] = []string{}
	}

	for podID, data := range pods {
		sid := ExtractShardID(podID)
		if sid == "" {
			continue
		}
		if _, ok := assignment[sid]; !ok {
			continue // shard ID outside our range — stale pod
		}
		for _, wv := range data.WarmVersions {
			if wv == version {
				port := data.Port
				if port == 0 {
					port = defaultReadServerPort // legacy pods that don't report a port
				}
				assignment[sid] = append(assignment[sid], fmt.Sprintf("%s:%d", data.PodIP, port))
				break
			}
		}
	}
	return assignment
}

// ExtractShardID parses the shard index from a pod name.
//
// Pod names follow the convention: {prefix}-shard-{N}-{suffix}
// where N is the numeric shard index. Examples:
//
//	"fs-features-shard-0-0"  → "0"
//	"recsys-catalog-shard-12-1" → "12"
func ExtractShardID(podID string) string {
	const marker = "-shard-"
	idx := strings.Index(podID, marker)
	if idx < 0 {
		return ""
	}
	rest := podID[idx+len(marker):]
	if rest == "" {
		return ""
	}
	end := strings.IndexByte(rest, '-')
	if end < 0 {
		return rest
	}
	return rest[:end]
}
