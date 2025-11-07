const { useState, useEffect, useCallback } = React;
const { ReactFlow, Controls, Background, MiniMap, useNodesState, useEdgesState } = window.ReactFlow;

function App() {
    const [flow, setFlow] = useState(null);
    const [nodes, setNodes, onNodesChange] = useNodesState([]);
    const [edges, setEdges, onEdgesChange] = useEdgesState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [selectedNode, setSelectedNode] = useState(null);
    const [inputs, setInputs] = useState({});
    const [executing, setExecuting] = useState(false);
    const [executionResult, setExecutionResult] = useState(null);

    // Load flow on mount
    useEffect(() => {
        loadFlow();
    }, []);

    async function loadFlow() {
        try {
            const response = await fetch('/api/flow');
            if (!response.ok) {
                throw new Error('Failed to load flow');
            }
            const data = await response.json();
            setFlow(data);
            buildGraph(data);
            setLoading(false);
        } catch (err) {
            setError(err.message);
            setLoading(false);
        }
    }

    function buildGraph(flowData) {
        const newNodes = [];
        const newEdges = [];
        const nodePositions = calculateNodePositions(flowData.nodes);

        // Create nodes
        flowData.nodes.forEach((node, index) => {
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
                    node: node
                },
            });

            // Create edges from inputs
            node.inputs.forEach(input => {
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

        setNodes(newNodes);
        setEdges(newEdges);
    }

    function calculateNodePositions(nodes) {
        // Simple layout: arrange nodes in levels based on dependencies
        const positions = {};
        const levels = {};
        const visited = new Set();

        // Build adjacency list
        const adjacencyList = {};
        nodes.forEach(node => {
            adjacencyList[node.id] = [];
            node.inputs.forEach(input => {
                if (input.from !== 'input') {
                    const parts = input.from.split('.');
                    if (parts.length === 2) {
                        adjacencyList[node.id].push(parts[0]);
                    }
                }
            });
        });

        // Calculate levels using DFS
        function getLevel(nodeId) {
            if (levels[nodeId] !== undefined) return levels[nodeId];

            const deps = adjacencyList[nodeId];
            if (deps.length === 0) {
                levels[nodeId] = 0;
                return 0;
            }

            let maxLevel = -1;
            deps.forEach(dep => {
                maxLevel = Math.max(maxLevel, getLevel(dep));
            });

            levels[nodeId] = maxLevel + 1;
            return levels[nodeId];
        }

        nodes.forEach(node => getLevel(node.id));

        // Arrange nodes by level
        const nodesByLevel = {};
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

    const onNodeClick = useCallback((event, node) => {
        setSelectedNode(node.data.node);
    }, []);

    async function executeFlow() {
        if (!flow) return;

        setExecuting(true);
        setExecutionResult(null);
        setError(null);

        try {
            const response = await fetch('/api/flow/execute', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    flow: flow,
                    inputs: inputs,
                }),
            });

            const result = await response.json();
            setExecutionResult(result);
            setExecuting(false);
        } catch (err) {
            setError(err.message);
            setExecuting(false);
        }
    }

    function updateInput(key, value) {
        setInputs(prev => ({
            ...prev,
            [key]: value,
        }));
    }

    if (loading) {
        return <div className="loading">Loading flow...</div>;
    }

    if (error && !flow) {
        return <div className="error">Error: {error}</div>;
    }

    return (
        <div className="app-container">
            <header className="header">
                <h1>Prompt Flow Visualizer</h1>
                {flow && <p>{flow.name} - {flow.description}</p>}
            </header>

            <div className="main-content">
                <aside className="sidebar">
                    {flow && (
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
                                <div className="info-value">{flow.config?.default_provider || 'N/A'}</div>
                            </div>
                            <div className="info-item">
                                <div className="info-label">Model</div>
                                <div className="info-value">{flow.config?.default_model || 'N/A'}</div>
                            </div>
                        </div>
                    )}

                    {selectedNode && (
                        <div className="node-details">
                            <h3>Node: {selectedNode.id}</h3>
                            <div className="detail-row">
                                <span className="detail-label">Type:</span>
                                <span className="detail-value">{selectedNode.type}</span>
                            </div>
                            {selectedNode.provider && (
                                <div className="detail-row">
                                    <span className="detail-label">Provider:</span>
                                    <span className="detail-value">{selectedNode.provider}</span>
                                </div>
                            )}
                            {selectedNode.model && (
                                <div className="detail-row">
                                    <span className="detail-label">Model:</span>
                                    <span className="detail-value">{selectedNode.model}</span>
                                </div>
                            )}
                            <div className="detail-row">
                                <span className="detail-label">Inputs:</span>
                                <span className="detail-value">{selectedNode.inputs.length}</span>
                            </div>
                            <div className="detail-row">
                                <span className="detail-label">Outputs:</span>
                                <span className="detail-value">{selectedNode.outputs.length}</span>
                            </div>
                            {selectedNode.prompt && (
                                <div style={{ marginTop: '0.75rem' }}>
                                    <div className="detail-label" style={{ marginBottom: '0.5rem' }}>Prompt:</div>
                                    <pre style={{
                                        fontSize: '0.75rem',
                                        backgroundColor: '#fff',
                                        padding: '0.5rem',
                                        borderRadius: '4px',
                                        overflow: 'auto',
                                        maxHeight: '150px'
                                    }}>{selectedNode.prompt}</pre>
                                </div>
                            )}
                        </div>
                    )}

                    <div className="test-section">
                        <h2>Test Flow</h2>
                        <div className="input-group">
                            <label>Input (user_input)</label>
                            <textarea
                                value={inputs.user_input || ''}
                                onChange={(e) => updateInput('user_input', e.target.value)}
                                placeholder="Enter test input..."
                            />
                        </div>
                        <button
                            className="btn"
                            onClick={executeFlow}
                            disabled={executing}
                        >
                            {executing ? 'Executing...' : 'Execute Flow'}
                        </button>
                    </div>

                    {executionResult && (
                        <div className="results-section">
                            <h2>Results</h2>
                            {executionResult.success ? (
                                <div className="success">✓ Execution successful</div>
                            ) : (
                                <div className="error">✗ Execution failed: {executionResult.error}</div>
                            )}

                            <div className="info-item">
                                <div className="info-label">Duration</div>
                                <div className="info-value">{(executionResult.duration / 1000000).toFixed(0)}ms</div>
                            </div>

                            {executionResult.node_results && executionResult.node_results.map((nodeResult, idx) => (
                                <div key={idx} className="result-item">
                                    <h4>Node: {nodeResult.node_id}</h4>
                                    {nodeResult.outputs && Object.entries(nodeResult.outputs).map(([key, value]) => (
                                        <div key={key}>
                                            <div className="detail-label">{key}:</div>
                                            <pre>{typeof value === 'string' ? value : JSON.stringify(value, null, 2)}</pre>
                                        </div>
                                    ))}
                                    <div className="metrics">
                                        <div className="metric">
                                            <span className="metric-label">Tokens: </span>
                                            <span className="metric-value">{nodeResult.metrics?.tokens_used || 0}</span>
                                        </div>
                                        <div className="metric">
                                            <span className="metric-label">Cost: </span>
                                            <span className="metric-value">${nodeResult.metrics?.estimated_cost?.toFixed(6) || '0.000000'}</span>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </aside>

                <div className="canvas">
                    <ReactFlow
                        nodes={nodes}
                        edges={edges}
                        onNodesChange={onNodesChange}
                        onEdgesChange={onEdgesChange}
                        onNodeClick={onNodeClick}
                        fitView
                    >
                        <Background />
                        <Controls />
                        <MiniMap />
                    </ReactFlow>
                </div>
            </div>
        </div>
    );
}

ReactDOM.render(<App />, document.getElementById('root'));
