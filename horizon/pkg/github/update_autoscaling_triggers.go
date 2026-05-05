package github

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/horizon/pkg/argocd"
	"github.com/rs/zerolog/log"
)

// UpdateAutoscalingTriggers updates the full triggers array in both values.yaml and values_properties.yaml
// This allows adding/updating multiple trigger types (CPU, cron, prometheus, etc.) in a single operation
// The triggers array replaces the existing triggers, preserving the structure
func UpdateAutoscalingTriggers(argocdApplication argocd.ArgoCDApplicationMetadata, triggers []interface{}, email string, workingEnv string) error {
	log.Info().
		Str("ApplicationName", argocdApplication.Name).
		Int("triggersCount", len(triggers)).
		Str("UpdatedBy", email).
		Str("WorkingEnv", workingEnv).
		Msg("Updating autoscaling triggers array")

	// Update values_properties.yaml
	err := updateTriggersInValuesProperties(argocdApplication, triggers, email, workingEnv)
	if err != nil {
		log.Error().Err(err).Msg("UpdateAutoscalingTriggers: Failed to update values_properties.yaml")
		return err
	}

	// Update values.yaml
	err = updateTriggersInValues(argocdApplication, triggers, email, workingEnv)
	if err != nil {
		log.Error().Err(err).Msg("UpdateAutoscalingTriggers: Failed to update values.yaml")
		return err
	}

	return nil
}

// updateTriggersInValues updates the triggers array in values.yaml
func updateTriggersInValues(argocdApplication argocd.ArgoCDApplicationMetadata, triggers []interface{}, email string, workingEnv string) error {
	log.Info().
		Str("ApplicationName", argocdApplication.Name).
		Int("triggersCount", len(triggers)).
		Str("WorkingEnv", workingEnv).
		Msg("Updating triggers in values.yaml")

	data, sha, err := GetFileYaml(argocdApplication, "values.yaml", workingEnv)
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve values.yaml")
		return err
	}

	// Ensure autoscaling section exists
	if _, ok := data["autoscaling"]; !ok {
		data["autoscaling"] = make(map[string]interface{})
	}
	autoscaling, ok := data["autoscaling"].(map[string]interface{})
	if !ok {
		autoscaling = make(map[string]interface{})
		data["autoscaling"] = autoscaling
	}

	// Update triggers array
	autoscaling["triggers"] = triggers
	data["autoscaling"] = autoscaling

	commitMsg := fmt.Sprintf("%s: Autoscaling triggers updated by %s", argocdApplication.Name, email)
	err = UpdateFileYaml(argocdApplication, data, "values.yaml", commitMsg, sha, workingEnv)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update values.yaml")
		return err
	}

	return nil
}

// updateTriggersInValuesProperties updates the triggers array in values_properties.yaml
func updateTriggersInValuesProperties(argocdApplication argocd.ArgoCDApplicationMetadata, triggers []interface{}, email string, workingEnv string) error {
	log.Info().
		Str("ApplicationName", argocdApplication.Name).
		Int("triggersCount", len(triggers)).
		Str("WorkingEnv", workingEnv).
		Msg("Updating triggers in values_properties.yaml")

	data, sha, err := GetFileYaml(argocdApplication, "values_properties.yaml", workingEnv)
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve values_properties.yaml")
		return err
	}

	// Update triggers array directly
	data["triggers"] = triggers

	commitMsg := fmt.Sprintf("%s: Autoscaling triggers updated by %s", argocdApplication.Name, email)
	err = UpdateFileYaml(argocdApplication, data, "values_properties.yaml", commitMsg, sha, workingEnv)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update values_properties.yaml")
		return err
	}

	return nil
}

