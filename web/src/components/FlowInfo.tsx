import type { Flow } from '../types/flow';

interface FlowInfoProps {
  flow: Flow;
}

export function FlowInfo({ flow }: FlowInfoProps) {
  return (
    <div className="flow-info">
      <h2>Flow Information</h2>
      <div className="info-item">
        <div className="info-label">Name</div>
        <div className="info-value">{flow.name}</div>
      </div>
      <div className="info-item">
        <div className="info-label">Version</div>
        <div className="info-value">{flow.version}</div>
      </div>
      <div className="info-item">
        <div className="info-label">Nodes</div>
        <div className="info-value">{flow.nodes.length}</div>
      </div>
      <div className="info-item">
        <div className="info-label">Provider</div>
        <div className="info-value">
          {flow.config?.default_provider || 'N/A'}
        </div>
      </div>
      <div className="info-item">
        <div className="info-label">Model</div>
        <div className="info-value">
          {flow.config?.default_model || 'N/A'}
        </div>
      </div>
    </div>
  );
}
