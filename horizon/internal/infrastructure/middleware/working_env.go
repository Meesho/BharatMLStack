package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SupportedInfraEnvs lists common infrastructure environments for reference
// Note: This list is maintained for backward compatibility and documentation
// The middleware now accepts any environment name to be environment-agnostic
// Users can onboard with any environment name as per their requirements
var SupportedInfraEnvs = []string{
	"prd",
	"gcp_prd",
	"int",
	"gcp_int",
	"dev",
	"gcp_dev",
	"stg",
	"gcp_stg",
	"ftr",
	"gcp_ftr",
}

// WorkingEnvMiddleware extracts and validates workingEnv from query string
// Stores it in the Gin context for use by handlers
// Accepts any non-empty workingEnv value to be environment-agnostic
// This allows users to onboard with any environment name as per their requirements
func WorkingEnvMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		workingEnv := ctx.Query("workingEnv")
		if workingEnv == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "workingEnv query parameter is required"})
			ctx.Abort()
			return
		}

		// Accept any non-empty workingEnv value
		// Environment names are subjective and should not be restricted
		// This makes the system environment-agnostic and flexible for different use cases
		
		// Store workingEnv in context for handlers to use
		ctx.Set("workingEnv", workingEnv)
		ctx.Next()
	}
}
