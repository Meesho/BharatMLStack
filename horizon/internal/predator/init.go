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
	IsGcsEnabled         bool
	IsDummyModelEnabled  bool
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
		IsGcsEnabled = config.GcsEnabled
		IsDummyModelEnabled = config.IsDummyModelEnabled
	})

}
