package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetdeps"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetregistry"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetstate"
)

// ComputePlan takes proposed AssetSpecs and computes the impact of applying
// them versus the current registered state. It diffs assets, resolves proposed
// necessity, and estimates the recomputation cost.
func (h *Handler) ComputePlan(ctx context.Context, req ProposePlanRequest) (*PlanResult, error) {
	currentSpecs, err := h.assetRegistry.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading current specs: %w", err)
	}

	currentEdges, err := h.assetDeps.GetAllEdges(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading current edges: %w", err)
	}

	currentMap := make(map[string]assetregistry.AssetSpecRow, len(currentSpecs))
	for _, s := range currentSpecs {
		currentMap[s.AssetName] = s
	}

	currentInputMap := make(map[string]map[string]bool)
	for _, e := range currentEdges {
		if currentInputMap[e.AssetName] == nil {
			currentInputMap[e.AssetName] = make(map[string]bool)
		}
		currentInputMap[e.AssetName][e.InputName] = true
	}

	proposedMap := make(map[string]AssetSpecPayload, len(req.Assets))
	proposedInputMap := make(map[string]map[string]bool)
	for _, a := range req.Assets {
		proposedMap[a.Name] = a
		proposedInputMap[a.Name] = make(map[string]bool)
		for _, inp := range a.Inputs {
			proposedInputMap[a.Name][inp.Name] = true
		}
	}

	var changes []AssetChange

	for name, proposed := range proposedMap {
		current, exists := currentMap[name]
		if !exists {
			changes = append(changes, AssetChange{
				AssetName:  name,
				ChangeType: "added",
			})
			continue
		}

		added, removed := diffInputSets(currentInputMap[name], proposedInputMap[name])
		codeChanged := current.AssetVersion != proposed.Version

		if codeChanged || len(added) > 0 || len(removed) > 0 {
			changes = append(changes, AssetChange{
				AssetName:     name,
				ChangeType:    "modified",
				InputsAdded:   added,
				InputsRemoved: removed,
				CodeChanged:   codeChanged,
			})
		}
	}

	for name := range currentMap {
		if _, exists := proposedMap[name]; !exists {
			changes = append(changes, AssetChange{
				AssetName:  name,
				ChangeType: "removed",
			})
		}
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].AssetName < changes[j].AssetName
	})

	var proposedNodes []AssetNode
	var proposedEdges []DAGEdge
	for _, a := range req.Assets {
		proposedNodes = append(proposedNodes, payloadToNode(a))
		proposedEdges = append(proposedEdges, payloadToEdges(a)...)
	}

	currentNecessity, err := h.loadNecessityMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading current necessity: %w", err)
	}

	proposedNecessity, err := h.ResolveNecessityDryRun(ctx, proposedNodes, proposedEdges)
	if err != nil {
		return nil, fmt.Errorf("resolving proposed necessity: %w", err)
	}

	necessityChanges := computeNecessityDiff(currentNecessity, proposedNecessity)

	proposedDAG := buildDAGFromNodesAndEdges(proposedNodes, proposedEdges)

	var directImpact []RecomputationImpact
	for _, ch := range changes {
		if ch.ChangeType == "removed" {
			continue
		}
		directImpact = append(directImpact, RecomputationImpact{
			AssetName:          ch.AssetName,
			Reason:             ch.ChangeType,
			PartitionsAffected: 1,
		})
	}

	var cascadeImpact []RecomputationImpact
	for _, ch := range changes {
		if ch.ChangeType != "modified" {
			continue
		}
		for _, ds := range proposedDAG.DownstreamOf(ch.AssetName) {
			cascadeImpact = append(cascadeImpact, RecomputationImpact{
				AssetName:          ds,
				Reason:             "cascade",
				TriggeredBy:        ch.AssetName,
				PartitionsAffected: 1,
			})
		}
	}

	return &PlanResult{
		PlanID:           uuid.New().String(),
		Changes:          changes,
		NecessityChanges: necessityChanges,
		DirectImpact:     directImpact,
		CascadeImpact:    cascadeImpact,
		TotalPartitions:  len(directImpact) + len(cascadeImpact),
	}, nil
}

// ApplyPlan registers the proposed asset specs, updates dependencies,
// removes deleted assets, and re-runs necessity resolution.
func (h *Handler) ApplyPlan(ctx context.Context, req ProposePlanRequest) error {
	currentSpecs, err := h.assetRegistry.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("loading current specs: %w", err)
	}

	currentNames := make(map[string]bool, len(currentSpecs))
	for _, s := range currentSpecs {
		currentNames[s.AssetName] = true
	}

	proposedNames := make(map[string]bool, len(req.Assets))
	rows := make([]assetregistry.AssetSpecRow, 0, len(req.Assets))
	for _, a := range req.Assets {
		proposedNames[a.Name] = true
		rows = append(rows, payloadToSpecRow(a))
	}

	if err := h.assetRegistry.BulkUpsert(ctx, rows); err != nil {
		return fmt.Errorf("upserting asset specs: %w", err)
	}

	for _, a := range req.Assets {
		depRows := payloadToDepRows(a)
		if err := h.assetDeps.ReplaceForAsset(ctx, a.Name, depRows); err != nil {
			return fmt.Errorf("replacing dependencies for %s: %w", a.Name, err)
		}
	}

	for name := range currentNames {
		if proposedNames[name] {
			continue
		}
		if err := h.assetDeps.DeleteForAsset(ctx, name); err != nil {
			return fmt.Errorf("deleting dependencies for %s: %w", name, err)
		}
		if err := h.assetRegistry.Delete(ctx, name); err != nil {
			return fmt.Errorf("deleting asset spec %s: %w", name, err)
		}
	}

	if _, err := h.ResolveNecessity(ctx); err != nil {
		return fmt.Errorf("re-resolving necessity: %w", err)
	}

	return nil
}

func diffInputSets(current, proposed map[string]bool) (added, removed []string) {
	for name := range proposed {
		if !current[name] {
			added = append(added, name)
		}
	}
	for name := range current {
		if !proposed[name] {
			removed = append(removed, name)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return
}

func computeNecessityDiff(current, proposed map[string]Necessity) map[string][2]string {
	all := make(map[string]bool)
	for n := range current {
		all[n] = true
	}
	for n := range proposed {
		all[n] = true
	}

	diff := make(map[string][2]string)
	for name := range all {
		before := string(current[name])
		after := string(proposed[name])
		if before == "" {
			before = "none"
		}
		if after == "" {
			after = "none"
		}
		if before != after {
			diff[name] = [2]string{before, after}
		}
	}
	return diff
}

func payloadToNode(p AssetSpecPayload) AssetNode {
	return AssetNode{
		AssetName:    p.Name,
		EntityName:   p.Entity,
		EntityKey:    p.EntityKey,
		Notebook:     p.Notebook,
		TriggerType:  p.Trigger,
		Schedule:     p.Schedule,
		Serving:      p.Serving,
		AssetVersion: p.Version,
	}
}

func payloadToEdges(p AssetSpecPayload) []DAGEdge {
	edges := make([]DAGEdge, len(p.Inputs))
	for i, inp := range p.Inputs {
		edges[i] = DAGEdge{
			From:       inp.Name,
			To:         p.Name,
			InputType:  inp.Type,
			WindowSize: inp.WindowSize,
		}
	}
	return edges
}

func payloadToSpecRow(p AssetSpecPayload) assetregistry.AssetSpecRow {
	var checksJSON *string
	if len(p.Checks) > 0 {
		b, _ := json.Marshal(p.Checks)
		s := string(b)
		checksJSON = &s
	}

	specBytes, _ := json.Marshal(p)

	return assetregistry.AssetSpecRow{
		AssetName:    p.Name,
		AssetVersion: p.Version,
		EntityName:   p.Entity,
		EntityKey:    p.EntityKey,
		Notebook:     p.Notebook,
		PartitionKey: p.Partition,
		TriggerType:  p.Trigger,
		Schedule:     p.Schedule,
		Serving:      p.Serving,
		Incremental:  p.Incremental,
		Freshness:    p.Freshness,
		ChecksJSON:   checksJSON,
		SpecJSON:     string(specBytes),
	}
}

func payloadToDepRows(p AssetSpecPayload) []assetdeps.AssetDependencyRow {
	rows := make([]assetdeps.AssetDependencyRow, len(p.Inputs))
	for i, inp := range p.Inputs {
		rows[i] = assetdeps.AssetDependencyRow{
			AssetName:    p.Name,
			InputName:    inp.Name,
			InputType:    inp.Type,
			PartitionKey: inp.Partition,
			WindowSize:   inp.WindowSize,
		}
	}
	return rows
}

// BulkUpsertNecessity is a convenience method to persist necessity rows from plan results.
func (h *Handler) bulkUpsertNecessity(ctx context.Context, necessity map[string]Necessity) error {
	rows := make([]assetstate.AssetNecessityRow, 0, len(necessity))
	for name, nec := range necessity {
		rows = append(rows, assetstate.AssetNecessityRow{
			AssetName: name,
			Necessity: string(nec),
			Reason:    necessityReason(nec, false),
		})
	}
	return h.assetState.BulkUpsert(ctx, rows)
}
