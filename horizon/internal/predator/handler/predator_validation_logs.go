package handler

import (
	"fmt"
	"path"
	"strings"
	"time"

	infrastructurehandler "github.com/Meesho/BharatMLStack/horizon/internal/infrastructure/handler"
	pred "github.com/Meesho/BharatMLStack/horizon/internal/predator"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/validationjob"
	"github.com/Meesho/BharatMLStack/horizon/pkg/argocd"
	"github.com/rs/zerolog/log"
)

const (
	validationLogsDir = "validation-logs"
	istOffsetSeconds  = 5*60*60 + 30*60
	istTimeFmt        = "2006-01-02 15:04:05 IST"
	podSeparator      = "=============================="
	headerSeparator   = "========================================"
)

var istLocation = time.FixedZone("IST", istOffsetSeconds)

type podLogEntry struct {
	name string
	logs string
	err  string
}

// captureAndUploadFailureLogs fetches previous container logs from degraded pods
// of the test deployable and uploads them to GCS in a human-readable format.
// Errors are logged but never propagated so that the validation failure flow
// is not affected.
func (p *Predator) captureAndUploadFailureLogs(job *validationjob.Table) string {
	if job == nil {
		log.Error().Msg("captureAndUploadFailureLogs: job is nil, skipping log capture")
		return ""
	}

	log.Info().Uint("jobID", job.ID).Str("groupID", job.GroupID).
		Msg("Starting failure log capture for validation job")

	degradedPods := p.findDegradedPods(job.ServiceName)
	if len(degradedPods) == 0 {
		log.Warn().Uint("jobID", job.ID).Str("serviceName", job.ServiceName).
			Msg("No degraded pods found, skipping log capture")
		return ""
	}

	var podLogs []podLogEntry
	for _, pod := range degradedPods {
		entry := podLogEntry{name: pod.Name}

		logs, err := p.fetchDegradedPodLogs(job.ServiceName, pod.Name)
		if err != nil {
			entry.err = fmt.Sprintf("failed to fetch logs: %v", err)
			log.Error().Err(err).Str("pod", pod.Name).
				Msg("Failed to fetch degraded pod logs")
		}
		entry.logs = logs
		podLogs = append(podLogs, entry)
	}

	data := formatValidationLogsPlainText(job, p.workingEnv, podLogs)

	bucket, objectPath := p.buildValidationLogsGCSPath(job)
	if bucket == "" {
		log.Error().Uint("jobID", job.ID).
			Msg("Failed to determine GCS bucket for validation logs")
		return ""
	}

	if err := p.GcsClient.UploadFile(bucket, objectPath, data); err != nil {
		log.Error().Err(err).Str("bucket", bucket).Str("path", objectPath).
			Msg("Failed to upload validation failure logs to GCS")
		return ""
	}

	gcsURL := fmt.Sprintf("https://storage.cloud.google.com/%s/%s", bucket, objectPath)
	log.Info().Str("gcsPath", gcsURL).Uint("jobID", job.ID).
		Msg("Validation failure logs uploaded to GCS")
	return gcsURL
}

// findDegradedPods queries the ArgoCD resource tree and returns only pods
// whose health status is "Degraded".
func (p *Predator) findDegradedPods(serviceName string) []infrastructurehandler.Node {
	resourceDetail, err := p.infrastructureHandler.GetResourceDetail(serviceName, p.workingEnv)
	if err != nil {
		log.Error().Err(err).Str("serviceName", serviceName).
			Msg("Failed to get resource detail for log capture")
		return nil
	}

	if resourceDetail == nil || len(resourceDetail.Nodes) == 0 {
		return nil
	}

	var degraded []infrastructurehandler.Node
	for _, node := range resourceDetail.Nodes {
		if node.Kind == "Pod" && node.Health.Status == "Degraded" {
			degraded = append(degraded, node)
		}
	}
	return degraded
}

// fetchDegradedPodLogs retrieves previous container logs for a degraded pod
// with timestamps converted to IST.
func (p *Predator) fetchDegradedPodLogs(serviceName, podName string) (string, error) {
	opts := &argocd.ApplicationLogsOptions{
		PodName:   podName,
		Container: serviceName,
		Previous:  true,
	}
	entries, err := p.infrastructureHandler.GetApplicationLogs(serviceName, p.workingEnv, opts)
	if err != nil {
		return "", fmt.Errorf("failed to fetch logs for pod %s: %w", podName, err)
	}

	var sb strings.Builder
	for _, e := range entries {
		ts := convertToIST(e.Result.TimeStampStr)
		sb.WriteString(fmt.Sprintf("[%s] %s\n", ts, e.Result.Content))
	}
	return sb.String(), nil
}

// convertToIST parses an RFC3339 timestamp string and formats it in IST.
// Falls back to the raw string if parsing fails, or uses the current IST time
// if the input is empty.
func convertToIST(timestampStr string) string {
	if timestampStr == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, timestampStr)
	if err != nil {
		if t, err = time.Parse(time.RFC3339, timestampStr); err != nil {
			return timestampStr
		}
	}
	return t.In(istLocation).Format(istTimeFmt)
}

// formatValidationLogsPlainText produces a human-readable log report with
// a metadata header followed by per-pod sections.
func formatValidationLogsPlainText(job *validationjob.Table, workingEnv string, pods []podLogEntry) []byte {
	var sb strings.Builder

	sb.WriteString(headerSeparator + "\n")
	sb.WriteString("Validation Failure Log Report\n")
	sb.WriteString(headerSeparator + "\n")
	sb.WriteString(fmt.Sprintf("Group ID     : %s\n", job.GroupID))
	sb.WriteString(fmt.Sprintf("Job ID       : %d\n", job.ID))
	sb.WriteString(fmt.Sprintf("Service Name : %s\n", job.ServiceName))
	sb.WriteString(fmt.Sprintf("Working Env  : %s\n", workingEnv))
	sb.WriteString(fmt.Sprintf("Captured At  : %s\n", time.Now().In(istLocation).Format(istTimeFmt)))
	sb.WriteString(headerSeparator + "\n\n")

	for _, pod := range pods {
		sb.WriteString(podSeparator + "\n")
		sb.WriteString(fmt.Sprintf("Pod: %s\n", pod.name))
		sb.WriteString(podSeparator + "\n")

		if pod.err != "" {
			sb.WriteString(fmt.Sprintf("[ERROR] %s\n", pod.err))
		}
		if pod.logs != "" {
			sb.WriteString(pod.logs)
		}
		sb.WriteString("\n")
	}

	return []byte(sb.String())
}

// buildValidationLogsGCSPath determines the GCS bucket and object path for
// uploading validation failure logs.
// Prod: GcsConfigBucket
// Non-prod: GcsModelBucket
func (p *Predator) buildValidationLogsGCSPath(job *validationjob.Table) (bucket, objectPath string) {
	if p.isNonProductionEnvironment() {
		bucket = pred.GcsModelBucket
		objectPath = path.Join(validationLogsDir, job.ServiceName, job.GroupID+".log")
	} else {
		bucket = pred.GcsConfigBucket
		objectPath = path.Join(validationLogsDir, job.ServiceName, job.GroupID+".log")
	}
	return bucket, objectPath
}
