//go:build meesho

package externalcall

import (
	"context"
	"fmt"
	"strings"

	pricingclient "github.com/Meesho/price-aggregator-go/pricingfeatureretrieval/client"
	"github.com/Meesho/price-aggregator-go/pricingfeatureretrieval/client/models"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PricingFeatureClient interface for pricing feature validation
type PricingFeatureClient interface {
	GetDataTypes(entity string) (*PricingDataTypesResponse, error)
	GetFeatureGroupDataTypeMap() (map[string]string, error)
	InitPricingClient()
}

const (
	RTPColonDelimiter      = ":"
	RTPUnderscoreDelimiter = "_"
)

// PricingDataTypesResponse represents the response from pricing feature service
type PricingDataTypesResponse struct {
	Entities []struct {
		Entity        string `json:"entity"`
		FeatureGroups []struct {
			Label    string   `json:"label"`
			Features []string `json:"features"`
			DataType string   `json:"dataType"`
		} `json:"featureGroups"`
	} `json:"entities"`
}

type pricingFeatureClientImpl struct{}

var PricingClient PricingFeatureClient = &pricingFeatureClientImpl{}

func (p *pricingFeatureClientImpl) InitPricingClient() {
	pricingclient.Init()
}

// GetDataTypes calls the pricing service to get data types for features
func (p *pricingFeatureClientImpl) GetDataTypes(entity string) (*PricingDataTypesResponse, error) {
	log.Info().Msgf("Calling pricing service for entity: %s", entity)

	// Create request for the pricing client (no entity parameter needed based on your requirement)
	clientInstance := pricingclient.Instance(1)

	// Call the real pricing client
	pricingResponse, err := clientInstance.GetDataTypes(context.Background(), &models.GetDataTypesRequest{}, map[string]string{})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to call pricing service for entity: %s", entity)

		// Check if it's a gRPC connection error and provide fallback
		if st, ok := status.FromError(err); ok && st.Code() == codes.Unavailable {
			log.Warn().Msgf("Pricing service unavailable for entity: %s, returning fallback response", entity)
			return p.getFallbackResponse(entity), nil
		}

		return nil, fmt.Errorf("failed to call pricing service: %w", err)
	}

	// Convert the pricing client response to our response format
	response := &PricingDataTypesResponse{
		Entities: make([]struct {
			Entity        string `json:"entity"`
			FeatureGroups []struct {
				Label    string   `json:"label"`
				Features []string `json:"features"`
				DataType string   `json:"dataType"`
			} `json:"featureGroups"`
		}, len(pricingResponse.Entities)),
	}

	// Map the response from pricing client to our format
	for i, pricingEntity := range pricingResponse.Entities {
		response.Entities[i].Entity = pricingEntity.Entity
		response.Entities[i].FeatureGroups = make([]struct {
			Label    string   `json:"label"`
			Features []string `json:"features"`
			DataType string   `json:"dataType"`
		}, len(pricingEntity.FeatureGroups))

		for j, pricingFeatureGroup := range pricingEntity.FeatureGroups {
			response.Entities[i].FeatureGroups[j].Label = pricingFeatureGroup.Label
			response.Entities[i].FeatureGroups[j].Features = pricingFeatureGroup.Features
			response.Entities[i].FeatureGroups[j].DataType = pricingFeatureGroup.DataType
		}
	}

	log.Info().Msgf("Successfully retrieved pricing data for entity %s with %d entities", entity, len(response.Entities))
	return response, nil
}

// GetFeatureGroupDataTypeMap fetches RTP feature group data types in transformed format
// This method transforms the pricing data into a map with composite keys: "entity:featuregroup:feature" -> dataType
func (p *pricingFeatureClientImpl) GetFeatureGroupDataTypeMap() (map[string]string, error) {
	rtpFeatureToDataType := make(map[string]string)
	clientInstance := pricingclient.Instance(1)
	rtpStruct, err := clientInstance.GetDataTypes(context.Background(), &models.GetDataTypesRequest{}, map[string]string{})
	if err != nil {
		return nil, err
	}
	for _, entity := range rtpStruct.Entities {
		for _, featureGroup := range entity.FeatureGroups {
			for _, feature := range featureGroup.Features {
				compositeName := entity.Entity + COLON_DELIMITER + featureGroup.Label + COLON_DELIMITER + feature
				rtpFeatureToDataType[compositeName] = featureGroup.DataType
			}
		}
	}
	return rtpFeatureToDataType, nil
}

// ValidatePricingFeatureExists checks if a pricing feature exists in the response
func ValidatePricingFeatureExists(featureName string, response *PricingDataTypesResponse) bool {
	if response == nil {
		return false
	}

	// Parse feature name: entity:feature_group:feature
	featureParts := strings.Split(featureName, ":")
	if len(featureParts) != 3 {
		return false
	}

	expectedEntity := featureParts[0]  // entity (e.g., "user_product")
	expectedFG := featureParts[1]      // feature group (e.g., "real_time_product_pricing")
	expectedFeature := featureParts[2] // feature name (e.g., "rto_charge")

	// Search through entities and feature groups
	for _, entity := range response.Entities {
		// Check if this is the correct entity
		if entity.Entity == expectedEntity {
			for _, featureGroup := range entity.FeatureGroups {
				// Check if this is the correct feature group
				if featureGroup.Label == expectedFG {
					// Check if feature exists in this feature group
					for _, f := range featureGroup.Features {
						if f == expectedFeature {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

// getFallbackResponse provides a fallback response when pricing service is unavailable
func (p *pricingFeatureClientImpl) getFallbackResponse(entity string) *PricingDataTypesResponse {
	return &PricingDataTypesResponse{
		Entities: []struct {
			Entity        string `json:"entity"`
			FeatureGroups []struct {
				Label    string   `json:"label"`
				Features []string `json:"features"`
				DataType string   `json:"dataType"`
			} `json:"featureGroups"`
		}{
			{
				Entity: entity,
				FeatureGroups: []struct {
					Label    string   `json:"label"`
					Features []string `json:"features"`
					DataType string   `json:"dataType"`
				}{
					{
						Label:    "fallback_feature_group",
						Features: []string{"fallback_feature"},
						DataType: "DataTypeString",
					},
				},
			},
		},
	}
}
