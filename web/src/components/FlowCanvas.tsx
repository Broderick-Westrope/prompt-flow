import { useEffect, useCallback, useMemo } from 'react';
import {
  ReactFlow,
  Controls,
  Background,
  MiniMap,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import type { Flow, FlowNode } from '../types/flow';
import { CustomNode } from './CustomNode';

interface FlowCanvasProps {
  flow: Flow;
  onNodeClick: (node: FlowNode) => void;
}

interface NodeData extends Record<string, unknown> {
  node: FlowNode;
}

interface NodeDimensions {
  width: number;
  height: number;
}

interface NodePositions {
  [nodeId: string]: { x: number; y: number };
}

function calculateNodeDimensions(node: FlowNode): NodeDimensions {
  // Calculate width based on node name length
  // Using approximately 8-9 pixels per character at 14px font size
  const charWidth = 9;
  const horizontalPadding = 32; // 16px padding on each side
  const width = Math.max(node.id.length * charWidth + horizontalPadding, 100);

  // Fixed height for simple node display
  const height = 50;

  return {
    width,
    height,
  };
}

function calculateNodePositions(
  nodes: FlowNode[],
  nodeDimensions: Map<string, NodeDimensions>
): NodePositions {
  const positions: NodePositions = {};
  const levels: Record<string, number> = {};
  const adjacencyList: Record<string, string[]> = {};

  // Build adjacency list
  nodes.forEach((node) => {
    adjacencyList[node.id] = [];
    node.inputs.forEach((input) => {
      if (input.from !== 'input') {
        const parts = input.from.split('.');
        if (parts.length === 2) {
          adjacencyList[node.id].push(parts[0]);
        }
      }
    });
  });

  // Calculate levels using DFS
  function getLevel(nodeId: string): number {
    if (levels[nodeId] !== undefined) return levels[nodeId];

    const deps = adjacencyList[nodeId];
    if (deps.length === 0) {
      levels[nodeId] = 0;
      return 0;
    }

    let maxLevel = -1;
    deps.forEach((dep) => {
      maxLevel = Math.max(maxLevel, getLevel(dep));
    });

    levels[nodeId] = maxLevel + 1;
    return levels[nodeId];
  }

  nodes.forEach((node) => getLevel(node.id));

  // Arrange nodes by level
  const nodesByLevel: Record<number, string[]> = {};
  Object.entries(levels).forEach(([nodeId, level]) => {
    if (!nodesByLevel[level]) nodesByLevel[level] = [];
    nodesByLevel[level].push(nodeId);
  });

  // Calculate positions with dynamic spacing
  const levelWidth = 300;
  const verticalSpacing = 30;

  Object.entries(nodesByLevel).forEach(([level, nodeIds]) => {
    let currentY = 50;

    nodeIds.forEach((nodeId) => {
      const dimensions = nodeDimensions.get(nodeId) || { width: 100, height: 50 };

      positions[nodeId] = {
        x: parseInt(level) * levelWidth + 50,
        y: currentY,
      };

      // Move to next position with spacing
      currentY += dimensions.height + verticalSpacing;
    });
  });

  return positions;
}

function buildGraphNodes(flowData: Flow): [Node<NodeData>[], Edge[]] {
  const newNodes: Node<NodeData>[] = [];
  const newEdges: Edge[] = [];

  // Calculate dimensions for all nodes
  const nodeDimensions = new Map<string, NodeDimensions>();
  flowData.nodes.forEach((node) => {
    nodeDimensions.set(node.id, calculateNodeDimensions(node));
  });

  const nodePositions = calculateNodePositions(flowData.nodes, nodeDimensions);

  // Create nodes
  flowData.nodes.forEach((node) => {
    const dimensions = nodeDimensions.get(node.id)!;

    newNodes.push({
      id: node.id,
      type: 'custom',
      position: nodePositions[node.id],
      data: {
        node: node,
      },
      style: {
        width: dimensions.width,
        height: dimensions.height,
      },
    });

    // Create edges from inputs
    node.inputs.forEach((input) => {
      if (input.from !== 'input') {
        const parts = input.from.split('.');
        if (parts.length === 2) {
          newEdges.push({
            id: `${parts[0]}-${node.id}-${input.name}`,
            source: parts[0],
            target: node.id,
            label: input.name,
            animated: true,
          });
        }
      }
    });
  });

  return [newNodes, newEdges];
}

export function FlowCanvas({ flow, onNodeClick }: FlowCanvasProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState<Node<NodeData>>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  // Register custom node types
  const nodeTypes = useMemo(() => ({ custom: CustomNode }), []);

  useEffect(() => {
    const [newNodes, newEdges] = buildGraphNodes(flow);
    setNodes(newNodes);
    setEdges(newEdges);
  }, [flow, setNodes, setEdges]);

  const handleNodeClick = useCallback(
    (_event: React.MouseEvent, node: Node<NodeData>) => {
      onNodeClick(node.data.node);
    },
    [onNodeClick]
  );

  return (
    <div className="canvas">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={handleNodeClick}
        fitView
      >
        <Background />
        <Controls />
        <MiniMap />
      </ReactFlow>
    </div>
  );
}
