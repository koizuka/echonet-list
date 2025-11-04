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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
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

  // Group entries by timestamp and collect all unique properties
  const { groupedEntries, propertyColumns } = useMemo(() => {
    // Group entries by timestamp
    const timestampMap = new Map<string, Map<string, typeof entries[0]>>();
    const uniqueProperties = new Set<string>();

    entries.forEach((entry) => {
      const timestamp = entry.timestamp;
      if (!timestampMap.has(timestamp)) {
        timestampMap.set(timestamp, new Map());
      }

      const isEvent = entry.origin === 'online' || entry.origin === 'offline';
      if (!isEvent && entry.epc) {
        uniqueProperties.add(entry.epc);
        timestampMap.get(timestamp)!.set(entry.epc, entry);
      } else if (isEvent) {
        // Store event entries with a special key
        timestampMap.get(timestamp)!.set('__event__', entry);
      }
    });

    // Sort properties by name for consistent column order
    const sortedProperties = Array.from(uniqueProperties).sort((a, b) => {
      const nameA = getPropertyName(a, propertyDescriptions, classCode);
      const nameB = getPropertyName(b, propertyDescriptions, classCode);
      return nameA.localeCompare(nameB);
    });

    // Convert to array and sort by timestamp (newest first)
    const grouped = Array.from(timestampMap.entries())
      .map(([timestamp, propertyMap]) => ({
        timestamp,
        properties: propertyMap,
        // Get the origin from the first entry at this timestamp
        origin: Array.from(propertyMap.values())[0]?.origin || 'notification',
      }))
      .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

    return {
      groupedEntries: grouped,
      propertyColumns: sortedProperties,
    };
  }, [entries, propertyDescriptions, classCode]);

  return (
    <AlertDialog open={isOpen} onOpenChange={onOpenChange}>
      <AlertDialogContent className="max-w-4xl max-h-[80vh] overflow-hidden flex flex-col">
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
        <div className="flex-1 overflow-auto min-h-[200px] scrollbar-visible">
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

          {!isLoading && !error && groupedEntries.length > 0 && (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[140px] sticky left-0 bg-background dark:bg-background z-20 border-r h-8 py-1 px-2">{texts.timestamp}</TableHead>
                  <TableHead className="w-[100px] h-8 py-1 px-2">{texts.origin}</TableHead>
                  {propertyColumns.map((epc) => {
                    const propertyName = getPropertyName(epc, propertyDescriptions, classCode);
                    return (
                      <TableHead
                        key={epc}
                        className="min-w-[120px] max-w-[200px] h-8 py-1 px-2 text-center"
                        title={propertyName}
                      >
                        <span className="truncate block text-xs">{propertyName}</span>
                      </TableHead>
                    );
                  })}
                </TableRow>
              </TableHeader>
              <TableBody>
                {groupedEntries.map((group, rowIndex) => {
                  const hasEvent = group.properties.has('__event__');
                  const eventEntry = hasEvent ? group.properties.get('__event__') : null;
                  const isOnline = eventEntry?.origin === 'online';
                  const isOffline = eventEntry?.origin === 'offline';

                  // Determine row color based on event type
                  const rowColorClass = isOnline
                    ? 'bg-green-200 dark:bg-green-800 text-green-900 dark:text-green-100 font-semibold'
                    : isOffline
                    ? 'bg-red-200 dark:bg-red-800 text-red-900 dark:text-red-100 font-semibold'
                    : '';

                  const testId = hasEvent ? `history-event-${eventEntry?.origin}` : undefined;

                  // Determine icon based on origin
                  const OriginIcon = isOnline
                    ? CheckCircle
                    : isOffline
                    ? XCircle
                    : group.origin === 'set'
                    ? Edit
                    : Eye;

                  // For event rows, show event description across all property columns
                  if (hasEvent) {
                    const eventDescription = isOnline
                      ? texts.eventOnline
                      : texts.eventOffline;

                    return (
                      <TableRow
                        key={rowIndex}
                        className={rowColorClass}
                        data-testid={testId}
                      >
                        {/* Timestamp */}
                        <TableCell className="font-mono text-xs text-muted-foreground sticky left-0 bg-background dark:bg-background z-20 border-r py-1 px-2">
                          {formatTimestamp(group.timestamp)}
                        </TableCell>

                        {/* Origin with icon */}
                        <TableCell className="text-xs py-1 px-2">
                          <div className="flex items-center gap-1.5">
                            <OriginIcon className="h-3 w-3 shrink-0" aria-hidden="true" />
                            <span className="px-1.5 py-0.5 rounded bg-muted/70 whitespace-nowrap text-xs">
                              {getOriginText(group.origin)}
                            </span>
                          </div>
                        </TableCell>

                        {/* Event description spans all property columns */}
                        <TableCell colSpan={propertyColumns.length} className="font-semibold py-1 px-2">
                          {eventDescription}
                        </TableCell>
                      </TableRow>
                    );
                  }

                  // For normal rows, show property values
                  return (
                    <TableRow
                      key={rowIndex}
                      className={rowColorClass}
                      data-testid={testId}
                    >
                      {/* Timestamp */}
                      <TableCell className="font-mono text-xs text-muted-foreground sticky left-0 bg-background dark:bg-background z-20 border-r py-1 px-2">
                        {formatTimestamp(group.timestamp)}
                      </TableCell>

                      {/* Origin with icon */}
                      <TableCell className="text-xs py-1 px-2">
                        <div className="flex items-center gap-1.5">
                          <OriginIcon className="h-3 w-3 shrink-0" aria-hidden="true" />
                          <span className="px-1.5 py-0.5 rounded bg-muted/70 whitespace-nowrap text-xs">
                            {getOriginText(group.origin)}
                          </span>
                        </div>
                      </TableCell>

                      {/* Property values */}
                      {propertyColumns.map((epc) => {
                        const entry = group.properties.get(epc);
                        if (!entry) {
                          return <TableCell key={epc} className="text-muted-foreground text-center py-1 px-2">-</TableCell>;
                        }

                        const descriptor = getPropertyDescriptor(epc, propertyDescriptions, classCode);
                        const formattedValue = formatPropertyValue(entry.value, descriptor);
                        const canShowHexViewer = shouldShowHexViewer(entry.value, descriptor);

                        // Apply blue background for settable properties
                        const cellColorClass = entry.settable
                          ? 'bg-blue-200 dark:bg-blue-800 text-blue-900 dark:text-blue-100 py-1 px-2'
                          : 'py-1 px-2';

                        return (
                          <TableCell key={epc} className={cellColorClass}>
                            <div className="flex items-center justify-center gap-1">
                              <span className="font-medium font-mono text-xs truncate" title={formattedValue}>
                                {formattedValue}
                              </span>
                              <HexViewer
                                canShowHexViewer={canShowHexViewer}
                                currentValue={entry.value}
                                size="sm"
                              />
                            </div>
                          </TableCell>
                        );
                      })}
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
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
