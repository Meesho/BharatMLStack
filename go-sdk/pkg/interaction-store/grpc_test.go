package interactionstore

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/keepalive"
)

func TestKeepaliveParamsFromConfig(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		wantOK     bool
		wantErr    bool
		wantParams keepalive.ClientParameters
	}{
		{
			name:   "disabled by default (zero value config)",
			config: Config{},
			wantOK: false,
		},
		{
			name:   "disabled when time is zero even if other keepalive fields are set",
			config: Config{KeepaliveTimeoutMs: 5000, KeepalivePermitWithoutStream: true},
			wantOK: false,
		},
		{
			name:   "disabled when time is negative",
			config: Config{KeepaliveTimeMs: -1, KeepaliveTimeoutMs: 5000},
			wantOK: false,
		},
		{
			name:   "enabled with full params, milliseconds converted to Duration",
			config: Config{KeepaliveTimeMs: 20000, KeepaliveTimeoutMs: 5000, KeepalivePermitWithoutStream: true},
			wantOK: true,
			wantParams: keepalive.ClientParameters{
				Time:                20 * time.Second,
				Timeout:             5 * time.Second,
				PermitWithoutStream: true,
			},
		},
		{
			name:   "enabled with permit-without-stream defaulting to false",
			config: Config{KeepaliveTimeMs: 10000, KeepaliveTimeoutMs: 3000},
			wantOK: true,
			wantParams: keepalive.ClientParameters{
				Time:                10 * time.Second,
				Timeout:             3 * time.Second,
				PermitWithoutStream: false,
			},
		},
		{
			name:    "enabled with negative timeout is rejected",
			config:  Config{KeepaliveTimeMs: 20000, KeepaliveTimeoutMs: -1},
			wantOK:  false,
			wantErr: true,
		},
		{
			name:    "enabled with zero timeout is rejected",
			config:  Config{KeepaliveTimeMs: 20000, KeepaliveTimeoutMs: 0},
			wantOK:  false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, ok, err := keepaliveParamsFromConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantParams, params)
		})
	}
}

// TestDefaultServiceConfigIsRoundRobin verifies the default dial behaviour directly:
// the service config must keep the round_robin load-balancing policy. Asserting the
// effective policy (not just an option count) means the test fails if round_robin is
// removed, renamed, or replaced.
func TestDefaultServiceConfigIsRoundRobin(t *testing.T) {
	var sc struct {
		LoadBalancingPolicy string `json:"loadBalancingPolicy"`
	}
	require.NoError(t, json.Unmarshal([]byte(roundRobinServiceConfig), &sc))
	assert.Equal(t, "round_robin", sc.LoadBalancingPolicy)
}

func TestDialOptions(t *testing.T) {
	t.Run("default path applies no keepalive and keeps the round-robin config", func(t *testing.T) {
		// The PR promises the default dial behaviour is unchanged. Assert it directly:
		// keepalive is off for a default config (no keepalive dial option is produced).
		_, enabled, err := keepaliveParamsFromConfig(Config{PlainText: true})
		require.NoError(t, err)
		assert.False(t, enabled, "default config must not enable keepalive")

		opts, err := dialOptions(Config{PlainText: true})
		require.NoError(t, err)
		assert.Len(t, opts, 2, "default plaintext dial = transport-credentials + service-config only")

		tlsOpts, err := dialOptions(Config{PlainText: false})
		require.NoError(t, err)
		assert.Len(t, tlsOpts, 2, "default TLS dial = transport-credentials + service-config only")
	})

	t.Run("valid keepalive appends exactly one dial option", func(t *testing.T) {
		opts, err := dialOptions(Config{PlainText: true, KeepaliveTimeMs: 20000, KeepaliveTimeoutMs: 5000})
		require.NoError(t, err)
		assert.Len(t, opts, 3, "enabling keepalive should append exactly one dial option")
	})

	t.Run("invalid keepalive timeout is rejected (no options returned)", func(t *testing.T) {
		opts, err := dialOptions(Config{PlainText: true, KeepaliveTimeMs: 20000, KeepaliveTimeoutMs: -1})
		assert.Error(t, err)
		assert.Nil(t, opts)
	})
}
