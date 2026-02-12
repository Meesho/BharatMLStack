package configstructexample

import (
	"embed"
	"fmt"
	"io"
	"os"

	"github.com/Meesho/BharatMLStack/skye/pkg/config"
)

type YourConfigStruct struct {
	App struct {
		Name        string `mapstructure:"name"`
		Version     string `mapstructure:"version"`
		Environment string `mapstructure:"environment"`
		APIKey      string `mapstructure:"api_key"`
		LogLevel    string `mapstructure:"log_level"`
	} `mapstructure:"app"`
}

//go:embed test.yaml
var content embed.FS

func start() {
	os.Setenv("APP_NAME", "test-app")
	os.Setenv("APP_VERSION", "1.0.0")
	os.Setenv("APP_ENVIRONMENT", "test")
	os.Setenv("APP_API_KEY", "test-api-key")
	os.Setenv("APP_LOG_LEVEL", "INFO")

	var sample YourConfigStruct
	file, err := content.Open("test.yaml")
	if err != nil {
		panic(err)
	}

	config.Init(&sample, io.Reader(file))

	fmt.Printf("YourConfigStruct config - %s\n", sample)
	println(sample.App.Name)
	println(sample.App.Version)
	println(sample.App.Environment)
	println(sample.App.APIKey)
	println(sample.App.LogLevel)
}
