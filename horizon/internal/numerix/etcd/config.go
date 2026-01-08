package etcd

type Manager interface {
	CreateConfig(configId string, expression string) error
	DeleteConfig(configId string) error
	UpdateConfig(configId string, expression string) error
}
