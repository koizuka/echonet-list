import { useEffect, useRef } from 'react';
import type { ConnectionState } from './types';

export interface AutoReconnectOptions {
  connectionState: ConnectionState;
  connect: () => void;
  disconnect: () => void;
  setConnectionState: (state: ConnectionState) => void;
  autoDisconnect?: boolean;
}

/**
 * Hook that automatically manages WebSocket connections based on page visibility:
 * - Disconnects when the page becomes hidden (if autoDisconnect is enabled)
 * - Triggers reconnection when the page becomes visible and the connection is disconnected
 */
export function useAutoReconnect({ 
  connectionState, 
  connect, 
  disconnect,
  setConnectionState,
  autoDisconnect = true
}: AutoReconnectOptions) {
  const hasReconnectedRef = useRef(false);
  const debounceTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  
  // Store current values in refs to avoid stale closures
  const connectionStateRef = useRef(connectionState);
  const connectRef = useRef(connect);
  const disconnectRef = useRef(disconnect);
  const setConnectionStateRef = useRef(setConnectionState);
  const autoDisconnectRef = useRef(autoDisconnect);
  
  // Update refs when values change
  useEffect(() => {
    connectionStateRef.current = connectionState;
    // Reset reconnection flag ONLY when successfully connected
    // This prevents automatic retry loops on connection failure
    if (connectionState === 'connected') {
      hasReconnectedRef.current = false;
    }
  }, [connectionState]);
  
  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);
  
  useEffect(() => {
    disconnectRef.current = disconnect;
  }, [disconnect]);
  
  useEffect(() => {
    setConnectionStateRef.current = setConnectionState;
  }, [setConnectionState]);
  
  useEffect(() => {
    autoDisconnectRef.current = autoDisconnect;
  }, [autoDisconnect]);

  // Handle reconnection when state changes to 'reconnecting'
  useEffect(() => {
    if (connectionState === 'reconnecting' && !hasReconnectedRef.current) {
      hasReconnectedRef.current = true;
      connectRef.current();
      // Note: フラグは接続成功時（connected状態）のみリセットされる
      // 接続失敗時の自動再試行は行わず、明示的なユーザーアクションを待つ
    }
  }, [connectionState]);
  
  // Main effect - only runs once on mount
  useEffect(() => {
    const triggerReconnectionDebounced = () => {
      // Clear any pending debounced reconnection
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current);
      }
      
      // Debounce multiple rapid events (like simultaneous visibilitychange + focus)
      debounceTimeoutRef.current = setTimeout(() => {
        // Prevent triggering reconnection if already reconnecting or connected
        if (connectionStateRef.current === 'disconnected' || connectionStateRef.current === 'error') {
          // Additional check: don't trigger if we're already in a reconnection attempt
          if (!hasReconnectedRef.current) {
            setConnectionStateRef.current('reconnecting');
          }
        }
      }, 100); // 100ms debounce to handle rapid events
    };

    const handleVisibilityChange = () => {
      if (document.hidden) {
        // Clear any pending reconnection when hiding
        if (debounceTimeoutRef.current) {
          clearTimeout(debounceTimeoutRef.current);
          debounceTimeoutRef.current = null;
        }
        // Page became hidden - disconnect if auto-disconnect is enabled
        if (autoDisconnectRef.current && connectionStateRef.current === 'connected') {
          disconnectRef.current();
        }
      } else {
        // Page became visible - trigger debounced reconnection
        triggerReconnectionDebounced();
      }
    };

    const handleFocus = () => {
      // Window became focused - trigger debounced reconnection
      triggerReconnectionDebounced();
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    window.addEventListener('focus', handleFocus);
    
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('focus', handleFocus);
      // Clear debounce timeout on cleanup to prevent memory leaks
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current);
      }
    };
  }, []); // No dependencies - event handlers use refs
}