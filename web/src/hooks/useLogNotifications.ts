import { useState, useEffect, useCallback, useRef } from 'react';
import { generateLogEntryId } from '@/libs/idHelper';
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
  localErrorNotification?: LogEntry;
  resolveAlias?: (ip: string, eoj: string) => string | null;
  maxLogs?: number;
  onLogsChange?: (logs: LogEntry[], unreadCount: number) => void;
}

export function useLogNotifications({ 
  notification, 
  deviceOnlineNotification,
  deviceOfflineNotification,
  localErrorNotification,
  resolveAlias,
  maxLogs = 50,
  onLogsChange
}: LogNotificationsProps) {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  
  // Use ref to store the latest resolveAlias function to avoid useEffect dependency issues
  const resolveAliasRef = useRef(resolveAlias);
  resolveAliasRef.current = resolveAlias;

  // Helper function to format device identification for messages
  const formatDeviceIdentification = useCallback((ip: string, eoj: string, alias?: string): string => {
    if (alias) {
      return `${alias} (${ip} ${eoj})`;
    }
    return `${ip} ${eoj}`;
  }, []);

  // Helper function to create device status log entry
  const createDeviceStatusLogEntry = useCallback((
    type: 'online' | 'offline',
    notification: DeviceOnline | DeviceOffline
  ): LogEntry => {
    // Resolve alias dynamically using the current resolver
    let resolvedAlias: string | undefined;
    
    if (resolveAliasRef.current) {
      resolvedAlias = resolveAliasRef.current(
        notification.payload.ip,
        notification.payload.eoj
      ) || undefined;
    }
    
    const deviceId = formatDeviceIdentification(
      notification.payload.ip, 
      notification.payload.eoj, 
      resolvedAlias
    );
    
    const action = type === 'online' ? 'came online' : 'went offline';
    const level = type === 'online' ? 'INFO' : 'WARN';
    
    return {
      id: generateLogEntryId(type),
      level: level as 'INFO' | 'WARN',
      message: `Device ${deviceId} ${action}`,
      time: new Date().toISOString(),
      attributes: {
        ip: notification.payload.ip,
        eoj: notification.payload.eoj,
        alias: resolvedAlias,
        event: `device_${type}`
      },
      isRead: false
    };
  }, [formatDeviceIdentification]);

  // Helper function to add log entry to the list
  const addLogEntry = useCallback((newLog: LogEntry) => {
    setLogs(prev => {
      const updated = [newLog, ...prev];
      return updated.slice(0, maxLogs);
    });
  }, [maxLogs]);

  // Add new log entry when notification is received
  useEffect(() => {
    if (!notification) return;

    const newLog: LogEntry = {
      id: generateLogEntryId('log'),
      ...notification.payload,
      isRead: false
    };

    addLogEntry(newLog);
  }, [notification, addLogEntry]);

  // Add device online notification
  useEffect(() => {
    if (!deviceOnlineNotification) return;

    const newLog = createDeviceStatusLogEntry('online', deviceOnlineNotification);
    addLogEntry(newLog);
  }, [deviceOnlineNotification, addLogEntry, createDeviceStatusLogEntry]);

  // Add device offline notification
  useEffect(() => {
    if (!deviceOfflineNotification) return;

    const newLog = createDeviceStatusLogEntry('offline', deviceOfflineNotification);
    addLogEntry(newLog);
  }, [deviceOfflineNotification, addLogEntry, createDeviceStatusLogEntry]);

  // Handle local error notifications
  useEffect(() => {
    if (localErrorNotification) {
      addLogEntry(localErrorNotification);
    }
  }, [localErrorNotification, addLogEntry]);

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