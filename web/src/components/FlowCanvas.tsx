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
import { CircleNode } from './CircleNode';

interface FlowCanvasProps {
  flow: Flow;
  onNodeSelect: (node: FlowNode | null) => void;
  showStartEndNode?: boolean;
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

function calculateCircleNodeDimensions(label: string): NodeDimensions {
  // Calculate diameter based on text length
  // For circles, we need more space to accommodate the text
  const charWidth = 9;
  const textWidth = label.length * charWidth;

  // Diameter needs to be large enough for text to fit comfortably
  // Using sqrt(2) * textWidth to ensure text fits in circle
  const diameter = Math.max(textWidth * 1.6, 70);

  return {
    width: diameter,
    height: diameter,
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

  // Calculate positions with dynamic spacing (vertical layout)
  const levelHeight = 150;
  const horizontalSpacing = 50;

  Object.entries(nodesByLevel).forEach(([level, nodeIds]) => {
    let currentX = 50;

    nodeIds.forEach((nodeId) => {
      const dimensions = nodeDimensions.get(nodeId) || { width: 100, height: 50 };

      positions[nodeId] = {
        x: currentX,
        y: parseInt(level) * levelHeight + 50,
      };

      // Move to next position with spacing
      currentX += dimensions.width + horizontalSpacing;
    });
  });

  return positions;
}

function buildGraphNodes(flowData: Flow, showStartEndNode: boolean): [Node[], Edge[]] {
  const newNodes: Node[] = [];
  const newEdges: Edge[] = [];

  // Calculate dimensions for all nodes
  const nodeDimensions = new Map<string, NodeDimensions>();
  flowData.nodes.forEach((node) => {
    nodeDimensions.set(node.id, calculateNodeDimensions(node));
  });

  const nodePositions = calculateNodePositions(flowData.nodes, nodeDimensions);

  // Only add start/end nodes if showStartEndNode is enabled
  if (showStartEndNode) {
    // Find nodes that take input from "input" (start edge nodes) and track their input names
    const startEdgeConnections: { nodeId: string; inputName: string }[] = [];
    flowData.nodes.forEach((node) => {
      node.inputs.forEach((input) => {
        if (input.from === 'input') {
          startEdgeConnections.push({ nodeId: node.id, inputName: input.name });
        }
      });
    });

    // Find nodes that output to "output" (end edge nodes) and track their output names
    const endEdgeConnections: { nodeId: string; outputName: string }[] = [];
    flowData.nodes.forEach((node) => {
      node.outputs.forEach((output) => {
        if (output.to === 'output') {
          endEdgeConnections.push({ nodeId: node.id, outputName: output.name });
        }
      });
    });

    // Calculate positions for start and end nodes (vertical layout)
    let minX = Infinity;
    let maxX = -Infinity;
    let maxY = -Infinity;
    Object.values(nodePositions).forEach((pos) => {
      minX = Math.min(minX, pos.x);
      maxX = Math.max(maxX, pos.x);
      maxY = Math.max(maxY, pos.y);
    });

    // Add start node if there are start edge connections
    if (startEdgeConnections.length > 0) {
      const startX = minX + (maxX - minX) / 2;
      const startDimensions = calculateCircleNodeDimensions('start');

      newNodes.push({
        id: 'start',
        type: 'circle',
        position: { x: startX, y: -150 },
        data: {
          label: 'start',
          isStart: true,
        },
        style: {
          width: startDimensions.width,
          height: startDimensions.height,
        },
      });

      // Create edges from start to start edge nodes with labels
      startEdgeConnections.forEach(({ nodeId, inputName }) => {
        newEdges.push({
          id: `start-${nodeId}-${inputName}`,
          source: 'start',
          target: nodeId,
          label: inputName,
          animated: true,
        });
      });
    }

    // Add end node if there are end edge connections
    if (endEdgeConnections.length > 0) {
      const endX = minX + (maxX - minX) / 2;
      const endDimensions = calculateCircleNodeDimensions('end');

      newNodes.push({
        id: 'end',
        type: 'circle',
        position: { x: endX, y: maxY + 250 },
        data: {
          label: 'end',
          isEnd: true,
        },
        style: {
          width: endDimensions.width,
          height: endDimensions.height,
        },
      });

      // Create edges from end edge nodes to end with labels
      endEdgeConnections.forEach(({ nodeId, outputName }) => {
        newEdges.push({
          id: `${nodeId}-${outputName}-end`,
          source: nodeId,
          target: 'end',
          label: outputName,
          animated: true,
        });
      });
    }
  }

  // Create regular nodes
  flowData.nodes.forEach((node) => {
    const dimensions = nodeDimensions.get(node.id)!;

    // Check if node has inputs/outputs from/to other nodes (not just "input"/"output")
    const hasInputsFromNodes = node.inputs.some((input) => input.from !== 'input');
    const hasOutputsToNodes = (() => {
      // Check if any other node references this node's outputs
      return flowData.nodes.some((otherNode) =>
        otherNode.inputs.some((input) => {
          if (input.from === 'input') return false;
          const parts = input.from.split('.');
          return parts.length === 2 && parts[0] === node.id;
        })
      );
    })();

    newNodes.push({
      id: node.id,
      type: 'custom',
      position: nodePositions[node.id],
      data: {
        node: node,
        hasInputsFromNodes: showStartEndNode || hasInputsFromNodes,
        hasOutputsToNodes: showStartEndNode || hasOutputsToNodes,
      },
      style: {
        width: dimensions.width,
        height: dimensions.height,
      },
    });

    // Create edges from inputs (excluding "input")
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

export function FlowCanvas({ flow, onNodeSelect, showStartEndNode = false }: FlowCanvasProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  // Register custom node types
  const nodeTypes = useMemo(() => ({
    custom: CustomNode,
    circle: CircleNode
  }), []);

  useEffect(() => {
    const [newNodes, newEdges] = buildGraphNodes(flow, showStartEndNode);
    setNodes(newNodes);
    setEdges(newEdges);
  }, [flow, showStartEndNode, setNodes, setEdges]);

  const handleSelectionChange = useCallback(
    ({ nodes: selectedNodes }: { nodes: Node[] }) => {
      if (selectedNodes.length > 0 && selectedNodes[0].data.node) {
        // A regular node is selected (not start/end)
        onNodeSelect(selectedNodes[0].data.node as FlowNode);
      } else {
        // No nodes selected or start/end node selected
        onNodeSelect(null);
      }
    },
    [onNodeSelect]
  );

  return (
    <div className="canvas">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onSelectionChange={handleSelectionChange}
        fitView
      >
        <Background />
        <Controls />
        <MiniMap />
      </ReactFlow>
    </div>
  );
}
