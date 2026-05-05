package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/Meesho/BharatMLStack/skye/pkg/api/http"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
)

// HTTPLogger logs the request
func HTTPLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next()
		endTime := time.Now()

		latency := endTime.Sub(startTime)

		// Example:
		//   r.GET("/users/:id", func(c *gin.Context) {
		//       // Request: GET /users/42
		//       route := c.FullPath() // "/users/:id"
		//   })
		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}

		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		userContext := c.Request.Header.Get(http.HeaderMeeshoUserContext)

		metricTags := metric.BuildTag(
			metric.NewTag(metric.TagPath, route),
			metric.NewTag(metric.TagMethod, method),
			metric.NewTag(metric.TagHttpStatusCode, strconv.Itoa(statusCode)),
			metric.NewTag(metric.TagUserContext, userContext),
		)
		metric.Incr(metric.ApiRequestCount, metricTags)
		metric.Timing(metric.ApiRequestLatency, latency, metricTags)
		log.Info().Msgf("[access] [%s] %s %s %d %v", clientIP, method, route, statusCode, latency)
	}
}
