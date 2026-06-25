// Package reconciler implements timer-based auto-promotion for mNemo versions.
//
// This is the first slice of the version rollout state machine (ADR-0009): a
// background loop that, on a fixed interval, scans every store opted into
// auto-promotion and promotes the newest READY version once every shard has a
// warm pod. It owns only the READY → ACTIVE transition; the producer-driven
// states and the richer rollout states in ADR-0009 are not implemented here.
//
// Design notes:
//   - Timer, not etcd watch: simplest to reason about and naturally tolerant of
//     late/restarting pods (each tick re-evaluates current truth).
//   - Idempotent: it only promotes a version newer than the active one, and
//     PromoteVersion flips status to ACTIVE + points activeVersion at it, so a
//     promoted version is never re-selected.
//   - Safe under races: PromoteVersion is CAS-guarded; an ErrCASConflict (a
//     concurrent manual promote, or another CP replica) is benign and retried
//     next tick.
package reconciler

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/placement"
	"github.com/Meesho/BharatMLStack/mnemo/controlplane/model"
)

// StateReader is the narrow slice of the control-plane state the reconciler
// needs. *etcdstate.EtcdStateClient satisfies it. Kept narrow so it does not
// widen the public StateClient interface and is trivially faked in tests.
type StateReader interface {
	ListStores(ctx context.Context) ([]etcdstate.StoreRef, error)
	GetStore(ctx context.Context, tenant, store string) (*etcdstate.StoreState, error)
	GetDataflow(ctx context.Context, tenant, store string) (*model.DataflowConfig, error)
	ListVersions(ctx context.Context, tenant, store string) (map[string]*model.VersionMeta, error)
	ListPods(ctx context.Context, tenant, store string) (map[string]model.PodData, error)
	PromoteVersion(ctx context.Context, tenant, store, vID string, assignment map[string][]string) error
	RetireVersion(ctx context.Context, tenant, store, vID string) error
}

// defaultKeepVersions is the retention window for auto-promote stores that don't
// set an explicit DataflowConfig.KeepVersions (active + one rollback).
const defaultKeepVersions = 2

// Reconciler periodically promotes ready, fully-warm versions.
type Reconciler struct {
	state    StateReader
	interval time.Duration
}

// New creates a Reconciler that ticks every interval.
func New(state StateReader, interval time.Duration) *Reconciler {
	return &Reconciler{state: state, interval: interval}
}

// Run blocks, reconciling once per interval, until ctx is cancelled.
func (r *Reconciler) Run(ctx context.Context) {
	log.Info().Dur("interval", r.interval).Msg("auto-promote reconciler started")
	t := time.NewTicker(r.interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("auto-promote reconciler stopping")
			return
		case <-t.C:
			r.reconcileOnce(ctx)
		}
	}
}

// reconcileOnce scans all auto-promote stores and promotes where coverage is met.
// It never returns an error: a failure on one store is logged and must not block
// the others or kill the loop.
func (r *Reconciler) reconcileOnce(ctx context.Context) {
	stores, err := r.state.ListStores(ctx)
	if err != nil {
		log.Error().Err(err).Msg("reconciler: failed to list stores")
		return
	}

	for _, ref := range stores {
		r.reconcileStore(ctx, ref)
	}
}

func (r *Reconciler) reconcileStore(ctx context.Context, ref etcdstate.StoreRef) {
	lg := log.With().Str("tenant", ref.Tenant).Str("store", ref.Store).Logger()

	df, err := r.state.GetDataflow(ctx, ref.Tenant, ref.Store)
	if err != nil || df == nil || !df.AutoPromote {
		return // not opted in (or no dataflow config yet)
	}

	st, err := r.state.GetStore(ctx, ref.Tenant, ref.Store)
	if err != nil {
		lg.Error().Err(err).Msg("reconciler: GetStore failed")
		return
	}

	versions, err := r.state.ListVersions(ctx, ref.Tenant, ref.Store)
	if err != nil {
		lg.Error().Err(err).Msg("reconciler: ListVersions failed")
		return
	}

	candidate := newestPromotable(versions, st.ActiveVersion)
	if candidate != "" {
		pods, err := r.state.ListPods(ctx, ref.Tenant, ref.Store)
		if err != nil {
			lg.Error().Err(err).Msg("reconciler: ListPods failed")
			return
		}

		assignment := placement.DeriveAssignment(st.Config.ShardCount, pods, candidate)
		if missing := uncoveredShards(st.Config.ShardCount, assignment); len(missing) > 0 {
			lg.Info().
				Str("version", candidate).
				Int("warmShards", st.Config.ShardCount-len(missing)).
				Int("shardCount", st.Config.ShardCount).
				Msg("reconciler: coverage incomplete, deferring promote")
		} else if err := r.state.PromoteVersion(ctx, ref.Tenant, ref.Store, candidate, assignment); err != nil {
			if err == etcdstate.ErrCASConflict {
				lg.Info().Str("version", candidate).Msg("reconciler: promote raced (CAS), will retry next tick")
			} else {
				lg.Error().Err(err).Str("version", candidate).Msg("reconciler: promote failed")
			}
			return // re-evaluate (incl. GC) next tick once the active version is settled
		} else {
			lg.Info().Str("version", candidate).Msg("reconciler: auto-promoted version")
			// Re-read state next tick so GC sees the new active/rollback.
			return
		}
	}

	// Keep-last-N GC: retire versions outside the retention window so pods free
	// their SSTs. Runs against the *current* active/rollback (re-read each tick).
	r.gcStore(ctx, ref, st, versions, effectiveKeep(df))
}

// effectiveKeep resolves the retention window for a store. An explicit
// KeepVersions wins; otherwise auto-promote stores default to keeping 2.
func effectiveKeep(df *model.DataflowConfig) int {
	if df.KeepVersions > 0 {
		return df.KeepVersions
	}
	if df.AutoPromote {
		return defaultKeepVersions
	}
	return 0 // GC disabled for unmanaged stores
}

// gcStore retires every version outside the keep window for the store.
func (r *Reconciler) gcStore(ctx context.Context, ref etcdstate.StoreRef, st *etcdstate.StoreState, versions map[string]*model.VersionMeta, keep int) {
	if keep <= 0 || st.ActiveVersion == "" {
		return // GC disabled, or nothing promoted yet (never delete pre-first-active)
	}
	for _, vID := range versionsToRetire(versions, st.ActiveVersion, st.RollbackVersion, keep) {
		if err := r.state.RetireVersion(ctx, ref.Tenant, ref.Store, vID); err != nil {
			log.Error().Err(err).Str("tenant", ref.Tenant).Str("store", ref.Store).
				Str("version", vID).Msg("reconciler: retire failed")
			continue
		}
		log.Info().Str("tenant", ref.Tenant).Str("store", ref.Store).
			Str("version", vID).Msg("reconciler: retired version (keep-last-N GC)")
	}
}

// newestPromotable returns the highest READY version ID strictly greater than the
// active version, or "" if none. Version IDs are {date}_{run}, which sort
// lexicographically, so plain string comparison gives chronological order.
func newestPromotable(versions map[string]*model.VersionMeta, active string) string {
	best := ""
	for vID, meta := range versions {
		if meta == nil || meta.Status != model.StatusReady {
			continue
		}
		if vID <= active {
			continue // already active, or older than active
		}
		if vID > best {
			best = vID
		}
	}
	return best
}

// versionsToRetire returns the version IDs that fall outside the keep window and
// should be retired. The keep set is:
//   - the active version and the rollback version (always kept), plus
//   - every version newer than active (in-flight: ALLOCATED/INGESTING/READY), plus
//   - the newest `keep` versions at or below active (active + its rollback chain).
//
// Versions already RETIRING are skipped (no re-emit). Version IDs sort
// lexicographically by {date}_{run}, giving chronological order.
func versionsToRetire(versions map[string]*model.VersionMeta, active, rollback string, keep int) []string {
	ids := make([]string, 0, len(versions))
	for vID := range versions {
		ids = append(ids, vID)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ids))) // newest first

	keepSet := map[string]struct{}{active: {}, rollback: {}}
	belowOrEqual := 0
	for _, vID := range ids {
		if vID > active {
			keepSet[vID] = struct{}{} // in-flight future version
			continue
		}
		if belowOrEqual < keep {
			keepSet[vID] = struct{}{}
			belowOrEqual++
		}
	}

	var retire []string
	for _, vID := range ids {
		if vID == "" {
			continue
		}
		if _, kept := keepSet[vID]; kept {
			continue
		}
		if m := versions[vID]; m != nil && m.Status == model.StatusRetiring {
			continue // already retiring
		}
		retire = append(retire, vID)
	}
	return retire
}

// uncoveredShards returns the shard IDs (as strings) that have no assigned pod.
func uncoveredShards(shardCount int, assignment map[string][]string) []string {
	var missing []string
	for i := 0; i < shardCount; i++ {
		sid := strconv.Itoa(i)
		if len(assignment[sid]) == 0 {
			missing = append(missing, sid)
		}
	}
	return missing
}
