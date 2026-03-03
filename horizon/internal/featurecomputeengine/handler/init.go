package handler

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetdeps"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetregistry"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/assetstate"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/datasetpartitions"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/materializations"
)

// Handler orchestrates the core logic of the feature computation platform:
// DAG building, necessity resolution, compute key caching, trigger evaluation,
// execution planning, and plan-stage impact analysis.
type Handler struct {
	config            Config
	assetRegistry     assetregistry.Repository
	assetDeps         assetdeps.Repository
	assetState        assetstate.Repository
	materializations  materializations.Repository
	datasetPartitions datasetpartitions.Repository
}

// New creates a Handler with all repository dependencies injected.
func New(
	cfg Config,
	ar assetregistry.Repository,
	ad assetdeps.Repository,
	as assetstate.Repository,
	mat materializations.Repository,
	dp datasetpartitions.Repository,
) *Handler {
	return &Handler{
		config:            cfg,
		assetRegistry:     ar,
		assetDeps:         ad,
		assetState:        as,
		materializations:  mat,
		datasetPartitions: dp,
	}
}
