package handler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	inferflowPkg "github.com/Meesho/BharatMLStack/horizon/internal/inferflow"
	pb "github.com/Meesho/BharatMLStack/horizon/internal/inferflow/handler/proto/protogen"
	inferflow "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/inferflow"
	"github.com/Meesho/BharatMLStack/horizon/pkg/grpc"
	"github.com/Meesho/BharatMLStack/horizon/pkg/random"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/metadata"
)

func (m *InferFlow) GenerateFunctionalTestRequest(request GenerateRequestFunctionalTestingRequest) (GenerateRequestFunctionalTestingResponse, error) {

	response := GenerateRequestFunctionalTestingResponse{
		RequestBody: RequestBody{
			Entities: []Entity{
				{
					Entity:   request.Entity + "_id",
					Ids:      []string{},
					Features: []FeatureValue{},
				},
			},
			ModelConfigID: request.ModelConfigID,
		},
	}

	batchSize, err := strconv.Atoi(strings.TrimSpace(request.BatchSize))
	if err != nil {
		response.Error = fmt.Errorf("invalid batch size: %w", err).Error()
		return response, errors.New("invalid batch size: " + err.Error())
	}
	if batchSize <= 0 {
		response.Error = "batch size must be a positive integer"
		return response, errors.New(response.Error)
	}

	response.RequestBody.Entities[0].Entity = request.Entity + "_id"
	response.RequestBody.Entities[0].Ids = random.GenerateRandomIntSliceWithRange(batchSize, 100000, 1000000)

	for feature, value := range request.DefaultFeatures {
		featureValues := make([]string, batchSize)
		for i := 0; i < batchSize; i++ {
			featureValues[i] = value
		}
		response.RequestBody.Entities[0].Features = append(response.RequestBody.Entities[0].Features, FeatureValue{
			Name:            feature,
			IdsFeatureValue: featureValues,
		})
	}

	response.MetaData = request.MetaData

	return response, nil
}

func (m *InferFlow) ExecuteFuncitonalTestRequest(request ExecuteRequestFunctionalTestingRequest) (ExecuteRequestFunctionalTestingResponse, error) {
	response := ExecuteRequestFunctionalTestingResponse{}

	normalizedEndpoint := func(raw string) string {
		ep := strings.TrimSpace(raw)
		if strings.HasPrefix(ep, "http://") {
			ep = strings.TrimPrefix(ep, "http://")
		} else if strings.HasPrefix(ep, "https://") {
			ep = strings.TrimPrefix(ep, "https://")
		}
		ep = strings.TrimSuffix(ep, "/")
		if idx := strings.LastIndex(ep, ":"); idx != -1 {
			if idx < len(ep)-1 {
				ep = ep[:idx]
			}
		}

		port := ":8080"
		env := strings.ToLower(strings.TrimSpace(inferflowPkg.AppEnv))
		if env == "stg" || env == "int" {
			port = ":80"
		}
		ep = ep + port
		return ep
	}(request.EndPoint)

	conn, err := grpc.GetConnection(normalizedEndpoint)
	if err != nil {
		response.Error = err.Error()
		return response, errors.New("failed to get connection: " + err.Error())
	}
	defer conn.Close()

	protoRequest := &pb.InferflowRequestProto{}
	protoRequest.ModelConfigId = request.RequestBody.ModelConfigID

	md := metadata.New(nil)
	if len(request.MetaData) > 0 {
		for key, value := range request.MetaData {
			md.Set(key, value)
		}
	}

	md.Set(setFunctionalTest, "true")

	protoRequest.Entities = make([]*pb.InferflowRequestProto_Entity, len(request.RequestBody.Entities))
	for i, entity := range request.RequestBody.Entities {

		protoFeatures := make([]*pb.InferflowRequestProto_Entity_Feature, len(entity.Features))

		for j, feature := range entity.Features {
			protoFeatures[j] = &pb.InferflowRequestProto_Entity_Feature{
				Name:            feature.Name,
				IdsFeatureValue: feature.IdsFeatureValue,
			}
		}

		protoRequest.Entities[i] = &pb.InferflowRequestProto_Entity{
			Entity:   entity.Entity,
			Ids:      entity.Ids,
			Features: protoFeatures,
		}
	}

	protoResponse := &pb.InferflowResponseProto{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()
	err = grpc.SendGRPCRequest(ctx, conn, inferFlowRetrieveModelScoreMethod, protoRequest, protoResponse, md)
	if err != nil {
		response.Error = err.Error()
		log.Error().Msgf("error: %v", err)
		return response, errors.New("failed to send grpc request: " + err.Error())
	}

	for _, compData := range protoResponse.GetComponentData() {
		response.ComponentData = append(response.ComponentData, ComponentData{
			Data: compData.GetData(),
		})
	}

	for i, compData := range response.ComponentData {
		if i == 0 {
			continue
		}
		for j, data := range compData.Data {
			if data == "" {
				response.Error = fmt.Sprintf("response data is empty for field: %s", response.ComponentData[0].Data[j])
				break
			}
		}
	}

	if protoResponse.GetError() != nil {
		response.Error = protoResponse.GetError().GetMessage()
	}

	inferFlowConfig, err := m.InferFlowConfigRepo.GetByID(request.RequestBody.ModelConfigID)

	if err != nil {
		response.Error = fmt.Sprintf("failed to get inferflow config: %v", err)
		return response, errors.New(response.Error)
	}
	if inferFlowConfig == nil {
		response.Error = fmt.Sprintf("inferflow config '%s' does not exist in DB", request.RequestBody.ModelConfigID)
		return response, errors.New(response.Error)
	}

	if response.Error != emptyResponse {
		inferFlowConfig.TestResults = inferflow.TestResults{
			Tested:  false,
			Message: response.Error,
		}
	} else {
		inferFlowConfig.TestResults = inferflow.TestResults{
			Tested:  true,
			Message: "Functional test request executed successfully",
		}
	}
	if err := m.InferFlowConfigRepo.Update(inferFlowConfig); err != nil {
		response.Error = fmt.Sprintf("failed to update inferflow config: %v", err)
		return response, errors.New(response.Error)
	}

	return response, nil
}
