package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Meesho/BharatMLStack/resource-manager/internal/adapters/etcd"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/adapters/kubernetes"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/adapters/redisq"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/application"
	"github.com/Meesho/BharatMLStack/resource-manager/internal/data/models"
	rmtypes "github.com/Meesho/BharatMLStack/resource-manager/internal/types"
)

func TestListShadowDeployables(t *testing.T) {
	store := etcd.NewMemoryShadowStateStore(map[string][]models.ShadowDeployable{
		"int": {
			{
				Name:          "int-predator-g2-std-16",
				NodePool:      "g2-std-16",
				DNS:           "predator-g2-std-16.meesho.int",
				State:         rmtypes.ShadowStateFree,
				MinPodCount:   1,
				LastUpdatedAt: time.Now(),
				Version:       1,
			},
		},
	})
	h := NewHandler(
		application.NewShadowService(store),
		application.NewOperationService(redisq.NewInMemoryPublisher(), kubernetes.NewMockExecutor(), nil),
		etcd.NewMemoryIdempotencyKeyStore(),
	)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/1.0/int/shadow-deployables", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "int-predator-g2-std-16") {
		t.Fatalf("expected deployable in response")
	}
}

func TestCreateDeployableRequiresIdempotencyKey(t *testing.T) {
	store := etcd.NewMemoryShadowStateStore(nil)
	h := NewHandler(
		application.NewShadowService(store),
		application.NewOperationService(redisq.NewInMemoryPublisher(), kubernetes.NewMockExecutor(), nil),
		etcd.NewMemoryIdempotencyKeyStore(),
	)
	mux := http.NewServeMux()
	h.Register(mux)

	payload := `{"name":"d1","workflow_run_id":"run-1","callback":{"url":"https://example.com/cb","method":"POST"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/1.0/int/deployables", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAllContractEndpointsAreWired(t *testing.T) {
	store := etcd.NewMemoryShadowStateStore(map[string][]models.ShadowDeployable{
		"int": {
			{
				Name:          "int-predator-g2-std-16",
				NodePool:      "g2-std-16",
				DNS:           "predator-g2-std-16.meesho.int",
				State:         rmtypes.ShadowStateFree,
				MinPodCount:   1,
				LastUpdatedAt: time.Now(),
				Version:       1,
			},
		},
	})
	h := NewHandler(
		application.NewShadowService(store),
		application.NewOperationService(redisq.NewInMemoryPublisher(), kubernetes.NewMockExecutor(), nil),
		etcd.NewMemoryIdempotencyKeyStore(),
	)
	mux := http.NewServeMux()
	h.Register(mux)

	testCases := []struct {
		name       string
		method     string
		path       string
		idempotKey string
		body       string
		wantCode   int
	}{
		{
			name:     "shadow list typo route alias",
			method:   http.MethodGet,
			path:     "/api/1.0/int/shadow-deploybles",
			body:     "",
			wantCode: http.StatusOK,
		},
		{
			name:       "procure shadow deployable",
			method:     http.MethodPost,
			path:       "/api/1.0/int/shadow-deployables",
			idempotKey: "k1",
			body:       `{"action":"PROCURE","name":"int-predator-g2-std-16","workflow_run_id":"run1","workflow_plan":"plan1"}`,
			wantCode:   http.StatusOK,
		},
		{
			name:       "min pod count",
			method:     http.MethodPost,
			path:       "/api/1.0/int/shadow-deployables/min-pod-count",
			idempotKey: "k2",
			body:       `{"action":"INCREASE","count":1,"name":"int-predator-g2-std-16","workflow_run_id":"run1","workflow_task_id":"t1","workflow_plan":"plan1"}`,
			wantCode:   http.StatusOK,
		},
		{
			name:       "create deployable",
			method:     http.MethodPost,
			path:       "/api/1.0/int/deployables",
			idempotKey: "k3",
			body:       `{"name":"int-predator-g2-std-16","image":"registry/image:v1","node_selector":"g2-standard-8","min_pod_count":0,"max_pod_count":1,"resources":{"cpu":{"request":"1","limit":"2"},"memory":{"request":"1Gi","limit":"2Gi"},"gpu":{"request":"1","limit":"1","memory":{"request":"8Gi","limit":"8Gi"}}},"workflow_run_id":"run1","workflow_plan":"plan1","workflow_task_id":"t1","callback":{"url":"https://example.com/callback","method":"POST","headers":{}}}`,
			wantCode:   http.StatusAccepted,
		},
		{
			name:       "load model",
			method:     http.MethodPost,
			path:       "/api/1.0/int/models/load",
			idempotKey: "k4",
			body:       `{"deployable_name":"int-predator-g2-std-16","model":{"name":"predator","version":"v2","artifact_location":"gs://bucket/model/v2"},"workflow_run_id":"run1","workflow_plan":"plan1","workflow_task_id":"t1","callback":{"url":"https://example.com/callback","method":"POST","headers":{}}}`,
			wantCode:   http.StatusAccepted,
		},
		{
			name:       "trigger job",
			method:     http.MethodPost,
			path:       "/api/1.0/int/jobs",
			idempotKey: "k5",
			body:       `{"job_name":"model-warmup-job","container":{"image":"registry/internal/ml-job-runner:latest"},"payload":{"model_name":"xyz","model_version":"v2"},"workflow_run_id":"run1","workflow_plan":"plan1","workflow_task_id":"t1","callback":{"url":"https://example.com/callback","method":"POST","headers":{}}}`,
			wantCode:   http.StatusAccepted,
		},
		{
			name:       "restart deployable",
			method:     http.MethodPost,
			path:       "/api/1.0/int/deployables/restart",
			idempotKey: "k6",
			body:       `{"namespace":"predator-g2-std-16","workflow_run_id":"run1","workflow_plan":"plan1","workflow_task_id":"t1","callback":{"url":"https://example.com/callback","method":"POST","headers":{}}}`,
			wantCode:   http.StatusAccepted,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			if tc.idempotKey != "" {
				req.Header.Set("X-Idempotency-Key", tc.idempotKey)
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			if w.Code != tc.wantCode {
				t.Fatalf("expected %d, got %d body=%s", tc.wantCode, w.Code, w.Body.String())
			}
		})
	}
}
