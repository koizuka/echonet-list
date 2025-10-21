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
  const latestRequestRef = useRef<number | null>(null);

  const { sendMessage } = connection;

  const fetchHistory = useCallback(async () => {
    const requestIndex = requestIdRef.current++;
    latestRequestRef.current = requestIndex;
    setIsLoading(true);
    setError(null);

    try {
      const requestId = `history-${target}-${requestIndex}`;

      const payload: {
        target: string;
        limit: number;
        since?: string;
        settableOnly: boolean;
      } = {
        target,
        limit,
        settableOnly,
      };

      if (since) {
        payload.since = since;
      }

      const response = await sendMessage({
        type: 'get_device_history',
        payload,
        requestId,
      }) as { entries: DeviceHistoryEntry[] };

      if (latestRequestRef.current === requestIndex) {
        setEntries(response.entries || []);
      }
    } catch (err) {
      if (latestRequestRef.current === requestIndex) {
        setError(err instanceof Error ? err : new Error(String(err)));
        setEntries([]);
      }
    } finally {
      if (latestRequestRef.current === requestIndex) {
        setIsLoading(false);
      }
    }
  }, [sendMessage, target, limit, since, settableOnly]);

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
