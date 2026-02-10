package jobs

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_onboarding_tasks"
	skyeEtcd "github.com/Meesho/BharatMLStack/horizon/internal/skye/etcd"
	"github.com/Meesho/BharatMLStack/horizon/internal/skye/etcd/enums"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
)

var (
	variantOnboardingOnce sync.Once
	variantOnboardingJob  *VariantOnboardingJob
)

type VariantOnboardingJob struct {
	taskRepo   variant_onboarding_tasks.VariantOnboardingTaskRepository
	etcdConfig skyeEtcd.Manager
	appConfig  configs.Configs
}

type CollectionStatusPayload struct {
	Status             string `json:"status"`
	IndexedVectorCount int64  `json:"indexed_vector_count"`
	PointsCount        int64  `json:"points_count"`
	SegmentsCount      int64  `json:"segments_count"`
}

func InitVariantOnboardingJob(config configs.Configs) *VariantOnboardingJob {
	variantOnboardingOnce.Do(func() {
		conn, err := infra.SQL.GetConnection()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get SQL connection for variant onboarding job")
		}
		sqlConn := conn.(*infra.SQLConnection)

		taskRepo, err := variant_onboarding_tasks.Repository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create variant onboarding task repository")
		}

		etcdConfig := skyeEtcd.NewEtcdConfig(config)
		variantOnboardingJob = &VariantOnboardingJob{
			taskRepo:   taskRepo,
			etcdConfig: etcdConfig,
			appConfig:  config,
		}
	})
	return variantOnboardingJob
}

func (j *VariantOnboardingJob) Run() {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Msg("panic in variant onboarding job")
		}
	}()
	log.Info().Msg("Starting variant onboarding job")

	// Process all IN_PROGRESS tasks
	inProgressTasks, err := j.taskRepo.GetByStatus("IN_PROGRESS")
	if err == nil && len(inProgressTasks) > 0 {
		log.Info().
			Int("count", len(inProgressTasks)).
			Msg("Found IN_PROGRESS tasks, checking status")

		for _, task := range inProgressTasks {
			if err := j.processInProgressTask(&task); err != nil {
				log.Error().Err(err).
					Int("task_id", task.TaskID).
					Msg("Failed to process IN_PROGRESS task")
			}
		}
	}

	// Now check for pending tasks
	pendingTask, err := j.taskRepo.GetFirstPending()
	if err != nil {
		log.Debug().Err(err).Msg("No pending tasks found")
		return
	}

	// Check if any previous run is running or queued
	if err := j.checkIfPreviousDAGRunIsPending(); err != nil {
		log.Warn().Err(err).
			Int("task_id", pendingTask.TaskID).
			Msg("Previous DAG run is pending, skipping task processing")
		return
	}

	log.Info().
		Int("task_id", pendingTask.TaskID).
		Str("entity", pendingTask.Entity).
		Str("model", pendingTask.Model).
		Str("variant", pendingTask.Variant).
		Msg("Processing pending task")

	if err := j.triggerPrismAndAirflow(pendingTask); err != nil {
		log.Error().Err(err).
			Int("task_id", pendingTask.TaskID).
			Msg("Failed to trigger Prism and Airflow")
		return
	}

	if err := j.taskRepo.UpdateStatus(pendingTask.TaskID, "IN_PROGRESS"); err != nil {
		log.Error().Err(err).
			Int("task_id", pendingTask.TaskID).
			Msg("Failed to update task status to IN_PROGRESS")
		return
	}

	log.Info().
		Int("task_id", pendingTask.TaskID).
		Msg("Successfully triggered Prism and Airflow for task")
}

func (j *VariantOnboardingJob) checkIfPreviousDAGRunIsPending() error {
	airflowClient := externalcall.GetAirflowClient()
	dagRuns, err := airflowClient.ListDAGRuns(j.appConfig.InitialIngestionAirflowDAGID)
	if err != nil {
		return fmt.Errorf("failed to list DAG runs: %w", err)
	}

	if len(dagRuns.DAGRuns) == 0 {
		log.Info().Msg("No previous DAG runs found, proceeding with task")
		return nil
	}

	for _, run := range dagRuns.DAGRuns {
		if run.State == "running" || run.State == "queued" {
			return fmt.Errorf("previous DAG run %s is still in progress (state: %s)", run.DagRunID, run.State)
		}
	}

	return nil
}

func (j *VariantOnboardingJob) processInProgressTask(task *variant_onboarding_tasks.VariantOnboardingTask) error {
	// Parse payload to get airflow_response
	var payload map[string]interface{}
	if task.Payload != "" {
		if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload: %w", err)
		}
	} else {
		payload = make(map[string]interface{})
	}

	// Check Airflow DAG run status first
	dagRunStatus, err := j.checkAirflowDAGRunStatus(payload)
	if err != nil {
		log.Warn().Err(err).
			Int("task_id", task.TaskID).
			Msg("Failed to check Airflow DAG run status, will retry next run")
		return nil // Don't fail the task, just log and retry
	}

	// Update payload with latest DAG run status
	payload["dag_run_status"] = dagRunStatus
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	if err := j.taskRepo.UpdatePayload(task.TaskID, string(payloadJSON)); err != nil {
		return fmt.Errorf("failed to update task payload with dag run status: %w", err)
	}

	// Handle different DAG run states
	switch dagRunStatus.State {
	case "failed":
		if err := j.taskRepo.UpdateStatus(task.TaskID, "FAILED"); err != nil {
			return fmt.Errorf("failed to update task status to FAILED: %w", err)
		}
		log.Error().
			Int("task_id", task.TaskID).
			Str("dag_run_id", dagRunStatus.DagRunID).
			Str("state", dagRunStatus.State).
			Msg("Task failed - Airflow DAG run failed")
		return nil

	case "running", "queued":
		log.Info().
			Int("task_id", task.TaskID).
			Str("dag_run_id", dagRunStatus.DagRunID).
			Str("state", dagRunStatus.State).
			Msg("Airflow DAG still running, will check again next run")
		return nil

	case "success":
		log.Info().
			Int("task_id", task.TaskID).
			Str("dag_run_id", dagRunStatus.DagRunID).
			Msg("Airflow DAG completed successfully, checking collection status")
		// Continue to check collection status
	default:
		log.Warn().
			Int("task_id", task.TaskID).
			Str("dag_run_id", dagRunStatus.DagRunID).
			Str("state", dagRunStatus.State).
			Msg("Unknown DAG run state, will check again next run")
		return nil
	}

	// Get variant details from etcd
	variant, err := j.etcdConfig.GetVariantConfig(task.Entity, task.Model, task.Variant)
	if err != nil {
		return fmt.Errorf("failed to get variants from etcd: %w", err)
	}

	// Get vector DB config
	vectorDBConfig := variant.VectorDbConfig
	collectionName := fmt.Sprintf("%s_%s_%d", task.Variant, task.Model, variant.VectorDbReadVersion)

	// Construct Qdrant API URL using write host
	apiURL := fmt.Sprintf("http://%s:%d/collections/%s", vectorDBConfig.WriteHost, 8080, collectionName)

	log.Info().
		Str("url", apiURL).
		Str("write_host", vectorDBConfig.WriteHost).
		Str("collection_name", collectionName).
		Msg("Checking collection status")

	statusPayload, err := j.checkCollectionStatus(apiURL)
	if err != nil {
		log.Warn().Err(err).
			Str("url", apiURL).
			Msg("Failed to check collection status, will retry next run")
		return nil // Don't fail the task, just log and retry
	}

	payload["collection_status"] = statusPayload.Status
	payload["indexed_vector_count"] = statusPayload.IndexedVectorCount
	payload["points_count"] = statusPayload.PointsCount
	payload["segments_count"] = statusPayload.SegmentsCount
	payloadJSON, err = json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	if err := j.taskRepo.UpdatePayload(task.TaskID, string(payloadJSON)); err != nil {
		return fmt.Errorf("failed to update task payload: %w", err)
	}
	if j.isIndexingComplete(statusPayload) {
		if err := j.taskRepo.UpdateStatus(task.TaskID, "COMPLETED"); err != nil {
			return fmt.Errorf("failed to update task status to COMPLETED: %w", err)
		}
		log.Info().
			Int("task_id", task.TaskID).
			Int64("indexed_count", statusPayload.IndexedVectorCount).
			Int64("points_count", statusPayload.PointsCount).
			Msg("Task completed successfully - indexing complete")
	}

	return nil
}

func (j *VariantOnboardingJob) checkAirflowDAGRunStatus(payload map[string]interface{}) (*externalcall.AirflowDAGRun, error) {
	// Extract airflow_response from payload
	airflowResponse, ok := payload["airflow_response"]
	if !ok {
		return nil, fmt.Errorf("airflow_response not found in payload")
	}

	// Convert airflow_response to map
	airflowResponseMap, ok := airflowResponse.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid airflow_response format in payload")
	}

	// Extract dag_run_id
	dagRunID, ok := airflowResponseMap["dag_run_id"].(string)
	if !ok || dagRunID == "" {
		return nil, fmt.Errorf("dag_run_id not found in airflow_response")
	}

	// Get DAG run status from Airflow
	airflowClient := externalcall.GetAirflowClient()
	dagRun, err := airflowClient.GetDAGRun(j.appConfig.InitialIngestionAirflowDAGID, dagRunID)
	if err != nil {
		return nil, fmt.Errorf("failed to get DAG run status: %w", err)
	}

	log.Info().
		Str("dag_id", dagRun.DagID).
		Str("dag_run_id", dagRun.DagRunID).
		Str("state", dagRun.State).
		Msg("Retrieved Airflow DAG run status")

	return dagRun, nil
}

func (j *VariantOnboardingJob) isIndexingComplete(status *CollectionStatusPayload) bool {
	if status.PointsCount == 0 {
		return false
	}

	// Calculate the difference percentage
	diff := math.Abs(float64(status.IndexedVectorCount - status.PointsCount))
	percentageDiff := (diff / float64(status.PointsCount)) * 100

	// Consider complete if difference is within 5% tolerance
	isComplete := percentageDiff <= 5.0 && status.IndexedVectorCount > 0

	log.Debug().
		Int64("indexed_count", status.IndexedVectorCount).
		Int64("points_count", status.PointsCount).
		Float64("percentage_diff", percentageDiff).
		Bool("is_complete", isComplete).
		Msg("Checking indexing completion status")

	return isComplete
}

func (j *VariantOnboardingJob) checkCollectionStatus(apiURL string) (*CollectionStatusPayload, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// The API response is a JSON object, with most data nested under the "result" key.
	var apiResponse struct {
		Result struct {
			Status             string `json:"status"`
			IndexedVectorCount int64  `json:"indexed_vectors_count"`
			PointsCount        int64  `json:"points_count"`
			SegmentsCount      int64  `json:"segments_count"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	statusPayload := CollectionStatusPayload{
		Status:             apiResponse.Result.Status,
		IndexedVectorCount: apiResponse.Result.IndexedVectorCount,
		PointsCount:        apiResponse.Result.PointsCount,
		SegmentsCount:      apiResponse.Result.SegmentsCount,
	}

	return &statusPayload, nil
}

func (j *VariantOnboardingJob) triggerPrismAndAirflow(task *variant_onboarding_tasks.VariantOnboardingTask) error {
	// Get model config from etcd
	model, err := j.etcdConfig.GetModelConfig(task.Entity, task.Model)
	if err != nil {
		return fmt.Errorf("failed to get model from etcd: %w", err)
	}

	// Get variant config to extract vector_db_type
	variant, err := j.etcdConfig.GetVariantConfig(task.Entity, task.Model, task.Variant)
	if err != nil {
		return fmt.Errorf("failed to get variants from etcd: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	trainingDataPath := model.TrainingDataPath
	if model.ModelType == enums.DELTA {
		trainingDataPath = payload["otd_path"].(string)
	}

	// Prepare Prism parameters
	prismClient := externalcall.GetPrismV2Client()
	parameters := make(map[string]interface{})
	parameters["entity_name"] = task.Entity
	parameters["frequency"] = model.JobFrequency
	parameters["environment"] = j.appConfig.AppEnv
	parameters["vector_db_type"] = variant.VectorDbType
	parameters["model_name"] = task.Model
	parameters["variant_name"] = task.Variant
	parameters["training_data_path"] = trainingDataPath

	// Update Prism step parameters
	if err := prismClient.UpdateStepParameters(j.appConfig.InitialIngestionPrismJobID, j.appConfig.InitialIngestionPrismStepID, parameters); err != nil {
		return fmt.Errorf("failed to update Prism step parameters: %w", err)
	}

	// Trigger Airflow DAG
	airflowClient := externalcall.GetAirflowClient()
	airflowResponse, err := airflowClient.TriggerDAG(j.appConfig.InitialIngestionAirflowDAGID)
	if err != nil {
		return fmt.Errorf("failed to trigger Airflow DAG: %w", err)
	}

	// Add or update the "airflow_response" key

	payload["airflow_response"] = airflowResponse
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	if err := j.taskRepo.UpdatePayload(task.TaskID, string(payloadJSON)); err != nil {
		return fmt.Errorf("failed to update variant onboarding payload with airflow response: %w", err)
	}
	return nil
}
