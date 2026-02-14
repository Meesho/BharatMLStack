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

func TestReplaceInstanceCountInConfigPreservingFormat(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		newCount  int
		wantData  string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "preserves formatting around count",
			data: []byte(`instance_group [
  {
    count : 2
  }
]
`),
			newCount: 10,
			wantData: "count : 10",
			wantErr:  false,
		},
		{
			name: "works with no spaces",
			data: []byte(`instance_group[{count:2}]`),
			newCount: 5,
			wantData: "count:5",
			wantErr:  false,
		},
		{
			name: "works with excessive spacing",
			data: []byte(`instance_group   [   {   count    :      3    } ]`),
			newCount: 8,
			wantData: "count    :      8",
			wantErr:  false,
		},
		{
			name: "count outside instance_group should not change",
			data: []byte(`count: 99
instance_group [
  { count: 1 }
]`),
			newCount: 4,
			wantData: "count: 4",
			wantErr:  false,
		},
		{
			name: "instance_group but no count",
			data: []byte(`instance_group [
  { kind: KIND_CPU }
]`),
			newCount:  1,
			wantErr:   true,
			errSubstr: errNoInstanceGroup,
		},
		{
			name: "nested formatting with line breaks",
			data: []byte(`instance_group[
{kind:KIND_CPU
count:1}]`),
			newCount: 6,
			wantData: "count:6",
			wantErr:  false,
		},
		{
			name:      "error when no instance_group",
			data:      []byte("name: \"model\"\nplatform: \"tensorflow\"\n"),
			newCount:  1,
			wantErr:   true,
			errSubstr: errNoInstanceGroup,
		},
		{
			name:      "error on empty input",
			data:      []byte(""),
			newCount:  1,
			wantErr:   true,
			errSubstr: errNoInstanceGroup,
		},
		{
			name: "real config with no instance_group",
			data: []byte(`name: "ensemble_clp_p13n_explore_v1"
platform: "ensemble"
max_batch_size: 200
input [
    {
        name: "input__0"
        data_type: TYPE_STRING
        dims: [ 50 ]
    } 
]
output [
  {
    name: "output__0"
    data_type: TYPE_FP32
    dims: [ 1 ]
  }
]
ensemble_scheduling {
  step [
    {
      model_name: "preprocessing_clp_p13n_explore_v1"
      model_version: 1
      input_map {
        key: "INPUT__0"
        value: "input__0"
      }
      output_map {
        key: "OUTPUT__0"
        value: "OUTPUT__0"
      }
    },
    {
      model_name: "clp_p13n_explore_v1"
      model_version: 1
      input_map {
        key: "input__0"
        value: "OUTPUT__0"
      }
      output_map {
        key: "output__0"
        value: "output__0"
      }
    }
  ]
}
`),
			newCount:  2,
			wantErr:   true,
			errSubstr: errNoInstanceGroup,
		},
		{
			name: "real config, updates instance_group count",
			data: []byte(`name: "clp_ad_pcvr_v1_2"
backend: "fil"
max_batch_size: 200

dynamic_batching {
  max_queue_delay_microseconds: 100
}

input [
 {
    name: "input__0"
    data_type: TYPE_FP32
    dims: [ 91 ]
  }
]
output [
 {
    name: "output__0"
    data_type: TYPE_FP32
    dims: [ 1 ]
  }
]
response_cache {
  enable: false
}
instance_group [
  {
    count: 1
    kind : KIND_CPU
  }
]
parameters [
  {
    key: "model_type"
    value: { string_value: "treelite_checkpoint" }
  },
  {
    key: "predict_proba"
    value: { string_value: "false" }
  },
  {
    key: "output_class"
    value: { string_value: "false" }
  },
  {
    key: "storage_type"
    value: { string_value: "AUTO" }
  },
  {
    key: "use_experimental_optimizations"
    value: { string_value: "true" }
  }
]
`),
			newCount: 4,
			wantData: "count: 4",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := replaceInstanceCountInConfigPreservingFormat(tt.data, tt.newCount)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstr)
				return
			}

			require.NoError(t, err)
			assert.Contains(t, string(got), tt.wantData)
		})
	}
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
