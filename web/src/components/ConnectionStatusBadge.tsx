import { Badge } from '@/components/ui/badge';
import { Wifi, WifiOff, Loader2, AlertCircle } from 'lucide-react';
import type { ConnectionState } from '@/hooks/types';

type ConnectionStatusBadgeProps = {
  connectionState: ConnectionState;
};

export function ConnectionStatusBadge({ connectionState }: ConnectionStatusBadgeProps) {
  const getConnectionIcon = (state: ConnectionState) => {
    const iconClass = "w-3 h-3 mr-1";
    
    switch (state) {
      case 'connected':
        return <Wifi className={iconClass} data-testid="connection-icon" />;
      case 'disconnected':
        return <WifiOff className={iconClass} data-testid="connection-icon" />;
      case 'connecting':
        return <Loader2 className={`${iconClass} animate-spin`} data-testid="connection-icon" />;
      case 'error':
        return <AlertCircle className={iconClass} data-testid="connection-icon" />;
      default:
        return <WifiOff className={iconClass} data-testid="connection-icon" />;
    }
  };

  const getConnectionColor = (state: ConnectionState) => {
    switch (state) {
      case 'connected':
        return 'bg-green-500 text-white dark:bg-green-900 dark:text-green-100';
      case 'connecting':
        return 'bg-yellow-500 text-black dark:bg-yellow-900 dark:text-yellow-100';
      case 'disconnected':
        return 'bg-gray-500 text-white dark:bg-gray-800 dark:text-gray-100';
      case 'error':
        return 'bg-red-500 text-white dark:bg-red-900 dark:text-red-100';
      default:
        return 'bg-gray-500 text-white dark:bg-gray-800 dark:text-gray-100';
    }
  };

  const getConnectionText = (state: ConnectionState) => {
    switch (state) {
      case 'connected':
        return 'Connected';
      case 'connecting':
        return 'Connecting';
      case 'disconnected':
        return 'Disconnected';
      case 'error':
        return 'Error';
      default:
        return 'Unknown';
    }
  };

  return (
    <Badge 
      variant="outline" 
      className={`${getConnectionColor(connectionState)} text-xs flex items-center`}
      data-testid="connection-status"
    >
      {getConnectionIcon(connectionState)}
      {getConnectionText(connectionState)}
    </Badge>
  );
}