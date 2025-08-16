import { RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import type { Device } from '@/hooks/types';

interface RefreshAllOfflineButtonProps {
  offlineDevices: Device[];
  onRefreshAll: () => void;
  isUpdating: boolean;
  isConnected: boolean;
}

export function RefreshAllOfflineButton({
  offlineDevices,
  onRefreshAll,
  isUpdating,
  isConnected
}: RefreshAllOfflineButtonProps) {
  if (offlineDevices.length === 0) {
    return null;
  }

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={onRefreshAll}
      disabled={isUpdating || !isConnected}
      className="h-7 sm:h-8 px-2 sm:px-3 flex items-center gap-1"
      title={isUpdating ? "Updating all offline devices..." : `Refresh ${offlineDevices.length} offline device(s)`}
      data-testid="refresh-all-offline-button"
    >
      <RefreshCw 
        className={`h-3 w-3 ${isUpdating ? 'animate-spin' : ''}`}
        data-testid="refresh-spinner"
      />
      <span className="hidden sm:inline text-xs">Refresh All Offline</span>
      <span className="text-xs font-medium">({offlineDevices.length})</span>
    </Button>
  );
}