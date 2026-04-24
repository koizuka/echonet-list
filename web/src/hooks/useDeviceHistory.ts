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

  // loadHistory performs the network fetch and only applies state updates
  // *after* the await, so it is safe to call from within an effect (the
  // synchronous portion touches no React state).
  const loadHistory = useCallback(async () => {
    const requestIndex = requestIdRef.current++;
    latestRequestRef.current = requestIndex;

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
        setError(null);
        setIsLoading(false);
      }
    } catch (err) {
      if (latestRequestRef.current === requestIndex) {
        setError(err instanceof Error ? err : new Error(String(err)));
        setEntries([]);
        setIsLoading(false);
      }
    }
  }, [sendMessage, target, limit, since, settableOnly]);

  useEffect(() => {
    // Schedule as a microtask so any setState inside loadHistory is not part
    // of the effect's synchronous body (react-hooks/set-state-in-effect).
    queueMicrotask(() => {
      void loadHistory();
    });
  }, [loadHistory]);

  const refetch = useCallback(() => {
    setIsLoading(true);
    setError(null);
    void loadHistory();
  }, [loadHistory]);

  return {
    entries,
    isLoading,
    error,
    refetch,
  };
}
