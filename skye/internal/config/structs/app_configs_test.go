package structs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAppConfig(t *testing.T) {
	cfg := GetAppConfig()
	assert.NotNil(t, cfg)
	assert.NotNil(t, &cfg.Configs)
	assert.NotNil(t, &cfg.DynamicConfigs)
}

func TestAppConfig_GetStaticConfig(t *testing.T) {
	cfg := GetAppConfig()
	static := cfg.GetStaticConfig()
	assert.NotNil(t, static)
	_, ok := static.(*Configs)
	assert.True(t, ok)
}

func TestAppConfig_GetDynamicConfig(t *testing.T) {
	cfg := GetAppConfig()
	dynamic := cfg.GetDynamicConfig()
	assert.NotNil(t, dynamic)
	_, ok := dynamic.(*DynamicConfigs)
	assert.True(t, ok)
}
