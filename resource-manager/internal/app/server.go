package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/adapters/etcd"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/adapters/kubernetes"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/adapters/redisq"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/api"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/application"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/ports"
	rmtypes "github.com/Meesho/BharatMLStack/resource-manager/internal/types"
	"github.com/Meesho/BharatMLStack/resource-manager/pkg/config"
)

type Server struct {
	httpServer *http.Server
}

func NewServer() *Server {
	envCfg := config.Instance()

	var shadowStore ports.ShadowStateStore
	var idempotencyStore ports.IdempotencyKeyStore
	var operationStore ports.OperationStore

	if envCfg.UseMockAdapters {
		shadowStore = etcd.NewMemoryShadowStateStore(seedShadowDeployables())
		idempotencyStore = etcd.NewMemoryIdempotencyKeyStore()
	} else {
		etcdClient, err := etcd.NewClient(etcd.ClientConfig{
			Endpoints: envCfg.EtcdEndpoints,
			Username:  envCfg.EtcdUsername,
			Password:  envCfg.EtcdPassword,
			Timeout:   envCfg.EtcdTimeout,
		})
		if err != nil {
			panic(err)
		}
		shadowStore = etcd.NewEtcdShadowStateStore(etcdClient.Raw())
		idempotencyStore = etcd.NewEtcdIdempotencyKeyStore(etcdClient.Raw())
		operationStore = etcd.NewEtcdOperationStore(etcdClient.Raw())
	}

	publisher := redisq.NewInMemoryPublisher()
	kubeExecutor := kubernetes.NewMockExecutor()

	shadowService := application.NewShadowService(shadowStore)
	operationService := application.NewOperationService(publisher, kubeExecutor, operationStore)
	handler := api.NewHandler(shadowService, operationService, idempotencyStore)

	mux := http.NewServeMux()
	handler.Register(mux)

	server := &http.Server{
		Addr:              ":" + intToString(envCfg.AppPort),
		Handler:           metricsMiddleware(requestIDMiddleware(mux)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return &Server{httpServer: server}
}

func (s *Server) Run() error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.httpServer.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-stop:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = sig
		return s.httpServer.Shutdown(ctx)
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func intToString(i int) string {
	return fmt.Sprintf("%d", i)
}

func seedShadowDeployables() map[string][]models.ShadowDeployable {
	now := time.Now().UTC()
	return map[string][]models.ShadowDeployable{
		"int": {
			{
				Name:          "int-predator-g2-std-8",
				NodePool:      "g2-std-8",
				DNS:           "predator-g2-std-8.meesho.int",
				State:         rmtypes.ShadowStateFree,
				MinPodCount:   1,
				LastUpdatedAt: now,
				Version:       1,
			},
			{
				Name:          "int-predator-g2-std-16",
				NodePool:      "g2-std-16",
				DNS:           "predator-g2-std-16.meesho.int",
				State:         rmtypes.ShadowStateFree,
				MinPodCount:   1,
				LastUpdatedAt: now,
				Version:       1,
			},
		},
	}
}
