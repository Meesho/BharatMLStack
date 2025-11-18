package externalcall

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
)

type FeatureValidationClient interface {
	ValidateOnlineFeatures(entity string, token string) (*OnlineValidationResponse, error)
	ValidateOfflineFeatures(offlineFeatures []string, token string) (*OfflineValidationResponse, error)
}

type featureValidationClientImpl struct {
	HTTPClient *http.Client
	BaseURL    string
}

var (
	Client                  FeatureValidationClient
	horizonBaseURL          string
	onlineFeatureMappingURL string
	featureGroupDataTypeURL string
)

// InitFeatureValidationClient initializes the feature validation client with config-based URLs
func InitFeatureValidationClient(config configs.Configs) {
	// Build horizon base URL from config
	if config.HorizonServer != "" {
		if config.HorizonPort != "" && config.HorizonPort != "80" {
			horizonBaseURL = fmt.Sprintf("http://%s:%s", config.HorizonServer, config.HorizonPort)
		} else {
			horizonBaseURL = fmt.Sprintf("http://%s", config.HorizonServer)
		}
	} else {
		panic("horizon server is not set")
	}

	onlineFeatureMappingURL = config.OnlineFeatureMappingUrl
	featureGroupDataTypeURL = config.FeatureGroupDataTypeMappingUrl

	// Initialize the client directly
	Client = &featureValidationClientImpl{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		BaseURL: horizonBaseURL,
	}
}

// OnlineValidationResponse represents the response from online feature validation
type OnlineValidationResponse []struct {
	EntityLabel       string `json:"entity-label"`
	FeatureGroupLabel string `json:"feature-group-label"`
	ID                int    `json:"id"`
	ActiveVersion     string `json:"active-version"`
	Features          map[string]struct {
		Labels interface{} `json:"labels"`
	} `json:"features"`
	StoreID                 string `json:"store-id"`
	DataType                string `json:"data-type"`
	TTLInSeconds            int    `json:"ttl-in-seconds"`
	TTLInEpoch              int    `json:"ttl-in-epoch"`
	JobID                   string `json:"job-id"`
	InMemoryCacheEnabled    bool   `json:"in-memory-cache-enabled"`
	DistributedCacheEnabled bool   `json:"distributed-cache-enabled"`
	LayoutVersion           int    `json:"layout-version"`
}

// OfflineValidationResponse represents the response from offline feature validation
type OfflineValidationResponse struct {
	Error string            `json:"error"`
	Data  map[string]string `json:"data"`
}

func (f *featureValidationClientImpl) ValidateOnlineFeatures(entity string, token string) (*OnlineValidationResponse, error) {
	// Use configured URL or fallback to default path
	var url string
	if featureGroupDataTypeURL != "" {
		url = fmt.Sprintf("%s/%s?entity=%s", f.BaseURL, featureGroupDataTypeURL, entity)
	} else {
		url = fmt.Sprintf("%s/api/v1/orion/retrieve-feature-groups?entity=%s", f.BaseURL, entity)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create online validation request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")

	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call online validation API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read online validation response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("online validation API failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response OnlineValidationResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal online validation response: %w", err)
	}

	return &response, nil
}

func (f *featureValidationClientImpl) ValidateOfflineFeatures(offlineFeatures []string, token string) (*OfflineValidationResponse, error) {
	// Use configured URL or fallback to default path
	var url string
	if onlineFeatureMappingURL != "" {
		url = fmt.Sprintf("%s/%s", f.BaseURL, onlineFeatureMappingURL)
	} else {
		url = fmt.Sprintf("%s/api/v1/orion/get-online-features-mapping", f.BaseURL)
	}

	requestBody := map[string][]string{
		"offline-feature-list": offlineFeatures,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal offline validation request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create offline validation request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call offline validation API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read offline validation response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("offline validation API failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response OfflineValidationResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal offline validation response: %w", err)
	}

	return &response, nil
}

// ParseFeatureString parses feature strings and determines validation requirements
// Returns featureType, entity, gf (entity:featureGroup), featureName, isValid
//
// Feature types and validation rules:
// 1. PARENT_OFFLINE_FEATURE|FEATURE -> validate as OFFLINE_FEATURE
// 2. OFFLINE_FEATURE|FEATURE -> validate as OFFLINE_FEATURE
// 3. ONLINE_FEATURE|FEATURE -> validate as ONLINE_FEATURE
// 4. PARENT_ONLINE_FEATURE|FEATURE -> validate as ONLINE_FEATURE
// 5. DEFAULT_FEATURE|FEATURE -> no validation needed
// 6. PARENT_DEFAULT_FEATURE|FEATURE -> no validation needed
// 7. MODEL_FEATURE|FEATURE -> no validation needed
// 8. CALIBRATION|FEATURE -> no validation needed
// 9. RTP_FEATURE|ENTITY:FEATURE_GROUP:FEATURE -> validate using pricing service
// 10. PARENT_RTP_FEATURE|ENTITY:FEATURE_GROUP:FEATURE -> validate using pricing service
func ParseFeatureString(feature string) (featureType, entity, gf, featureName string, isValid bool) {
	parts := strings.Split(feature, "|")
	if len(parts) < 2 {
		return "", "", "", "", false
	}

	featureType = strings.TrimSpace(parts[0])
	featureData := strings.TrimSpace(parts[1])

	switch featureType {
	case "PARENT_OFFLINE_FEATURE":
		// Add PARENT| prefix for validation but keep original feature name
		remainingParts := strings.Join(parts[1:], "|")
		validationFeature := "PARENT|" + remainingParts
		return featureType, "", validationFeature, remainingParts, true

	case "OFFLINE_FEATURE":
		// These need offline validation - return everything after the first pipe
		remainingParts := strings.Join(parts[1:], "|")
		return featureType, "", remainingParts, remainingParts, true

	case "ONLINE_FEATURE", "PARENT_ONLINE_FEATURE":
		// These need online validation - extract entity from e:fg:f format
		featureParts := strings.Split(featureData, ":")
		if len(featureParts) == 3 {
			entity := featureParts[0]                     // e
			gf := featureParts[1] + ":" + featureParts[2] // fg:f
			featureName := featureParts[2]                // f
			return featureType, entity, gf, featureName, true
		}
		return featureType, "", featureData, featureData, false

	case "DEFAULT_FEATURE", "PARENT_DEFAULT_FEATURE", "MODEL_FEATURE", "CALIBRATION":
		// These don't need API validation - they are correct by default
		// Return everything after the first pipe
		remainingParts := strings.Join(parts[1:], "|")
		return featureType, "", remainingParts, remainingParts, true

	case "RTP_FEATURE", "PARENT_RTP_FEATURE":
		// These need pricing service validation - extract entity from e:fg:f format
		featureData := strings.TrimSpace(parts[1])
		featureParts := strings.Split(featureData, ":")
		if len(featureParts) == 3 {
			entity := featureParts[0]                     // e
			gf := featureParts[1] + ":" + featureParts[2] // fg:f
			featureName := featureParts[2]                // f
			return featureType, entity, gf, featureName, true
		}
		return featureType, "", featureData, featureData, false

	default:
		// Unknown feature type
		return "", "", "", "", false
	}
}

// ValidateFeatureExists checks if a feature exists in the validation response
func ValidateFeatureExists(featureName string, response *OnlineValidationResponse) bool {
	if response == nil {
		return false
	}
	featureParts := strings.Split(featureName, ":")

	fg := featureParts[0]
	feature := featureParts[1]

	for _, featureGroup := range *response {
		if featureGroup.FeatureGroupLabel != fg {
			continue
		}

		// Iterate through all versions in features (e.g., "2", "3", etc.)
		for _, featureVersion := range featureGroup.Features {
			// Handle the labels field which can be a comma-separated string
			switch labels := featureVersion.Labels.(type) {
			case string:
				// Labels is a comma-separated string - split and check
				labelList := strings.Split(labels, ",")
				for _, label := range labelList {
					if strings.TrimSpace(label) == feature {
						return true
					}
				}
			case []string:
				// Labels is an array of strings
				for _, label := range labels {
					if strings.TrimSpace(label) == feature {
						return true
					}
				}
			case []interface{}:
				// Labels is an array of interfaces (convert to strings)
				for _, labelInterface := range labels {
					if labelStr, ok := labelInterface.(string); ok && strings.TrimSpace(labelStr) == feature {
						return true
					}
				}
			}
		}
	}
	return false
}
