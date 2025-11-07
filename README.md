# Prompt Flow

A vendor-agnostic CLI tool for creating, testing, and visualizing prompt flows locally. Build reproducible, version-controlled generative AI workflows without being locked into cloud-based tools.

## What is a Prompt Flow?

A prompt flow is a structured workflow that orchestrates interactions between large language models, data sources, and business logic to accomplish multi-step generative AI tasks. Instead of a single, complex prompt, flows break down tasks into discrete nodes that form a directed acyclic graph (DAG), making them easier to test, debug, and maintain.

## Features

- **Vendor Agnostic**: Define flows in YAML or JSON, not tied to any cloud provider
- **Version Control**: Store flow definitions alongside your code in Git
- **Visual DAG Editor**: Web UI for visualizing and testing flows
- **Multi-Provider Support**: Built-in support for OpenAI and Anthropic, with extensible provider interface
- **Local Development**: Run and test flows entirely on your local machine
- **Cost Tracking**: Automatic token usage and cost estimation for each execution
- **Test Mode**: Execute flows with sample inputs and inspect node-by-node results

## Installation

```bash
# Clone the repository
git clone https://github.com/broderick/prompt-flow.git
cd prompt-flow

# Build the CLI
go build -o pfctl ./cmd/pfctl

# Optionally, install to your PATH
go install ./cmd/pfctl
```

## Quick Start

### 1. Initialize a new flow

```bash
pfctl init my-first-flow
```

This creates `my-first-flow.flow.yaml` with a basic structure:

```yaml
version: "1.0"
name: "my-first-flow"
description: "A sample prompt flow"
config:
  default_provider: "openai"
  default_model: "gpt-3.5-turbo"
nodes:
  - id: "process"
    type: "llm"
    inputs:
      - name: "user_input"
        from: "input"
    prompt: |
      You are a helpful assistant.
      User input: {{.user_input}}
      Please provide a helpful response.
    outputs:
      - name: "response"
        to: "output"
```

### 2. Validate your flow

```bash
pfctl validate my-first-flow.flow.yaml
```

### 3. Test your flow

Set up your API keys:

```bash
export OPENAI_API_KEY="your-key-here"
# or
export ANTHROPIC_API_KEY="your-key-here"
```

Run the flow:

```bash
pfctl test my-first-flow.flow.yaml -i user_input="Hello, what's the weather like?"
```

### 4. Launch the web UI

```bash
pfctl serve -f my-first-flow.flow.yaml
```

Open http://localhost:8080 to visualize and test your flow in the browser.

## Flow Definition Format

Flows are defined in YAML or JSON with the following structure:

### Top-Level Fields

- `version` (string): Flow definition version (currently "1.0")
- `name` (string): Flow name
- `description` (string): Optional description
- `config` (object): Flow-level configuration
  - `default_provider` (string): Default LLM provider ("openai", "anthropic")
  - `default_model` (string): Default model name
- `nodes` (array): List of nodes in the flow

### Node Structure

Each node represents a step in the flow:

```yaml
- id: "node_id"              # Unique identifier
  type: "llm"                # Node type (currently only "llm" supported)
  provider: "openai"         # Optional: override default provider
  model: "gpt-4"             # Optional: override default model
  inputs:                    # Input sources
    - name: "input_name"
      from: "input"          # "input" for flow input, or "node_id.output_name"
  prompt: |                  # Go template for the prompt
    Your prompt here with {{.input_name}} placeholders
  outputs:                   # Output definitions
    - name: "output_name"
      to: "output"           # "output" to expose as flow output (optional)
  settings:                  # Optional provider-specific settings
    temperature: 0.7
    max_tokens: 1000
```

### Data Flow

Nodes connect through inputs and outputs:

- **Flow inputs**: Use `from: "input"` to accept data when the flow is executed
- **Node outputs**: Reference as `from: "node_id.output_name"` in downstream nodes
- **Flow outputs**: Set `to: "output"` to expose node output as final result

### Example: Multi-Node Flow

```yaml
version: "1.0"
name: "support-ticket-classifier"
description: "Classifies and routes support tickets"

config:
  default_provider: "openai"
  default_model: "gpt-3.5-turbo"

nodes:
  # Node 1: Classify urgency
  - id: "classify_urgency"
    type: "llm"
    inputs:
      - name: "ticket_text"
        from: "input"
    prompt: |
      Classify this ticket's urgency as high, medium, or low:
      {{.ticket_text}}

      Output only: high, medium, or low
    outputs:
      - name: "urgency_level"

  # Node 2: Classify department
  - id: "classify_department"
    type: "llm"
    inputs:
      - name: "ticket_text"
        from: "input"
    prompt: |
      Classify this ticket by department: billing, engineering, or sales
      {{.ticket_text}}

      Output only the department name.
    outputs:
      - name: "department"

  # Node 3: Draft response (depends on both classifications)
  - id: "draft_response"
    type: "llm"
    inputs:
      - name: "ticket_text"
        from: "input"
      - name: "urgency"
        from: "classify_urgency.urgency_level"
      - name: "department"
        from: "classify_department.department"
    prompt: |
      Draft a response to this {{.urgency}} urgency {{.department}} ticket:
      {{.ticket_text}}
    outputs:
      - name: "response"
        to: "output"
```

## CLI Commands

### `pfctl init <name>`

Create a new flow definition.

**Options:**
- `-o, --output <path>`: Output file path (default: `<name>.flow.yaml`)
- `-f, --format <yaml|json>`: Output format (default: yaml)

**Example:**
```bash
pfctl init customer-support -o flows/support.yaml
```

### `pfctl validate <flow-file>`

Validate a flow definition for syntax errors, cycles, and invalid references.

**Example:**
```bash
pfctl validate my-flow.flow.yaml
```

### `pfctl test <flow-file>`

Execute a flow with test inputs.

**Options:**
- `-i, --input <key=value>`: Input values (can be repeated)
- `-t, --timeout <duration>`: Execution timeout (default: 5m)

**Example:**
```bash
pfctl test my-flow.flow.yaml \
  -i user_input="Explain quantum computing" \
  -t 2m
```

### `pfctl serve`

Start the web UI server for visual flow editing and testing.

**Options:**
- `-p, --port <port>`: Port to listen on (default: 8080)
- `-f, --flow <path>`: Specific flow file to load

**Example:**
```bash
pfctl serve -p 3000 -f examples/support-ticket.flow.yaml
```

### `pfctl version`

Display version information.

## Web UI

The web UI provides:

1. **Visual DAG**: See your flow as a graph with nodes and connections
2. **Node Inspector**: Click nodes to view prompts, inputs, and outputs
3. **Test Mode**: Execute flows with sample inputs
4. **Results View**: Inspect outputs, token usage, and costs for each node
5. **Performance Metrics**: Track execution time and estimated costs

## Extending with Custom Providers

To add support for a new LLM provider:

1. Implement the `Provider` interface in `pkg/providers/`:

```go
type Provider interface {
    Name() string
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}
```

2. Register your provider in the executor:

```go
registry := providers.NewRegistry()
registry.Register(providers.NewOpenAIProvider(""))
registry.Register(providers.NewAnthropicProvider(""))
registry.Register(NewMyCustomProvider("")) // Your provider
```

See `pkg/providers/openai.go` and `pkg/providers/anthropic.go` for examples.

## Project Structure

```
prompt-flow/
├── cmd/pfctl/              # CLI entrypoint and commands
├── pkg/
│   ├── flow/               # Flow types, parser, and validator
│   ├── executor/           # DAG execution engine
│   ├── providers/          # LLM provider implementations
│   ├── server/             # HTTP server for web UI
│   └── web/                # Embedded static assets
├── examples/               # Sample flow definitions
└── docs/                   # Additional documentation
```

## Examples

Check the `examples/` directory for:

- `simple.flow.yaml`: Basic single-node flow
- `support-ticket.flow.yaml`: Multi-node classification and response flow

## Environment Variables

- `OPENAI_API_KEY`: OpenAI API key
- `ANTHROPIC_API_KEY`: Anthropic API key

## Roadmap

- [ ] Support for custom node types (HTTP calls, database queries)
- [ ] Flow composition (call flows from other flows)
- [ ] Conditional branching and loops
- [ ] Live editing in web UI with auto-save
- [ ] Flow versioning and snapshots
- [ ] Deployment to serverless functions
- [ ] Integration testing framework
- [ ] Prompt variant comparison (A/B testing)

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

MIT License - see LICENSE file for details

## Acknowledgments

Inspired by Azure Prompt Flow and AWS Bedrock Flows, but designed to be open, local-first, and vendor-agnostic.
