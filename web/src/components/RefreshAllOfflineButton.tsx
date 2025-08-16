import { RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { getCurrentLocale } from '@/libs/languageHelper';
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

  const currentLocale = getCurrentLocale();
  const labels = {
    en: {
      buttonText: 'Refresh All Offline',
      updatingTitle: 'Updating all offline devices...',
      refreshTitle: (count: number) => `Refresh ${count} offline device${count === 1 ? '' : 's'}`
    },
    ja: {
      buttonText: 'オフライン端末を更新',
      updatingTitle: 'すべてのオフライン端末を更新中...',
      refreshTitle: (count: number) => `${count}台のオフライン端末を更新`
    }
  };

  const t = labels[currentLocale];

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={onRefreshAll}
      disabled={isUpdating || !isConnected}
      className="h-7 sm:h-8 px-2 sm:px-3 flex items-center gap-1"
      title={isUpdating ? t.updatingTitle : t.refreshTitle(offlineDevices.length)}
      data-testid="refresh-all-offline-button"
    >
      <RefreshCw 
        className={`h-3 w-3 ${isUpdating ? 'animate-spin' : ''}`}
        data-testid="refresh-spinner"
      />
      <span className="hidden sm:inline text-xs">{t.buttonText}</span>
      <span className="text-xs font-medium">({offlineDevices.length})</span>
    </Button>
  );
}