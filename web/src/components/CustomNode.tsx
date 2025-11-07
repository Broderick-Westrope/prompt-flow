import { memo } from 'react';
import { Handle, Position } from '@xyflow/react';
import type { FlowNode } from '../types/flow';

interface CustomNodeProps {
  data: {
    node: FlowNode;
    hasInputsFromNodes?: boolean;
    hasOutputsToNodes?: boolean;
  };
  selected?: boolean;
}

export const CustomNode = memo(({ data, selected }: CustomNodeProps) => {
  const { node, hasInputsFromNodes = true, hasOutputsToNodes = true } = data;

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
        boxShadow: selected
          ? '0 0 0 3px rgba(0, 123, 255, 0.5)'
          : '0 2px 4px rgba(0, 0, 0, 0.1)',
        borderColor: selected ? '#007bff' : '#333',
      }}
    >
      {/* Input handle - only show if there are inputs and they come from other nodes */}
      {node.inputs.length > 0 && hasInputsFromNodes && (
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

      {/* Output handle - only show if there are outputs and they go to other nodes */}
      {node.outputs.length > 0 && hasOutputsToNodes && (
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
