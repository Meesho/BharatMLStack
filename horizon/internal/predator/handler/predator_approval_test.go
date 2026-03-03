package handler

import (
	"encoding/json"
	"fmt"
	"path"
	"sync"
	"testing"

	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	pred "github.com/Meesho/BharatMLStack/horizon/internal/predator"
	"github.com/Meesho/BharatMLStack/horizon/internal/predator/proto/protogen"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/predatorrequest"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/prototext"
	"gorm.io/gorm"
)

// approvalMockGCSClient records ReadFile/UploadFile for instance-count update tests.
type approvalMockGCSClient struct {
	readFileFunc   func(bucket, objectPath string) ([]byte, error)
	uploadFileFunc func(bucket, objectPath string, data []byte) error
	// uploadCalls records (bucket, path) and uploaded data for assertions
	uploadCalls []uploadCall
	mu          sync.Mutex
}

type uploadCall struct {
	Bucket string
	Path  string
	Data  []byte
}

func (m *approvalMockGCSClient) ReadFile(bucket, objectPath string) ([]byte, error) {
	if m.readFileFunc != nil {
		return m.readFileFunc(bucket, objectPath)
	}
	return nil, fmt.Errorf("ReadFile not mocked")
}

func (m *approvalMockGCSClient) UploadFile(bucket, objectPath string, data []byte) error {
	m.mu.Lock()
	m.uploadCalls = append(m.uploadCalls, uploadCall{Bucket: bucket, Path: objectPath, Data: data})
	m.mu.Unlock()
	if m.uploadFileFunc != nil {
		return m.uploadFileFunc(bucket, objectPath, data)
	}
	return nil
}

func (m *approvalMockGCSClient) TransferFolder(_, _, _, _, _, _ string) error          { return nil }
func (m *approvalMockGCSClient) TransferAndDeleteFolder(_, _, _, _, _, _ string) error { return nil }
func (m *approvalMockGCSClient) TransferFolderWithSplitSources(_, _, _, _, _, _, _, _ string) error {
	return nil
}
func (m *approvalMockGCSClient) DeleteFolder(_, _, _ string) error           { return nil }
func (m *approvalMockGCSClient) ListFolders(_, _ string) ([]string, error)   { return nil, nil }
func (m *approvalMockGCSClient) CheckFileExists(_, _ string) (bool, error)    { return false, nil }
func (m *approvalMockGCSClient) CheckFolderExists(_, _ string) (bool, error)  { return false, nil }
func (m *approvalMockGCSClient) UploadFolderFromLocal(_, _, _ string) error  { return nil }
func (m *approvalMockGCSClient) GetFolderInfo(_, _ string) (*externalcall.GCSFolderInfo, error) {
	return nil, nil
}
func (m *approvalMockGCSClient) ListFoldersWithTimestamp(_, _ string) ([]externalcall.GCSFolderInfo, error) {
	return nil, nil
}
func (m *approvalMockGCSClient) FindFileWithSuffix(_, _, _ string) (bool, string, error) {
	return false, "", nil
}
func (m *approvalMockGCSClient) ListFilesWithSuffix(_, _, _ string) ([]string, error) { return nil, nil }

// validConfigPbtxt returns protobuf text for ModelConfig with one instance group count.
func validConfigPbtxt(count int32) []byte {
	cfg := &protogen.ModelConfig{
		InstanceGroup: []*protogen.ModelInstanceGroup{
			{Count: count},
		},
	}
	data, err := prototext.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	return data
}

func TestPredator_UpdateInstanceCountInConfigSource(t *testing.T) {
	const bucket, basePath, modelName = "cfg-bucket", "cfg/base", "my_model"
	const newCount = 2

	readFileCalls := 0
	mock := &approvalMockGCSClient{
		readFileFunc: func(b, p string) ([]byte, error) {
			readFileCalls++
			expectPath := path.Join(basePath, modelName, configFile)
			assert.Equal(t, bucket, b)
			assert.Equal(t, expectPath, p)
			return validConfigPbtxt(1), nil
		},
	}

	p := &Predator{GcsClient: mock}
	err := p.updateInstanceCountInConfigSource(bucket, basePath, modelName, newCount)
	require.NoError(t, err)

	require.Len(t, mock.uploadCalls, 1)
	uc := mock.uploadCalls[0]
	assert.Equal(t, bucket, uc.Bucket)
	assert.Equal(t, path.Join(basePath, modelName, configFile), uc.Path)

	var uploaded protogen.ModelConfig
	err = prototext.Unmarshal(uc.Data, &uploaded)
	require.NoError(t, err)
	require.Len(t, uploaded.InstanceGroup, 1)
	assert.Equal(t, int32(newCount), uploaded.InstanceGroup[0].Count)
	assert.Equal(t, 1, readFileCalls)
}

func TestPredator_UpdateInstanceCountInConfigSource_UnchangedSkipsUpload(t *testing.T) {
	const bucket, basePath, modelName = "b", "p", "m"
	const count = 1

	mock := &approvalMockGCSClient{
		readFileFunc: func(_, _ string) ([]byte, error) {
			return validConfigPbtxt(count), nil
		},
	}

	p := &Predator{GcsClient: mock}
	err := p.updateInstanceCountInConfigSource(bucket, basePath, modelName, count)
	require.NoError(t, err)

	// Count unchanged -> no upload
	assert.Len(t, mock.uploadCalls, 0)
}

func TestPredator_ProcessGCSCloneStage_Promote_Production_UpdatesInstanceCount(t *testing.T) {
	// Set production and config-source so the Promote instance-count path runs
	saveAppEnv := pred.AppEnv
	saveGcsConfigBucket := pred.GcsConfigBucket
	saveGcsConfigBasePath := pred.GcsConfigBasePath
	saveGcsModelBucket := pred.GcsModelBucket
	saveGcsModelBasePath := pred.GcsModelBasePath
	defer func() {
		pred.AppEnv = saveAppEnv
		pred.GcsConfigBucket = saveGcsConfigBucket
		pred.GcsConfigBasePath = saveGcsConfigBasePath
		pred.GcsModelBucket = saveGcsModelBucket
		pred.GcsModelBasePath = saveGcsModelBasePath
	}()

	pred.AppEnv = "prd"
	pred.GcsConfigBucket = "cfg-bucket"
	pred.GcsConfigBasePath = "cfg-path"
	pred.GcsModelBucket = "model-bucket"
	pred.GcsModelBasePath = "model-path"

	const requestID uint = 1
	const deployableID = 42
	const instanceCount = 2
	payload := Payload{
		ModelName:   "promoted_model",
		ModelSource: "gs://model-bucket/model-path/source_model",
		MetaData:    MetaData{InstanceCount: instanceCount},
		ConfigMapping: ConfigMapping{ServiceDeployableID: uint(deployableID)},
	}
	payloadBytes, _ := json.Marshal(payload)

	deployableConfig := PredatorDeployableConfig{GCSBucketPath: "gs://dest-bucket/dest-path/"}
	deployableConfigBytes, _ := json.Marshal(deployableConfig)

	mockGCS := &approvalMockGCSClient{
		readFileFunc: func(bucket, objectPath string) ([]byte, error) {
			// Config-source read for updateInstanceCountInConfigSource
			return validConfigPbtxt(1), nil
		},
	}

	mockDeployableRepo := &predatorMockServiceDeployableRepo{
		getById: func(id int) (*servicedeployableconfig.ServiceDeployableConfig, error) {
			if id != deployableID {
				return nil, fmt.Errorf("unexpected id %d", id)
			}
			return &servicedeployableconfig.ServiceDeployableConfig{
				ID:     id,
				Config: deployableConfigBytes,
			}, nil
		},
	}

	requestIdPayloadMap := map[uint]*Payload{requestID: &payload}
	predatorRequestList := []predatorrequest.PredatorRequest{
		{
			RequestID:   requestID,
			ModelName:  payload.ModelName,
			RequestType: PromoteRequestType,
			Payload:    string(payloadBytes),
			RequestStage: predatorStagePending,
		},
	}
	req := ApproveRequest{ApprovedBy: "test@test.com"}

	// Mock repo so updateRequestStatusAndStage does not panic
	mockRequestRepo := &approvalMockRequestRepo{updateMany: func(_ []predatorrequest.PredatorRequest) error { return nil }}

	p := &Predator{
		GcsClient:             mockGCS,
		ServiceDeployableRepo: mockDeployableRepo,
		Repo:                  mockRequestRepo,
	}

	transferred, err := p.processGCSCloneStage(requestIdPayloadMap, predatorRequestList, req)
	require.NoError(t, err)

	// Instance count update should have written to config-source (UploadFile called)
	require.GreaterOrEqual(t, len(mockGCS.uploadCalls), 1, "expected at least one UploadFile call for instance count update")
	var found bool
	for _, uc := range mockGCS.uploadCalls {
		var cfg protogen.ModelConfig
		if err := prototext.Unmarshal(uc.Data, &cfg); err != nil || len(cfg.InstanceGroup) == 0 {
			continue
		}
		if cfg.InstanceGroup[0].Count == int32(instanceCount) {
			found = true
			break
		}
	}
	assert.True(t, found, "expected UploadFile with instance_count %d", instanceCount)
	assert.Len(t, transferred, 1)
	assert.Equal(t, "dest-bucket", transferred[0].Bucket)
	assert.True(t, transferred[0].Path == "dest-path" || transferred[0].Path == "dest-path/", "Path: %s", transferred[0].Path)
	assert.Equal(t, payload.ModelName, transferred[0].Name)
}

func TestPredator_ProcessGCSCloneStage_Onboard_Production_DoesNotUpdateInstanceCount(t *testing.T) {
	saveAppEnv := pred.AppEnv
	saveGcsConfigBucket := pred.GcsConfigBucket
	saveGcsConfigBasePath := pred.GcsConfigBasePath
	saveGcsModelBucket := pred.GcsModelBucket
	saveGcsModelBasePath := pred.GcsModelBasePath
	defer func() {
		pred.AppEnv = saveAppEnv
		pred.GcsConfigBucket = saveGcsConfigBucket
		pred.GcsConfigBasePath = saveGcsConfigBasePath
		pred.GcsModelBucket = saveGcsModelBucket
		pred.GcsModelBasePath = saveGcsModelBasePath
	}()

	pred.AppEnv = "prd"
	pred.GcsConfigBucket = "cfg-bucket"
	pred.GcsConfigBasePath = "cfg-path"
	pred.GcsModelBucket = "model-bucket"
	pred.GcsModelBasePath = "model-path"

	const requestID uint = 1
	const deployableID = 42
	payload := Payload{
		ModelName:   "onboard_model",
		ModelSource: "gs://model-bucket/model-path/onboard_model",
		MetaData:    MetaData{InstanceCount: 1},
		ConfigMapping: ConfigMapping{ServiceDeployableID: uint(deployableID)},
	}
	payloadBytes, _ := json.Marshal(payload)

	deployableConfig := PredatorDeployableConfig{GCSBucketPath: "gs://dest-bucket/dest-path/"}
	deployableConfigBytes, _ := json.Marshal(deployableConfig)

	mockGCS := &approvalMockGCSClient{
		readFileFunc: func(_, _ string) ([]byte, error) {
			t.Error("ReadFile should not be called for Onboard (no instance count update)")
			return nil, fmt.Errorf("unexpected ReadFile")
		},
	}

	mockDeployableRepo := &predatorMockServiceDeployableRepo{
		getById: func(id int) (*servicedeployableconfig.ServiceDeployableConfig, error) {
			if id != deployableID {
				return nil, fmt.Errorf("unexpected id %d", id)
			}
			return &servicedeployableconfig.ServiceDeployableConfig{
				ID:     id,
				Config: deployableConfigBytes,
			}, nil
		},
	}

	requestIdPayloadMap := map[uint]*Payload{requestID: &payload}
	predatorRequestList := []predatorrequest.PredatorRequest{
		{
			RequestID:    requestID,
			ModelName:    payload.ModelName,
			RequestType:  OnboardRequestType,
			Payload:     string(payloadBytes),
			RequestStage: predatorStagePending,
		},
	}
	req := ApproveRequest{ApprovedBy: "test@test.com"}

	mockRequestRepo := &approvalMockRequestRepo{updateMany: func(_ []predatorrequest.PredatorRequest) error { return nil }}

	p := &Predator{
		GcsClient:             mockGCS,
		ServiceDeployableRepo: mockDeployableRepo,
		Repo:                  mockRequestRepo,
	}

	_, err := p.processGCSCloneStage(requestIdPayloadMap, predatorRequestList, req)
	require.NoError(t, err)

	// Onboard in production does not call updateInstanceCountInConfigSource, so no config UploadFile
	assert.Len(t, mockGCS.uploadCalls, 0, "Onboard should not update instance count in config-source")
}

// approvalMockRequestRepo implements predatorrequest.PredatorRequestRepository for processGCSCloneStage tests.
type approvalMockRequestRepo struct {
	updateMany func([]predatorrequest.PredatorRequest) error
}

func (m *approvalMockRequestRepo) UpdateMany(reqs []predatorrequest.PredatorRequest) error {
	if m.updateMany != nil {
		return m.updateMany(reqs)
	}
	return nil
}

func (m *approvalMockRequestRepo) GetAll() ([]predatorrequest.PredatorRequest, error)           { return nil, nil }
func (m *approvalMockRequestRepo) GetByID(uint) (*predatorrequest.PredatorRequest, error)       { return nil, nil }
func (m *approvalMockRequestRepo) GetAllByEmail(string) ([]predatorrequest.PredatorRequest, error) { return nil, nil }
func (m *approvalMockRequestRepo) Create(*predatorrequest.PredatorRequest) error                { return nil }
func (m *approvalMockRequestRepo) Update(*predatorrequest.PredatorRequest) error                { return nil }
func (m *approvalMockRequestRepo) Delete(uint) error                                            { return nil }
func (m *approvalMockRequestRepo) DB() *gorm.DB                                               { return nil }
func (m *approvalMockRequestRepo) UpdateStatusAndStage(*gorm.DB, *predatorrequest.PredatorRequest) error {
	return nil
}
func (m *approvalMockRequestRepo) ActiveModelRequestExistForRequestType([]string, string) (bool, error) {
	return false, nil
}
func (m *approvalMockRequestRepo) WithTx(*gorm.DB) predatorrequest.PredatorRequestRepository { return m }
func (m *approvalMockRequestRepo) CreateMany([]predatorrequest.PredatorRequest) error            { return nil }
func (m *approvalMockRequestRepo) GetAllByGroupID(uint) ([]predatorrequest.PredatorRequest, error) {
	return nil, nil
}
