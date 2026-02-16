package main

import (
	"os"
	"strings"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	mcpserver "github.com/Meesho/BharatMLStack/horizon/internal/mcp"
	onlinefeaturestore "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store"
	ofsConfig "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config"
	ofsHandler "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/handler"
	"github.com/Meesho/BharatMLStack/horizon/pkg/etcd"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/Meesho/BharatMLStack/horizon/pkg/logger"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"
)

// AppConfig holds the application configuration for the MCP server.
type AppConfig struct {
	Configs        configs.Configs
	DynamicConfigs configs.DynamicConfigs
}

func (cfg *AppConfig) GetStaticConfig() interface{} {
	return &cfg.Configs
}

func (cfg *AppConfig) GetDynamicConfig() interface{} {
	return &cfg.DynamicConfigs
}

func main() {
	var appConfig AppConfig

	configs.InitConfig(&appConfig)
	logger.Init(appConfig.Configs)
	infra.InitDBConnectors(appConfig.Configs)
	etcd.InitFromAppName(&ofsConfig.FeatureRegistry{}, appConfig.Configs.OnlineFeatureStoreAppName, appConfig.Configs)
	onlinefeaturestore.Init(appConfig.Configs)

	configHandler := ofsHandler.NewConfigHandler(1)
	if configHandler == nil {
		log.Fatal().Msg("failed to initialize online-feature-store config handler")
	}

	mcpSrv := mcpserver.NewServer(configHandler)

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
		log.Info().Str("addr", addr).Msg("Starting Horizon MCP server with TLS (Streamable HTTP)")
	} else {
		log.Info().Str("addr", addr).Msg("Starting Horizon MCP server (Streamable HTTP)")
	}

	httpServer := server.NewStreamableHTTPServer(mcpSrv, httpOpts...)
	if err := httpServer.Start(addr); err != nil {
		log.Fatal().Err(err).Msg("MCP server error")
	}
}
