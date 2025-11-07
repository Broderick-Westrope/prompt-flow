import type { ChangeEvent } from 'react';

interface TestSectionProps {
  inputs: Record<string, string>;
  executing: boolean;
  onInputChange: (key: string, value: string) => void;
  onExecute: () => void;
}

export function TestSection({
  inputs,
  executing,
  onInputChange,
  onExecute,
}: TestSectionProps) {
  const handleChange = (e: ChangeEvent<HTMLTextAreaElement>) => {
    onInputChange('user_input', e.target.value);
  };

  return (
    <div className="test-section">
      <h2>Test Flow</h2>
      <div className="input-group">
        <label htmlFor="user-input">Input (user_input)</label>
        <textarea
          id="user-input"
          value={inputs.user_input || ''}
          onChange={handleChange}
          placeholder="Enter test input..."
        />
      </div>
      <button className="btn" onClick={onExecute} disabled={executing}>
        {executing ? 'Executing...' : 'Execute Flow'}
      </button>
    </div>
  );
}
