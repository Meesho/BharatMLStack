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

// PrismV2Client interface defines the operations for Prism V2 job management
type PrismV2Client interface {
	GetStepParameters(jobID int, stepID int) (*PrismV2StepResponse, error)
	UpdateStepParameters(jobID int, stepID int, modifications map[string]interface{}) error
	UpdateStepParametersAndTrigger(jobID int, stepID int, modifications map[string]interface{}) error
}

type prismV2ClientImpl struct {
	BaseURL    string
	HTTPClient *http.Client
	AppUserID  string
}

var (
	prismV2Once     sync.Once
	prismV2Instance PrismV2Client
)

// GetPrismV2Client returns a singleton instance of the Prism V2 client
func InitPrismV2Client(baseURL string, appUserID string) PrismV2Client {
	prismV2Once.Do(func() {
		prismV2Instance = &prismV2ClientImpl{
			BaseURL: baseURL,
			HTTPClient: &http.Client{
				Timeout: 30 * time.Second,
			},
			AppUserID: appUserID,
		}
	})
	log.Info().
		Str("base_url", baseURL).
		Str("app_user_id", appUserID).
		Msg("Prism V2 client initialized")
	return prismV2Instance
}

func GetPrismV2Client() PrismV2Client {
	return prismV2Instance
}

// PrismV2StepResponse represents the response from getting step information
type PrismV2StepResponse struct {
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// PrismV2UpdateRequest represents the request to update step parameters
type PrismV2UpdateRequest struct {
	Parameters  string `json:"parameters"`
	Description string `json:"description"`
}

// GetStepParameters retrieves the step parameters for a job
func (p *prismV2ClientImpl) GetStepParameters(jobID int, stepID int) (*PrismV2StepResponse, error) {
	url := fmt.Sprintf("%s/api/v2/jobs/%d/step/%d", p.BaseURL, jobID, stepID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("app-user-id", p.AppUserID)

	log.Info().
		Str("url", url).
		Int("job_id", jobID).
		Int("step_id", stepID).
		Msg("Getting Prism V2 step parameters")

	resp, err := p.HTTPClient.Do(req)
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
			Msg("Failed to get Prism V2 step parameters")
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var response PrismV2StepResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	log.Info().
		Int("job_id", jobID).
		Int("step_id", stepID).
		Msg("Successfully retrieved Prism V2 step parameters")

	return &response, nil
}

// UnmarshalJSON implements custom unmarshaling to handle parameters as either string or map
func (p *PrismV2StepResponse) UnmarshalJSON(data []byte) error {
	// First, unmarshal into a temporary struct that accepts parameters as interface{}
	var temp struct {
		Description string      `json:"description"`
		Parameters  interface{} `json:"parameters"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	p.Description = temp.Description

	// Handle parameters - it might be a string or a map
	switch v := temp.Parameters.(type) {
	case string:
		// Parameters is a JSON string, need to unmarshal it
		if err := json.Unmarshal([]byte(v), &p.Parameters); err != nil {
			return fmt.Errorf("failed to unmarshal parameters string to map: %w", err)
		}
	case map[string]interface{}:
		// Parameters is already a map
		p.Parameters = v
	case nil:
		// Parameters is null
		p.Parameters = make(map[string]interface{})
	default:
		// Try to marshal and unmarshal as a workaround for other types
		bytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal parameters: %w", err)
		}
		if err := json.Unmarshal(bytes, &p.Parameters); err != nil {
			return fmt.Errorf("failed to unmarshal parameters to map: %w", err)
		}
	}

	return nil
}

// UpdateStepParameters updates the step parameters for a job
func (p *prismV2ClientImpl) UpdateStepParameters(jobID int, stepID int, modifications map[string]interface{}) error {
	// First, get the current step parameters
	stepResponse, err := p.GetStepParameters(jobID, stepID)
	if err != nil {
		return fmt.Errorf("failed to get step parameters: %w", err)
	}

	// Extract the actual params from the nested structure
	// The parameters map contains a "params" key with a JSON string value
	var actualParams map[string]interface{}

	if stepResponse.Parameters != nil {
		if paramsValue, exists := stepResponse.Parameters["params"]; exists {
			// paramsValue is a JSON string, need to unmarshal it
			if paramsStr, ok := paramsValue.(string); ok {
				if err := json.Unmarshal([]byte(paramsStr), &actualParams); err != nil {
					log.Warn().Err(err).Msg("Failed to unmarshal existing params, creating new params map")
					actualParams = make(map[string]interface{})
				}
			} else {
				// If params is not a string, try to use it directly as a map
				if paramsMap, ok := paramsValue.(map[string]interface{}); ok {
					actualParams = paramsMap
				} else {
					actualParams = make(map[string]interface{})
				}
			}
		} else {
			// No "params" key, check if parameters itself is the params map
			actualParams = stepResponse.Parameters
		}
	}

	if actualParams == nil {
		actualParams = make(map[string]interface{})
	}

	// Merge modifications with existing params
	for k, v := range modifications {
		actualParams[k] = v
	}

	log.Debug().
		Interface("actual_params", actualParams).
		Interface("modifications", modifications).
		Msg("Merged parameters with modifications")

	// Marshal the actual params back to a JSON string
	paramsJSON, err := json.Marshal(actualParams)
	if err != nil {
		return fmt.Errorf("failed to marshal actual params: %w", err)
	}

	// Create the nested structure: parameters contains "params" as a JSON string
	parametersWrapper := map[string]interface{}{
		"params": string(paramsJSON),
	}

	// Serialize the wrapper to JSON string for the update request
	parametersJSON, err := json.Marshal(parametersWrapper)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters wrapper: %w", err)
	}

	// Prepare description (use existing or default)
	description := stepResponse.Description
	if description == "" {
		description = "Updated step parameters"
	}

	// Create update request
	updateRequest := PrismV2UpdateRequest{
		Parameters:  string(parametersJSON),
		Description: description,
	}

	url := fmt.Sprintf("%s/api/v2/jobs/%d/step/%d", p.BaseURL, jobID, stepID)

	jsonData, err := json.Marshal(updateRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal update request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("app-user-id", p.AppUserID)

	log.Info().
		Str("url", url).
		Int("job_id", jobID).
		Int("step_id", stepID).
		Msg("Updating Prism V2 step parameters")

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Error().
			Int("status_code", resp.StatusCode).
			Str("response_body", string(body)).
			Msg("Failed to update Prism V2 step parameters")
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	log.Info().
		Int("job_id", jobID).
		Int("step_id", stepID).
		Msg("Successfully updated Prism V2 step parameters")

	return nil
}

// UpdateStepParametersAndTrigger updates step parameters and then triggers the Airflow job
func (p *prismV2ClientImpl) UpdateStepParametersAndTrigger(jobID int, stepID int, modifications map[string]interface{}) error {
	// First, update the step parameters
	if err := p.UpdateStepParameters(jobID, stepID, modifications); err != nil {
		return fmt.Errorf("failed to update step parameters: %w", err)
	}

	// Then, trigger the Airflow DAG
	airflowClient := GetAirflowClient()
	dagRunID := fmt.Sprintf("prism_job_%d_step_%d_%d", jobID, stepID, time.Now().Unix())

	airflowResponse, err := airflowClient.TriggerDAG(dagRunID)
	if err != nil {
		return fmt.Errorf("failed to trigger Airflow DAG: %w", err)
	}

	if airflowResponse.Status == "error" {
		return fmt.Errorf("airflow DAG trigger failed: %s", airflowResponse.Error)
	}

	log.Info().
		Int("job_id", jobID).
		Int("step_id", stepID).
		Str("dag_run_id", dagRunID).
		Msg("Successfully updated parameters and triggered Airflow DAG")

	return nil
}
