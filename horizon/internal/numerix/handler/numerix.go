package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	numerixPkg "github.com/Meesho/BharatMLStack/horizon/internal/numerix"
	numerixConfig "github.com/Meesho/BharatMLStack/horizon/internal/numerix/etcd"
	pb "github.com/Meesho/BharatMLStack/horizon/internal/numerix/proto/protogen"
	util "github.com/Meesho/BharatMLStack/horizon/internal/numerix/util"
	numerix_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/numerix/config"
	numerix_request "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/numerix/request"
	"github.com/Meesho/BharatMLStack/horizon/pkg/grpc"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/Meesho/BharatMLStack/horizon/pkg/random"
	"github.com/Meesho/BharatMLStack/horizon/pkg/serializer"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type Numerix struct {
	Config             numerixConfig.Manager
	NumerixConfigRepo  numerix_config.Repository
	NumerixRequestRepo numerix_request.Repository
}

const (
	emptyResponse        = ""
	pendingApproval      = "PENDING APPROVAL"
	approved             = "APPROVED"
	rejected             = "REJECTED"
	cancelled            = "CANCELLED"
	promoteRequestType   = "PROMOTE"
	adminRole            = "ADMIN"
	onboardRequestType   = "ONBOARD"
	editRequestType      = "EDIT"
	activeTrue           = true
	typeFloat32          = "fp32"
	typeFloat64          = "fp64"
	numerixComputeMethod = "/numerix.Numerix/Compute"
	defaultPage          = 1
	defaultPageSize      = 25
)

func InitV1ConfigHandler() Config {
	if config == nil {
		conn, err := infra.SQL.GetConnection()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get SQL connection")
		}
		sqlConn := conn.(*infra.SQLConnection)

		numerixConfigRepo, err := numerix_config.NewRepository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create config repository")
		}

		numerixRequestRepo, err := numerix_request.NewRepository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create request repository")
		}

		config = &Numerix{
			Config:             numerixConfig.NewEtcdInstance(),
			NumerixConfigRepo:  numerixConfigRepo,
			NumerixRequestRepo: numerixRequestRepo,
		}
	}
	return config
}

func (i *Numerix) Edit(request EditConfigRequest) (Response, error) {
	exists, err := i.NumerixConfigRepo.DoesConfigIDExist(request.Payload.ConfigID)
	if err != nil {
		return Response{}, err
	}
	if !exists {
		return Response{
			Error: "Config ID does not exist",
			Data:  Message{emptyResponse},
		}, nil
	}

	table := &numerix_request.Table{
		ConfigID: request.Payload.ConfigID,
		Payload: numerix_request.RequestExpression{
			Expression: numerix_request.Expression{
				InfixExpression:   request.Payload.ConfigValue.InfixExpression,
				PostfixExpression: request.Payload.ConfigValue.PostfixExpression,
			},
		},
		CreatedBy:   request.CreatedBy,
		RequestType: editRequestType,
		Status:      pendingApproval,
	}

	err = i.NumerixRequestRepo.Create(table)
	if err != nil {
		return Response{}, err
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{fmt.Sprintf("numerix Config edit request created successfully with Request Id %d and Config Id %d", table.RequestID, request.Payload.ConfigID)},
	}, nil
}

func (i *Numerix) CancelRequest(request CancelConfigRequest) (Response, error) {

	exists, err := i.NumerixRequestRepo.DoesRequestIDExistWithStatus(request.RequestID, pendingApproval)
	if err != nil {
		return Response{}, err
	}
	if !exists {
		return Response{
			Error: "Request ID does not exist or request is not pending approval",
			Data:  Message{emptyResponse},
		}, nil
	}

	table := &numerix_request.Table{
		RequestID: request.RequestID,
		Status:    cancelled,
		UpdatedBy: request.UpdatedBy,
	}

	err = i.NumerixRequestRepo.Update(table)
	if err != nil {
		return Response{}, err
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{"Numerix Config onboarding request cancelled successfully."},
	}, nil
}

func (i *Numerix) GetAllRequests(request GetAllRequestConfigsRequest) (GetAllRequestConfigsResponse, error) {

	var tables []numerix_request.Table
	var err error
	request.Role = strings.ToUpper(request.Role)
	if request.Role == adminRole {

		tables, err = i.NumerixRequestRepo.GetAll()
		if err != nil {

			return GetAllRequestConfigsResponse{}, err
		}
	} else {

		tables, err = i.NumerixRequestRepo.GetByUser(request.Email)
		if err != nil {

			return GetAllRequestConfigsResponse{}, err
		}
	}

	requestConfigs := make([]RequestConfig, len(tables))
	for i, table := range tables {
		requestConfigs[i] = RequestConfig{
			RequestID: table.RequestID,
			ComputeId: table.ConfigID,
			Payload: RequestExpression{
				ConfigValue: Expression{
					InfixExpression:   table.Payload.Expression.InfixExpression,
					PostfixExpression: table.Payload.Expression.PostfixExpression,
				},
			},
			CreatedBy:    table.CreatedBy,
			CreatedAt:    table.CreatedAt,
			UpdatedBy:    table.UpdatedBy,
			UpdatedAt:    table.UpdatedAt,
			RequestType:  table.RequestType,
			Status:       table.Status,
			RejectReason: table.RejectReason,
			Reviewer:     table.Reviewer,
		}
	}

	response := GetAllRequestConfigsResponse{
		Error: emptyResponse,
		Data:  requestConfigs,
	}
	return response, nil
}

func (i *Numerix) ReviewRequest(request ReviewRequestConfigRequest) (Response, error) {
	err := i.NumerixRequestRepo.Transaction(func(tx *gorm.DB) error {
		request.Status = strings.ToUpper(request.Status)

		if request.Status != approved && request.Status != rejected {
			return errors.New("invalid status")
		}

		if request.Status == rejected {
			if request.RejectReason == emptyResponse {
				return errors.New("rejection reason is needed")
			}
		}

		exists, err := i.NumerixRequestRepo.DoesRequestIDExistWithStatus(request.RequestID, pendingApproval)
		if err != nil {
			return err
		}
		if !exists {
			return errors.New("request id does not exist or request is not pending approval")
		}

		table := &numerix_request.Table{
			RequestID:    request.RequestID,
			Status:       request.Status,
			RejectReason: request.RejectReason,
			Reviewer:     request.Reviewer,
		}

		tableResponse, err := i.NumerixRequestRepo.UpdateTx(tx, table)
		if err != nil {
			return err
		}

		if request.Status == approved {

			err = i.CreateOrUpdatenumerixConfig(tx, &tableResponse)
			if err != nil {
				return err
			}

			configIdStr := fmt.Sprintf("%d", tableResponse.ConfigID)

			err = i.createOrUpdateEtcdConfig(configIdStr, tableResponse.Payload.Expression.PostfixExpression, tableResponse.RequestType)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create config for etcd")
				return err
			}
		}

		return nil
	})

	if err != nil {
		return Response{}, err
	}

	return Response{
		Error: emptyResponse,
		Data:  Message{"Numerix Config onboarding request reviewed successfully."},
	}, nil
}

func (i *Numerix) CreateOrUpdatenumerixConfig(tx *gorm.DB, table *numerix_request.Table) error {
	newTable := &numerix_config.Table{
		ConfigID: table.ConfigID,
		Active:   activeTrue,
		ConfigValue: numerix_config.Expression{
			InfixExpression:   table.Payload.Expression.InfixExpression,
			PostfixExpression: table.Payload.Expression.PostfixExpression,
		},
	}

	switch table.RequestType {
	case onboardRequestType, promoteRequestType:
		if table.UpdatedBy != "" {
			newTable.CreatedBy = table.UpdatedBy
		} else {
			newTable.CreatedBy = table.CreatedBy
		}
		err := i.NumerixConfigRepo.CreateTx(tx, newTable)
		if err != nil {
			return err
		}
	case editRequestType:
		if table.UpdatedBy != "" {
			newTable.UpdatedBy = table.UpdatedBy
		} else {
			newTable.UpdatedBy = table.CreatedBy
		}
		err := i.NumerixConfigRepo.UpdateTx(tx, newTable)
		if err != nil {
			return err
		}
	default:
		return errors.New("invalid request type")
	}

	return nil
}

func (i *Numerix) createOrUpdateEtcdConfig(configID string, expression string, requestType string) error {
	switch requestType {
	case onboardRequestType, promoteRequestType:
		return i.Config.CreateConfig(configID, expression)
	case editRequestType:
		return i.Config.UpdateConfig(configID, expression)
	default:
		return errors.New("invalid request type")
	}
}

func (i *Numerix) GenerateExpression(request ExpressionGenerateRequest) (ExpressionGenerateResponse, error) {
	err := util.ValidateExpression(request.Expression)
	if err != nil {
		return ExpressionGenerateResponse{}, err
	}
	expression, err := util.InfixToPostfix(request.Expression)
	if err != nil {
		return ExpressionGenerateResponse{}, err
	}
	expressionString := strings.Join(expression, " ")

	response := ExpressionGenerateResponse{
		Error: emptyResponse,
		Data:  expressionString,
	}
	return response, nil
}

func (i *Numerix) GetExpressionVariables(request ExpressionVariablesRequest) (ExpressionVariablesResponse, error) {

	expression, err := i.NumerixConfigRepo.GetExpression(request.ConfigID)
	if err != nil {
		return ExpressionVariablesResponse{}, err
	}
	variables := util.ExtractVariables(expression)

	response := ExpressionVariablesResponse{
		Error: emptyResponse,
		Data:  variables,
	}
	return response, nil
}

func (i *Numerix) GetAll(request GetAllConfigsRequest) (GetAllConfigsResponse, error) {
	if request.Page <= 0 {
		request.Page = defaultPage
	}
	if request.PageSize <= 0 {
		request.PageSize = defaultPageSize
	}

	tables, totalCount, err := i.NumerixConfigRepo.GetAllPaginated(request.Page, request.PageSize, request.ConfigID)
	if err != nil {
		return GetAllConfigsResponse{}, err
	}

	numerixConfigs := make([]NumerixConfig, len(tables))
	for i, table := range tables {
		numerixConfigs[i] = NumerixConfig{
			ConfigID:          table.ConfigID,
			InfixExpression:   table.ConfigValue.InfixExpression,
			PostfixExpression: table.ConfigValue.PostfixExpression,
			CreatedBy:         table.CreatedBy,
			CreatedAt:         table.CreatedAt,
			UpdatedBy:         table.UpdatedBy,
			UpdatedAt:         table.UpdatedAt,
			MonitoringUrl:     numerixPkg.NumerixMonitoringUrl,
			TestResults:       table.TestResults,
		}
	}

	totalPages := int(totalCount) / request.PageSize
	if int(totalCount)%request.PageSize != 0 {
		totalPages++
	}

	response := GetAllConfigsResponse{
		Error: emptyResponse,
		Data:  numerixConfigs,
		Pagination: &PaginationMeta{
			Page:       request.Page,
			PageSize:   request.PageSize,
			TotalCount: totalCount,
			TotalPages: totalPages,
		},
	}
	return response, nil
}

func (i *Numerix) Onboard(request OnboardConfigRequest) (Response, error) {

	table := &numerix_request.Table{
		Payload: numerix_request.RequestExpression{
			Expression: numerix_request.Expression{
				InfixExpression:   request.Payload.ConfigValue.InfixExpression,
				PostfixExpression: request.Payload.ConfigValue.PostfixExpression,
			},
		},
		CreatedBy:   request.CreatedBy,
		RequestType: onboardRequestType,
		Status:      pendingApproval,
	}

	err := i.NumerixRequestRepo.Create(table)
	if err != nil {
		return Response{}, err
	}

	response := Response{
		Error: emptyResponse,
		Data:  Message{fmt.Sprintf("Numerix Config onboarding request raised successfully with Compute ID %d. and Request Id %d", table.RequestID, table.RequestID)},
	}
	return response, nil
}

func (i *Numerix) Promote(request PromoteConfigRequest) (Response, error) {

	exists, err := i.NumerixRequestRepo.DoesConfigIdExistWithRequestType(request.Payload.ConfigID, promoteRequestType)
	if err != nil {
		return Response{}, err
	}
	if exists {
		return Response{
			Error: "Request with this Compute Id is already raised",
			Data:  Message{emptyResponse},
		}, nil
	}

	if request.Payload.ConfigValue.InfixExpression == emptyResponse {
		return Response{
			Error: "Infix expression is required",
			Data:  Message{emptyResponse},
		}, nil
	}

	if request.Payload.ConfigValue.PostfixExpression == emptyResponse {
		return Response{
			Error: "Postfix expression is required",
			Data:  Message{emptyResponse},
		}, nil
	}

	table := &numerix_request.Table{
		Payload: numerix_request.RequestExpression{
			Expression: numerix_request.Expression{
				InfixExpression:   request.Payload.ConfigValue.InfixExpression,
				PostfixExpression: request.Payload.ConfigValue.PostfixExpression,
			},
		},
		ConfigID:    request.Payload.ConfigID,
		RequestType: promoteRequestType,
		CreatedBy:   request.UpdatedBy,
		Status:      pendingApproval,
	}

	err = i.NumerixRequestRepo.Create(table)
	if err != nil {
		return Response{}, err
	}

	response := Response{
		Error: emptyResponse,
		Data:  Message{fmt.Sprintf("Numerix Config Promote request raised successfully with Request Id %d and Compute Id %d", table.RequestID, request.Payload.ConfigID)},
	}
	return response, nil
}

func (g *Numerix) GenerateFuncitonalTestRequest(request RequestGenerationRequest) (FuncitonalRequestGenerationResponse, error) {
	computeId, err := strconv.ParseUint(request.ComputeId, 10, 64)
	if err != nil {
		return FuncitonalRequestGenerationResponse{}, err
	}
	computeIdUint := uint(computeId)
	exists, err := g.NumerixConfigRepo.DoesConfigIDExist(computeIdUint)
	if err != nil {
		return FuncitonalRequestGenerationResponse{}, err
	}

	if !exists {
		return FuncitonalRequestGenerationResponse{}, fmt.Errorf("compute id does not exist")
	}
	expression, err := g.NumerixConfigRepo.GetExpression(computeIdUint)
	if err != nil {
		return FuncitonalRequestGenerationResponse{}, err
	}

	variables := util.ExtractVariables(expression)

	// Make variables unique
	seen := make(map[string]bool)
	unique := []string{"catalog_id"}
	seen["catalog_id"] = true

	for _, v := range variables {
		if !seen[v] {
			unique = append(unique, v)
			seen[v] = true
		}
	}
	variables = unique

	EntityScoreData := EntityScoreData{
		Schema:       variables,
		EntityScores: []Data{},
	}

	batchSize, err := strconv.Atoi(request.BatchSize)
	if err != nil {
		return FuncitonalRequestGenerationResponse{}, err
	}
	if request.DataType == typeFloat32 {
		for i := 0; i < batchSize; i++ {
			EntityScoreData.EntityScores = append(EntityScoreData.EntityScores, Data{
				StringData: DataValues{
					Values: append([]string{"10000000"}, random.GenerateRandomFloat32Slice(len(variables)-1)...),
				},
			})
		}
	} else {
		for i := 0; i < batchSize; i++ {
			EntityScoreData.EntityScores = append(EntityScoreData.EntityScores, Data{
				StringData: DataValues{
					Values: append([]string{"10000000"}, random.GenerateRandomFloat64Slice(len(variables)-1)...),
				},
			})
		}
	}

	return FuncitonalRequestGenerationResponse{
		ComputeId: request.ComputeId,
		DataType:  request.DataType,
		RequestBody: RequestBody{
			EntityScoreData: EntityScoreData,
		},
	}, nil
}

func (g *Numerix) ExecuteFuncitonalTestRequest(request ExecuteRequestFunctionalRequest) (ExecuteRequestFunctionalResponse, error) {

	conn, err := grpc.GetConnection(request.EndPoint)

	//log.Printf("Request: %v", request)
	if err != nil {
		log.Printf("Error getting connection: %v", err)
		return ExecuteRequestFunctionalResponse{}, err
	}
	defer conn.Close()
	var entityScores []*pb.Score
	for _, entityScore := range request.RequestBody.EntityScoreData.EntityScores {
		var values [][]byte
		if request.DataType == typeFloat32 {
			for _, val := range entityScore.StringData.Values {
				v, _ := serializer.Float32ToBytesLE(val)
				values = append(values, v)
			}
		} else {
			for _, val := range entityScore.StringData.Values {
				v, _ := serializer.Float64ToBytesLE(val)
				values = append(values, v)
			}
		}
		score := &pb.Score{
			MatrixFormat: &pb.Score_ByteData{
				ByteData: &pb.ByteList{
					Values: values,
				},
			},
		}
		entityScores = append(entityScores, score)
	}
	protoReq := &pb.NumerixRequestProto{
		EntityScoreData: &pb.EntityScoreData{
			Schema:       request.RequestBody.EntityScoreData.Schema,
			ComputeId:    request.ComputeId,
			DataType:     &request.DataType,
			EntityScores: entityScores,
		},
	}

	protoResponse := &pb.NumerixResponseProto{}

	response := ExecuteRequestFunctionalResponse{}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	//log.Printf("Attempting to connect to endpoint: %s", request.EndPoint)
	err = grpc.SendGRPCRequest(ctx, conn, numerixComputeMethod, protoReq, protoResponse, nil)
	//log.Printf("Proto Request: %v", protoReq)
	//log.Printf("Proto Response: %v", protoResponse)
	if err != nil {
		log.Printf("Error sending grpc request: %v", err)
		return ExecuteRequestFunctionalResponse{}, err
	}

	response.ComputationalScoreData.Schema = protoResponse.GetComputationScoreData().GetSchema()

	computationScores := protoResponse.GetComputationScoreData().GetComputationScores()

	for _, b := range computationScores {
		byteValues := b.GetByteData().Values
		var data []string
		if request.DataType == typeFloat32 {
			for _, byteValue := range byteValues {
				f, _ := serializer.BytesToFloat32LE(byteValue)
				// Use 'f' format to avoid scientific notation
				data = append(data, strconv.FormatFloat(float64(f), 'f', 7, 32))
			}
		} else {
			for _, byteValue := range byteValues {
				f, _ := serializer.BytesToFloat64LE(byteValue)
				// Use 'f' format to avoid scientific notation
				data = append(data, strconv.FormatFloat(f, 'f', 15, 64))
			}
		}
		computationScore := ComputationalScore{
			Data: Data{
				StringData: DataValues{
					Values: data,
				},
			},
		}
		response.ComputationalScoreData.ComputationalScores = append(response.ComputationalScoreData.ComputationalScores, computationScore)
	}

	numerixConfig, err := g.NumerixConfigRepo.GetByConfigID(request.ComputeId)
	if err != nil {
		fmt.Println("Error getting numerix config: ", err)
	} else {
		numerixConfig.TestResults = json.RawMessage(`{"is_functionally_tested": true}`)
		err = g.NumerixConfigRepo.Update(&numerixConfig)
		if err != nil {
			fmt.Println("Error updating numerix config: ", err)
		}
	}

	return response, nil
}
