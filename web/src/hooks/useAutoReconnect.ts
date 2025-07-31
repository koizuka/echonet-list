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
 * Hook that automatically manages WebSocket connections based on page visibility and focus:
 * - Disconnects when the page becomes hidden (if autoDisconnect is enabled)
 * - Reconnects when the page becomes visible and the connection is disconnected
 * - Reconnects when the window receives focus and the connection is disconnected
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
  const lastEventTimeRef = useRef<number>(0);
  const eventDedupeDelayMs = 100; // 100ms window to dedupe rapid events
  
  // Store current values in refs to avoid stale closures
  const connectionStateRef = useRef(connectionState);
  const connectRef = useRef(connect);
  const disconnectRef = useRef(disconnect);
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
    autoDisconnectRef.current = autoDisconnect;
  }, [autoDisconnect]);
  
  // Main effect - only runs once on mount
  useEffect(() => {
    const attemptReconnection = () => {
      const currentTime = Date.now();
      
      // Check if this event is too close to the previous one (mobile deduplication)
      if (currentTime - lastEventTimeRef.current < eventDedupeDelayMs) {
        return; // Skip this event as it's likely a duplicate
      }
      
      lastEventTimeRef.current = currentTime;
      
      // Only attempt if disconnected and not already attempting reconnection
      if (connectionStateRef.current === 'disconnected' && !hasReconnectedRef.current) {
        hasReconnectedRef.current = true;
        connectRef.current();
        // Reset flag after a delay to allow for future reconnection attempts
        // but only if we're still disconnected
        if (timeoutRef.current) {
          clearTimeout(timeoutRef.current);
        }
        timeoutRef.current = setTimeout(() => {
          // Only reset the flag if we're still disconnected
          // This prevents reconnection loops when connection succeeds
          if (connectionStateRef.current === 'disconnected') {
            hasReconnectedRef.current = false;
          }
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

    const handleFocus = () => {
      // Window received focus - attempt reconnection if disconnected
      attemptReconnection();
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
  }, [delayMs]); // Only depend on delayMs, not connection state or functions
}