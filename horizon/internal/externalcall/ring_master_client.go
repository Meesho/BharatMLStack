// Package externalcall provides deprecated RingMaster client functionality.
// Deprecated: This package is deprecated. Use the infrastructure handler and GitHub package instead.
// The RingMaster client is kept for backward compatibility during migration.
package externalcall

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
)

var (
	initRingmasterOnce      sync.Once
	ringmasterBaseUrl       string
	ringmasterMiscSession   string
	ringmasterAuthorization string
	ringmasterEnvironment   string
	ringmasterApiKey        string

	// Working environment - generalized variable (not RingMaster specific)
	workingEnvironment string
	initWorkingEnvOnce sync.Once
)

// InitRingmasterClient initializes the deprecated RingMaster client.
// Deprecated: This function is deprecated. Use infrastructure handler initialization instead.
func InitRingmasterClient(RingmasterBaseUrl string, RingmasterMiscSession string, RingmasterAuthorization string, RingmasterEnvironment string, RingmasterApiKey string) {
	initRingmasterOnce.Do(func() {
		ringmasterBaseUrl = RingmasterBaseUrl
		ringmasterMiscSession = RingmasterMiscSession
		ringmasterAuthorization = RingmasterAuthorization
		ringmasterEnvironment = RingmasterEnvironment
		ringmasterApiKey = RingmasterApiKey
	})
}

// RingmasterClient is a deprecated interface for interacting with RingMaster APIs.
// Deprecated: Use infrastructure handler and GitHub package instead.
// This interface is kept for backward compatibility during migration.
type RingmasterClient interface {
	GetConfig(serviceName, workflowID, runID string) Config
	RestartDeployable(sd *servicedeployableconfig.ServiceDeployableConfig) error
	CreateDeployable(payload map[string]interface{}) ([]byte, error)
	UpdateDeployable(name string, payload map[string]interface{}) error
	UpdateCPUThreshold(appName string, cpuThreshold string) error
	UpdateGPUThreshold(appName string, gpuThreshold string) error
	GetResourceDetail(serviceName string) (*ResourceDetail, error)
}

type Config struct {
	MinReplica    string `json:"min_replica"`
	MaxReplica    string `json:"max_replica"`
	RunningStatus string `json:"running_status"`
}

type Activity struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type ResourceDetail struct {
	Nodes []Node `json:"nodes"`
}

var respBody struct {
	Min     int `json:"min"`
	Max     int `json:"max"`
	Desired int `json:"desired"`
	Current int `json:"current"`
}

type DeployableConfig struct {
	DeploymentStrategy string `json:"deploymentStrategy"`
}

type Node struct {
	Kind   string     `json:"kind"`
	Name   string     `json:"name"`
	Health Health     `json:"health"`
	Info   []InfoItem `json:"info"`
}

type Health struct {
	Status string `json:"status"`
}

type InfoItem struct {
	Value string `json:"value"`
}

type WorkflowResultRequest struct {
	ApplicationName string `json:"applicationName"`
	WorkflowID      string `json:"workflowId"`
	RunID           string `json:"runId"`
}

type ringmasterClientImpl struct {
	BaseURL       string
	HTTPClient    *http.Client
	MiscSession   string
	Authorization string
	Environment   string
}

var (
	clientInstance RingmasterClient
	ringmasterOnce sync.Once
)

// Helper function to read raw response
func readRawResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// GetRingmasterClient returns the deprecated RingMaster client instance.
// Deprecated: Use infrastructure handler instead.
func GetRingmasterClient() RingmasterClient {
	ringmasterOnce.Do(func() {
		clientInstance = &ringmasterClientImpl{
			BaseURL: ringmasterBaseUrl,
			HTTPClient: &http.Client{
				Timeout: 200 * time.Second,
			},
			MiscSession:   ringmasterMiscSession,
			Authorization: ringmasterAuthorization,
			Environment:   ringmasterEnvironment,
		}
	})
	return clientInstance
}

// InitWorkingEnvironment initializes the working environment from config
// Uses WorkingEnv if set, otherwise falls back to RingmasterEnvironment for backward compatibility
func InitWorkingEnvironment(workingEnv, ringmasterEnv string) {
	initWorkingEnvOnce.Do(func() {
		if workingEnv != "" {
			workingEnvironment = workingEnv
		} else {
			// Fallback to RingmasterEnvironment for backward compatibility
			workingEnvironment = ringmasterEnv
		}
	})
}

// GetWorkingEnvironment returns the working environment (generalized, not RingMaster specific)
func GetWorkingEnvironment() string {
	return workingEnvironment
}

// GetRingmasterEnvironment returns the working environment (deprecated - use GetWorkingEnvironment instead)
// Kept for backward compatibility
func GetRingmasterEnvironment() string {
	return GetWorkingEnvironment()
}

func (r *ringmasterClientImpl) GetConfig(serviceName, workflowID, runID string) Config {
	parts := strings.Split(r.Environment, "_")
	url := fmt.Sprintf("%s/api/v1/mlp/application/resource/%s-%s/HorizontalPodAutoscaler?workingEnv=%s",
		r.BaseURL, parts[1], serviceName, r.Environment)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Error().Msgf("failed to create request: %s", err)
		return Config{
			MinReplica:    "0",
			MaxReplica:    "0",
			RunningStatus: "false",
		}
	}

	r.setCommonHeaders(req)

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		log.Error().Msgf("failed to call ringmaster GetConfig: %s", err)
		return Config{
			MinReplica:    "0",
			MaxReplica:    "0",
			RunningStatus: "false",
		}
	}

	rawBody, err := readRawResponse(resp)
	if err != nil {
		log.Error().Msgf("failed to read response body: %s", err)
		return Config{
			MinReplica:    "0",
			MaxReplica:    "0",
			RunningStatus: "false",
		}
	}

	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("ringmaster GetConfig failed, status: %d, body: %s", resp.StatusCode, string(rawBody))
		return Config{
			MinReplica:    "0",
			MaxReplica:    "0",
			RunningStatus: "false",
		}
	}

	if err := json.Unmarshal(rawBody, &respBody); err != nil {
		log.Error().Msgf("failed to decode response body: %s, raw response: %s", err, string(rawBody))
		return Config{
			MinReplica:    "0",
			MaxReplica:    "0",
			RunningStatus: "false",
		}
	}

	// Get workflow result to determine running status
	result, err := r.GetResourceDetail(serviceName)
	if err != nil {
		log.Error().Msgf("failed to get resource detail: %s", err)
		return Config{
			MinReplica:    strconv.Itoa(respBody.Min),
			MaxReplica:    strconv.Itoa(respBody.Max),
			RunningStatus: "false",
		}
	}
	healthyPodCount := 0
	for _, node := range result.Nodes {
		if node.Kind == "Pod" && node.Health.Status == "Healthy" {
			for _, info := range node.Info {
				if info.Value == "Running" {
					healthyPodCount += 1
				}
			}
		}
	}

	if healthyPodCount == 0 {
		log.Error().Msgf("No healthy pods found for service: %s", serviceName)
		return Config{
			MinReplica:    strconv.Itoa(respBody.Min),
			MaxReplica:    strconv.Itoa(respBody.Max),
			RunningStatus: "false",
		}
	}

	return Config{
		MinReplica:    strconv.Itoa(respBody.Min),
		MaxReplica:    strconv.Itoa(respBody.Max),
		RunningStatus: "true",
	}
}

func (r *ringmasterClientImpl) RestartDeployable(sd *servicedeployableconfig.ServiceDeployableConfig) error {
	var deployableConfig DeployableConfig

	if err := json.Unmarshal(sd.Config, &deployableConfig); err != nil {
		return fmt.Errorf("failed to unmarshal deployable config: %w", err)
	}

	parts := strings.Split(r.Environment, "_")
	if len(parts) < 2 {
		return fmt.Errorf("invalid environment format: %s", r.Environment)
	}

	url := fmt.Sprintf("%s/api/v1/mlp/application/%s-%s/resource/deployment/restart?workingEnv=%s",
		r.BaseURL, parts[1], sd.Name, r.Environment)

	// Always send isCanary: "true" or "false"
	isCanary := "false"
	if deployableConfig.DeploymentStrategy == "canary" {
		isCanary = "true"
	}

	body, err := json.Marshal(map[string]string{
		"isCanary": isCanary,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	r.setCommonHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call ringmaster RestartDeployable: %w", err)
	}

	rawBody, err := readRawResponse(resp)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ringmaster RestartDeployable failed, status: %d, body: %s", resp.StatusCode, string(rawBody))
	}

	return nil
}

func (r *ringmasterClientImpl) setCommonHeaders(req *http.Request) {
	req.Header.Set("misc_session", r.MiscSession)
	req.Header.Set("Authorization", r.Authorization)
}

func (r *ringmasterClientImpl) CreateDeployable(payload map[string]interface{}) ([]byte, error) {
	url := fmt.Sprintf("%s/api/v1/mlp/gpu/cicd/onboarding?workingEnv=%s", r.BaseURL, r.Environment)

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	r.setCommonHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", ringmasterApiKey)

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ringmaster CreateDeployable request failed: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ringmaster CreateDeployable failed, status: %d, body: %s", resp.StatusCode, string(rawBody))
	}

	return rawBody, nil
}

func (r *ringmasterClientImpl) UpdateDeployable(name string, payload map[string]interface{}) error {
	parts := strings.Split(r.Environment, "_")
	var url string

	if _, ok := payload["cpuThreshold"]; ok {
		url = fmt.Sprintf("%s/api/v1/mlp/application/%s-%s/resource/hpa/cpu?workingEnv=%s",
			r.BaseURL, parts[1], name, r.Environment)
	} else if _, ok := payload["gpuThreshold"]; ok {
		url = fmt.Sprintf("%s/api/v1/mlp/application/%s-%s/resource/hpa/gpu?workingEnv=%s",
			r.BaseURL, parts[1], name, r.Environment)
	} else {
		return fmt.Errorf("invalid threshold update payload")
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal update deployable payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}

	r.setCommonHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call ringmaster UpdateDeployable: %w", err)
	}

	rawBody, err := readRawResponse(resp)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ringmaster UpdateDeployable failed, status: %d, body: %s", resp.StatusCode, string(rawBody))
	}

	return nil
}

func (r *ringmasterClientImpl) UpdateCPUThreshold(appName string, cpuThreshold string) error {
	parts := strings.Split(r.Environment, "_")
	url := fmt.Sprintf("%s/api/v1/mlp/application/%s-%s/resource/hpa/cpu?workingEnv=%s",
		r.BaseURL, parts[1], appName, r.Environment)

	payload := map[string]string{
		"cpuThreshold": cpuThreshold,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal CPU threshold payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	r.setCommonHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update CPU threshold: %w", err)
	}

	rawBody, err := readRawResponse(resp)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update CPU threshold failed, status: %d, body: %s", resp.StatusCode, string(rawBody))
	}

	return nil
}

func (r *ringmasterClientImpl) UpdateGPUThreshold(appName string, gpuThreshold string) error {
	parts := strings.Split(r.Environment, "_")
	url := fmt.Sprintf("%s/api/v1/mlp/application/%s-%s/resource/hpa/gpu?workingEnv=%s",
		r.BaseURL, parts[1], appName, r.Environment)

	payload := map[string]string{
		"gpuThreshold": gpuThreshold,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal GPU threshold payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	r.setCommonHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update GPU threshold: %w", err)
	}

	rawBody, err := readRawResponse(resp)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update GPU threshold failed, status: %d, body: %s", resp.StatusCode, string(rawBody))
	}

	return nil
}

func (r *ringmasterClientImpl) GetResourceDetail(serviceName string) (*ResourceDetail, error) {
	env := r.Environment
	parts := strings.Split(env, "_")
	var servicePrefix string
	if len(parts) == 2 {
		servicePrefix = parts[1] + "-"
	}

	url := fmt.Sprintf("%s/api/v1/mlp/application/resource-detail?appName=%s&workingEnv=%s",
		r.BaseURL, servicePrefix+serviceName, r.Environment)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// Set headers
	r.setCommonHeaders(req)

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call ringmaster GetResourceDetail: %w", err)
	}

	defer func() {
		_ = resp.Body.Close() // safe deferred close
	}()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ringmaster GetResourceDetail failed, status: %d, body: %s", resp.StatusCode, string(rawBody))
	}

	var detail ResourceDetail
	if err := json.Unmarshal(rawBody, &detail); err != nil {
		return nil, fmt.Errorf("failed to decode resource detail: %w\nraw response:\n%s", err, string(rawBody))
	}

	return &detail, nil
}
