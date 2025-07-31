import { renderHook, act } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { useAutoReconnect } from './useAutoReconnect';
import type { ConnectionState } from './types';

describe('useAutoReconnect', () => {
  let mockConnect: ReturnType<typeof vi.fn>;
  let mockDisconnect: ReturnType<typeof vi.fn>;
  let mockSetConnectionState: ReturnType<typeof vi.fn>;
  let mockAddEventListener: ReturnType<typeof vi.fn>;
  let mockRemoveEventListener: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    mockConnect = vi.fn();
    mockDisconnect = vi.fn();
    mockSetConnectionState = vi.fn();
    mockAddEventListener = vi.fn();
    mockRemoveEventListener = vi.fn();

    // Mock document and window event listeners
    Object.defineProperty(document, 'addEventListener', {
      value: mockAddEventListener,
      writable: true,
    });
    Object.defineProperty(document, 'removeEventListener', {
      value: mockRemoveEventListener,
      writable: true,
    });
    Object.defineProperty(window, 'addEventListener', {
      value: mockAddEventListener,
      writable: true,
    });
    Object.defineProperty(window, 'removeEventListener', {
      value: mockRemoveEventListener,
      writable: true,
    });

    // Mock document.hidden as visible by default
    Object.defineProperty(document, 'hidden', {
      value: false,
      writable: true,
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
    vi.clearAllTimers();
  });

  it('should set up event listeners on mount', () => {
    const { unmount } = renderHook(() =>
      useAutoReconnect({
        connectionState: 'disconnected',
        connect: mockConnect,
        disconnect: mockDisconnect,
        setConnectionState: mockSetConnectionState,
      })
    );

    expect(mockAddEventListener).toHaveBeenCalledWith('visibilitychange', expect.any(Function));
    expect(mockAddEventListener).toHaveBeenCalledWith('focus', expect.any(Function));

    unmount();

    expect(mockRemoveEventListener).toHaveBeenCalledWith('visibilitychange', expect.any(Function));
    expect(mockRemoveEventListener).toHaveBeenCalledWith('focus', expect.any(Function));
  });

  it('should trigger reconnecting state when page becomes visible and connection is disconnected', async () => {
    vi.useFakeTimers();
    
    renderHook(() =>
      useAutoReconnect({
        connectionState: 'disconnected',
        connect: mockConnect,
        disconnect: mockDisconnect,
        setConnectionState: mockSetConnectionState,
      })
    );

    const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
      (call) => call[0] === 'visibilitychange'
    )?.[1];

    expect(visibilityChangeHandler).toBeDefined();

    Object.defineProperty(document, 'hidden', { value: false, writable: true });
    visibilityChangeHandler();

    // Wait for debounce timeout (100ms)
    act(() => {
      vi.advanceTimersByTime(100);
    });

    expect(mockSetConnectionState).toHaveBeenCalledWith('reconnecting');
    
    vi.useRealTimers();
  });

  it('should trigger reconnecting state on focus when connection is disconnected', () => {
    vi.useFakeTimers();
    
    renderHook(() =>
      useAutoReconnect({
        connectionState: 'disconnected',
        connect: mockConnect,
        disconnect: mockDisconnect,
        setConnectionState: mockSetConnectionState,
      })
    );

    const focusHandler = mockAddEventListener.mock.calls.find(
      (call) => call[0] === 'focus'
    )?.[1];

    expect(focusHandler).toBeDefined();
    focusHandler();

    // Wait for debounce timeout (100ms)
    act(() => {
      vi.advanceTimersByTime(100);
    });

    expect(mockSetConnectionState).toHaveBeenCalledWith('reconnecting');
    
    vi.useRealTimers();
  });

  it('should connect when state changes to reconnecting', () => {
    const { rerender } = renderHook(
      ({ connectionState }: { connectionState: ConnectionState }) =>
        useAutoReconnect({
          connectionState,
          connect: mockConnect,
          disconnect: mockDisconnect,
          setConnectionState: mockSetConnectionState,
        }),
      {
        initialProps: { connectionState: 'disconnected' as ConnectionState },
      }
    );

    expect(mockConnect).not.toHaveBeenCalled();

    rerender({ connectionState: 'reconnecting' as const });

    expect(mockConnect).toHaveBeenCalledTimes(1);
  });

  it('should disconnect when page becomes hidden and autoDisconnect is enabled', () => {
    renderHook(() =>
      useAutoReconnect({
        connectionState: 'connected',
        connect: mockConnect,
        disconnect: mockDisconnect,
        setConnectionState: mockSetConnectionState,
        autoDisconnect: true,
      })
    );

    // Get the visibilitychange handler
    const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
      (call) => call[0] === 'visibilitychange'
    )?.[1];

    expect(visibilityChangeHandler).toBeDefined();

    // Simulate page becoming hidden
    Object.defineProperty(document, 'hidden', { value: true, writable: true });
    visibilityChangeHandler();

    expect(mockDisconnect).toHaveBeenCalledTimes(1);
  });

  it('should not disconnect when page becomes hidden and autoDisconnect is disabled', () => {
    renderHook(() =>
      useAutoReconnect({
        connectionState: 'connected',
        connect: mockConnect,
        disconnect: mockDisconnect,
        setConnectionState: mockSetConnectionState,
        autoDisconnect: false,
      })
    );

    // Get the visibilitychange handler
    const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
      (call) => call[0] === 'visibilitychange'
    )?.[1];

    expect(visibilityChangeHandler).toBeDefined();

    // Simulate page becoming hidden
    Object.defineProperty(document, 'hidden', { value: true, writable: true });
    visibilityChangeHandler();

    expect(mockDisconnect).not.toHaveBeenCalled();
  });

  it('should not disconnect when page becomes hidden but connection is not connected', () => {
    renderHook(() =>
      useAutoReconnect({
        connectionState: 'disconnected',
        connect: mockConnect,
        disconnect: mockDisconnect,
        setConnectionState: mockSetConnectionState,
        autoDisconnect: true,
      })
    );

    // Get the visibilitychange handler
    const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
      (call) => call[0] === 'visibilitychange'
    )?.[1];

    expect(visibilityChangeHandler).toBeDefined();

    // Simulate page becoming hidden
    Object.defineProperty(document, 'hidden', { value: true, writable: true });
    visibilityChangeHandler();

    expect(mockDisconnect).not.toHaveBeenCalled();
  });


  it('should not trigger reconnecting when page becomes visible but connection is already connected', () => {
    renderHook(() =>
      useAutoReconnect({
        connectionState: 'connected',
        connect: mockConnect,
        disconnect: mockDisconnect,
        setConnectionState: mockSetConnectionState,
      })
    );

    // Get the visibilitychange handler
    const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
      (call) => call[0] === 'visibilitychange'
    )?.[1];

    expect(visibilityChangeHandler).toBeDefined();

    // Simulate page becoming visible
    Object.defineProperty(document, 'hidden', { value: false, writable: true });
    visibilityChangeHandler();

    expect(mockSetConnectionState).not.toHaveBeenCalled();
  });


  it('should use default values for optional parameters', () => {
    const { unmount } = renderHook(() =>
      useAutoReconnect({
        connectionState: 'disconnected',
        connect: mockConnect,
        disconnect: mockDisconnect,
        setConnectionState: mockSetConnectionState,
      })
    );

    // Should not throw an error and should work with defaults
    expect(() => unmount()).not.toThrow();
  });

  describe('state management behavior', () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('should prevent reconnection attempts when already in progress', () => {
      const { rerender } = renderHook(
        ({ connectionState }: { connectionState: ConnectionState }) =>
          useAutoReconnect({
            connectionState,
            connect: mockConnect,
            disconnect: mockDisconnect,
            setConnectionState: mockSetConnectionState,
          }),
        {
          initialProps: { connectionState: 'disconnected' as ConnectionState },
        }
      );

      // Get the visibilitychange handler
      const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'visibilitychange'
      )?.[1];

      expect(visibilityChangeHandler).toBeDefined();

      // First trigger should set reconnecting state after debounce
      Object.defineProperty(document, 'hidden', { value: false, writable: true });
      act(() => {
        visibilityChangeHandler();
      });

      // Wait for debounce timeout (100ms)
      act(() => {
        vi.advanceTimersByTime(100);
      });

      expect(mockSetConnectionState).toHaveBeenCalledWith('reconnecting');

      // Simulate state change to reconnecting
      act(() => {
        rerender({ connectionState: 'reconnecting' as const });
      });

      expect(mockConnect).toHaveBeenCalledTimes(1);

      // Subsequent triggers should not set reconnecting state while in progress
      mockSetConnectionState.mockClear();
      act(() => {
        visibilityChangeHandler();
      });

      // Wait for debounce timeout again
      act(() => {
        vi.advanceTimersByTime(100);
      });

      expect(mockSetConnectionState).not.toHaveBeenCalled();
    });


    it('should use updated refs for autoDisconnect', () => {
      let currentAutoDisconnect = true;
      
      const { rerender } = renderHook(
        () =>
          useAutoReconnect({
            connectionState: 'connected',
            connect: mockConnect,
            disconnect: mockDisconnect,
            setConnectionState: mockSetConnectionState,
            autoDisconnect: currentAutoDisconnect,
          })
      );

      // Get the visibilitychange handler
      const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'visibilitychange'
      )?.[1];

      expect(visibilityChangeHandler).toBeDefined();

      // Disable autoDisconnect
      currentAutoDisconnect = false;
      rerender();

      // Trigger disconnect (should not disconnect because autoDisconnect is false)
      Object.defineProperty(document, 'hidden', { value: true, writable: true });
      act(() => {
        visibilityChangeHandler();
      });

      expect(mockDisconnect).not.toHaveBeenCalled();
    });
  });
});