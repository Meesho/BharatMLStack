package internal

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	deployableHandler "github.com/Meesho/BharatMLStack/horizon/internal/deployable/handler"
	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	inferflow "github.com/Meesho/BharatMLStack/horizon/internal/inferflow"
	"github.com/Meesho/BharatMLStack/horizon/internal/numerix"
	onlinefeaturestore "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store"
	"github.com/Meesho/BharatMLStack/horizon/internal/predator"
)

func InitAll(config configs.Configs) {

	externalcall.Init(config)
	numerix.Init(config)
	predator.Init(config)
	deployableHandler.Init(config)
	inferflow.Init(config)
	onlinefeaturestore.Init(config)
}
