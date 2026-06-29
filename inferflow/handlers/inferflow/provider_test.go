//go:build !meesho

package inferflow

import (
	"testing"

	"github.com/emirpasic/gods/maps/linkedhashmap"

	"github.com/Meesho/BharatMLStack/inferflow/dag-topology-executor/handlers/dag"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
)

// newConfigWithComponents builds a minimal *config.ModelConfig containing one
// model with the given feature / predator / numerix component names.
func newConfigWithComponents(featureNames, predatorNames, numerixNames []string) *config.ModelConfig {
	fMap := linkedhashmap.New()
	for _, n := range featureNames {
		fMap.Put(n, struct{}{})
	}
	pMap := linkedhashmap.New()
	for _, n := range predatorNames {
		pMap.Put(n, struct{}{})
	}
	nMap := linkedhashmap.New()
	for _, n := range numerixNames {
		nMap.Put(n, struct{}{})
	}
	return &config.ModelConfig{
		ConfigMap: map[string]config.Config{
			"model-a": {
				DAGExecutionConfig: config.DAGExecutionConfig{
					ComponentDependency: map[string][]string{"a": {"b"}},
				},
				ComponentConfig: config.ComponentConfig{
					FeatureComponentConfig:  *fMap,
					PredatorComponentConfig: *pMap,
					NumerixComponentConfig:  *nMap,
				},
			},
		},
	}
}

func newHandler() *ComponentProviderHandler {
	return &ComponentProviderHandler{
		componentMap: make(map[string]dag.AbstractComponent),
	}
}

// TestRegisterComponent_RebuildsNotAccumulates is the main behavioural
// guarantee of the rebuild-and-swap fix: when the same handler is registered
// twice with different configs, the resulting map MUST reflect ONLY the
// second config — not the union of both.
//
// Prior to the fix, RegisterComponent appended to the existing map without
// pruning. Running it on every etcd config-change callback caused the map
// to accumulate entries from every model that had ever been seen, even
// after they were removed from the live config — an unbounded leak.
func TestRegisterComponent_RebuildsNotAccumulates(t *testing.T) {
	cp := newHandler()

	// First call: 2 feature + 1 predator components.
	cp.RegisterComponent(newConfigWithComponents(
		[]string{"feat_v1_a", "feat_v1_b"},
		[]string{"pred_v1"},
		nil,
	))

	// Second call: a DIFFERENT set of components (simulates a config-change
	// event that renames/removes the v1 components and ships v2).
	cp.RegisterComponent(newConfigWithComponents(
		[]string{"feat_v2_x"},
		[]string{"pred_v2"},
		[]string{"num_v2"},
	))

	// The map MUST now reflect only the v2 components (+ the always-present
	// feature_initializer), NOT the union of v1 and v2.
	expected := map[string]struct{}{
		"feature_initializer": {}, // always present
		"feat_v2_x":           {},
		"pred_v2":             {},
		"num_v2":              {},
	}
	cp.mapMutex.RLock()
	got := make(map[string]struct{}, len(cp.componentMap))
	for k := range cp.componentMap {
		got[k] = struct{}{}
	}
	cp.mapMutex.RUnlock()

	if len(got) != len(expected) {
		t.Fatalf("map size %d, want %d. Got keys: %v", len(got), len(expected), keys(got))
	}
	for k := range expected {
		if _, ok := got[k]; !ok {
			t.Errorf("expected key %q in map, missing", k)
		}
	}
	for k := range got {
		if _, ok := expected[k]; !ok {
			t.Errorf("unexpected key %q in map (would have been leaked from prior call)", k)
		}
	}
}

// TestRegisterComponent_StableSizeAcrossRepeatedSameCall verifies that calling
// RegisterComponent N times with the same config produces a map of constant
// size — i.e. no accumulation across identical events. This is the simplest
// regression test for the leak.
func TestRegisterComponent_StableSizeAcrossRepeatedSameCall(t *testing.T) {
	cp := newHandler()
	cfg := newConfigWithComponents(
		[]string{"feat_a", "feat_b", "feat_c"},
		[]string{"pred_x"},
		[]string{"num_q"},
	)

	const calls = 100
	for i := 0; i < calls; i++ {
		cp.RegisterComponent(cfg)
	}

	// 3 feature + 1 predator + 1 numerix + 1 always-present init = 6 entries,
	// no matter how many times we register the same config.
	cp.mapMutex.RLock()
	size := len(cp.componentMap)
	cp.mapMutex.RUnlock()
	if size != 6 {
		t.Fatalf("after %d identical RegisterComponent calls, map size = %d, want 6", calls, size)
	}
}

// TestRegisterComponent_NonModelConfigInputIsNoOp verifies the early-return
// path when the request isn't a *config.ModelConfig — the map should remain
// untouched (in particular, the feature_initializer entry should not be
// re-created when the input is invalid).
func TestRegisterComponent_NonModelConfigInputIsNoOp(t *testing.T) {
	cp := newHandler()
	cp.RegisterComponent("not a model config")

	cp.mapMutex.RLock()
	defer cp.mapMutex.RUnlock()
	if len(cp.componentMap) != 0 {
		t.Errorf("expected map untouched on invalid input; got %d entries", len(cp.componentMap))
	}
}

// TestRegisterComponent_NilModelConfigStillWritesInitializer covers the edge
// case where a valid *ModelConfig pointer carries an empty ConfigMap — the
// feature_initializer should still land in the rebuilt map.
func TestRegisterComponent_EmptyModelConfigKeepsInitializer(t *testing.T) {
	cp := newHandler()
	cp.RegisterComponent(&config.ModelConfig{ConfigMap: nil})

	cp.mapMutex.RLock()
	defer cp.mapMutex.RUnlock()
	if _, ok := cp.componentMap[featureInitComponent]; !ok {
		t.Error("feature_initializer entry missing after RegisterComponent with empty ConfigMap")
	}
	if len(cp.componentMap) != 1 {
		t.Errorf("expected only feature_initializer in map; got %d entries", len(cp.componentMap))
	}
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
