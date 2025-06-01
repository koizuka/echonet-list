import { renderHook, act } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { useCardExpansion } from './useCardExpansion';

describe('useCardExpansion', () => {
  it('should initialize with no expanded cards', () => {
    const { result } = renderHook(() => useCardExpansion());
    
    expect(result.current.expandedCount).toBe(0);
    expect(result.current.isCardExpanded('device1')).toBe(false);
    expect(result.current.isCardExpanded('device2')).toBe(false);
  });

  it('should toggle card expansion state', () => {
    const { result } = renderHook(() => useCardExpansion());
    
    // Initially not expanded
    expect(result.current.isCardExpanded('device1')).toBe(false);
    
    // Toggle to expand
    act(() => {
      result.current.toggleCard('device1');
    });
    expect(result.current.isCardExpanded('device1')).toBe(true);
    expect(result.current.expandedCount).toBe(1);
    
    // Toggle to collapse
    act(() => {
      result.current.toggleCard('device1');
    });
    expect(result.current.isCardExpanded('device1')).toBe(false);
    expect(result.current.expandedCount).toBe(0);
  });

  it('should expand specific card', () => {
    const { result } = renderHook(() => useCardExpansion());
    
    act(() => {
      result.current.expandCard('device1');
    });
    
    expect(result.current.isCardExpanded('device1')).toBe(true);
    expect(result.current.expandedCount).toBe(1);
    
    // Expanding again should not change state
    act(() => {
      result.current.expandCard('device1');
    });
    expect(result.current.expandedCount).toBe(1);
  });

  it('should collapse specific card', () => {
    const { result } = renderHook(() => useCardExpansion());
    
    // First expand the card
    act(() => {
      result.current.expandCard('device1');
    });
    expect(result.current.isCardExpanded('device1')).toBe(true);
    
    // Then collapse it
    act(() => {
      result.current.collapseCard('device1');
    });
    expect(result.current.isCardExpanded('device1')).toBe(false);
    expect(result.current.expandedCount).toBe(0);
    
    // Collapsing again should not change state
    act(() => {
      result.current.collapseCard('device1');
    });
    expect(result.current.expandedCount).toBe(0);
  });

  it('should handle multiple cards independently', () => {
    const { result } = renderHook(() => useCardExpansion());
    
    act(() => {
      result.current.expandCard('device1');
      result.current.expandCard('device2');
    });
    
    expect(result.current.isCardExpanded('device1')).toBe(true);
    expect(result.current.isCardExpanded('device2')).toBe(true);
    expect(result.current.expandedCount).toBe(2);
    
    act(() => {
      result.current.collapseCard('device1');
    });
    
    expect(result.current.isCardExpanded('device1')).toBe(false);
    expect(result.current.isCardExpanded('device2')).toBe(true);
    expect(result.current.expandedCount).toBe(1);
  });

  it('should expand all cards', () => {
    const { result } = renderHook(() => useCardExpansion());
    const deviceKeys = ['device1', 'device2', 'device3'];
    
    act(() => {
      result.current.expandAll(deviceKeys);
    });
    
    deviceKeys.forEach(key => {
      expect(result.current.isCardExpanded(key)).toBe(true);
    });
    expect(result.current.expandedCount).toBe(3);
  });

  it('should collapse all cards', () => {
    const { result } = renderHook(() => useCardExpansion());
    
    // First expand some cards
    act(() => {
      result.current.expandCard('device1');
      result.current.expandCard('device2');
    });
    expect(result.current.expandedCount).toBe(2);
    
    // Then collapse all
    act(() => {
      result.current.collapseAll();
    });
    
    expect(result.current.isCardExpanded('device1')).toBe(false);
    expect(result.current.isCardExpanded('device2')).toBe(false);
    expect(result.current.expandedCount).toBe(0);
  });

  it('should replace expanded cards when expandAll is called', () => {
    const { result } = renderHook(() => useCardExpansion());
    
    // Expand some cards manually
    act(() => {
      result.current.expandCard('device1');
      result.current.expandCard('device2');
    });
    expect(result.current.expandedCount).toBe(2);
    
    // Call expandAll with different set
    act(() => {
      result.current.expandAll(['device3', 'device4']);
    });
    
    expect(result.current.isCardExpanded('device1')).toBe(false);
    expect(result.current.isCardExpanded('device2')).toBe(false);
    expect(result.current.isCardExpanded('device3')).toBe(true);
    expect(result.current.isCardExpanded('device4')).toBe(true);
    expect(result.current.expandedCount).toBe(2);
  });
});