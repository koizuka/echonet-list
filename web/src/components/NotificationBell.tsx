import { useState, useRef, useEffect } from 'react';
import { Bell, AlertCircle, AlertTriangle, Search, Info } from 'lucide-react';
import { cn } from '../libs/utils';
import { formatValue } from '../libs/formatValue';
import { Button } from './ui/button';
import type { LogEntry } from '../hooks/useLogNotifications';

interface NotificationBellProps {
  logs: LogEntry[];
  unreadCount: number;
  onMarkAllAsRead: () => void;
  onClearAll: () => void;
  connectedAt: Date | null;
  onDiscoverDevices?: () => Promise<unknown>;
}

export function NotificationBell({ 
  logs, 
  unreadCount, 
  onMarkAllAsRead, 
  onClearAll,
  connectedAt,
  onDiscoverDevices
}: NotificationBellProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [isDiscovering, setIsDiscovering] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Mark all as read when opening dropdown
  const handleToggleDropdown = () => {
    setIsOpen(!isOpen);
    if (!isOpen && unreadCount > 0) {
      onMarkAllAsRead();
    }
  };

  const hasUnreadLogs = unreadCount > 0;

  // Handle discover devices
  const handleDiscoverDevices = async () => {
    if (!onDiscoverDevices || isDiscovering) return;
    
    setIsDiscovering(true);
    try {
      await onDiscoverDevices();
    } catch (error) {
      console.error('Discover devices failed:', error);
    } finally {
      setIsDiscovering(false);
    }
  };

  return (
    <div className="relative" ref={dropdownRef}>
      {/* Bell Button */}
      <Button
        variant="ghost"
        size="sm"
        className={cn(
          "relative p-2 h-auto",
          hasUnreadLogs && "text-red-600 hover:text-red-700"
        )}
        onClick={handleToggleDropdown}
        data-testid="notification-bell-button"
      >
        <Bell className={cn(
          "h-5 w-5",
          hasUnreadLogs && "animate-pulse"
        )} />
        
        {/* Unread count badge */}
        {unreadCount > 0 && (
          <span className="absolute -top-1 -right-1 bg-red-500 text-white text-xs rounded-full h-5 w-5 flex items-center justify-center min-w-[20px]" data-testid="notification-count">
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </Button>

      {/* Dropdown */}
      {isOpen && (
        <div className="absolute right-0 top-full mt-2 w-80 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg z-50" data-testid="notification-dropdown">
          {/* Header */}
          <div className="p-3 border-b border-gray-200 dark:border-gray-700">
            <div className="flex items-center justify-between mb-2">
              <h3 className="font-semibold text-sm text-gray-900 dark:text-gray-100">Server Logs</h3>
              <div className="flex items-center gap-2">
                {/* Discover Devices Button */}
                {onDiscoverDevices && (
                  <Button
                    variant="outline"
                    size="sm"
                    className="text-xs h-7 px-2 border-blue-300 dark:border-blue-600 text-blue-700 dark:text-blue-300 hover:bg-blue-50 dark:hover:bg-blue-900"
                    onClick={handleDiscoverDevices}
                    disabled={isDiscovering}
                    title="Discover new devices on the network"
                  >
                    <Search className={cn("h-3 w-3 mr-1", isDiscovering && "animate-spin")} />
                    {isDiscovering ? 'Searching...' : 'Discover'}
                  </Button>
                )}
                <Button
                  variant="outline"
                  size="sm"
                  className={cn(
                    "text-xs h-7 px-2",
                    logs.length === 0 
                      ? "border-gray-200 dark:border-gray-700 text-gray-400 dark:text-gray-600 cursor-not-allowed"
                      : "border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700"
                  )}
                  onClick={() => {
                    onClearAll();
                    setIsOpen(false);
                  }}
                  disabled={logs.length === 0}
                >
                  Clear All
                </Button>
              </div>
            </div>
            {connectedAt && (
              <div className="text-xs text-gray-500 dark:text-gray-400">
                Connected at: {connectedAt.toLocaleString()}
              </div>
            )}
          </div>

          {/* Content */}
          <div className="max-h-64 overflow-y-auto">
            {logs.length === 0 ? (
              <div className="p-4 text-center text-gray-500 dark:text-gray-400 text-sm">
                No logs yet
              </div>
            ) : (
              <div className="divide-y divide-gray-100 dark:divide-gray-700">
                {logs.map((log) => (
                  <div
                    key={log.id}
                    className={cn(
                      "p-3 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors",
                      log.level === 'ERROR' ? "border-l-4 border-l-red-500" : 
                      log.level === 'WARN' ? "border-l-4 border-l-yellow-500" :
                      "border-l-4 border-l-blue-500"
                    )}
                  >
                    <div className="flex items-start gap-2">
                      {log.level === 'ERROR' ? (
                        <AlertCircle className="h-4 w-4 text-red-500 flex-shrink-0 mt-0.5" />
                      ) : log.level === 'WARN' ? (
                        <AlertTriangle className="h-4 w-4 text-yellow-500 flex-shrink-0 mt-0.5" />
                      ) : (
                        <Info className="h-4 w-4 text-blue-500 flex-shrink-0 mt-0.5" />
                      )}
                      
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center justify-between">
                          <span className={cn(
                            "text-xs font-medium",
                            log.level === 'ERROR' ? "text-red-700 dark:text-red-400" :
                            log.level === 'WARN' ? "text-yellow-700 dark:text-yellow-400" :
                            "text-blue-700 dark:text-blue-400"
                          )}>
                            {log.level}
                          </span>
                          <time className="text-xs text-gray-500 dark:text-gray-400">
                            {new Date(log.time).toLocaleTimeString()}
                          </time>
                        </div>
                        
                        <p className="text-sm text-gray-900 dark:text-gray-100 mt-1 break-words">
                          {log.message}
                        </p>
                        
                        {Object.keys(log.attributes).length > 0 && 
                         log.attributes.event !== 'device_online' && 
                         log.attributes.event !== 'device_offline' && (
                          <div className="mt-2 text-xs text-gray-600 dark:text-gray-300">
                            {Object.entries(log.attributes).map(([key, value]) => (
                              <div key={key} className="break-all">
                                <span className="font-medium">{key}:</span> 
                                <span className="ml-1 font-mono">{formatValue(value)}</span>
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Footer */}
          {logs.length > 0 && (
            <div className="p-2 border-t border-gray-200 dark:border-gray-700 text-center">
              <span className="text-xs text-gray-500 dark:text-gray-400">
                {logs.length} log{logs.length !== 1 ? 's' : ''} total
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  );
}