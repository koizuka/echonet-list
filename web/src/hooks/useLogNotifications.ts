import { useState, useEffect, useCallback } from 'react';
import type { LogNotification, DeviceOnline, DeviceOffline } from './types';

export interface LogEntry {
  id: string;
  level: 'ERROR' | 'WARN' | 'INFO';
  message: string;
  time: string;
  attributes: Record<string, unknown>;
  isRead: boolean;
}

interface LogNotificationsProps {
  notification?: LogNotification;
  deviceOnlineNotification?: DeviceOnline;
  deviceOfflineNotification?: DeviceOffline;
  maxLogs?: number;
  onLogsChange?: (logs: LogEntry[], unreadCount: number) => void;
}

export function useLogNotifications({ 
  notification, 
  deviceOnlineNotification,
  deviceOfflineNotification,
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

  // Add device online notification
  useEffect(() => {
    if (!deviceOnlineNotification) return;

    const newLog: LogEntry = {
      id: `online-${Date.now()}-${Math.random()}`,
      level: 'INFO',
      message: `Device ${deviceOnlineNotification.payload.ip} ${deviceOnlineNotification.payload.eoj} came online`,
      time: new Date().toISOString(),
      attributes: {
        ip: deviceOnlineNotification.payload.ip,
        eoj: deviceOnlineNotification.payload.eoj,
        event: 'device_online'
      },
      isRead: false
    };

    setLogs(prev => {
      const updated = [newLog, ...prev];
      return updated.slice(0, maxLogs);
    });
  }, [deviceOnlineNotification, maxLogs]);

  // Add device offline notification
  useEffect(() => {
    if (!deviceOfflineNotification) return;

    const newLog: LogEntry = {
      id: `offline-${Date.now()}-${Math.random()}`,
      level: 'WARN',
      message: `Device ${deviceOfflineNotification.payload.ip} ${deviceOfflineNotification.payload.eoj} went offline`,
      time: new Date().toISOString(),
      attributes: {
        ip: deviceOfflineNotification.payload.ip,
        eoj: deviceOfflineNotification.payload.eoj,
        event: 'device_offline'
      },
      isRead: false
    };

    setLogs(prev => {
      const updated = [newLog, ...prev];
      return updated.slice(0, maxLogs);
    });
  }, [deviceOfflineNotification, maxLogs]);

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