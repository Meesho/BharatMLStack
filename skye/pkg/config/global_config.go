package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/Meesho/go-core/v2/config/configutils"
	"github.com/Meesho/go-core/v2/zookeeper"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	envPodNamespace                = "POD_NAMESPACE"
	envConfigLocation              = "CONFIG_LOCATION"
	envEnvironment                 = "ENVIRONMENT"
	envDeployableName              = "DEPLOYABLE_NAME"
	configKeyDynamicSource         = "dynamic-config.source"
	configKeyDynamicSourceVersion  = "dynamic-config.version"
	configKeyDynamicSourceOptional = "dynamic-config.optional"
)

type GlobalConf interface {
	GetStaticConfig() interface{}
	GetDynamicConfig() interface{}
}

func InitGlobalConfig(conf GlobalConf) {
	InitEnv()

	environment, deployableName := getEnvironmentAndDeployableName()

	configLocation := viper.GetString(envConfigLocation)
	if configLocation == "" {
		configLocation = "/opt/config"
	}

	staticConfigFilePath := fmt.Sprintf("%s/application-%s.yml", configLocation, environment)
	dynamicConfigFilePath := fmt.Sprintf("%s/application-dyn-%s.yml", configLocation, environment)

	if err := loadStaticConfig(staticConfigFilePath, conf.GetStaticConfig(), deployableName); err != nil {
		panic(fmt.Sprintf("Failed to load static config: %v", err))
	}

	if err := loadDynamicConfig(dynamicConfigFilePath, conf.GetDynamicConfig(), deployableName); err != nil {
		panic(fmt.Sprintf("Failed to load dynamic config: %v", err))
	}
}

func loadStaticConfig(filePath string, config interface{}, deployableName string) error {
	configBytes, err := readConfig(filePath, deployableName)
	if err != nil {
		return err
	}
	if configBytes == nil || len(configBytes) == 0 {
		return nil
	}

	viper.SetConfigType("yaml")
	viper.ReadConfig(bytes.NewBuffer(configBytes))

	replaceEnvVarPlaceholders(viper.GetViper())
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(configBytes, &configMap); err != nil {
		return fmt.Errorf("failed to unmarshal YAML to map: %w", err)
	}

	//setting the config keys in the format of `key1.key2` to `KEY1_KEY2`
	//like `db.host` to `DB_HOST`
	setFlattenedKeys(viper.AllSettings(), "")

	if err := viper.Unmarshal(config); err != nil {
		return fmt.Errorf("failed to unmarshal Viper settings into struct: %w", err)
	}

	return nil
}

func loadDynamicConfig(filePath string, targetConfig interface{}, deployableName string) error {
	configData, err := readConfig(filePath, deployableName)
	if err != nil {
		return err
	}
	if configData == nil || len(configData) == 0 {
		configData = []byte{}
	}

	var yamlConfig map[string]interface{}
	if err := yaml.Unmarshal(configData, &yamlConfig); err != nil {
		return fmt.Errorf("failed to decode dynamic configuration as YAML: %w", err)
	}

	// normalizedPathToValue stores flattened configuration values with normalized keys
	// Example: For YAML like:
	//   database:
	//     connect-string: "postgres://localhost:5432/db"
	// This map would contain:
	//   "database.connectstring" → "postgres://localhost:5432/db"
	// normalizedPathToOriginalPath maintains mapping of normalized paths to original paths
	// Example: "database.connectstring" → "database.connect-string"

	normalizedPathToValue := make(map[string]string)
	normalizedPathToOriginalPath := make(map[string]string)

	configutils.NestedMapToPathMap(yamlConfig, "", normalizedPathToValue)
	configutils.NormalizePathMap(normalizedPathToValue, normalizedPathToOriginalPath)
	err = configutils.MapToStruct(&normalizedPathToValue, &normalizedPathToOriginalPath, targetConfig, "")
	if err != nil {
		return fmt.Errorf("failed to map dynamic configuration to struct: %w", err)
	}

	initializeDynamicConfigSource(targetConfig)
	return nil
}

func initializeDynamicConfigSource(config interface{}) {
	if !viper.IsSet(configKeyDynamicSource) {
		return
	}
	dynamicConfigSource := viper.GetString(configKeyDynamicSource)
	switch dynamicConfigSource {
	case "zookeeper":
		initializeZookeeper(config)
	case "etcd":
		initializeEtcd(config)
	default:
		handleDynamicConfigError(fmt.Sprintf("Unknown dynamic config source: %s", dynamicConfigSource))
	}
}

func initializeZookeeper(config interface{}) {
	version := viper.GetInt(configKeyDynamicSourceVersion)
	if version <= 0 {
		version = 1
	}
	log.Info().Msg("Initializing Zookeeper Connection")
	zookeeper.Init(version, config)
}

func initializeEtcd(config interface{}) {
	log.Info().Msg("ETCD initialization not implemented yet")
}

func handleDynamicConfigError(message string) {
	isOptional := viper.GetBool(configKeyDynamicSourceOptional)
	if isOptional {
		log.Error().Msg(message)
	} else {
		log.Fatal().Msg(message)
	}
}

// Each YAML file will have a default section common to all deployables,
// and a deployable-specific section with a key `deployable-name` to define the deployable name.
// We need to merge the deployable-specific section into the default section.
//
// Example:
//
//	log_level: INFO
//	db_host: localhost
//	db_port: 5432
//
//	---
//
//	deployable-name: deployable-primary
//	db_host: prod-db.example.com
//	db_port: 5432
//	db_user: prod_user
//	db_password: prod_password
//
//	---
//
//	deployable-name: deployable-secondary
//	db_host: staging-db.example.com
//	db_port: 5432
//	db_user: prod_user_secondary
//	db_password: secondary_password
//
// Result after merging for deployable-primary:
//
//	deployable-name: deployable-primary
//	log_level: INFO
//	db_host: prod-db.example.com
//	db_port: 5432
//	db_user: prod_user
//	db_password: prod_password

func readConfig(filePath, deployableName string) ([]byte, error) {
	sections, err := getSections(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get sections from file: %w", err)
	}
	if sections == nil || len(sections) == 0 {
		return nil, nil
	}

	defaultSection := sections[0]
	deployableSpecificSection := make(map[string]interface{})

	for _, section := range sections {
		if section["deployable-name"] == deployableName {
			deployableSpecificSection = section
			break
		}
	}

	mergedSection := mergeMaps(deployableSpecificSection, defaultSection)

	mergedConfigData, err := yaml.Marshal(mergedSection)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged config: %w", err)
	}

	return mergedConfigData, nil
}

func getSections(filePath string) ([]map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var sections []map[string]interface{}
	decoder := yaml.NewDecoder(file)

	for {
		var section map[string]interface{}
		if err := decoder.Decode(&section); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to decode YAML section: %w", err)
		}
		sections = append(sections, section)
	}
	return sections, nil
}

// mergeMaps merges two maps, with values from the source map overwriting those in the target map.
// If both maps contain a nested map for the same key, the nested maps are recursively merged.
//
// Parameters:
// - source: The source map whose values will overwrite those in the target map.
// - target: The target map that will be updated with values from the source map.
//
// Returns:
// - A map[string]interface{} that contains the merged values from both the source and target maps.
//
// Example:
//
//	source := map[string]interface{}{
//	    "key1": "value1",
//	    "key2": map[string]interface{}{
//	        "nestedKey1": "nestedValue1",
//	    },
//	}
//	target := map[string]interface{}{
//	    "key2": map[string]interface{}{
//	        "nestedKey1": "oldValue",
//	        "nestedKey2": "nestedValue2",
//	    },
//	    "key3": "value3",
//	}
//	result := mergeMaps(source, target)
//	   result will be:
//	   map[string]interface{}{
//	       "key1": "value1",
//	       "key2": map[string]interface{}{
//	           "nestedKey1": "nestedValue1",
//	           "nestedKey2": "nestedValue2",
//	       },
//	       "key3": "value3",
//	   }
func mergeMaps(source, target map[string]interface{}) map[string]interface{} {
	for key, sourceValue := range source {
		if targetValue, exists := target[key]; exists {
			if targetMap, isTargetMap := targetValue.(map[string]interface{}); isTargetMap {
				if sourceMap, isSourceMap := sourceValue.(map[string]interface{}); isSourceMap {
					target[key] = mergeMaps(sourceMap, targetMap)
					continue
				}
			}
		}
		target[key] = sourceValue
	}
	return target
}

func getEnvironmentAndDeployableName() (string, string) {
	podNamespace := viper.GetString(envPodNamespace)
	namespaceParts := strings.SplitN(podNamespace, "-", 2)
	if len(namespaceParts) < 2 {
		if !viper.IsSet(envEnvironment) || !viper.IsSet(envDeployableName) {
			log.Panic().Msg("Environment and Deployable Name not set")
		}
		return viper.GetString(envEnvironment), viper.GetString(envDeployableName)
	}
	return namespaceParts[0], namespaceParts[1]
}

func normaliseKeys(input map[string]interface{}) map[string]interface{} {
	normalised := make(map[string]interface{})
	for key, value := range input {
		newKey := strings.ToLower(strings.ReplaceAll(key, "-", ""))
		if nestedMap, ok := value.(map[string]interface{}); ok {
			normalised[newKey] = normaliseKeys(nestedMap)
			continue
		}
		normalised[newKey] = value
	}
	return normalised
}

func setFlattenedKeys(configMap map[string]interface{}, prefix string) {
	for key, value := range configMap {
		fullKey := key
		if len(prefix) != 0 {
			fullKey = prefix + "." + key
		}

		if nestedMap, ok := value.(map[string]interface{}); ok {
			setFlattenedKeys(nestedMap, fullKey)
		} else {
			underscoreKey := strings.ToUpper(strings.ReplaceAll(fullKey, ".", "_"))
			viper.Set(underscoreKey, value)
		}
	}
}
