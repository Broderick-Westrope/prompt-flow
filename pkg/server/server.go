package server

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/broderick/prompt-flow/pkg/executor"
	"github.com/broderick/prompt-flow/pkg/flow"
	"github.com/broderick/prompt-flow/pkg/providers"
)

//go:embed static/dist/*
var staticFiles embed.FS

const maxRequestBodySize = 1 << 20 // 1MB

const (
	ModeDev  = "dev"
	ModeProd = "prod"
)

type Config struct {
	Port             int
	FlowPath         string
	ShowStartEndNode bool
	ExecutionTimeout time.Duration
	Logger           *slog.Logger
	Mode             string // ModeDev or ModeProd; defaults to ModeDev
}

type Server struct {
	port             int
	flowPath         string
	showStartEndNode bool
	executionTimeout time.Duration
	registry         *providers.Registry
	executor         *executor.Executor
	logger           *slog.Logger
	mode             string
	flow             *flow.Flow
}

func New(cfg Config) (*Server, error) {
	if cfg.FlowPath == "" {
		return nil, fmt.Errorf("flow path is required")
	}

	mode := cfg.Mode
	if mode == "" {
		mode = ModeDev
	}
	if mode != ModeDev && mode != ModeProd {
		return nil, fmt.Errorf("invalid mode %q: must be %q or %q", mode, ModeDev, ModeProd)
	}

	f, err := flow.Parse(cfg.FlowPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse flow file: %w", err)
	}

	registry := providers.NewRegistry().WithDefaultProviders()

	executionTimeout := cfg.ExecutionTimeout
	if executionTimeout == 0 {
		executionTimeout = 5 * time.Minute
	}

	logger := cfg.Logger
	if logger == nil {
		var handler slog.Handler
		if os.Getenv("LOG_FORMAT") == "json" {
			handler = slog.NewJSONHandler(os.Stdout, nil)
		} else {
			handler = slog.NewTextHandler(os.Stdout, nil)
		}
		logger = slog.New(handler)
	}

	return &Server{
		port:             cfg.Port,
		flowPath:         cfg.FlowPath,
		showStartEndNode: cfg.ShowStartEndNode,
		executionTimeout: executionTimeout,
		registry:         registry,
		executor:         executor.New(registry),
		logger:           logger,
		mode:             mode,
		flow:             f,
	}, nil
}

func (s *Server) registerRoutes(mux *http.ServeMux) error {
	// Health check endpoints (always registered)
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /readyz", s.handleReadyz)

	// Execution endpoint (always registered)
	mux.HandleFunc("POST /api/flow/execute", s.handleExecuteFlow)

	// Dev-only endpoints
	if s.mode == ModeDev {
		mux.HandleFunc("GET /api/flow", s.handleGetFlow)
		mux.HandleFunc("POST /api/flow/validate", s.handleValidateFlow)
		mux.HandleFunc("GET /api/providers", s.handleGetProviders)
		mux.HandleFunc("GET /api/config", s.handleGetConfig)

		// Serve static files
		staticFS, err := fs.Sub(staticFiles, "static/dist")
		if err != nil {
			return fmt.Errorf("failed to get static file system: %w", err)
		}
		mux.Handle("/", http.FileServer(http.FS(staticFS)))
	}

	return nil
}

func (s *Server) testHandler(t interface{ Helper(); Fatal(args ...any) }) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	if err := s.registerRoutes(mux); err != nil {
		t.Fatal(err)
	}
	return mux
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	if err := s.registerRoutes(mux); err != nil {
		return err
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      s.requestLoggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: s.executionTimeout + 30*time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("server starting", "port", s.port, "flow", s.flowPath, "mode", s.mode)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		s.logger.Info("shutdown signal received, draining connections")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
		s.logger.Info("server stopped")
		return nil
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (s *Server) requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)
		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.statusCode,
			"duration", time.Since(start),
		)
	})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.logger.Error("failed to encode JSON response", "error", err)
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	if s.flow == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetFlow(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, s.flow)
}

func (s *Server) handleValidateFlow(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request: %v", err), http.StatusBadRequest)
		return
	}

	f, err := flow.ParseBytes(body, "flow.yaml")
	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"valid": false,
			"error": fmt.Sprintf("Parse error: %v", err),
		})
		return
	}

	err = flow.Validate(f)
	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"valid": false,
			"error": fmt.Sprintf("Validation error: %v", err),
		})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"valid": true,
		"nodes": len(f.Nodes),
	})
}

func (s *Server) handleExecuteFlow(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse request: %v", err), http.StatusBadRequest)
		return
	}

	if _, hasFlow := raw["flow"]; hasFlow {
		http.Error(w, `The 'flow' field is no longer accepted; the server executes its loaded flow file. Send only {"inputs": {...}}`, http.StatusBadRequest)
		return
	}

	var inputs map[string]any
	if rawInputs, ok := raw["inputs"]; ok {
		if err := json.Unmarshal(rawInputs, &inputs); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse inputs: %v", err), http.StatusBadRequest)
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.executionTimeout)
	defer cancel()

	result, err := s.executor.Execute(ctx, s.flow, inputs)
	if err != nil {
		s.writeJSON(w, http.StatusOK, result)
		return
	}

	s.writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleGetProviders(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{
		"providers": s.registry.List(),
	})
}

func (s *Server) handleGetConfig(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{
		"showStartEndNode": s.showStartEndNode,
	})
}
