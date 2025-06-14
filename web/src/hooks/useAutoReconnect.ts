import { useEffect, useRef } from 'react';
import type { ConnectionState } from './types';

export interface AutoReconnectOptions {
  connectionState: ConnectionState;
  connect: () => void;
  delayMs?: number;
}

/**
 * Hook that automatically attempts to reconnect when the page/browser becomes active
 * and the connection is in a disconnected state
 */
export function useAutoReconnect({ 
  connectionState, 
  connect, 
  delayMs = 2000 
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
      if (!document.hidden) {
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
  }, [connectionState, connect, delayMs]);
}