// Package coverage computes warm-pod coverage for a OnyxDB version across a store's shards.
package coverage

import (
	"context"
	"fmt"
	"sort"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/internal/placement"
	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
)

// PodLister retrieves all pod registrations for a store.
type PodLister interface {
	ListPods(ctx context.Context, tenant, store string) (map[string]model.PodData, error)
}

// Result holds the coverage outcome for a version.
type Result struct {
	Total   int      // total shard count
	Warm    int      // shards with at least one warm pod
	Missing []string // shard IDs with no warm pod, sorted ascending
}

// Ratio returns the fraction of shards that are warm (0.0–1.0).
// Returns 1.0 vacuously when Total == 0.
func (r *Result) Ratio() float64 {
	if r.Total == 0 {
		return 1.0
	}
	return float64(r.Warm) / float64(r.Total)
}

// IsComplete returns true when every shard has at least one warm pod.
func (r *Result) IsComplete() bool {
	return r.Warm == r.Total
}

// Checker is the interface for checking version coverage.
type Checker interface {
	Check(ctx context.Context, tenant, store, version string, shardCount int) (*Result, error)
}

// checker implements Checker using a PodLister.
type checker struct {
	lister PodLister
}

// New returns a Checker backed by the given PodLister.
func New(lister PodLister) Checker {
	return &checker{lister: lister}
}

// Check returns coverage for version across shardCount shards of tenant/store.
func (c *checker) Check(ctx context.Context, tenant, store, version string, shardCount int) (*Result, error) {
	pods, err := c.lister.ListPods(ctx, tenant, store)
	if err != nil {
		return nil, fmt.Errorf("listing pods for %s/%s: %w", tenant, store, err)
	}

	assignment := placement.DeriveAssignment(shardCount, pods, version)

	result := &Result{Total: shardCount}
	for i := 0; i < shardCount; i++ {
		sid := fmt.Sprintf("%d", i)
		if len(assignment[sid]) > 0 {
			result.Warm++
		} else {
			result.Missing = append(result.Missing, sid)
		}
	}
	sort.Strings(result.Missing)
	return result, nil
}
