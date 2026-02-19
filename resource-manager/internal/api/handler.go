package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/application"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	rmerrors "github.com/Meesho/BharatMLStack/resource-manager/internal/errors"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/ports"
	rmtypes "github.com/Meesho/BharatMLStack/resource-manager/internal/types"
)

const idempotencyHeader = "X-Idempotency-Key"

type Handler struct {
	shadows     *application.ShadowService
	operations  *application.OperationService
	idempotency ports.IdempotencyKeyStore
}

func NewHandler(shadows *application.ShadowService, operations *application.OperationService, idempotency ports.IdempotencyKeyStore) *Handler {
	return &Handler{
		shadows:     shadows,
		operations:  operations,
		idempotency: idempotency,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health/self", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"message": "true"})
	})
	mux.HandleFunc("/api/1.0/", h.handleAPI)
}

func (h *Handler) handleAPI(w http.ResponseWriter, r *http.Request) {
	env, route, ok := parseRoute(r.URL.Path)
	if !ok {
		writeErr(w, http.StatusNotFound, "route not found")
		return
	}
	if !rmtypes.IsSupportedPoolEnv(env) {
		writeErr(w, http.StatusBadRequest, "env must be one of: int, prod")
		return
	}
	env = string(rmtypes.NormalizePoolEnv(env))
	route = normalizeRoute(route)

	switch {
	case route == "shadow-deployables" && r.Method == http.MethodGet:
		h.listShadowDeployables(w, r, env)
	case route == "shadow-deployables" && r.Method == http.MethodPost:
		h.mutateShadowDeployables(w, r, env)
	case route == "shadow-deployables/min-pod-count" && r.Method == http.MethodPost:
		h.changeMinPodCount(w, r, env)
	case route == "deployables" && r.Method == http.MethodPost:
		h.createDeployable(w, r, env)
	case route == "deployables/restart" && r.Method == http.MethodPost:
		h.restartDeployable(w, r, env)
	case route == "models/load" && r.Method == http.MethodPost:
		h.loadModel(w, r, env)
	case route == "jobs" && r.Method == http.MethodPost:
		h.triggerJob(w, r, env)
	default:
		writeErr(w, http.StatusNotFound, "route not found")
	}
}

func (h *Handler) listShadowDeployables(w http.ResponseWriter, r *http.Request, env string) {
	filter := models.ShadowFilter{
		NodePool: strings.TrimSpace(r.URL.Query().Get("node_pool")),
	}

	if inUseRaw := strings.TrimSpace(r.URL.Query().Get("in_use")); inUseRaw != "" {
		parsed, err := strconv.ParseBool(inUseRaw)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid in_use query param")
			return
		}
		filter.InUse = &parsed
	}

	items, err := h.shadows.List(r.Context(), env, filter)
	if err != nil {
		if errors.Is(err, rmerrors.ErrUnsupportedEnv) {
			writeErr(w, http.StatusBadRequest, "env must be one of: int, prod")
			return
		}
		writeErr(w, http.StatusInternalServerError, "failed to list shadow deployables")
		return
	}

	response := ListShadowDeployablesResponse{Data: make([]ShadowDeployableView, 0, len(items))}
	for _, item := range items {
		response.Data = append(response.Data, ShadowDeployableView{
			Name:     item.Name,
			DNS:      item.DNS,
			PodCount: item.MinPodCount,
			InUse:    item.State == rmtypes.ShadowStateProcured,
			NodePool: item.NodePool,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) mutateShadowDeployables(w http.ResponseWriter, r *http.Request, env string) {
	var req ShadowDeployableCommandRequest
	hash, reused, done := h.decodeWithIdempotency(w, r, env+"/shadow-deployables", &req)
	if done {
		return
	}

	if req.Name == "" || req.WorkflowRunID == "" {
		writeErr(w, http.StatusBadRequest, "name and workflow_run_id are required")
		return
	}

	var response ShadowDeployableCommandResponse
	var status = http.StatusOK
	var err error

	switch req.Action {
	case rmtypes.ActionProcure:
		_, conflict, opErr := h.shadows.Procure(r.Context(), env, req.Name, req.WorkflowRunID, req.WorkflowPlan)
		err = opErr
		if conflict {
			response = ShadowDeployableCommandResponse{
				Status: "FAILED",
				Delay:  300,
			}
		} else {
			response = ShadowDeployableCommandResponse{Status: "PROCURED"}
		}
	case rmtypes.ActionRelease:
		_, conflict, opErr := h.shadows.Release(r.Context(), env, req.Name, req.WorkflowRunID)
		err = opErr
		if conflict {
			response = ShadowDeployableCommandResponse{Status: "FAILED", Delay: 300}
		} else {
			response = ShadowDeployableCommandResponse{Status: "RELEASED"}
		}
	default:
		writeErr(w, http.StatusBadRequest, "action must be PROCURE or RELEASE")
		return
	}

	if err != nil {
		switch {
		case errors.Is(err, rmerrors.ErrUnsupportedEnv):
			writeErr(w, http.StatusBadRequest, "env must be one of: int, prod")
		case errors.Is(err, rmerrors.ErrNotFound):
			writeErr(w, http.StatusBadRequest, "shadow deployable not found")
		case errors.Is(err, rmerrors.ErrInvalidOwner):
			writeErr(w, http.StatusBadRequest, "caller is not the current owner")
		default:
			writeErr(w, http.StatusInternalServerError, "failed to process shadow deployable action")
		}
		return
	}

	writeJSON(w, status, response)
	if !reused {
		_ = h.idempotency.Put(r.Context(), env+"/shadow-deployables", r.Header.Get(idempotencyHeader), models.IdempotencyRecord{
			RequestHash:  hash,
			StatusCode:   status,
			ResponseBody: mustJSON(response),
			ContentType:  "application/json",
		})
	}
}

func (h *Handler) changeMinPodCount(w http.ResponseWriter, r *http.Request, env string) {
	var req MinPodCountRequest
	hash, reused, done := h.decodeWithIdempotency(w, r, env+"/shadow-deployables/min-pod-count", &req)
	if done {
		return
	}

	if req.Name == "" || req.WorkflowRunID == "" {
		writeErr(w, http.StatusBadRequest, "name and workflow_run_id are required")
		return
	}
	if req.Count < 0 {
		writeErr(w, http.StatusBadRequest, "count must be >= 0")
		return
	}

	_, err := h.shadows.ChangeMinPodCount(r.Context(), env, req.Name, req.Action, req.Count)
	if err != nil {
		switch {
		case errors.Is(err, rmerrors.ErrUnsupportedEnv):
			writeErr(w, http.StatusBadRequest, "env must be one of: int, prod")
		case errors.Is(err, rmerrors.ErrNotFound):
			writeErr(w, http.StatusBadRequest, "shadow deployable not found")
		case errors.Is(err, rmerrors.ErrInvalidAction):
			writeErr(w, http.StatusBadRequest, "invalid action")
		default:
			writeErr(w, http.StatusInternalServerError, "failed to update pod count")
		}
		return
	}

	response := map[string]string{"status": "UPDATED"}
	writeJSON(w, http.StatusOK, response)
	if !reused {
		_ = h.idempotency.Put(r.Context(), env+"/shadow-deployables/min-pod-count", r.Header.Get(idempotencyHeader), models.IdempotencyRecord{
			RequestHash:  hash,
			StatusCode:   http.StatusOK,
			ResponseBody: mustJSON(response),
			ContentType:  "application/json",
		})
	}
}

func (h *Handler) createDeployable(w http.ResponseWriter, r *http.Request, env string) {
	var req CreateDeployableRequest
	hash, reused, done := h.decodeWithIdempotency(w, r, env+"/deployables", &req)
	if done {
		return
	}
	if req.Name == "" || req.WorkflowRunID == "" || req.Callback.URL == "" {
		writeErr(w, http.StatusBadRequest, "name, workflow_run_id and callback.url are required")
		return
	}
	if err := validateCallback(req.Callback); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.MaxPodCount < req.MinPodCount {
		writeErr(w, http.StatusBadRequest, "max_pod_count must be >= min_pod_count")
		return
	}

	if err := h.operations.CreateDeployable(r.Context(), env, models.DeployableSpec{
		Name:         req.Name,
		Namespace:    env,
		NodeSelector: req.NodeSelector,
		MinPodCount:  req.MinPodCount,
		MaxPodCount:  req.MaxPodCount,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to submit deployable operation")
		return
	}

	requestID := RequestIDFromContext(r.Context())
	_, err := h.operations.SubmitAsyncOperation(
		r.Context(),
		requestID,
		"CREATE_DEPLOYABLE",
		models.WatchResource{
			Kind:          "Deployment",
			Namespace:     env,
			LabelSelector: "name=" + req.Name,
			Name:          req.Name,
		},
		"Available",
		toDomainCallback(req.Callback),
		models.WorkflowContext{RunID: req.WorkflowRunID, Plan: req.WorkflowPlan, TaskID: req.WorkflowTaskID},
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to accept deployable creation request")
		return
	}

	response := AsyncAcceptedResponse{
		Message: "Deployable creation accepted",
		Name:    req.Name,
		Status:  "IN_PROGRESS",
	}
	writeJSON(w, http.StatusAccepted, response)
	if !reused {
		_ = h.idempotency.Put(r.Context(), env+"/deployables", r.Header.Get(idempotencyHeader), models.IdempotencyRecord{
			RequestHash:  hash,
			StatusCode:   http.StatusAccepted,
			ResponseBody: mustJSON(response),
			ContentType:  "application/json",
		})
	}
}

func (h *Handler) loadModel(w http.ResponseWriter, r *http.Request, env string) {
	var req LoadModelRequest
	hash, reused, done := h.decodeWithIdempotency(w, r, env+"/models/load", &req)
	if done {
		return
	}
	if req.DeployableName == "" || req.WorkflowRunID == "" || req.Callback.URL == "" || req.Model.Name == "" || req.Model.Version == "" {
		writeErr(w, http.StatusBadRequest, "deployable_name, model.name, model.version, workflow_run_id and callback.url are required")
		return
	}
	if err := validateCallback(req.Callback); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.operations.LoadModel(r.Context(), env, models.ModelLoadSpec{
		DeployableName: req.DeployableName,
		ModelName:      req.Model.Name,
		ModelVersion:   req.Model.Version,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to submit model load operation")
		return
	}

	requestID := RequestIDFromContext(r.Context())
	_, err := h.operations.SubmitAsyncOperation(
		r.Context(),
		requestID,
		"LOAD_MODEL",
		models.WatchResource{
			Kind:          "Pod",
			Namespace:     env,
			LabelSelector: "deployable_name=" + req.DeployableName,
			Name:          req.DeployableName,
		},
		"Ready",
		toDomainCallback(req.Callback),
		models.WorkflowContext{RunID: req.WorkflowRunID, Plan: req.WorkflowPlan, TaskID: req.WorkflowTaskID},
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to accept model load request")
		return
	}

	response := AsyncAcceptedResponse{
		Message: "Model load request accepted",
		Name:    req.DeployableName,
		Status:  "IN_PROGRESS",
	}
	writeJSON(w, http.StatusAccepted, response)
	if !reused {
		_ = h.idempotency.Put(r.Context(), env+"/models/load", r.Header.Get(idempotencyHeader), models.IdempotencyRecord{
			RequestHash:  hash,
			StatusCode:   http.StatusAccepted,
			ResponseBody: mustJSON(response),
			ContentType:  "application/json",
		})
	}
}

func (h *Handler) triggerJob(w http.ResponseWriter, r *http.Request, env string) {
	var req TriggerJobRequest
	hash, reused, done := h.decodeWithIdempotency(w, r, env+"/jobs", &req)
	if done {
		return
	}
	if req.JobName == "" || req.WorkflowRunID == "" || req.Callback.URL == "" {
		writeErr(w, http.StatusBadRequest, "job_name, workflow_run_id and callback.url are required")
		return
	}
	if err := validateCallback(req.Callback); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.operations.TriggerJob(r.Context(), env, models.JobSpec{
		JobName: req.JobName,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to submit job operation")
		return
	}

	requestID := RequestIDFromContext(r.Context())
	_, err := h.operations.SubmitAsyncOperation(
		r.Context(),
		requestID,
		"TRIGGER_JOB",
		models.WatchResource{
			Kind:          "Job",
			Namespace:     env,
			LabelSelector: "job_name=" + req.JobName,
			Name:          req.JobName,
		},
		"Complete",
		toDomainCallback(req.Callback),
		models.WorkflowContext{RunID: req.WorkflowRunID, Plan: req.WorkflowPlan, TaskID: req.WorkflowTaskID},
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to submit job request")
		return
	}

	response := AsyncAcceptedResponse{
		Message: "Job creation request accepted",
		Name:    req.JobName,
		Status:  "IN_PROGRESS",
	}
	writeJSON(w, http.StatusAccepted, response)
	if !reused {
		_ = h.idempotency.Put(r.Context(), env+"/jobs", r.Header.Get(idempotencyHeader), models.IdempotencyRecord{
			RequestHash:  hash,
			StatusCode:   http.StatusAccepted,
			ResponseBody: mustJSON(response),
			ContentType:  "application/json",
		})
	}
}

func (h *Handler) restartDeployable(w http.ResponseWriter, r *http.Request, env string) {
	var req RestartDeployableRequest
	hash, reused, done := h.decodeWithIdempotency(w, r, env+"/deployables/restart", &req)
	if done {
		return
	}
	if req.Namespace == "" || req.WorkflowRunID == "" || req.Callback.URL == "" {
		writeErr(w, http.StatusBadRequest, "namespace, workflow_run_id and callback.url are required")
		return
	}
	if err := validateCallback(req.Callback); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.operations.RestartDeployable(r.Context(), env, models.RestartSpec{
		Namespace: req.Namespace,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to submit restart operation")
		return
	}

	requestID := RequestIDFromContext(r.Context())
	_, err := h.operations.SubmitAsyncOperation(
		r.Context(),
		requestID,
		"RESTART_DEPLOYABLE",
		models.WatchResource{
			Kind:          "Deployment",
			Namespace:     req.Namespace,
			LabelSelector: "namespace=" + req.Namespace,
			Name:          req.Namespace,
		},
		"Progressing",
		toDomainCallback(req.Callback),
		models.WorkflowContext{RunID: req.WorkflowRunID, Plan: req.WorkflowPlan, TaskID: req.WorkflowTaskID},
	)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to submit restart request")
		return
	}

	response := AsyncAcceptedResponse{
		Message: "Restart request accepted",
		Name:    req.Namespace,
		Status:  "IN_PROGRESS",
	}
	writeJSON(w, http.StatusAccepted, response)
	if !reused {
		_ = h.idempotency.Put(r.Context(), env+"/deployables/restart", r.Header.Get(idempotencyHeader), models.IdempotencyRecord{
			RequestHash:  hash,
			StatusCode:   http.StatusAccepted,
			ResponseBody: mustJSON(response),
			ContentType:  "application/json",
		})
	}
}

func parseRoute(path string) (string, string, bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 4 || parts[0] != "api" || parts[1] != "1.0" {
		return "", "", false
	}
	env := parts[2]
	route := strings.Join(parts[3:], "/")
	if env == "" || route == "" {
		return "", "", false
	}
	return env, route, true
}

func normalizeRoute(route string) string {
	// Keep support for the typo shared in early contracts.
	route = strings.ReplaceAll(route, "shadow-deploybles", "shadow-deployables")
	return route
}

func toDomainCallback(c CallbackInput) models.Callback {
	method := strings.ToUpper(strings.TrimSpace(c.Method))
	if method == "" {
		method = "POST"
	}
	return models.Callback{
		URL:     c.URL,
		Method:  method,
		Headers: c.Headers,
	}
}

func validateCallback(cb CallbackInput) error {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(cb.URL))
	if err != nil {
		return fmt.Errorf("invalid callback.url")
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("callback.url must use https")
	}
	if parsed.Hostname() == "" {
		return fmt.Errorf("callback.url host is required")
	}
	if isBlockedHost(parsed.Hostname()) {
		return fmt.Errorf("callback.url points to blocked host")
	}
	return nil
}

func isBlockedHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		return isBlockedIP(ip)
	}
	// DNS resolution based checks are performed by watcher/callback executor,
	// which is the component making outbound network calls.
	return false
}

func isBlockedIP(ip netip.Addr) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

func (h *Handler) decodeWithIdempotency(w http.ResponseWriter, r *http.Request, scope string, out interface{}) (string, bool, bool) {
	key := strings.TrimSpace(r.Header.Get(idempotencyHeader))
	if key == "" {
		writeErr(w, http.StatusBadRequest, idempotencyHeader+" header is required")
		return "", false, true
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return "", false, true
	}
	defer r.Body.Close()

	hash := sha256.Sum256(body)
	requestHash := hex.EncodeToString(hash[:])

	record, err := h.idempotency.Get(r.Context(), scope, key)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "idempotency lookup failed")
		return "", false, true
	}
	if record != nil {
		if record.RequestHash != requestHash {
			writeErr(w, http.StatusConflict, rmerrors.ErrIdempotencyMismatch.Error())
			return "", false, true
		}
		w.Header().Set("Content-Type", record.ContentType)
		w.WriteHeader(record.StatusCode)
		_, _ = w.Write(record.ResponseBody)
		return requestHash, true, true
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request payload")
		return "", false, true
	}

	return requestHash, false, false
}

func writeErr(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	raw, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal serialization error"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func mustJSON(payload interface{}) []byte {
	raw, _ := json.Marshal(payload)
	return raw
}

type requestIDContextKey string

const requestIDKey requestIDContextKey = "request_id"

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	val := ctx.Value(requestIDKey)
	id, _ := val.(string)
	return id
}
