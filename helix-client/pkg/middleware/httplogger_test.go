package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	http2 "github.com/Meesho/BharatMLStack/helix-client/pkg/api/http"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestHTTPLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logBuffer bytes.Buffer
	log.Logger = zerolog.New(&logBuffer).With().Timestamp().Logger()

	t.Run("logs successful request with all parameters", func(t *testing.T) {
		logBuffer.Reset()
		router := gin.New()
		router.Use(HTTPLogger())
		router.GET("/users/:id", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/users/123", nil)
		req.Header.Set(http2.HeaderUserContext, "anonymous")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, "[access]")
		assert.Contains(t, logOutput, "GET")
		assert.Contains(t, logOutput, "/users/:id")
		assert.Contains(t, logOutput, "200")
		assert.Contains(t, logOutput, "192.0.2.1")
	})

	t.Run("handles error response", func(t *testing.T) {
		logBuffer.Reset()
		router := gin.New()
		router.Use(HTTPLogger())
		router.GET("/error", func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		})

		req := httptest.NewRequest("GET", "/error", nil)
		req.Header.Set(http2.HeaderUserContext, "anonymous")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, "[access]")
		assert.Contains(t, logOutput, "GET")
		assert.Contains(t, logOutput, "/error")
		assert.Contains(t, logOutput, "500")
	})

	t.Run("handles unknown route", func(t *testing.T) {
		logBuffer.Reset()
		router := gin.New()
		router.Use(HTTPLogger())
		req := httptest.NewRequest("GET", "/*any", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)

		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, "[access]")
		assert.Contains(t, logOutput, "GET")
		assert.Contains(t, logOutput, "unknown")
		assert.Contains(t, logOutput, "404")
	})

	t.Run("measures request latency", func(t *testing.T) {
		logBuffer.Reset()
		router := gin.New()
		router.Use(HTTPLogger())
		router.GET("/slow", func(c *gin.Context) {
			time.Sleep(5 * time.Millisecond)
			c.JSON(http.StatusOK, gin.H{"message": "slow response"})
		})

		req := httptest.NewRequest("GET", "/slow", nil)
		w := httptest.NewRecorder()

		start := time.Now()
		router.ServeHTTP(w, req)
		elapsed := time.Since(start)
		assert.Equal(t, http.StatusOK, w.Code)

		assert.GreaterOrEqual(t, elapsed, 5*time.Millisecond)

		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, "[access]")
		assert.Contains(t, logOutput, "/slow")
		assert.Regexp(t, `\d+(\.\d+)?(ms|µs|ns)`, logOutput)
	})
}
