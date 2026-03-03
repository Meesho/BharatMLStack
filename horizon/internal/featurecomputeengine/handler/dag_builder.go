package handler

import (
	"context"
	"fmt"
	"sort"
)

// BuildDAG loads all assets and dependencies from the database and constructs
// the in-memory LineageDAG used by the Necessity Resolver and Trigger Evaluator.
//
// This is called fresh on each plan/trigger evaluation — it's NOT cached long-term.
func (h *Handler) BuildDAG(ctx context.Context) (*LineageDAG, error) {
	specs, err := h.assetRegistry.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading asset specs: %w", err)
	}

	edges, err := h.assetDeps.GetAllEdges(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading dependency edges: %w", err)
	}

	dag := &LineageDAG{
		Nodes:    make(map[string]AssetNode),
		Parents:  make(map[string][]string),
		Children: make(map[string][]string),
	}

	for _, spec := range specs {
		dag.Nodes[spec.AssetName] = AssetNode{
			AssetName:    spec.AssetName,
			EntityName:   spec.EntityName,
			EntityKey:    spec.EntityKey,
			Notebook:     spec.Notebook,
			TriggerType:  spec.TriggerType,
			Schedule:     spec.Schedule,
			Serving:      spec.Serving,
			AssetVersion: spec.AssetVersion,
		}
	}

	for _, edge := range edges {
		dag.Parents[edge.AssetName] = append(dag.Parents[edge.AssetName], edge.InputName)
		dag.Children[edge.InputName] = append(dag.Children[edge.InputName], edge.AssetName)
		dag.Edges = append(dag.Edges, DAGEdge{
			From:       edge.InputName,
			To:         edge.AssetName,
			InputType:  edge.InputType,
			WindowSize: edge.WindowSize,
		})
	}

	return dag, nil
}

// TopologicalSort returns asset names in dependency-respecting order.
// Uses Kahn's algorithm. Only includes registered assets (not external sources).
// Returns an error if a cycle is detected.
func (dag *LineageDAG) TopologicalSort() ([]string, error) {
	inDegree := make(map[string]int, len(dag.Nodes))
	for name := range dag.Nodes {
		inDegree[name] = 0
	}

	for name := range dag.Nodes {
		for _, parent := range dag.Parents[name] {
			if _, registered := dag.Nodes[parent]; registered {
				inDegree[name]++
			}
		}
	}

	queue := make([]string, 0)
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue)

	result := make([]string, 0, len(dag.Nodes))
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		for _, child := range dag.Children[node] {
			if _, registered := dag.Nodes[child]; !registered {
				continue
			}
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
				sort.Strings(queue)
			}
		}
	}

	if len(result) != len(dag.Nodes) {
		return nil, fmt.Errorf("cycle detected in DAG: processed %d of %d nodes", len(result), len(dag.Nodes))
	}

	return result, nil
}

// UpstreamOf returns all transitive ancestors of the given asset.
// Only includes registered assets (those present in Nodes).
func (dag *LineageDAG) UpstreamOf(assetName string) []string {
	visited := map[string]bool{assetName: true}
	queue := []string{assetName}

	var result []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, parent := range dag.Parents[current] {
			if visited[parent] {
				continue
			}
			visited[parent] = true
			if _, registered := dag.Nodes[parent]; registered {
				result = append(result, parent)
			}
			queue = append(queue, parent)
		}
	}

	sort.Strings(result)
	return result
}

// DownstreamOf returns all transitive descendants of the given asset.
// Only includes registered assets (those present in Nodes).
func (dag *LineageDAG) DownstreamOf(assetName string) []string {
	visited := map[string]bool{assetName: true}
	queue := []string{assetName}

	var result []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, child := range dag.Children[current] {
			if visited[child] {
				continue
			}
			visited[child] = true
			if _, registered := dag.Nodes[child]; registered {
				result = append(result, child)
			}
			queue = append(queue, child)
		}
	}

	sort.Strings(result)
	return result
}

// Validate checks for cycles, missing internal dependencies, and other
// structural issues. Returns a list of issues (empty means valid).
func (dag *LineageDAG) Validate() []string {
	var issues []string

	if _, err := dag.TopologicalSort(); err != nil {
		issues = append(issues, err.Error())
	}

	for _, edge := range dag.Edges {
		if edge.InputType == "internal" {
			if _, registered := dag.Nodes[edge.From]; !registered {
				issues = append(issues, fmt.Sprintf(
					"asset %q depends on unregistered internal asset %q", edge.To, edge.From))
			}
		}
	}

	return issues
}

// buildDAGFromNodesAndEdges constructs a LineageDAG from in-memory slices,
// bypassing the database. Used by dry-run and plan-stage operations.
func buildDAGFromNodesAndEdges(nodes []AssetNode, edges []DAGEdge) *LineageDAG {
	dag := &LineageDAG{
		Nodes:    make(map[string]AssetNode, len(nodes)),
		Parents:  make(map[string][]string),
		Children: make(map[string][]string),
		Edges:    edges,
	}

	for _, n := range nodes {
		dag.Nodes[n.AssetName] = n
	}

	for _, e := range edges {
		dag.Parents[e.To] = append(dag.Parents[e.To], e.From)
		dag.Children[e.From] = append(dag.Children[e.From], e.To)
	}

	return dag
}
