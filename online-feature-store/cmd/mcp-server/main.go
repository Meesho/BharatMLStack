package main

import (
	"os"
	"strings"

	"github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	featureConfig "github.com/Meesho/BharatMLStack/online-feature-store/internal/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/data/repositories/provider"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/handler/feature"
	mcpserver "github.com/Meesho/BharatMLStack/online-feature-store/internal/mcp"
	"github.com/Meesho/BharatMLStack/online-feature-store/internal/system"
	pkgConfig "github.com/Meesho/BharatMLStack/online-feature-store/pkg/config"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/etcd"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/infra"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/logger"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/metric"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"
)

const configManagerVersion = 1
const normalizedEntitiesWatchPath = "/entities"
const registeredClientsWatchPath = "/security/reader"
const cbWatchPath = "/circuitbreaker"

func main() {
	pkgConfig.InitEnv()
	logger.Init()
	etcd.Init(configManagerVersion, &config.FeatureRegistry{})
	metric.Init()
	infra.InitDBConnectors()
	system.Init()
	featureConfig.InitEtcDBridge()

	configManager := featureConfig.Instance(featureConfig.DefaultVersion)
	configManager.UpdateCBConfigs()
	if err := etcd.Instance().RegisterWatchPathCallback(cbWatchPath, configManager.UpdateCBConfigs); err != nil {
		log.Error().Err(err).Msg("Error registering watch path callback for circuit breaker")
	}
	configManager.GetNormalizedEntities()
	if err := etcd.Instance().RegisterWatchPathCallback(normalizedEntitiesWatchPath, configManager.GetNormalizedEntities); err != nil {
		log.Error().Err(err).Msg("Error registering watch path callback for normalized entities")
	}
	configManager.RegisterClients()
	if err := etcd.Instance().RegisterWatchPathCallback(registeredClientsWatchPath, configManager.RegisterClients); err != nil {
		log.Error().Err(err).Msg("Error registering watch path callback for registered clients")
	}
	provider.InitProvider(configManager, etcd.Instance())

	handler := feature.InitV1()

	mcpSrv := mcpserver.NewServer(handler)

	addr := os.Getenv("MCP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	var httpOpts []server.StreamableHTTPOption

	if strings.EqualFold(os.Getenv("MCP_TLS_ENABLED"), "true") {
		certFile := os.Getenv("MCP_TLS_CERT_FILE")
		keyFile := os.Getenv("MCP_TLS_KEY_FILE")
		if certFile == "" || keyFile == "" {
			log.Fatal().Msg("MCP_TLS_CERT_FILE and MCP_TLS_KEY_FILE must be set when MCP_TLS_ENABLED=true")
		}
		httpOpts = append(httpOpts, server.WithTLSCert(certFile, keyFile))
		log.Info().Str("addr", addr).Msg("Starting OFS MCP server with TLS (Streamable HTTP)")
	} else {
		log.Info().Str("addr", addr).Msg("Starting OFS MCP server (Streamable HTTP)")
	}

	httpServer := server.NewStreamableHTTPServer(mcpSrv, httpOpts...)
	if err := httpServer.Start(addr); err != nil {
		log.Fatal().Err(err).Msg("MCP server error")
	}
}
