package handler

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/rs/zerolog/log"
)

func NewDeployable(version int, appConfig configs.Configs) (Config, error) {
	switch version {
	case 1:
		return InitV1ConfigHandler(appConfig), nil
	default:
		log.Error().Msg("Invalid version for deployable handler")
		return nil, fmt.Errorf("invalid version: %d", version)
	}
}
