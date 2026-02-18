package configstructexample

import (
	"fmt"
	"os"

	"github.com/Meesho/BharatMLStack/skye/pkg/config"
)

type StaticConfig struct {
	App struct {
		Name        string `mapstructure:"name"`
		Version     string `mapstructure:"version"`
		Environment string `mapstructure:"environment"`
		LogLevel    string `mapstructure:"log_level"`
	} `mapstructure:"app"`
}

type DynamicConfig struct {
	FeatureToggle bool   `mapstructure:"feature_toggle"`
	APIEndpoint   string `mapstructure:"api_endpoint"`
	Timeout       int    `mapstructure:"timeout"`
}

type MyConfig struct {
	StaticConfig  StaticConfig
	DynamicConfig DynamicConfig
}

func (c *MyConfig) GetStaticConfig() interface{} {
	return &c.StaticConfig
}

func (c *MyConfig) GetDynamicConfig() interface{} {
	return &c.DynamicConfig
}

//lint:ignore U1000 example entrypoint: go run .
func main() {
	os.Setenv("POD_NAMESPACE", "dev-myapp")
	os.Setenv("CONFIG_LOCATION", "/absolute/path/to/config/directory")

	myConfig := &MyConfig{
		DynamicConfig: DynamicConfig{},
		StaticConfig:  StaticConfig{},
	}

	config.InitGlobalConfig(myConfig)

	fmt.Println("Static Config:", myConfig.StaticConfig)
	fmt.Println("Dynamic Config:", myConfig.DynamicConfig)
}
