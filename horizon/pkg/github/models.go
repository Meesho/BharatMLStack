package github

import (
	"errors"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type AutoScalinGitHubProperties struct {
	MinReplicas string `json:"minReplicas" yaml:"minReplicas"`
	MaxReplicas string `json:"maxReplicas" yaml:"maxReplicas"`
}

type AutoScalinBulkMinMaxGitHubProperties struct {
	ApplicationName string `json:"applicationName" yaml:"applicationName"`
	MinReplicas     string `json:"minReplicas" yaml:"minReplicas"`
	MaxReplicas     string `json:"maxReplicas" yaml:"maxReplicas"`
}

type AutoScalinGitHubCronProperties struct {
	Start           string `json:"start" yaml:"start"`
	End             string `json:"end" yaml:"end"`
	DesiredReplicas string `json:"desiredReplicas" yaml:"desiredReplicas"`
}

type AutoScalinGitHubCPUThreshold struct {
	CPUThreshold string `json:"cpuThreshold" yaml:"cpuThreshold"`
}

type AutoScalinGitHubGPUThreshold struct {
	GPUThreshold string `json:"gpuThreshold" yaml:"gpuThreshold"`
}

// ThresholdProperties interface for unified threshold handling
type ThresholdProperties interface {
	GetThreshold() string
	GetTriggerType() string
	Validate() error
}

// Implement ThresholdProperties for CPU
func (c AutoScalinGitHubCPUThreshold) GetThreshold() string {
	return c.CPUThreshold
}

func (c AutoScalinGitHubCPUThreshold) GetTriggerType() string {
	return "cpu"
}

func (c AutoScalinGitHubCPUThreshold) Validate() error {
	return c.ValidateAutoScalinGitHubCPU()
}

// Implement ThresholdProperties for GPU
func (g AutoScalinGitHubGPUThreshold) GetThreshold() string {
	return g.GPUThreshold
}

func (g AutoScalinGitHubGPUThreshold) GetTriggerType() string {
	return "gpu"
}

func (g AutoScalinGitHubGPUThreshold) Validate() error {
	return g.ValidateAutoScalinGitHubGPU()
}

func (a AutoScalinGitHubProperties) ValidateAutoScalinGitHubProperties() error {
	log.Info().Str("MinReplicas", a.MinReplicas).Str("MaxReplicas", a.MaxReplicas).Msg("Validating AutoScalinGitHubProperties")
	if a.MinReplicas == "" || a.MaxReplicas == "" {
		log.Error().Msg("Validation failed: Empty values provided")
		return errors.New("why supplying empty values?")
	}
	// Convert to int
	intMin, err := strconv.Atoi(a.MinReplicas)
	if err != nil {
		log.Error().Err(err).Msg("ValidateAutoScalinGitHubProperties: Failed to convert MinReplicas to integer")
		return err
	}

	intMax, err := strconv.Atoi(a.MaxReplicas)
	if err != nil {
		log.Error().Err(err).Msg("ValidateAutoScalinGitHubProperties: Failed to convert MaxReplicas to integer")
		return err
	}

	if intMin <= 0 || intMax <= 0 {
		log.Error().Int("MinReplicas", intMin).Int("MaxReplicas", intMax).Msg("Validation failed: Min/Max cannot be zero or negative")
		return errors.New("min/max cannot be zero or negative")
	}

	if intMin > intMax {
		log.Error().Int("MinReplicas", intMin).Int("MaxReplicas", intMax).Msg("Validation failed: MinReplicas greater than MaxReplicas")
		return errors.New("min cannot be greater than max")
	}

	return nil
}

func (a AutoScalinGitHubCronProperties) ValidateCronGitHubProperties() error {
	log.Info().Str("Start", a.Start).Str("End", a.End).Str("DesiredReplicas", a.DesiredReplicas).Msg("Entered ValidateCronGitHubProperties")
	if a.Start == "" || a.End == "" || a.DesiredReplicas == "" {
		log.Error().Msg("ValidateCronGitHubProperties: Validation failed: Empty values provided")
		return errors.New("why supplying empty values?")
	}
	// Convert to int
	intDesiredReplicas, err := strconv.Atoi(a.DesiredReplicas)
	if err != nil {
		log.Error().Err(err).Msg("ValidateCronGitHubProperties: Failed to convert DesiredReplicas to integer")
		return err
	}

	if intDesiredReplicas <= 0 {
		log.Error().Int("DesiredReplicas", intDesiredReplicas).Msg("ValidateCronGitHubProperties: Validation failed: DesiredReplicas cannot be zero or negative")
		return errors.New("desiredReplicas cannot be zero")
	}

	return nil
}

func (a AutoScalinGitHubCPUThreshold) ValidateAutoScalinGitHubCPU() error {
	log.Info().Str("CPUThreshold", a.CPUThreshold).Msg("In ValidateAutoScalinGitHubCPUThreshold")
	// Trim whitespace before validation
	trimmedThreshold := strings.TrimSpace(a.CPUThreshold)
	if trimmedThreshold == "" {
		log.Error().Msg("ValidateAutoScalinGitHubCPU: Validation failed: Empty CPUThreshold provided")
		return errors.New("why supplying empty values?")
	}

	// Convert to int
	intMin, err := strconv.Atoi(trimmedThreshold)
	if err != nil {
		log.Error().Err(err).Msg("ValidateAutoScalinGitHubCPU: Failed to convert CPUThreshold to integer")
		return err
	}

	if intMin <= 0 {
		log.Error().Int("CPUThreshold", intMin).Msg("ValidateAutoScalinGitHubCPU: Validation failed: CPUThreshold cannot be zero or negative")
		return errors.New("cpu threshold cannot be zero or negative")
	}

	return nil
}

func (a AutoScalinGitHubGPUThreshold) ValidateAutoScalinGitHubGPU() error {
	log.Info().Str("GPUThreshold", a.GPUThreshold).Msg("In ValidateAutoScalinGitHubGPUThreshold")
	// Trim whitespace before validation
	trimmedThreshold := strings.TrimSpace(a.GPUThreshold)
	if trimmedThreshold == "" {
		log.Error().Msg("ValidateAutoScalinGitHubGPU: Validation failed: Empty GPUThreshold provided")
		return errors.New("why supplying empty values?")
	}

	intMin, err := strconv.Atoi(trimmedThreshold)
	if err != nil {
		log.Error().Err(err).Msg("ValidateAutoScalinGitHubGPU: Failed to convert GPUThreshold to integer")
		return err
	}

	if intMin <= 0 {
		log.Error().Int("GPUThreshold", intMin).Msg("ValidateAutoScalinGitHubGPU: Validation failed: GPUThreshold cannot be zero or negative")
		return errors.New("gpu threshold cannot be zero or negative")
	}

	return nil
}
