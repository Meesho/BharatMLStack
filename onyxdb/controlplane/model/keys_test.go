package model

import "testing"

func TestStorePrefix(t *testing.T) {
	tests := []struct {
		tenant, store, want string
	}{
		{"fs", "features", "/config/mnemo/tenants/fs/stores/features"},
		{"tenant-2", "store_name", "/config/mnemo/tenants/tenant-2/stores/store_name"},
		{"a", "b", "/config/mnemo/tenants/a/stores/b"},
	}
	for _, tc := range tests {
		if got := StorePrefix(tc.tenant, tc.store); got != tc.want {
			t.Errorf("StorePrefix(%q, %q) = %q, want %q", tc.tenant, tc.store, got, tc.want)
		}
	}
}

func TestEntityKeyPath(t *testing.T) {
	got := EntityKeyPath("fs", "features")
	want := "/config/mnemo/tenants/fs/stores/features/entityKey"
	if got != want {
		t.Errorf("EntityKeyPath = %q, want %q", got, want)
	}
}

func TestShardCountPath(t *testing.T) {
	got := ShardCountPath("fs", "features")
	want := "/config/mnemo/tenants/fs/stores/features/shardCount"
	if got != want {
		t.Errorf("ShardCountPath = %q, want %q", got, want)
	}
}

func TestActiveVersionPath(t *testing.T) {
	got := ActiveVersionPath("fs", "features")
	want := "/config/mnemo/tenants/fs/stores/features/activeVersion"
	if got != want {
		t.Errorf("ActiveVersionPath = %q, want %q", got, want)
	}
}

func TestRollbackVersionPath(t *testing.T) {
	got := RollbackVersionPath("fs", "features")
	want := "/config/mnemo/tenants/fs/stores/features/rollbackVersion"
	if got != want {
		t.Errorf("RollbackVersionPath = %q, want %q", got, want)
	}
}

func TestTopologyVersionPath(t *testing.T) {
	got := TopologyVersionPath("fs", "features")
	want := "/config/mnemo/tenants/fs/stores/features/topologyVersion"
	if got != want {
		t.Errorf("TopologyVersionPath = %q, want %q", got, want)
	}
}

func TestVersionPrefix(t *testing.T) {
	tests := []struct {
		tenant, store, versionID, want string
	}{
		{
			"fs", "features", "20260528_001",
			"/config/mnemo/tenants/fs/stores/features/versions/20260528_001",
		},
		{
			"recsys", "catalog", "20260601_003",
			"/config/mnemo/tenants/recsys/stores/catalog/versions/20260601_003",
		},
	}
	for _, tc := range tests {
		if got := VersionPrefix(tc.tenant, tc.store, tc.versionID); got != tc.want {
			t.Errorf("VersionPrefix(%q, %q, %q) = %q, want %q", tc.tenant, tc.store, tc.versionID, got, tc.want)
		}
	}
}

func TestVersionsWatchPrefix(t *testing.T) {
	got := VersionsWatchPrefix("fs", "features")
	want := "/config/mnemo/tenants/fs/stores/features/versions/"
	if got != want {
		t.Errorf("VersionsWatchPrefix = %q, want %q", got, want)
	}
}

func TestPodDataPath(t *testing.T) {
	tests := []struct {
		tenant, store, podID, want string
	}{
		{
			"fs", "features", "pod-abc-0",
			"/config/mnemo-cluster-manager/fs/features/pod-abc-0",
		},
		{
			"recsys", "catalog", "fs-features-shard-0-1",
			"/config/mnemo-cluster-manager/recsys/catalog/fs-features-shard-0-1",
		},
	}
	for _, tc := range tests {
		if got := PodDataPath(tc.tenant, tc.store, tc.podID); got != tc.want {
			t.Errorf("PodDataPath(%q, %q, %q) = %q, want %q", tc.tenant, tc.store, tc.podID, got, tc.want)
		}
	}
}

func TestPodWatchPrefix(t *testing.T) {
	got := PodWatchPrefix("fs", "features")
	want := "/config/mnemo-cluster-manager/fs/features/"
	if got != want {
		t.Errorf("PodWatchPrefix = %q, want %q", got, want)
	}
}
