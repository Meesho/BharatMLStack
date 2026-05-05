package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
)

const (
	StatusPending      = "PENDING"
	StatusApproved     = "APPROVED"
	StatusRejected     = "REJECTED"
	StatusInProgress   = "IN_PROGRESS"
	StatusCompleted    = "COMPLETED"
	StatusFailed       = "FAILED"
	StatusSuccess      = "SUCCESS"
	RequestTypeCreate  = "CREATE"
	RequestTypeEdit    = "EDIT"
	RequestTypeDelete  = "DELETE"
	RequestTypePromote = "PROMOTE"
)

// RequestInfo holds common fields extracted from various request types
type RequestInfo struct {
	RequestID int
	Status    string
	Payload   string
	CreatedAt time.Time
}

// createRequest is a generic helper for creating any type of request
// validateFn is optional - if provided, it will be called before creating the request
func createRequest(
	payload interface{},
	reason, requestor, requestType, entityName string,
	createFn func(payloadJSON string) (int, time.Time, error),
	validateFn func(payload interface{}) error,
) (RequestStatus, error) {
	// Validate payload if validation function is provided
	if validateFn != nil {
		if err := validateFn(payload); err != nil {
			return RequestStatus{}, fmt.Errorf("validation failed for %s request: %w", entityName, err)
		}
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to marshal payload: %w", err)
	}

	requestID, createdAt, err := createFn(string(payloadJSON))
	if err != nil {
		return RequestStatus{}, fmt.Errorf("failed to create %s request: %w", entityName, err)
	}

	return RequestStatus{
		RequestID: requestID,
		Status:    StatusPending,
		Message:   fmt.Sprintf("%s request submitted successfully. Awaiting admin approval.", entityName),
		CreatedAt: createdAt,
	}, nil
}

// approveRequest is a generic helper for approving any type of request
func approveRequest(
	requestID int,
	approval ApprovalRequest,
	entityName string,
	getRequest func(int) (RequestInfo, error),
	updateStatus func(int, string, string) error,
	onApprove func(payloadJSON string) error,
) (ApprovalResponse, error) {
	reqInfo, err := getRequest(requestID)
	if err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to get %s request: %w", entityName, err)
	}

	if reqInfo.Status != StatusPending {
		return ApprovalResponse{}, fmt.Errorf("request %d is not in pending status", requestID)
	}

	// Handle rejection
	if approval.ApprovalDecision == StatusRejected {
		if err := updateStatus(requestID, approval.ApprovalDecision, approval.AdminID); err != nil {
			return ApprovalResponse{}, fmt.Errorf("failed to update %s request status: %w", entityName, err)
		}
		return ApprovalResponse{
			RequestID:        requestID,
			Status:           StatusRejected,
			Message:          fmt.Sprintf("%s request rejected.", entityName),
			ApprovedBy:       approval.AdminID,
			ApprovedAt:       time.Now(),
			ProcessingStatus: StatusCompleted,
		}, nil
	}

	// Handle approval
	if approval.ApprovalDecision == StatusApproved && onApprove != nil {
		if err := onApprove(reqInfo.Payload); err != nil {
			return ApprovalResponse{}, err
		}
	}

	if err := updateStatus(requestID, approval.ApprovalDecision, approval.AdminID); err != nil {
		return ApprovalResponse{}, fmt.Errorf("failed to update %s request status: %w", entityName, err)
	}

	return ApprovalResponse{
		RequestID:        requestID,
		Status:           approval.ApprovalDecision,
		Message:          fmt.Sprintf("%s request processed successfully.", entityName),
		ApprovedBy:       approval.AdminID,
		ApprovedAt:       time.Now(),
		ProcessingStatus: StatusApproved,
	}, nil
}

// initRepository is a generic helper for initializing repositories
func initRepository[T any](name string, initFn func(*infra.SQLConnection) (T, error), sqlConn *infra.SQLConnection) T {
	repo, err := initFn(sqlConn)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to create %s repository", name)
	}
	return repo
}

// parsePayload unmarshals JSON payload into the given type
func parsePayload[T any](payloadJSON string) (T, error) {
	var payload T
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return payload, fmt.Errorf("failed to parse payload: %w", err)
	}
	return payload, nil
}

// parseMQIdTopicsMapping parses the MQ ID to topics mapping from config string
// Expected format: "1:topic1,2:topic2,3:topic3"
func parseMQIdTopicsMapping(configStr string) map[int]string {
	mapping := make(map[int]string)
	if configStr == "" {
		return mapping
	}

	pairs := splitAndTrim(configStr, ",")
	for _, pair := range pairs {
		parts := splitAndTrim(pair, ":")
		if len(parts) != 2 {
			continue
		}
		mqID, err := strconv.Atoi(parts[0])
		if err != nil {
			log.Warn().Str("pair", pair).Msg("Invalid MQ ID in mapping, skipping")
			continue
		}
		mapping[mqID] = parts[1]
	}
	return mapping
}

// parseVariantsList parses the variants list from config string
// Expected format: "variant1,variant2,variant3"
func parseVariantsList(configStr string) []string {
	if configStr == "" {
		return []string{}
	}
	return splitAndTrim(configStr, ",")
}

// splitAndTrim splits a string and trims whitespace from each part
func splitAndTrim(s string, sep string) []string {
	parts := make([]string, 0)
	for _, part := range strings.Split(s, sep) {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// parseHorizonToSkyeScyllaConfIdMap parses the Horizon to Skye Scylla config ID mapping from config string
// Expected format: "2:1,3:2" (maps horizon confId 2 to skye storeKey 1, horizon confId 3 to skye storeKey 2)
func parseHorizonToSkyeScyllaConfIdMap(configStr string) map[int]int {
	mapping := make(map[int]int)
	if configStr == "" {
		return mapping
	}

	pairs := splitAndTrim(configStr, ",")
	for _, pair := range pairs {
		parts := splitAndTrim(pair, ":")
		if len(parts) != 2 {
			continue
		}
		horizonConfId, err := strconv.Atoi(parts[0])
		if err != nil {
			log.Warn().Str("pair", pair).Msg("Invalid horizon conf ID in mapping, skipping")
			continue
		}
		skyeStoreKey, err := strconv.Atoi(parts[1])
		if err != nil {
			log.Warn().Str("pair", pair).Msg("Invalid skye store key in mapping, skipping")
			continue
		}
		mapping[horizonConfId] = skyeStoreKey
	}
	return mapping
}
