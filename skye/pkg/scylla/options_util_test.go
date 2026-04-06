package scylla

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildClusterConfigFromEnv_NoContactPoints(t *testing.T) {
	_, err := BuildClusterConfigFromEnv("TEST")
	assert.NotNil(t, err)
	assert.Equal(t, "TEST_CONTACT_POINTS not set", err.Error())
}

func TestBuildClusterConfigFromEnv_NoPort(t *testing.T) {
	viper.Set("TEST_CONTACT_POINTS", "127.0.0.1")

	_, err := BuildClusterConfigFromEnv("TEST")
	assert.NotNil(t, err)
	assert.Equal(t, "TEST_PORT not set", err.Error())
}

func TestBuildClusterConfigFromEnv_NoKeyspace(t *testing.T) {
	viper.Set("TEST_CONTACT_POINTS", "127.0.0.1")
	viper.Set("TEST_PORT", "9042")

	_, err := BuildClusterConfigFromEnv("TEST")
	assert.NotNil(t, err)
	assert.Equal(t, "TEST_KEYSPACE not set", err.Error())
}

func TestBuildClusterConfigFromEnv_Success(t *testing.T) {
	viper.Set("TEST_CONTACT_POINTS", "127.0.0.1")
	viper.Set("TEST_PORT", "9042")
	viper.Set("TEST_KEYSPACE", "test_keyspace")

	cfg, err := BuildClusterConfigFromEnv("TEST")
	assert.Nil(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, []string{"127.0.0.1"}, cfg.Hosts)
	assert.Equal(t, 9042, cfg.Port)
	assert.Equal(t, "test_keyspace", cfg.Keyspace)
}
