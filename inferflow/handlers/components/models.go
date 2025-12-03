package components

import (
	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/matrix"
)

type ComponentRequest struct {
	ComponentData   *matrix.ComponentMatrix
	Entities        *[]string
	EntityIds       *[][]string
	ComponentConfig *config.ComponentConfig
	Features        *map[string][]string
	ModelId         string
	Headers         map[string]string
}
