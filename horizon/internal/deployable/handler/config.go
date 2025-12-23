package handler

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
)

type Config interface {
	GetMetaData() (map[string][]string, error)
	CreateDeployable(request *DeployableRequest) error
	UpdateDeployable(request *DeployableRequest) error
	GetDeployablesByService(serviceName string) ([]servicedeployableconfig.ServiceDeployableConfig, error)
	RefreshDeployable(appName, serviceType string) (*servicedeployableconfig.ServiceDeployableConfig, error)
	GetRingMasterConfig(appName, workflowID, runID string) externalcall.Config
	TuneThresholds(request *TuneThresholdsRequest) error
}
