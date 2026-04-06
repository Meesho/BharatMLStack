package etcd

import mapset "github.com/deckarep/golang-set/v2"

type Manager interface {
	GetComponentData(componentName string) *ComponentData
	CreateConfig(serviceName string, ConfigId string, InferflowConfig InferflowConfig) error
	UpdateConfig(serviceName string, ConfigId string, InferflowConfig InferflowConfig) error
	DeleteConfig(serviceName string, ConfigId string) error
	GetConfiguredEndpoints(serviceDeployableName string) mapset.Set[string]
}
