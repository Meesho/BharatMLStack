package model

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestStoreConfig_JSONRoundtrip(t *testing.T) {
	orig := StoreConfig{
		Tenant:     "fs",
		Store:      "features",
		EntityKey:  "catalog_id|geohash",
		ShardCount: 10,
	}
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got StoreConfig
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != orig {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", got, orig)
	}
}

func TestVersionMeta_JSONRoundtrip(t *testing.T) {
	orig := VersionMeta{
		Date:       "20260528",
		Run:        "001",
		ShardCount: 10,
		Status:     StatusActive,
		Assignment: map[string][]string{
			"0": {"10.0.1.10:9091", "10.0.1.11:9091"},
			"1": {"10.0.1.12:9091", "10.0.1.13:9091"},
		},
	}
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got VersionMeta
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Date != orig.Date || got.Run != orig.Run || got.Status != orig.Status || got.ShardCount != orig.ShardCount {
		t.Errorf("scalar fields mismatch: got %+v, want %+v", got, orig)
	}
	if !reflect.DeepEqual(got.Assignment, orig.Assignment) {
		t.Errorf("assignment mismatch: got %v, want %v", got.Assignment, orig.Assignment)
	}
}

func TestVersionStatus_Values(t *testing.T) {
	tests := []struct {
		status VersionStatus
		want   string
	}{
		{StatusAllocated, "ALLOCATED"},
		{StatusIngesting, "INGESTING"},
		{StatusReady, "READY"},
		{StatusActive, "ACTIVE"},
		{StatusRetiring, "RETIRING"},
	}
	for _, tc := range tests {
		if string(tc.status) != tc.want {
			t.Errorf("status = %q, want %q", tc.status, tc.want)
		}
	}
}

func TestVersionMeta_StatusRoundtrip(t *testing.T) {
	statuses := []VersionStatus{
		StatusAllocated, StatusIngesting, StatusReady, StatusActive, StatusRetiring,
	}
	for _, s := range statuses {
		vm := VersionMeta{Status: s}
		b, err := json.Marshal(vm)
		if err != nil {
			t.Fatalf("marshal status %q: %v", s, err)
		}
		var got VersionMeta
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal status %q: %v", s, err)
		}
		if got.Status != s {
			t.Errorf("status roundtrip: got %q, want %q", got.Status, s)
		}
	}
}

func TestPodData_JSONRoundtrip(t *testing.T) {
	t.Run("with_loading_version", func(t *testing.T) {
		orig := PodData{
			NodeIP:         "10.0.1.10",
			PodIP:          "10.0.1.10",
			ServingVersion: "20260528_001",
			WarmVersions:   []string{"20260528_001", "20260527_003"},
			LoadingVersion: "20260529_001",
		}
		b, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var got PodData
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got.NodeIP != orig.NodeIP || got.PodIP != orig.PodIP ||
			got.ServingVersion != orig.ServingVersion || got.LoadingVersion != orig.LoadingVersion {
			t.Errorf("scalar fields mismatch: got %+v, want %+v", got, orig)
		}
		if !reflect.DeepEqual(got.WarmVersions, orig.WarmVersions) {
			t.Errorf("WarmVersions mismatch: got %v, want %v", got.WarmVersions, orig.WarmVersions)
		}
	})

	t.Run("omits_empty_loading_version", func(t *testing.T) {
		orig := PodData{
			NodeIP:         "10.0.1.10",
			PodIP:          "10.0.1.10",
			ServingVersion: "20260528_001",
			WarmVersions:   []string{"20260528_001"},
		}
		b, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var raw map[string]interface{}
		if err := json.Unmarshal(b, &raw); err != nil {
			t.Fatalf("unmarshal to map: %v", err)
		}
		if _, ok := raw["loadingVersion"]; ok {
			t.Error("loadingVersion should be omitted from JSON when empty")
		}
	})
}

func TestRolloutConfig_WarmOmittedByDefault(t *testing.T) {
	b, err := json.Marshal(RolloutConfig{Steps: []int{10, 100}, StepIntervalSec: 60})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got, want := string(b), `{"steps":[10,100],"stepIntervalSec":60}`; got != want {
		t.Errorf("marshal = %s, want %s (warm must be omitted when unset)", got, want)
	}
	var rc RolloutConfig
	if err := json.Unmarshal([]byte(`{"steps":[10,100]}`), &rc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rc.Warm != nil {
		t.Errorf("warm = %+v, want nil for a config that predates it", rc.Warm)
	}
}

func TestRolloutConfig_WarmRoundtrip(t *testing.T) {
	in := `{"steps":[1,2,3,5,100],"stepIntervalSec":45,"warm":{"enabled":true,"lookaheadPct":5,` +
		`"maxKeysPerSec":15000,"targetHitRatio":0.95,"maxDwellSec":180}}`
	var rc RolloutConfig
	if err := json.Unmarshal([]byte(in), &rc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := WarmConfig{Enabled: true, LookaheadPct: 5, MaxKeysPerSec: 15000, TargetHitRatio: 0.95, MaxDwellSec: 180}
	if rc.Warm == nil || *rc.Warm != want {
		t.Fatalf("warm = %+v, want %+v", rc.Warm, want)
	}
	b, err := json.Marshal(rc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back RolloutConfig
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal back: %v", err)
	}
	if !reflect.DeepEqual(back, rc) {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", back, rc)
	}
}

func TestWarmConfig_DisabledByDefault(t *testing.T) {
	var w WarmConfig
	if err := json.Unmarshal([]byte(`{"lookaheadPct":5}`), &w); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if w.Enabled {
		t.Error("enabled must default to false")
	}
	b, _ := json.Marshal(WarmConfig{})
	if string(b) != `{"enabled":false}` {
		t.Errorf("zero WarmConfig marshals to %s", b)
	}
}

func TestDataflowConfig_LoadCfg(t *testing.T) {
	var df DataflowConfig
	if err := json.Unmarshal([]byte(`{"rolloutCfg":{"steps":[100]}}`), &df); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if df.LoadCfg != nil {
		t.Errorf("loadCfg = %+v, want nil for a config that predates it", df.LoadCfg)
	}

	for _, tc := range []struct {
		in   string
		want *bool
	}{
		{`{"loadCfg":{}}`, nil},
		{`{"loadCfg":{"preloadIndexFilter":true}}`, boolPtr(true)},
		{`{"loadCfg":{"preloadIndexFilter":false}}`, boolPtr(false)},
	} {
		var got DataflowConfig
		if err := json.Unmarshal([]byte(tc.in), &got); err != nil {
			t.Fatalf("%s: unmarshal: %v", tc.in, err)
		}
		if got.LoadCfg == nil {
			t.Fatalf("%s: loadCfg nil", tc.in)
		}
		if !reflect.DeepEqual(got.LoadCfg.PreloadIndexFilter, tc.want) {
			t.Errorf("%s: preloadIndexFilter = %v, want %v", tc.in, got.LoadCfg.PreloadIndexFilter, tc.want)
		}
		b, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("%s: marshal: %v", tc.in, err)
		}
		var back DataflowConfig
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("%s: unmarshal back: %v", tc.in, err)
		}
		if !reflect.DeepEqual(back.LoadCfg, got.LoadCfg) {
			t.Errorf("%s: roundtrip = %+v, want %+v", tc.in, back.LoadCfg, got.LoadCfg)
		}
	}
}

func TestWarmConfig_PauseOverrides(t *testing.T) {
	var w WarmConfig
	if err := json.Unmarshal([]byte(`{"enabled":true,"pauseEngineConcurrency":48,"pauseP99Ms":40}`), &w); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if w.PauseEngineConcurrency != 48 || w.PauseP99Ms != 40 {
		t.Errorf("pause overrides = %v / %v, want 48 / 40", w.PauseEngineConcurrency, w.PauseP99Ms)
	}
	b, _ := json.Marshal(WarmConfig{Enabled: true})
	if string(b) != `{"enabled":true}` {
		t.Errorf("unset pause overrides must be omitted, got %s", b)
	}
}

func boolPtr(b bool) *bool { return &b }

func TestPodData_SlowStartRoundtrip(t *testing.T) {
	orig := PodData{PodIP: "10.0.0.7", WarmVersions: []string{"v1"}, ServingSince: 1791281000123, SlowStartSec: 300}
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got PodData
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, orig) {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", got, orig)
	}
}

func TestPodData_RegistrationWithoutSlowStart(t *testing.T) {
	// A registration written by a loader that predates slow start: no fields,
	// so clients must read "no ramp".
	var pd PodData
	if err := json.Unmarshal([]byte(`{"nodeIP":"n","podIP":"10.0.0.7","servingVersion":"v1","warmVersions":["v1"]}`), &pd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if pd.ServingSince != 0 || pd.SlowStartSec != 0 {
		t.Errorf("servingSince=%d slowStartSec=%d, want 0/0", pd.ServingSince, pd.SlowStartSec)
	}
	b, _ := json.Marshal(PodData{PodIP: "10.0.0.7"})
	for _, k := range []string{"servingSince", "slowStartSec"} {
		if strings.Contains(string(b), k) {
			t.Errorf("unset %s must be omitted, got %s", k, b)
		}
	}
}

func TestPodData_Addr(t *testing.T) {
	cases := []struct {
		name string
		pd   PodData
		want string
	}{
		{"port unset falls back to the default", PodData{PodIP: "10.0.0.7"}, "10.0.0.7:9091"},
		{"explicit port", PodData{PodIP: "10.0.0.7", Port: 9300}, "10.0.0.7:9300"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.pd.Addr(); got != tc.want {
				t.Errorf("Addr() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDataflowConfig_SlowStartCfg(t *testing.T) {
	var df DataflowConfig
	if err := json.Unmarshal([]byte(`{"slowStartCfg":{"windowSec":300}}`), &df); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if df.SlowStartCfg == nil || df.SlowStartCfg.WindowSec != 300 {
		t.Fatalf("slowStartCfg = %+v, want windowSec 300", df.SlowStartCfg)
	}
	var old DataflowConfig
	if err := json.Unmarshal([]byte(`{"rolloutCfg":{"steps":[100]}}`), &old); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if old.SlowStartCfg != nil {
		t.Errorf("slowStartCfg = %+v, want nil for a config that predates it", old.SlowStartCfg)
	}
	b, _ := json.Marshal(DataflowConfig{})
	if strings.Contains(string(b), "slowStartCfg") {
		t.Errorf("nil slowStartCfg must be omitted, got %s", b)
	}
}

// The SDK decodes these keys with a struct of its own (it builds against the
// released model, which predates the fields), so the JSON names are a wire
// contract between the dataloader and the SDK, not just field tags.
func TestPodData_SlowStartWireKeys(t *testing.T) {
	b, err := json.Marshal(PodData{PodIP: "10.0.0.7", Port: 9300, ServingSince: 1791281000123, SlowStartSec: 300})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{`"podIP":"10.0.0.7"`, `"port":9300`, `"servingSince":1791281000123`, `"slowStartSec":300`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("registration %s lacks %s", b, want)
		}
	}
}
