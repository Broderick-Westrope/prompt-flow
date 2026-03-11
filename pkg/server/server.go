package server

import (
	"context"
	"embed"
	"encoding/json"
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

type Server struct {
	port             int
	flowPath         string
	showStartEndNode bool
	executionTimeout time.Duration
	registry         *providers.Registry
	executor         *executor.Executor
	logger           *slog.Logger
}

func New(port int, flowPath string, showStartEndNode bool, executionTimeout time.Duration) *Server {
	registry := providers.NewRegistry().WithDefaultProviders()

	var handler slog.Handler
	if os.Getenv("LOG_FORMAT") == "json" {
		handler = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		handler = slog.NewTextHandler(os.Stdout, nil)
	}
	logger := slog.New(handler)

	return &Server{
		port:             port,
		flowPath:         flowPath,
		showStartEndNode: showStartEndNode,
		executionTimeout: executionTimeout,
		registry:         registry,
		executor:         executor.New(registry),
		logger:           logger,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)

	// API endpoints
	mux.HandleFunc("/api/flow", s.handleGetFlow)
	mux.HandleFunc("/api/flow/validate", s.handleValidateFlow)
	mux.HandleFunc("/api/flow/execute", s.handleExecuteFlow)
	mux.HandleFunc("/api/providers", s.handleGetProviders)
	mux.HandleFunc("/api/config", s.handleGetConfig)

	// Serve static files
	staticFS, err := fs.Sub(staticFiles, "static/dist")
	if err != nil {
		return fmt.Errorf("failed to get static file system: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

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
		s.logger.Info("server starting", "port", s.port, "flow", s.flowPath)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
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

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleGetFlow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var f *flow.Flow
	var err error

	if r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read request: %v", err), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		f, err = flow.ParseBytes(body, "flow.yaml")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse flow: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		if s.flowPath == "" {
			http.Error(w, "No flow file specified", http.StatusBadRequest)
			return
		}

		f, err = flow.Parse(s.flowPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load flow: %v", err), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(f)
}

func (s *Server) handleValidateFlow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	f, err := flow.ParseBytes(body, "flow.yaml")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"valid": false,
			"error": fmt.Sprintf("Parse error: %v", err),
		})
		return
	}

	err = flow.Validate(f)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"valid": false,
			"error": fmt.Sprintf("Validation error: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"valid": true,
		"nodes": len(f.Nodes),
	})
}

func (s *Server) handleExecuteFlow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Flow   json.RawMessage `json:"flow"`
		Inputs map[string]any  `json:"inputs"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse request: %v", err), http.StatusBadRequest)
		return
	}

	f, err := flow.ParseBytes(req.Flow, "flow.yaml")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse flow: %v", err), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.executionTimeout)
	defer cancel()

	result, err := s.executor.Execute(ctx, f, req.Inputs)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleGetProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providers := s.registry.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"providers": providers,
	})
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"showStartEndNode": s.showStartEndNode,
	})
}
