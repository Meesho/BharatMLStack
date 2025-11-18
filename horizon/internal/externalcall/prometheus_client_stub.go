//go:build !meesho

package externalcall

import (
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"
)

type PrometheusClient interface {
	GetModelNames(serviceName string) ([]string, error)
	GetInferflowConfigNames(serviceName string) ([]string, error)
	GetNumerixConfigNames() ([]string, error)
}

type prometheusClientImpl struct {
	BaseURL    string
	HTTPClient *http.Client
	APIKey     string
}

func InitPrometheusClient(VmselectStartDaysAgo int, VmselectBaseUrl string, VmselectApiKey string) {
	log.Warn().Msg("Prometheus client is not supported without meesho build tag")
}

func GetPrometheusClient() PrometheusClient {
	log.Warn().Msg("Prometheus client is not supported without meesho build tag")
	return nil
}

func (p *prometheusClientImpl) GetModelNames(serviceName string) ([]string, error) {
	return nil, errors.New("Prometheus client is not supported without meesho build tag")
}

func (p *prometheusClientImpl) GetInferflowConfigNames(serviceName string) ([]string, error) {
	return nil, errors.New("Prometheus client is not supported without meesho build tag")
}

func (p *prometheusClientImpl) GetNumerixConfigNames() ([]string, error) {
	return nil, errors.New("Prometheus client is not supported without meesho build tag")
}
