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
        return 'bg-teal-500 text-white shadow-sm shadow-teal-500/30 dark:bg-teal-600 dark:text-teal-50';
      case 'connecting':
        return 'bg-amber-500 text-white dark:bg-amber-600 dark:text-amber-50';
      case 'disconnected':
        return 'bg-slate-500 text-white dark:bg-slate-700 dark:text-slate-200';
      case 'error':
        return 'bg-red-500 text-white shadow-sm shadow-red-500/30 dark:bg-red-600 dark:text-red-50';
      default:
        return 'bg-slate-500 text-white dark:bg-slate-700 dark:text-slate-200';
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