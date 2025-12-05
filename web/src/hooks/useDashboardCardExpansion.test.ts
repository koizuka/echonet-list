import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useDashboardCardExpansion } from './useDashboardCardExpansion';

describe('useDashboardCardExpansion', () => {
  const STORAGE_KEY = 'echonet-list-dashboard-expanded-cards';

  beforeEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
  });

  it('should start with no expanded cards by default', () => {
    const { result } = renderHook(() => useDashboardCardExpansion());

    expect(result.current.isExpanded('device1')).toBe(false);
    expect(result.current.isExpanded('device2')).toBe(false);
  });

  it('should toggle expansion state', () => {
    const { result } = renderHook(() => useDashboardCardExpansion());

    // Expand
    act(() => {
      result.current.toggleExpansion('device1');
    });
    expect(result.current.isExpanded('device1')).toBe(true);

    // Collapse
    act(() => {
      result.current.toggleExpansion('device1');
    });
    expect(result.current.isExpanded('device1')).toBe(false);
  });

  it('should manage multiple cards independently', () => {
    const { result } = renderHook(() => useDashboardCardExpansion());

    act(() => {
      result.current.toggleExpansion('device1');
      result.current.toggleExpansion('device2');
    });

    expect(result.current.isExpanded('device1')).toBe(true);
    expect(result.current.isExpanded('device2')).toBe(true);
    expect(result.current.isExpanded('device3')).toBe(false);
  });

  it('should persist expansion state to localStorage', () => {
    const { result } = renderHook(() => useDashboardCardExpansion());

    act(() => {
      result.current.toggleExpansion('device1');
    });

    const saved = localStorage.getItem(STORAGE_KEY);
    expect(saved).toBe(JSON.stringify(['device1']));
  });

  it('should restore expansion state from localStorage', () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(['device1', 'device2']));

    const { result } = renderHook(() => useDashboardCardExpansion());

    expect(result.current.isExpanded('device1')).toBe(true);
    expect(result.current.isExpanded('device2')).toBe(true);
    expect(result.current.isExpanded('device3')).toBe(false);
  });

  it('should handle invalid localStorage data gracefully', () => {
    localStorage.setItem(STORAGE_KEY, 'invalid-json');

    const { result } = renderHook(() => useDashboardCardExpansion());

    expect(result.current.isExpanded('device1')).toBe(false);
  });
});
