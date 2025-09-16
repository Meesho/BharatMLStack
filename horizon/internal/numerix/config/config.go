package etcd

type Manager interface {
	CreateConfig(configId string, expression string) error
	UpdateConfig(configId string, expression string) error
}
