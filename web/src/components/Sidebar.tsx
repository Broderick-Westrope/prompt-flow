import type { Flow, FlowNode, ExecutionResult } from '../types/flow';
import { FlowInfo } from './FlowInfo';
import { NodeDetails } from './NodeDetails';
import { TestSection } from './TestSection';
import { ResultsSection } from './ResultsSection';

interface SidebarProps {
  flow: Flow | null;
  selectedNode: FlowNode | null;
  inputs: Record<string, string>;
  rootInputs: string[];
  executing: boolean;
  executionResult: ExecutionResult | null;
  onInputChange: (key: string, value: string) => void;
  onExecute: () => void;
}

export function Sidebar({
  flow,
  selectedNode,
  inputs,
  rootInputs,
  executing,
  executionResult,
  onInputChange,
  onExecute,
}: SidebarProps) {
  return (
    <aside className="sidebar">
      {flow && <FlowInfo flow={flow} />}

      {selectedNode && <NodeDetails node={selectedNode} />}

      <TestSection
        inputs={inputs}
        rootInputs={rootInputs}
        executing={executing}
        onInputChange={onInputChange}
        onExecute={onExecute}
      />

      {executionResult && <ResultsSection result={executionResult} />}
    </aside>
  );
}
