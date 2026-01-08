package handler

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/validationjob"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/validationlock"
)

type Config interface {
	FetchModelConfig(req FetchModelConfigRequest) (ModelParamsResponse, int, error)
	HandleModelRequest(req ModelRequest, requestType string) (string, int, error)
	HandleEditModel(req ModelRequest, createdBy string) (string, int, error)
	HandleDeleteModel(deleteRequest DeleteRequest, createdBy string) (string, uint, int, error)
	FetchModels() ([]ModelResponse, error)
	ProcessRequest(req ApproveRequest) error
	FetchAllPredatorRequests(role, email string) ([]map[string]interface{}, error)
	ValidateRequest(groupId string) (string, int)
	CleanupExpiredValidationLocks() error
	GetValidationStatus() (bool, *validationlock.Table, error)
	GetValidationJobStatus(groupId string) (*validationjob.Table, error)
	GenerateFunctionalTestRequest(req RequestGenerationRequest) (RequestGenerationResponse, error)
	ExecuteFunctionalTestRequest(req ExecuteRequestFunctionalRequest) (ExecuteRequestFunctionalResponse, error)
	SendLoadTestRequest(req ExecuteRequestLoadTest) (PhoenixClientResponse, error)
	GetGCSModels() (*GCSFoldersResponse, error)
	UploadJSONToGCS(bucketName, fileName string, data []byte) (string, error)
	UploadModelFolderFromLocal(req UploadModelFolderRequest, isPartial bool, authToken string) (UploadModelFolderResponse, int, error)
	CheckModelExists(bucket, path string) (bool, error)
	UploadFileToGCS(bucket, path string, data []byte) error
}
