import { useEffect, useRef, useCallback } from 'react';
import type { ConnectionState } from './types';

// Default timeout constants for better maintainability
const DEFAULT_DISCONNECT_DELAY_MS = 3000;
const VISIBILITY_TIMEOUT_MS = 100;
const PAGESHOW_TIMEOUT_MS = 150;
const PAGESHOW_PERSISTED_TIMEOUT_MS = 200;

export interface AutoReconnectOptions {
  connectionState: ConnectionState;
  connect: () => void;
  disconnect: () => void;
  checkConnection?: () => Promise<boolean>;
  delayMs?: number;
  autoDisconnect?: boolean;
  /**
   * Delay in milliseconds before disconnecting when page becomes hidden.
   * Helps prevent disconnection during brief mobile app switches.
   * @default 3000
   */
  disconnectDelayMs?: number;
  /**
   * Callback for diagnostic logging to help troubleshoot reconnection issues.
   * Useful for iOS Safari background/foreground transition debugging.
   */
  onDiagnosticLog?: (level: 'INFO' | 'WARN', message: string, attributes?: Record<string, unknown>) => void;
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
  autoDisconnect = true,
  disconnectDelayMs = DEFAULT_DISCONNECT_DELAY_MS,
  onDiagnosticLog
}: AutoReconnectOptions) {
  const hasReconnectedRef = useRef(false);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);
  const lastVisibilityChangeRef = useRef<string | null>(null);
  const visibilityTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const pageshowTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const disconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  // Store current values in refs to avoid stale closures
  const connectionStateRef = useRef(connectionState);
  const connectRef = useRef(connect);
  const disconnectRef = useRef(disconnect);
  const checkConnectionRef = useRef(checkConnection);
  const autoDisconnectRef = useRef(autoDisconnect);
  const disconnectDelayMsRef = useRef(disconnectDelayMs);
  const onDiagnosticLogRef = useRef(onDiagnosticLog);
  
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

  useEffect(() => {
    disconnectDelayMsRef.current = disconnectDelayMs;
  }, [disconnectDelayMs]);

  useEffect(() => {
    onDiagnosticLogRef.current = onDiagnosticLog;
  }, [onDiagnosticLog]);
  
  // Helper function to schedule delayed disconnect
  const scheduleDelayedDisconnect = useCallback(() => {
    // Clear any existing disconnect timeout
    if (disconnectTimeoutRef.current) {
      clearTimeout(disconnectTimeoutRef.current);
      disconnectTimeoutRef.current = null;
    }

    if (import.meta.env.DEV) {
      console.log(`📱 Scheduling disconnect in ${disconnectDelayMsRef.current}ms`);
    }

    disconnectTimeoutRef.current = setTimeout(() => {
      // Re-check visibility state before executing disconnect
      const isCurrentlyHidden = document.hidden;
      if (autoDisconnectRef.current && connectionStateRef.current === 'connected' && isCurrentlyHidden) {
        if (import.meta.env.DEV) {
          console.log('📱 Executing delayed disconnect (page still hidden)');
        }
        disconnectRef.current();
      } else {
        if (import.meta.env.DEV) {
          console.log('📱 Canceling delayed disconnect - page became visible', {
            autoDisconnect: autoDisconnectRef.current,
            connectionState: connectionStateRef.current,
            isHidden: isCurrentlyHidden
          });
        }
      }
      disconnectTimeoutRef.current = null;
    }, disconnectDelayMsRef.current);
  }, []);

  // Helper function to cancel delayed disconnect
  const cancelDelayedDisconnect = useCallback(() => {
    if (disconnectTimeoutRef.current) {
      if (import.meta.env.DEV) {
        console.log('📱 Canceling delayed disconnect');
      }
      clearTimeout(disconnectTimeoutRef.current);
      disconnectTimeoutRef.current = null;
    }
  }, []);

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
      } else if (connectionStateRef.current === 'connected') {
        // Check if the connection is actually alive (zombie detection)
        // Only perform zombie detection if checkConnection function is provided
        if (checkConnectionRef.current) {
          try {
            const isAlive = await checkConnectionRef.current();
            if (!isAlive) {
              // Connection is zombie, force disconnect and reconnect
              onDiagnosticLogRef.current?.('WARN', 'Zombie WebSocket connection detected - forcing reconnection', {
                component: 'AutoReconnect',
                event: 'zombie_detection',
                trigger: 'page_visibility'
              });
              disconnectRef.current();
            }
          } catch (error) {
            // If connection check fails, assume connection is dead
            if (import.meta.env.DEV) {
              console.warn('Connection health check failed:', error);
            }
            onDiagnosticLogRef.current?.('WARN', 'WebSocket connection check failed - forcing reconnection', {
              component: 'AutoReconnect',
              event: 'connection_check_failed',
              error: String(error)
            });
            disconnectRef.current();
          }
        }
        // If checkConnection is not provided, we trust the WebSocket's readyState
        // No additional action needed - the WebSocket will handle its own state
      }
    };

    const handleVisibilityChange = () => {
      const currentVisibility = document.visibilityState;
      lastVisibilityChangeRef.current = currentVisibility;

      if (import.meta.env.DEV) {
        console.log('👁️ Visibility changed:', {
          visibilityState: currentVisibility,
          hidden: document.hidden,
          connectionState: connectionStateRef.current
        });
      }

      if (document.hidden) {
        // Page became hidden - schedule delayed disconnect if auto-disconnect is enabled
        if (autoDisconnectRef.current && connectionStateRef.current === 'connected') {
          scheduleDelayedDisconnect();
        }
        // Clear any pending visibility timeout to prevent stale reconnection attempts
        if (visibilityTimeoutRef.current) {
          clearTimeout(visibilityTimeoutRef.current);
          visibilityTimeoutRef.current = null;
        }
      } else {
        // Page became visible - cancel any pending disconnect and attempt reconnection
        if (import.meta.env.DEV) {
          console.log('👁️ Page became visible - canceling disconnect and attempting reconnection');
        }

        // Notify about page visibility change (useful for iOS Safari debugging)
        onDiagnosticLogRef.current?.('INFO', 'Page returned from background', {
          component: 'AutoReconnect',
          event: 'visibility_change',
          visibilityState: currentVisibility,
          connectionState: connectionStateRef.current
        });

        cancelDelayedDisconnect();

        // Use timeout to avoid race conditions with pageshow event
        if (visibilityTimeoutRef.current) {
          clearTimeout(visibilityTimeoutRef.current);
        }
        visibilityTimeoutRef.current = setTimeout(() => {
          if (lastVisibilityChangeRef.current === 'visible') {
            if (import.meta.env.DEV) {
              console.log('👁️ Visibility timeout triggered, attempting reconnection');
            }
            attemptReconnection();
          }
          visibilityTimeoutRef.current = null;
        }, VISIBILITY_TIMEOUT_MS);
      }
    };

    const handlePageShow = (event: PageTransitionEvent) => {
      // Page was shown (includes cache restoration on iOS/Safari)
      lastVisibilityChangeRef.current = 'visible';

      if (import.meta.env.DEV) {
        console.log('📱 Page show event', { persisted: event.persisted });
      }

      // Cancel any pending disconnect since the page is now visible
      cancelDelayedDisconnect();

      // Clear any existing pageshow timeout to prevent duplicate reconnection attempts
      if (pageshowTimeoutRef.current) {
        clearTimeout(pageshowTimeoutRef.current);
        pageshowTimeoutRef.current = null;
      }

      // If page was restored from cache, force reconnection
      if (event.persisted) {
        // Force full reconnection for pages restored from cache
        // iOS Safari: bfcache (back-forward cache) may have stale WebSocket connections
        if (import.meta.env.DEV) {
          console.log('📱 Page restored from bfcache - forcing full reconnection');
        }

        // Notify about bfcache restoration
        onDiagnosticLogRef.current?.('INFO', 'Page restored from browser cache (bfcache) - forcing full reconnection', {
          component: 'AutoReconnect',
          event: 'bfcache_restore',
          connectionState: connectionStateRef.current,
          persisted: true
        });

        if (connectionStateRef.current === 'connected') {
          disconnectRef.current();
          // Wait for disconnect to complete before reconnecting
          // Add buffer to avoid race condition between disconnect and connect
          pageshowTimeoutRef.current = setTimeout(() => {
            connectRef.current();
            pageshowTimeoutRef.current = null;
          }, PAGESHOW_PERSISTED_TIMEOUT_MS + 100);
        } else {
          // Not connected, just reconnect normally
          pageshowTimeoutRef.current = setTimeout(() => {
            connectRef.current();
            pageshowTimeoutRef.current = null;
          }, PAGESHOW_PERSISTED_TIMEOUT_MS);
        }
      } else {
        // Normal page show, check connection if needed
        // Use longer delay for iOS Safari compatibility
        pageshowTimeoutRef.current = setTimeout(() => {
          attemptReconnection();
          pageshowTimeoutRef.current = null;
        }, PAGESHOW_TIMEOUT_MS);
      }
    };

    const handlePageHide = () => {
      // Page is being hidden (including cache storage)
      lastVisibilityChangeRef.current = 'hidden';
      if (import.meta.env.DEV) {
        console.log('📱 Page hide event');
      }

      // Clear any pending pageshow timeouts to prevent stale reconnection attempts
      if (pageshowTimeoutRef.current) {
        clearTimeout(pageshowTimeoutRef.current);
        pageshowTimeoutRef.current = null;
      }

      if (autoDisconnectRef.current && connectionStateRef.current === 'connected') {
        // Schedule delayed disconnect instead of immediate disconnect
        scheduleDelayedDisconnect();
      }
    };

    // Fallback handler for iOS Safari when pageshow doesn't fire reliably
    const handleWindowFocus = () => {
      // Only trigger if we haven't received a pageshow/visibility event recently
      // Use a simple timestamp check instead of relying on the visibility state object
      if (import.meta.env.DEV) {
        console.log('📱 Window focus fallback triggered');
      }

      // Cancel any pending disconnect since the window is now focused
      cancelDelayedDisconnect();

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
      // Clear all timeouts on cleanup to prevent memory leaks
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
        timeoutRef.current = null;
      }
      if (visibilityTimeoutRef.current) {
        clearTimeout(visibilityTimeoutRef.current);
        visibilityTimeoutRef.current = null;
      }
      if (pageshowTimeoutRef.current) {
        clearTimeout(pageshowTimeoutRef.current);
        pageshowTimeoutRef.current = null;
      }
      if (disconnectTimeoutRef.current) {
        clearTimeout(disconnectTimeoutRef.current);
        disconnectTimeoutRef.current = null;
      }
    };
  }, [delayMs, scheduleDelayedDisconnect, cancelDelayedDisconnect]); // Include helper functions in dependencies
}