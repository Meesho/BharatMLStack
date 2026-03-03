package handler

import (
	"context"
	"sort"
)

// TriggerAction represents a notebook execution that should be triggered.
type TriggerAction struct {
	Notebook    string `json:"notebook"`
	TriggerType string `json:"trigger_type"`
	Partition   string `json:"partition"`
	AssetName   string `json:"asset_name"`
}

// EvaluateUpstreamTriggers is called when an asset reports READY.
// It checks all downstream assets with trigger="upstream" and evaluates
// whether all their inputs are now ready.
//
// For downstream assets that are cache hits, it marks them READY and
// recursively evaluates further downstream without triggering execution.
//
// Returns a list of (notebook, trigger_type) pairs that should be triggered.
func (h *Handler) EvaluateUpstreamTriggers(ctx context.Context, readyAssetName string, partition string) ([]TriggerAction, error) {
	dag, err := h.BuildDAG(ctx)
	if err != nil {
		return nil, err
	}

	necessityMap, err := h.loadNecessityMap(ctx)
	if err != nil {
		return nil, err
	}

	visited := make(map[string]bool)
	return h.evaluateTriggersWithDAG(ctx, dag, necessityMap, readyAssetName, partition, visited)
}

func (h *Handler) evaluateTriggersWithDAG(
	ctx context.Context,
	dag *LineageDAG,
	necessityMap map[string]Necessity,
	readyAssetName string,
	partition string,
	visited map[string]bool,
) ([]TriggerAction, error) {
	if visited[readyAssetName] {
		return nil, nil
	}
	visited[readyAssetName] = true

	children := dag.Children[readyAssetName]
	if len(children) == 0 {
		return nil, nil
	}

	var actions []TriggerAction
	for _, childName := range children {
		childNode, registered := dag.Nodes[childName]
		if !registered || childNode.TriggerType != "upstream" {
			continue
		}

		if necessityMap[childName] == NecessitySkipped {
			continue
		}

		allReady, err := h.areInputsReady(ctx, childName, partition, dag)
		if err != nil {
			return nil, err
		}
		if !allReady {
			continue
		}

		_, cacheHit, err := h.ResolveComputeKey(ctx, childName, partition)
		if err != nil {
			return nil, err
		}
		if cacheHit {
			_ = h.datasetPartitions.MarkReady(ctx, childName, partition, 0)
			cascaded, err := h.evaluateTriggersWithDAG(ctx, dag, necessityMap, childName, partition, visited)
			if err != nil {
				return nil, err
			}
			actions = append(actions, cascaded...)
			continue
		}

		actions = append(actions, TriggerAction{
			Notebook:    childNode.Notebook,
			TriggerType: "upstream",
			Partition:   partition,
			AssetName:   childName,
		})
	}

	return actions, nil
}

// areInputsReady checks whether all inputs for the given asset are ready at
// the specified partition. For windowed inputs, all partitions within the
// window must be ready.
func (h *Handler) areInputsReady(ctx context.Context, assetName string, partition string, dag *LineageDAG) (bool, error) {
	inputs := dag.Parents[assetName]
	if len(inputs) == 0 {
		return true, nil
	}

	edgeMap := make(map[string]DAGEdge)
	for _, edge := range dag.Edges {
		if edge.To == assetName {
			edgeMap[edge.From] = edge
		}
	}

	for _, inputName := range inputs {
		edge := edgeMap[inputName]

		if edge.WindowSize != nil && *edge.WindowSize > 1 {
			ready, err := h.isWindowedInputReady(ctx, inputName, partition, *edge.WindowSize)
			if err != nil {
				return false, err
			}
			if !ready {
				return false, nil
			}
		} else {
			ready, err := h.datasetPartitions.IsReady(ctx, inputName, partition)
			if err != nil {
				return false, err
			}
			if !ready {
				return false, nil
			}
		}
	}

	return true, nil
}

func (h *Handler) isWindowedInputReady(ctx context.Context, inputName string, partition string, windowSize int) (bool, error) {
	partitions, err := h.datasetPartitions.GetPartitions(ctx, inputName)
	if err != nil {
		return false, err
	}

	sort.Slice(partitions, func(i, j int) bool {
		return partitions[i].PartitionKey < partitions[j].PartitionKey
	})

	targetIdx := -1
	for i, p := range partitions {
		if p.PartitionKey <= partition {
			targetIdx = i
		}
	}

	if targetIdx < 0 || targetIdx-windowSize+1 < 0 {
		return false, nil
	}

	for i := targetIdx - windowSize + 1; i <= targetIdx; i++ {
		if !partitions[i].IsReady {
			return false, nil
		}
	}

	return true, nil
}
