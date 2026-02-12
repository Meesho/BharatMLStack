package components

import (
	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/matrix"
)

type ComponentRequest struct {
	ComponentData   *matrix.ComponentMatrix
	SlateData       *matrix.ComponentMatrix // Slate-level matrix (one row per slate); nil when no slate components
	Entities        *[]string
	EntityIds       *[][]string
	ComponentConfig *config.ComponentConfig
	Features        *map[string][]string
	ModelId         string
	ModelIdHash     string // Pre-computed CRC32 hash of ModelId for cache key construction
	Headers         map[string]string
}
