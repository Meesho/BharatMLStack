package handler

import (
	"fmt"
	"time"

	connection_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/connectionconfig"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
)

var (
	emptyResponse = ""
	activeTrue    = true
	grpcConfig    = "grpc"
	httpConfig    = "http"
)

type ConnectionConfig struct {
	connectionConfigRepo connection_config.ConnectionConfigRepository
}

func InitV1ConfigHandler() Config {
	if config == nil {
		conn, err := infra.SQL.GetConnection()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get SQL connection")
		}
		sqlConn := conn.(*infra.SQLConnection)

		connectionConfigRepo, err := connection_config.Repository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create config repository")
		}

		config = &ConnectionConfig{
			connectionConfigRepo: connectionConfigRepo,
		}
	}
	return config
}

func (c *ConnectionConfig) Onboard(request OnboardRequest) (Response, error) {
	var application connection_config.ConnectionConfig

	if request.Payload.ConnProtocol == grpcConfig {
		grpcCfg := connection_config.GrpcConfig{
			Deadline:              request.Payload.Deadline,
			PlainText:             request.Payload.PlainText,
			GrpcChannelAlgorithm:  request.Payload.GrpcChannelAlgorithm,
			ChannelThreadPoolSize: request.Payload.ChannelThreadPoolSize,
			BoundedQueueSize:      request.Payload.BoundedQueueSize,
		}

		application = connection_config.ConnectionConfig{
			Service:      request.Payload.Service,
			ConnProtocol: request.Payload.ConnProtocol,
			Default:      request.Payload.Default,
			GrpcConfig:   grpcCfg,
			Active:       activeTrue,
			CreatedBy:    request.CreatedBy,
		}
	} else if request.Payload.ConnProtocol == httpConfig {
		httpCfg := connection_config.HttpConfig{
			Timeout:               request.Payload.Timeout,
			MaxIdleConnection:     request.Payload.MaxIdleConnection,
			MaxConnectionPerHost:  request.Payload.MaxConnectionPerHost,
			IdleConnectionTimeout: request.Payload.IdleConnectionTimeout,
			KeepAliveTime:         request.Payload.KeepAliveTime,
		}

		application = connection_config.ConnectionConfig{
			Service:      request.Payload.Service,
			ConnProtocol: request.Payload.ConnProtocol,
			Default:      request.Payload.Default,
			HttpConfig:   httpCfg,
			Active:       activeTrue,
			CreatedBy:    request.CreatedBy,
		}
	}

	err := c.connectionConfigRepo.Create(&application)
	if err != nil {
		return Response{
			Error: err.Error(),
			Data:  Message{Message: emptyResponse},
		}, err
	}
	return Response{
		Error: emptyResponse,
		Data:  Message{Message: fmt.Sprintf("Application onboarded successfully with id: %d", application.Id)},
	}, nil
}

func (c *ConnectionConfig) GetAll() (GetAllResponse, error) {
	connectionConfigs, err := c.connectionConfigRepo.GetAll()
	if err != nil {
		return GetAllResponse{}, err
	}

	response := []Configs{}

	for _, connectionConfig := range connectionConfigs {

		if connectionConfig.ConnProtocol == grpcConfig {
			response = append(response, Configs{
				Id:           connectionConfig.Id,
				Default:      connectionConfig.Default,
				Service:      connectionConfig.Service,
				ConnProtocol: connectionConfig.ConnProtocol,
				Config: ConnConfig{
					GrpcConfig: GrpcConfig{
						Deadline:              connectionConfig.GrpcConfig.Deadline,
						PlainText:             connectionConfig.GrpcConfig.PlainText,
						GrpcChannelAlgorithm:  connectionConfig.GrpcConfig.GrpcChannelAlgorithm,
						ChannelThreadPoolSize: connectionConfig.GrpcConfig.ChannelThreadPoolSize,
						BoundedQueueSize:      connectionConfig.GrpcConfig.BoundedQueueSize,
					},
				},
				CreatedBy: connectionConfig.CreatedBy,
				UpdatedBy: connectionConfig.UpdatedBy,
				CreatedAt: connectionConfig.CreatedAt.Format(time.RFC3339),
				UpdatedAt: connectionConfig.UpdatedAt.Format(time.RFC3339),
			})
		} else if connectionConfig.ConnProtocol == httpConfig {
			response = append(response, Configs{
				Id:           connectionConfig.Id,
				Default:      connectionConfig.Default,
				Service:      connectionConfig.Service,
				ConnProtocol: connectionConfig.ConnProtocol,
				Config: ConnConfig{
					HttpConfig: HttpConfig{
						Timeout:               connectionConfig.HttpConfig.Timeout,
						MaxIdleConnection:     connectionConfig.HttpConfig.MaxIdleConnection,
						MaxConnectionPerHost:  connectionConfig.HttpConfig.MaxConnectionPerHost,
						IdleConnectionTimeout: connectionConfig.HttpConfig.IdleConnectionTimeout,
						KeepAliveTime:         connectionConfig.HttpConfig.KeepAliveTime,
					},
				},
				UpdatedBy: connectionConfig.UpdatedBy,
				CreatedAt: connectionConfig.CreatedAt.Format(time.RFC3339),
				UpdatedAt: connectionConfig.UpdatedAt.Format(time.RFC3339),
			})
		}
	}
	return GetAllResponse{
		Data: response,
	}, nil
}

func (c *ConnectionConfig) Edit(request EditRequest) (Response, error) {
	var connectionConfig connection_config.ConnectionConfig

	// First check if the connection exists
	configs, err := c.connectionConfigRepo.GetAll()
	if err != nil {
		return Response{
			Error: err.Error(),
			Data:  Message{Message: emptyResponse},
		}, err
	}

	found := false
	for _, config := range configs {
		if config.Id == request.Payload.Id {
			found = true
			break
		}
	}

	if !found {
		return Response{
			Error: "Connection config not found",
			Data:  Message{Message: emptyResponse},
		}, fmt.Errorf("connection config with id %d not found", request.Payload.Id)
	}

	if request.Payload.ConnProtocol == grpcConfig {
		grpcCfg := connection_config.GrpcConfig{
			Deadline:              request.Payload.Deadline,
			PlainText:             request.Payload.PlainText,
			GrpcChannelAlgorithm:  request.Payload.GrpcChannelAlgorithm,
			ChannelThreadPoolSize: request.Payload.ChannelThreadPoolSize,
			BoundedQueueSize:      request.Payload.BoundedQueueSize,
		}

		connectionConfig = connection_config.ConnectionConfig{
			Id:           request.Payload.Id,
			Default:      request.Payload.Default,
			Service:      request.Payload.Service,
			ConnProtocol: request.Payload.ConnProtocol,
			GrpcConfig:   grpcCfg,
			Active:       activeTrue,
			UpdatedBy:    request.UpdatedBy,
		}
	} else if request.Payload.ConnProtocol == httpConfig {
		httpCfg := connection_config.HttpConfig{
			Timeout:               request.Payload.Timeout,
			MaxIdleConnection:     request.Payload.MaxIdleConnection,
			MaxConnectionPerHost:  request.Payload.MaxConnectionPerHost,
			IdleConnectionTimeout: request.Payload.IdleConnectionTimeout,
			KeepAliveTime:         request.Payload.KeepAliveTime,
		}

		connectionConfig = connection_config.ConnectionConfig{
			Id:           request.Payload.Id,
			Default:      request.Payload.Default,
			Service:      request.Payload.Service,
			ConnProtocol: request.Payload.ConnProtocol,
			HttpConfig:   httpCfg,
			Active:       activeTrue,
			UpdatedBy:    request.UpdatedBy,
		}
	} else {
		return Response{
			Error: "Invalid connection protocol",
			Data:  Message{Message: emptyResponse},
		}, fmt.Errorf("invalid connection protocol: %s", request.Payload.ConnProtocol)
	}

	err = c.connectionConfigRepo.Update(&connectionConfig)
	if err != nil {
		return Response{
			Error: err.Error(),
			Data:  Message{Message: emptyResponse},
		}, err
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{Message: fmt.Sprintf("Connection config updated successfully with id: %d", connectionConfig.Id)},
	}, nil
}
