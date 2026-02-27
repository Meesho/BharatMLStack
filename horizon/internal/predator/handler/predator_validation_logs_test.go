package handler

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	infrastructurehandler "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/handler"
	pred "github.com/Meesho/BharatMLStack/horizon/internal/predator"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/validationjob"
	"github.com/Meesho/BharatMLStack/horizon/pkg/argocd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock: InfrastructureHandler ---

type mockInfrastructureHandler struct {
	getResourceDetailFn  func(appName, workingEnv string) (*infrastructurehandler.ResourceDetail, error)
	getApplicationLogsFn func(appName, workingEnv string, opts *argocd.ApplicationLogsOptions) ([]argocd.ApplicationLogEntry, error)
	restartDeploymentFn  func(appName, workingEnv string, isCanary bool) error
}

func (m *mockInfrastructureHandler) GetHPAProperties(_, _ string) (*infrastructurehandler.HPAConfig, error) {
	return nil, nil
}
func (m *mockInfrastructureHandler) GetConfig(_, _ string) infrastructurehandler.Config {
	return infrastructurehandler.Config{}
}
func (m *mockInfrastructureHandler) GetResourceDetail(appName, workingEnv string) (*infrastructurehandler.ResourceDetail, error) {
	if m.getResourceDetailFn != nil {
		return m.getResourceDetailFn(appName, workingEnv)
	}
	return nil, errors.New("not implemented")
}
func (m *mockInfrastructureHandler) GetApplicationLogs(appName, workingEnv string, opts *argocd.ApplicationLogsOptions) ([]argocd.ApplicationLogEntry, error) {
	if m.getApplicationLogsFn != nil {
		return m.getApplicationLogsFn(appName, workingEnv, opts)
	}
	return nil, errors.New("not implemented")
}
func (m *mockInfrastructureHandler) RestartDeployment(appName, workingEnv string, isCanary bool) error {
	if m.restartDeploymentFn != nil {
		return m.restartDeploymentFn(appName, workingEnv, isCanary)
	}
	return nil
}
func (m *mockInfrastructureHandler) UpdateCPUThreshold(_, _, _, _ string) error  { return nil }
func (m *mockInfrastructureHandler) UpdateGPUThreshold(_, _, _, _ string) error  { return nil }
func (m *mockInfrastructureHandler) UpdateSharedMemory(_, _, _, _ string) error  { return nil }
func (m *mockInfrastructureHandler) UpdatePodAnnotations(_ string, _ map[string]string, _, _ string) error {
	return nil
}
func (m *mockInfrastructureHandler) UpdateAutoscalingTriggers(_ string, _ []interface{}, _, _ string) error {
	return nil
}

// --- Mock: GCSClientInterface ---

type mockGCSClient struct {
	uploadFileFn func(bucket, objectPath string, data []byte) error
}

func (m *mockGCSClient) ReadFile(_, _ string) ([]byte, error)                  { return nil, nil }
func (m *mockGCSClient) TransferFolder(_, _, _, _, _, _ string) error          { return nil }
func (m *mockGCSClient) TransferAndDeleteFolder(_, _, _, _, _, _ string) error { return nil }
func (m *mockGCSClient) TransferFolderWithSplitSources(_, _, _, _, _, _, _, _ string) error {
	return nil
}
func (m *mockGCSClient) DeleteFolder(_, _, _ string) error         { return nil }
func (m *mockGCSClient) ListFolders(_, _ string) ([]string, error) { return nil, nil }
func (m *mockGCSClient) UploadFile(bucket, objectPath string, data []byte) error {
	if m.uploadFileFn != nil {
		return m.uploadFileFn(bucket, objectPath, data)
	}
	return nil
}
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

// --- Mock: ValidationJobRepository ---

type mockValidationJobRepo struct {
	getByIDFn              func(id uint) (*validationjob.Table, error)
	updateValidationResult func(id uint, result bool, errorMessage string) error
}

func (m *mockValidationJobRepo) Create(_ *validationjob.Table) error { return nil }
func (m *mockValidationJobRepo) GetByID(id uint) (*validationjob.Table, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return nil, errors.New("not implemented")
}
func (m *mockValidationJobRepo) GetByGroupID(_ string) (*validationjob.Table, error) {
	return nil, nil
}
func (m *mockValidationJobRepo) GetPendingJobs() ([]validationjob.Table, error) { return nil, nil }
func (m *mockValidationJobRepo) UpdateStatus(_ uint, _, _ string) error         { return nil }
func (m *mockValidationJobRepo) UpdateValidationResult(id uint, result bool, errorMessage string) error {
	if m.updateValidationResult != nil {
		return m.updateValidationResult(id, result, errorMessage)
	}
	return nil
}
func (m *mockValidationJobRepo) IncrementHealthCheck(_ uint) error { return nil }
func (m *mockValidationJobRepo) GetJobsToCleanup(_ time.Duration) ([]validationjob.Table, error) {
	return nil, nil
}
func (m *mockValidationJobRepo) Delete(_ uint) error { return nil }

// --- helper to save/restore package-level predator vars ---

type predVarSnapshot struct {
	modelBucket  string
	modelBase    string
	configBucket string
	configBase   string
	appEnv       string
}

func savePredVars() predVarSnapshot {
	return predVarSnapshot{
		modelBucket:  pred.GcsModelBucket,
		modelBase:    pred.GcsModelBasePath,
		configBucket: pred.GcsConfigBucket,
		configBase:   pred.GcsConfigBasePath,
		appEnv:       pred.AppEnv,
	}
}

func (s predVarSnapshot) restore() {
	pred.GcsModelBucket = s.modelBucket
	pred.GcsModelBasePath = s.modelBase
	pred.GcsConfigBucket = s.configBucket
	pred.GcsConfigBasePath = s.configBase
	pred.AppEnv = s.appEnv
}

func setNonProd() {
	pred.GcsModelBucket = "model-bucket"
	pred.GcsModelBasePath = "models/v1"
	pred.AppEnv = "int"
}

func setProd() {
	pred.GcsConfigBucket = "config-bucket"
	pred.GcsConfigBasePath = "configs/v1"
	pred.AppEnv = "prd"
}

// ============================================================
// findDegradedPods
// ============================================================

func TestFindDegradedPods_ReturnsDegradedOnly(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{
				Nodes: []infrastructurehandler.Node{
					{Kind: "Pod", Name: "pod-healthy", Health: infrastructurehandler.Health{Status: "Healthy"}},
					{Kind: "Pod", Name: "pod-degraded-1", Health: infrastructurehandler.Health{Status: "Degraded"}},
					{Kind: "Pod", Name: "pod-missing", Health: infrastructurehandler.Health{Status: "Missing"}},
					{Kind: "Pod", Name: "pod-degraded-2", Health: infrastructurehandler.Health{Status: "Degraded"}},
					{Kind: "Deployment", Name: "deploy-1", Health: infrastructurehandler.Health{Status: "Degraded"}},
				},
			}, nil
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	pods := p.findDegradedPods("test-svc")

	assert.Len(t, pods, 2)
	assert.Equal(t, "pod-degraded-1", pods[0].Name)
	assert.Equal(t, "pod-degraded-2", pods[1].Name)
}

func TestFindDegradedPods_NoDegraded_ReturnsNil(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{
				Nodes: []infrastructurehandler.Node{
					{Kind: "Pod", Name: "pod-1", Health: infrastructurehandler.Health{Status: "Healthy"}},
					{Kind: "Pod", Name: "pod-2", Health: infrastructurehandler.Health{Status: "Missing"}},
				},
			}, nil
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	pods := p.findDegradedPods("test-svc")

	assert.Nil(t, pods, "should return nil when no pods are Degraded")
}

func TestFindDegradedPods_OnlyNonPodNodes_ReturnsNil(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{
				Nodes: []infrastructurehandler.Node{
					{Kind: "Deployment", Name: "deploy-1", Health: infrastructurehandler.Health{Status: "Degraded"}},
					{Kind: "ReplicaSet", Name: "rs-1", Health: infrastructurehandler.Health{Status: "Degraded"}},
				},
			}, nil
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	pods := p.findDegradedPods("test-svc")

	assert.Nil(t, pods)
}

func TestFindDegradedPods_APIError_ReturnsNil(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return nil, errors.New("connection refused")
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	pods := p.findDegradedPods("test-svc")

	assert.Nil(t, pods)
}

func TestFindDegradedPods_NilResourceDetail_ReturnsNil(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return nil, nil
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	pods := p.findDegradedPods("test-svc")

	assert.Nil(t, pods)
}

func TestFindDegradedPods_EmptyNodes_ReturnsNil(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{Nodes: []infrastructurehandler.Node{}}, nil
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	pods := p.findDegradedPods("test-svc")

	assert.Nil(t, pods)
}

func TestFindDegradedPods_PassesCorrectArgs(t *testing.T) {
	var capturedApp, capturedEnv string
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(appName, workingEnv string) (*infrastructurehandler.ResourceDetail, error) {
			capturedApp = appName
			capturedEnv = workingEnv
			return nil, nil
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "staging"}

	p.findDegradedPods("my-service")

	assert.Equal(t, "my-service", capturedApp)
	assert.Equal(t, "staging", capturedEnv)
}

// ============================================================
// fetchDegradedPodLogs
// ============================================================

func TestFetchDegradedPodLogs_SetsCorrectOptsAndConvertsIST(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getApplicationLogsFn: func(appName, _ string, opts *argocd.ApplicationLogsOptions) ([]argocd.ApplicationLogEntry, error) {
			assert.Equal(t, "test-svc", appName)
			assert.Equal(t, "pod-1", opts.PodName)
			assert.Equal(t, "test-svc", opts.Container)
			assert.True(t, opts.Previous, "Previous flag must be true for crash logs")

			return []argocd.ApplicationLogEntry{
				{Result: argocd.ApplicationLogResult{Content: "OOM killed", TimeStampStr: "2024-01-15T10:30:00Z"}},
				{Result: argocd.ApplicationLogResult{Content: "container exited", TimeStampStr: "2024-01-15T10:30:01Z"}},
			}, nil
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	logs, err := p.fetchDegradedPodLogs("test-svc", "pod-1")

	require.NoError(t, err)
	assert.Contains(t, logs, "[2024-01-15 16:00:00 IST] OOM killed")
	assert.Contains(t, logs, "[2024-01-15 16:00:01 IST] container exited")
}

func TestFetchDegradedPodLogs_Error_WrapsWithPodName(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getApplicationLogsFn: func(_, _ string, _ *argocd.ApplicationLogsOptions) ([]argocd.ApplicationLogEntry, error) {
			return nil, errors.New("ArgoCD unavailable")
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	_, err := p.fetchDegradedPodLogs("test-svc", "pod-crash")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch logs for pod pod-crash")
	assert.Contains(t, err.Error(), "ArgoCD unavailable")
}

func TestFetchDegradedPodLogs_EmptyEntries_ReturnsEmptyString(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getApplicationLogsFn: func(_, _ string, _ *argocd.ApplicationLogsOptions) ([]argocd.ApplicationLogEntry, error) {
			return []argocd.ApplicationLogEntry{}, nil
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	logs, err := p.fetchDegradedPodLogs("svc", "pod-1")

	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestFetchDegradedPodLogs_NilEntries_ReturnsEmptyString(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getApplicationLogsFn: func(_, _ string, _ *argocd.ApplicationLogsOptions) ([]argocd.ApplicationLogEntry, error) {
			return nil, nil
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	logs, err := p.fetchDegradedPodLogs("svc", "pod-1")

	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestFetchDegradedPodLogs_EntryWithEmptyTimestamp_UsesCurrentTime(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getApplicationLogsFn: func(_, _ string, _ *argocd.ApplicationLogsOptions) ([]argocd.ApplicationLogEntry, error) {
			return []argocd.ApplicationLogEntry{
				{Result: argocd.ApplicationLogResult{Content: "panic: nil pointer", TimeStampStr: ""}},
			}, nil
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	logs, err := p.fetchDegradedPodLogs("svc", "pod-1")

	require.NoError(t, err)
	assert.Contains(t, logs, "IST")
	assert.Contains(t, logs, "panic: nil pointer")
}

func TestFetchDegradedPodLogs_EntryWithEmptyContent_FormatsCorrectly(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getApplicationLogsFn: func(_, _ string, _ *argocd.ApplicationLogsOptions) ([]argocd.ApplicationLogEntry, error) {
			return []argocd.ApplicationLogEntry{
				{Result: argocd.ApplicationLogResult{Content: "", TimeStampStr: "2024-01-15T10:30:00Z"}},
			}, nil
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	logs, err := p.fetchDegradedPodLogs("svc", "pod-1")

	require.NoError(t, err)
	assert.Contains(t, logs, "[2024-01-15 16:00:00 IST] \n")
}

// ============================================================
// convertToIST
// ============================================================

func TestConvertToIST_RFC3339(t *testing.T) {
	result := convertToIST("2024-01-15T10:30:00Z")
	assert.Equal(t, "2024-01-15 16:00:00 IST", result)
}

func TestConvertToIST_RFC3339Nano(t *testing.T) {
	result := convertToIST("2024-01-15T10:30:00.123456789Z")
	assert.Equal(t, "2024-01-15 16:00:00 IST", result)
}

func TestConvertToIST_Empty_UsesCurrentTime(t *testing.T) {
	before := time.Now().In(istLocation)
	result := convertToIST("")
	after := time.Now().In(istLocation)

	assert.Contains(t, result, "IST")
	parsed, err := time.ParseInLocation(istTimeFmt, result, istLocation)
	require.NoError(t, err, "result should be parseable as IST time")
	assert.False(t, parsed.Before(before.Truncate(time.Second)))
	assert.False(t, parsed.After(after.Add(time.Second)))
}

func TestConvertToIST_InvalidFormat_ReturnsRaw(t *testing.T) {
	result := convertToIST("not-a-timestamp")
	assert.Equal(t, "not-a-timestamp", result)
}

func TestConvertToIST_WithTimezoneOffset(t *testing.T) {
	result := convertToIST("2024-01-15T16:00:00+05:30")
	assert.Equal(t, "2024-01-15 16:00:00 IST", result)
}

func TestConvertToIST_RFC3339_NonUTC(t *testing.T) {
	result := convertToIST("2024-01-15T05:30:00-05:00")
	assert.Equal(t, "2024-01-15 16:00:00 IST", result)
}

// ============================================================
// formatValidationLogsPlainText
// ============================================================

func TestFormatLogs_MultiplePods(t *testing.T) {
	job := &validationjob.Table{ID: 1, GroupID: "42", ServiceName: "test-svc"}
	pods := []podLogEntry{
		{name: "pod-bad-1", logs: "[2024-01-15 16:00:00 IST] OOM killed\n"},
		{name: "pod-bad-2", logs: "[2024-01-15 16:05:00 IST] segfault\n"},
	}

	data := formatValidationLogsPlainText(job, "int", pods)
	output := string(data)

	assert.Contains(t, output, "Validation Failure Log Report")
	assert.Contains(t, output, "Group ID     : 42")
	assert.Contains(t, output, "Job ID       : 1")
	assert.Contains(t, output, "Service Name : test-svc")
	assert.Contains(t, output, "Working Env  : int")
	assert.Contains(t, output, "Captured At  :")
	assert.Contains(t, output, "IST")

	assert.Contains(t, output, "Pod: pod-bad-1")
	assert.Contains(t, output, "OOM killed")
	assert.Contains(t, output, "Pod: pod-bad-2")
	assert.Contains(t, output, "segfault")

	assert.Equal(t, 2, strings.Count(output, "Pod: "), "should have exactly 2 pod headers")
}

func TestFormatLogs_PodWithError(t *testing.T) {
	job := &validationjob.Table{ID: 1, GroupID: "42", ServiceName: "test-svc"}
	pods := []podLogEntry{
		{name: "pod-err", err: "failed to fetch logs: connection refused"},
	}

	data := formatValidationLogsPlainText(job, "int", pods)
	output := string(data)

	assert.Contains(t, output, "Pod: pod-err")
	assert.Contains(t, output, "[ERROR] failed to fetch logs: connection refused")
}

func TestFormatLogs_PodWithBothErrorAndLogs(t *testing.T) {
	job := &validationjob.Table{ID: 1, GroupID: "42", ServiceName: "test-svc"}
	pods := []podLogEntry{
		{name: "pod-partial", err: "partial failure", logs: "some recovered logs\n"},
	}

	data := formatValidationLogsPlainText(job, "int", pods)
	output := string(data)

	assert.Contains(t, output, "[ERROR] partial failure")
	assert.Contains(t, output, "some recovered logs")
}

func TestFormatLogs_EmptyPodsList_HeaderOnly(t *testing.T) {
	job := &validationjob.Table{ID: 5, GroupID: "99", ServiceName: "empty-svc"}

	data := formatValidationLogsPlainText(job, "prd", []podLogEntry{})
	output := string(data)

	assert.Contains(t, output, "Validation Failure Log Report")
	assert.Contains(t, output, "Group ID     : 99")
	assert.Contains(t, output, "Job ID       : 5")
	assert.NotContains(t, output, "Pod: ")
}

func TestFormatLogs_PodWithEmptyName(t *testing.T) {
	job := &validationjob.Table{ID: 1, GroupID: "1", ServiceName: "svc"}
	pods := []podLogEntry{
		{name: "", logs: "log line\n"},
	}

	data := formatValidationLogsPlainText(job, "int", pods)
	output := string(data)

	assert.Contains(t, output, "Pod: \n")
	assert.Contains(t, output, "log line")
}

func TestFormatLogs_PodWithEmptyLogsAndNoError(t *testing.T) {
	job := &validationjob.Table{ID: 1, GroupID: "1", ServiceName: "svc"}
	pods := []podLogEntry{
		{name: "pod-empty"},
	}

	data := formatValidationLogsPlainText(job, "int", pods)
	output := string(data)

	assert.Contains(t, output, "Pod: pod-empty")
	assert.NotContains(t, output, "[ERROR]")
}

// ============================================================
// buildValidationLogsGCSPath
// ============================================================

func TestBuildGCSPath_NonProd(t *testing.T) {
	snap := savePredVars()
	defer snap.restore()
	setNonProd()

	p := &Predator{workingEnv: "int"}
	job := &validationjob.Table{ServiceName: "test-svc", GroupID: "42"}

	bucket, objectPath := p.buildValidationLogsGCSPath(job)

	assert.Equal(t, "model-bucket", bucket)
	assert.Equal(t, "validation-logs/test-svc/42.log", objectPath)
}

func TestBuildGCSPath_Prod(t *testing.T) {
	snap := savePredVars()
	defer snap.restore()
	setProd()

	p := &Predator{workingEnv: "prd"}
	job := &validationjob.Table{ServiceName: "prod-svc", GroupID: "99"}

	bucket, objectPath := p.buildValidationLogsGCSPath(job)

	assert.Equal(t, "config-bucket", bucket)
	assert.Equal(t, "validation-logs/prod-svc/99.log", objectPath)
}

func TestBuildGCSPath_Deterministic(t *testing.T) {
	snap := savePredVars()
	defer snap.restore()
	setNonProd()

	p := &Predator{workingEnv: "int"}
	job := &validationjob.Table{ServiceName: "svc-a", GroupID: "77"}

	b1, p1 := p.buildValidationLogsGCSPath(job)
	b2, p2 := p.buildValidationLogsGCSPath(job)

	assert.Equal(t, b1, b2)
	assert.Equal(t, p1, p2)
}

func TestBuildGCSPath_EmptyBucketVar_ReturnsEmpty(t *testing.T) {
	snap := savePredVars()
	defer snap.restore()
	pred.GcsModelBucket = ""
	pred.AppEnv = "int"

	p := &Predator{workingEnv: "int"}
	job := &validationjob.Table{ServiceName: "svc", GroupID: "1"}

	bucket, _ := p.buildValidationLogsGCSPath(job)

	assert.Empty(t, bucket, "should return empty bucket when GcsModelBucket is unset")
}

func TestBuildGCSPath_EmptyServiceName(t *testing.T) {
	snap := savePredVars()
	defer snap.restore()
	setNonProd()

	p := &Predator{workingEnv: "int"}
	job := &validationjob.Table{ServiceName: "", GroupID: "42"}

	_, objectPath := p.buildValidationLogsGCSPath(job)

	assert.Equal(t, "validation-logs/42.log", objectPath,
		"path.Join collapses empty segment")
}

func TestBuildGCSPath_EmptyGroupID(t *testing.T) {
	snap := savePredVars()
	defer snap.restore()
	setNonProd()

	p := &Predator{workingEnv: "int"}
	job := &validationjob.Table{ServiceName: "svc", GroupID: ""}

	_, objectPath := p.buildValidationLogsGCSPath(job)

	assert.Equal(t, "validation-logs/svc/.log", objectPath)
}

// ============================================================
// captureAndUploadFailureLogs — end-to-end
// ============================================================

func TestCapture_EndToEnd_HappyPath(t *testing.T) {
	snap := savePredVars()
	defer snap.restore()
	pred.GcsModelBucket = "test-bucket"
	pred.AppEnv = "int"

	var capturedBucket, capturedPath string
	var capturedData []byte

	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{
				Nodes: []infrastructurehandler.Node{
					{Kind: "Pod", Name: "pod-bad", Health: infrastructurehandler.Health{Status: "Degraded"}},
					{Kind: "Pod", Name: "pod-ok", Health: infrastructurehandler.Health{Status: "Healthy"}},
				},
			}, nil
		},
		getApplicationLogsFn: func(_, _ string, opts *argocd.ApplicationLogsOptions) ([]argocd.ApplicationLogEntry, error) {
			assert.True(t, opts.Previous, "Previous flag must be true")
			return []argocd.ApplicationLogEntry{
				{Result: argocd.ApplicationLogResult{Content: "OOM killed", TimeStampStr: "2024-01-15T10:30:00Z"}},
			}, nil
		},
	}
	gcs := &mockGCSClient{
		uploadFileFn: func(bucket, objectPath string, data []byte) error {
			capturedBucket = bucket
			capturedPath = objectPath
			capturedData = data
			return nil
		},
	}

	p := &Predator{
		infrastructureHandler: infra,
		GcsClient:             gcs,
		workingEnv:            "int",
	}
	job := &validationjob.Table{ID: 1, GroupID: "42", ServiceName: "test-svc"}

	gcsURI := p.captureAndUploadFailureLogs(job)

	assert.Equal(t, "https://storage.cloud.google.com/test-bucket/validation-logs/test-svc/42.log", gcsURI)
	assert.Equal(t, "test-bucket", capturedBucket)
	assert.Equal(t, "validation-logs/test-svc/42.log", capturedPath)

	output := string(capturedData)
	assert.Contains(t, output, "Group ID     : 42")
	assert.Contains(t, output, "Pod: pod-bad")
	assert.NotContains(t, output, "Pod: pod-ok", "healthy pod must not appear")
	assert.Contains(t, output, "OOM killed")
	assert.Contains(t, output, "IST")
	assert.True(t, len(capturedData) > 0, "uploaded data must not be empty")
}

func TestCapture_NilJob_ReturnsEmpty(t *testing.T) {
	p := &Predator{}
	result := p.captureAndUploadFailureLogs(nil)
	assert.Empty(t, result)
}

func TestCapture_GCSUploadFails_ReturnsEmpty(t *testing.T) {
	snap := savePredVars()
	defer snap.restore()
	pred.GcsModelBucket = "bucket"
	pred.AppEnv = "int"

	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{
				Nodes: []infrastructurehandler.Node{
					{Kind: "Pod", Name: "pod-bad", Health: infrastructurehandler.Health{Status: "Degraded"}},
				},
			}, nil
		},
		getApplicationLogsFn: func(_, _ string, _ *argocd.ApplicationLogsOptions) ([]argocd.ApplicationLogEntry, error) {
			return []argocd.ApplicationLogEntry{
				{Result: argocd.ApplicationLogResult{Content: "log line", TimeStampStr: "2024-01-15T10:30:00Z"}},
			}, nil
		},
	}
	gcs := &mockGCSClient{
		uploadFileFn: func(_, _ string, _ []byte) error {
			return errors.New("GCS unavailable")
		},
	}

	p := &Predator{infrastructureHandler: infra, GcsClient: gcs, workingEnv: "int"}
	job := &validationjob.Table{ID: 1, GroupID: "10", ServiceName: "svc"}

	result := p.captureAndUploadFailureLogs(job)

	assert.Empty(t, result, "must return empty on GCS failure")
}

func TestCapture_NoDegradedPods_ReturnsEmpty_NoUpload(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{
				Nodes: []infrastructurehandler.Node{
					{Kind: "Pod", Name: "pod-ok", Health: infrastructurehandler.Health{Status: "Healthy"}},
				},
			}, nil
		},
	}
	uploaded := false
	gcs := &mockGCSClient{
		uploadFileFn: func(_, _ string, _ []byte) error {
			uploaded = true
			return nil
		},
	}

	p := &Predator{infrastructureHandler: infra, GcsClient: gcs, workingEnv: "int"}
	job := &validationjob.Table{ID: 1, GroupID: "7", ServiceName: "svc"}

	result := p.captureAndUploadFailureLogs(job)

	assert.Empty(t, result)
	assert.False(t, uploaded, "must not upload when no degraded pods")
}

func TestCapture_GetResourceDetailFails_ReturnsEmpty_NoUpload(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return nil, errors.New("ArgoCD connection timeout")
		},
	}
	uploaded := false
	gcs := &mockGCSClient{
		uploadFileFn: func(_, _ string, _ []byte) error {
			uploaded = true
			return nil
		},
	}

	p := &Predator{infrastructureHandler: infra, GcsClient: gcs, workingEnv: "int"}
	job := &validationjob.Table{ID: 1, GroupID: "10", ServiceName: "svc"}

	result := p.captureAndUploadFailureLogs(job)

	assert.Empty(t, result, "must return empty when ArgoCD fails")
	assert.False(t, uploaded, "must not upload when ArgoCD fails")
}

func TestCapture_AllPodsFailLogFetch_StillUploadsWithErrors(t *testing.T) {
	snap := savePredVars()
	defer snap.restore()
	pred.GcsModelBucket = "bucket"
	pred.AppEnv = "int"

	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{
				Nodes: []infrastructurehandler.Node{
					{Kind: "Pod", Name: "pod-a", Health: infrastructurehandler.Health{Status: "Degraded"}},
					{Kind: "Pod", Name: "pod-b", Health: infrastructurehandler.Health{Status: "Degraded"}},
				},
			}, nil
		},
		getApplicationLogsFn: func(_, _ string, _ *argocd.ApplicationLogsOptions) ([]argocd.ApplicationLogEntry, error) {
			return nil, errors.New("logs unavailable")
		},
	}
	var capturedData []byte
	gcs := &mockGCSClient{
		uploadFileFn: func(_, _ string, data []byte) error {
			capturedData = data
			return nil
		},
	}

	p := &Predator{infrastructureHandler: infra, GcsClient: gcs, workingEnv: "int"}
	job := &validationjob.Table{ID: 3, GroupID: "50", ServiceName: "svc"}

	result := p.captureAndUploadFailureLogs(job)

	assert.NotEmpty(t, result, "must still upload report even when all log fetches fail")

	output := string(capturedData)
	assert.Contains(t, output, "Pod: pod-a")
	assert.Contains(t, output, "Pod: pod-b")
	assert.Equal(t, 2, strings.Count(output, "[ERROR]"), "each pod should have an [ERROR] entry")
	assert.Contains(t, output, "logs unavailable")
}

func TestCapture_MixedLogFetch_SomeSucceedSomeFail(t *testing.T) {
	snap := savePredVars()
	defer snap.restore()
	pred.GcsModelBucket = "bucket"
	pred.AppEnv = "int"

	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{
				Nodes: []infrastructurehandler.Node{
					{Kind: "Pod", Name: "pod-ok-logs", Health: infrastructurehandler.Health{Status: "Degraded"}},
					{Kind: "Pod", Name: "pod-no-logs", Health: infrastructurehandler.Health{Status: "Degraded"}},
				},
			}, nil
		},
		getApplicationLogsFn: func(_, _ string, opts *argocd.ApplicationLogsOptions) ([]argocd.ApplicationLogEntry, error) {
			if opts.PodName == "pod-ok-logs" {
				return []argocd.ApplicationLogEntry{
					{Result: argocd.ApplicationLogResult{Content: "segfault at 0x0", TimeStampStr: "2024-06-01T12:00:00Z"}},
				}, nil
			}
			return nil, errors.New("pod evicted")
		},
	}
	var capturedData []byte
	gcs := &mockGCSClient{
		uploadFileFn: func(_, _ string, data []byte) error {
			capturedData = data
			return nil
		},
	}

	p := &Predator{infrastructureHandler: infra, GcsClient: gcs, workingEnv: "int"}
	job := &validationjob.Table{ID: 5, GroupID: "88", ServiceName: "svc"}

	result := p.captureAndUploadFailureLogs(job)

	assert.NotEmpty(t, result)

	output := string(capturedData)
	assert.Contains(t, output, "Pod: pod-ok-logs")
	assert.Contains(t, output, "segfault at 0x0")
	assert.Contains(t, output, "Pod: pod-no-logs")
	assert.Contains(t, output, "[ERROR]")
	assert.Contains(t, output, "pod evicted")
}

func TestCapture_EmptyBucket_ReturnsEmpty_NoUpload(t *testing.T) {
	snap := savePredVars()
	defer snap.restore()
	pred.GcsModelBucket = ""
	pred.AppEnv = "int"

	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{
				Nodes: []infrastructurehandler.Node{
					{Kind: "Pod", Name: "pod-bad", Health: infrastructurehandler.Health{Status: "Degraded"}},
				},
			}, nil
		},
		getApplicationLogsFn: func(_, _ string, _ *argocd.ApplicationLogsOptions) ([]argocd.ApplicationLogEntry, error) {
			return []argocd.ApplicationLogEntry{
				{Result: argocd.ApplicationLogResult{Content: "err", TimeStampStr: "2024-01-15T10:30:00Z"}},
			}, nil
		},
	}
	uploaded := false
	gcs := &mockGCSClient{
		uploadFileFn: func(_, _ string, _ []byte) error {
			uploaded = true
			return nil
		},
	}

	p := &Predator{infrastructureHandler: infra, GcsClient: gcs, workingEnv: "int"}
	job := &validationjob.Table{ID: 1, GroupID: "1", ServiceName: "svc"}

	result := p.captureAndUploadFailureLogs(job)

	assert.Empty(t, result, "must return empty when bucket is not configured")
	assert.False(t, uploaded, "must not attempt upload with empty bucket")
}

func TestCapture_MultipleDegradedPods_AllLogsInReport(t *testing.T) {
	snap := savePredVars()
	defer snap.restore()
	pred.GcsModelBucket = "bucket"
	pred.AppEnv = "int"

	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{
				Nodes: []infrastructurehandler.Node{
					{Kind: "Pod", Name: "pod-a", Health: infrastructurehandler.Health{Status: "Degraded"}},
					{Kind: "Pod", Name: "pod-b", Health: infrastructurehandler.Health{Status: "Degraded"}},
					{Kind: "Pod", Name: "pod-c", Health: infrastructurehandler.Health{Status: "Degraded"}},
				},
			}, nil
		},
		getApplicationLogsFn: func(_, _ string, opts *argocd.ApplicationLogsOptions) ([]argocd.ApplicationLogEntry, error) {
			return []argocd.ApplicationLogEntry{
				{Result: argocd.ApplicationLogResult{
					Content:      "crash in " + opts.PodName,
					TimeStampStr: "2024-01-15T10:30:00Z",
				}},
			}, nil
		},
	}
	var capturedData []byte
	gcs := &mockGCSClient{
		uploadFileFn: func(_, _ string, data []byte) error {
			capturedData = data
			return nil
		},
	}

	p := &Predator{infrastructureHandler: infra, GcsClient: gcs, workingEnv: "int"}
	job := &validationjob.Table{ID: 1, GroupID: "42", ServiceName: "svc"}

	result := p.captureAndUploadFailureLogs(job)

	assert.NotEmpty(t, result)
	output := string(capturedData)
	assert.Equal(t, 3, strings.Count(output, "Pod: "))
	assert.Contains(t, output, "crash in pod-a")
	assert.Contains(t, output, "crash in pod-b")
	assert.Contains(t, output, "crash in pod-c")
}

func TestCapture_GCSURIFormat(t *testing.T) {
	snap := savePredVars()
	defer snap.restore()
	pred.GcsModelBucket = "my-bucket"
	pred.AppEnv = "int"

	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{
				Nodes: []infrastructurehandler.Node{
					{Kind: "Pod", Name: "p", Health: infrastructurehandler.Health{Status: "Degraded"}},
				},
			}, nil
		},
		getApplicationLogsFn: func(_, _ string, _ *argocd.ApplicationLogsOptions) ([]argocd.ApplicationLogEntry, error) {
			return nil, nil
		},
	}
	gcs := &mockGCSClient{}

	p := &Predator{infrastructureHandler: infra, GcsClient: gcs, workingEnv: "int"}
	job := &validationjob.Table{ID: 1, GroupID: "55", ServiceName: "my-svc"}

	uri := p.captureAndUploadFailureLogs(job)

	assert.True(t, strings.HasPrefix(uri, "https://storage.cloud.google.com/"), "must start with GCS browser URL")
	assert.Equal(t, "https://storage.cloud.google.com/my-bucket/validation-logs/my-svc/55.log", uri)
}

func TestCapture_NilResourceDetail_ReturnsEmpty(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return nil, nil
		},
	}
	uploaded := false
	gcs := &mockGCSClient{
		uploadFileFn: func(_, _ string, _ []byte) error {
			uploaded = true
			return nil
		},
	}

	p := &Predator{infrastructureHandler: infra, GcsClient: gcs, workingEnv: "int"}
	job := &validationjob.Table{ID: 1, GroupID: "1", ServiceName: "svc"}

	result := p.captureAndUploadFailureLogs(job)

	assert.Empty(t, result)
	assert.False(t, uploaded)
}

func TestCapture_EmptyNodesSlice_ReturnsEmpty(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{Nodes: []infrastructurehandler.Node{}}, nil
		},
	}

	p := &Predator{infrastructureHandler: infra, GcsClient: &mockGCSClient{}, workingEnv: "int"}
	job := &validationjob.Table{ID: 1, GroupID: "1", ServiceName: "svc"}

	result := p.captureAndUploadFailureLogs(job)

	assert.Empty(t, result)
}

// ============================================================
// checkDeploymentHealth
// ============================================================

func TestCheckHealth_AllDeploymentsHealthy(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{
				Nodes: []infrastructurehandler.Node{
					{Kind: "Deployment", Name: "deploy-1", Health: infrastructurehandler.Health{Status: "Healthy"}},
					{Kind: "Pod", Name: "pod-1", Health: infrastructurehandler.Health{Status: "Healthy"}},
				},
			}, nil
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	healthy, err := p.checkDeploymentHealth("svc")

	require.NoError(t, err)
	assert.True(t, healthy)
}

func TestCheckHealth_SomeDeploymentsUnhealthy(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{
				Nodes: []infrastructurehandler.Node{
					{Kind: "Deployment", Name: "deploy-1", Health: infrastructurehandler.Health{Status: "Healthy"}},
					{Kind: "Deployment", Name: "deploy-2", Health: infrastructurehandler.Health{Status: "Degraded"}},
				},
			}, nil
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	healthy, err := p.checkDeploymentHealth("svc")

	require.NoError(t, err)
	assert.False(t, healthy)
}

func TestCheckHealth_APIError(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return nil, errors.New("timeout")
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	healthy, err := p.checkDeploymentHealth("svc")

	require.Error(t, err)
	assert.False(t, healthy)
	assert.Contains(t, err.Error(), "failed to get resource detail")
}

func TestCheckHealth_NilResourceDetail_ReturnsFalse(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return nil, nil
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	healthy, err := p.checkDeploymentHealth("svc")

	require.NoError(t, err)
	assert.False(t, healthy)
}

func TestCheckHealth_EmptyNodes_ReturnsFalse(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{Nodes: []infrastructurehandler.Node{}}, nil
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	healthy, err := p.checkDeploymentHealth("svc")

	require.NoError(t, err)
	assert.False(t, healthy)
}

func TestCheckHealth_ZeroDeployments_ReturnsTrue(t *testing.T) {
	infra := &mockInfrastructureHandler{
		getResourceDetailFn: func(_, _ string) (*infrastructurehandler.ResourceDetail, error) {
			return &infrastructurehandler.ResourceDetail{
				Nodes: []infrastructurehandler.Node{
					{Kind: "Pod", Name: "pod-1", Health: infrastructurehandler.Health{Status: "Healthy"}},
					{Kind: "ReplicaSet", Name: "rs-1", Health: infrastructurehandler.Health{Status: "Healthy"}},
				},
			}, nil
		},
	}
	p := &Predator{infrastructureHandler: infra, workingEnv: "int"}

	healthy, err := p.checkDeploymentHealth("svc")

	require.NoError(t, err)
	// 0 total == 0 healthy → returns true (edge: vacuously healthy)
	assert.True(t, healthy)
}  
