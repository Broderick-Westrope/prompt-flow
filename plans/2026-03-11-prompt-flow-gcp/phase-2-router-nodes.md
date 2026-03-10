# Phase 2: Node Type System + Router Nodes

> **Status:** COMPLETED

## Specification

**Problem:** The engine only supports LLM nodes. There's no way to express conditional branching in a flow — every node always executes. The chat app use case requires routing to different handlers based on a classifier's output (e.g., route "billing" tickets to a billing handler, "support" tickets to a support handler).

**Goal:** A `type` field on nodes distinguishes node behavior. A new `router` node type evaluates simple string expressions against its input and selectively executes only the matching branch. Unmatched branches are skipped entirely. Existing flows (which have no `type` field) continue to work unchanged — they default to `type: llm`.

**Design Decision — How Skipping Works:**
Router nodes participate in the dependency graph like any other node. A router's `routes[].next` targets are treated as dependency edges in the topological sort and cycle detection. When a router executes, it produces a single output: the `next` node ID of the matched route. Branch nodes declare a dependency on the router via a standard input (`from: "router_id.selected"`). The executor checks whether a node's router dependency selected it — if not, the node is marked as skipped and never executes. Descendants of skipped nodes are also skipped (their inputs are never satisfied).

This approach is compatible with parallel execution (Phase 3) because skipping is resolved *before* a node is scheduled, not after.

**Scope:**
- In: `type` field on Node struct, `router` node type with simple expression matching (==, !=, contains, startsWith, endsWith), `default` route, validator support, executor support, example flow, tests
- Out: Edge conditions, CEL expressions, web UI rendering of router nodes

**Success Criteria:**
- [ ] Existing flows with no `type` field still parse and execute correctly
- [ ] Router nodes evaluate expressions and only execute the matched branch
- [ ] Validator rejects router nodes with no routes, missing inputs, or invalid expressions
- [ ] Router `routes[].next` edges are included in cycle detection and topological sort
- [ ] An example flow demonstrates classification → routing → branch-specific handling
- [ ] All new and existing tests pass

## Context Loading

```bash
read pkg/flow/types.go
read pkg/flow/validator.go
read pkg/executor/executor.go
read examples/flows/support-ticket.flow.yaml
```

## Tasks

### Type System + Router Data Model

### Task 1: Add Type Field and Router Types

**Context:** `pkg/flow/types.go`, `pkg/flow/parser.go`

**Steps:**
1. [ ] Add `Type` field to `Node` struct: `Type string \`yaml:"type,omitempty" json:"type,omitempty"\`` — defaults to `"llm"` when empty
2. [ ] Add `Route` struct:
   ```go
   type Route struct {
       When    string `yaml:"when,omitempty" json:"when,omitempty"`     // expression like `== "billing"`
       Default bool   `yaml:"default,omitempty" json:"default,omitempty"`
       Next    string `yaml:"next" json:"next"`                         // target node ID
   }
   ```
3. [ ] Add `Routes` field to `Node` struct: `Routes []Route \`yaml:"routes,omitempty" json:"routes,omitempty"\``
4. [ ] Verify existing example flows still parse correctly with `go test ./pkg/flow/`

**Verify:** `go test ./pkg/flow/ -v`

---

### Expression Evaluator

### Task 2: Simple Expression Evaluator

**Context:** `pkg/flow/types.go` (Route struct from Task 1)

**Steps:**
1. [ ] Create `pkg/executor/expression.go` with:
   - `EvaluateRouteExpression(value string, expression string) (bool, error)`
   - Supported operators: `== "value"`, `!= "value"`, `contains "value"`, `startsWith "value"`, `endsWith "value"`
   - Expression parsing: extract operator (everything before first `"`), extract operand (content between quotes). Require quoted operands to avoid ambiguity with spaces.
   - Return error for unrecognized operators or missing/malformed quotes
2. [ ] Create `pkg/executor/expression_test.go` with tests for:
   - Each operator with matching and non-matching values
   - Case sensitivity (exact match by default)
   - Error case: unknown operator
   - Error case: missing quotes around operand
   - Error case: empty expression
   - Values containing spaces (e.g., `contains "high priority"`)
   - Edge cases: empty value, empty quoted operand

**Verify:** `go test ./pkg/executor/ -v`

---

### Validator Updates

### Task 3: Validate Router Nodes + Update Graph Traversals

**Context:** `pkg/flow/validator.go`, `pkg/flow/types.go`

**Steps:**
1. [ ] Update `validateNode` to dispatch based on `node.Type`:
   - `""` or `"llm"`: existing validation (prompt required, etc.)
   - `"router"`: new validation path
   - Unknown type: return error
2. [ ] Add `validateRouterNode` function:
   - Must have at least one route
   - Must have at least one input (the value to route on)
   - Each route must have a `next` target
   - Each non-default route must have a `when` expression
   - At most one `default` route
   - All `next` targets must reference existing node IDs
   - Prompt must NOT be set (router nodes don't call LLMs)
3. [ ] **Update `checkCycles`** to also walk `routes[].next` edges (not just `inputs[].from`):
   - When building the adjacency list, for router nodes, add edges from the router to each `routes[].next` target
4. [ ] **Update `validateReferences`** to also validate router `next` targets reference existing nodes
5. [ ] Add tests to `pkg/flow/validator_test.go`:
   - Valid router node passes
   - Router with no routes fails
   - Router with no inputs fails
   - Router with invalid next target fails
   - Router with multiple defaults fails
   - Router with prompt set fails
   - Mixed flow with LLM + router nodes passes
   - Cycle through router routes is detected

**Verify:** `go test ./pkg/flow/ -v`

---

### Executor Updates

### Task 4: Execute Router Nodes

**Context:** `pkg/executor/executor.go`, `pkg/executor/expression.go`

**Steps:**
1. [ ] **Update `topologicalSort`** to include router route edges:
   - When building the graph, for router nodes, add edges from the router to each `routes[].next` target (same as cycle detection)
   - This ensures branch nodes are always scheduled *after* their router
2. [ ] Update `executeNode` to dispatch based on `node.Type`:
   - `""` or `"llm"`: existing `executeLLMNode` path
   - `"router"`: new `executeRouterNode` path
3. [ ] Implement `executeRouterNode`:
   - Resolve the input value from `inputData` (first input)
   - Convert input value to string for expression evaluation
   - Iterate routes, evaluate each `when` expression against the input value
   - First matching route wins; if none match, use `default` route
   - If no route matches and no default: return error
   - Set the node's output: `outputs["selected"] = matchedRoute.Next`
4. [ ] **Update the execution loop** to handle router-based skipping:
   - Maintain a `skippedNodes` set (populated before scheduling, not after)
   - Before executing any node, check: does this node have an input `from` that references a router's `selected` output? If so, was this node the selected target?
   - More specifically: after a router executes, iterate all its routes. For each route that was NOT selected, add `route.Next` to `skippedNodes`.
   - Before executing a node: if it's in `skippedNodes`, mark it skipped and add all nodes that depend exclusively on it to `skippedNodes` too (transitive skip propagation).
   - A node that has inputs from both a skipped branch and a non-skipped branch is an error (ambiguous merge after conditional). The validator could catch this statically in a future phase, but for now a runtime error is acceptable.
5. [ ] Add `Skipped` field to `NodeResult`: `Skipped bool \`json:"skipped,omitempty"\``
6. [ ] Add tests to `pkg/executor/executor_test.go`:
   - Router selects correct branch based on expression match
   - Unmatched branches are skipped (not executed, no LLM call)
   - Default route is used when no `when` matches
   - Error when no route matches and no default
   - Transitive skipping: if A is skipped, B (which depends on A) is also skipped
   - Node with inputs from both skipped and non-skipped branches returns error

**Verify:** `go test ./pkg/executor/ -v`

---

### Example + CLI Compatibility

### Task 5: Example Flow and End-to-End Verification

**Context:** `examples/flows/`, `cmd/pfctl/test_cmd.go`, `cmd/pfctl/validate_cmd.go`

**Steps:**
1. [ ] Create `examples/flows/routed-support-ticket.flow.yaml`:
   ```yaml
   nodes:
     - id: classifier
       type: llm
       inputs:
         - name: ticket_text
           from: input
       prompt: |
         Classify this ticket into exactly one category: billing, technical, general.
         Output only the category name.
         Ticket: {{.ticket_text}}
       outputs:
         - name: category

     - id: route_department
       type: router
       inputs:
         - name: category
           from: classifier.category
       routes:
         - when: '== "billing"'
           next: handle_billing
         - when: '== "technical"'
           next: handle_technical
         - default: true
           next: handle_general

     - id: handle_billing
       type: llm
       inputs:
         - name: ticket_text
           from: input
       prompt: |
         Draft a billing department response for: {{.ticket_text}}
       outputs:
         - name: response
           to: output

     - id: handle_technical
       type: llm
       inputs:
         - name: ticket_text
           from: input
       prompt: |
         Draft a technical support response for: {{.ticket_text}}
       outputs:
         - name: response
           to: output

     - id: handle_general
       type: llm
       inputs:
         - name: ticket_text
           from: input
       prompt: |
         Draft a general support response for: {{.ticket_text}}
       outputs:
         - name: response
           to: output
   ```
   Note: Branch nodes get their input from flow `input` directly, not through the router. The router only controls which branch executes.
2. [ ] Verify `pfctl validate` works with the new flow
3. [ ] Run full test suite: `go test ./...`

**Verify:** `go test ./... && go build -o pfctl ./cmd/pfctl && ./pfctl validate examples/flows/routed-support-ticket.flow.yaml`
