package handler

import (
	"fmt"
	"github.com/rs/zerolog/log"
)

func NewDeployable(version int) (Config, error) {
	switch version {
	case 1:
		return InitV1ConfigHandler(), nil
	default:
		log.Error().Msg("Invalid version for deployable handler")
		return nil, fmt.Errorf("invalid version: %d", version)
	}
}
