package api

import "github.com/gin-gonic/gin"

var (
	healthProvider = func(context *gin.Context) {
		context.JSON(200, gin.H{"message": "pong"})
	}
)
