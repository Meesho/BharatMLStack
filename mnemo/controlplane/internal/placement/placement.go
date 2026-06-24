// Package placement implements the shard→pod assignment strategy for mNemo.
//
// V1 uses the S1 topology: each shard is a separate K8s StatefulSet.
// Pod names encode the shard they serve: {tenant}-{store}-shard-{shardID}-{replicaIdx}.
package placement

import (
	"fmt"
	"strings"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/model"
)

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
				assignment[sid] = append(assignment[sid], data.PodIP+":9091")
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
