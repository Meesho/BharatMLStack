package inferflow

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
)

var (
	InferflowAppName string
	AppEnv           string
	HorizonAppName   string
	initOnce         sync.Once
)

func Init(config configs.Configs) {
	initOnce.Do(func() {
		InferflowAppName = config.InferflowAppName
		AppEnv = config.AppEnv
		HorizonAppName = config.HorizonAppName
	})
}
