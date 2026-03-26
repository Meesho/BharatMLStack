package predator

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
)

var (
	DefaultModelPathKey  string
	PhoenixServerBaseUrl string
	GcsModelBucket       string
	GcsModelBasePath     string
	TestDeployableID     int
	TestGpuDeployableID  int
	initOnce             sync.Once
	IsMeeshoEnabled      bool
	AppEnv           string
	GcsConfigBucket   string
	GcsConfigBasePath string
)

func Init(config configs.Configs) {
	initOnce.Do(func() {
		DefaultModelPathKey = config.DefaultModelPath
		PhoenixServerBaseUrl = config.PhoenixServerBaseUrl
		GcsModelBucket = config.GcsModelBucket
		GcsModelBasePath = config.GcsModelBasePath
		TestDeployableID = config.TestDeployableID
		TestGpuDeployableID = config.TestGpuDeployableID
		IsMeeshoEnabled = config.IsMeeshoEnabled
		AppEnv = config.AppEnv
		GcsConfigBasePath = config.GcsConfigBasePath
		GcsConfigBucket = config.GcsConfigBucket
	})

}
