package argocd

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// ---------------------------------------------------------------------------
// parseNDJSONLogs
// ---------------------------------------------------------------------------

func TestParseNDJSONLogs_MultipleValidEntries(t *testing.T) {
	ts1 := "2026-02-24T11:53:35Z"
	ts2 := "2026-02-24T11:53:36Z"
	ent1 := ApplicationLogEntry{Result: ApplicationLogResult{Content: "a", TimeStamp: &ts1, PodName: "p1"}}
	ent2 := ApplicationLogEntry{Result: ApplicationLogResult{Content: "b", TimeStamp: &ts2, PodName: "p2"}}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	_ = enc.Encode(ent1)
	_ = enc.Encode(ent2)

	entries, err := parseNDJSONLogs(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Result.Content != "a" {
		t.Errorf("entry[0] content = %q, want %q", entries[0].Result.Content, "a")
	}
	if entries[1].Result.Content != "b" {
		t.Errorf("entry[1] content = %q, want %q", entries[1].Result.Content, "b")
	}
}

func TestParseNDJSONLogs_EmptyInput(t *testing.T) {
	entries, err := parseNDJSONLogs(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestParseNDJSONLogs_LastEntryStopsDecoding(t *testing.T) {
	ts := "2026-02-24T11:53:35Z"
	regular := ApplicationLogEntry{Result: ApplicationLogResult{Content: "line1", TimeStamp: &ts, PodName: "p1"}}
	last := ApplicationLogEntry{Result: ApplicationLogResult{Content: "", Last: true, PodName: "p1"}}
	trailing := ApplicationLogEntry{Result: ApplicationLogResult{Content: "should-not-appear", TimeStamp: &ts, PodName: "p1"}}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	_ = enc.Encode(regular)
	_ = enc.Encode(last)
	_ = enc.Encode(trailing)

	entries, err := parseNDJSONLogs(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (Last stops decoding), got %d", len(entries))
	}
	if entries[0].Result.Content != "line1" {
		t.Errorf("entry content = %q, want %q", entries[0].Result.Content, "line1")
	}
}

func TestParseNDJSONLogs_NilTimestampNonLast_StreamError(t *testing.T) {
	errEntry := ApplicationLogEntry{Result: ApplicationLogResult{
		Content: "rpc error: code = NotFound",
	}}

	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(errEntry)

	_, err := parseNDJSONLogs(&buf)
	if err == nil {
		t.Fatal("expected error for nil TimeStamp on non-last entry")
	}
	if !strings.Contains(err.Error(), "argo log stream error") {
		t.Errorf("error = %q, want substring %q", err.Error(), "argo log stream error")
	}
	if !strings.Contains(err.Error(), "rpc error: code = NotFound") {
		t.Errorf("error should contain original content, got %q", err.Error())
	}
}

func TestParseNDJSONLogs_MalformedJSON(t *testing.T) {
	_, err := parseNDJSONLogs(strings.NewReader(`{"result": INVALID}`))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "failed to decode log entry") {
		t.Errorf("error = %q, want substring %q", err.Error(), "failed to decode log entry")
	}
}

func TestParseNDJSONLogs_ValidEntriesThenLast(t *testing.T) {
	ts := "2026-02-24T12:00:00Z"
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for i := 0; i < 5; i++ {
		_ = enc.Encode(ApplicationLogEntry{Result: ApplicationLogResult{
			Content: "line", TimeStamp: &ts, PodName: "pod",
		}})
	}
	_ = enc.Encode(ApplicationLogEntry{Result: ApplicationLogResult{Last: true}})

	entries, err := parseNDJSONLogs(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(entries))
	}
}

// ---------------------------------------------------------------------------
// buildLogsURL
// ---------------------------------------------------------------------------

func TestBuildLogsURL_UnconfiguredEnv(t *testing.T) {
	viper.Set("NONEXISTENT_ENV_ARGOCD_API", "")
	viper.Set("ARGOCD_API", "")
	defer func() {
		viper.Set("NONEXISTENT_ENV_ARGOCD_API", nil)
		viper.Set("ARGOCD_API", nil)
	}()

	_, err := buildLogsURL("nonexistent_env", "my-app", &ApplicationLogsOptions{})
	if err == nil {
		t.Fatal("expected error for unconfigured env")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("error = %q, want substring %q", err.Error(), "not configured")
	}
}

func TestBuildLogsURL_ZeroTailLinesFallsBackToDefault(t *testing.T) {
	viper.Set("TEST_ARGOCD_API", "https://argocd.example.com")
	defer viper.Set("TEST_ARGOCD_API", nil)

	u, err := buildLogsURL("test", "my-app", &ApplicationLogsOptions{Container: "main"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(u, "tailLines=1000") {
		t.Errorf("expected tailLines=1000 (zero value falls back to default) in URL, got %q", u)
	}
	if !strings.Contains(u, "container=main") {
		t.Errorf("expected container=main in URL, got %q", u)
	}
}

func TestBuildLogsURL_NegativeTailLinesFallsBackToDefault(t *testing.T) {
	viper.Set("TEST_ARGOCD_API", "https://argocd.example.com")
	defer viper.Set("TEST_ARGOCD_API", nil)

	u, err := buildLogsURL("test", "my-app", &ApplicationLogsOptions{Container: "main", TailLines: -1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(u, "tailLines=1000") {
		t.Errorf("expected default tailLines=1000 for negative value, got %q", u)
	}
}

func TestBuildLogsURL_CustomTailLines(t *testing.T) {
	viper.Set("TEST_ARGOCD_API", "https://argocd.example.com")
	defer viper.Set("TEST_ARGOCD_API", nil)

	u, err := buildLogsURL("test", "my-app", &ApplicationLogsOptions{TailLines: 50})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(u, "tailLines=50") {
		t.Errorf("expected tailLines=50 in URL, got %q", u)
	}
}

func TestBuildLogsURL_AllOptions(t *testing.T) {
	viper.Set("TEST_ARGOCD_API", "https://argocd.example.com")
	defer viper.Set("TEST_ARGOCD_API", nil)

	opts := &ApplicationLogsOptions{
		PodName:      "pod-abc",
		Container:    "sidecar",
		Previous:     true,
		SinceSeconds: 3600,
		TailLines:    200,
		Filter:       "ERROR",
	}
	u, err := buildLogsURL("test", "my-app", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := map[string]string{
		"container":    "container=sidecar",
		"previous":     "previous=true",
		"sinceSeconds": "sinceSeconds=3600",
		"tailLines":    "tailLines=200",
		"filter":       "filter=ERROR",
		"podName":      "podName=pod-abc",
	}
	for name, want := range checks {
		if !strings.Contains(u, want) {
			t.Errorf("expected %s (%q) in URL, got %q", name, want, u)
		}
	}
}

func TestBuildLogsURL_OptionalFieldsOmitted(t *testing.T) {
	viper.Set("TEST_ARGOCD_API", "https://argocd.example.com")
	defer viper.Set("TEST_ARGOCD_API", nil)

	u, err := buildLogsURL("test", "my-app", &ApplicationLogsOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(u, "previous=") {
		t.Errorf("previous should be omitted when false, got %q", u)
	}
	if strings.Contains(u, "sinceSeconds=") {
		t.Errorf("sinceSeconds should be omitted when 0, got %q", u)
	}
	if strings.Contains(u, "filter=") {
		t.Errorf("filter should be omitted when empty, got %q", u)
	}
	if strings.Contains(u, "podName=") {
		t.Errorf("podName should be omitted when empty, got %q", u)
	}
}

func TestBuildLogsURL_NameIsPathEscaped(t *testing.T) {
	viper.Set("TEST_ARGOCD_API", "https://argocd.example.com")
	defer viper.Set("TEST_ARGOCD_API", nil)

	u, err := buildLogsURL("test", "ns/app-name", &ApplicationLogsOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(u, "ns/app-name/logs") {
		t.Errorf("name with slash should be escaped, got %q", u)
	}
	if !strings.Contains(u, "ns%2Fapp-name") {
		t.Errorf("expected path-escaped name in URL, got %q", u)
	}
}

// ---------------------------------------------------------------------------
// GetApplicationLogs
// ---------------------------------------------------------------------------

func setViperForTest(t *testing.T, serverURL string) {
	t.Helper()
	oldAPI := viper.GetString("GCP_INT_ARGOCD_API")
	oldToken := viper.GetString("GCP_INT_ARGOCD_TOKEN")
	viper.Set("GCP_INT_ARGOCD_API", serverURL)
	viper.Set("GCP_INT_ARGOCD_TOKEN", "test-token")
	t.Cleanup(func() {
		viper.Set("GCP_INT_ARGOCD_API", oldAPI)
		viper.Set("GCP_INT_ARGOCD_TOKEN", oldToken)
	})
}

func TestGetApplicationLogs_EmptyName(t *testing.T) {
	_, err := GetApplicationLogs("", "gcp_int", &ApplicationLogsOptions{})
	if err == nil {
		t.Fatal("expected error when application name is empty")
	}
	if err.Error() != "application name is required" {
		t.Errorf("error = %q, want %q", err.Error(), "application name is required")
	}
}

func TestGetApplicationLogs_NilOpts(t *testing.T) {
	_, err := GetApplicationLogs("my-app", "gcp_int", nil)
	if err == nil {
		t.Fatal("expected error when opts is nil")
	}
	if err.Error() != "log options must not be nil" {
		t.Errorf("error = %q, want %q", err.Error(), "log options must not be nil")
	}
}

func TestGetApplicationLogs_EmptyContainer(t *testing.T) {
	_, err := GetApplicationLogs("my-app", "gcp_int", &ApplicationLogsOptions{})
	if err == nil {
		t.Fatal("expected error when container is empty")
	}
	if err.Error() != "container is required" {
		t.Errorf("error = %q, want %q", err.Error(), "container is required")
	}
}

func TestGetApplicationLogs_BuildURLFails(t *testing.T) {
	viper.Set("BADENV_ARGOCD_API", "")
	viper.Set("ARGOCD_API", "")
	defer func() {
		viper.Set("BADENV_ARGOCD_API", nil)
		viper.Set("ARGOCD_API", nil)
	}()

	_, err := GetApplicationLogs("my-app", "badenv", &ApplicationLogsOptions{Container: "main"})
	if err == nil {
		t.Fatal("expected error when buildLogsURL fails")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("error = %q, want substring %q", err.Error(), "not configured")
	}
}

func TestGetApplicationLogs_Success(t *testing.T) {
	ts := "2026-02-24T11:53:35Z"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/v1/applications/") || !strings.HasSuffix(r.URL.Path, "/logs") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		_ = enc.Encode(ApplicationLogEntry{Result: ApplicationLogResult{
			Content: "line1", TimeStamp: &ts, PodName: "pod-1",
			TimeStampStr: "2026-02-24T11:53:35.085112331Z",
		}})
		_ = enc.Encode(ApplicationLogEntry{Result: ApplicationLogResult{
			Content: "line2", TimeStamp: &ts, PodName: "pod-1",
			TimeStampStr: "2026-02-24T11:53:36.000000000Z",
		}})
		_ = enc.Encode(ApplicationLogEntry{Result: ApplicationLogResult{Last: true}})
	}))
	defer server.Close()
	setViperForTest(t, server.URL)

	entries, err := GetApplicationLogs("int-myapp", "gcp_int", &ApplicationLogsOptions{Container: "main"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Result.Content != "line1" {
		t.Errorf("entry[0] content = %q, want %q", entries[0].Result.Content, "line1")
	}
	if entries[1].Result.Content != "line2" {
		t.Errorf("entry[1] content = %q, want %q", entries[1].Result.Content, "line2")
	}
}

func TestGetApplicationLogs_SuccessEmptyLogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ApplicationLogEntry{Result: ApplicationLogResult{Last: true}})
	}))
	defer server.Close()
	setViperForTest(t, server.URL)

	entries, err := GetApplicationLogs("int-myapp", "gcp_int", &ApplicationLogsOptions{Container: "main"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries for empty log stream, got %d", len(entries))
	}
}

func TestGetApplicationLogs_QueryParamsPropagated(t *testing.T) {
	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ApplicationLogEntry{Result: ApplicationLogResult{Last: true}})
	}))
	defer server.Close()
	setViperForTest(t, server.URL)

	opts := &ApplicationLogsOptions{
		PodName:      "pod-xyz",
		Container:    "worker",
		Previous:     true,
		SinceSeconds: 600,
		TailLines:    100,
		Filter:       "WARN",
	}
	_, err := GetApplicationLogs("int-myapp", "gcp_int", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []string{
		"podName=pod-xyz",
		"container=worker",
		"previous=true",
		"sinceSeconds=600",
		"tailLines=100",
		"filter=WARN",
	}
	for _, want := range checks {
		if !strings.Contains(capturedQuery, want) {
			t.Errorf("query %q missing %q", capturedQuery, want)
		}
	}
}

func TestGetApplicationLogs_Non200_ValidErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"details":     []interface{}{},
				"grpc_code":   5,
				"http_code":   404,
				"http_status": "Not Found",
				"message":     "application not found",
			},
		})
	}))
	defer server.Close()
	setViperForTest(t, server.URL)

	_, err := GetApplicationLogs("int-nonexistent", "gcp_int", &ApplicationLogsOptions{Container: "main"})
	if err == nil {
		t.Fatal("expected error on 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should contain status code, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "application not found") {
		t.Errorf("error should contain message, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "Not Found") {
		t.Errorf("error should contain HTTP status text, got %q", err.Error())
	}
	var apiErr *ArgoCDAPIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected error to be *ArgoCDAPIError")
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected StatusCode 404, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "application not found" {
		t.Errorf("expected Message %q, got %q", "application not found", apiErr.Message)
	}
}

func TestGetApplicationLogs_Non200_InvalidErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()
	setViperForTest(t, server.URL)

	_, err := GetApplicationLogs("int-myapp", "gcp_int", &ApplicationLogsOptions{Container: "main"})
	if err == nil {
		t.Fatal("expected error on 500 with invalid body")
	}
	if !strings.Contains(err.Error(), "failed to decode error response") {
		t.Errorf("error = %q, want substring about decode failure", err.Error())
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should contain status code 500, got %q", err.Error())
	}
	var apiErr *ArgoCDAPIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected error to be *ArgoCDAPIError even for invalid body")
	}
	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected StatusCode 500, got %d", apiErr.StatusCode)
	}
}

func TestGetApplicationLogs_Non200_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"grpc_code":   7,
				"http_code":   403,
				"http_status": "Forbidden",
				"message":     "permission denied",
			},
		})
	}))
	defer server.Close()
	setViperForTest(t, server.URL)

	_, err := GetApplicationLogs("int-myapp", "gcp_int", &ApplicationLogsOptions{Container: "main"})
	if err == nil {
		t.Fatal("expected error on 403 response")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should contain 403, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error should contain message, got %q", err.Error())
	}
	var apiErr *ArgoCDAPIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected error to be *ArgoCDAPIError")
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("expected StatusCode 403, got %d", apiErr.StatusCode)
	}
}

func TestGetApplicationLogs_StreamErrorInBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ApplicationLogEntry{Result: ApplicationLogResult{
			Content: "rpc error: code = NotFound desc = app not found",
		}})
	}))
	defer server.Close()
	setViperForTest(t, server.URL)

	_, err := GetApplicationLogs("int-myapp", "gcp_int", &ApplicationLogsOptions{Container: "main"})
	if err == nil {
		t.Fatal("expected error for stream error in body")
	}
	if !strings.Contains(err.Error(), "argo log stream error") {
		t.Errorf("error = %q, want substring %q", err.Error(), "argo log stream error")
	}
}

func TestGetApplicationLogs_MalformedResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result": BROKEN}`))
	}))
	defer server.Close()
	setViperForTest(t, server.URL)

	_, err := GetApplicationLogs("int-myapp", "gcp_int", &ApplicationLogsOptions{Container: "main"})
	if err == nil {
		t.Fatal("expected error for malformed response body")
	}
	if !strings.Contains(err.Error(), "failed to decode log entry") {
		t.Errorf("error = %q, want substring about decode failure", err.Error())
	}
}

func TestGetApplicationLogs_ClientRequestFails(t *testing.T) {
	viper.Set("GCP_INT_ARGOCD_API", "http://127.0.0.1:1")
	viper.Set("GCP_INT_ARGOCD_TOKEN", "test-token")
	defer func() {
		viper.Set("GCP_INT_ARGOCD_API", nil)
		viper.Set("GCP_INT_ARGOCD_TOKEN", nil)
	}()

	_, err := GetApplicationLogs("int-myapp", "gcp_int", &ApplicationLogsOptions{Container: "main"})
	if err == nil {
		t.Fatal("expected error when HTTP client cannot connect")
	}
	if !strings.Contains(err.Error(), "failed to get logs from Argo CD") {
		t.Errorf("error = %q, want substring about request failure", err.Error())
	}
}
