import type { FlowNode } from '../types/flow';

interface NodeDetailsProps {
  node: FlowNode;
}

export function NodeDetails({ node }: NodeDetailsProps) {
  return (
    <div className="node-details">
      <h3>Node: {node.id}</h3>
      <div className="detail-row">
        <span className="detail-label">Type:</span>
        <span className="detail-value">{node.type}</span>
      </div>
      {node.provider && (
        <div className="detail-row">
          <span className="detail-label">Provider:</span>
          <span className="detail-value">{node.provider}</span>
        </div>
      )}
      {node.model && (
        <div className="detail-row">
          <span className="detail-label">Model:</span>
          <span className="detail-value">{node.model}</span>
        </div>
      )}
      <div className="detail-row">
        <span className="detail-label">Inputs:</span>
        <span className="detail-value">{node.inputs.length}</span>
      </div>
      <div className="detail-row">
        <span className="detail-label">Outputs:</span>
        <span className="detail-value">{node.outputs.length}</span>
      </div>
      {node.prompt && (
        <div style={{ marginTop: '0.75rem' }}>
          <div className="detail-label" style={{ marginBottom: '0.5rem' }}>
            Prompt:
          </div>
          <pre
            style={{
              fontSize: '0.75rem',
              backgroundColor: '#fff',
              padding: '0.5rem',
              borderRadius: '4px',
              overflow: 'auto',
              maxHeight: '150px',
            }}
          >
            {node.prompt}
          </pre>
        </div>
      )}
    </div>
  );
}
