package interactionstore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/keepalive"
)

func TestKeepaliveParamsFromConfig(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		wantOK     bool
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, ok := keepaliveParamsFromConfig(tt.config)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantParams, params)
		})
	}
}

func TestDialOptions_KeepaliveAppendedOnlyWhenConfigured(t *testing.T) {
	// Backward compatibility: without keepalive config the dial-option set is unchanged
	// (transport credentials + service config). Enabling keepalive appends exactly one.
	plaintextNoKeepalive := dialOptions(Config{PlainText: true})
	tlsNoKeepalive := dialOptions(Config{PlainText: false})
	withKeepalive := dialOptions(Config{
		PlainText:          true,
		KeepaliveTimeMs:    20000,
		KeepaliveTimeoutMs: 5000,
	})

	assert.Len(t, plaintextNoKeepalive, 2, "plaintext dial should be transport-credentials + service-config only")
	assert.Len(t, tlsNoKeepalive, 2, "TLS dial should be transport-credentials + service-config only")
	assert.Len(t, withKeepalive, 3, "enabling keepalive should append exactly one dial option")
}
