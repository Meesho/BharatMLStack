package main

import (
	"log"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/app"
	"github.com/Meesho/BharatMLStack/resource-manager/pkg/config"
	pkgetcd "github.com/Meesho/BharatMLStack/resource-manager/pkg/etcd"
	"github.com/Meesho/BharatMLStack/resource-manager/pkg/logger"
	"github.com/Meesho/BharatMLStack/resource-manager/pkg/metric"
)

func main() {
	if err := config.InitEnv(); err != nil {
		log.Fatalf("failed to initialize env: %v", err)
	}
	logger.Init()
	metric.Init()
	pkgetcd.Init(pkgetcd.DefaultVersion, &struct{}{})

	server := app.NewServer()
	if err := server.Run(); err != nil {
		log.Fatalf("api-server exited with error: %v", err)
	}
}
