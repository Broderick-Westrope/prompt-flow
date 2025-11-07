export interface FlowConfig {
  default_provider?: string;
  default_model?: string;
}

export interface NodeInput {
  name: string;
  from: string;
}

export interface NodeOutput {
  name: string;
  to?: string;
}

export interface NodeSettings {
  temperature?: number;
  max_tokens?: number;
  [key: string]: unknown;
}

export interface FlowNode {
  id: string;
  type: string;
  provider?: string;
  model?: string;
  inputs: NodeInput[];
  outputs: NodeOutput[];
  prompt?: string;
  settings?: NodeSettings;
}

export interface Flow {
  version: string;
  name: string;
  description: string;
  config?: FlowConfig;
  nodes: FlowNode[];
}

export interface NodeMetrics {
  tokens_used?: number;
  estimated_cost?: number;
  duration?: number;
}

export interface NodeResult {
  node_id: string;
  outputs?: Record<string, unknown>;
  metrics?: NodeMetrics;
  error?: string;
}

export interface ExecutionResult {
  success: boolean;
  error?: string;
  duration: number;
  node_results?: NodeResult[];
  outputs?: Record<string, unknown>;
}

export interface ExecuteFlowRequest {
  flow: Flow;
  inputs: Record<string, unknown>;
}

export interface ValidateFlowResponse {
  valid: boolean;
  error?: string;
  nodes?: number;
}

export interface ProvidersResponse {
  providers: string[];
}
