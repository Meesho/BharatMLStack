package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPredator_ValidateRequest_InvalidGroupIDFormat(t *testing.T) {
	p := &Predator{}
	msg, code := p.ValidateRequest("not_a_number")
	assert.Equal(t, "Invalid request ID format", msg)
	assert.Equal(t, http.StatusBadRequest, code)
}

func TestPredator_ReplaceModelNameInConfigPreservingFormat(t *testing.T) {
	p := &Predator{}
	tests := []struct {
		name          string
		data          []byte
		destModelName string
		wantContains  string
	}{
		{
			name:          "replaces top-level name",
			data:          []byte("name: \"old_model\"\n"),
			destModelName: "new_model",
			wantContains:  "name: \"new_model\"",
		},
		{
			name: "preserves nested indented name",
			data: []byte(`name: "top"
  name: "nested"
`),
			destModelName: "replaced",
			wantContains:  "name: \"replaced\"",
		},
		{
			name:          "no name field unchanged",
			data:          []byte("platform: \"tensorflow\"\n"),
			destModelName: "any",
			wantContains:  "platform: \"tensorflow\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.replaceModelNameInConfigPreservingFormat(tt.data, tt.destModelName)
			assert.Contains(t, string(got), tt.wantContains)
		})
	}
}

func TestPredator_GetDerivedModelName_NonScaleUp(t *testing.T) {
	p := &Predator{}
	payload := Payload{ModelName: "my_model", ConfigMapping: ConfigMapping{ServiceDeployableID: 1}}
	got, err := p.GetDerivedModelName(payload, OnboardRequestType)
	require.NoError(t, err)
	assert.Equal(t, "my_model", got)
}

func TestPredator_GetDerivedModelName_ScaleUp_EmptyTag(t *testing.T) {
	mockRepo := &predatorMockServiceDeployableRepo{
		getById: func(id int) (*servicedeployableconfig.ServiceDeployableConfig, error) {
			return &servicedeployableconfig.ServiceDeployableConfig{
				ID:            id,
				DeployableTag: "",
			}, nil
		},
	}
	p := &Predator{ServiceDeployableRepo: mockRepo}
	payload := Payload{ModelName: "base_model", ConfigMapping: ConfigMapping{ServiceDeployableID: 1}}
	got, err := p.GetDerivedModelName(payload, ScaleUpRequestType)
	require.NoError(t, err)
	assert.Equal(t, "base_model", got)
}

func TestPredator_GetDerivedModelName_ScaleUp_WithTag(t *testing.T) {
	mockRepo := &predatorMockServiceDeployableRepo{
		getById: func(id int) (*servicedeployableconfig.ServiceDeployableConfig, error) {
			return &servicedeployableconfig.ServiceDeployableConfig{
				ID:            id,
				DeployableTag: "tag1",
			}, nil
		},
	}
	p := &Predator{ServiceDeployableRepo: mockRepo}
	payload := Payload{ModelName: "base_model", ConfigMapping: ConfigMapping{ServiceDeployableID: 1}}
	got, err := p.GetDerivedModelName(payload, ScaleUpRequestType)
	require.NoError(t, err)
	assert.Equal(t, "base_model_tag1_scaleup", got)
}

func TestPredator_GetDerivedModelName_ScaleUp_RepoError(t *testing.T) {
	mockRepo := &predatorMockServiceDeployableRepo{
		getById: func(id int) (*servicedeployableconfig.ServiceDeployableConfig, error) {
			return nil, errors.New("db error")
		},
	}
	p := &Predator{ServiceDeployableRepo: mockRepo}
	payload := Payload{ModelName: "base_model", ConfigMapping: ConfigMapping{ServiceDeployableID: 1}}
	_, err := p.GetDerivedModelName(payload, ScaleUpRequestType)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch")
}

func TestPredator_GetOriginalModelName_EmptyTag(t *testing.T) {
	mockRepo := &predatorMockServiceDeployableRepo{
		getById: func(id int) (*servicedeployableconfig.ServiceDeployableConfig, error) {
			return &servicedeployableconfig.ServiceDeployableConfig{
				ID:            id,
				DeployableTag: "",
			}, nil
		},
	}
	p := &Predator{ServiceDeployableRepo: mockRepo}
	got, err := p.GetOriginalModelName("derived_model", 1)
	require.NoError(t, err)
	assert.Equal(t, "derived_model", got)
}

func TestPredator_GetOriginalModelName_WithTag(t *testing.T) {
	mockRepo := &predatorMockServiceDeployableRepo{
		getById: func(id int) (*servicedeployableconfig.ServiceDeployableConfig, error) {
			return &servicedeployableconfig.ServiceDeployableConfig{
				ID:            id,
				DeployableTag: "tag1",
			}, nil
		},
	}
	p := &Predator{ServiceDeployableRepo: mockRepo}
	got, err := p.GetOriginalModelName("base_model_tag1_scaleup", 1)
	require.NoError(t, err)
	assert.Equal(t, "base_model", got)
}

func TestPredator_GetOriginalModelName_RepoError(t *testing.T) {
	mockRepo := &predatorMockServiceDeployableRepo{
		getById: func(id int) (*servicedeployableconfig.ServiceDeployableConfig, error) {
			return nil, errors.New("db error")
		},
	}
	p := &Predator{ServiceDeployableRepo: mockRepo}
	_, err := p.GetOriginalModelName("any", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch")
}

// predatorMockServiceDeployableRepo implements servicedeployableconfig.ServiceDeployableRepository for tests.
type predatorMockServiceDeployableRepo struct {
	getById func(id int) (*servicedeployableconfig.ServiceDeployableConfig, error)
}

func (m *predatorMockServiceDeployableRepo) GetById(id int) (*servicedeployableconfig.ServiceDeployableConfig, error) {
	if m.getById != nil {
		return m.getById(id)
	}
	return nil, errors.New("not implemented")
}

func (m *predatorMockServiceDeployableRepo) Create(_ *servicedeployableconfig.ServiceDeployableConfig) error {
	return nil
}
func (m *predatorMockServiceDeployableRepo) Update(_ *servicedeployableconfig.ServiceDeployableConfig) error {
	return nil
}
func (m *predatorMockServiceDeployableRepo) DeactivateServiceDeployable(_ int, _ string) error {
	return nil
}
func (m *predatorMockServiceDeployableRepo) GetByService(_ string) ([]servicedeployableconfig.ServiceDeployableConfig, error) {
	return nil, nil
}
func (m *predatorMockServiceDeployableRepo) GetAllActive() ([]servicedeployableconfig.ServiceDeployableConfig, error) {
	return nil, nil
}
func (m *predatorMockServiceDeployableRepo) GetByWorkflowStatus(_ string) ([]servicedeployableconfig.ServiceDeployableConfig, error) {
	return nil, nil
}
func (m *predatorMockServiceDeployableRepo) GetByDeployableHealth(_ string) ([]servicedeployableconfig.ServiceDeployableConfig, error) {
	return nil, nil
}
func (m *predatorMockServiceDeployableRepo) GetByNameAndService(_, _ string) (*servicedeployableconfig.ServiceDeployableConfig, error) {
	return nil, nil
}
func (m *predatorMockServiceDeployableRepo) GetByIds(_ []int) ([]servicedeployableconfig.ServiceDeployableConfig, error) {
	return nil, nil
}
