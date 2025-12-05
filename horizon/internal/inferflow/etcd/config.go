package etcd

type Manager interface {
	GetComponentData(componentName string) *ComponentData
	CreateConfig(serviceName string, ConfigId string, InferflowConfig InferflowConfig) error
	UpdateConfig(serviceName string, ConfigId string, InferflowConfig InferflowConfig) error
	DeleteConfig(serviceName string, ConfigId string) error
}
