// Package handlers contains the Gin HTTP handlers for the OnyxDB control plane.
package handlers

import (
	"github.com/Meesho/BharatMLStack/onyxdb/controlplane/internal/etcdstate"
)

// Handlers groups all HTTP handler methods and their dependencies.
type Handlers struct {
	state etcdstate.StateClient
}

// New returns a Handlers backed by the given StateClient.
func New(state etcdstate.StateClient) *Handlers {
	return &Handlers{state: state}
}
