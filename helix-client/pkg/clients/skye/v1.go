package skye

import (
	"context"
	"fmt"
	"sync"
	"time"

	grpc2 "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye/client/grpc"
	"github.com/Meesho/BharatMLStack/helix-client/pkg/grpcclient"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"google.golang.org/grpc/metadata"
)

var (
	client *ClientV1
	once   sync.Once
)

const (
	V1Prefix         = "SKYE_CLIENT_V1_"
	CallerIDMetadata = "skye-caller-id"
	AuthMetadata     = "skye-auth-token"
)

func InitV1Client() SkyeClient {
	if client == nil {
		once.Do(func() {
			clientConfig, err := getClientConfigs(V1Prefix)
			if err != nil {
				log.Panic().Err(err).Msgf("Invalid Skye client configs: %#v", clientConfig)
			}
			grpcClient, grpcErr := getGrpcClient(clientConfig)
			if grpcErr != nil {
				log.Panic().Err(grpcErr).Msgf("Error creating skye service grpc client, client: %#v", grpcClient)
			}
			client = &ClientV1{
				ClientConfigs: clientConfig,
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

func (c *ClientV1) GetSimilarCandidates(req *grpc2.SkyeRequest) (*grpc2.SkyeResponse, error) {
	skyeClient := grpc2.NewSkyeSimilarCandidateServiceClient(c.GrpcClient)
	timeout := time.Duration(c.ClientConfigs.DeadlineExceedMS) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	//get metadata headers
	md := getMetadata(c.ClientConfigs)
	ctx = metadata.NewOutgoingContext(ctx, md)
	// call grpc method
	protoResponse, err := skyeClient.GetSimilarCandidates(ctx, req)
	if err != nil {
		log.Error().Msgf("Error while fetching similar candidates from skye service, err: %v", err)
		return nil, err
	} else if protoResponse == nil {
		log.Error().Msgf("Empty response from skye service")
		return nil, fmt.Errorf("empty response from skye service")
	}
	return protoResponse, nil
}

func (c *ClientV1) GetEmbeddingsForCandidateIds(request *grpc2.SkyeBulkEmbeddingRequest) (*grpc2.SkyeBulkEmbeddingResponse, error) {
	skyeClient := grpc2.NewSkyeEmbeddingServiceClient(c.GrpcClient)
	timeout := time.Duration(c.ClientConfigs.DeadlineExceedMS) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	//get metadata headers
	md := getMetadata(c.ClientConfigs)
	ctx = metadata.NewOutgoingContext(ctx, md)
	// call grpc method
	protoResponse, err := skyeClient.GetEmbeddingsForCandidates(ctx, request)
	if err != nil {
		log.Error().Msgf("Error while fetching bulk embeddings from skye service, err: %v", err)
		return nil, err
	} else if protoResponse == nil {
		log.Error().Msgf("Empty response from skye service")
		return nil, fmt.Errorf("empty response from skye service")
	}
	return protoResponse, nil
}

func (c *ClientV1) GetDotProductOfCandidatesForEmbedding(request *grpc2.EmbeddingDotProductRequest) (*grpc2.EmbeddingDotProductResponse, error) {
	skyeClient := grpc2.NewSkyeEmbeddingServiceClient(c.GrpcClient)
	timeout := time.Duration(c.ClientConfigs.DeadlineExceedMS) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	//get metadata headers
	md := getMetadata(c.ClientConfigs)
	ctx = metadata.NewOutgoingContext(ctx, md)
	// call grpc method
	protoResponse, err := skyeClient.GetCandidateEmbeddingScores(ctx, request)
	if err != nil {
		log.Error().Msgf("Error while fetching bulk embeddings from skye service, err: %v", err)
		return nil, err
	} else if protoResponse == nil {
		log.Error().Msgf("Empty response from skye service")
		return nil, fmt.Errorf("empty response from skye service")
	}
	return protoResponse, nil
}

func getMetadata(config *ClientConfig) metadata.MD {
	md := metadata.New(nil)
	appName := viper.GetString("APP_NAME")
	if appName == "" {
		log.Panic().Msgf("APP_NAME not set!")
	}
	md.Append(CallerIDMetadata, appName)
	md.Append(AuthMetadata, config.AuthToken)
	return md
}
