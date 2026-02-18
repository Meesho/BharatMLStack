package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/internal/externalcall"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/job_locks"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/embedding/variant_scaleup_tasks"
	skyeEtcd "github.com/Meesho/BharatMLStack/horizon/internal/skye/etcd"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
)

var (
	variantScaleUpOnce sync.Once
	variantScaleUpJob  *VariantScaleUpJob
)

type VariantScaleUpJob struct {
	taskRepo   variant_scaleup_tasks.VariantScaleUpTaskRepository
	etcdConfig skyeEtcd.Manager
	appConfig  configs.Configs
	lockRepo   job_locks.JobLocksRepository
}

type ScaleUpCollectionStatusPayload struct {
	Status             string `json:"status"`
	IndexedVectorCount int64  `json:"indexed_vector_count"`
	PointsCount        int64  `json:"points_count"`
	SegmentsCount      int64  `json:"segments_count"`
}

func InitVariantScaleUpJob(config configs.Configs) *VariantScaleUpJob {
	variantScaleUpOnce.Do(func() {
		conn, err := infra.SQL.GetConnection()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get SQL connection for variant scale-up job")
		}
		sqlConn := conn.(*infra.SQLConnection)

		taskRepo, err := variant_scaleup_tasks.Repository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create variant scale-up task repository")
		}

		lockRepo, err := job_locks.Repository(sqlConn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create job locks repository")
		}

		etcdConfig := skyeEtcd.NewEtcdConfig(config)
		variantScaleUpJob = &VariantScaleUpJob{
			taskRepo:   taskRepo,
			etcdConfig: etcdConfig,
			appConfig:  config,
			lockRepo:   lockRepo,
		}
	})
	return variantScaleUpJob
}

func (j *VariantScaleUpJob) Run() {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Msg("panic in variant scale-up job")
		}
	}()
	log.Info().Msg("Starting variant scale-up job")
	if err := j.lockRepo.AcquireLockNowait(context.Background(), "variant_scaleup_job"); err != nil {
		log.Error().Err(err).Msg("Failed to acquire job lock")
		return
	}
	defer func() {
		if err := j.lockRepo.ReleaseLock(context.Background(), "variant_scaleup_job"); err != nil {
			log.Error().Err(err).Msg("Failed to release job lock")
		}
	}()

	// Process all IN_PROGRESS tasks
	inProgressTasks, err := j.taskRepo.GetByStatus("IN_PROGRESS")
	if err == nil && len(inProgressTasks) > 0 {
		log.Info().
			Int("count", len(inProgressTasks)).
			Msg("Found IN_PROGRESS scale-up tasks, checking status")

		for _, task := range inProgressTasks {
			if err := j.processInProgressTask(&task); err != nil {
				log.Error().Err(err).
					Int("task_id", task.TaskID).
					Msg("Failed to process IN_PROGRESS scale-up task")
			}
		}
	}

	// Now check for pending tasks
	pendingTask, err := j.taskRepo.GetFirstPending()
	if err != nil {
		log.Debug().Err(err).Msg("No pending scale-up tasks found")
		return
	}

	// Check if any previous run is running or queued
	if err := j.checkPreviousDAGRunSuccess(); err != nil {
		log.Warn().Err(err).
			Int("task_id", pendingTask.TaskID).
			Msg("Previous scale-up DAG run not successful, skipping task processing")
		return
	}

	log.Info().
		Int("task_id", pendingTask.TaskID).
		Str("entity", pendingTask.Entity).
		Str("model", pendingTask.Model).
		Str("variant", pendingTask.Variant).
		Str("scale_up_host", pendingTask.ScaleUpHost).
		Msg("Processing pending scale-up task")

	if err := j.triggerPrismAndAirflow(pendingTask); err != nil {
		log.Error().Err(err).
			Int("task_id", pendingTask.TaskID).
			Msg("Failed to trigger Prism and Airflow for scale-up")
		return
	}

	if err := j.taskRepo.UpdateStatus(pendingTask.TaskID, "IN_PROGRESS"); err != nil {
		log.Error().Err(err).
			Int("task_id", pendingTask.TaskID).
			Msg("Failed to update scale-up task status to IN_PROGRESS")
		return
	}

	log.Info().
		Int("task_id", pendingTask.TaskID).
		Msg("Successfully triggered Prism and Airflow for scale-up task")
}

func (j *VariantScaleUpJob) checkPreviousDAGRunSuccess() error {
	if j.appConfig.UseSkyeTriggerInsteadOfAirflow {
		return nil
	}
	airflowClient := externalcall.GetAirflowClient()
	// TODO: Update to use scale-up specific Airflow DAG ID from config
	dagRuns, err := airflowClient.ListDAGRuns(j.appConfig.VariantScaleUpAirflowDAGID)
	if err != nil {
		return fmt.Errorf("failed to list scale-up DAG runs: %w", err)
	}

	if len(dagRuns.DAGRuns) == 0 {
		log.Info().Msg("No previous scale-up DAG runs found, proceeding with task")
		return nil
	}

	for _, run := range dagRuns.DAGRuns {
		if run.State == "running" || run.State == "queued" {
			return fmt.Errorf("previous scale-up DAG run %s is still in progress (state: %s)", run.DagRunID, run.State)
		}
	}

	return nil
}

func (j *VariantScaleUpJob) processInProgressTask(task *variant_scaleup_tasks.VariantScaleUpTask) error {
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
			Msg("Failed to check Airflow DAG run status for scale-up, will retry next run")
		return nil // Don't fail the task, just log and retry
	}

	// Update payload with latest DAG run status
	payload["dag_run_status"] = dagRunStatus
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	if err := j.taskRepo.UpdatePayload(task.TaskID, string(payloadJSON)); err != nil {
		return fmt.Errorf("failed to update scale-up task payload with dag run status: %w", err)
	}

	// Handle different DAG run states
	switch dagRunStatus.State {
	case "failed":
		if err := j.taskRepo.UpdateStatus(task.TaskID, "FAILED"); err != nil {
			return fmt.Errorf("failed to update scale-up task status to FAILED: %w", err)
		}
		log.Error().
			Int("task_id", task.TaskID).
			Str("dag_run_id", dagRunStatus.DagRunID).
			Str("state", dagRunStatus.State).
			Msg("Scale-up task failed - Airflow DAG run failed")
		return nil

	case "running", "queued":
		log.Info().
			Int("task_id", task.TaskID).
			Str("dag_run_id", dagRunStatus.DagRunID).
			Str("state", dagRunStatus.State).
			Msg("Airflow DAG still running for scale-up, will check again next run")
		return nil

	case "success":
		log.Info().
			Int("task_id", task.TaskID).
			Str("dag_run_id", dagRunStatus.DagRunID).
			Msg("Airflow DAG completed successfully for scale-up, checking collection status")
		// Continue to check collection status
	default:
		log.Warn().
			Int("task_id", task.TaskID).
			Str("dag_run_id", dagRunStatus.DagRunID).
			Str("state", dagRunStatus.State).
			Msg("Unknown DAG run state for scale-up, will check again next run")
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

	// Use the scale-up host from task instead of write host
	apiURL := fmt.Sprintf("http://%s:%d/collections/%s", task.ScaleUpHost, 8080, collectionName)

	log.Info().
		Str("url", apiURL).
		Str("scale_up_host", task.ScaleUpHost).
		Str("collection_name", collectionName).
		Msg("Checking scale-up collection status")

	statusPayload, err := j.checkCollectionStatus(apiURL)
	if err != nil {
		log.Warn().Err(err).
			Str("url", apiURL).
			Msg("Failed to check scale-up collection status, will retry next run")
		return nil // Don't fail the task, just log and retry
	}

	payload["collection_status"] = statusPayload
	payloadJSON, err = json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	if err := j.taskRepo.UpdatePayload(task.TaskID, string(payloadJSON)); err != nil {
		return fmt.Errorf("failed to update scale-up task payload: %w", err)
	}

	// Compare with original write host collection to verify scale-up completion
	originalURL := fmt.Sprintf("http://%s:%d/collections/%s", vectorDBConfig.ReadHost, 8080, collectionName)
	originalStatus, err := j.checkCollectionStatus(originalURL)
	if err != nil {
		log.Warn().Err(err).
			Str("url", originalURL).
			Msg("Failed to check original collection status for comparison")
		return nil
	}

	if j.isScaleUpComplete(statusPayload, originalStatus) {
		if err := j.taskRepo.UpdateStatus(task.TaskID, "COMPLETED"); err != nil {
			return fmt.Errorf("failed to update scale-up task status to COMPLETED: %w", err)
		}
		log.Info().
			Int("task_id", task.TaskID).
			Int64("scale_up_indexed_count", statusPayload.IndexedVectorCount).
			Int64("original_indexed_count", originalStatus.IndexedVectorCount).
			Msg("Scale-up task completed successfully - indexing complete")
	}

	return nil
}

func (j *VariantScaleUpJob) checkAirflowDAGRunStatus(payload map[string]interface{}) (*externalcall.AirflowDAGRun, error) {
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

	// OSS: when using skye-trigger, we stored a synthetic response; return success without calling Airflow
	if dagRunID == externalcall.SkyeTriggerDAGRunID {
		return &externalcall.AirflowDAGRun{
			DagID:    j.appConfig.VariantScaleUpAirflowDAGID,
			DagRunID: externalcall.SkyeTriggerDAGRunID,
			State:    "success",
		}, nil
	}

	// Get DAG run status from Airflow
	airflowClient := externalcall.GetAirflowClient()
	dagRun, err := airflowClient.GetDAGRun(j.appConfig.VariantScaleUpAirflowDAGID, dagRunID)
	if err != nil {
		return nil, fmt.Errorf("failed to get DAG run status: %w", err)
	}

	log.Info().
		Str("dag_id", dagRun.DagID).
		Str("dag_run_id", dagRun.DagRunID).
		Str("state", dagRun.State).
		Msg("Retrieved Airflow DAG run status for scale-up")

	return dagRun, nil
}

func (j *VariantScaleUpJob) isScaleUpComplete(scaleUpStatus, originalStatus *ScaleUpCollectionStatusPayload) bool {
	if scaleUpStatus.PointsCount == 0 || originalStatus.PointsCount == 0 {
		return false
	}

	// Calculate the difference percentage between scale-up and original
	diff := math.Abs(float64(scaleUpStatus.IndexedVectorCount - originalStatus.IndexedVectorCount))
	percentageDiff := (diff / float64(originalStatus.IndexedVectorCount)) * 100

	// Also check that scale-up indexing itself is complete
	selfDiff := math.Abs(float64(scaleUpStatus.IndexedVectorCount - scaleUpStatus.PointsCount))
	selfPercentageDiff := (selfDiff / float64(scaleUpStatus.PointsCount)) * 100

	// Consider complete if both conditions are met within 5% tolerance
	isComplete := percentageDiff <= 5.0 && selfPercentageDiff <= 5.0 && scaleUpStatus.IndexedVectorCount > 0

	log.Debug().
		Int64("scale_up_indexed_count", scaleUpStatus.IndexedVectorCount).
		Int64("original_indexed_count", originalStatus.IndexedVectorCount).
		Float64("comparison_percentage_diff", percentageDiff).
		Float64("self_percentage_diff", selfPercentageDiff).
		Bool("is_complete", isComplete).
		Msg("Checking scale-up completion status")

	return isComplete
}

func (j *VariantScaleUpJob) checkCollectionStatus(apiURL string) (*ScaleUpCollectionStatusPayload, error) {
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

	var statusPayload ScaleUpCollectionStatusPayload
	if err := json.Unmarshal(body, &statusPayload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &statusPayload, nil
}

func (j *VariantScaleUpJob) triggerPrismAndAirflow(task *variant_scaleup_tasks.VariantScaleUpTask) error {
	// Prism flow: only when not using OSS skye-trigger
	if !j.appConfig.UseSkyeTriggerInsteadOfAirflow {
		model, err := j.etcdConfig.GetModelConfig(task.Entity, task.Model)
		if err != nil {
			return fmt.Errorf("failed to get model from etcd: %w", err)
		}
		prismClient := externalcall.GetPrismV2Client()
		parameters := make(map[string]interface{})
		parameters["entity_name"] = task.Entity
		parameters["frequency"] = model.JobFrequency
		parameters["environment"] = j.appConfig.AppEnv
		parameters["vector_db_type"] = task.VectorDBType
		parameters["model_name"] = task.Model
		parameters["variant_name"] = task.Variant
		parameters["training_data_path"] = task.TrainingDataPath
		parameters["host"] = task.ScaleUpHost
		if err := prismClient.UpdateStepParameters(j.appConfig.VariantScaleUpPrismJobID, j.appConfig.VariantScaleUpPrismStepID, parameters); err != nil {
			return fmt.Errorf("failed to update Prism step parameters for scale-up: %w", err)
		}
	}

	var payload map[string]interface{}
	if task.Payload != "" {
		if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
			payload = make(map[string]interface{})
		}
	} else {
		payload = make(map[string]interface{})
	}

	var airflowResponse interface{}
	if j.appConfig.UseSkyeTriggerInsteadOfAirflow {
		// OSS: call skye-trigger instead of Airflow
		if err := externalcall.TriggerSkyeTrigger(
			j.appConfig.SkyeTriggerURL,
			task.Entity,
			task.Model,
			[]string{task.Variant},
			j.appConfig.AppEnv,
		); err != nil {
			return fmt.Errorf("failed to trigger skye-trigger for scale-up: %w", err)
		}
		airflowResponse = map[string]interface{}{
			"dag_run_id": externalcall.SkyeTriggerDAGRunID,
			"state":      "success",
		}
	} else {
		// Trigger Airflow DAG
		airflowClient := externalcall.GetAirflowClient()
		var err error
		airflowResponse, err = airflowClient.TriggerDAG(j.appConfig.VariantScaleUpAirflowDAGID)
		if err != nil {
			return fmt.Errorf("failed to trigger Airflow DAG for scale-up: %w", err)
		}
	}
	payload["airflow_response"] = airflowResponse
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	if err := j.taskRepo.UpdatePayload(task.TaskID, string(payloadJSON)); err != nil {
		return fmt.Errorf("failed to update variant scale-up payload with airflow response: %w", err)
	}
	return nil
}
