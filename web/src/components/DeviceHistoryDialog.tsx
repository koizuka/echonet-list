import { useState } from 'react';
import { RefreshCw, Loader2 } from 'lucide-react';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { HexViewer } from '@/components/HexViewer';
import { useDeviceHistory } from '@/hooks/useDeviceHistory';
import { isJapanese } from '@/libs/languageHelper';
import { getPropertyName, formatPropertyValue, getPropertyDescriptor, shouldShowHexViewer } from '@/libs/propertyHelper';
import type { Device, PropertyDescriptionData } from '@/hooks/types';
import type { WebSocketConnection } from '@/hooks/useWebSocketConnection';

type DialogMessages = {
  title: string;
  settableOnlyLabel: string;
  loading: string;
  noHistory: string;
  close: string;
  reload: string;
  timestamp: string;
  property: string;
  value: string;
  origin: string;
  originSet: string;
  originNotification: string;
  originOnline: string;
  originOffline: string;
  eventOnline: string;
  eventOffline: string;
};

interface DeviceHistoryDialogProps {
  device: Device;
  connection: WebSocketConnection;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  propertyDescriptions: Record<string, PropertyDescriptionData>;
  classCode: string;
  isConnected: boolean;
}

export function DeviceHistoryDialog({
  device,
  connection,
  isOpen,
  onOpenChange,
  propertyDescriptions,
  classCode,
  isConnected,
}: DeviceHistoryDialogProps) {
  const [settableOnly, setSettableOnly] = useState(false);
  const deviceTarget = `${device.ip} ${device.eoj}`;

  const { entries, isLoading, error, refetch } = useDeviceHistory({
    connection,
    target: deviceTarget,
    limit: 50,
    settableOnly,
  });

  const messages: Record<'en' | 'ja', DialogMessages> = {
    en: {
      title: 'Device History',
      settableOnlyLabel: 'Settable properties only',
      loading: 'Loading history...',
      noHistory: 'No history available',
      close: 'Close',
      reload: 'Reload history',
      timestamp: 'Time',
      property: 'Property',
      value: 'Value',
      origin: 'Origin',
      originSet: 'Operation',
      originNotification: 'Notification',
      originOnline: 'Online',
      originOffline: 'Offline',
      eventOnline: 'Device came online',
      eventOffline: 'Device went offline',
    },
    ja: {
      title: 'デバイス履歴',
      settableOnlyLabel: '操作可能プロパティのみ',
      loading: '履歴を読み込み中...',
      noHistory: '履歴がありません',
      close: '閉じる',
      reload: '履歴を再読み込み',
      timestamp: '時刻',
      property: 'プロパティ',
      value: '値',
      origin: '発生源',
      originSet: '操作',
      originNotification: '通知',
      originOnline: 'オンライン',
      originOffline: 'オフライン',
      eventOnline: 'デバイスがオンラインになりました',
      eventOffline: 'デバイスがオフラインになりました',
    },
  };

  const texts = isJapanese() ? messages.ja : messages.en;

  const formatTimestamp = (timestamp: string): string => {
    const date = new Date(timestamp);
    return date.toLocaleString();
  };

  const getOriginText = (origin: 'set' | 'notification' | 'online' | 'offline'): string => {
    switch (origin) {
      case 'set':
        return texts.originSet;
      case 'notification':
        return texts.originNotification;
      case 'online':
        return texts.originOnline;
      case 'offline':
        return texts.originOffline;
    }
  };

  const getEventDescription = (origin: 'set' | 'notification' | 'online' | 'offline'): string | null => {
    switch (origin) {
      case 'online':
        return texts.eventOnline;
      case 'offline':
        return texts.eventOffline;
      default:
        return null;
    }
  };

  return (
    <AlertDialog open={isOpen} onOpenChange={onOpenChange}>
      <AlertDialogContent className="max-w-2xl max-h-[80vh] overflow-hidden flex flex-col">
        <AlertDialogHeader>
          <AlertDialogTitle>{texts.title}</AlertDialogTitle>
          <AlertDialogDescription className="text-xs text-muted-foreground">
            {device.name} ({device.ip} - {device.eoj})
          </AlertDialogDescription>
        </AlertDialogHeader>

        {/* Filter Controls */}
        <div className="flex items-center justify-between gap-4 py-2 border-b">
          <div className="flex items-center gap-2">
            <Switch
              checked={settableOnly}
              onCheckedChange={setSettableOnly}
              disabled={isLoading || !isConnected}
            />
            <label className="text-sm">{texts.settableOnlyLabel}</label>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={refetch}
            disabled={isLoading || !isConnected}
            title={texts.reload}
            className="h-8 w-8 p-0"
          >
            <RefreshCw className={`h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
          </Button>
        </div>

        {/* History Content */}
        <div className="flex-1 overflow-y-auto min-h-[200px]">
          {isLoading && (
            <div className="flex items-center justify-center h-full">
              <div className="flex flex-col items-center gap-2">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
                <p className="text-sm text-muted-foreground">{texts.loading}</p>
              </div>
            </div>
          )}

          {!isLoading && error && (
            <div className="flex items-center justify-center h-full">
              <div className="text-center">
                <p className="text-sm text-destructive">{error.message}</p>
              </div>
            </div>
          )}

          {!isLoading && !error && entries.length === 0 && (
            <div className="flex items-center justify-center h-full">
              <p className="text-sm text-muted-foreground">{texts.noHistory}</p>
            </div>
          )}

          {!isLoading && !error && entries.length > 0 && (
            <div className="space-y-2">
              {entries.map((entry, index) => {
                // Check if this is an event entry (online/offline)
                const isEvent = entry.origin === 'online' || entry.origin === 'offline';
                const eventDescription = getEventDescription(entry.origin);

                // For property changes
                const propertyName = entry.epc
                  ? getPropertyName(entry.epc, propertyDescriptions, classCode)
                  : '';
                const descriptor = entry.epc
                  ? getPropertyDescriptor(entry.epc, propertyDescriptions, classCode)
                  : undefined;
                const formattedValue = !isEvent
                  ? formatPropertyValue(entry.value, descriptor)
                  : '';
                const canShowHexViewer = !isEvent && shouldShowHexViewer(entry.value, descriptor);

                return (
                  <div
                    key={`${entry.timestamp}-${entry.epc || entry.origin}-${index}`}
                    className={`border rounded-lg p-3 text-sm ${
                      isEvent
                        ? 'border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-950'
                        : ''
                    }`}
                  >
                    <div className="flex items-start justify-between gap-2 mb-2">
                      <span className="font-semibold">
                        {isEvent ? eventDescription : propertyName}
                      </span>
                      <span className="text-xs text-muted-foreground">
                        {formatTimestamp(entry.timestamp)}
                      </span>
                    </div>
                    <div className="flex items-center justify-between gap-2">
                      {!isEvent && (
                        <div className="flex items-center gap-2 relative">
                          <span className="text-muted-foreground">{texts.value}:</span>
                          <span className="font-medium">{formattedValue}</span>
                          <HexViewer
                            canShowHexViewer={canShowHexViewer}
                            currentValue={entry.value}
                            size="sm"
                          />
                        </div>
                      )}
                      <span className="text-xs px-2 py-1 rounded bg-muted">
                        {getOriginText(entry.origin)}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        <AlertDialogFooter>
          <AlertDialogAction onClick={() => onOpenChange(false)}>
            {texts.close}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
