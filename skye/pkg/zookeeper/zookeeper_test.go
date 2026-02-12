package zookeeper

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestV1_deserialize_2(t *testing.T) {
	type AlgoConfigs struct {
		Enabled       bool
		Common        map[string]string
		LoggedInUser  map[string]string
		AnonymousUser map[string]string
	}
	type TenantsConfigs struct {
		Ads     map[string]map[string]AlgoConfigs
		Exploit map[string]map[string]AlgoConfigs
		Explore map[string]map[string]AlgoConfigs
	}
	type ZKConfigs struct {
		FY                    TenantsConfigs
		Clp                   TenantsConfigs
		Collection            TenantsConfigs
		CatalogRecommendation TenantsConfigs
	}

	dataMap := map[string]string{
		"/config/testservice/fy/ads/prodsim/1": `{"enabled": true, "common": {"a": "astring", "b": "bstring"}, "loggedinuser": {"a": "astring", "b": "bstring"}, "anonymoususer": {"a": "astring", "b": "bstring"}}`,
		"/config/testservice/fy/ads/prodsim/2": `{"enabled": false, "common": {"a": "ASTRING", "b": "bstring"}, "loggedinuser": {"a": "astring", "b": "bstring"}, "anonymoususer": {"a": "astring", "b": "bstring"}}`,
	}
	metaMap := map[string]string{
		"/config/testservice/fy/ads/prodsim/1": `/config/test-service/fy/ads/prod-sim/1`,
		"/config/testservice/fy/ads/prodsim/2": `/config/test-service/fy/ads/prod-sim/2`,
	}

	config := &ZKConfigs{
		FY: TenantsConfigs{
			Ads: map[string]map[string]AlgoConfigs{},
		},
	}
	zk := &V1{
		config:        config,
		handledPrefix: make(map[string]bool),
	}
	err := zk.handleStruct(&dataMap, &metaMap, config, "/config/test-service", false)
	assert.Nil(t, err)
	assert.Equal(t, true, config.FY.Ads["prod-sim"]["1"].Enabled)
	assert.Equal(t, false, config.FY.Ads["prod-sim"]["2"].Enabled)
	assert.Equal(t, "astring", config.FY.Ads["prod-sim"]["1"].Common["a"])
	assert.Equal(t, "bstring", config.FY.Ads["prod-sim"]["1"].Common["b"])
	assert.Equal(t, "ASTRING", config.FY.Ads["prod-sim"]["2"].Common["a"])
	assert.Equal(t, "bstring", config.FY.Ads["prod-sim"]["2"].Common["b"])
}
