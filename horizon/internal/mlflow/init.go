package mlflow

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
)

var (
	initMLFlowOnce sync.Once
	MLFlowHostURL  string
)

func Init(config configs.Configs) {
	initMLFlowOnce.Do(func() {
		MLFlowHostURL = config.MLFlowHostURL
	})
}
