import { describe, it, expect } from 'vitest';
import { formatValue } from './formatValue';

describe('formatValue', () => {
  it('handles null and undefined', () => {
    expect(formatValue(null)).toBe('null');
    expect(formatValue(undefined)).toBe('undefined');
  });

  it('handles primitive types', () => {
    expect(formatValue('hello')).toBe('hello');
    expect(formatValue(123)).toBe('123');
    expect(formatValue(true)).toBe('true');
    expect(formatValue(false)).toBe('false');
  });

  it('handles objects', () => {
    const obj = { key: 'value', number: 42 };
    const result = formatValue(obj);
    expect(result).toContain('"key": "value"');
    expect(result).toContain('"number": 42');
  });

  it('handles arrays', () => {
    const arr = [1, 2, 3];
    const result = formatValue(arr);
    expect(result).toBe('[\n  1,\n  2,\n  3\n]');
  });

  it('handles circular references gracefully', () => {
    const obj: any = { name: 'test' };
    obj.self = obj; // Circular reference
    
    const result = formatValue(obj);
    expect(result).toBe('[Object]');
  });

  it('handles nested objects', () => {
    const nested = {
      level1: {
        level2: {
          value: 'deep'
        }
      }
    };
    
    const result = formatValue(nested);
    expect(result).toContain('"value": "deep"');
  });
});