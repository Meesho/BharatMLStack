package infra

import (
	"fmt"

	"github.com/spf13/viper"
)

const (
	p2PCachePrefix                        = "P2P_CACHE_"
	p2PCacheEnabledSuffix                 = "_ENABLED"
	p2PCacheOwnPartitionSizeInBytesSuffix = "_OWN_PARTITION_SIZE_IN_BYTES"
	p2PCacheGlobalSizeInBytesSuffix       = "_GLOBAL_SIZE_IN_BYTES"
	p2PCacheNameSuffix                    = "_NAME"
)

type P2PCacheConf struct {
	Enabled                 bool
	OwnPartitionSizeInBytes int
	GlobalSizeInBytes       int
	Name                    string
}

func BuildP2PCacheConfFromEnv(envPrefix string) (*P2PCacheConf, error) {
	if !viper.IsSet(envPrefix+p2PCacheEnabledSuffix) ||
		!viper.IsSet(envPrefix+p2PCacheOwnPartitionSizeInBytesSuffix) ||
		!viper.IsSet(envPrefix+p2PCacheGlobalSizeInBytesSuffix) ||
		!viper.IsSet(envPrefix+p2PCacheNameSuffix) {
		return nil, fmt.Errorf("failed to load distributed in-memory cache. invalid in-memory configs, keys: %s %s %s %s", envPrefix+p2PCacheEnabledSuffix, envPrefix+p2PCacheOwnPartitionSizeInBytesSuffix, envPrefix+p2PCacheGlobalSizeInBytesSuffix, envPrefix+p2PCacheNameSuffix)
	}
	enabled := viper.GetBool(envPrefix + p2PCacheEnabledSuffix)
	ownPartitionSizeInBytes := viper.GetInt(envPrefix + p2PCacheOwnPartitionSizeInBytesSuffix)
	globalSizeInBytes := viper.GetInt(envPrefix + p2PCacheGlobalSizeInBytesSuffix)
	name := viper.GetString(envPrefix + p2PCacheNameSuffix)
	return &P2PCacheConf{
		Enabled:                 enabled,
		OwnPartitionSizeInBytes: ownPartitionSizeInBytes,
		GlobalSizeInBytes:       globalSizeInBytes,
		Name:                    name,
	}, nil
}
