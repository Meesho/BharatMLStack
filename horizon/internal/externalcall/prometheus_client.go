package externalcall

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
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

var (
	prometheusOnce       sync.Once
	prometheusInstance   PrometheusClient
	vmselectStartDaysAgo int
	vmselectBaseUrl      string
	vmselectApiKey       string
	initPrometheusOnce   sync.Once
)

func InitPrometheusClient(VmselectStartDaysAgo int, VmselectBaseUrl string, VmselectApiKey string) {
	initPrometheusOnce.Do(func() {
		vmselectStartDaysAgo = VmselectStartDaysAgo
		vmselectBaseUrl = VmselectBaseUrl
		vmselectApiKey = VmselectApiKey
	})
}

func GetPrometheusClient() PrometheusClient {
	prometheusOnce.Do(func() {
		prometheusInstance = &prometheusClientImpl{
			BaseURL: vmselectBaseUrl,
			HTTPClient: &http.Client{
				Timeout: 10 * time.Second,
			},
			APIKey: vmselectApiKey,
		}
	})
	return prometheusInstance
}

type prometheusResponse struct {
	Status    string `json:"status"`
	IsPartial bool   `json:"isPartial"`
	Data      struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				ModelName string `json:"model_name"`
			} `json:"metric"`
		} `json:"result"`
	} `json:"data"`
}

type prometheusInferflowConfigResponse struct {
	Status    string `json:"status"`
	IsPartial bool   `json:"isPartial"`
	Data      struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				ModelID string `json:"model_id"`
			} `json:"metric"`
		} `json:"result"`
	} `json:"data"`
}

type prometheusNumerixConfigResponse struct {
	Status    string `json:"status"`
	IsPartial bool   `json:"isPartial"`
	Data      struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				ComputeID string `json:"compute_id"`
			} `json:"metric"`
		} `json:"result"`
	} `json:"data"`
}

func (p *prometheusClientImpl) GetModelNames(serviceName string) ([]string, error) {
	end := time.Now().Unix()
	daysAgo := vmselectStartDaysAgo
	start := end - int64(daysAgo*24*60*60)
	step := "1h40m"

	query := fmt.Sprintf(
		"sum by (model_name)(rate(modelinferenceorchestrator_retrievemodelinferencescore_request_total_value{service=\"%s\"}[1m]))",
		serviceName,
	)

	url := fmt.Sprintf("%s/select/100/prometheus/api/v1/query_range?query=%s&start=%d&end=%d&step=%s",
		p.BaseURL,
		escapePrometheusQuery(query),
		start,
		end,
		step,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.APIKey)

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Prometheus: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prometheus call failed, status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var pr prometheusResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("failed to decode Prometheus response: %w", err)
	}

	var modelNames []string
	for _, item := range pr.Data.Result {
		modelNames = append(modelNames, item.Metric.ModelName)
	}

	return modelNames, nil
}

func (p *prometheusClientImpl) GetInferflowConfigNames(serviceName string) ([]string, error) {
	end := time.Now().Unix()
	daysAgo := vmselectStartDaysAgo
	start := end - int64(daysAgo*24*60*60)
	step := "1h40m"

	query := fmt.Sprintf(
		"sum by (model_id)(rate(inferflow_retrievemodelscore_request_total_value{service=\"%s\"}[1m]))",
		serviceName,
	)

	url := fmt.Sprintf("%s/select/100/prometheus/api/v1/query_range?query=%s&start=%d&end=%d&step=%s",
		p.BaseURL,
		escapePrometheusQuery(query),
		start,
		end,
		step,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.APIKey)

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Prometheus: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prometheus call failed, status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var pr prometheusInferflowConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("failed to decode Prometheus response: %w", err)
	}

	var modelNames []string
	for _, item := range pr.Data.Result {
		modelNames = append(modelNames, item.Metric.ModelID)
	}

	return modelNames, nil
}

func (p *prometheusClientImpl) GetNumerixConfigNames() ([]string, error) {
	end := time.Now().Unix()
	daysAgo := vmselectStartDaysAgo
	start := end - int64(daysAgo*24*60*60)
	step := "1h40m"

	query := "sum by (compute_id)(rate(numerix_numerix_computation_request_total_value[1m]))"

	url := fmt.Sprintf("%s/select/100/prometheus/api/v1/query_range?query=%s&start=%d&end=%d&step=%s",
		p.BaseURL,
		escapePrometheusQuery(query),
		start,
		end,
		step,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.APIKey)

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Prometheus: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prometheus call failed, status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var pr prometheusNumerixConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("failed to decode Prometheus response: %w", err)
	}

	var computeIDs []string
	for _, item := range pr.Data.Result {
		computeIDs = append(computeIDs, item.Metric.ComputeID)
	}

	return computeIDs, nil
}

func escapePrometheusQuery(query string) string {
	// Spaces, quotes, braces and brackets are URL-encoded
	return url.QueryEscape(query)
}
