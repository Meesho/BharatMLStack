package handler

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
)

type Config interface {
	GetMetaData() (map[string][]string, error)
	CreateDeployable(request *DeployableRequest, workingEnv string) (string, error)
	CreateDeployableMultiEnvironment(request *DeployableRequest) (map[string]string, error)
	UpdateDeployable(request *DeployableRequest, workingEnv string) error
	GetDeployablesByService(serviceName string) ([]servicedeployableconfig.ServiceDeployableConfig, error)
	RefreshDeployable(appName, serviceType, workingEnv string) (*servicedeployableconfig.ServiceDeployableConfig, error)
	GetRingMasterConfig(appName, workflowID, runID string) externalcall.Config
	TuneThresholds(request *TuneThresholdsRequest, workingEnv string) (string, error)
}
