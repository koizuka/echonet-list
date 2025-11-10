import { renderHook, act } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach, afterEach, type MockedFunction } from 'vitest';
import { useAutoReconnect } from './useAutoReconnect';
import type { ConnectionState } from './types';

describe('useAutoReconnect', () => {
  let mockConnect: MockedFunction<() => void>;
  let mockDisconnect: MockedFunction<() => void>;
  let mockAddEventListener: ReturnType<typeof vi.fn>;
  let mockRemoveEventListener: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.useFakeTimers();
    mockConnect = vi.fn(() => {});
    mockDisconnect = vi.fn(() => {});
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
    vi.useRealTimers();
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

    unmount();

    expect(mockRemoveEventListener).toHaveBeenCalledWith('visibilitychange', expect.any(Function));
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

    // Should not disconnect immediately (due to guard time)
    expect(mockDisconnect).not.toHaveBeenCalled();

    // Should disconnect after default guard time (3000ms)
    vi.advanceTimersByTime(3000);
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

  it('should not attempt reconnection when page becomes visible', () => {
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

    // Should NOT connect on visibility change (only on focus)
    expect(mockConnect).not.toHaveBeenCalled();
  });

  it('should attempt reconnection when page is shown and connection is disconnected', () => {
    renderHook(() =>
      useAutoReconnect({
        connectionState: 'disconnected',
        connect: mockConnect,
        disconnect: mockDisconnect,
      })
    );

    // Get the pageshow handler
    const pageshowHandler = mockAddEventListener.mock.calls.find(
      (call) => call[0] === 'pageshow'
    )?.[1];

    expect(pageshowHandler).toBeDefined();

    // Simulate page being shown
    pageshowHandler({ persisted: false });

    // Should trigger reconnection after timeout
    vi.advanceTimersByTime(200);
    expect(mockConnect).toHaveBeenCalledTimes(1);
  });

  it('should not attempt reconnection when page is shown but connection is already connected', () => {
    renderHook(() =>
      useAutoReconnect({
        connectionState: 'connected',
        connect: mockConnect,
        disconnect: mockDisconnect,
      })
    );

    // Get the pageshow handler
    const pageshowHandler = mockAddEventListener.mock.calls.find(
      (call) => call[0] === 'pageshow'
    )?.[1];

    expect(pageshowHandler).toBeDefined();

    // Simulate page being shown
    pageshowHandler({ persisted: false });

    // Advance timers to complete any delayed operations
    vi.advanceTimersByTime(200);
    
    expect(mockConnect).not.toHaveBeenCalled();
  });

  it('should handle asymmetric event processing correctly', () => {
    // Test to verify that pageshow triggers connection and visibility only triggers disconnection
    renderHook(() =>
      useAutoReconnect({
        connectionState: 'disconnected',
        connect: mockConnect,
        disconnect: mockDisconnect,
      })
    );

    // Get both handlers
    const pageshowHandler = mockAddEventListener.mock.calls.find(
      (call) => call[0] === 'pageshow'
    )?.[1];
    
    const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
      (call) => call[0] === 'visibilitychange'
    )?.[1];

    expect(pageshowHandler).toBeDefined();
    expect(visibilityChangeHandler).toBeDefined();

    // Test 1: Pageshow should trigger connection when disconnected
    pageshowHandler({ persisted: false });
    vi.advanceTimersByTime(200);
    expect(mockConnect).toHaveBeenCalledTimes(1);
    expect(mockDisconnect).not.toHaveBeenCalled();

    // Test 2: Visibility change to visible should NOT trigger connection
    mockConnect.mockClear();
    Object.defineProperty(document, 'hidden', { value: false, writable: true });
    visibilityChangeHandler();
    expect(mockConnect).not.toHaveBeenCalled();
    expect(mockDisconnect).not.toHaveBeenCalled();

    // Test 3: Visibility change to hidden should NOT trigger disconnection when disconnected
    // (disconnection only happens when connected)
    mockDisconnect.mockClear();
    Object.defineProperty(document, 'hidden', { value: true, writable: true });
    visibilityChangeHandler();
    expect(mockDisconnect).not.toHaveBeenCalled();
    expect(mockConnect).not.toHaveBeenCalled();
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
      renderHook(() =>
        useAutoReconnect({
          connectionState: 'disconnected',
          connect: mockConnect,
          disconnect: mockDisconnect,
          delayMs: 2000,
        })
      );

      // Get the pageshow handler
      const pageshowHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'pageshow'
      )?.[1];

      expect(pageshowHandler).toBeDefined();

      // Simulate page being shown (should trigger reconnection)
      act(() => {
        pageshowHandler({ persisted: false });
      });

      // Advance timers to complete delayed reconnection
      vi.advanceTimersByTime(200);
      expect(mockConnect).toHaveBeenCalledTimes(1);

      // Try to trigger reconnection again immediately (should be prevented)
      act(() => {
        pageshowHandler({ persisted: false });
      });

      expect(mockConnect).toHaveBeenCalledTimes(1); // Still only called once

      // Advance time but not enough to reset the flag
      act(() => {
        vi.advanceTimersByTime(1000);
      });

      // Try to trigger reconnection again (should still be prevented)
      act(() => {
        pageshowHandler({ persisted: false });
      });

      expect(mockConnect).toHaveBeenCalledTimes(1); // Still only called once

      // Advance time to reset the flag
      act(() => {
        vi.advanceTimersByTime(1000);
      });

      // Now reconnection should be allowed again
      act(() => {
        pageshowHandler({ persisted: false });
      });

      // Advance timers to complete the delayed operation
      vi.advanceTimersByTime(200);
      expect(mockConnect).toHaveBeenCalledTimes(2); // Called twice now
    });

    it('should use updated connection state from ref', () => {
      let currentConnectionState: 'connected' | 'disconnected' = 'connected';

      const { rerender } = renderHook(
        () =>
          useAutoReconnect({
            connectionState: currentConnectionState,
            connect: mockConnect,
            disconnect: mockDisconnect,
          })
      );

      // Get the focus handler
      const pageshowHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'pageshow'
      )?.[1];

      expect(pageshowHandler).toBeDefined();

      // Simulate window gaining focus with connected state (should not reconnect)
      act(() => {
        pageshowHandler({ persisted: false });
      });

      // Advance timers
      vi.advanceTimersByTime(200);
      expect(mockConnect).not.toHaveBeenCalled();

      // Update connection state to disconnected
      currentConnectionState = 'disconnected';
      rerender();

      // Now attempt reconnection (should work with updated state)
      act(() => {
        pageshowHandler({ persisted: false });
      });

      // Advance timers
      vi.advanceTimersByTime(200);
      expect(mockConnect).toHaveBeenCalledTimes(1);
    });

    it('should use updated connect function from ref', () => {
      const newMockConnect = vi.fn();
      let currentConnect = mockConnect;

      const { rerender } = renderHook(
        () =>
          useAutoReconnect({
            connectionState: 'disconnected',
            connect: currentConnect,
            disconnect: mockDisconnect,
          })
      );

      // Get the focus handler
      const pageshowHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'pageshow'
      )?.[1];

      expect(pageshowHandler).toBeDefined();

      // Update connect function
      currentConnect = newMockConnect;
      rerender();

      // Trigger reconnection (should use new connect function)
      act(() => {
        pageshowHandler({ persisted: false });
      });

      // Advance timers
      vi.advanceTimersByTime(200);
      expect(mockConnect).not.toHaveBeenCalled();
      expect(newMockConnect).toHaveBeenCalledTimes(1);
    });

    it('should use updated disconnect and autoDisconnect from ref', () => {
      const newMockDisconnect = vi.fn();
      let currentDisconnect = mockDisconnect;
      let currentAutoDisconnect = true;

      const { rerender } = renderHook(
        () =>
          useAutoReconnect({
            connectionState: 'connected',
            connect: mockConnect,
            disconnect: currentDisconnect,
            autoDisconnect: currentAutoDisconnect,
          })
      );

      // Get the visibilitychange handler
      const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'visibilitychange'
      )?.[1];

      expect(visibilityChangeHandler).toBeDefined();

      // Update disconnect function and autoDisconnect
      currentDisconnect = newMockDisconnect;
      currentAutoDisconnect = false;
      rerender();

      // Trigger disconnect (should not disconnect because autoDisconnect is false)
      Object.defineProperty(document, 'hidden', { value: true, writable: true });
      act(() => {
        visibilityChangeHandler();
      });

      expect(mockDisconnect).not.toHaveBeenCalled();
      expect(newMockDisconnect).not.toHaveBeenCalled();

      // Enable autoDisconnect
      currentAutoDisconnect = true;
      rerender();

      // Trigger disconnect (should schedule delayed disconnect with new function)
      act(() => {
        visibilityChangeHandler();
      });

      // Should not disconnect immediately
      expect(mockDisconnect).not.toHaveBeenCalled();
      expect(newMockDisconnect).not.toHaveBeenCalled();

      // Advance timers to trigger delayed disconnect (using default 3000ms)
      act(() => {
        vi.advanceTimersByTime(3000);
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

      // Get the focus handler
      const pageshowHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'pageshow'
      )?.[1];

      expect(pageshowHandler).toBeDefined();

      // Trigger reconnection to start timeout
      act(() => {
        pageshowHandler({ persisted: false });
      });

      // Advance timers
      vi.advanceTimersByTime(200);
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

      // Get the focus handler
      const pageshowHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'pageshow'
      )?.[1];

      expect(pageshowHandler).toBeDefined();

      // Trigger reconnection to start timeout
      act(() => {
        pageshowHandler({ persisted: false });
      });

      // Advance timers
      vi.advanceTimersByTime(200);
      expect(mockConnect).toHaveBeenCalledTimes(1);

      // Advance time to allow the flag to reset
      act(() => {
        vi.advanceTimersByTime(2000);
      });

      // Trigger reconnection again (should clear previous timeout and set new one)
      act(() => {
        pageshowHandler({ persisted: false });
      });

      // Advance timers for the second reconnection
      vi.advanceTimersByTime(200);
      expect(mockConnect).toHaveBeenCalledTimes(2); // Should be called again
      expect(clearTimeoutSpy).toHaveBeenCalledTimes(1); // Should have cleared the previous timeout
      
      clearTimeoutSpy.mockRestore();
    });

    it('should not reset flag when connection becomes connected within delay period', () => {
      const { rerender } = renderHook(
        ({ connectionState }: { connectionState: ConnectionState }) =>
          useAutoReconnect({
            connectionState,
            connect: mockConnect,
            disconnect: mockDisconnect,
            delayMs: 2000,
          }),
        {
          initialProps: { connectionState: 'disconnected' as ConnectionState },
        }
      );

      // Get the focus handler
      const pageshowHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'pageshow'
      )?.[1];

      // Trigger reconnection
      act(() => {
        pageshowHandler({ persisted: false });
      });

      // Advance timers
      vi.advanceTimersByTime(200);
      expect(mockConnect).toHaveBeenCalledTimes(1);

      // Simulate connection becoming connected
      rerender({ connectionState: 'connected' as const });

      // Advance time past the delay period
      act(() => {
        vi.advanceTimersByTime(2000);
      });

      // Try to trigger reconnection again (should not reconnect as we're connected)
      act(() => {
        pageshowHandler({ persisted: false });
      });

      expect(mockConnect).toHaveBeenCalledTimes(1); // Should still be 1
    });

    it('should allow reconnection after delay if still disconnected', () => {
      renderHook(() =>
        useAutoReconnect({
          connectionState: 'disconnected',
          connect: mockConnect,
          disconnect: mockDisconnect,
          delayMs: 2000,
        })
      );

      // Get the focus handler
      const pageshowHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'pageshow'
      )?.[1];

      // First reconnection attempt
      act(() => {
        pageshowHandler({ persisted: false });
      });

      // Advance timers
      vi.advanceTimersByTime(200);
      expect(mockConnect).toHaveBeenCalledTimes(1);

      // Advance time past the delay period (still disconnected)
      act(() => {
        vi.advanceTimersByTime(2000);
      });

      // Second reconnection attempt should work
      act(() => {
        pageshowHandler({ persisted: false });
      });

      // Advance timers
      vi.advanceTimersByTime(200);
      expect(mockConnect).toHaveBeenCalledTimes(2);
    });

    it('should handle rapid visibility events', () => {
      renderHook(() =>
        useAutoReconnect({
          connectionState: 'disconnected',
          connect: mockConnect,
          disconnect: mockDisconnect,
          delayMs: 2000,
        })
      );

      // Get focus handler
      const pageshowHandler = mockAddEventListener.mock.calls.find(
        (call) => call[0] === 'pageshow'
      )?.[1];

      // Simulate rapid events
      // Focus event
      act(() => {
        pageshowHandler({ persisted: false });
      });

      // Multiple focus events in quick succession
      act(() => {
        pageshowHandler({ persisted: false });
        pageshowHandler({ persisted: false });
      });

      // Advance timers to complete any delayed operations
      vi.advanceTimersByTime(200);
      // Should only connect once despite multiple events
      expect(mockConnect).toHaveBeenCalledTimes(1);
    });
  });

  describe('delayed disconnect guard time', () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('should delay disconnect when page becomes hidden', () => {
      renderHook(() =>
        useAutoReconnect({
          connectionState: 'connected',
          connect: mockConnect,
          disconnect: mockDisconnect,
          autoDisconnect: true,
          disconnectDelayMs: 2000,
        })
      );

      // Simulate page becoming hidden
      act(() => {
        Object.defineProperty(document, 'hidden', { value: true, writable: true });
        Object.defineProperty(document, 'visibilityState', { value: 'hidden', writable: true });
        const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
          call => call[0] === 'visibilitychange'
        )?.[1];
        visibilityChangeHandler?.();
      });

      // Should not disconnect immediately
      expect(mockDisconnect).not.toHaveBeenCalled();

      // Should disconnect after delay
      act(() => {
        vi.advanceTimersByTime(2000);
      });

      expect(mockDisconnect).toHaveBeenCalledTimes(1);
    });

    it('should cancel delayed disconnect when page becomes visible before timeout', () => {
      renderHook(() =>
        useAutoReconnect({
          connectionState: 'connected',
          connect: mockConnect,
          disconnect: mockDisconnect,
          autoDisconnect: true,
          disconnectDelayMs: 2000,
        })
      );

      // Simulate page becoming hidden
      act(() => {
        Object.defineProperty(document, 'hidden', { value: true, writable: true });
        Object.defineProperty(document, 'visibilityState', { value: 'hidden', writable: true });
        const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
          call => call[0] === 'visibilitychange'
        )?.[1];
        visibilityChangeHandler?.();
      });

      // Advance time partially
      act(() => {
        vi.advanceTimersByTime(1000);
      });

      // Page becomes visible again before timeout
      act(() => {
        Object.defineProperty(document, 'hidden', { value: false, writable: true });
        Object.defineProperty(document, 'visibilityState', { value: 'visible', writable: true });
        const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
          call => call[0] === 'visibilitychange'
        )?.[1];
        visibilityChangeHandler?.();
      });

      // Advance remaining time
      act(() => {
        vi.advanceTimersByTime(2000);
      });

      // Should not have disconnected
      expect(mockDisconnect).not.toHaveBeenCalled();
    });

    it('should cancel delayed disconnect on pageshow event', () => {
      renderHook(() =>
        useAutoReconnect({
          connectionState: 'connected',
          connect: mockConnect,
          disconnect: mockDisconnect,
          autoDisconnect: true,
          disconnectDelayMs: 2000,
        })
      );

      // Simulate page hide to trigger delayed disconnect
      act(() => {
        const pageHideHandler = mockAddEventListener.mock.calls.find(
          call => call[0] === 'pagehide'
        )?.[1];
        pageHideHandler?.();
      });

      // Advance time partially
      act(() => {
        vi.advanceTimersByTime(1000);
      });

      // Simulate pageshow event
      act(() => {
        const pageShowHandler = mockAddEventListener.mock.calls.find(
          call => call[0] === 'pageshow'
        )?.[1];
        pageShowHandler?.({ persisted: false });
      });

      // Advance remaining time
      act(() => {
        vi.advanceTimersByTime(2000);
      });

      // Should not have disconnected
      expect(mockDisconnect).not.toHaveBeenCalled();
    });

    it('should cancel delayed disconnect on window focus event', () => {
      renderHook(() =>
        useAutoReconnect({
          connectionState: 'connected',
          connect: mockConnect,
          disconnect: mockDisconnect,
          autoDisconnect: true,
          disconnectDelayMs: 2000,
        })
      );

      // Simulate page hide to trigger delayed disconnect
      act(() => {
        const pageHideHandler = mockAddEventListener.mock.calls.find(
          call => call[0] === 'pagehide'
        )?.[1];
        pageHideHandler?.();
      });

      // Advance time partially
      act(() => {
        vi.advanceTimersByTime(1000);
      });

      // Simulate window focus event
      act(() => {
        const focusHandler = mockAddEventListener.mock.calls.find(
          call => call[0] === 'focus'
        )?.[1];
        focusHandler?.();
      });

      // Advance remaining time
      act(() => {
        vi.advanceTimersByTime(2000);
      });

      // Should not have disconnected
      expect(mockDisconnect).not.toHaveBeenCalled();
    });

    it('should use custom disconnect delay time', () => {
      renderHook(() =>
        useAutoReconnect({
          connectionState: 'connected',
          connect: mockConnect,
          disconnect: mockDisconnect,
          autoDisconnect: true,
          disconnectDelayMs: 5000, // Custom 5 second delay
        })
      );

      // Simulate page becoming hidden
      act(() => {
        Object.defineProperty(document, 'hidden', { value: true, writable: true });
        Object.defineProperty(document, 'visibilityState', { value: 'hidden', writable: true });
        const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
          call => call[0] === 'visibilitychange'
        )?.[1];
        visibilityChangeHandler?.();
      });

      // Should not disconnect before custom delay
      act(() => {
        vi.advanceTimersByTime(4999);
      });
      expect(mockDisconnect).not.toHaveBeenCalled();

      // Should disconnect after custom delay
      act(() => {
        vi.advanceTimersByTime(1);
      });
      expect(mockDisconnect).toHaveBeenCalledTimes(1);
    });

    it('should use default 3000ms disconnect delay when not specified', () => {
      renderHook(() =>
        useAutoReconnect({
          connectionState: 'connected',
          connect: mockConnect,
          disconnect: mockDisconnect,
          autoDisconnect: true,
          // disconnectDelayMs not specified, should use default 3000ms
        })
      );

      // Simulate page becoming hidden
      act(() => {
        Object.defineProperty(document, 'hidden', { value: true, writable: true });
        Object.defineProperty(document, 'visibilityState', { value: 'hidden', writable: true });
        const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
          call => call[0] === 'visibilitychange'
        )?.[1];
        visibilityChangeHandler?.();
      });

      // Should not disconnect before default delay
      act(() => {
        vi.advanceTimersByTime(2999);
      });
      expect(mockDisconnect).not.toHaveBeenCalled();

      // Should disconnect after default delay
      act(() => {
        vi.advanceTimersByTime(1);
      });
      expect(mockDisconnect).toHaveBeenCalledTimes(1);
    });

    it('should handle multiple rapid visibility changes correctly', () => {
      renderHook(() =>
        useAutoReconnect({
          connectionState: 'connected',
          connect: mockConnect,
          disconnect: mockDisconnect,
          autoDisconnect: true,
          disconnectDelayMs: 2000,
        })
      );

      const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
        call => call[0] === 'visibilitychange'
      )?.[1];

      // Simulate page becoming hidden
      act(() => {
        Object.defineProperty(document, 'hidden', { value: true, writable: true });
        Object.defineProperty(document, 'visibilityState', { value: 'hidden', writable: true });
        visibilityChangeHandler?.();
      });

      // Advance time partially
      act(() => {
        vi.advanceTimersByTime(500);
      });

      // Page becomes visible (should cancel disconnect)
      act(() => {
        Object.defineProperty(document, 'hidden', { value: false, writable: true });
        Object.defineProperty(document, 'visibilityState', { value: 'visible', writable: true });
        visibilityChangeHandler?.();
      });

      // Page becomes hidden again quickly
      act(() => {
        Object.defineProperty(document, 'hidden', { value: true, writable: true });
        Object.defineProperty(document, 'visibilityState', { value: 'hidden', writable: true });
        visibilityChangeHandler?.();
      });

      // Advance time partially
      act(() => {
        vi.advanceTimersByTime(1000);
      });

      // Page becomes visible again (should cancel second disconnect)
      act(() => {
        Object.defineProperty(document, 'hidden', { value: false, writable: true });
        Object.defineProperty(document, 'visibilityState', { value: 'visible', writable: true });
        visibilityChangeHandler?.();
      });

      // Advance remaining time
      act(() => {
        vi.advanceTimersByTime(2000);
      });

      // Should not have disconnected at all
      expect(mockDisconnect).not.toHaveBeenCalled();
    });

    it('should not disconnect if page becomes visible before delayed disconnect executes (Safari fix)', () => {
      renderHook(() =>
        useAutoReconnect({
          connectionState: 'connected',
          connect: mockConnect,
          disconnect: mockDisconnect,
          autoDisconnect: true,
          disconnectDelayMs: 3000,
        })
      );

      // Simulate page becoming hidden
      act(() => {
        Object.defineProperty(document, 'hidden', { value: true, writable: true });
        Object.defineProperty(document, 'visibilityState', { value: 'hidden', writable: true });
        const visibilityChangeHandler = mockAddEventListener.mock.calls.find(
          call => call[0] === 'visibilitychange'
        )?.[1];
        visibilityChangeHandler?.();
      });

      // Advance time - but before the disconnect timeout fires
      act(() => {
        vi.advanceTimersByTime(2000);
      });

      // Page becomes visible again (Safari coming back from background)
      act(() => {
        Object.defineProperty(document, 'hidden', { value: false, writable: true });
        Object.defineProperty(document, 'visibilityState', { value: 'visible', writable: true });
      });

      // Now advance time to when the delayed disconnect timer would fire
      // This simulates Safari's behavior where timers fire after returning from background
      act(() => {
        vi.advanceTimersByTime(1000);
      });

      // Should NOT have disconnected because the timeout callback re-checks visibility
      expect(mockDisconnect).not.toHaveBeenCalled();
    });
  });
});