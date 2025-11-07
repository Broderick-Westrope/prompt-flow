export interface Config {
  showStartEndNode: boolean;
}

export async function fetchConfig(): Promise<Config> {
  const response = await fetch('/api/config');
  if (!response.ok) {
    throw new Error('Failed to fetch config');
  }
  return response.json();
}
