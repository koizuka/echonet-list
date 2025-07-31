import { useEffect, useRef } from 'react';
import type { ConnectionState } from './types';

export interface AutoReconnectOptions {
  connectionState: ConnectionState;
  connect: () => void;
  disconnect: () => void;
  setConnectionState: (state: ConnectionState) => void;
  delayMs?: number;
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
  delayMs = 2000,
  autoDisconnect = true
}: AutoReconnectOptions) {
  const hasReconnectedRef = useRef(false);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);
  
  // Store current values in refs to avoid stale closures
  const connectionStateRef = useRef(connectionState);
  const connectRef = useRef(connect);
  const disconnectRef = useRef(disconnect);
  const setConnectionStateRef = useRef(setConnectionState);
  const autoDisconnectRef = useRef(autoDisconnect);
  
  // Update refs when values change
  useEffect(() => {
    connectionStateRef.current = connectionState;
    // Reset reconnection flag when connected
    if (connectionState === 'connected') {
      hasReconnectedRef.current = false;
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
        timeoutRef.current = null;
      }
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
      // Reset flag after a delay to allow for future reconnection attempts
      // but only if we're still disconnected
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      timeoutRef.current = setTimeout(() => {
        // Only reset the flag if we're still disconnected or in error state
        // This prevents reconnection loops when connection succeeds
        if (connectionStateRef.current === 'disconnected' || connectionStateRef.current === 'error') {
          hasReconnectedRef.current = false;
        }
      }, delayMs);
    }
  }, [connectionState, delayMs]);
  
  // Main effect - only runs once on mount
  useEffect(() => {
    const triggerReconnection = () => {
      if (connectionStateRef.current === 'disconnected' || connectionStateRef.current === 'error') {
        setConnectionStateRef.current('reconnecting');
      }
    };

    const handleVisibilityChange = () => {
      if (document.hidden) {
        // Page became hidden - disconnect if auto-disconnect is enabled
        if (autoDisconnectRef.current && connectionStateRef.current === 'connected') {
          disconnectRef.current();
        }
      } else {
        // Page became visible - trigger reconnection
        triggerReconnection();
      }
    };

    const handleFocus = () => {
      // Window became focused - trigger reconnection (for PC browsers)
      triggerReconnection();
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    window.addEventListener('focus', handleFocus);
    
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('focus', handleFocus);
      // Clear timeout on cleanup to prevent memory leaks
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []); // No dependencies - event handlers use refs
}