package config

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	configType = "yaml"
)

func Init(yamlConf interface{}, yamlConfigReader io.Reader) {
	viper.SetConfigType(configType)
	//viper.AddConfigPath(configPath) // Add the current directory as the search path for the config file

	if err := viper.ReadConfig(yamlConfigReader); err != nil {
		panic(fmt.Errorf("failed to read the configuration file: %w", err))
	}

	// Replace environment variable placeholders in the configuration
	replaceEnvVarPlaceholders(viper.GetViper())

	// Load environment variables
	viper.AutomaticEnv()

	// Unmarshal the configuration into the struct
	if err := viper.Unmarshal(yamlConf); err != nil {
		panic(fmt.Errorf("failed to unmarshal configuration: %w", err))
	}
	log.Info().Msg("Viper initialized!")
}

func replaceEnvVarPlaceholders(v *viper.Viper) {
	re := regexp.MustCompile(`\${([^}]+)}`)
	missedEnvVars := make([]string, 0)
	for _, key := range v.AllKeys() {
		value := v.GetString(key)
		matches := re.FindAllStringSubmatch(value, -1)
		for _, match := range matches {
			envVarName := match[1]
			envVarValue := os.Getenv(envVarName)
			if len(envVarValue) == 0 {
				missedEnvVars = append(missedEnvVars, envVarName)
			}
			value = strings.ReplaceAll(value, match[0], envVarValue)
		}
		v.Set(key, value)
	}
	if len(missedEnvVars) != 0 {
		panic("Missing environment variables: " + strings.Join(missedEnvVars, ","))
	}
}
