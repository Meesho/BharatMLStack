package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetregistry"
	"gorm.io/gorm"
)

type ReportAssetFailedRequest struct {
	AssetName string `json:"asset_name" validate:"required"`
	Partition string `json:"partition" validate:"required"`
	Error     string `json:"error" validate:"required"`
}

type MarkDatasetReadyRequest struct {
	DatasetName  string `json:"dataset_name" validate:"required"`
	Partition    string `json:"partition" validate:"required"`
	DeltaVersion int64  `json:"delta_version"`
}

type LineageResponse struct {
	Nodes []AssetNode `json:"nodes"`
	Edges []DAGEdge   `json:"edges"`
}

type DatasetPartitionInfo struct {
	DatasetName  string `json:"dataset_name"`
	PartitionKey string `json:"partition_key"`
	DeltaVersion *int64 `json:"delta_version"`
	IsReady      bool   `json:"is_ready"`
}

// RegisterAssets upserts the given asset specs and their dependencies,
// then re-resolves necessity. Returns the count of registered assets.
func (h *Handler) RegisterAssets(ctx context.Context, req RegisterAssetsRequest) (int, error) {
	rows := make([]assetregistry.AssetSpecRow, 0, len(req.Assets))
	for _, a := range req.Assets {
		rows = append(rows, payloadToSpecRow(a))
	}

	if err := h.assetRegistry.BulkUpsert(ctx, rows); err != nil {
		return 0, fmt.Errorf("upserting asset specs: %w", err)
	}

	for _, a := range req.Assets {
		depRows := payloadToDepRows(a)
		if err := h.assetDeps.ReplaceForAsset(ctx, a.Name, depRows); err != nil {
			return 0, fmt.Errorf("replacing dependencies for %s: %w", a.Name, err)
		}
	}

	if _, err := h.ResolveNecessity(ctx); err != nil {
		return 0, fmt.Errorf("resolving necessity after registration: %w", err)
	}

	return len(req.Assets), nil
}

func (h *Handler) GetAllNecessity(ctx context.Context) (map[string]Necessity, error) {
	return h.loadNecessityMap(ctx)
}

// GetNecessity returns the necessity for a single asset, or nil if not found.
func (h *Handler) GetNecessity(ctx context.Context, assetName string) (*Necessity, error) {
	row, err := h.assetState.Get(ctx, assetName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	nec := Necessity(row.Necessity)
	return &nec, nil
}

func (h *Handler) SetServingOverride(ctx context.Context, req SetServingOverrideRequest) error {
	return h.assetState.SetServingOverride(ctx, req.AssetName, req.Serving, req.Reason, req.UpdatedBy)
}

func (h *Handler) ClearServingOverride(ctx context.Context, assetName string) error {
	return h.assetState.ClearServingOverride(ctx, assetName)
}

func (h *Handler) GetFullLineage(ctx context.Context) (*LineageResponse, error) {
	dag, err := h.BuildDAG(ctx)
	if err != nil {
		return nil, fmt.Errorf("building DAG: %w", err)
	}

	nodes := make([]AssetNode, 0, len(dag.Nodes))
	for _, n := range dag.Nodes {
		nodes = append(nodes, n)
	}

	return &LineageResponse{Nodes: nodes, Edges: dag.Edges}, nil
}

func (h *Handler) GetUpstream(ctx context.Context, assetName string) ([]string, error) {
	dag, err := h.BuildDAG(ctx)
	if err != nil {
		return nil, fmt.Errorf("building DAG: %w", err)
	}
	if _, exists := dag.Nodes[assetName]; !exists {
		return nil, fmt.Errorf("asset %q not found", assetName)
	}
	return dag.UpstreamOf(assetName), nil
}

func (h *Handler) GetDownstream(ctx context.Context, assetName string) ([]string, error) {
	dag, err := h.BuildDAG(ctx)
	if err != nil {
		return nil, fmt.Errorf("building DAG: %w", err)
	}
	if _, exists := dag.Nodes[assetName]; !exists {
		return nil, fmt.Errorf("asset %q not found", assetName)
	}
	return dag.DownstreamOf(assetName), nil
}

// ReportAssetFailed marks the latest materialization for the asset/partition as failed.
func (h *Handler) ReportAssetFailed(ctx context.Context, req ReportAssetFailedRequest) error {
	mat, err := h.materializations.GetLatest(ctx, req.AssetName, req.Partition)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("finding materialization: %w", err)
	}
	return h.materializations.UpdateStatus(ctx, mat.ID, "failed")
}

// MarkDatasetReady marks a dataset partition as ready and evaluates
// downstream triggers, returning any actions that should be fired.
func (h *Handler) MarkDatasetReady(ctx context.Context, req MarkDatasetReadyRequest) ([]TriggerAction, error) {
	if err := h.datasetPartitions.MarkReady(ctx, req.DatasetName, req.Partition, req.DeltaVersion); err != nil {
		return nil, fmt.Errorf("marking dataset ready: %w", err)
	}
	return h.EvaluateUpstreamTriggers(ctx, req.DatasetName, req.Partition)
}

func (h *Handler) GetDatasetPartitions(ctx context.Context, datasetName string) ([]DatasetPartitionInfo, error) {
	rows, err := h.datasetPartitions.GetPartitions(ctx, datasetName)
	if err != nil {
		return nil, fmt.Errorf("getting partitions: %w", err)
	}
	result := make([]DatasetPartitionInfo, len(rows))
	for i, r := range rows {
		result[i] = DatasetPartitionInfo{
			DatasetName:  r.DatasetName,
			PartitionKey: r.PartitionKey,
			DeltaVersion: r.DeltaVersion,
			IsReady:      r.IsReady,
		}
	}
	return result, nil
}
