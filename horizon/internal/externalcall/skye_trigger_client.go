package externalcall

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SkyeTriggerDAGRunID is the synthetic dag_run_id stored when using skye-trigger instead of Airflow.
// checkAirflowDAGRunStatus returns success when it sees this ID without calling Airflow.
const SkyeTriggerDAGRunID = "skye-trigger"

// TriggerSkyeTrigger calls the skye-trigger service POST /trigger with entity, model, variants.
// Used when UseSkyeTriggerInsteadOfAirflow is true (OSS quick-start).
func TriggerSkyeTrigger(baseURL, entity, model string, variants []string, environment string) error {
	if baseURL == "" {
		return fmt.Errorf("skye_trigger_url is required")
	}
	url := baseURL + "/trigger"
	if len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
		url = baseURL + "trigger"
	}
	body := map[string]interface{}{
		"entity":      entity,
		"model_name":  model,
		"variants":    variants,
		"environment": environment,
	}
	if environment == "" {
		body["environment"] = "local"
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal skye-trigger request: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create skye-trigger request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("skye-trigger request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("skye-trigger returned status %d", resp.StatusCode)
	}
	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if !result.OK && result.Error != "" {
		return fmt.Errorf("skye-trigger: %s", result.Error)
	}
	return nil
}
