import type { ExecutionResult } from '../types/flow';

interface ResultsSectionProps {
  result: ExecutionResult;
}

export function ResultsSection({ result }: ResultsSectionProps) {
  return (
    <div className="results-section">
      <h2>Results</h2>
      {result.success ? (
        <div className="success">✓ Execution successful</div>
      ) : (
        <div className="error">✗ Execution failed: {result.error}</div>
      )}

      <div className="info-item">
        <div className="info-label">Duration</div>
        <div className="info-value">
          {(result.duration / 1000000).toFixed(0)}ms
        </div>
      </div>

      {result.node_results &&
        result.node_results.map((nodeResult, idx) => (
          <div key={idx} className="result-item">
            <h4>Node: {nodeResult.node_id}</h4>
            {nodeResult.outputs &&
              Object.entries(nodeResult.outputs).map(([key, value]) => (
                <div key={key}>
                  <div className="detail-label">{key}:</div>
                  <pre>
                    {typeof value === 'string'
                      ? value
                      : JSON.stringify(value, null, 2)}
                  </pre>
                </div>
              ))}
            <div className="metrics">
              <div className="metric">
                <span className="metric-label">Tokens: </span>
                <span className="metric-value">
                  {nodeResult.metrics?.tokens_used || 0}
                </span>
              </div>
              <div className="metric">
                <span className="metric-label">Cost: </span>
                <span className="metric-value">
                  ${nodeResult.metrics?.estimated_cost?.toFixed(6) || '0.000000'}
                </span>
              </div>
            </div>
          </div>
        ))}
    </div>
  );
}
