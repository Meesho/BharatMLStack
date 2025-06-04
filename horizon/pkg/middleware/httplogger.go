package middleware

import (
	"strconv"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/api/http"
	"github.com/Meesho/BharatMLStack/horizon/pkg/metric"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// HTTPLogger logs the request
func HTTPLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next()
		endTime := time.Now()

		latency := endTime.Sub(startTime)
		path := c.Request.URL.Path
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		userContext := c.Request.Header.Get(http.HeaderMeeshoUserContext)

		metricTags := metric.BuildTag(
			metric.NewTag(metric.TagPath, path),
			metric.NewTag(metric.TagMethod, method),
			metric.NewTag(metric.TagHttpStatusCode, strconv.Itoa(statusCode)),
			metric.NewTag(metric.TagUserContext, userContext),
		)
		metric.Incr(metric.ApiRequestCount, metricTags)
		metric.Timing(metric.ApiRequestLatency, latency, metricTags)
		log.Info().Msgf("[access] [%s] %s %s %d %v", clientIP, method, path, statusCode, latency)
	}
}
