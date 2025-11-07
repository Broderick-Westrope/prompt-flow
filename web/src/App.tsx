import { useState } from 'react';
import { Header } from './components/Header';
import { Sidebar } from './components/Sidebar';
import { FlowCanvas } from './components/FlowCanvas';
import { useFlow } from './hooks/useFlow';
import { api } from './services/api';
import type { FlowNode, ExecutionResult } from './types/flow';
import './App.css';

function App() {
  const { flow, loading, error } = useFlow();
  const [selectedNode, setSelectedNode] = useState<FlowNode | null>(null);
  const [inputs, setInputs] = useState<Record<string, string>>({});
  const [executing, setExecuting] = useState(false);
  const [executionResult, setExecutionResult] = useState<ExecutionResult | null>(
    null
  );
  const [executionError, setExecutionError] = useState<string | null>(null);

  const handleInputChange = (key: string, value: string) => {
    setInputs((prev) => ({
      ...prev,
      [key]: value,
    }));
  };

  const handleExecuteFlow = async () => {
    if (!flow) return;

    setExecuting(true);
    setExecutionResult(null);
    setExecutionError(null);

    try {
      const result = await api.executeFlow({
        flow,
        inputs,
      });
      setExecutionResult(result);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Execution failed';
      setExecutionError(message);
    } finally {
      setExecuting(false);
    }
  };

  const handleNodeClick = (node: FlowNode) => {
    setSelectedNode(node);
  };

  if (loading) {
    return <div className="loading">Loading flow...</div>;
  }

  if (error && !flow) {
    return <div className="error">Error: {error}</div>;
  }

  if (!flow) {
    return <div className="error">No flow loaded</div>;
  }

  return (
    <div className="app-container">
      <Header flow={flow} />

      <div className="main-content">
        <Sidebar
          flow={flow}
          selectedNode={selectedNode}
          inputs={inputs}
          executing={executing}
          executionResult={executionResult}
          onInputChange={handleInputChange}
          onExecute={handleExecuteFlow}
        />

        <FlowCanvas flow={flow} onNodeClick={handleNodeClick} />
      </div>

      {executionError && (
        <div className="error" style={{ margin: '1rem' }}>
          Error: {executionError}
        </div>
      )}
    </div>
  );
}

export default App;
