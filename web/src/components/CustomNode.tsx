import { memo } from 'react';
import { Handle, Position } from '@xyflow/react';
import type { FlowNode } from '../types/flow';

interface CustomNodeProps {
  data: {
    node: FlowNode;
  };
}

export const CustomNode = memo(({ data }: CustomNodeProps) => {
  const { node } = data;

  return (
    <div
      style={{
        padding: '12px 16px',
        borderRadius: '8px',
        border: '2px solid #333',
        background: '#fff',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100%',
        boxShadow: '0 2px 4px rgba(0, 0, 0, 0.1)',
      }}
    >
      {/* Input handle */}
      {node.inputs.length > 0 && (
        <Handle
          type="target"
          position={Position.Top}
          style={{ background: '#555' }}
        />
      )}

      {/* Node name */}
      <div
        style={{
          fontWeight: '600',
          fontSize: '14px',
          color: '#333',
          whiteSpace: 'nowrap',
        }}
      >
        {node.id}
      </div>

      {/* Output handle */}
      {node.outputs.length > 0 && (
        <Handle
          type="source"
          position={Position.Bottom}
          style={{ background: '#555' }}
        />
      )}
    </div>
  );
});

CustomNode.displayName = 'CustomNode';
