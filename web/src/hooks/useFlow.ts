import { useState, useEffect } from 'react';
import type { Flow } from '../types/flow';
import { api } from '../services/api';

interface UseFlowResult {
  flow: Flow | null;
  loading: boolean;
  error: string | null;
  reload: () => Promise<void>;
}

export function useFlow(): UseFlowResult {
  const [flow, setFlow] = useState<Flow | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadFlow = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await api.getFlow();
      setFlow(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load flow';
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadFlow();
  }, []);

  return {
    flow,
    loading,
    error,
    reload: loadFlow,
  };
}
