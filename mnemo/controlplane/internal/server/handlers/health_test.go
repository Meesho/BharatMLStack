package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func readyRouter(h *Handlers) *gin.Engine {
	r := gin.New()
	r.GET("/api/v1/ready", h.Ready)
	return r
}

func TestReady_OK(t *testing.T) {
	m := &mockStateClient{}
	m.On("Health", mock.Anything).Return(nil)

	w := httptest.NewRecorder()
	readyRouter(New(m)).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/ready", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ready")
	m.AssertExpectations(t)
}

func TestReady_EtcdUnreachable(t *testing.T) {
	m := &mockStateClient{}
	m.On("Health", mock.Anything).Return(errors.New("etcd dial timeout"))

	w := httptest.NewRecorder()
	readyRouter(New(m)).ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/ready", nil))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "not ready")
	m.AssertExpectations(t)
}
