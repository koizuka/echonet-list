/**
 * Format a value for display in logs or UI
 */
export function formatValue(value: unknown): string {
  if (value === null) {
    return 'null';
  }
  
  if (value === undefined) {
    return 'undefined';
  }
  
  if (typeof value === 'string') {
    return value;
  }
  
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }
  
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value, null, 2);
    } catch {
      return '[Object]';
    }
  }
  
  // Remaining types (bigint, symbol, function) have meaningful string forms
  // oxlint-disable-next-line typescript/no-base-to-string
  return String(value);
}
