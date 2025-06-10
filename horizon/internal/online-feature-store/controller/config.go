package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	authHandler "github.com/Meesho/BharatMLStack/horizon/internal/auth/handler"
	"github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/handler"
	"github.com/Meesho/BharatMLStack/horizon/pkg/api"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Config interface {
	RegisterStore(ctx *gin.Context)
	RegisterEntity(ctx *gin.Context)
	RegisterFeatureGroup(ctx *gin.Context)
	RegisterJob(ctx *gin.Context)
	GetEntities(ctx *gin.Context)
	GetJobs(ctx *gin.Context)
	GetStores(ctx *gin.Context)
	GetConfig(ctx *gin.Context)
	GetFeatureGroupLabels(ctx *gin.Context)
	AddFeatures(ctx *gin.Context)
	DeleteFeatures(ctx *gin.Context)
	RetrieveEntities(ctx *gin.Context)
	RetrieveFeatureGroups(ctx *gin.Context)
	ProcessStore(ctx *gin.Context)
	ProcessEntity(ctx *gin.Context)
	ProcessFeatureGroup(ctx *gin.Context)
	ProcessJob(ctx *gin.Context)
	ProcessAddFeatures(ctx *gin.Context)
	ProcessDeleteFeatures(ctx *gin.Context)
	GetAllEntitiesRequestForUser(ctx *gin.Context)
	GetAllFeatureGroupsRequestForUser(ctx *gin.Context)
	GetAllJobsRequestForUser(ctx *gin.Context)
	GetAllStoresRequestForUser(ctx *gin.Context)
	GetAllFeaturesRequestForUser(ctx *gin.Context)
	RetrieveSourceMapping(ctx *gin.Context)
	EditEntity(ctx *gin.Context)
	EditFeatureGroup(ctx *gin.Context)
	EditFeatures(ctx *gin.Context)
	GetOnlineFeatureMapping(ctx *gin.Context)
}

var (
	configController Config
	once             sync.Once
)

type V1 struct {
	Config handler.Config
}

func NewConfigController() Config {
	if configController == nil {
		once.Do(func() {
			configController = &V1{
				Config: handler.NewConfigHandler(1),
			}
		})
	}
	return configController
}

func (v *V1) RegisterStore(ctx *gin.Context) {
	// Proceed with the request
	var request handler.RegisterStoreRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	request.UserId = email
	request.Role = role

	requestId, apiError := v.Config.RegisterStore(&request)
	if apiError != nil {
		log.Error().Err(apiError).Msg("Error in registering store")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Store Creation Request Registered successfully with RequestId %v", requestId)})
}

func (v *V1) RegisterEntity(ctx *gin.Context) {
	var request handler.RegisterEntityRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	request.UserId = email
	request.Role = role

	requestId, apiError := v.Config.RegisterEntity(&request)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Entity Creation Request Registered successfully with RequestId %v", requestId)})
}

func (v *V1) EditEntity(ctx *gin.Context) {
	var request handler.EditEntityRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	request.UserId = email
	request.Role = role

	requestId, apiError := v.Config.EditEntity(&request)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Entity Creation Request Registered successfully with RequestId %v", requestId)})
}

func (v *V1) RegisterFeatureGroup(ctx *gin.Context) {
	var request handler.RegisterFeatureGroupRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	request.UserId = email
	request.Role = role
	requestId, apiError := v.Config.RegisterFeatureGroup(&request)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Feature Group Creation Request Registered successfully with RequestId %v", requestId)})
}

func (v *V1) EditFeatureGroup(ctx *gin.Context) {
	var request handler.EditFeatureGroupRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	request.UserId = email
	request.Role = role
	requestId, apiError := v.Config.EditFeatureGroup(&request)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Feature Group Creation Request Registered successfully with RequestId %v", requestId)})
}

func (v *V1) RegisterJob(ctx *gin.Context) {
	var request handler.RegisterJobRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	request.UserId = email
	request.Role = role
	requestId, apiError := v.Config.RegisterJob(&request)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Job Creation Request Registered successfully with RequestId %v", requestId)})
}

func (v *V1) GetEntities(ctx *gin.Context) {
	entityIds, err := v.Config.GetAllEntities()
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, entityIds)
}

func (v *V1) GetJobs(ctx *gin.Context) {
	jobType := ctx.DefaultQuery("jobType", "")
	jobs, err := v.Config.GetJobsByJobType(jobType)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, jobs)
}

func (v *V1) GetStores(ctx *gin.Context) {
	stores, err := v.Config.GetStores()
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, stores)
}

func (v *V1) GetConfig(ctx *gin.Context) {
	configs, err := v.Config.GetConfig()
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, configs)
}

func (v *V1) GetFeatureGroupLabels(ctx *gin.Context) {
	entityLabel := ctx.DefaultQuery("entityLabel", "")
	if entityLabel == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "EntityLabel is required"})
		return
	}

	featureGroupLabels, err := v.Config.GetFeatureGroupLabelsForEntity(entityLabel)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, featureGroupLabels)
}

func (v *V1) AddFeatures(ctx *gin.Context) {
	var request handler.AddFeatureRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	request.UserId = email
	request.Role = role
	requestId, err := v.Config.AddFeatures(&request)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Feature Addition Request Registered successfully with RequestId %v", requestId)})
}

func (v *V1) DeleteFeatures(ctx *gin.Context) {
	var request handler.DeleteFeaturesRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	request.UserId = email
	request.Role = role
	requestId, err := v.Config.DeleteFeatures(&request)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Feature Deletion Request Registered successfully with RequestId %v", requestId)})
}

func (v *V1) EditFeatures(ctx *gin.Context) {
	var request handler.EditFeatureRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	request.UserId = email
	request.Role = role
	requestId, err := v.Config.EditFeatures(&request)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Feature Addition Request Registered successfully with RequestId %v", requestId)})
}

func (v *V1) RetrieveEntities(ctx *gin.Context) {
	entities, err := v.Config.RetrieveEntities()
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, entities)
}

func (v *V1) RetrieveFeatureGroups(ctx *gin.Context) {
	entityLabel := ctx.Query("entity")
	featureGroups, err := v.Config.RetrieveFeatureGroups(entityLabel)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, featureGroups)
}

func (v *V1) ProcessStore(ctx *gin.Context) {
	var request handler.ProcessStoreRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	request.ApproverId = email
	request.Role = role
	if role != "admin" {
		err = errors.New("not authorized to process request")
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if apiError := v.Config.ProcessStore(&request); apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Store Processed successfully"})
}

func (v *V1) ProcessEntity(ctx *gin.Context) {
	var request handler.ProcessEntityRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	request.ApproverId = email
	request.Role = role
	if role != "admin" {
		err = errors.New("not authorized to process request")
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if apiError := v.Config.ProcessEntity(&request); apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Entity Processed successfully"})
}

func (v *V1) ProcessFeatureGroup(ctx *gin.Context) {
	var request handler.ProcessFeatureGroupRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	request.ApproverId = email
	request.Role = role
	if role != "admin" {
		err = errors.New("not authorized to process request")
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if apiError := v.Config.ProcessFeatureGroup(&request); apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Feature Group Processed successfully"})
}

func (v *V1) ProcessJob(ctx *gin.Context) {
	var request handler.ProcessJobRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	request.ApproverId = email
	request.Role = role
	if role != "admin" {
		err = errors.New("not authorized to process request")
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if apiError := v.Config.ProcessJob(&request); apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Job Processed successfully"})
}

func (v *V1) ProcessAddFeatures(ctx *gin.Context) {
	var request handler.ProcessAddFeatureRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	request.ApproverId = email
	request.Role = role
	if role != "admin" {
		err = errors.New("not authorized to process request")
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = v.Config.ProcessAddFeature(&request)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Features Processed successfully"})
}

func (v *V1) ProcessDeleteFeatures(ctx *gin.Context) {
	var request handler.ProcessDeleteFeaturesRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	request.ApproverId = email
	request.Role = role
	if role != "admin" {
		err = errors.New("not authorized to process request")
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := v.Config.ProcessDeleteFeatures(&request); err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Feature Deletion Processed successfully"})
}

func (v *V1) GetAllEntitiesRequestForUser(ctx *gin.Context) {
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	response, apiError := v.Config.GetAllEntitiesRequest(email, role)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, response)
}

func (v *V1) GetAllFeatureGroupsRequestForUser(ctx *gin.Context) {
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	response, apiError := v.Config.GetAllFeatureGroupsRequest(email, role)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, response)
}

func (v *V1) GetAllStoresRequestForUser(ctx *gin.Context) {
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	response, apiError := v.Config.GetAllStoresRequest(email, role)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, response)
}

func (v *V1) GetAllJobsRequestForUser(ctx *gin.Context) {
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	response, apiError := v.Config.GetAllJobsRequest(email, role)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, response)
}

func (v *V1) GetAllFeaturesRequestForUser(ctx *gin.Context) {
	email, role, err := ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	response, apiError := v.Config.GetAllFeaturesRequest(email, role)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, response)
}

func (v *V1) RetrieveSourceMapping(ctx *gin.Context) {
	//Get query param jobId and jobToken from ctx
	jobId := ctx.Query("jobId")
	jobToken := ctx.Query("jobToken")
	sourceMapping, err := v.Config.RetrieveSourceMapping(jobId, jobToken)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, sourceMapping)
}

func (v *V1) GetOnlineFeatureMapping(ctx *gin.Context) {
	var request handler.GetOnlineFeatureMappingRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	response, err := v.Config.GetOnlineFeatureMapping(request)
	if err != nil {
		ctx.JSON(api.NewInternalServerError(err.Error()).StatusCode, handler.GetOnlineFeatureMappingResponse{
			Error: err.Error(),
			Data:  []string{},
		})
		return
	}
	ctx.JSON(200, response)
}

func ParseAuthenticationHeader(ctx *gin.Context) (string, string, error) {
	// Extract the Authorization header
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		log.Error().Msg("Authorization header is missing")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
		return "", "", errors.New("authorization header required")
	}

	// Extract the token from the header
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		log.Error().Msg("Invalid Authorization header format")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token must be in Bearer format"})
		return "", "", errors.New("authorization token must be in Bearer format")
	}

	// Parse the token and validate
	claims := &authHandler.Claims{} // Assuming Claims struct is defined in middleware
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return authHandler.JwtKey, nil // Replace with your actual JWT key
	})
	if err != nil || !token.Valid {
		log.Error().Err(err).Msg("Invalid or expired token")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
		return "", "", err
	}

	// Extract email from claims
	email := claims.Email
	role := claims.Role // Assuming the email is stored in the Username field
	log.Info().Msgf("Authenticated email: %s", email)

	return email, role, nil

}
