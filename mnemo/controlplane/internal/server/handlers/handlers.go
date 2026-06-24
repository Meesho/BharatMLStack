// Package handlers contains the Gin HTTP handlers for the mNemo control plane.
package handlers

import (
	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/etcdstate"
)

// Handlers groups all HTTP handler methods and their dependencies.
type Handlers struct {
	state etcdstate.StateClient
}

// New returns a Handlers backed by the given StateClient.
func New(state etcdstate.StateClient) *Handlers {
	return &Handlers{state: state}
}
