# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Prompt Flow is a vendor-agnostic CLI tool (`pfctl`) for creating, testing, and visualizing prompt flows locally. It enables building reproducible, version-controlled generative AI workflows using YAML/JSON flow definitions that orchestrate LLM interactions in a directed acyclic graph (DAG).

## Build and Development Commands

### Building

```bash
# Build the CLI binary
go build -o pfctl ./cmd/pfctl

# Install to $GOPATH/bin
go install ./cmd/pfctl

# Build web UI (required after changing web source files)
cd web && npm run build
# This outputs to pkg/server/static/dist/
# Then rebuild Go binary to embed new static files
cd .. && go build -o pfctl ./cmd/pfctl

# Build more easily with one command
task build.cli
```

**IMPORTANT - Web UI Source Files:**

- Web UI source: `web/src/` (TypeScript/React with Vite)
- Build output: `pkg/server/static/dist/` (embedded via `//go:embed`)
- `pkg/server/static/app.js` is OLD and NOT the source - DO NOT EDIT
- After changing web UI: rebuild web (`cd web && npm run build`) then rebuild Go binary

### Testing

```bash
# Run all tests (currently no test files exist)
go test ./...

# Run tests with verbose output
go test -v ./...
```

### Running the CLI

```bash
# After building, the binary is available as ./pfctl
./pfctl --help

# Common commands
./pfctl init my-flow
./pfctl validate examples/simple.flow.yaml
./pfctl test examples/simple.flow.yaml -i user_input="test input"
./pfctl serve -p 8080 -f examples/simple.flow.yaml
```

### Environment Setup

The application requires API keys for LLM providers:

```bash
export OPENAI_API_KEY="your-key-here"
export ANTHROPIC_API_KEY="your-key-here"
export GITHUB_PLAYGROUND_PAT="your-token-here"  # Optional: for GitHub playground
```

## Architecture

### Core Components

**Flow Definition & Validation** (`pkg/flow/`)

- `types.go`: Core data structures (Flow, Node, Input, Output, ExecutionResult)
- `parser.go`: YAML/JSON parsing for flow definitions
- `validator.go`: Flow validation including cycle detection, reference checking, and schema validation

**Execution Engine** (`pkg/executor/`)

- `executor.go`: DAG execution engine that:
  - Performs topological sort using Kahn's algorithm to determine node execution order
  - Executes nodes sequentially based on dependencies
  - Manages data flow between nodes using a `nodeOutputs` map
  - Renders prompt templates using Go's `text/template` package
  - Tracks metrics (tokens, costs, timing) for each node

**Provider System** (`pkg/providers/`)

- `provider.go`: Provider interface and Registry for managing LLM providers
- `openai.go`: OpenAI API integration with cost estimation
- `anthropic.go`: Anthropic API integration with cost estimation
- `github_playground_openai.go`: GitHub Models integration
- All providers implement the `Provider` interface with `Name()` and `Complete()` methods

**Web Server** (`pkg/server/`)

- `server.go`: HTTP server exposing REST API for:
  - `/api/flow`: Load and return flow definitions
  - `/api/flow/validate`: Validate flow syntax and structure
  - `/api/flow/execute`: Execute flows with inputs and return results
  - `/api/providers`: List available LLM providers
- `static/`: Embedded web UI assets for visual DAG editing and testing

**CLI Commands** (`cmd/pfctl/`)

- `main.go`: CLI entry point using kong for command parsing
- `init_cmd.go`: Create new flow definitions
- `validate_cmd.go`: Validate flow files
- `test_cmd.go`: Execute flows with inputs
- `serve_cmd.go`: Start web UI server
- `version_cmd.go`: Display version info

### Flow Execution Model

1. **Validation**: Flow is validated for cycles, missing references, and schema errors
2. **Topological Sort**: Nodes are ordered based on dependencies (e.g., if node B depends on node A's output, A executes first)
3. **Sequential Execution**: Nodes execute in dependency order (not in parallel)
4. **Data Flow**:
   - Flow inputs come from CLI flags (`-i key=value`)
   - Node outputs are stored in `nodeOutputs[nodeID][outputName]`
   - Nodes reference other nodes' outputs via `from: "node_id.output_name"`
   - Nodes can export outputs to flow results via `to: "output"`
5. **Template Rendering**: Prompts use Go templates with input data (e.g., `{{.user_input}}`)
6. **Provider Abstraction**: Nodes can override flow-level provider/model settings

### Key Design Patterns

- **Registry Pattern**: Providers are registered and retrieved by name from a central Registry
- **Strategy Pattern**: Provider interface allows swapping LLM backends without changing executor code
- **Template Method**: Executor defines the execution flow, providers implement completion logic
- **Embedded Resources**: Web UI static files are embedded using `//go:embed` for single-binary distribution

## Flow Definition Structure

Flows are YAML/JSON files with:

- `version`: Currently "1.0"
- `name`, `description`: Metadata
- `config`: Default provider and model settings
- `nodes`: Array of processing steps, each with:
  - `id`: Unique identifier
  - `type`: Currently only "llm" supported
  - `provider`, `model`: Optional overrides for this node
  - `inputs`: Array of `{name, from}` where `from` is either "input" or "node_id.output_name"
  - `prompt`: Go template string for the LLM prompt
  - `outputs`: Array of `{name, to}` where `to` can be "output" to expose as flow output
  - `settings`: Optional map for provider-specific settings (temperature, max_tokens, etc.)

## Adding New Providers

1. Create a new file in `pkg/providers/` (e.g., `custom.go`)
2. Implement the `Provider` interface:
   ```go
   type Provider interface {
       Name() string
       Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
   }
   ```
3. Handle the `CompletionRequest.Settings` map for provider-specific options
4. Calculate token usage and costs in the response
5. Register the provider in `pkg/server/server.go` (New function) and in CLI commands that need it

## Known Limitations

- No test coverage exists yet
- Only "llm" node type is implemented (no HTTP, database, or custom nodes)
- Node execution is sequential, not parallel
- Web UI serves embedded static files but doesn't support live editing with auto-save
- Cost estimation is approximate and doesn't account for cached tokens or batch discounts
