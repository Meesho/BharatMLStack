package zookeeper

const (
	IN_MEMORY_CACHE_CONFIG_ZK_PATH = "/model-config/storage/in-memory-cache"
	MODEL_CONFIG_ZK_PATH           = "/model-config/config-map"
)

type ZKConfig struct {
	Server string `koanf:"server"`
}
