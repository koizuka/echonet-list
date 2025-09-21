/**
 * Utilities for generating unique IDs throughout the application
 */

/**
 * Generates a unique ID with timestamp and random component for robust uniqueness
 *
 * @param prefix - Optional prefix for the ID (e.g., 'error', 'slider-error')
 * @returns A unique ID string
 *
 * @example
 * generateUniqueId() // "1640123456789-abc123def"
 * generateUniqueId('error') // "error-1640123456789-abc123def"
 * generateUniqueId('slider-error') // "slider-error-1640123456789-abc123def"
 */
export function generateUniqueId(prefix?: string): string {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substr(2, 9);

  if (prefix) {
    return `${prefix}-${timestamp}-${random}`;
  }

  return `${timestamp}-${random}`;
}

/**
 * Generates a unique ID for LogEntry objects
 * This is a convenience function that uses generateUniqueId with appropriate formatting
 *
 * @param type - The type/category of the log entry (e.g., 'error', 'info', 'warn')
 * @returns A unique ID string formatted for LogEntry usage
 *
 * @example
 * generateLogEntryId('error') // "error-1640123456789-abc123def"
 * generateLogEntryId('online') // "online-1640123456789-abc123def"
 */
export function generateLogEntryId(type: string): string {
  return generateUniqueId(type);
}