package handler

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
)

var (
	predatorOnce sync.Once
	predator     Config
)

func NewPredator(version int) (Config, error) {
	switch version {
	case 1:
		return InitV1ConfigHandler()
	default:
		log.Error().Msg("Invalid version for predator handler")
		return nil, fmt.Errorf("invalid version: %d", version)
	}
}
