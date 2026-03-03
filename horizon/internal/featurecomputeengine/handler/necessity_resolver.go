package handler

import (
	"context"
	"fmt"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetstate"
)

// ResolveNecessity determines the computation state for every asset in the DAG.
//
// Algorithm:
//  1. Build DAG from the database.
//  2. Load serving overrides from the asset_necessity table.
//  3. Determine effective serving for each asset:
//     if an override exists, use it; otherwise use the AssetSpec flag.
//  4. Mark all effective-serving=true assets as ACTIVE.
//  5. Walk upstream from each ACTIVE asset — any ancestor not already ACTIVE
//     is marked TRANSIENT.
//  6. Everything else is SKIPPED.
//  7. Persist results via BulkUpsert.
func (h *Handler) ResolveNecessity(ctx context.Context) (map[string]Necessity, error) {
	dag, err := h.BuildDAG(ctx)
	if err != nil {
		return nil, fmt.Errorf("building DAG: %w", err)
	}

	overrides, err := h.loadServingOverrides(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading serving overrides: %w", err)
	}

	result := resolveNecessityFromDAG(dag, overrides)

	rows := make([]assetstate.AssetNecessityRow, 0, len(result))
	for name, nec := range result {
		rows = append(rows, assetstate.AssetNecessityRow{
			AssetName: name,
			Necessity: string(nec),
			Reason:    necessityReason(nec, overrides[name] != nil),
		})
	}

	if err := h.assetState.BulkUpsert(ctx, rows); err != nil {
		return nil, fmt.Errorf("persisting necessity: %w", err)
	}

	return result, nil
}

// ResolveNecessityDryRun performs the same computation as ResolveNecessity but
// operates on a proposed set of specs and edges instead of the current database
// state. It does NOT persist results. Used by the Plan stage to preview changes.
func (h *Handler) ResolveNecessityDryRun(
	ctx context.Context,
	proposedSpecs []AssetNode,
	proposedEdges []DAGEdge,
) (map[string]Necessity, error) {
	dag := buildDAGFromNodesAndEdges(proposedSpecs, proposedEdges)

	overrides, err := h.loadServingOverrides(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading serving overrides: %w", err)
	}

	return resolveNecessityFromDAG(dag, overrides), nil
}

// resolveNecessityFromDAG runs the necessity algorithm on an arbitrary DAG
// with the given serving overrides. Pure computation — no side effects.
func resolveNecessityFromDAG(dag *LineageDAG, overrides map[string]*bool) map[string]Necessity {
	necessity := make(map[string]Necessity, len(dag.Nodes))

	for name, node := range dag.Nodes {
		serving := node.Serving
		if override, ok := overrides[name]; ok {
			serving = *override
		}
		if serving {
			necessity[name] = NecessityActive
		}
	}

	for name, nec := range necessity {
		if nec != NecessityActive {
			continue
		}
		for _, ancestor := range dag.UpstreamOf(name) {
			if necessity[ancestor] != NecessityActive {
				necessity[ancestor] = NecessityTransient
			}
		}
	}

	for name := range dag.Nodes {
		if _, set := necessity[name]; !set {
			necessity[name] = NecessitySkipped
		}
	}

	return necessity
}

// loadServingOverrides returns a map of asset_name → *bool for assets that
// have an explicit serving override in the asset_necessity table.
func (h *Handler) loadServingOverrides(ctx context.Context) (map[string]*bool, error) {
	rows, err := h.assetState.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	overrides := make(map[string]*bool, len(rows))
	for _, row := range rows {
		if row.ServingOverride != nil {
			v := *row.ServingOverride
			overrides[row.AssetName] = &v
		}
	}
	return overrides, nil
}

// loadNecessityMap returns the current necessity assignments from the database.
func (h *Handler) loadNecessityMap(ctx context.Context) (map[string]Necessity, error) {
	rows, err := h.assetState.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]Necessity, len(rows))
	for _, row := range rows {
		result[row.AssetName] = Necessity(row.Necessity)
	}
	return result, nil
}

func necessityReason(nec Necessity, hasOverride bool) string {
	switch nec {
	case NecessityActive:
		if hasOverride {
			return "serving override: true"
		}
		return "serving=true in asset spec"
	case NecessityTransient:
		return "needed by downstream active asset"
	case NecessitySkipped:
		return "not needed by any active asset"
	default:
		return ""
	}
}
