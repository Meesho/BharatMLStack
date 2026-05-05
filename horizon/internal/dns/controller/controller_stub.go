//go:build !meesho

package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Controller struct{}

func NewController() *Controller {
	return &Controller{}
}

// CreateCloudDNS is a stub implementation for open-source builds
func (c *Controller) CreateCloudDNS(ctx *gin.Context) {
	log.Warn().Msg("DNS endpoints are not available in open-source builds. Provide organization-specific implementations to enable DNS functionality.")
	ctx.JSON(http.StatusNotImplemented, gin.H{
		"status":  "fail",
		"message": "DNS endpoints are not available in open-source builds. Provide organization-specific implementations to enable DNS functionality.",
	})
}

// CreateCoreDNS is a stub implementation for open-source builds
func (c *Controller) CreateCoreDNS(ctx *gin.Context) {
	log.Warn().Msg("DNS endpoints are not available in open-source builds. Provide organization-specific implementations to enable DNS functionality.")
	ctx.JSON(http.StatusNotImplemented, gin.H{
		"status":  "fail",
		"message": "DNS endpoints are not available in open-source builds. Provide organization-specific implementations to enable DNS functionality.",
	})
}
