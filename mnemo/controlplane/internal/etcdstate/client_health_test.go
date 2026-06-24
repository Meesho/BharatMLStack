package etcdstate

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHealth(t *testing.T) {
	// Reachable etcd: a "not found" probe read is still healthy.
	require.NoError(t, newTestStateClient(newMemKVOps()).Health(context.Background()))

	// Unreachable etcd: the underlying get errors → Health surfaces it.
	err := newTestStateClient(&errKVOps{err: errors.New("dial timeout")}).Health(context.Background())
	require.Error(t, err)
}
