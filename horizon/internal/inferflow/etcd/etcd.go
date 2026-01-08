package etcd

import (
	"encoding/json"
	"fmt"

	"github.com/Meesho/BharatMLStack/horizon/internal/inferflow"

	"github.com/Meesho/BharatMLStack/horizon/pkg/etcd"
	"github.com/rs/zerolog/log"
)

type Etcd struct {
	inferflowInstance etcd.Etcd
	horizonInstance   etcd.Etcd
	appName           string
	env               string
}

func NewEtcdInstance() *Etcd {
	return &Etcd{
		inferflowInstance: etcd.Instance()[inferflow.InferflowAppName],
		horizonInstance:   etcd.Instance()[inferflow.HorizonAppName],
		appName:           inferflow.InferflowAppName,
		env:               inferflow.AppEnv,
	}
}

func (e *Etcd) GetInferflowEtcdInstance() *ModelConfigRegistery {
	instance, ok := e.inferflowInstance.GetConfigInstance().(*ModelConfigRegistery)
	if !ok {
		log.Panic().Msg("invalid etcd instance")
	}
	return instance
}

func (e *Etcd) GetHorizonEtcdInstance() *HorizonRegistry {
	instance, ok := e.horizonInstance.GetConfigInstance().(*HorizonRegistry)
	if !ok {
		log.Panic().Msg("invalid etcd instanc	e")
	}
	return instance
}

func (e *Etcd) GetComponentData(componentName string) *ComponentData {
	registry := e.GetHorizonEtcdInstance()
	if registry == nil {
		log.Error().Msg("GetComponentData called on nil registry")
		return nil
	}

	component, ok := registry.Inferflow.InferflowComponents[componentName]
	if !ok {
		log.Error().Msgf("component data for '%s' not found in registry", componentName)
		return nil
	}

	return &component
}

func (e *Etcd) CreateConfig(serviceName string, ConfigId string, InferflowConfig InferflowConfig) error {
	// Marshal the struct directly to preserve field order as defined in the struct
	configJson, err := json.Marshal(InferflowConfig)
	if err != nil {
		return err
	}

	return e.inferflowInstance.CreateNode(fmt.Sprintf("/config/%s/services/%s/model-config/config-map/%s", e.appName, serviceName, ConfigId), string(configJson))
}

func (e *Etcd) UpdateConfig(serviceName string, ConfigId string, InferflowConfig InferflowConfig) error {
	// Marshal the struct directly to preserve field order as defined in the struct
	configJson, err := json.Marshal(InferflowConfig)
	if err != nil {
		return err
	}
	return e.inferflowInstance.SetValue(fmt.Sprintf("/config/%s/services/%s/model-config/config-map/%s", e.appName, serviceName, ConfigId), string(configJson))
}

func (e *Etcd) DeleteConfig(serviceName string, ConfigId string) error {
	return e.inferflowInstance.DeleteNode(fmt.Sprintf("/config/%s/services/%s/model-config/config-map/%s", e.appName, serviceName, ConfigId))
}
