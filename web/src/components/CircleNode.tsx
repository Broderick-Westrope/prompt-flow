import { memo } from 'react';
import { Handle, Position } from '@xyflow/react';

interface CircleNodeProps {
  data: {
    label: string;
    isStart?: boolean;
    isEnd?: boolean;
  };
  selected?: boolean;
}

export const CircleNode = memo(({ data, selected }: CircleNodeProps) => {
  const { label, isStart, isEnd } = data;

  return (
    <div
      style={{
        width: '100%',
        height: '100%',
        borderRadius: '50%',
        border: '2px solid #333',
        background: '#fff',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        boxShadow: selected
          ? '0 0 0 3px rgba(0, 123, 255, 0.5)'
          : '0 2px 4px rgba(0, 0, 0, 0.1)',
        borderColor: selected ? '#007bff' : '#333',
      }}
    >
      {/* Output handle for start node (bottom) */}
      {isStart && (
        <Handle
          type="source"
          position={Position.Bottom}
          style={{ background: '#555' }}
        />
      )}

      {/* Node label */}
      <div
        style={{
          fontWeight: '600',
          fontSize: '14px',
          color: '#333',
          textTransform: 'capitalize',
        }}
      >
        {label}
      </div>

      {/* Input handle for end node (top) */}
      {isEnd && (
        <Handle
          type="target"
          position={Position.Top}
          style={{ background: '#555' }}
        />
      )}
    </div>
  );
});

CircleNode.displayName = 'CircleNode';
