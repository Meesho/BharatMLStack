package kubernetes

import (
	"encoding/json"
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/tidwall/gjson"
)

type HPA struct {
	Min             int               `json:"min"`
	Max             int               `json:"max"`
	Desired         int               `json:"desired"`
	Current         int               `json:"current"`
	InternalMetrics []InternalMetrics `json:"internalMetrics"`
}

type InternalMetrics struct {
	Name               string `json:"name"`
	AverageUtilization string `json:"averageUtilization"`
	AverageValue       string `json:"averageValue"`
}

func ParseHPAJson(hpa interface{}) (HPA, error) {
	log.Info().Msg("Entered ParseHPAJson func")
	HPAJson, err := json.Marshal(hpa)
	if err != nil {
		log.Error().Err(err).Msg("ParseHPAJson: Unable to convert HPA to Json")
		return HPA{}, errors.New("unable to convert HPA to Json")
	}

	manifest := gjson.Get(string(HPAJson), "manifest").String()
	manifestJSON := json.RawMessage(manifest)

	var hpaObj HPA
	hpaObj.Min = int(gjson.Get(string(manifestJSON), "spec.minReplicas").Int())
	hpaObj.Max = int(gjson.Get(string(manifestJSON), "spec.maxReplicas").Int())
	hpaObj.Desired = int(gjson.Get(string(manifestJSON), "status.desiredReplicas").Int())
	hpaObj.Current = int(gjson.Get(string(manifestJSON), "status.currentReplicas").Int())

	var internalMetrics []InternalMetrics
	internalMetrics = append(internalMetrics, InternalMetrics{
		Name:               "cpu",
		AverageUtilization: gjson.Get(string(manifestJSON), "status.currentMetrics.#(resource.name==\"cpu\").resource.current.averageUtilization").String(),
		AverageValue:       gjson.Get(string(manifestJSON), "status.currentMetrics.#(resource.name==\"cpu\").resource.current.averageValue").String(),
	})
	hpaObj.InternalMetrics = internalMetrics

	return hpaObj, nil
}

type AutoScalingPolicy struct {
	MinReplicas int `json:"minReplicas"`
	MaxReplicas int `json:"maxReplicas"`
}

func ParseScaledObjectJson(scaledObject interface{}) (AutoScalingPolicy, error) {
	log.Info().Msg("Entered ParseScaledObjectJson func")
	scaledObjectJson, err := json.Marshal(scaledObject)
	if err != nil {
		log.Error().Err(err).Msg("ParseScaledObjectJson: Unable to convert ScaledObject to Json")
		return AutoScalingPolicy{}, errors.New("unable to convert ScaledObject to Json")
	}

	manifest := gjson.Get(string(scaledObjectJson), "manifest").String()
	manifestJSON := json.RawMessage(manifest)

	var policy AutoScalingPolicy
	policy.MinReplicas = int(gjson.Get(string(manifestJSON), "spec.minReplicaCount").Int())
	policy.MaxReplicas = int(gjson.Get(string(manifestJSON), "spec.maxReplicaCount").Int())

	return policy, nil
}
