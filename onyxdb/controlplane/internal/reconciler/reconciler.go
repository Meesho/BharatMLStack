// Package reconciler implements timer-based auto-promotion for OnyxDB versions.
//
// This is the first slice of the version rollout state machine (ADR-0009): a
// background loop that, on a fixed interval, scans every store opted into
// auto-promotion and promotes the newest READY version that has a warm pod on
// every shard — an older one if the newest is not yet covered. It owns only the
// READY → ACTIVE transition; the producer-driven
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

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/internal/placement"
	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/model"
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
	UpdateAssignment(ctx context.Context, tenant, store, vID string, assignment map[string][]string) (bool, error)
	RetireVersion(ctx context.Context, tenant, store, vID string) error
	PurgeVersion(ctx context.Context, tenant, store, vID string) error
}

// defaultKeepVersions is the retention window for auto-promote stores that don't
// set an explicit DataflowConfig.KeepVersions (active + one rollback).
const defaultKeepVersions = 2

// pausedLogInterval bounds how often the reconciler logs that a store with
// dataflow writesPaused has versions held back: every tick (5 s) would be
// ~17K lines a day per paused store.
const pausedLogInterval = 10 * time.Minute

// Reconciler periodically promotes ready, fully-warm versions.
type Reconciler struct {
	state    StateReader
	interval time.Duration
	// pausedLoggedAt is when each paused store ("tenant/store") was last
	// logged. Touched only by the reconcile goroutine.
	pausedLoggedAt map[string]time.Time
	cellLoggedAt   map[string]time.Time
}

// New creates a Reconciler that ticks every interval.
func New(state StateReader, interval time.Duration) *Reconciler {
	return &Reconciler{state: state, interval: interval, pausedLoggedAt: map[string]time.Time{}, cellLoggedAt: map[string]time.Time{}}
}

// Run blocks, reconciling once per interval, until ctx is cancelled.
func (r *Reconciler) Run(ctx context.Context) {
	log.Warn().Dur("interval", r.interval).Msg("auto-promote reconciler started")
	t := time.NewTicker(r.interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Warn().Msg("auto-promote reconciler stopping")
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

	pods, err := r.state.ListPods(ctx, ref.Tenant, ref.Store)
	if err != nil {
		lg.Error().Err(err).Msg("reconciler: ListPods failed")
		return
	}
	// Promote and the assignment refresh see only the store's cell; purge
	// below still sees every pod, since any pod holding a version blocks it.
	routable := placement.InCell(pods, df.Cell)
	r.logOutOfCell(lg, ref, df.Cell, pods)

	// Newest first, and an uncovered newer version does not hide a covered
	// older one: data loaders park a newer version while an older one is
	// resident and unpromoted (docs/design/onyxdb-version-memory-guard.md §4), so
	// waiting for the newest to be covered would never end.
	//
	// Writes paused (docs/design/onyxdb-pause-writes.md): promote nothing, but
	// keep the assignment refresh and GC below running — pods rescheduled or
	// added by the HPA during the pause must still enter the active version's
	// assignment.
	candidates := promotable(versions, st.ActiveVersion)
	if df.WritesPaused {
		r.logPaused(lg, ref, candidates)
		candidates = nil
	} else {
		delete(r.pausedLoggedAt, ref.Tenant+"/"+ref.Store)
	}
	for _, candidate := range candidates {
		assignment := placement.DeriveAssignment(st.Config.ShardCount, routable, candidate)
		if missing := uncoveredShards(st.Config.ShardCount, assignment); len(missing) > 0 {
			lg.Warn().
				Str("version", candidate).
				Int("warmShards", st.Config.ShardCount-len(missing)).
				Int("shardCount", st.Config.ShardCount).
				Msg("reconciler: coverage incomplete, deferring promote")
			continue
		}
		if err := r.state.PromoteVersion(ctx, ref.Tenant, ref.Store, candidate, assignment); err != nil {
			if err == etcdstate.ErrCASConflict {
				lg.Warn().Str("version", candidate).Msg("reconciler: promote raced (CAS), will retry next tick")
			} else {
				lg.Error().Err(err).Str("version", candidate).Msg("reconciler: promote failed")
			}
			return // re-evaluate (incl. GC) next tick once the active version is settled
		}
		lg.Warn().Str("version", candidate).Msg("reconciler: auto-promoted version")
		// Re-read state next tick so GC sees the new active/rollback.
		return
	}

	// Refresh the active version's assignment from current pod registrations.
	// Pods may be rescheduled (new IPs) without a version change — the etcd
	// VersionMeta.Assignment must stay in sync so SDK clients that read it
	// directly (without the topology API) route to the right pods.
	if st.ActiveVersion != "" {
		assignment := placement.DeriveAssignment(st.Config.ShardCount, routable, st.ActiveVersion)
		changed, err := r.state.UpdateAssignment(ctx, ref.Tenant, ref.Store, st.ActiveVersion, assignment)
		if err != nil {
			if err == etcdstate.ErrCASConflict {
				lg.Debug().Msg("reconciler: assignment refresh raced (CAS), will retry next tick")
			} else {
				lg.Warn().Err(err).Msg("reconciler: assignment refresh failed")
			}
		} else if changed {
			lg.Warn().Str("version", st.ActiveVersion).Msg("reconciler: refreshed stale assignment from pod registrations")
		}
	}

	// Keep-last-N GC: retire versions outside the retention window so pods free
	// their SSTs. Runs against the *current* active/rollback (re-read each tick).
	r.gcStore(ctx, ref, st, versions, effectiveKeep(df))

	// Purge: delete the keys of RETIRING versions that no pod holds any more.
	// It reads this tick's snapshot, so a version GC retired just now is
	// purged on a later tick, once the pods have dropped it.
	if df.PurgeRetired {
		r.purgeStore(ctx, ref, st, versions, pods, effectiveKeep(df))
	}
}

// logPaused logs, at most once per pausedLogInterval per store, that writes
// are paused and which READY versions are held back. held is newest first.
func (r *Reconciler) logPaused(lg zerolog.Logger, ref etcdstate.StoreRef, held []string) {
	k := ref.Tenant + "/" + ref.Store
	if last, ok := r.pausedLoggedAt[k]; ok && time.Since(last) < pausedLogInterval {
		return
	}
	r.pausedLoggedAt[k] = time.Now()
	ev := lg.Warn().Int("held", len(held))
	if len(held) > 0 {
		ev = ev.Str("newestHeld", held[0])
	}
	ev.Msg("reconciler: writes paused — not promoting")
}

// logOutOfCell reports, at most once per pausedLogInterval per store, the pods
// registered under the store for another cell, which routing leaves out
// (placement.InCell). Silent while there are none.
func (r *Reconciler) logOutOfCell(lg zerolog.Logger, ref etcdstate.StoreRef, cell string, pods map[string]model.PodData) {
	k := ref.Tenant + "/" + ref.Store
	out := placement.OutOfCell(pods, cell)
	if len(out) == 0 {
		delete(r.cellLoggedAt, k)
		return
	}
	if last, ok := r.cellLoggedAt[k]; ok && time.Since(last) < pausedLogInterval {
		return
	}
	r.cellLoggedAt[k] = time.Now()
	lg.Warn().Str("cell", cell).Strs("pods", out).
		Msg("reconciler: pods registered for another cell left out of routing")
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
		log.Warn().Str("tenant", ref.Tenant).Str("store", ref.Store).
			Str("version", vID).Msg("reconciler: retired version (keep-last-N GC)")
	}
}

// purgeStore deletes the etcd key of every purgeable version (versionsToPurge).
// A CAS conflict means another replica or a promote got there first; the next
// tick re-reads and decides again.
func (r *Reconciler) purgeStore(ctx context.Context, ref etcdstate.StoreRef, st *etcdstate.StoreState, versions map[string]*model.VersionMeta, pods map[string]model.PodData, keep int) {
	if keep <= 0 || st.ActiveVersion == "" {
		return // same gate as GC: never before the first promote
	}
	for _, vID := range versionsToPurge(versions, st.ActiveVersion, st.RollbackVersion, keep, pods) {
		if err := r.state.PurgeVersion(ctx, ref.Tenant, ref.Store, vID); err != nil {
			if err == etcdstate.ErrCASConflict {
				log.Debug().Str("tenant", ref.Tenant).Str("store", ref.Store).
					Str("version", vID).Msg("reconciler: purge raced (CAS), will retry next tick")
			} else {
				log.Error().Err(err).Str("tenant", ref.Tenant).Str("store", ref.Store).
					Str("version", vID).Msg("reconciler: purge failed")
			}
			continue
		}
		log.Warn().Str("tenant", ref.Tenant).Str("store", ref.Store).
			Str("version", vID).Msg("reconciler: purged retired version")
	}
}

// promotable returns every READY version ID strictly greater than the active
// version, newest first. Version IDs are {date}_{run}, which sort
// lexicographically, so plain string comparison gives chronological order.
func promotable(versions map[string]*model.VersionMeta, active string) []string {
	var out []string
	for vID, meta := range versions {
		if meta == nil || meta.Status != model.StatusReady {
			continue
		}
		if vID <= active {
			continue // already active, or older than active
		}
		out = append(out, vID)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(out)))
	return out
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
	ids, kept := keepSet(versions, active, rollback, keep)

	var retire []string
	for _, vID := range ids {
		if vID == "" {
			continue
		}
		if _, ok := kept[vID]; ok {
			continue
		}
		if m := versions[vID]; m != nil && m.Status == model.StatusRetiring {
			continue // already retiring
		}
		retire = append(retire, vID)
	}
	return retire
}

// keepSet returns every version ID newest first, and the set GC keeps: active,
// rollback, everything newer than active, and the newest keep at or below it.
func keepSet(versions map[string]*model.VersionMeta, active, rollback string, keep int) ([]string, map[string]struct{}) {
	ids := make([]string, 0, len(versions))
	for vID := range versions {
		ids = append(ids, vID)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ids))) // newest first

	kept := map[string]struct{}{active: {}, rollback: {}}
	belowOrEqual := 0
	for _, vID := range ids {
		if vID > active {
			kept[vID] = struct{}{} // in-flight future version
			continue
		}
		if belowOrEqual < keep {
			kept[vID] = struct{}{}
			belowOrEqual++
		}
	}
	return ids, kept
}

// versionsToPurge returns, oldest first, the RETIRING versions whose etcd key
// can be deleted: outside the keep set (so never active, rollback or a version
// newer than active) and listed by no pod registration as serving, loading,
// rolling out or warm. A pod that lists one has not finished dropping it.
func versionsToPurge(versions map[string]*model.VersionMeta, active, rollback string, keep int, pods map[string]model.PodData) []string {
	ids, kept := keepSet(versions, active, rollback, keep)
	held := heldVersions(pods)

	var purge []string
	for i := len(ids) - 1; i >= 0; i-- { // oldest first
		vID := ids[i]
		if vID == "" {
			continue
		}
		if m := versions[vID]; m == nil || m.Status != model.StatusRetiring {
			continue
		}
		if _, ok := kept[vID]; ok {
			continue
		}
		if _, ok := held[vID]; ok {
			continue
		}
		purge = append(purge, vID)
	}
	return purge
}

// heldVersions is every version some pod registration lists.
func heldVersions(pods map[string]model.PodData) map[string]struct{} {
	held := make(map[string]struct{})
	for _, pd := range pods {
		for _, v := range append([]string{pd.ServingVersion, pd.LoadingVersion, pd.RolloutVersion}, pd.WarmVersions...) {
			if v != "" {
				held[v] = struct{}{}
			}
		}
	}
	return held
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
