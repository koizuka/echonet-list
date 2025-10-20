import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { useDeviceHistory } from './useDeviceHistory';
import type { WebSocketConnection } from './useWebSocketConnection';

describe('useDeviceHistory', () => {
  let mockConnection: WebSocketConnection;

  beforeEach(() => {
    mockConnection = {
      connectionState: 'connected',
      sendMessage: vi.fn().mockResolvedValue({
        entries: [],
      }),
      connect: vi.fn(),
      disconnect: vi.fn(),
      connectedAt: new Date(),
      checkConnection: vi.fn().mockResolvedValue(true),
    };
  });

  it('should initialize with loading state', async () => {
    const { result } = renderHook(() =>
      useDeviceHistory({
        connection: mockConnection,
        target: '192.168.1.10 0130:1',
      })
    );

    expect(result.current.isLoading).toBe(true);
    expect(result.current.entries).toEqual([]);
    expect(result.current.error).toBeNull();

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });
  });

  it('should fetch history successfully', async () => {
    const mockEntries = [
      {
        timestamp: '2024-05-01T12:34:56.789Z',
        epc: '80',
        value: { string: 'on', EDT: 'MzA=' },
        origin: 'set' as const,
        settable: true,
      },
      {
        timestamp: '2024-05-01T12:35:10.123Z',
        epc: 'B0',
        value: { number: 24, EDT: 'Eg==' },
        origin: 'notification' as const,
        settable: true,
      },
    ];

    const mockSendMessage = vi.fn().mockResolvedValue({
      entries: mockEntries,
    });
    mockConnection.sendMessage = mockSendMessage;

    const { result } = renderHook(() =>
      useDeviceHistory({
        connection: mockConnection,
        target: '192.168.1.10 0130:1',
        limit: 50,
        settableOnly: true,
      })
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.entries).toEqual(mockEntries);
    expect(result.current.error).toBeNull();
    expect(mockSendMessage).toHaveBeenCalledWith({
      type: 'get_device_history',
      payload: {
        target: '192.168.1.10 0130:1',
        limit: 50,
        settableOnly: true,
      },
      requestId: expect.any(String),
    });
  });

  it('should handle empty history', async () => {
    const mockSendMessage = vi.fn().mockResolvedValue({
      entries: [],
    });
    mockConnection.sendMessage = mockSendMessage;

    const { result } = renderHook(() =>
      useDeviceHistory({
        connection: mockConnection,
        target: '192.168.1.10 0130:1',
      })
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.entries).toEqual([]);
    expect(result.current.error).toBeNull();
  });

  it('should handle errors', async () => {
    const mockError = new Error('Failed to fetch history');
    const mockSendMessage = vi.fn().mockRejectedValue(mockError);
    mockConnection.sendMessage = mockSendMessage;

    const { result } = renderHook(() =>
      useDeviceHistory({
        connection: mockConnection,
        target: '192.168.1.10 0130:1',
      })
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.entries).toEqual([]);
    expect(result.current.error).toBe(mockError);
  });

  it('should refetch history when refetch is called', async () => {
    const mockEntries = [
      {
        timestamp: '2024-05-01T12:34:56.789Z',
        epc: '80',
        value: { string: 'on', EDT: 'MzA=' },
        origin: 'set' as const,
        settable: true,
      },
    ];

    const mockSendMessage = vi.fn().mockResolvedValue({
      entries: mockEntries,
    });
    mockConnection.sendMessage = mockSendMessage;

    const { result } = renderHook(() =>
      useDeviceHistory({
        connection: mockConnection,
        target: '192.168.1.10 0130:1',
      })
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    // Clear mock call history
    mockSendMessage.mockClear();

    // Call refetch
    await act(async () => {
      result.current.refetch();
    });

    await waitFor(() => {
      expect(mockSendMessage).toHaveBeenCalledTimes(1);
    });
  });

  it('should refetch when settableOnly changes', async () => {
    const mockSendMessage = vi.fn().mockResolvedValue({
      entries: [],
    });
    mockConnection.sendMessage = mockSendMessage;

    const { result, rerender } = renderHook(
      ({ settableOnly }) =>
        useDeviceHistory({
          connection: mockConnection,
          target: '192.168.1.10 0130:1',
          settableOnly,
        }),
      {
        initialProps: { settableOnly: true },
      }
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(mockSendMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        payload: expect.objectContaining({
          settableOnly: true,
        }),
      })
    );

    // Change settableOnly - should trigger refetch automatically
    rerender({ settableOnly: false });

    await waitFor(() => {
      expect(mockSendMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          payload: expect.objectContaining({
            settableOnly: false,
          }),
        })
      );
    });

    await waitFor(() => {
      expect(mockSendMessage).toHaveBeenCalledTimes(2);
    });
  });

  it('should use updated settableOnly value when refetch is called', async () => {
    const mockSendMessage = vi.fn().mockResolvedValue({
      entries: [],
    });
    mockConnection.sendMessage = mockSendMessage;

    const { result, rerender } = renderHook(
      ({ settableOnly }) =>
        useDeviceHistory({
          connection: mockConnection,
          target: '192.168.1.10 0130:1',
          settableOnly,
        }),
      {
        initialProps: { settableOnly: true },
      }
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(mockSendMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        payload: expect.objectContaining({
          settableOnly: true,
        }),
      })
    );

    // Change settableOnly
    rerender({ settableOnly: false });

    await waitFor(() => {
      expect(mockSendMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          payload: expect.objectContaining({
            settableOnly: false,
          }),
        })
      );
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    mockSendMessage.mockClear();

    // Manually call refetch - should use the new settableOnly value
    await act(async () => {
      result.current.refetch();
    });

    await waitFor(() => {
      expect(mockSendMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          payload: expect.objectContaining({
            settableOnly: false,
          }),
        })
      );
    });
  });

  it('should pass since parameter when provided', async () => {
    const mockSendMessage = vi.fn().mockResolvedValue({
      entries: [],
    });
    mockConnection.sendMessage = mockSendMessage;

    const since = '2024-05-01T00:00:00Z';

    renderHook(() =>
      useDeviceHistory({
        connection: mockConnection,
        target: '192.168.1.10 0130:1',
        since,
      })
    );

    await waitFor(() => {
      expect(mockSendMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          payload: expect.objectContaining({
            since,
          }),
        })
      );
    });
  });

  it('should use default limit of 50 when not specified', async () => {
    const mockSendMessage = vi.fn().mockResolvedValue({
      entries: [],
    });
    mockConnection.sendMessage = mockSendMessage;

    renderHook(() =>
      useDeviceHistory({
        connection: mockConnection,
        target: '192.168.1.10 0130:1',
      })
    );

    await waitFor(() => {
      expect(mockSendMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          payload: expect.objectContaining({
            limit: 50,
          }),
        })
      );
    });
  });

  it('should NOT refetch when connection object reference changes but sendMessage stays the same', async () => {
    const mockSendMessage = vi.fn().mockResolvedValue({
      entries: [],
    });
    mockConnection.sendMessage = mockSendMessage;

    const { result, rerender } = renderHook(
      ({ connection }) =>
        useDeviceHistory({
          connection,
          target: '192.168.1.10 0130:1',
        }),
      {
        initialProps: { connection: mockConnection },
      }
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    const initialCallCount = mockSendMessage.mock.calls.length;

    // Create a new connection object with the same sendMessage function
    const newConnection = {
      ...mockConnection,
      connectedAt: new Date(), // Different value to ensure object reference is different
    };

    // Rerender with new connection object
    rerender({ connection: newConnection });

    // Wait a bit to ensure no refetch happens
    await new Promise(resolve => setTimeout(resolve, 50));

    // Should NOT have made additional calls because sendMessage is the same
    expect(mockSendMessage).toHaveBeenCalledTimes(initialCallCount);
  });
});
