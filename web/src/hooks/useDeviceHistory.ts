import { useState, useEffect, useCallback, useRef } from 'react';
import type { WebSocketConnection } from './useWebSocketConnection';
import type { DeviceHistoryEntry } from './types';

export type UseDeviceHistoryOptions = {
  connection: WebSocketConnection;
  target: string;
  limit?: number;
  since?: string;
  settableOnly?: boolean;
};

export type UseDeviceHistoryResult = {
  entries: DeviceHistoryEntry[];
  isLoading: boolean;
  error: Error | null;
  refetch: () => void;
};

export function useDeviceHistory({
  connection,
  target,
  limit = 50,
  since,
  settableOnly = true,
}: UseDeviceHistoryOptions): UseDeviceHistoryResult {
  const [entries, setEntries] = useState<DeviceHistoryEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const requestIdRef = useRef(0);

  // Store current parameters in refs to avoid recreating fetchHistory unnecessarily
  const settableOnlyRef = useRef(settableOnly);
  const limitRef = useRef(limit);
  const sinceRef = useRef(since);

  // Update refs when parameters change
  settableOnlyRef.current = settableOnly;
  limitRef.current = limit;
  sinceRef.current = since;

  // Extract sendMessage to narrow dependency and avoid refetch on connection object reference changes
  const sendMessage = connection.sendMessage;

  const fetchHistory = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      const requestId = `history-${target}-${requestIdRef.current++}`;

      const payload: {
        target: string;
        limit: number;
        since?: string;
        settableOnly: boolean;
      } = {
        target,
        limit: limitRef.current,
        settableOnly: settableOnlyRef.current,
      };

      if (sinceRef.current) {
        payload.since = sinceRef.current;
      }

      const response = await sendMessage({
        type: 'get_device_history',
        payload,
        requestId,
      }) as { entries: DeviceHistoryEntry[] };

      setEntries(response.entries || []);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
      setEntries([]);
    } finally {
      setIsLoading(false);
    }
  }, [sendMessage, target]);

  useEffect(() => {
    fetchHistory();
  }, [fetchHistory]);

  const refetch = useCallback(() => {
    fetchHistory();
  }, [fetchHistory]);

  return {
    entries,
    isLoading,
    error,
    refetch,
  };
}
