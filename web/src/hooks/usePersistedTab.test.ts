import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { usePersistedTab } from './usePersistedTab';

// Mock localStorage
const mockLocalStorage = (() => {
  let store: Record<string, string> = {};
  
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: vi.fn(() => {
      store = {};
    }),
    get length() {
      return Object.keys(store).length;
    },
    key: vi.fn((index: number) => Object.keys(store)[index] || null)
  };
})();

Object.defineProperty(window, 'localStorage', {
  value: mockLocalStorage
});

describe('usePersistedTab', () => {
  beforeEach(() => {
    mockLocalStorage.clear();
    vi.clearAllMocks();
  });

  it('should initialize with first available tab when no saved tab exists', () => {
    const availableTabs = ['All', 'Living Room', 'Kitchen'];
    const { result } = renderHook(() => usePersistedTab(availableTabs));
    
    expect(result.current.selectedTab).toBe('All');
  });

  it('should initialize with default tab when provided', () => {
    const availableTabs = ['All', 'Living Room', 'Kitchen'];
    const { result } = renderHook(() => usePersistedTab(availableTabs, 'Kitchen'));
    
    expect(result.current.selectedTab).toBe('Kitchen');
  });

  it('should restore saved tab from localStorage', () => {
    mockLocalStorage.setItem('echonet-list-selected-tab', 'Living Room');
    
    const availableTabs = ['All', 'Living Room', 'Kitchen'];
    const { result } = renderHook(() => usePersistedTab(availableTabs));
    
    expect(result.current.selectedTab).toBe('Living Room');
  });

  it('should fallback to default when saved tab is not available', () => {
    mockLocalStorage.setItem('echonet-list-selected-tab', 'Bathroom'); // Not in available tabs
    
    const availableTabs = ['All', 'Living Room', 'Kitchen'];
    const { result } = renderHook(() => usePersistedTab(availableTabs));
    
    expect(result.current.selectedTab).toBe('All');
  });

  it('should save tab selection to localStorage', () => {
    const availableTabs = ['All', 'Living Room', 'Kitchen'];
    const { result } = renderHook(() => usePersistedTab(availableTabs));
    
    act(() => {
      result.current.selectTab('Kitchen');
    });
     
    expect(result.current.selectedTab).toBe('Kitchen');
    expect(mockLocalStorage.setItem).toHaveBeenCalledWith('echonet-list-selected-tab', 'Kitchen');
  });

  it('should not select invalid tabs', () => {
    // Suppress console warnings for this test
    const originalWarn = console.warn;
    console.warn = vi.fn();
    
    const availableTabs = ['All', 'Living Room', 'Kitchen'];
    const { result } = renderHook(() => usePersistedTab(availableTabs));
    
    const initialTab = result.current.selectedTab;
    
    act(() => {
      result.current.selectTab('Invalid Tab');
    });
    
    // Should remain unchanged
    expect(result.current.selectedTab).toBe(initialTab);
    expect(mockLocalStorage.setItem).not.toHaveBeenCalledWith('echonet-list-selected-tab', 'Invalid Tab');
    
    // Verify console.warn was called
    expect(console.warn).toHaveBeenCalledWith('Attempted to select invalid tab: Invalid Tab');
    
    // Restore console.warn
    console.warn = originalWarn;
  });

  it('should update selected tab when available tabs change and current tab becomes invalid', () => {
    const { result, rerender } = renderHook(
      ({ tabs }) => usePersistedTab(tabs),
      { initialProps: { tabs: ['All', 'Living Room', 'Kitchen'] } }
    );
    
    // Select Kitchen
    act(() => {
      result.current.selectTab('Kitchen');
    });
    expect(result.current.selectedTab).toBe('Kitchen');
    
    // Update available tabs to remove Kitchen
    rerender({ tabs: ['All', 'Living Room', 'Bedroom'] });
    
    // Should fallback to first available tab
    expect(result.current.selectedTab).toBe('All');
  });

  it('should wait for available tabs before applying saved tab', () => {
    mockLocalStorage.setItem('echonet-list-selected-tab', 'Living Room');
    
    const { result, rerender } = renderHook(
      ({ tabs }) => usePersistedTab(tabs),
      { initialProps: { tabs: [] as string[]} } // Start with empty tabs
    );
    
    // Should start with saved tab even if tabs are empty
    expect(result.current.selectedTab).toBe('Living Room');
    
    // When tabs become available, should restore saved tab if it exists
    rerender({ tabs: ['All', 'Living Room', 'Kitchen'] });
    
    expect(result.current.selectedTab).toBe('Living Room');
  });

  it('should fallback when saved tab is not in available tabs after tabs load', () => {
    mockLocalStorage.setItem('echonet-list-selected-tab', 'Bathroom');
    
    const { result, rerender } = renderHook(
      ({ tabs }) => usePersistedTab(tabs),
      { initialProps: { tabs: [] as string[]} } // Start with empty tabs
    );
    
    // Should start with saved tab
    expect(result.current.selectedTab).toBe('Bathroom');
    
    // When tabs become available without the saved tab, should fallback
    rerender({ tabs: ['All', 'Living Room', 'Kitchen'] });
    
    expect(result.current.selectedTab).toBe('All');
  });

  it('should clear persisted tab', () => {
    mockLocalStorage.setItem('echonet-list-selected-tab', 'Living Room');
    
    const availableTabs = ['All', 'Living Room', 'Kitchen'];
    const { result } = renderHook(() => usePersistedTab(availableTabs));
    
    act(() => {
      result.current.clearPersistedTab();
    });
    
    expect(mockLocalStorage.removeItem).toHaveBeenCalledWith('echonet-list-selected-tab');
  });

  it('should handle localStorage errors gracefully', () => {
    // Suppress console warnings for this test
    const originalWarn = console.warn;
    console.warn = vi.fn();
    
    // Mock localStorage to throw errors
    const mockGetItem = vi.fn(() => {
      throw new Error('localStorage not available');
    });
    const mockSetItem = vi.fn(() => {
      throw new Error('localStorage not available');
    });
    
    Object.defineProperty(window, 'localStorage', {
      value: {
        getItem: mockGetItem,
        setItem: mockSetItem,
        removeItem: vi.fn(() => {
          throw new Error('localStorage not available');
        })
      }
    });
    
    const availableTabs = ['All', 'Living Room', 'Kitchen'];
    const { result } = renderHook(() => usePersistedTab(availableTabs));
    
    // Should still work with fallback
    expect(result.current.selectedTab).toBe('All');
    
    act(() => {
      result.current.selectTab('Kitchen');
    });
    
    // Should still update state even if localStorage fails
    expect(result.current.selectedTab).toBe('Kitchen');
    
    // Verify console.warn was called with localStorage errors
    expect(console.warn).toHaveBeenCalledWith('Failed to read from localStorage:', expect.any(Error));
    expect(console.warn).toHaveBeenCalledWith('Failed to save to localStorage:', expect.any(Error));
    
    // Restore console.warn
    console.warn = originalWarn;
  });

  it('should handle empty available tabs array', () => {
    // Suppress console warnings for this test if localStorage has been mocked to fail
    const originalWarn = console.warn;
    console.warn = vi.fn();
    
    const { result } = renderHook(() => usePersistedTab([]));
    
    expect(result.current.selectedTab).toBe('All'); // Default fallback
    
    // Restore console.warn
    console.warn = originalWarn;
  });
});