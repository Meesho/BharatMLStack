package github

import (
	"fmt"

	"github.com/maolinc/copier"
	"github.com/rs/zerolog/log"
)

type TriggerUpdater interface {
	UpdateTriggers(triggers []interface{}, threshold string) []interface{}
	CreateNewTrigger(threshold string) map[string]interface{}
	GetTriggerType() string
	GetTriggerIdentifier() string
	MatchesTrigger(trigger map[string]interface{}) bool
}

func GetTriggerUpdater(triggerType, appName string) TriggerUpdater {
	switch triggerType {
	case "cpu":
		return &CPUTriggerUpdater{}
	case "gpu":
		return &GPUTriggerUpdater{appName: appName}
	default:
		log.Error().Str("triggerType", triggerType).Msg("Unknown trigger type")
		return nil
	}
}

// CPUTriggerUpdater implements TriggerUpdater for CPU triggers
type CPUTriggerUpdater struct{}

func (c *CPUTriggerUpdater) GetTriggerType() string {
	return "cpu"
}

func (c *CPUTriggerUpdater) GetTriggerIdentifier() string {
	return "type"
}

func (c *CPUTriggerUpdater) MatchesTrigger(trigger map[string]interface{}) bool {
	triggerType, exists := trigger["type"].(string)
	return exists && triggerType == "cpu"
}

func (c *CPUTriggerUpdater) UpdateTriggers(triggers []interface{}, threshold string) []interface{} {
	log.Info().Int("existingTriggersCount", len(triggers)).Msg("Entered CPUTriggerUpdater.UpdateTriggers")

	var newTriggers []interface{}
	copier.CopyWithOption(&newTriggers, triggers, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	log.Info().Int("preservedTriggersCount", len(newTriggers)).Msg("Preserved all existing triggers")

	var triggerFound bool = false
	for i, trigger := range newTriggers {
		triggerMap, ok := trigger.(map[string]interface{})
		if !ok {
			log.Warn().Int("triggerIndex", i).Msg("Invalid trigger format, skipping")
			continue
		}

		if c.MatchesTrigger(triggerMap) {
			triggerFound = true
			log.Info().Int("triggerIndex", i).Str("newThreshold", threshold).Msg("Found existing CPU trigger, updating threshold")

			var metadataMap map[string]interface{}
			copier.CopyWithOption(&metadataMap, triggerMap["metadata"], copier.Option{IgnoreEmpty: true, DeepCopy: true})
			metadataMap["value"] = threshold

			triggerMap["metadata"] = metadataMap
			break
		} else {
			triggerType, _ := triggerMap["type"].(string)
			log.Info().Int("triggerIndex", i).Str("triggerType", triggerType).Msg("Preserving existing non-CPU trigger")
		}
	}

	if !triggerFound {
		log.Info().Str("threshold", threshold).Msg("No existing CPU trigger found, adding new CPU trigger")
		newTriggers = append(newTriggers, c.CreateNewTrigger(threshold))
	}

	log.Info().Int("totalTriggersCount", len(newTriggers)).Msg("Final triggers configuration")
	return newTriggers
}

func (c *CPUTriggerUpdater) CreateNewTrigger(threshold string) map[string]interface{} {
	log.Info().Msg("Creating new CPU trigger")
	metadata := make(map[string]interface{})
	metadata["value"] = threshold

	trigger := make(map[string]interface{})
	trigger["metadata"] = metadata
	trigger["type"] = "cpu"
	trigger["metricType"] = "Utilization"

	return trigger
}

// GPUTriggerUpdater implements TriggerUpdater for GPU triggers
type GPUTriggerUpdater struct {
	appName string
}

func (g *GPUTriggerUpdater) GetTriggerType() string {
	return "prometheus"
}

func (g *GPUTriggerUpdater) GetTriggerIdentifier() string {
	return "metricName"
}

func (g *GPUTriggerUpdater) MatchesTrigger(trigger map[string]interface{}) bool {
	triggerType, typeExists := trigger["type"].(string)
	if !typeExists || triggerType != "prometheus" {
		return false
	}

	if metadata, ok := trigger["metadata"].(map[string]interface{}); ok {
		if metricName, exists := metadata["metricName"].(string); exists {
			return metricName == "gpu_utilisation"
		}
	}
	return false
}

func (g *GPUTriggerUpdater) UpdateTriggers(triggers []interface{}, threshold string) []interface{} {
	log.Info().Int("existingTriggersCount", len(triggers)).Msg("Entered GPUTriggerUpdater.UpdateTriggers")

	var newTriggers []interface{}
	copier.CopyWithOption(&newTriggers, triggers, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	log.Info().Int("preservedTriggersCount", len(newTriggers)).Msg("Preserved all existing triggers")

	var triggerFound bool = false
	for i, trigger := range newTriggers {
		triggerMap, ok := trigger.(map[string]interface{})
		if !ok {
			log.Warn().Int("triggerIndex", i).Msg("Invalid trigger format, skipping")
			continue
		}

		triggerType, typeExists := triggerMap["type"].(string)
		if !typeExists {
			log.Warn().Int("triggerIndex", i).Msg("Trigger missing type field, skipping")
			continue
		}

		if g.MatchesTrigger(triggerMap) {
			triggerFound = true
			log.Info().Int("triggerIndex", i).Str("newThreshold", threshold).Msg("Found existing GPU trigger, updating threshold")

			var metadataMap map[string]interface{}
			copier.CopyWithOption(&metadataMap, triggerMap["metadata"], copier.Option{IgnoreEmpty: true, DeepCopy: true})
			metadataMap["threshold"] = threshold

			triggerMap["metadata"] = metadataMap
			break
		} else {
			log.Info().Int("triggerIndex", i).Str("triggerType", triggerType).Msg("Preserving existing non-GPU trigger")
		}
	}

	if !triggerFound {
		log.Info().Str("threshold", threshold).Msg("No existing GPU trigger found, adding new GPU trigger")
		newTriggers = append(newTriggers, g.CreateNewTrigger(threshold))
	}

	log.Info().Int("totalTriggersCount", len(newTriggers)).Msg("Final triggers configuration")
	return newTriggers
}

func (g *GPUTriggerUpdater) CreateNewTrigger(threshold string) map[string]interface{} {
	log.Info().Msg("Creating new GPU trigger")
	metadata := make(map[string]interface{})
	metadata["metricName"] = "gpu_utilisation"
	metadata["query"] = fmt.Sprintf("(avg(nv_gpu_utilization{kubernetes_namespace=\"%s\"}[1m])*count(nv_gpu_utilization{kubernetes_namespace=\"%s\"}))*100", g.appName, g.appName)
	// serverAddress should be configured via VICTORIAMETRICS_SERVER_ADDRESS env var
	// Default value provided for backward compatibility
	serverAddress := getVictoriaMetricsServerAddress()
	metadata["serverAddress"] = serverAddress
	metadata["threshold"] = threshold

	trigger := make(map[string]interface{})
	trigger["metadata"] = metadata
	trigger["type"] = "prometheus"

	return trigger
}

// getVictoriaMetricsServerAddress returns the configured VictoriaMetrics server address
// This should be set via VICTORIAMETRICS_SERVER_ADDRESS environment variable
var victoriaMetricsServerAddress = "http://vmselect-datascience-prd-proxy.victoriametrics.svc.cluster.local:8481/select/100/prometheus/"

func SetVictoriaMetricsServerAddress(address string) {
	if address != "" {
		victoriaMetricsServerAddress = address
	}
}

func getVictoriaMetricsServerAddress() string {
	return victoriaMetricsServerAddress
}
