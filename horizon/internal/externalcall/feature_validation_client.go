package externalcall

import (
	"fmt"
	"strings"

	ofshandler "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/handler"
)

type FeatureValidationClient interface {
	ValidateOnlineFeatures(entity string, token string) (*OnlineValidationResponse, error)
	ValidateOfflineFeatures(offlineFeatures []string, token string) (*OfflineValidationResponse, error)
}

type featureValidationClientImpl struct {
	ofsHandler ofshandler.Config
}

var (
	Client FeatureValidationClient
)

// InitFeatureValidationClient initializes the feature validation client with local OFS handler
func InitFeatureValidationClient() {
	// Initialize the client to use local online feature store handler
	Client = &featureValidationClientImpl{
		ofsHandler: ofshandler.InitV1ConfigHandler(),
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
	// Call local online feature store handler directly
	featureGroups, err := f.ofsHandler.RetrieveFeatureGroups(entity)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve feature groups: %w", err)
	}

	// Convert the OFS response to the expected format
	var response OnlineValidationResponse
	for _, fg := range *featureGroups {
		responseItem := struct {
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
		}{
			EntityLabel:             fg.EntityLabel,
			FeatureGroupLabel:       fg.FeatureGroupLabel,
			ID:                      fg.Id,
			ActiveVersion:           fg.ActiveVersion,
			StoreID:                 fg.StoreId,
			DataType:                string(fg.DataType),
			TTLInSeconds:            fg.TtlInSeconds,
			JobID:                   fg.JobId,
			InMemoryCacheEnabled:    fg.InMemoryCacheEnabled,
			DistributedCacheEnabled: fg.DistributedCacheEnabled,
			LayoutVersion:           fg.LayoutVersion,
		}

		// Convert features map
		responseItem.Features = make(map[string]struct {
			Labels interface{} `json:"labels"`
		})
		for version, feature := range fg.Features {
			responseItem.Features[version] = struct {
				Labels interface{} `json:"labels"`
			}{
				Labels: feature.Labels,
			}
		}

		response = append(response, responseItem)
	}

	return &response, nil
}

func (f *featureValidationClientImpl) ValidateOfflineFeatures(offlineFeatures []string, token string) (*OfflineValidationResponse, error) {
	// Call local online feature store handler directly
	request := ofshandler.GetOnlineFeatureMappingRequest{
		OfflineFeatureList: offlineFeatures,
	}

	mappingResponse, err := f.ofsHandler.GetOnlineFeatureMapping(request)
	if err != nil {
		return nil, fmt.Errorf("failed to get online feature mapping: %w", err)
	}

	// Convert the response to the expected format
	response := &OfflineValidationResponse{
		Error: mappingResponse.Error,
		Data:  mappingResponse.Data,
	}

	return response, nil
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
