import { useEffect, useRef } from 'react';
import type { ConnectionState } from './types';

export interface AutoReconnectOptions {
  connectionState: ConnectionState;
  connect: () => void;
  disconnect: () => void;
  delayMs?: number;
  autoDisconnect?: boolean;
}

/**
 * Hook that automatically manages WebSocket connections based on page visibility:
 * - Disconnects when the page becomes hidden (if autoDisconnect is enabled)
 * - Reconnects when the page becomes visible and the connection is disconnected
 */
export function useAutoReconnect({ 
  connectionState, 
  connect, 
  disconnect,
  delayMs = 2000,
  autoDisconnect = true
}: AutoReconnectOptions) {
  const hasReconnectedRef = useRef(false);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);
  
  // Store current values in refs to avoid stale closures
  const connectionStateRef = useRef(connectionState);
  const connectRef = useRef(connect);
  const disconnectRef = useRef(disconnect);
  const autoDisconnectRef = useRef(autoDisconnect);
  
  // Update refs when values change
  useEffect(() => {
    connectionStateRef.current = connectionState;
  }, [connectionState]);
  
  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);
  
  useEffect(() => {
    disconnectRef.current = disconnect;
  }, [disconnect]);
  
  useEffect(() => {
    autoDisconnectRef.current = autoDisconnect;
  }, [autoDisconnect]);
  
  // Main effect - only runs once on mount
  useEffect(() => {
    const attemptReconnection = () => {
      if (connectionStateRef.current === 'disconnected' && !hasReconnectedRef.current) {
        hasReconnectedRef.current = true;
        connectRef.current();
        // Reset flag after a delay to allow for future reconnection attempts
        if (timeoutRef.current) {
          clearTimeout(timeoutRef.current);
        }
        timeoutRef.current = setTimeout(() => {
          hasReconnectedRef.current = false;
        }, delayMs);
      }
    };

    const handleVisibilityChange = () => {
      if (document.hidden) {
        // Page became hidden - disconnect if auto-disconnect is enabled
        if (autoDisconnectRef.current && connectionStateRef.current === 'connected') {
          disconnectRef.current();
        }
      } else {
        // Page became visible - attempt reconnection
        attemptReconnection();
      }
    };

    const handleWindowFocus = () => {
      attemptReconnection();
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    window.addEventListener('focus', handleWindowFocus);
    
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('focus', handleWindowFocus);
      // Clear timeout on cleanup to prevent memory leaks
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, [delayMs]); // Only depend on delayMs, not connection state or functions
}