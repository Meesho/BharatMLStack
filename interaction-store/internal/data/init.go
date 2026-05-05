package data

import (
	"github.com/Meesho/BharatMLStack/interaction-store/internal/config"
	"github.com/Meesho/BharatMLStack/interaction-store/internal/data/scylla"
)

func Init(config config.Configs) {
	scylla.Init(config)
}
