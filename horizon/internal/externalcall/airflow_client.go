package externalcall

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type AirflowClient interface {
	TriggerDAG(dagRunID string) (*AirflowResponse, error)
	ListDAGRuns(dagID string) (*AirflowDAGRunsResponse, error)
	GetDAGRun(dagID, dagRunID string) (*AirflowDAGRun, error)
}

type airflowClientImpl struct {
	BaseURL    string
	HTTPClient *http.Client
	Username   string
	Password   string
}

type AirflowDAGRunsResponse struct {
	DAGRuns      []AirflowDAGRun `json:"dag_runs"`
	TotalEntries int             `json:"total_entries"`
}

type AirflowDAGRun struct {
	DagID           string     `json:"dag_id"`
	DagRunID        string     `json:"dag_run_id"`
	State           string     `json:"state"`
	ExecutionDate   time.Time  `json:"execution_date"`
	StartDate       *time.Time `json:"start_date"`
	EndDate         *time.Time `json:"end_date"`
	ExternalTrigger bool       `json:"external_trigger"`
	RunType         string     `json:"run_type"`
}

var (
	airflowOnce     sync.Once
	airflowInstance AirflowClient
)

func InitAirflowClient(baseURL string, username string, password string) AirflowClient {
	airflowOnce.Do(func() {
		airflowInstance = &airflowClientImpl{
			BaseURL: baseURL,
			HTTPClient: &http.Client{
				Timeout: 30 * time.Second,
			},
			Username: username,
			Password: password,
		}
	})
	return airflowInstance
}

func GetAirflowClient() AirflowClient {
	return airflowInstance
}

type AirflowResponse struct {
	Status   string `json:"status"`
	DagRunID string `json:"dag_run_id"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
}

type AirflowTriggerRequest struct {
	DagRunID string `json:"dag_run_id"`
}

func (a *airflowClientImpl) TriggerDAG(dagRunID string) (*AirflowResponse, error) {
	url := fmt.Sprintf("%s/api/v1/dags/%s/dagRuns", a.BaseURL, dagRunID)

	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}

	timestamp := time.Now().Format(time.RFC3339)
	payload := AirflowTriggerRequest{
		DagRunID: dagRunID + "_" + timestamp,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set basic auth
	req.SetBasicAuth(a.Username, a.Password)

	log.Info().
		Str("url", url).
		Str("dag_run_id", dagRunID).
		Msg("Triggering Airflow DAG")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Error().
			Int("status_code", resp.StatusCode).
			Str("response_body", string(body)).
			Msg("Airflow DAG trigger failed")
		return &AirflowResponse{
			Status: "error",
			Error:  fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		}, nil
	}

	var response AirflowResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	log.Info().
		Str("dag_run_id", dagRunID).
		Str("status", response.Status).
		Msg("Airflow DAG triggered successfully")

	return &response, nil
}

func (a *airflowClientImpl) ListDAGRuns(dagID string) (*AirflowDAGRunsResponse, error) {
	url := fmt.Sprintf("%s/api/v1/dags/%s/dagRuns", a.BaseURL, dagID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(a.Username, a.Password)

	log.Info().
		Str("url", url).
		Str("dag_id", dagID).
		Msg("Listing Airflow DAG runs")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Error().
			Int("status_code", resp.StatusCode).
			Str("response_body", string(body)).
			Msg("Failed to list Airflow DAG runs")
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var response AirflowDAGRunsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

func (a *airflowClientImpl) GetDAGRun(dagID, dagRunID string) (*AirflowDAGRun, error) {
	url := fmt.Sprintf("%s/api/v1/dags/%s/dagRuns/%s", a.BaseURL, dagID, dagRunID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(a.Username, a.Password)

	log.Debug().
		Str("url", url).
		Str("dag_id", dagID).
		Str("dag_run_id", dagRunID).
		Msg("Getting Airflow DAG run status")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Error().
			Int("status_code", resp.StatusCode).
			Str("response_body", string(body)).
			Msg("Failed to get Airflow DAG run")
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var dagRun AirflowDAGRun
	if err := json.Unmarshal(body, &dagRun); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &dagRun, nil
}
