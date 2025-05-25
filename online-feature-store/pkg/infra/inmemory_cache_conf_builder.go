package infra

import (
	"fmt"
	"github.com/spf13/viper"
)

const (
	inMemoryCachePrefix = "IN_MEM_CACHE_"
	enabledSuffix       = "_ENABLED"
	sizeInBytesSuffix   = "_SIZE_IN_BYTES"
	nameSuffix          = "_NAME"
)

type IMCacheConf struct {
	Enabled     bool
	SizeInBytes int
	Name        string
}

func BuildInMemoryCacheConfFromEnv(envPrefix string) (*IMCacheConf, error) {
	if !viper.IsSet(envPrefix+enabledSuffix) || !viper.IsSet(envPrefix+nameSuffix) || !viper.IsSet(envPrefix+sizeInBytesSuffix) {
		return nil, fmt.Errorf("failed to load in-memory cache. invalid in-memory configs, prefix: %s", envPrefix)
	}
	size := viper.GetInt(envPrefix + sizeInBytesSuffix)
	name := viper.GetString(envPrefix + nameSuffix)
	return &IMCacheConf{
		Enabled:     viper.GetBool(envPrefix + enabledSuffix),
		SizeInBytes: size,
		Name:        name,
	}, nil

}
