package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"gorm.io/gorm"
)

// ComputeKeyInput captures everything that affects an asset's output.
// The compute key is a deterministic hash of these fields, enabling
// content-addressed caching: identical inputs always produce the same key.
type ComputeKeyInput struct {
	AssetVersion   string
	InputVersions  map[string]string // input_name → delta_version (as string)
	Params         map[string]string
	EnvFingerprint string
}

// GenerateComputeKey creates a deterministic SHA-256 hash from the inputs,
// truncated to 16 hex characters.
func GenerateComputeKey(input ComputeKeyInput) string {
	h := sha256.New()

	fmt.Fprintf(h, "v=%s\n", input.AssetVersion)

	inputKeys := make([]string, 0, len(input.InputVersions))
	for k := range input.InputVersions {
		inputKeys = append(inputKeys, k)
	}
	sort.Strings(inputKeys)
	for _, k := range inputKeys {
		fmt.Fprintf(h, "i:%s=%s\n", k, input.InputVersions[k])
	}

	paramKeys := make([]string, 0, len(input.Params))
	for k := range input.Params {
		paramKeys = append(paramKeys, k)
	}
	sort.Strings(paramKeys)
	for _, k := range paramKeys {
		fmt.Fprintf(h, "p:%s=%s\n", k, input.Params[k])
	}

	fmt.Fprintf(h, "e=%s\n", input.EnvFingerprint)

	return hex.EncodeToString(h.Sum(nil))[:16]
}

// ResolveComputeKey resolves the compute key for an asset at a given partition.
// It loads the asset spec, its dependencies, and the current delta versions
// of all inputs to produce a deterministic key.
//
// Returns (computeKey, cacheHit, error) where cacheHit is true when a
// successful materialization already exists for this key.
func (h *Handler) ResolveComputeKey(ctx context.Context, assetName string, partition string) (string, bool, error) {
	spec, err := h.assetRegistry.Get(ctx, assetName)
	if err != nil {
		return "", false, fmt.Errorf("loading asset spec for %s: %w", assetName, err)
	}

	deps, err := h.assetDeps.GetDependencies(ctx, assetName)
	if err != nil {
		return "", false, fmt.Errorf("loading dependencies for %s: %w", assetName, err)
	}

	inputVersions := make(map[string]string, len(deps))
	for _, dep := range deps {
		if dep.WindowSize != nil && *dep.WindowSize > 1 {
			if err := h.collectWindowedVersions(ctx, dep.InputName, partition, *dep.WindowSize, inputVersions); err != nil {
				return "", false, err
			}
		} else {
			dpRow, err := h.datasetPartitions.Get(ctx, dep.InputName, partition)
			if err != nil {
				inputVersions[dep.InputName] = "0"
				continue
			}
			inputVersions[dep.InputName] = deltaVersionString(dpRow.DeltaVersion)
		}
	}

	ckInput := ComputeKeyInput{
		AssetVersion:   spec.AssetVersion,
		InputVersions:  inputVersions,
		Params:         make(map[string]string),
		EnvFingerprint: "",
	}

	computeKey := GenerateComputeKey(ckInput)

	mat, err := h.materializations.GetByComputeKey(ctx, assetName, partition, computeKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return computeKey, false, nil
		}
		return "", false, fmt.Errorf("checking materialization cache: %w", err)
	}

	return computeKey, mat.Status == "succeeded", nil
}

// collectWindowedVersions retrieves delta versions for all partitions within
// the window and adds them to the inputVersions map. Keys are formatted as
// "inputName@partitionKey" to distinguish per-partition contributions.
func (h *Handler) collectWindowedVersions(
	ctx context.Context,
	inputName string,
	targetPartition string,
	windowSize int,
	inputVersions map[string]string,
) error {
	partitions, err := h.datasetPartitions.GetPartitions(ctx, inputName)
	if err != nil {
		return fmt.Errorf("loading partitions for windowed input %s: %w", inputName, err)
	}

	sort.Slice(partitions, func(i, j int) bool {
		return partitions[i].PartitionKey < partitions[j].PartitionKey
	})

	targetIdx := -1
	for i, p := range partitions {
		if p.PartitionKey <= targetPartition {
			targetIdx = i
		}
	}

	if targetIdx < 0 {
		return nil
	}

	windowStart := targetIdx - windowSize + 1
	if windowStart < 0 {
		windowStart = 0
	}

	for i := windowStart; i <= targetIdx; i++ {
		key := fmt.Sprintf("%s@%s", inputName, partitions[i].PartitionKey)
		inputVersions[key] = deltaVersionString(partitions[i].DeltaVersion)
	}

	return nil
}

func deltaVersionString(v *int64) string {
	if v == nil {
		return "0"
	}
	return strconv.FormatInt(*v, 10)
}
