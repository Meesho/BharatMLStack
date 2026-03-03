package handler

import (
	"context"
	"fmt"
	"sort"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/materializations"
)

// GetExecutionPlan generates the plan for a specific notebook + trigger + partition.
// This is called by the Airflow AssetExecutionOperator before launching Databricks.
//
// It returns which assets to execute (cache miss + necessary), which to skip
// (cache hit or unnecessary), input bindings (exact delta versions to read),
// and artifact paths (where to write output).
func (h *Handler) GetExecutionPlan(ctx context.Context, req GetExecutionPlanRequest) (*NotebookExecutionPlan, error) {
	dag, err := h.BuildDAG(ctx)
	if err != nil {
		return nil, fmt.Errorf("building DAG: %w", err)
	}

	specs, err := h.assetRegistry.GetByNotebook(ctx, req.Notebook)
	if err != nil {
		return nil, fmt.Errorf("loading notebook assets: %w", err)
	}

	necessityMap, err := h.loadNecessityMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading necessity: %w", err)
	}

	var assetPlans []AssetExecutionPlan
	sharedInputs := make(map[string]string)

	for _, spec := range specs {
		if spec.TriggerType != req.TriggerType {
			continue
		}

		nec := necessityMap[spec.AssetName]
		if nec == "" {
			nec = NecessityActive
		}

		if nec == NecessitySkipped {
			assetPlans = append(assetPlans, AssetExecutionPlan{
				AssetName: spec.AssetName,
				Action:    ActionSkipUnnecessary,
				Necessity: nec,
				Reason:    "asset necessity is skipped",
			})
			continue
		}

		computeKey, cacheHit, err := h.ResolveComputeKey(ctx, spec.AssetName, req.Partition)
		if err != nil {
			return nil, fmt.Errorf("resolving compute key for %s: %w", spec.AssetName, err)
		}

		if cacheHit {
			existingPath := ""
			mat, _ := h.materializations.GetByComputeKey(ctx, spec.AssetName, req.Partition, computeKey)
			if mat != nil {
				existingPath = mat.ArtifactPath
			}

			assetPlans = append(assetPlans, AssetExecutionPlan{
				AssetName:        spec.AssetName,
				Action:           ActionSkipCached,
				Necessity:        nec,
				ComputeKey:       computeKey,
				ExistingArtifact: existingPath,
				Reason:           "compute key cache hit",
			})
			continue
		}

		bindings, err := h.resolveInputBindings(ctx, spec.AssetName, req.Partition)
		if err != nil {
			return nil, fmt.Errorf("resolving input bindings for %s: %w", spec.AssetName, err)
		}

		artifactPath := fmt.Sprintf("%s/%s/%s/%s/",
			h.config.ArtifactBasePath, spec.AssetName, req.Partition, computeKey)

		assetPlans = append(assetPlans, AssetExecutionPlan{
			AssetName:     spec.AssetName,
			Action:        ActionExecute,
			Necessity:     nec,
			ComputeKey:    computeKey,
			InputBindings: bindings,
			ArtifactPath:  artifactPath,
			Reason:        "execution required",
		})

		for k, v := range bindings {
			sharedInputs[k] = v
		}
	}

	sortAssetPlansByDAG(assetPlans, dag)

	return &NotebookExecutionPlan{
		Notebook:     req.Notebook,
		TriggerType:  req.TriggerType,
		Partition:    req.Partition,
		Assets:       assetPlans,
		SharedInputs: sharedInputs,
	}, nil
}

// ReportAssetReady is called by the Airflow operator after an asset completes.
// It updates dataset_partitions, creates a materialization record,
// and triggers downstream evaluation.
func (h *Handler) ReportAssetReady(ctx context.Context, req ReportAssetReadyRequest) ([]TriggerAction, error) {
	if err := h.datasetPartitions.MarkReady(ctx, req.AssetName, req.Partition, req.DeltaVersion); err != nil {
		return nil, fmt.Errorf("marking partition ready: %w", err)
	}

	if err := h.materializations.Create(ctx, materializations.MaterializationRow{
		AssetName:        req.AssetName,
		PartitionKey:     req.Partition,
		ComputeKey:       req.ComputeKey,
		ArtifactPath:     req.ArtifactPath,
		InputFingerprint: req.ComputeKey,
		Status:           "succeeded",
	}); err != nil {
		return nil, fmt.Errorf("creating materialization: %w", err)
	}

	return h.EvaluateUpstreamTriggers(ctx, req.AssetName, req.Partition)
}

func (h *Handler) resolveInputBindings(ctx context.Context, assetName string, partition string) (map[string]string, error) {
	deps, err := h.assetDeps.GetDependencies(ctx, assetName)
	if err != nil {
		return nil, err
	}

	bindings := make(map[string]string, len(deps))
	for _, dep := range deps {
		dpRow, err := h.datasetPartitions.Get(ctx, dep.InputName, partition)
		if err != nil {
			bindings[dep.InputName] = ""
			continue
		}
		bindings[dep.InputName] = deltaVersionString(dpRow.DeltaVersion)
	}

	return bindings, nil
}

// sortAssetPlansByDAG sorts asset plans according to their position in the
// global topological ordering. Falls back to alphabetical on error.
func sortAssetPlansByDAG(plans []AssetExecutionPlan, dag *LineageDAG) {
	if len(plans) <= 1 {
		return
	}

	sorted, err := dag.TopologicalSort()
	if err != nil {
		sort.Slice(plans, func(i, j int) bool {
			return plans[i].AssetName < plans[j].AssetName
		})
		return
	}

	position := make(map[string]int, len(sorted))
	for i, name := range sorted {
		position[name] = i
	}

	sort.SliceStable(plans, func(i, j int) bool {
		pi, oki := position[plans[i].AssetName]
		pj, okj := position[plans[j].AssetName]
		if !oki || !okj {
			return plans[i].AssetName < plans[j].AssetName
		}
		return pi < pj
	})
}
