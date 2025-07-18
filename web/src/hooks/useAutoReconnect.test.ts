import { renderHook, act } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { useAutoReconnect } from './useAutoReconnect';

describe('useAutoReconnect', () => {
  let mockConnect: ReturnType<typeof vi.fn>;
  let mockDisconnect: ReturnType<typeof vi.fn>;
  let mockAddEventListener: ReturnType<typeof vi.fn>;
  let mockRemoveEventListener: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    mockConnect = vi.fn();
    mockDisconnect = vi.fn();
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
      })
    );

    expect(mockAddEventListener).toHaveBeenCalledWith('visibilitychange', expect.any(Function));
    expect(mockAddEventListener).toHaveBeenCalledWith('focus', expect.any(Function));

    unmount();

    expect(mockRemoveEventListener).toHaveBeenCalledWith('visibilitychange', expect.any(Function));
    expect(mockRemoveEventListener).toHaveBeenCalledWith('focus', expect.any(Function));
  });

  it('should disconnect when page becomes hidden and autoDisconnect is enabled', () => {
    renderHook(() =>
      useAutoReconnect({
        connectionState: 'connected',
        connect: mockConnect,
        disconnect: mockDisconnect,
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

  it('should attempt reconnection when page becomes visible and connection is disconnected', () => {
    renderHook(() =>
      useAutoReconnect({
        connectionState: 'disconnected',
        connect: mockConnect,
        disconnect: mockDisconnect,
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

    expect(mockConnect).toHaveBeenCalledTimes(1);
  });

  it('should not attempt reconnection when page becomes visible but connection is already connected', () => {
    renderHook(() =>
      useAutoReconnect({
        connectionState: 'connected',
        connect: mockConnect,
        disconnect: mockDisconnect,
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

    expect(mockConnect).not.toHaveBeenCalled();
  });

  it('should attempt reconnection on window focus when disconnected', () => {
    renderHook(() =>
      useAutoReconnect({
        connectionState: 'disconnected',
        connect: mockConnect,
        disconnect: mockDisconnect,
      })
    );

    // Get the window focus handler
    const focusHandler = mockAddEventListener.mock.calls.find(
      (call) => call[0] === 'focus'
    )?.[1];

    expect(focusHandler).toBeDefined();
    focusHandler();

    expect(mockConnect).toHaveBeenCalledTimes(1);
  });

  it('should use default values for optional parameters', () => {
    const { unmount } = renderHook(() =>
      useAutoReconnect({
        connectionState: 'disconnected',
        connect: mockConnect,
        disconnect: mockDisconnect,
      })
    );

    // Should not throw an error and should work with defaults
    expect(() => unmount()).not.toThrow();
  });

  describe('ref pattern behavior', () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('should not attempt reconnection multiple times within delay period', () => {
      const { rerender } = renderHook(
        ({ connectionState }) =>
          useAutoReconnect({
            connectionState,
            connect: mockConnect,
            disconnect: mockDisconnect,
            delayMs: 2000,
          }),
        { initialProps: { connectionState: 'disconnected' as const } }
      );

      // Get the visibilitychange handler
      const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'visibilitychange'
      )?.[1];

      expect(visibilityChangeHandler).toBeDefined();

      // Simulate page becoming visible (should trigger reconnection)
      Object.defineProperty(document, 'hidden', { value: false, writable: true });
      act(() => {
        visibilityChangeHandler();
      });

      expect(mockConnect).toHaveBeenCalledTimes(1);

      // Try to trigger reconnection again immediately (should be prevented)
      act(() => {
        visibilityChangeHandler();
      });

      expect(mockConnect).toHaveBeenCalledTimes(1); // Still only called once

      // Advance time but not enough to reset the flag
      act(() => {
        vi.advanceTimersByTime(1000);
      });

      // Try to trigger reconnection again (should still be prevented)
      act(() => {
        visibilityChangeHandler();
      });

      expect(mockConnect).toHaveBeenCalledTimes(1); // Still only called once

      // Advance time to reset the flag
      act(() => {
        vi.advanceTimersByTime(1000);
      });

      // Now reconnection should be allowed again
      act(() => {
        visibilityChangeHandler();
      });

      expect(mockConnect).toHaveBeenCalledTimes(2); // Called twice now
    });

    it('should use updated connection state from ref', () => {
      const { rerender } = renderHook(
        ({ connectionState }) =>
          useAutoReconnect({
            connectionState,
            connect: mockConnect,
            disconnect: mockDisconnect,
          }),
        { initialProps: { connectionState: 'connected' as const } }
      );

      // Get the visibilitychange handler
      const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'visibilitychange'
      )?.[1];

      expect(visibilityChangeHandler).toBeDefined();

      // Simulate page becoming visible with connected state (should not reconnect)
      Object.defineProperty(document, 'hidden', { value: false, writable: true });
      act(() => {
        visibilityChangeHandler();
      });

      expect(mockConnect).not.toHaveBeenCalled();

      // Update connection state to disconnected
      rerender({ connectionState: 'disconnected' });

      // Now attempt reconnection (should work with updated state)
      act(() => {
        visibilityChangeHandler();
      });

      expect(mockConnect).toHaveBeenCalledTimes(1);
    });

    it('should use updated connect function from ref', () => {
      const newMockConnect = vi.fn();
      const { rerender } = renderHook(
        ({ connect }) =>
          useAutoReconnect({
            connectionState: 'disconnected',
            connect,
            disconnect: mockDisconnect,
          }),
        { initialProps: { connect: mockConnect } }
      );

      // Get the visibilitychange handler
      const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'visibilitychange'
      )?.[1];

      expect(visibilityChangeHandler).toBeDefined();

      // Update connect function
      rerender({ connect: newMockConnect });

      // Trigger reconnection (should use new connect function)
      Object.defineProperty(document, 'hidden', { value: false, writable: true });
      act(() => {
        visibilityChangeHandler();
      });

      expect(mockConnect).not.toHaveBeenCalled();
      expect(newMockConnect).toHaveBeenCalledTimes(1);
    });

    it('should use updated disconnect and autoDisconnect from ref', () => {
      const newMockDisconnect = vi.fn();
      const { rerender } = renderHook(
        ({ disconnect, autoDisconnect }) =>
          useAutoReconnect({
            connectionState: 'connected',
            connect: mockConnect,
            disconnect,
            autoDisconnect,
          }),
        { initialProps: { disconnect: mockDisconnect, autoDisconnect: true } }
      );

      // Get the visibilitychange handler
      const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'visibilitychange'
      )?.[1];

      expect(visibilityChangeHandler).toBeDefined();

      // Update disconnect function and autoDisconnect
      rerender({ disconnect: newMockDisconnect, autoDisconnect: false });

      // Trigger disconnect (should not disconnect because autoDisconnect is false)
      Object.defineProperty(document, 'hidden', { value: true, writable: true });
      act(() => {
        visibilityChangeHandler();
      });

      expect(mockDisconnect).not.toHaveBeenCalled();
      expect(newMockDisconnect).not.toHaveBeenCalled();

      // Enable autoDisconnect
      rerender({ disconnect: newMockDisconnect, autoDisconnect: true });

      // Trigger disconnect (should use new disconnect function)
      act(() => {
        visibilityChangeHandler();
      });

      expect(mockDisconnect).not.toHaveBeenCalled();
      expect(newMockDisconnect).toHaveBeenCalledTimes(1);
    });

    it('should cleanup timeout on unmount', () => {
      const clearTimeoutSpy = vi.spyOn(global, 'clearTimeout');
      
      const { unmount } = renderHook(() =>
        useAutoReconnect({
          connectionState: 'disconnected',
          connect: mockConnect,
          disconnect: mockDisconnect,
          delayMs: 2000,
        })
      );

      // Get the visibilitychange handler
      const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'visibilitychange'
      )?.[1];

      expect(visibilityChangeHandler).toBeDefined();

      // Trigger reconnection to start timeout
      Object.defineProperty(document, 'hidden', { value: false, writable: true });
      act(() => {
        visibilityChangeHandler();
      });

      expect(mockConnect).toHaveBeenCalledTimes(1);

      // Unmount component
      unmount();

      // Verify timeout was cleared
      expect(clearTimeoutSpy).toHaveBeenCalled();
      
      clearTimeoutSpy.mockRestore();
    });

    it('should clear existing timeout before setting new one', () => {
      const clearTimeoutSpy = vi.spyOn(global, 'clearTimeout');
      
      renderHook(() =>
        useAutoReconnect({
          connectionState: 'disconnected',
          connect: mockConnect,
          disconnect: mockDisconnect,
          delayMs: 2000,
        })
      );

      // Get the visibilitychange handler
      const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'visibilitychange'
      )?.[1];

      expect(visibilityChangeHandler).toBeDefined();

      // Trigger reconnection to start timeout
      Object.defineProperty(document, 'hidden', { value: false, writable: true });
      act(() => {
        visibilityChangeHandler();
      });

      expect(mockConnect).toHaveBeenCalledTimes(1);

      // Advance time to allow the flag to reset
      act(() => {
        vi.advanceTimersByTime(2000);
      });

      // Trigger reconnection again (should clear previous timeout and set new one)
      act(() => {
        visibilityChangeHandler();
      });

      expect(mockConnect).toHaveBeenCalledTimes(2); // Should be called again
      expect(clearTimeoutSpy).toHaveBeenCalledTimes(1); // Should have cleared the previous timeout
      
      clearTimeoutSpy.mockRestore();
    });
  });
});