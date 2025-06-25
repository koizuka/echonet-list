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
  
  useEffect(() => {
    const attemptReconnection = () => {
      if (connectionState === 'disconnected' && !hasReconnectedRef.current) {
        hasReconnectedRef.current = true;
        connect();
        // Reset flag after a delay to allow for future reconnection attempts
        setTimeout(() => {
          hasReconnectedRef.current = false;
        }, delayMs);
      }
    };

    const handleVisibilityChange = () => {
      if (document.hidden) {
        // Page became hidden - disconnect if auto-disconnect is enabled
        if (autoDisconnect && connectionState === 'connected') {
          disconnect();
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
    };
  }, [connectionState, connect, disconnect, delayMs, autoDisconnect]);
}