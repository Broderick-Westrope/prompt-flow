package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/broderick/prompt-flow/pkg/executor"
	"github.com/broderick/prompt-flow/pkg/flow"
	"github.com/broderick/prompt-flow/pkg/providers"
)

//go:embed static/dist/*
var staticFiles embed.FS

// Server represents the web server
type Server struct {
	port             int
	flowPath         string
	showStartEndNode bool
	registry         *providers.Registry
	executor         *executor.Executor
}

// New creates a new server instance
func New(port int, flowPath string, showStartEndNode bool) *Server {
	registry := providers.NewRegistry()
	registry.Register(providers.NewOpenAIProvider(os.Getenv("OPENAI_API_KEY")))
	registry.Register(providers.NewAnthropicProvider(os.Getenv("ANTHROPIC_API_KEY")))
	registry.Register(providers.NewGithubPlaygroundOpenAIProvider(os.Getenv("GITHUB_PLAYGROUND_PAT")))

	return &Server{
		port:             port,
		flowPath:         flowPath,
		showStartEndNode: showStartEndNode,
		registry:         registry,
		executor:         executor.New(registry),
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

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

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return server.ListenAndServe()
}

// handleGetFlow returns the flow definition
func (s *Server) handleGetFlow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var f *flow.Flow
	var err error

	if r.Method == http.MethodPost {
		// Parse flow from request body
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
		// Load flow from file
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

// handleValidateFlow validates a flow definition
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

// handleExecuteFlow executes a flow with provided inputs
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := s.executor.Execute(ctx, f, req.Inputs)
	if err != nil {
		// Still return the result even if there's an error
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // Use 200 since we're returning structured error info
		json.NewEncoder(w).Encode(result)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleGetProviders returns available providers
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

// handleGetConfig returns server configuration
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
