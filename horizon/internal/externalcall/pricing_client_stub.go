//go:build !meesho

package externalcall

import (
	"errors"

	"github.com/rs/zerolog/log"
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

// GetDataTypes calls the pricing service to get data types for features
func (p *pricingFeatureClientImpl) GetDataTypes(entity string) (*PricingDataTypesResponse, error) {
	return nil, errors.New("pricing client GetDataTypes is not supported in open-source builds. Provide organization-specific implementations to enable pricing client functionality")
}

// GetFeatureGroupDataTypeMap returns an error for stub implementation
func (p *pricingFeatureClientImpl) GetFeatureGroupDataTypeMap() (map[string]string, error) {
	log.Warn().Msg("pricing client GetFeatureGroupDataTypeMap is not supported in open-source builds. Provide organization-specific implementations to enable pricing client functionality")
	return nil, errors.New("pricing client GetFeatureGroupDataTypeMap is not supported in open-source builds. Provide organization-specific implementations to enable pricing client functionality")
}

// ValidatePricingFeatureExists checks if a pricing feature exists in the response
func ValidatePricingFeatureExists(featureName string, response *PricingDataTypesResponse) bool {
	log.Warn().Msgf("pricing client ValidatePricingFeatureExists is not supported in open-source builds. Provide organization-specific implementations to enable pricing client functionality")
	return false
}

func (p *pricingFeatureClientImpl) InitPricingClient() {
	log.Warn().Msgf("pricing client Init is not supported in open-source builds. Provide organization-specific implementations to enable pricing client functionality")
}
