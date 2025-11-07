import { useEffect, useCallback } from 'react';
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

interface FlowCanvasProps {
  flow: Flow;
  onNodeClick: (node: FlowNode) => void;
}

interface NodeData extends Record<string, unknown> {
  label: React.ReactElement;
  node: FlowNode;
}

interface NodePositions {
  [nodeId: string]: { x: number; y: number };
}

function calculateNodePositions(nodes: FlowNode[]): NodePositions {
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

  // Calculate positions
  const levelWidth = 300;
  const nodeHeight = 120;

  Object.entries(nodesByLevel).forEach(([level, nodeIds]) => {
    nodeIds.forEach((nodeId, index) => {
      positions[nodeId] = {
        x: parseInt(level) * levelWidth + 50,
        y: index * nodeHeight + 50,
      };
    });
  });

  return positions;
}

function buildGraphNodes(flowData: Flow): [Node<NodeData>[], Edge[]] {
  const newNodes: Node<NodeData>[] = [];
  const newEdges: Edge[] = [];
  const nodePositions = calculateNodePositions(flowData.nodes);

  // Create nodes
  flowData.nodes.forEach((node) => {
    newNodes.push({
      id: node.id,
      type: 'default',
      position: nodePositions[node.id],
      data: {
        label: (
          <div>
            <div style={{ fontWeight: 'bold' }}>{node.id}</div>
            <div style={{ fontSize: '12px', color: '#666' }}>{node.type}</div>
          </div>
        ),
        node: node,
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
