import { useState, useEffect, useCallback } from 'react';
import type { LogNotification } from './types';

export interface LogEntry {
  id: string;
  level: 'ERROR' | 'WARN';
  message: string;
  time: string;
  attributes: Record<string, unknown>;
  isRead: boolean;
}

interface LogNotificationsProps {
  notification?: LogNotification;
  maxLogs?: number;
  onLogsChange?: (logs: LogEntry[], unreadCount: number) => void;
}

export function useLogNotifications({ 
  notification, 
  maxLogs = 50,
  onLogsChange
}: LogNotificationsProps) {
  const [logs, setLogs] = useState<LogEntry[]>([]);

  // Add new log entry when notification is received
  useEffect(() => {
    if (!notification) return;

    const newLog: LogEntry = {
      id: `${Date.now()}-${Math.random()}`,
      ...notification.payload,
      isRead: false
    };

    setLogs(prev => {
      const updated = [newLog, ...prev];
      return updated.slice(0, maxLogs);
    });
  }, [notification, maxLogs]);

  // Notify parent component about logs changes
  useEffect(() => {
    const unreadCount = logs.filter(log => !log.isRead).length;
    onLogsChange?.(logs, unreadCount);
  }, [logs, onLogsChange]);


  const markAllAsRead = useCallback(() => {
    setLogs(prev => prev.map(log => ({ ...log, isRead: true })));
  }, []);

  const clearAllLogs = useCallback(() => {
    setLogs([]);
  }, []);

  const clearByCategory = useCallback((category: string) => {
    setLogs(prev => prev.filter(log => 
      log.attributes.component !== category
    ));
  }, []);

  // Return functions for parent component to use
  return {
    logs,
    markAllAsRead,
    clearAllLogs,
    clearByCategory
  } as const;
}