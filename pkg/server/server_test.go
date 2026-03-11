package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func newTestServer(t *testing.T, mode string) *Server {
	t.Helper()
	srv, err := New(Config{
		Port:             0,
		FlowPath:         testdataPath("simple.flow.yaml"),
		Mode:             mode,
		ExecutionTimeout: 5 * time.Minute,
		Logger:           slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	})
	if err != nil {
		t.Fatalf("newTestServer: %v", err)
	}
	return srv
}

func TestHealthzEndpoint(t *testing.T) {
	srv := newTestServer(t, "dev")
	handler := srv.testHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
}

func TestReadyzWithLoadedFlow(t *testing.T) {
	srv := newTestServer(t, "dev")
	handler := srv.testHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
}

func TestReadyzWithoutFlow(t *testing.T) {
	srv := &Server{
		flow:   nil,
		logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
		mode:   "dev",
	}
	handler := srv.testHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "unavailable" {
		t.Fatalf("expected status unavailable, got %q", body["status"])
	}
}

func TestDevModeRegistersAllEndpoints(t *testing.T) {
	srv := newTestServer(t, "dev")
	handler := srv.testHandler(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/flow"},
		{http.MethodPost, "/api/flow/validate"},
		{http.MethodGet, "/api/providers"},
		{http.MethodGet, "/api/config"},
	}

	for _, ep := range endpoints {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(ep.method, ep.path, nil)
		handler.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound {
			t.Errorf("%s %s returned 404 in dev mode; expected it to be registered", ep.method, ep.path)
		}
	}
}

func TestProdModeReturns404ForDevEndpoints(t *testing.T) {
	srv := newTestServer(t, "prod")
	handler := srv.testHandler(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/flow"},
		{http.MethodPost, "/api/flow/validate"},
		{http.MethodGet, "/api/providers"},
		{http.MethodGet, "/api/config"},
	}

	for _, ep := range endpoints {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(ep.method, ep.path, nil)
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("%s %s returned %d in prod mode; expected 404", ep.method, ep.path, rec.Code)
		}
	}
}

func TestProdModeExecuteWorks(t *testing.T) {
	srv := newTestServer(t, "prod")
	handler := srv.testHandler(t)

	body := `{"inputs": {"user_input": "hello"}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/flow/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(rec, req)

	// The endpoint should be reachable (not 404). The actual LLM call may fail,
	// but we only care that routing works.
	if rec.Code == http.StatusNotFound {
		t.Fatalf("POST /api/flow/execute returned 404 in prod mode; expected it to be registered")
	}
}

func TestExecuteRejects400WithFlowField(t *testing.T) {
	srv := newTestServer(t, "dev")
	handler := srv.testHandler(t)

	body := `{"flow": {"name": "test"}, "inputs": {"user_input": "hello"}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/flow/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	respBody := rec.Body.String()
	if !strings.Contains(respBody, "flow") || !strings.Contains(respBody, "no longer accepted") {
		t.Fatalf("expected error message about 'flow' field no longer accepted, got: %s", respBody)
	}
}

func TestGetFlowReturnsPreloadedFlow(t *testing.T) {
	srv := newTestServer(t, "dev")
	handler := srv.testHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/flow", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	name, ok := body["name"].(string)
	if !ok || name != "test-flow" {
		t.Fatalf("expected flow name 'test-flow', got %v", body["name"])
	}
}

func TestRequestLoggingMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	srv := newTestServer(t, "dev")
	srv.logger = logger

	handler := srv.testHandler(t)
	loggingHandler := srv.requestLoggingMiddleware(handler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	loggingHandler.ServeHTTP(rec, req)

	logOutput := buf.String()
	for _, want := range []string{"method=GET", "path=/healthz", "status=200"} {
		if !bytes.Contains([]byte(logOutput), []byte(want)) {
			t.Errorf("log output missing %q, got: %s", want, logOutput)
		}
	}
}

func TestRequestLoggingMiddlewareRecordsStatusCode(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	srv := newTestServer(t, "dev")
	srv.logger = logger

	notFound := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	handler := srv.requestLoggingMiddleware(notFound)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	handler.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !bytes.Contains([]byte(logOutput), []byte("status=404")) {
		t.Errorf("log output missing status=404, got: %s", logOutput)
	}
}
