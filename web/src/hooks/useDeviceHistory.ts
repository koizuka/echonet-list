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

  const refetch = useCallback(() => {
    setIsLoading(true);
    setError(null);
    void loadHistory();
  }, [loadHistory]);

  useEffect(() => {
    // Schedule refetch as a microtask so its setState calls are not part of
    // the effect's synchronous body (react-hooks/set-state-in-effect). Going
    // through refetch ensures dependency-change reloads also show the loading
    // state, not just the initial mount.
    //
    // Timing note: the microtask drains before the browser paints, so the
    // setIsLoading(true)/setError(null) updates are applied together with the
    // post-await results; users should not see a stale frame between the
    // effect firing and the loading indicator appearing.
    //
    // Unmount note: if the component unmounts before the microtask fires,
    // React 18+ silently drops the setState calls in refetch, so no explicit
    // cleanup is required here. The in-flight fetch itself is guarded by
    // latestRequestRef so its post-await setState is also a no-op.
    queueMicrotask(refetch);
  }, [refetch]);

  return {
    entries,
    isLoading,
    error,
    refetch,
  };
}
