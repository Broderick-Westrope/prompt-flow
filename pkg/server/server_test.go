package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthzEndpoint(t *testing.T) {
	srv := newTestServer()
	mux := srv.testMux(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	mux.ServeHTTP(rec, req)

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

func TestReadyzEndpoint(t *testing.T) {
	srv := newTestServer()
	mux := srv.testMux(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	mux.ServeHTTP(rec, req)

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

func TestRequestLoggingMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	srv := newTestServer()
	srv.logger = logger

	mux := srv.testMux(t)
	handler := srv.requestLoggingMiddleware(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	handler.ServeHTTP(rec, req)

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

	srv := newTestServer()
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

func newTestServer() *Server {
	return New(0, "", false, 5*time.Minute)
}

func (s *Server) testMux(t *testing.T) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/api/flow", s.handleGetFlow)
	mux.HandleFunc("/api/flow/validate", s.handleValidateFlow)
	mux.HandleFunc("/api/flow/execute", s.handleExecuteFlow)
	mux.HandleFunc("/api/providers", s.handleGetProviders)
	mux.HandleFunc("/api/config", s.handleGetConfig)
	return mux
}
