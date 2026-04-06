package grpcclient

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewConn(t *testing.T) {
	// Mocking viper settings
	viperSettings := map[string]interface{}{
		"EXAMPLE_HOST":            "example.com",
		"EXAMPLE_PORT":            "50051",
		"EXAMPLE_DEADLINE":        300_000,
		"EXAMPLE_GRPC_PLAIN_TEXT": true,
		"EXAMPLE_TIMEOUT_IN_MS":   300000,
	}

	for key, value := range viperSettings {
		viper.Set(key, value)
		defer viper.Reset()
	}

	conn := NewConn("EXAMPLE")

	assert.NotNil(t, conn)
	assert.NotNil(t, conn.Conn)
	assert.Equal(t, int64(300_000), conn.DeadLine)
}

func TestNewConfig_InvalidHost(t *testing.T) {
	var panicked bool
	defer func() {
		if r := recover(); r != nil {
			panicMsg := r.(string)
			assert.Equal(t, "invalidEnvPrefix_HOST not set", panicMsg)
			panicked = true
		}
	}()
	envPrefix := "invalidEnvPrefix"
	_ = NewConn(envPrefix)
	assert.True(t, panicked)
}

func TestNewConn_InvalidPort(t *testing.T) {
	var panicked bool
	defer func() {
		if r := recover(); r != nil {
			panicMsg := r.(string)
			assert.Equal(t, "EXAMPLE_PORT not set", panicMsg)
			panicked = true
		}
	}()
	viperSettings := map[string]interface{}{
		"EXAMPLE_HOST":            "example.com",
		"EXAMPLE_DEADLINE":        300_000,
		"EXAMPLE_GRPC_PLAIN_TEXT": true,
	}
	for key, value := range viperSettings {
		viper.Set(key, value)
		defer viper.Reset()
	}
	_ = NewConn("EXAMPLE")
	assert.True(t, panicked)
}
func TestNewConfig(t *testing.T) {
	// Mocking viper settings
	viperSettings := map[string]interface{}{
		"EXAMPLE_HOST":                  "example.com",
		"EXAMPLE_PORT":                  "50051",
		"EXAMPLE_DEADLINE":              300_000,
		"EXAMPLE_GRPC_PLAIN_TEXT":       true,
		"EXAMPLE_TIMEOUT_IN_MS":         300000,
		"EXAMPLE_LOAD_BALANCING_POLICY": "pick_first",
	}

	for key, value := range viperSettings {
		viper.Set(key, value)
		defer viper.Reset()
	}

	config := newConfig("EXAMPLE")

	assert.NotNil(t, config)
	assert.Equal(t, "example.com", config.Host)
	assert.Equal(t, "50051", config.Port)
	assert.Equal(t, 300_000, config.DeadLine)
	assert.Equal(t, "pick_first", config.LoadBalancingPolicy)
	assert.True(t, config.PlainText)
}

func TestNewConfig_DefaultChannelAlgo(t *testing.T) {
	// Mocking viper settings
	viperSettings := map[string]interface{}{
		"EXAMPLE_HOST":            "example.com",
		"EXAMPLE_PORT":            "50051",
		"EXAMPLE_DEADLINE":        300_000,
		"EXAMPLE_GRPC_PLAIN_TEXT": true,
		"EXAMPLE_TIMEOUT_IN_MS":   300000,
	}

	for key, value := range viperSettings {
		viper.Set(key, value)
		defer viper.Reset()
	}

	config := newConfig("EXAMPLE")

	assert.NotNil(t, config)
	assert.Equal(t, "example.com", config.Host)
	assert.Equal(t, "50051", config.Port)
	assert.Equal(t, 300_000, config.DeadLine)
	assert.Equal(t, "round_robin", config.LoadBalancingPolicy)
	assert.True(t, config.PlainText)
}

func TestGetGRPCConnections(t *testing.T) {
	config := Config{
		Host:      "example.com",
		Port:      "50051",
		DeadLine:  300_000,
		PlainText: false,
	}

	conn, err := getGRPCConnections(config)

	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.NotNil(t, conn.Conn)
	assert.Equal(t, int64(300_000), conn.DeadLine)
}

func TestGetGRPCConnections_Error(t *testing.T) {
	config := Config{
		Host:      "",
		Port:      "",
		DeadLine:  300_000,
		PlainText: false,
	}
	conn, err := getGRPCConnections(config)
	assert.Nil(t, conn)
	assert.NotNil(t, err)
}

func TestNewConnFromConfig(t *testing.T) {
	config := &Config{
		Host:      "example.com",
		Port:      "50051",
		DeadLine:  300_000,
		PlainText: true,
	}

	conn := NewConnFromConfig(config, "")

	assert.NotNil(t, conn)
	assert.NotNil(t, conn.Conn)
	assert.Equal(t, int64(300_000), conn.DeadLine)
}
