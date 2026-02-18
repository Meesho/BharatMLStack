package handler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGCSClient implements externalcall.GCSClientInterface for unit tests.
// Only methods needed by the functions under test carry real logic;
// the rest return nil/zero to satisfy the interface.
type mockGCSClient struct {
	readFile            func(bucket, objectPath string) ([]byte, error)
	listFilesWithSuffix func(bucket, folderPath, suffix string) ([]string, error)
}

func (m *mockGCSClient) ReadFile(bucket, objectPath string) ([]byte, error) {
	if m.readFile != nil {
		return m.readFile(bucket, objectPath)
	}
	return nil, fmt.Errorf("ReadFile not mocked")
}

func (m *mockGCSClient) ListFilesWithSuffix(bucket, folderPath, suffix string) ([]string, error) {
	if m.listFilesWithSuffix != nil {
		return m.listFilesWithSuffix(bucket, folderPath, suffix)
	}
	return nil, fmt.Errorf("ListFilesWithSuffix not mocked")
}

func (m *mockGCSClient) TransferFolder(_, _, _, _, _, _ string) error          { return nil }
func (m *mockGCSClient) TransferAndDeleteFolder(_, _, _, _, _, _ string) error { return nil }
func (m *mockGCSClient) TransferFolderWithSplitSources(_, _, _, _, _, _, _, _ string) error {
	return nil
}
func (m *mockGCSClient) DeleteFolder(_, _, _ string) error           { return nil }
func (m *mockGCSClient) ListFolders(_, _ string) ([]string, error)   { return nil, nil }
func (m *mockGCSClient) UploadFile(_, _ string, _ []byte) error      { return nil }
func (m *mockGCSClient) CheckFileExists(_, _ string) (bool, error)   { return false, nil }
func (m *mockGCSClient) CheckFolderExists(_, _ string) (bool, error) { return false, nil }
func (m *mockGCSClient) UploadFolderFromLocal(_, _, _ string) error  { return nil }
func (m *mockGCSClient) GetFolderInfo(_, _ string) (*externalcall.GCSFolderInfo, error) {
	return nil, nil
}
func (m *mockGCSClient) ListFoldersWithTimestamp(_, _ string) ([]externalcall.GCSFolderInfo, error) {
	return nil, nil
}
func (m *mockGCSClient) FindFileWithSuffix(_, _, _ string) (bool, string, error) {
	return false, "", nil
}

// Tests for isEnsembleModel
func TestIsEnsembleModel(t *testing.T) {
	tests := []struct {
		name   string
		config *ModelConfig
		want   bool
	}{
		{
			name:   "backend is ensemble",
			config: &ModelConfig{Backend: "ensemble"},
			want:   true,
		},
		{
			name:   "platform is ensemble",
			config: &ModelConfig{Platform: "ensemble"},
			want:   true,
		},
		{
			name: "ensemble_scheduling is set",
			config: &ModelConfig{
				Backend: "python",
				SchedulingChoice: &ModelConfig_EnsembleScheduling{
					EnsembleScheduling: &ModelEnsembling{},
				},
			},
			want: true,
		},
		{
			name:   "python backend - not ensemble",
			config: &ModelConfig{Backend: "python"},
			want:   false,
		},
		{
			name:   "onnxruntime backend - not ensemble",
			config: &ModelConfig{Backend: "onnxruntime"},
			want:   false,
		},
		{
			name:   "empty config - not ensemble",
			config: &ModelConfig{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEnsembleModel(tt.config)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Tests for isPythonBackendModel
func TestIsPythonBackendModel(t *testing.T) {
	tests := []struct {
		name   string
		config *ModelConfig
		want   bool
	}{
		{
			name:   "python backend",
			config: &ModelConfig{Backend: "python"},
			want:   true,
		},
		{
			name:   "onnxruntime backend",
			config: &ModelConfig{Backend: "onnxruntime"},
			want:   false,
		},
		{
			name:   "tensorrt_llm backend",
			config: &ModelConfig{Backend: "tensorrt_llm"},
			want:   false,
		},
		{
			name:   "ensemble backend",
			config: &ModelConfig{Backend: "ensemble"},
			want:   false,
		},
		{
			name:   "empty backend",
			config: &ModelConfig{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPythonBackendModel(tt.config)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Tests for hasPythonLoggerOrPrintStatements
func TestHasPythonLoggerOrPrintStatements(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantFound   bool
		wantDetails int // expected number of violations
	}{
		{
			name:        "clean file with no violations",
			content:     "import numpy as np\n\ndef execute(requests):\n    result = np.array([1, 2])\n    return result\n",
			wantFound:   false,
			wantDetails: 0,
		},
		{
			name:        "logger.info outside any function",
			content:     "import triton\nlogger.info('loading model')\n",
			wantFound:   true,
			wantDetails: 1,
		},
		{
			name:        "print statement outside any function",
			content:     "import triton\nprint('hello')\n",
			wantFound:   true,
			wantDetails: 1,
		},
		{
			name:        "logger.debug inside execute function",
			content:     "def execute(requests):\n    logger.debug('processing')\n    return []\n",
			wantFound:   true,
			wantDetails: 1,
		},
		{
			name:        "logger.info inside initialize is allowed",
			content:     "def initialize(self, args):\n    logger.info('model loaded')\n    self.model = load()\n",
			wantFound:   false,
			wantDetails: 0,
		},
		{
			name:        "print inside finalize is allowed",
			content:     "def finalize(self):\n    print('cleanup done')\n",
			wantFound:   false,
			wantDetails: 0,
		},
		{
			name: "logger in initialize allowed but in execute flagged",
			content: `def initialize(self, args):
    logger.info('init ok')

def execute(self, requests):
    logger.info('processing request')
    return []
`,
			wantFound:   true,
			wantDetails: 1,
		},
		{
			name:        "commented out logger.info is skipped",
			content:     "def execute(requests):\n    # logger.info('debug')\n    return []\n",
			wantFound:   false,
			wantDetails: 0,
		},
		{
			name:        "commented out print is skipped",
			content:     "# print('debug')\ndef execute(requests):\n    return []\n",
			wantFound:   false,
			wantDetails: 0,
		},
		{
			name:        "fingerprint should not match print pattern",
			content:     "def execute(requests):\n    fp = fingerprint(data)\n    return fp\n",
			wantFound:   false,
			wantDetails: 0,
		},
		{
			name:        "blueprint should not match print pattern",
			content:     "def execute(requests):\n    bp = blueprint(config)\n    return bp\n",
			wantFound:   false,
			wantDetails: 0,
		},
		{
			name: "multiple violations across functions",
			content: `import os

def helper():
    print('helper debug')
    logger.info('helper info')

def execute(requests):
    print('exec debug')
    return []
`,
			wantFound:   true,
			wantDetails: 3,
		},
		{
			name:        "logger.INFO case insensitive match",
			content:     "def execute(requests):\n    LOGGER.INFO('upper case')\n    return []\n",
			wantFound:   true,
			wantDetails: 1,
		},
		{
			name: "scope exit resets after initialize",
			content: `def initialize(self, args):
    logger.info('init ok')

logger.info('module level - should flag')
`,
			wantFound:   true,
			wantDetails: 1,
		},
		{
			name:        "empty file",
			content:     "",
			wantFound:   false,
			wantDetails: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, details := hasPythonLoggerOrPrintStatements([]byte(tt.content))
			assert.Equal(t, tt.wantFound, found)
			assert.Equal(t, tt.wantDetails, len(details), "violation details: %v", details)
		})
	}
}

// Tests for buildBalancedViolationSummary
func TestBuildBalancedViolationSummary(t *testing.T) {
	tests := []struct {
		name            string
		violations      []fileViolationInfo
		totalViolations int
		wantContains    []string
		wantNotContain  []string
	}{
		{
			name: "single file single violation",
			violations: []fileViolationInfo{
				{fileName: "model.py", details: []string{"line 5: print('x')"}},
			},
			totalViolations: 1,
			wantContains:    []string{"model.py (1 violation)", "line 5: print('x')"},
		},
		{
			name: "single file multiple violations within cap",
			violations: []fileViolationInfo{
				{fileName: "model.py", details: []string{
					"line 5: print('a')",
					"line 10: print('b')",
					"line 15: print('c')",
				}},
			},
			totalViolations: 3,
			wantContains:    []string{"model.py (3 violations)", "line 5", "line 10", "line 15"},
		},
		{
			name: "single file exceeds cap shows truncation",
			violations: []fileViolationInfo{
				{fileName: "model.py", details: []string{
					"line 1: print('a')",
					"line 2: print('b')",
					"line 3: print('c')",
					"line 4: print('d')",
					"line 5: print('e')",
					"line 6: print('f')",
					"line 7: print('g')",
				}},
			},
			totalViolations: 7,
			wantContains:    []string{"model.py (7 violations)", "...and 2 more"},
		},
		{
			name: "two files with round-robin distribution",
			violations: []fileViolationInfo{
				{fileName: "a.py", details: []string{"line 1: print('a1')", "line 2: print('a2')", "line 3: print('a3')"}},
				{fileName: "b.py", details: []string{"line 10: print('b1')", "line 20: print('b2')", "line 30: print('b3')"}},
			},
			totalViolations: 6,
			wantContains:    []string{"a.py", "b.py"},
		},
		{
			name: "uses singular 'violation' for single-violation file",
			violations: []fileViolationInfo{
				{fileName: "single.py", details: []string{"line 1: print('x')"}},
			},
			totalViolations: 1,
			wantContains:    []string{"1 violation)"},
			wantNotContain:  []string{"1 violations)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildBalancedViolationSummary(tt.violations, tt.totalViolations)
			for _, want := range tt.wantContains {
				assert.Contains(t, got, want)
			}
			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, got, notWant)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests for validateModelConfiguration (warmup check via GCS mock)
// ---------------------------------------------------------------------------

func TestValidateModelConfiguration_WarmupCheck(t *testing.T) {
	pythonWithWarmup := `name: "my_model"
backend: "python"
max_batch_size: 8
model_warmup {
  name: "warmup_request"
  batch_size: 1
  inputs {
    key: "input"
    value: {
      data_type: TYPE_FP32
      dims: [1, 3]
      zero_data: true
    }
  }
}
`

	pythonWithoutWarmup := `name: "my_model"
backend: "python"
max_batch_size: 8
`

	ensembleConfig := `name: "ensemble_model"
platform: "ensemble"
max_batch_size: 8
ensemble_scheduling {
  step {
    model_name: "preprocess"
    model_version: 1
  }
}
`

	ensembleBackendConfig := `name: "ensemble_model"
backend: "ensemble"
max_batch_size: 8
`

	tests := []struct {
		name       string
		configData string
		wantErr    bool
		errContain string
	}{
		{
			name:       "python model with warmup passes",
			configData: pythonWithWarmup,
			wantErr:    false,
		},
		{
			name:       "python model without warmup fails",
			configData: pythonWithoutWarmup,
			wantErr:    true,
			errContain: "model_warmup configuration is missing",
		},
		{
			name:       "ensemble model skips warmup check",
			configData: ensembleConfig,
			wantErr:    false,
		},
		{
			name:       "ensemble backend model skips warmup check",
			configData: ensembleBackendConfig,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gcs := &mockGCSClient{
				readFile: func(bucket, objectPath string) ([]byte, error) {
					return []byte(tt.configData), nil
				},
			}

			p := &Predator{GcsClient: gcs}
			err := p.validateModelConfiguration("gs://test-bucket/models/my_model")

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateModelConfiguration_InvalidGCSPath(t *testing.T) {
	p := &Predator{}
	err := p.validateModelConfiguration("invalid-path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid GCS path format")
}

func TestValidateModelConfiguration_ReadFileError(t *testing.T) {
	gcs := &mockGCSClient{
		readFile: func(_, _ string) ([]byte, error) {
			return nil, fmt.Errorf("bucket not found")
		},
	}
	p := &Predator{GcsClient: gcs}
	err := p.validateModelConfiguration("gs://test-bucket/models/my_model")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config.pbtxt")
}

func TestValidateModelConfiguration_InvalidProto(t *testing.T) {
	gcs := &mockGCSClient{
		readFile: func(_, _ string) ([]byte, error) {
			return []byte("this is not valid proto text {{{"), nil
		},
	}
	p := &Predator{GcsClient: gcs}
	err := p.validateModelConfiguration("gs://test-bucket/models/my_model")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config.pbtxt")
}

// Tests for validateNoLoggerOrPrintStatements (integration with GCS mock)
func TestValidateNoLoggerOrPrintStatements_NonPythonBackend(t *testing.T) {
	configData := `name: "onnx_model"
backend: "onnxruntime"
max_batch_size: 8
`
	gcs := &mockGCSClient{
		readFile: func(_, objectPath string) ([]byte, error) {
			if strings.HasSuffix(objectPath, "config.pbtxt") {
				return []byte(configData), nil
			}
			return nil, fmt.Errorf("unexpected read: %s", objectPath)
		},
	}

	p := &Predator{GcsClient: gcs}
	err := p.validateNoLoggerOrPrintStatements("gs://test-bucket/models/onnx_model")
	require.NoError(t, err)
}

func TestValidateNoLoggerOrPrintStatements_PythonBackendClean(t *testing.T) {
	configData := `name: "py_model"
backend: "python"
max_batch_size: 8
`
	cleanPython := `import numpy as np

def initialize(self, args):
    logger.info('init')

def execute(self, requests):
    result = np.array([1, 2, 3])
    return result

def finalize(self):
    print('done')
`

	gcs := &mockGCSClient{
		readFile: func(_, objectPath string) ([]byte, error) {
			if strings.HasSuffix(objectPath, "config.pbtxt") {
				return []byte(configData), nil
			}
			return []byte(cleanPython), nil
		},
		listFilesWithSuffix: func(_, _, _ string) ([]string, error) {
			return []string{"models/py_model/1/model.py"}, nil
		},
	}

	p := &Predator{GcsClient: gcs}
	err := p.validateNoLoggerOrPrintStatements("gs://test-bucket/models/py_model")
	require.NoError(t, err)
}

func TestValidateNoLoggerOrPrintStatements_PythonBackendWithViolations(t *testing.T) {
	configData := `name: "py_model"
backend: "python"
max_batch_size: 8
`
	dirtyPython := `import numpy as np

def execute(self, requests):
    logger.info('processing')
    print('debug output')
    return []
`

	gcs := &mockGCSClient{
		readFile: func(_, objectPath string) ([]byte, error) {
			if strings.HasSuffix(objectPath, "config.pbtxt") {
				return []byte(configData), nil
			}
			return []byte(dirtyPython), nil
		},
		listFilesWithSuffix: func(_, _, _ string) ([]string, error) {
			return []string{"models/py_model/1/model.py"}, nil
		},
	}

	p := &Predator{GcsClient: gcs}
	err := p.validateNoLoggerOrPrintStatements("gs://test-bucket/models/py_model")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "logger/print statements")
	assert.Contains(t, err.Error(), "model.py")
}

func TestValidateNoLoggerOrPrintStatements_NoPythonFiles(t *testing.T) {
	configData := `name: "py_model"
backend: "python"
max_batch_size: 8
`
	gcs := &mockGCSClient{
		readFile: func(_, objectPath string) ([]byte, error) {
			if strings.HasSuffix(objectPath, "config.pbtxt") {
				return []byte(configData), nil
			}
			return nil, fmt.Errorf("not found")
		},
		listFilesWithSuffix: func(_, _, _ string) ([]string, error) {
			return []string{}, nil
		},
	}

	p := &Predator{GcsClient: gcs}
	err := p.validateNoLoggerOrPrintStatements("gs://test-bucket/models/py_model")
	require.NoError(t, err)
}

func TestValidateNoLoggerOrPrintStatements_InvalidGCSPath(t *testing.T) {
	p := &Predator{}
	err := p.validateNoLoggerOrPrintStatements("invalid-path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid GCS path format")
}

func TestValidateNoLoggerOrPrintStatements_MultipleFiles(t *testing.T) {
	configData := `name: "py_model"
backend: "python"
max_batch_size: 8
`
	fileContents := map[string]string{
		"models/py_model/config.pbtxt": configData,
		"models/py_model/1/model.py": `def execute(self, requests):
    print('debug in model')
    return []
`,
		"models/py_model/1/utils.py": `def helper():
    logger.info('helper log')
    return True
`,
	}

	gcs := &mockGCSClient{
		readFile: func(_, objectPath string) ([]byte, error) {
			if content, ok := fileContents[objectPath]; ok {
				return []byte(content), nil
			}
			return nil, fmt.Errorf("file not found: %s", objectPath)
		},
		listFilesWithSuffix: func(_, _, _ string) ([]string, error) {
			return []string{"models/py_model/1/model.py", "models/py_model/1/utils.py"}, nil
		},
	}

	p := &Predator{GcsClient: gcs}
	err := p.validateNoLoggerOrPrintStatements("gs://test-bucket/models/py_model")
	require.Error(t, err)
	errMsg := err.Error()
	assert.Contains(t, errMsg, "model.py")
	assert.Contains(t, errMsg, "utils.py")
}
