import { useEffect, useRef } from 'react';
import type { ConnectionState } from './types';

export interface AutoReconnectOptions {
  connectionState: ConnectionState;
  connect: () => void;
  disconnect: () => void;
  checkConnection?: () => Promise<boolean>;
  delayMs?: number;
  autoDisconnect?: boolean;
}

/**
 * Hook that automatically manages WebSocket connections based on page lifecycle events:
 * - Disconnects when the page becomes hidden (if autoDisconnect is enabled) 
 * - Reconnects when the page is shown and the connection is disconnected
 * 
 * Uses asymmetric event handling to avoid mobile browser reconnection loops:
 * - pageshow/visibility visible events trigger connection attempts
 * - visibilitychange hidden/pagehide events trigger disconnection
 * - Includes zombie connection detection for mobile browser background/foreground transitions
 */
export function useAutoReconnect({ 
  connectionState, 
  connect, 
  disconnect,
  checkConnection,
  delayMs = 2000,
  autoDisconnect = true
}: AutoReconnectOptions) {
  const hasReconnectedRef = useRef(false);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);
  const lastVisibilityChangeRef = useRef<string | null>(null);
  
  // Store current values in refs to avoid stale closures
  const connectionStateRef = useRef(connectionState);
  const connectRef = useRef(connect);
  const disconnectRef = useRef(disconnect);
  const checkConnectionRef = useRef(checkConnection);
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
    checkConnectionRef.current = checkConnection;
  }, [checkConnection]);
  
  useEffect(() => {
    autoDisconnectRef.current = autoDisconnect;
  }, [autoDisconnect]);
  
  // Main effect - only runs once on mount
  useEffect(() => {
    const attemptReconnection = async () => {
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
      } else if (connectionStateRef.current === 'connected' && checkConnectionRef.current) {
        // Check if the connection is actually alive (zombie detection)
        const isAlive = await checkConnectionRef.current();
        if (!isAlive) {
          // Connection is zombie, force disconnect and reconnect
          disconnectRef.current();
        }
      }
    };

    const handleVisibilityChange = () => {
      const currentVisibility = document.visibilityState;
      lastVisibilityChangeRef.current = currentVisibility;
      
      if (document.hidden) {
        // Page became hidden - disconnect if auto-disconnect is enabled
        if (autoDisconnectRef.current && connectionStateRef.current === 'connected') {
          disconnectRef.current();
        }
      } else {
        // Page became visible - attempt reconnection after a short delay
        // Use timeout to avoid race conditions with pageshow event
        setTimeout(() => {
          if (lastVisibilityChangeRef.current === 'visible') {
            attemptReconnection();
          }
        }, 100);
      }
    };

    const handlePageShow = (event: PageTransitionEvent) => {
      // Page was shown (includes cache restoration on iOS/Safari)
      lastVisibilityChangeRef.current = 'visible';
      
      if (import.meta.env.DEV) {
        console.log('📱 Page show event', { persisted: event.persisted });
      }
      
      // If page was restored from cache, force reconnection
      if (event.persisted) {
        // Force full reconnection for pages restored from cache
        if (connectionStateRef.current === 'connected') {
          disconnectRef.current();
        }
        setTimeout(() => attemptReconnection(), 200);
      } else {
        // Normal page show, check connection if needed
        // Use longer delay for iOS Safari compatibility
        setTimeout(() => attemptReconnection(), 150);
      }
    };

    const handlePageHide = () => {
      // Page is being hidden (including cache storage)
      lastVisibilityChangeRef.current = 'hidden';
      if (import.meta.env.DEV) {
        console.log('📱 Page hide event');
      }
      if (autoDisconnectRef.current && connectionStateRef.current === 'connected') {
        // Cleanly close connection before page is hidden
        disconnectRef.current();
      }
    };

    // Fallback handler for iOS Safari when pageshow doesn't fire reliably
    const handleWindowFocus = () => {
      // Only trigger if we haven't received a pageshow/visibility event recently
      // Use a simple timestamp check instead of relying on the visibility state object
      if (import.meta.env.DEV) {
        console.log('📱 Window focus fallback triggered');
      }
      setTimeout(() => attemptReconnection(), 100);
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    window.addEventListener('pageshow', handlePageShow);
    window.addEventListener('pagehide', handlePageHide);
    // Add focus as fallback for iOS Safari
    window.addEventListener('focus', handleWindowFocus);
    
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('pageshow', handlePageShow);
      window.removeEventListener('pagehide', handlePageHide);
      window.removeEventListener('focus', handleWindowFocus);
      // Clear timeout on cleanup to prevent memory leaks
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, [delayMs]); // Only depend on delayMs, not connection state or functions
}