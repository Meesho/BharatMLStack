package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
	predatorHandler "github.com/Meesho/BharatMLStack/horizon/internal/predator/handler"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Config interface {
	FetchModelConfig(ctx *gin.Context)
	OnboardModel(ctx *gin.Context)
	PromoteModel(ctx *gin.Context)
	ScaleUpModel(ctx *gin.Context)
	EditModel(ctx *gin.Context)
	DeleteModel(ctx *gin.Context)
	GetModels(ctx *gin.Context)
	ProcessRequest(ctx *gin.Context)
	GetAllPredatorRequests(ctx *gin.Context)
	Validate(ctx *gin.Context)
	GenerateFunctionalTestRequest(ctx *gin.Context)
	ExecuteFunctionalTestRequest(ctx *gin.Context)
	SendLoadTestRequest(ctx *gin.Context)
	GetSourceModels(ctx *gin.Context)
	UploadJSONToGCS(ctx *gin.Context)
	UploadModelFolder(ctx *gin.Context)
	GetFeatureTypes(ctx *gin.Context)
}

const (
	modelPathKey = "model_path"
)

var (
	configController Config
	predatorOnce     sync.Once
)

type V1 struct {
	config predatorHandler.Config
}

func NewConfigController() Config {
	if configController == nil {
		predatorOnce.Do(func() {
			config, err := predatorHandler.NewPredator(1)
			if err != nil {
				log.Panic().Err(err).Msg("Failed to create predator handler")
			}
			configController = &V1{
				config: config,
			}
		})
	}
	return configController
}

func (p V1) FetchModelConfig(ctx *gin.Context) {
	modelPath := ctx.Query(modelPathKey)
	req := predatorHandler.FetchModelConfigRequest{ModelPath: modelPath}

	resp, status, err := p.config.FetchModelConfig(req)
	if err != nil {
		ctx.JSON(status, gin.H{constant.Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

func (p V1) OnboardModel(ctx *gin.Context) {
	var req predatorHandler.ModelRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{constant.Error: "invalid request body"})
		return
	}

	resp, status, err := p.config.HandleModelRequest(req, predatorHandler.OnboardRequestType)
	if err != nil {
		ctx.JSON(status, gin.H{constant.Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		constant.Error: nil,
		constant.Data: gin.H{
			"message": resp,
			"status":  true,
		},
	})
}

func (p V1) PromoteModel(ctx *gin.Context) {
	var req predatorHandler.ModelRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{constant.Error: "invalid request body"})
		return
	}

	resp, status, err := p.config.HandleModelRequest(req, predatorHandler.PromoteRequestType)
	if err != nil {
		ctx.JSON(status, gin.H{constant.Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		constant.Error: nil,
		constant.Data: gin.H{
			"message": resp,
			"status":  true,
		},
	})
}

func (p V1) ScaleUpModel(ctx *gin.Context) {
	var req predatorHandler.ModelRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{constant.Error: "invalid request body"})
		return
	}

	resp, status, err := p.config.HandleModelRequest(req, predatorHandler.ScaleUpRequestType)
	if err != nil {
		ctx.JSON(status, gin.H{constant.Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		constant.Error: nil,
		constant.Data: gin.H{
			"message": resp,
			"status":  true,
		},
	})
}

func (p V1) EditModel(ctx *gin.Context) {
	var req predatorHandler.ModelRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{constant.Error: "invalid request body"})
		return
	}

	createdBy := ctx.GetString("email")
	if createdBy == "" {
		log.Error().Msg("Unauthenticated user trying to edit model")
		ctx.JSON(http.StatusUnauthorized, gin.H{constant.Error: "unauthenticated user"})
		return
	}

	resp, status, err := p.config.HandleEditModel(req, createdBy)
	if err != nil {
		ctx.JSON(status, gin.H{constant.Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		constant.Error: nil,
		constant.Data: gin.H{
			"message": resp,
			"status":  true,
		},
	})
}

func (p V1) DeleteModel(ctx *gin.Context) {

	var deleteRequest predatorHandler.DeleteRequest

	if err := ctx.ShouldBindJSON(&deleteRequest); err != nil || len(deleteRequest.IDs) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{constant.Error: "missing or invalid model ids"})
		return
	}

	createdBy := ctx.GetString("email")
	if createdBy == "" {
		log.Error().Msg("Unauthenticated user trying to delete model")
		ctx.JSON(http.StatusUnauthorized, gin.H{constant.Error: "unauthenticated user"})
		return
	}

	msg, groupId, status, err := p.config.HandleDeleteModel(deleteRequest, createdBy)
	if err != nil {
		ctx.JSON(status, gin.H{constant.Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		constant.Error: nil,
		constant.Data: gin.H{
			"message": fmt.Sprintf("%s with group id %d.", msg, groupId),
		},
	})
}

func (p V1) GetModels(ctx *gin.Context) {
	models, err := p.config.FetchModels()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{constant.Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		constant.Error: nil,
		constant.Data:  models,
	})
}

func (p V1) ProcessRequest(ctx *gin.Context) {
	var req predatorHandler.ApproveRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{constant.Error: "Invalid request"})
		return
	}

	if err := p.config.ProcessRequest(req); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error ": err.Error(), "group id": req.GroupID})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		constant.Data: gin.H{
			"message": "Request processing initiated asynchronously. for group id " + req.GroupID,
		},
	})
}

func (p V1) GetAllPredatorRequests(ctx *gin.Context) {
	role := ctx.GetString("role")
	email := ctx.GetString("email")

	responses, err := p.config.FetchAllPredatorRequests(role, email)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			constant.Error: "Failed to fetch predator requests",
			"details":      err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		constant.Error: nil,
		constant.Data:  responses,
	})
}

func (p V1) Validate(ctx *gin.Context) {
	groupId := ctx.Param("group_id")
	message, statusCode := p.config.ValidateRequest(groupId)
	ctx.JSON(statusCode, gin.H{
		constant.Error: nil,
		constant.Data:  message,
	})
}

func (p V1) GenerateFunctionalTestRequest(ctx *gin.Context) {
	var req predatorHandler.RequestGenerationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	response, err := p.config.GenerateFunctionalTestRequest(req)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (p V1) ExecuteFunctionalTestRequest(ctx *gin.Context) {
	var req predatorHandler.ExecuteRequestFunctionalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	response, err := p.config.ExecuteFunctionalTestRequest(req)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (p V1) SendLoadTestRequest(ctx *gin.Context) {
	var req predatorHandler.ExecuteRequestLoadTest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	response, err := p.config.SendLoadTestRequest(req)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (p V1) GetSourceModels(ctx *gin.Context) {
	response, err := p.config.GetGCSModels()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get GCS folders: %v", err),
			"data":  nil,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"error": nil,
		"data":  response,
	})
}

type UploadJSONRequest struct {
	GCSPath string `json:"gcs_path" binding:"required"`
}

func (p V1) UploadJSONToGCS(ctx *gin.Context) {
	// Read GCS path from multipart form field
	gcsPath := ctx.PostForm("gcs_path")
	if gcsPath == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "GCS path is required"})
		return
	}

	// Get the uploaded file header
	header, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Open the uploaded file
	file, err := header.Open()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file"})
		return
	}
	defer file.Close()

	// Read file contents
	data, err := io.ReadAll(file)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// Validate JSON
	if !json.Valid(data) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON file"})
		return
	}

	// Parse GCS path
	parts := strings.SplitN(strings.TrimPrefix(gcsPath, "gs://"), "/", 2)
	if len(parts) != 2 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid GCS path format. Expected: gs://bucket/path"})
		return
	}

	bucket := parts[0]
	objectPath := path.Join(parts[1], header.Filename)

	_, err = p.config.UploadJSONToGCS(bucket, objectPath, data)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload file to GCS: %v", err)})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "File uploaded successfully",
		"path":    fmt.Sprintf("gs://%s/%s", bucket, objectPath),
	})
}

func (p V1) UploadModelFolder(ctx *gin.Context) {
	// Parse request body as array of model upload items
	var req predatorHandler.UploadModelFolderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{constant.Error: "invalid request body: " + err.Error()})
		return
	}

	// Validate request_type
	if req.RequestType != "create" && req.RequestType != "update" {
		ctx.JSON(http.StatusBadRequest, gin.H{constant.Error: "request_type must be either 'create' or 'update'"})
		return
	}

	// Validate that we have at least one model to upload
	if len(req.Models) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{constant.Error: "at least one model is required"})
		return
	}

	// Get is_partial query parameter
	isPartial := ctx.Query("is_partial") == "true"

	// Extract authorization token from headers
	authToken := ctx.GetHeader("Authorization")
	if authToken == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{constant.Error: "Authorization header is required"})
		return
	}

	// Remove "Bearer " prefix if present
	authToken = strings.TrimPrefix(authToken, "Bearer ")

	// Call handler to process the upload
	resp, status, err := p.config.UploadModelFolderFromLocal(req, isPartial, authToken)
	if err != nil {
		ctx.JSON(status, gin.H{constant.Error: err.Error()})
		return
	}

	ctx.JSON(status, gin.H{
		constant.Error: nil,
		constant.Data:  resp,
	})
}

// GetFeatureTypes returns all available feature types for dropdown selection
func (p V1) GetFeatureTypes(ctx *gin.Context) {
	featureTypes := []string{
		"ONLINE_FEATURE",
		"PARENT_ONLINE_FEATURE",
		"OFFLINE_FEATURE",
		"PARENT_OFFLINE_FEATURE",
		"RTP_FEATURE",
		"PARENT_RTP_FEATURE",
		"DEFAULT_FEATURE",
		"PARENT_DEFAULT_FEATURE",
		"MODEL_FEATURE",
		"CALIBRATION",
		"PCTR_CALIBRATION",
		"PCVR_CALIBRATION",
	}

	ctx.JSON(http.StatusOK, gin.H{
		constant.Error: nil,
		constant.Data:  featureTypes,
	})
}
