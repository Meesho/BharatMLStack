//go:build !meesho

package externalcall

import (
	"errors"
	"sync"

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
)

func InitRingmasterClient(RingmasterBaseUrl string, RingmasterMiscSession string, RingmasterAuthorization string, RingmasterEnvironment string, RingmasterApiKey string) {
	initRingmasterOnce.Do(func() {
		ringmasterBaseUrl = RingmasterBaseUrl
		ringmasterMiscSession = RingmasterMiscSession
		ringmasterAuthorization = RingmasterAuthorization
		ringmasterEnvironment = RingmasterEnvironment
		ringmasterApiKey = RingmasterApiKey
	})
}

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

type ringmasterClientImpl struct{}

var (
	clientInstance RingmasterClient
	ringmasterOnce sync.Once
)

func GetRingmasterClient() RingmasterClient {
	ringmasterOnce.Do(func() {
		clientInstance = &ringmasterClientImpl{}
	})
	return clientInstance
}

func (r *ringmasterClientImpl) GetConfig(serviceName, workflowID, runID string) Config {
	log.Warn().Msgf("ringmaster client GetConfig is not supported without meesho build tag")
	return Config{
		MinReplica:    "0",
		MaxReplica:    "0",
		RunningStatus: "false",
	}
}

func (r *ringmasterClientImpl) RestartDeployable(sd *servicedeployableconfig.ServiceDeployableConfig) error {
	log.Warn().Msgf("ringmaster client RestartDeployable is not supported without meesho build tag")
	return errors.New("ringmaster client RestartDeployable is not supported without meesho build tag")
}

func (r *ringmasterClientImpl) CreateDeployable(payload map[string]interface{}) ([]byte, error) {
	log.Warn().Msgf("ringmaster client CreateDeployable is not supported without meesho build tag")
	return nil, errors.New("ringmaster client CreateDeployable is not supported without meesho build tag")
}

func (r *ringmasterClientImpl) UpdateDeployable(name string, payload map[string]interface{}) error {
	log.Warn().Msgf("ringmaster client UpdateDeployable is not supported without meesho build tag")
	return errors.New("ringmaster client UpdateDeployable is not supported without meesho build tag")
}

func (r *ringmasterClientImpl) UpdateCPUThreshold(appName string, cpuThreshold string) error {
	log.Warn().Msgf("ringmaster client UpdateCPUThreshold is not supported without meesho build tag")
	return errors.New("ringmaster client UpdateCPUThreshold is not supported without meesho build tag")
}

func (r *ringmasterClientImpl) UpdateGPUThreshold(appName string, gpuThreshold string) error {
	log.Warn().Msgf("ringmaster client UpdateGPUThreshold is not supported without meesho build tag")
	return errors.New("ringmaster client UpdateGPUThreshold is not supported without meesho build tag")
}

func (r *ringmasterClientImpl) GetResourceDetail(serviceName string) (*ResourceDetail, error) {
	log.Warn().Msgf("ringmaster client GetResourceDetail is not supported without meesho build tag")
	return nil, errors.New("ringmaster client GetResourceDetail is not supported without meesho build tag")
}
