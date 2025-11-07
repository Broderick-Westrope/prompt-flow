import type { ChangeEvent } from 'react';

interface TestSectionProps {
  inputs: Record<string, string>;
  rootInputs: string[];
  executing: boolean;
  onInputChange: (key: string, value: string) => void;
  onExecute: () => void;
}

export function TestSection({
  inputs,
  rootInputs,
  executing,
  onInputChange,
  onExecute,
}: TestSectionProps) {
  const handleChange = (inputName: string) => (e: ChangeEvent<HTMLTextAreaElement>) => {
    onInputChange(inputName, e.target.value);
  };

  return (
    <div className="test-section">
      <h2>Test Flow</h2>
      {rootInputs.length === 0 ? (
        <div className="info-message">No inputs required for this flow</div>
      ) : (
        rootInputs.map(inputName => (
          <div key={inputName} className="input-group">
            <label htmlFor={`input-${inputName}`}>Input ({inputName})</label>
            <textarea
              id={`input-${inputName}`}
              value={inputs[inputName] || ''}
              onChange={handleChange(inputName)}
              placeholder={`Enter ${inputName}...`}
            />
          </div>
        ))
      )}
      <button className="btn" onClick={onExecute} disabled={executing}>
        {executing ? 'Executing...' : 'Execute Flow'}
      </button>
    </div>
  );
}
