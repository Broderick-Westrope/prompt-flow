import type {
  Flow,
  ExecutionResult,
  ExecuteFlowRequest,
  ValidateFlowResponse,
  ProvidersResponse,
} from '../types/flow';

const API_BASE = '/api';

class ApiError extends Error {
  status?: number;

  constructor(message: string, status?: number) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const text = await response.text();
    throw new ApiError(text || response.statusText, response.status);
  }
  return response.json();
}

export const api = {
  async getFlow(): Promise<Flow> {
    const response = await fetch(`${API_BASE}/flow`);
    return handleResponse<Flow>(response);
  },

  async validateFlow(flowData: string): Promise<ValidateFlowResponse> {
    const response = await fetch(`${API_BASE}/flow/validate`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/yaml',
      },
      body: flowData,
    });
    return handleResponse<ValidateFlowResponse>(response);
  },

  async executeFlow(request: ExecuteFlowRequest): Promise<ExecutionResult> {
    const response = await fetch(`${API_BASE}/flow/execute`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(request),
    });
    return handleResponse<ExecutionResult>(response);
  },

  async getProviders(): Promise<ProvidersResponse> {
    const response = await fetch(`${API_BASE}/providers`);
    return handleResponse<ProvidersResponse>(response);
  },
};
