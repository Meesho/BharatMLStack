package schema

import (
	"encoding/json"
	"reflect"
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
