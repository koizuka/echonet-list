import { useState, useMemo } from 'react';
import { RefreshCw, Loader2, CheckCircle, XCircle, Edit, Eye } from 'lucide-react';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
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
import { deviceHasAlias } from '@/libs/deviceIdHelper';
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
  aliases?: Record<string, string>;
  allDevices?: Record<string, Device>;
}

export function DeviceHistoryDialog({
  device,
  connection,
  isOpen,
  onOpenChange,
  propertyDescriptions,
  classCode,
  isConnected,
  aliases = {},
  allDevices = {},
}: DeviceHistoryDialogProps) {
  const [settableOnly, setSettableOnly] = useState(false);
  const deviceTarget = `${device.ip} ${device.eoj}`;

  // Get device alias information (memoized for performance)
  const aliasInfo = useMemo(
    () => deviceHasAlias(device, allDevices, aliases),
    [device, allDevices, aliases]
  );
  const displayName = useMemo(
    () => aliasInfo.aliasName || device.name,
    [aliasInfo.aliasName, device.name]
  );

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

  // Generate dialog title with device name (memoized for performance)
  const dialogTitle = useMemo(
    () => isJapanese()
      ? `${displayName}のデバイス履歴`
      : `${displayName} - Device History`,
    [displayName]
  );

  const formatTimestamp = (timestamp: string): string => {
    const date = new Date(timestamp);
    // Use Intl.DateTimeFormat for better localization support
    const formatted = new Intl.DateTimeFormat('en-US', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    }).format(date);
    // Format: "MM/DD, HH:MM:SS" -> "MM/DD HH:MM:SS"
    return formatted.replace(',', '');
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

  // Memoize processed entries to prevent re-rendering on propertyDescriptions updates
  /* eslint-disable react-hooks/preserve-manual-memoization */
  const processedEntries = useMemo(() => {
    return entries.map((entry, index) => {
      const isEvent = entry.origin === 'online' || entry.origin === 'offline';
      const eventDescription = entry.origin === 'online'
        ? texts.eventOnline
        : entry.origin === 'offline'
        ? texts.eventOffline
        : null;

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

      return {
        ...entry,
        index,
        isEvent,
        eventDescription,
        propertyName,
        formattedValue,
        canShowHexViewer,
      };
    });
  }, [entries, propertyDescriptions, classCode, texts.eventOnline, texts.eventOffline]);
  /* eslint-enable react-hooks/preserve-manual-memoization */

  return (
    <AlertDialog open={isOpen} onOpenChange={onOpenChange}>
      <AlertDialogContent className="max-w-2xl max-h-[80vh] overflow-hidden flex flex-col">
        <AlertDialogHeader>
          <AlertDialogTitle>{dialogTitle}</AlertDialogTitle>
          <div className="text-xs text-muted-foreground space-y-0.5">
            {aliasInfo.hasAlias && <div>Device: {device.name}</div>}
            <div>{device.ip} - {device.eoj}</div>
          </div>
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
        <div className="flex-1 overflow-y-scroll min-h-[200px] scrollbar-visible">
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

          {!isLoading && !error && processedEntries.length > 0 && (
            <div className="space-y-0.5 font-mono text-xs">
              {processedEntries.map((entry) => {
                // Determine color scheme based on entry type
                // Events (online/offline) get their own colors
                // Settable properties get blue background
                // Other entries have no background color
                const eventColorClass = entry.isEvent
                  ? entry.origin === 'online'
                    ? 'bg-green-200 dark:bg-green-800 text-green-900 dark:text-green-100 font-semibold border-l-4 border-green-600 dark:border-green-400'
                    : 'bg-red-200 dark:bg-red-800 text-red-900 dark:text-red-100 font-semibold border-l-4 border-red-600 dark:border-red-400'
                  : entry.settable
                  ? 'bg-blue-200 dark:bg-blue-800 text-blue-900 dark:text-blue-100 border-l-4 border-blue-600 dark:border-blue-400'
                  : '';

                // Generate testid for event entries
                const testId = entry.isEvent ? `history-event-${entry.origin}` : undefined;

                // Determine icon based on entry type
                const EntryIcon = entry.isEvent
                  ? entry.origin === 'online'
                    ? CheckCircle
                    : XCircle
                  : entry.settable
                  ? Edit
                  : Eye;

                return (
                  <div
                    key={entry.index}
                    className={`flex items-center gap-2 px-2 py-1 hover:bg-muted/50 ${eventColorClass}`}
                    data-testid={testId}
                  >
                    {/* Icon for accessibility */}
                    <EntryIcon className="h-3 w-3 shrink-0" aria-hidden="true" />

                    {/* Timestamp */}
                    <span className="text-muted-foreground shrink-0">
                      {formatTimestamp(entry.timestamp)}
                    </span>

                    {/* Origin badge */}
                    <span className="text-xs px-1.5 py-0.5 rounded bg-muted/70 shrink-0">
                      {getOriginText(entry.origin)}
                    </span>

                    {/* Property/Event name */}
                    <span className="font-medium truncate">
                      {entry.isEvent ? entry.eventDescription : entry.propertyName}
                    </span>

                    {/* Value (for property changes only) */}
                    {!entry.isEvent && (
                      <>
                        <span className="text-muted-foreground">:</span>
                        <span className="font-medium">{entry.formattedValue}</span>
                        <HexViewer
                          canShowHexViewer={entry.canShowHexViewer}
                          currentValue={entry.value}
                          size="sm"
                        />
                      </>
                    )}
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
