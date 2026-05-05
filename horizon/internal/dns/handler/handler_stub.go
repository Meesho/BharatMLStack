//go:build !meesho

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type DNSHandler interface {
	CreateCloudDNS(ctx *gin.Context)
	CreateCoreDNS(ctx *gin.Context)
}

type dnsHandlerStub struct{}

func NewDNSHandler() DNSHandler {
	return &dnsHandlerStub{}
}

func (h *dnsHandlerStub) CreateCloudDNS(ctx *gin.Context) {
	log.Warn().Msg("DNS endpoints are not available in open-source builds. Provide organization-specific implementations to enable DNS functionality.")
	ctx.JSON(http.StatusNotImplemented, gin.H{
		"status":  "fail",
		"message": "DNS endpoints are not available in open-source builds. Provide organization-specific implementations to enable DNS functionality.",
	})
}

func (h *dnsHandlerStub) CreateCoreDNS(ctx *gin.Context) {
	log.Warn().Msg("DNS endpoints are not available in open-source builds. Provide organization-specific implementations to enable DNS functionality.")
	ctx.JSON(http.StatusNotImplemented, gin.H{
		"status":  "fail",
		"message": "DNS endpoints are not available in open-source builds. Provide organization-specific implementations to enable DNS functionality.",
	})
}
