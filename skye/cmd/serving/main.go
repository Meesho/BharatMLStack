package main

import (
	pb "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye/client/grpc"
	"github.com/Meesho/BharatMLStack/skye/internal/bootstrap"
	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
	"github.com/Meesho/BharatMLStack/skye/internal/config/structs"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/vector"
	"github.com/Meesho/BharatMLStack/skye/internal/server/api"
	"github.com/Meesho/BharatMLStack/skye/internal/server/middlewares"
	"github.com/Meesho/BharatMLStack/skye/internal/serving/handlers/embedding"
	"github.com/Meesho/BharatMLStack/skye/internal/serving/handlers/similar_candidate"
	"github.com/Meesho/BharatMLStack/skye/pkg/etcd"
	"github.com/Meesho/BharatMLStack/skye/pkg/grpc"
	"github.com/Meesho/BharatMLStack/skye/pkg/infra"
	"github.com/Meesho/BharatMLStack/skye/pkg/logger"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/Meesho/BharatMLStack/skye/pkg/profiling"
)

const (
	EntityWatchPath = "/entity"
)

func main() {
	bootstrap.InitServing()
	appConfig := structs.GetAppConfig()
	logger.Init()
	metric.Init()
	infra.InitRedis()
	profiling.Init()
	etcd.InitFromAppName(&config.Skye{}, appConfig.Configs.AppName, appConfig.Configs)
	grpc.Init(middlewares.ServerInterceptor)
	api.Init()
	configManager := config.NewManager(config.DefaultVersion)
	configManager.RegisterWatchPathCallbackWithEvent(EntityWatchPath, vector.GetRepository(enums.QDRANT).RefreshClients)
	configManager.RegisterWatchPathCallbackWithEvent(EntityWatchPath, vector.GetRepository(enums.NGT).RefreshClients)
	configManager.RegisterWatchPathCallbackWithEvent(EntityWatchPath, vector.GetRepository(enums.EIGENIX).RefreshClients)
	pb.RegisterSkyeSimilarCandidateServiceServer(grpc.Instance().GRPCServer, similar_candidate.GetHandler(1))
	pb.RegisterSkyeEmbeddingServiceServer(grpc.Instance().GRPCServer, embedding.GetHandler(1))
	grpc.Instance().Run()
}
