import { useEffect, useRef, useCallback, useState } from 'react';
import type { 
  ServerMessage, 
  ClientMessage, 
  CommandResult, 
  ConnectionState
} from './types';

export type WebSocketConnectionOptions = {
  url: string;
  reconnectAttempts?: number;
  reconnectDelay?: number;
  maxReconnectDelay?: number;
  heartbeatInterval?: number;
  onMessage?: (message: ServerMessage) => void;
  onConnectionStateChange?: (state: ConnectionState) => void;
  onWebSocketConnected?: () => void;
};

export type WebSocketConnection = {
  connectionState: ConnectionState;
  sendMessage: <T extends ClientMessage>(message: T) => Promise<unknown>;
  connect: () => void;
  disconnect: () => void;
  connectedAt: Date | null;
  checkConnection: () => Promise<boolean>;
};

export function useWebSocketConnection(options: WebSocketConnectionOptions): WebSocketConnection {
  const [connectionState, setConnectionState] = useState<ConnectionState>('disconnected');
  const [connectedAt, setConnectedAt] = useState<Date | null>(null);
  
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const connectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const heartbeatIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const heartbeatTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const requestIdCounterRef = useRef(0);
  const pendingRequestsRef = useRef<Map<string, {
    resolve: (value: unknown) => void;
    reject: (error: unknown) => void;
    timeout: ReturnType<typeof setTimeout>;
  }>>(new Map());
  
  const reconnectAttemptsRef = useRef(0);
  const maxReconnectAttempts = options.reconnectAttempts ?? 5;
  const baseReconnectDelay = options.reconnectDelay ?? 1000;
  const maxReconnectDelay = options.maxReconnectDelay ?? 30000;
  const heartbeatInterval = options.heartbeatInterval ?? 30000; // 30 seconds

  const updateConnectionState = useCallback((state: ConnectionState) => {
    setConnectionState(state);
    options.onConnectionStateChange?.(state);
  }, [options]);

  const sendLogNotification = useCallback((level: 'ERROR' | 'WARN', message: string, attributes: Record<string, unknown> = {}) => {
    const logMessage = {
      type: 'log_notification' as const,
      payload: {
        level,
        message,
        time: new Date().toISOString(),
        attributes
      }
    };
    options.onMessage?.(logMessage);
  }, [options]);

  const cleanup = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    
    if (connectTimeoutRef.current) {
      clearTimeout(connectTimeoutRef.current);
      connectTimeoutRef.current = null;
    }
    
    if (heartbeatIntervalRef.current) {
      clearInterval(heartbeatIntervalRef.current);
      heartbeatIntervalRef.current = null;
    }
    
    if (heartbeatTimeoutRef.current) {
      clearTimeout(heartbeatTimeoutRef.current);
      heartbeatTimeoutRef.current = null;
    }
    
    // Reject all pending requests
    pendingRequestsRef.current.forEach(({ reject, timeout }) => {
      clearTimeout(timeout);
      reject(new Error('Connection closed'));
    });
    pendingRequestsRef.current.clear();
    
    if (wsRef.current) {
      // React StrictMode対策：CONNECTING状態でのcloseは静かに処理
      if (wsRef.current.readyState === WebSocket.CONNECTING) {
        // エラーハンドラを無効化してからclose
        wsRef.current.onerror = null;
        wsRef.current.onclose = null;
      }
      wsRef.current.close();
      wsRef.current = null;
    }
  }, []); // cleanupは外部の状態に依存しない

  const connectRef = useRef<(() => void) | null>(null);

  const checkConnection = useCallback(async (): Promise<boolean> => {
    if (!wsRef.current) {
      return false;
    }

    // Check WebSocket readyState
    const state = wsRef.current.readyState;
    if (state !== WebSocket.OPEN) {
      if (import.meta.env.DEV) {
        console.warn('WebSocket not in OPEN state:', state);
      }
      return false;
    }

    // For mobile browsers, we consider the connection alive if WebSocket state is OPEN
    // The browser will handle the underlying TCP connection state
    return true;
  }, []);

  const startHeartbeat = useCallback(() => {
    if (heartbeatIntervalRef.current) {
      clearInterval(heartbeatIntervalRef.current);
    }

    heartbeatIntervalRef.current = setInterval(() => {
      if (typeof document !== 'undefined' && document.visibilityState === 'hidden') {
        // Skip heartbeat while page is hidden
        return;
      }

      // Check WebSocket state periodically
      if (wsRef.current && wsRef.current.readyState !== WebSocket.OPEN) {
        if (import.meta.env.DEV) {
          console.warn('💔 WebSocket state changed to:', wsRef.current.readyState);
        }
        // WebSocket state changed, let the onclose handler deal with it
      }
    }, heartbeatInterval);
  }, [heartbeatInterval]);

  const stopHeartbeat = useCallback(() => {
    if (heartbeatIntervalRef.current) {
      clearInterval(heartbeatIntervalRef.current);
      heartbeatIntervalRef.current = null;
    }
    if (heartbeatTimeoutRef.current) {
      clearTimeout(heartbeatTimeoutRef.current);
      heartbeatTimeoutRef.current = null;
    }
  }, []);


  const scheduleReconnect = useCallback(() => {
    // Don't reconnect while the page is in background to prevent mobile browser reconnection loops
    if (typeof document !== 'undefined' && document.hidden) {
      if (import.meta.env.DEV) {
        console.log('🚫 Skip reconnect - page is hidden (document.hidden)');
      }
      return;
    }

    if (reconnectAttemptsRef.current >= maxReconnectAttempts) {
      const errorMessage = `Failed to reconnect after ${maxReconnectAttempts} attempts`;
      console.error(errorMessage);
      sendLogNotification('ERROR', errorMessage, { 
        component: 'WebSocket',
        reconnectAttempts: maxReconnectAttempts 
      });
      updateConnectionState('error');
      return;
    }

    const delay = Math.min(
      baseReconnectDelay * Math.pow(2, reconnectAttemptsRef.current),
      maxReconnectDelay
    );

    // Update connection state to 'connecting' when scheduling reconnection
    updateConnectionState('connecting');

    reconnectTimeoutRef.current = setTimeout(() => {
      reconnectAttemptsRef.current++;
      connectRef.current?.();
    }, delay);
  }, [maxReconnectAttempts, baseReconnectDelay, maxReconnectDelay, updateConnectionState, sendLogNotification]);

  const handleMessage = useCallback((event: MessageEvent) => {
    try {
      const message = JSON.parse(event.data);
      
      if (message.requestId && pendingRequestsRef.current.has(message.requestId)) {
        // Handle command result
        const pending = pendingRequestsRef.current.get(message.requestId);
        if (pending) {
          clearTimeout(pending.timeout);
          pendingRequestsRef.current.delete(message.requestId);
          
          const result = message as CommandResult;
          if (result.payload.success) {
            pending.resolve(result.payload.data);
          } else {
            pending.reject(result.payload.error);
          }
        }
      } else {
        // Handle server notification
        const serverMessage = message as ServerMessage;
        
        // Handle log notifications specially
        if (serverMessage.type === 'log_notification') {
          const { level, message: logMessage, attributes } = serverMessage.payload as {
            level: string;
            message: string;
            time: string;
            attributes: Record<string, unknown>;
          };
          
          // Log to console based on level
          if (level === 'ERROR') {
            console.error(`[Server ${level}] ${logMessage}`, attributes);
          } else if (level === 'WARN') {
            console.warn(`[Server ${level}] ${logMessage}`, attributes);
          }
        }
        
        // Always pass the message to the handler
        options.onMessage?.(serverMessage);
      }
    } catch (error) {
      console.error('Failed to parse WebSocket message:', error);
    }
  }, [options]);

  const actualConnect = useCallback(() => {
    cleanup();
    
    if (import.meta.env.DEV) {
      console.log('🔄 WebSocket接続を開始:', options.url);
    }
    updateConnectionState('connecting');
    
    try {
      const ws = new WebSocket(options.url);
      wsRef.current = ws;
      
      ws.onopen = () => {
        console.log('WebSocket connected');
        reconnectAttemptsRef.current = 0;
        setConnectedAt(new Date());
        updateConnectionState('connected');
        // Start heartbeat to detect zombie connections
        startHeartbeat();
        // Call the onWebSocketConnected callback to clear WebSocket error logs
        options.onWebSocketConnected?.();
      };
      
      ws.onmessage = handleMessage;
      
      ws.onerror = (event) => {
        console.error('WebSocket error:', event);
        sendLogNotification('ERROR', `WebSocket connection error: ${event.type}`, {
          component: 'WebSocket',
          eventType: event.type
        });
      };
      
      ws.onclose = (event) => {
        console.log('WebSocket disconnected:', {
          code: event.code,
          reason: event.reason,
          wasClean: event.wasClean
        });
        // Stop heartbeat when connection closes
        stopHeartbeat();
        setConnectedAt(null);
        updateConnectionState('disconnected');
        
        // Log specific error conditions for debugging and user notification
        if (event.code === 1006) {
          const errorMessage = 'Connection failed - possibly due to SSL certificate issues or server unavailable';
          console.error(errorMessage);
          sendLogNotification('ERROR', errorMessage, {
            component: 'WebSocket',
            closeCode: event.code,
            reason: event.reason || 'No reason provided'
          });
        } else if (event.code === 1005) {
          const errorMessage = 'No status received - server rejected connection';
          console.error(errorMessage);
          sendLogNotification('ERROR', errorMessage, {
            component: 'WebSocket', 
            closeCode: event.code,
            reason: event.reason || 'No reason provided'
          });
        } else if (event.code !== 1000 && !event.wasClean) {
          // Log other unexpected disconnections
          sendLogNotification('WARN', `WebSocket disconnected unexpectedly`, {
            component: 'WebSocket',
            closeCode: event.code,
            reason: event.reason || 'No reason provided',
            wasClean: event.wasClean
          });
        }
        
        // Don't reconnect for certain error codes that indicate permanent failures
        const permanentFailureCodes = [1002, 1003, 1007, 1008, 1011];
        const shouldReconnect = event.code !== 1000 && 
                              !permanentFailureCodes.includes(event.code) && 
                              reconnectAttemptsRef.current < maxReconnectAttempts;
        
        if (shouldReconnect) {
          if (import.meta.env.DEV) {
            console.log('❌ 再接続条件をチェック:', {
              currentAttempts: reconnectAttemptsRef.current,
              maxAttempts: maxReconnectAttempts,
              willReconnect: reconnectAttemptsRef.current < maxReconnectAttempts
            });
          }
          // Unexpected disconnection, schedule reconnect
          scheduleReconnect();
        } else {
          if (import.meta.env.DEV) {
            console.log('🛑 再接続しません:', {
              code: event.code,
              currentAttempts: reconnectAttemptsRef.current,
              maxAttempts: maxReconnectAttempts
            });
          }
        }
      };
    } catch (error) {
      const errorMessage = `Failed to create WebSocket connection: ${error}`;
      console.error(errorMessage);
      sendLogNotification('ERROR', errorMessage, {
        component: 'WebSocket',
        error: String(error)
      });
      updateConnectionState('error');
    }
  }, [options, handleMessage, updateConnectionState, scheduleReconnect, maxReconnectAttempts, cleanup, sendLogNotification, startHeartbeat, stopHeartbeat]);

  // Debounced connect function to handle React StrictMode double mounting
  const connect = useCallback(() => {
    // Clear any pending connection attempt
    if (connectTimeoutRef.current) {
      clearTimeout(connectTimeoutRef.current);
      connectTimeoutRef.current = null;
    }

    // In development mode (but not test), add a small delay to handle StrictMode double mounting
    if (import.meta.env.DEV && !import.meta.env.MODE?.includes('test')) {
      connectTimeoutRef.current = setTimeout(() => {
        actualConnect();
      }, 50); // 50ms delay in dev mode
    } else {
      actualConnect();
    }
  }, [actualConnect]);

  // Assign connect function to ref for use in scheduleReconnect
  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  const disconnect = useCallback(() => {
    cleanup();
    setConnectedAt(null);
    updateConnectionState('disconnected');
  }, [cleanup, updateConnectionState]);

  const sendMessage = useCallback(<T extends ClientMessage>(message: T): Promise<unknown> => {
    return new Promise((resolve, reject) => {
      if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
        reject(new Error('WebSocket is not connected'));
        return;
      }

      const requestId = `req-${++requestIdCounterRef.current}`;
      const messageWithId = { ...message, requestId };

      // Set up timeout for the request
      const timeout = setTimeout(() => {
        pendingRequestsRef.current.delete(requestId);
        reject(new Error(`Request ${requestId} timed out`));
      }, 10000); // 10 second timeout

      pendingRequestsRef.current.set(requestId, { resolve, reject, timeout });

      try {
        wsRef.current.send(JSON.stringify(messageWithId));
      } catch (error) {
        clearTimeout(timeout);
        pendingRequestsRef.current.delete(requestId);
        reject(error);
      }
    });
  }, []);

  // Auto-connect on mount - URLが変更された場合のみ再接続
  useEffect(() => {
    // 初回接続時は再接続カウンターをリセット
    reconnectAttemptsRef.current = 0;
    connect();

    return () => {
      cleanup();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [options.url]); // URLが変更された場合のみ再接続、connectとcleanupは安定化済みだが依存に入れると無限ループの可能性

  return {
    connectionState,
    sendMessage,
    connect,
    disconnect,
    connectedAt,
    checkConnection,
  };
}