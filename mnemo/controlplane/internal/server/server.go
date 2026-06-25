// Package server wires the Gin router and starts the HTTP server.
package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/etcdstate"
	"github.com/Meesho/BharatMLStack/mnemo/controlplane/internal/server/handlers"
)

// Server is the mNemo control plane HTTP server.
type Server struct {
	addr   string
	router *gin.Engine
}

// New creates a Server, registers all routes, and wires the handlers.
func New(addr string, state etcdstate.StateClient) *Server {
	r := gin.New()
	r.Use(gin.Recovery())

	s := &Server{addr: addr, router: r}
	h := handlers.New(state)
	s.registerRoutes(h)
	return s
}

// Handler returns the underlying http.Handler (useful in tests).
func (s *Server) Handler() http.Handler {
	return s.router
}

// Run starts the HTTP server and blocks until ctx is cancelled.
// It performs a graceful 5-second shutdown after cancellation.
func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:    s.addr,
		Handler: s.router,
	}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) registerRoutes(h *handlers.Handlers) {
	api := s.router.Group("/api/v1")

	api.POST("/tenants/:tenant/stores", h.OnboardStore)
	api.GET("/tenants/:tenant/stores/:store", h.GetStore)

	api.POST("/tenants/:tenant/stores/:store/versions/:vId/publish", h.PublishVersion)
	api.POST("/tenants/:tenant/stores/:store/versions/:vId/promote", h.PromoteVersion)
	api.POST("/tenants/:tenant/stores/:store/rollback", h.Rollback)
	api.POST("/tenants/:tenant/stores/:store/versions/:vId/retire", h.RetireVersion)

	api.PUT("/tenants/:tenant/stores/:store/dataflow", h.PutDataflow)
	api.GET("/tenants/:tenant/stores/:store/dataflow", h.GetDataflow)

	api.PUT("/tenants/:tenant/stores/:store/clientConfig", h.PutClientConfig)
	api.GET("/tenants/:tenant/stores/:store/clientConfig", h.GetClientConfig)

	api.GET("/tenants/:tenant/stores/:store/topology", h.GetTopology)

	s.router.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Readiness: etcd-aware — pod leaves the Service endpoints when etcd is down.
	s.router.GET("/api/v1/ready", h.Ready)
}
