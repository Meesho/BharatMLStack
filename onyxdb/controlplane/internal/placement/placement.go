// Package placement implements the shard→pod assignment strategy for OnyxDB.
//
// V1 uses the S1 topology: each shard is a separate K8s StatefulSet.
// Pod names encode the shard they serve: {tenant}-{store}-shard-{shardID}-{replicaIdx}.
package placement

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
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
				assignment[sid] = append(assignment[sid], data.Addr())
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

// InCell returns the pods a store in cell may route to: every pod when cell is
// empty; otherwise the pods registered for cell and the pods that name no cell
// (a loader built before the cell field). A pod of another cell is one that
// registered under this store by mistake — a cluster deploying the wrong
// store name — and must never receive the store's reads or count as its
// coverage (docs/design/onyxdb-multi-cluster-cells.md). pods is not modified.
func InCell(pods map[string]model.PodData, cell string) map[string]model.PodData {
	if cell == "" {
		return pods
	}
	in := make(map[string]model.PodData, len(pods))
	for podID, pd := range pods {
		if pd.Cell == "" || pd.Cell == cell {
			in[podID] = pd
		}
	}
	return in
}

// OutOfCell lists the pods InCell leaves out, as "podID=cell", sorted.
func OutOfCell(pods map[string]model.PodData, cell string) []string {
	if cell == "" {
		return nil
	}
	var out []string
	for podID, pd := range pods {
		if pd.Cell != "" && pd.Cell != cell {
			out = append(out, podID+"="+pd.Cell)
		}
	}
	sort.Strings(out)
	return out
}
