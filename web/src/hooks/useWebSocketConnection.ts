import { useEffect, useRef, useCallback, useState } from 'react';
import type { 
  ServerMessage, 
  ClientMessage, 
  CommandResult, 
  ConnectionState,
  ErrorInfo 
} from './types';

export type WebSocketConnectionOptions = {
  url: string;
  reconnectAttempts?: number;
  reconnectDelay?: number;
  maxReconnectDelay?: number;
  onMessage?: (message: ServerMessage) => void;
  onConnectionStateChange?: (state: ConnectionState) => void;
  onError?: (error: ErrorInfo) => void;
};

export type WebSocketConnection = {
  connectionState: ConnectionState;
  sendMessage: <T extends ClientMessage>(message: T) => Promise<unknown>;
  connect: () => void;
  disconnect: () => void;
  error: ErrorInfo | null;
};

export function useWebSocketConnection(options: WebSocketConnectionOptions): WebSocketConnection {
  const [connectionState, setConnectionState] = useState<ConnectionState>('disconnected');
  const [error, setError] = useState<ErrorInfo | null>(null);
  
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
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

  const updateConnectionState = useCallback((state: ConnectionState) => {
    setConnectionState(state);
    options.onConnectionStateChange?.(state);
  }, [options]);

  const updateError = useCallback((errorInfo: ErrorInfo | null) => {
    setError(errorInfo);
    if (errorInfo) {
      options.onError?.(errorInfo);
    }
  }, [options]);

  const cleanup = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    
    // Reject all pending requests
    pendingRequestsRef.current.forEach(({ reject, timeout }) => {
      clearTimeout(timeout);
      reject(new Error('Connection closed'));
    });
    pendingRequestsRef.current.clear();
    
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, []);

  const connectRef = useRef<(() => void) | null>(null);

  const scheduleReconnect = useCallback(() => {
    if (reconnectAttemptsRef.current >= maxReconnectAttempts) {
      updateError({
        code: 'MAX_RECONNECT_ATTEMPTS_REACHED',
        message: `Failed to reconnect after ${maxReconnectAttempts} attempts`
      });
      updateConnectionState('error');
      return;
    }

    const delay = Math.min(
      baseReconnectDelay * Math.pow(2, reconnectAttemptsRef.current),
      maxReconnectDelay
    );

    reconnectTimeoutRef.current = setTimeout(() => {
      reconnectAttemptsRef.current++;
      connectRef.current?.();
    }, delay);
  }, [maxReconnectAttempts, baseReconnectDelay, maxReconnectDelay, updateError, updateConnectionState]);

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
        options.onMessage?.(message as ServerMessage);
      }
    } catch (error) {
      console.error('Failed to parse WebSocket message:', error);
      updateError({
        code: 'MESSAGE_PARSE_ERROR',
        message: 'Failed to parse received message'
      });
    }
  }, [options, updateError]);

  const connect = useCallback(() => {
    cleanup();
    
    console.log('🔄 WebSocket接続を開始:', options.url);
    updateConnectionState('connecting');
    updateError(null);
    
    try {
      const ws = new WebSocket(options.url);
      wsRef.current = ws;
      
      ws.onopen = () => {
        console.log('WebSocket connected');
        reconnectAttemptsRef.current = 0;
        updateConnectionState('connected');
        updateError(null);
      };
      
      ws.onmessage = handleMessage;
      
      ws.onerror = (event) => {
        console.error('WebSocket error:', event);
        updateError({
          code: 'WEBSOCKET_ERROR',
          message: `WebSocket connection error: ${event.type}`
        });
      };
      
      ws.onclose = (event) => {
        console.log('WebSocket disconnected:', {
          code: event.code,
          reason: event.reason,
          wasClean: event.wasClean
        });
        updateConnectionState('disconnected');
        
        // More specific error handling
        if (event.code === 1006) {
          updateError({
            code: 'CONNECTION_FAILED',
            message: 'Connection failed - possibly due to SSL certificate issues or server unavailable'
          });
        } else if (event.code === 1005) {
          updateError({
            code: 'CONNECTION_FAILED',
            message: 'No status received - server rejected connection'
          });
        }
        
        // Don't reconnect for certain error codes that indicate permanent failures
        const permanentFailureCodes = [1005, 1002, 1003, 1007, 1008, 1011];
        const shouldReconnect = event.code !== 1000 && 
                              !permanentFailureCodes.includes(event.code) && 
                              reconnectAttemptsRef.current < maxReconnectAttempts;
        
        if (shouldReconnect) {
          console.log('❌ 再接続条件をチェック:', {
            currentAttempts: reconnectAttemptsRef.current,
            maxAttempts: maxReconnectAttempts,
            willReconnect: reconnectAttemptsRef.current < maxReconnectAttempts
          });
          // Unexpected disconnection, schedule reconnect
          scheduleReconnect();
        } else {
          console.log('🛑 再接続しません:', {
            code: event.code,
            currentAttempts: reconnectAttemptsRef.current,
            maxAttempts: maxReconnectAttempts
          });
        }
      };
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
      updateError({
        code: 'CONNECTION_FAILED',
        message: 'Failed to create WebSocket connection'
      });
      updateConnectionState('error');
    }
  }, [options.url, handleMessage, cleanup, updateConnectionState, updateError, scheduleReconnect, maxReconnectAttempts]);

  // Assign connect function to ref for use in scheduleReconnect
  connectRef.current = connect;

  const disconnect = useCallback(() => {
    cleanup();
    updateConnectionState('disconnected');
    updateError(null);
  }, [cleanup, updateConnectionState, updateError]);

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
    console.log('🚀 useEffect実行 - 接続開始');
    // 初回接続時は再接続カウンターをリセット
    reconnectAttemptsRef.current = 0;
    connect();
    
    return () => {
      console.log('🔄 useEffect cleanup');
      cleanup();
    };
  }, [options.url]); // connectとcleanupを依存から除外

  return {
    connectionState,
    sendMessage,
    connect,
    disconnect,
    error,
  };
}