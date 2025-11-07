import type { Flow } from '../types/flow';

interface HeaderProps {
  flow: Flow | null;
}

export function Header({ flow }: HeaderProps) {
  return (
    <header className="header">
      <h1>Prompt Flow Visualizer</h1>
      {flow && (
        <p>
          {flow.name} - {flow.description}
        </p>
      )}
    </header>
  );
}
