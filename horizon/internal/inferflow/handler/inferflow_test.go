package handler

import (
	"errors"
	"testing"

	service_deployable_config "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestInferFlow_GetLoggingTTL(t *testing.T) {
	m := &InferFlow{}
	got, err := m.GetLoggingTTL()
	require.NoError(t, err)
	require.NotNil(t, got.Data)
	assert.Equal(t, []int{30, 60, 90}, got.Data)
}

func TestInferFlow_GenerateFunctionalTestRequest_InvalidBatchSize(t *testing.T) {
	m := &InferFlow{}
	req := GenerateRequestFunctionalTestingRequest{
		Entity:          "user",
		BatchSize:       "not_a_number",
		ModelConfigID:    "cfg-1",
		DefaultFeatures:  map[string]string{},
	}
	_, err := m.GenerateFunctionalTestRequest(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid batch size")
}

func TestInferFlow_GenerateFunctionalTestRequest_Valid(t *testing.T) {
	m := &InferFlow{}
	req := GenerateRequestFunctionalTestingRequest{
		Entity:          "user",
		BatchSize:       "5",
		ModelConfigID:   "cfg-1",
		DefaultFeatures: map[string]string{"f1": "v1"},
		MetaData:        map[string]string{"k": "v"},
	}
	got, err := m.GenerateFunctionalTestRequest(req)
	require.NoError(t, err)
	assert.Equal(t, "cfg-1", got.RequestBody.ModelConfigID)
	assert.Len(t, got.RequestBody.Entities, 1)
	assert.Equal(t, "user_id", got.RequestBody.Entities[0].Entity)
	assert.Len(t, got.RequestBody.Entities[0].Ids, 5)
	assert.Equal(t, "v", got.MetaData["k"])
	// Default feature f1=v1 should produce one FeatureValue with 5 elements
	var found bool
	for _, fv := range got.RequestBody.Entities[0].Features {
		if fv.Name == "f1" {
			found = true
			assert.Len(t, fv.IdsFeatureValue, 5)
			for _, v := range fv.IdsFeatureValue {
				assert.Equal(t, "v1", v)
			}
			break
		}
	}
	assert.True(t, found)
}

func TestInferFlow_batchFetchDiscoveryConfigs_Empty(t *testing.T) {
	m := &InferFlow{}
	discoveryMap, serviceDeployableMap, err := m.batchFetchDiscoveryConfigs(nil)
	require.NoError(t, err)
	assert.Empty(t, discoveryMap)
	assert.Empty(t, serviceDeployableMap)
}

func TestInferFlow_GetDerivedConfigID_EmptyDeployableTag(t *testing.T) {
	mockRepo := &mockServiceDeployableRepo{
		getById: func(id int) (*service_deployable_config.ServiceDeployableConfig, error) {
			return &service_deployable_config.ServiceDeployableConfig{
				ID:            id,
				Name:          "svc",
				DeployableTag: "",
			}, nil
		},
	}
	m := &InferFlow{ServiceDeployableConfigRepo: mockRepo}
	got, err := m.GetDerivedConfigID("base-config", 1)
	require.NoError(t, err)
	assert.Equal(t, "base-config", got)
}

func TestInferFlow_GetDerivedConfigID_WithDeployableTag(t *testing.T) {
	mockRepo := &mockServiceDeployableRepo{
		getById: func(id int) (*service_deployable_config.ServiceDeployableConfig, error) {
			return &service_deployable_config.ServiceDeployableConfig{
				ID:            id,
				DeployableTag: "tag1",
			}, nil
		},
	}
	m := &InferFlow{ServiceDeployableConfigRepo: mockRepo}
	got, err := m.GetDerivedConfigID("base-config", 1)
	require.NoError(t, err)
	assert.Equal(t, "base-config_tag1_scaleup", got)
}

func TestInferFlow_GetDerivedConfigID_RepoError(t *testing.T) {
	mockRepo := &mockServiceDeployableRepo{
		getById: func(id int) (*service_deployable_config.ServiceDeployableConfig, error) {
			return nil, errors.New("db error")
		},
	}
	m := &InferFlow{ServiceDeployableConfigRepo: mockRepo}
	_, err := m.GetDerivedConfigID("base-config", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch")
}

// mockServiceDeployableRepo implements service_deployable_config.ServiceDeployableRepository for tests.
type mockServiceDeployableRepo struct {
	getById func(id int) (*service_deployable_config.ServiceDeployableConfig, error)
}

func (m *mockServiceDeployableRepo) GetById(id int) (*service_deployable_config.ServiceDeployableConfig, error) {
	if m.getById != nil {
		return m.getById(id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockServiceDeployableRepo) Create(_ *service_deployable_config.ServiceDeployableConfig) error {
	return nil
}
func (m *mockServiceDeployableRepo) Update(_ *service_deployable_config.ServiceDeployableConfig) error {
	return nil
}
func (m *mockServiceDeployableRepo) DeactivateServiceDeployable(_ int, _ string) error {
	return nil
}
func (m *mockServiceDeployableRepo) GetByService(_ string) ([]service_deployable_config.ServiceDeployableConfig, error) {
	return nil, nil
}
func (m *mockServiceDeployableRepo) GetAllActive() ([]service_deployable_config.ServiceDeployableConfig, error) {
	return nil, nil
}
func (m *mockServiceDeployableRepo) GetByWorkflowStatus(_ string) ([]service_deployable_config.ServiceDeployableConfig, error) {
	return nil, nil
}
func (m *mockServiceDeployableRepo) GetByDeployableHealth(_ string) ([]service_deployable_config.ServiceDeployableConfig, error) {
	return nil, nil
}
func (m *mockServiceDeployableRepo) GetByNameAndService(_, _ string) (*service_deployable_config.ServiceDeployableConfig, error) {
	return nil, nil
}
func (m *mockServiceDeployableRepo) GetByIds(_ []int) ([]service_deployable_config.ServiceDeployableConfig, error) {
	return nil, nil
}
func (m *mockServiceDeployableRepo) GetTestDeployableIDByNodePool(_ string) (int, error) {
	return 0, gorm.ErrRecordNotFound
}
