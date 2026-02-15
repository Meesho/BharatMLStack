package inferflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	grpc2 "github.com/Meesho/BharatMLStack/go-sdk/pkg/clients/inferflow/client/grpc"
	"github.com/Meesho/BharatMLStack/go-sdk/pkg/grpcclient"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"google.golang.org/grpc/metadata"
)

var (
	client  *ClientV1
	once    sync.Once
	headers metadata.MD
)

const (
	V1Prefix         = "INFERFLOW_CLIENT_V1_"
	CallerIDMetadata = "inferflow-caller-id"
	AuthMetadata     = "inferflow-auth-token"
)

func InitV1Client() InferflowClient {
	if client == nil {
		once.Do(func() {
			clientConfig, err := getClientConfigs(V1Prefix)
			if err != nil {
				log.Panic().Err(err).Msgf("Invalid Inferflow client configs: %#v", clientConfig)
			}
			grpcClient, grpcErr := getGrpcClient(clientConfig)
			headers = getMetadata(clientConfig.AuthToken)
			if grpcErr != nil {
				log.Panic().Err(grpcErr).Msgf("Error creating inferflow service grpc client, client: %#v", grpcClient)
			}
			client = &ClientV1{
				ClientConfigs: clientConfig,
				GrpcClient:    grpcClient,
			}
		})
	}
	return client
}

func InitV1ClientFromConfig(conf ClientConfig, callerId string) InferflowClient {
	if client == nil {
		once.Do(func() {
			grpcClient, grpcErr := getGrpcClient(&conf)
			if grpcErr != nil {
				log.Panic().Err(grpcErr).Msgf("Error creating inferflow service grpc client, client: %#v", grpcClient)
			}
			headers = metadata.New(map[string]string{
				CallerIDMetadata: callerId,
				AuthMetadata:     conf.AuthToken,
			})
			client = &ClientV1{
				ClientConfigs: &conf,
				GrpcClient:    grpcClient,
			}
		})
	}
	return client
}

func getGrpcClient(conf *ClientConfig) (*grpcclient.GRPCClient, error) {
	var client *grpcclient.GRPCClient
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic creating grpc client from prefix: %v", r)
		}
	}()
	client = grpcclient.NewConnFromConfig(&grpcclient.Config{
		Host:                conf.Host,
		Port:                conf.Port,
		DeadLine:            conf.DeadlineExceedMS,
		LoadBalancingPolicy: "round_robin",
		PlainText:           conf.PlainText,
	}, V1Prefix)
	return client, err
}

func (c *ClientV1) InferPointWise(req *grpc2.PointWiseRequest) (*grpc2.PointWiseResponse, error) {
	predictClient := grpc2.NewPredictClient(c.GrpcClient)
	timeout := time.Duration(c.ClientConfigs.DeadlineExceedMS) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, headers)
	protoResponse, err := predictClient.InferPointWise(ctx, req)
	if err != nil {
		log.Error().Msgf("Error while calling InferPointWise on inferflow service, err: %v", err)
		return nil, err
	} else if protoResponse == nil {
		log.Error().Msgf("Empty response from inferflow InferPointWise")
		return nil, fmt.Errorf("empty response from inferflow InferPointWise")
	}
	return protoResponse, nil
}

func (c *ClientV1) InferPairWise(req *grpc2.PairWiseRequest) (*grpc2.PairWiseResponse, error) {
	predictClient := grpc2.NewPredictClient(c.GrpcClient)
	timeout := time.Duration(c.ClientConfigs.DeadlineExceedMS) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, headers)
	protoResponse, err := predictClient.InferPairWise(ctx, req)
	if err != nil {
		log.Error().Msgf("Error while calling InferPairWise on inferflow service, err: %v", err)
		return nil, err
	} else if protoResponse == nil {
		log.Error().Msgf("Empty response from inferflow InferPairWise")
		return nil, fmt.Errorf("empty response from inferflow InferPairWise")
	}
	return protoResponse, nil
}

func (c *ClientV1) InferSlateWise(req *grpc2.SlateWiseRequest) (*grpc2.SlateWiseResponse, error) {
	predictClient := grpc2.NewPredictClient(c.GrpcClient)
	timeout := time.Duration(c.ClientConfigs.DeadlineExceedMS) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, headers)
	protoResponse, err := predictClient.InferSlateWise(ctx, req)
	if err != nil {
		log.Error().Msgf("Error while calling InferSlateWise on inferflow service, err: %v", err)
		return nil, err
	} else if protoResponse == nil {
		log.Error().Msgf("Empty response from inferflow InferSlateWise")
		return nil, fmt.Errorf("empty response from inferflow InferSlateWise")
	}
	return protoResponse, nil
}

func getMetadata(authToken string) metadata.MD {
	callerId := viper.GetString("INFERFLOW_CALLER_ID")
	if callerId == "" {
		log.Panic().Msgf("INFERFLOW_CALLER_ID not set!")
	}
	md := metadata.New(map[string]string{
		CallerIDMetadata: callerId,
	})
	if authToken != "" {
		md.Set(AuthMetadata, authToken)
	}
	return md
}
