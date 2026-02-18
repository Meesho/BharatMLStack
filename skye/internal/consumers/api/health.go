package api

import "github.com/gin-gonic/gin"

var (
	healthProvider = func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Application is up!!!"})
	}
)
