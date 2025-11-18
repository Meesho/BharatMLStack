//go:build !meesho

package externalcall

import (
	"errors"

	"github.com/rs/zerolog/log"
)

const (
	RTPClientVersion       = 1
	RTPColonDelimiter      = ":"
	RTPUnderscoreDelimiter = "_"
)

// RTPClient interface for RTP (Real-Time Pricing) feature operations
type RTPClient interface {
	Init()
	GetFeatureGroupDataTypeMap() (map[string]string, error)
}

type rtpClientImpl struct {
	version int
}

var rtpClientInstance RTPClient

// InitRTPClient initializes the RTP client (stub for non-Meesho)
func InitRTPClient() {
	log.Warn().Msg("RTP client is not supported without meesho build tag")
	rtpClientInstance = &rtpClientImpl{
		version: RTPClientVersion,
	}
}

// GetRTPClient returns the RTP client instance
func GetRTPClient() RTPClient {
	if rtpClientInstance == nil {
		InitRTPClient()
	}
	return rtpClientInstance
}

// Init is a no-op for stub implementation
func (r *rtpClientImpl) Init() {
	log.Warn().Msg("RTP client Init() called but not supported without meesho build tag")
}

// GetFeatureGroupDataTypeMap returns an error for stub implementation
func (r *rtpClientImpl) GetFeatureGroupDataTypeMap() (map[string]string, error) {
	log.Warn().Msg("RTP client GetFeatureGroupDataTypeMap() called but not supported without meesho build tag")
	return nil, errors.New("RTP client is not supported without meesho build tag")
}
