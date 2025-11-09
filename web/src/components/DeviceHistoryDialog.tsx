import { useState, useMemo, useEffect, useRef } from 'react';
import { RefreshCw, Loader2, CheckCircle, XCircle, Edit, Eye, Binary, X } from 'lucide-react';
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
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useDeviceHistory } from '@/hooks/useDeviceHistory';
import { isJapanese } from '@/libs/languageHelper';
import { getPropertyName, formatPropertyValue, getPropertyDescriptor, shouldShowHexViewer, edtToHexString } from '@/libs/propertyHelper';
import { deviceHasAlias } from '@/libs/deviceIdHelper';
import { getDevicePrimaryProperties } from '@/libs/deviceTypeHelper';
import type { Device, PropertyDescriptionData, HistoryOrigin, DeviceHistoryEntry } from '@/hooks/types';
import type { WebSocketConnection } from '@/hooks/useWebSocketConnection';

// Special key used to store event entries (online/offline) in the property map
const EVENT_KEY = '__event__' as const;

/**
 * Represents a group of history entries that occurred at the same time with the same origin and settable status.
 * Each group becomes a single row in the history table.
 */
interface HistoryGroup {
  timestamp: string;
  origin: HistoryOrigin;
  settable: boolean;
  properties: Map<string, DeviceHistoryEntry>;
}

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
  const [selectedHexData, setSelectedHexData] = useState<{ epc: string; edt: string; propertyName: string; timestamp: string } | null>(null);
  const [lastFetchTime, setLastFetchTime] = useState<string>('');
  const deviceTarget = `${device.ip} ${device.eoj}`;

  // Track last fetch time for cache management (10 second cache)
  const lastFetchTimeRef = useRef<number>(0);

  // Helper function to format timestamp as HH:MM:SS
  const formatFetchTime = (timestamp: number): string => {
    const date = new Date(timestamp);
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    const seconds = String(date.getSeconds()).padStart(2, '0');
    return `${hours}:${minutes}:${seconds}`;
  };

  // Wrapper function to handle refetch with timestamp update
  const handleRefetch = () => {
    const now = Date.now();
    lastFetchTimeRef.current = now;
    setLastFetchTime(formatFetchTime(now));
    refetch();
  };

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

  // Auto-fetch history when dialog opens (with 10 second cache)
  useEffect(() => {
    if (isOpen && isConnected) {
      const now = Date.now();
      const cacheTime = 10000; // 10 seconds

      // Only refetch if cache has expired
      if (now - lastFetchTimeRef.current >= cacheTime) {
        handleRefetch();
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, isConnected]);

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

  /**
   * Groups history entries by timestamp, origin, and settable status, and collects unique properties for column headers.
   *
   * This transforms the flat list of history entries into a table structure where:
   * - Each row represents entries with the same timestamp (truncated to seconds), origin, and settable status
   * - Each column represents a property (EPC)
   * - Event entries (online/offline) are stored with a special EVENT_KEY
   * - Property names are pre-computed for performance
   * - Entries with different settable values are separated into different rows
   *
   * @returns Object containing:
   *   - groupedEntries: Rows sorted by timestamp (newest first)
   *   - propertyColumns: Sorted array of unique EPCs
   *   - propertyNames: Map of EPC to human-readable property names
   */
  const { groupedEntries, propertyColumns, propertyNames } = useMemo(() => {
    // Group entries by timestamp (truncated to seconds), origin, AND settable (for non-event entries)
    // Events (online/offline) always get their own row
    const groupMap = new Map<string, HistoryGroup>();
    const uniqueProperties = new Set<string>();

    entries.forEach((entry) => {
      const isEvent = entry.origin === 'online' || entry.origin === 'offline';

      // Truncate timestamp to seconds (remove milliseconds) in local timezone
      // This groups entries that occurred within the same second in the user's local time
      const date = new Date(entry.timestamp);
      const year = date.getFullYear();
      const month = String(date.getMonth() + 1).padStart(2, '0');
      const day = String(date.getDate()).padStart(2, '0');
      const hours = String(date.getHours()).padStart(2, '0');
      const minutes = String(date.getMinutes()).padStart(2, '0');
      const seconds = String(date.getSeconds()).padStart(2, '0');
      const timestampSeconds = `${year}-${month}-${day}T${hours}:${minutes}:${seconds}`;

      // Create a grouping key: for events, use timestamp only (they always get their own row)
      // For properties, use timestamp (truncated to seconds in local time) + origin + settable to ensure
      // settable and non-settable properties are never mixed in the same row
      const groupKey = isEvent ? entry.timestamp : `${timestampSeconds}:${entry.origin}:${entry.settable}`;

      if (!groupMap.has(groupKey)) {
        groupMap.set(groupKey, {
          timestamp: entry.timestamp, // Keep the original timestamp for display
          origin: entry.origin,
          settable: entry.settable,
          properties: new Map(),
        });
      }

      const group = groupMap.get(groupKey);
      if (!group) {
        // This should never happen since we just created the group above
        // If it does, it indicates a logic error in the grouping algorithm
        console.error('Group not found for key:', groupKey, 'This indicates a logic error.');
        return;
      }

      if (!isEvent && entry.epc && typeof entry.epc === 'string') {
        uniqueProperties.add(entry.epc);
        // If the same property already exists, keep the latest entry by comparing timestamps
        const existing = group.properties.get(entry.epc);
        if (!existing || new Date(entry.timestamp) >= new Date(existing.timestamp)) {
          group.properties.set(entry.epc, entry);
        }
      } else if (isEvent) {
        // Store event entries with a special key
        group.properties.set(EVENT_KEY, entry);
      }
    });

    // Memoize property names for better performance
    const propertyNameMap = new Map<string, string>();
    uniqueProperties.forEach(epc => {
      propertyNameMap.set(epc, getPropertyName(epc, propertyDescriptions, classCode));
    });

    /**
     * Sort properties to match DeviceCard's full mode display order.
     *
     * This ensures consistent property column ordering across the UI:
     * 1. Primary properties appear first in their predefined order (from DEVICE_PRIMARY_PROPERTIES)
     * 2. Secondary properties follow in insertion order (matching Object.entries behavior)
     *
     * Note: JavaScript Set maintains insertion order per ECMAScript 2015+ specification,
     * so uniqueProperties preserves the order properties were encountered in history entries.
     */
    const primaryProperties = getDevicePrimaryProperties(classCode);
    const allProperties = Array.from(uniqueProperties);

    // Separate properties into primary and secondary groups
    const primaryEPCs = allProperties.filter(epc => primaryProperties.includes(epc));
    const secondaryEPCs = allProperties.filter(epc => !primaryProperties.includes(epc));

    // Sort primary properties by their order in the predefined list
    const sortedPrimary = primaryEPCs.sort((a, b) => {
      const indexA = primaryProperties.indexOf(a);
      const indexB = primaryProperties.indexOf(b);
      return indexA - indexB;
    });

    // Combine: primary first, then secondary (which maintains insertion order)
    const sortedProperties = [...sortedPrimary, ...secondaryEPCs];

    // Convert to array and sort by timestamp (newest first), then by origin
    const grouped = Array.from(groupMap.values())
      .sort((a, b) => {
        const timeDiff = new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime();
        if (timeDiff !== 0) return timeDiff;
        // If timestamps are equal, sort by origin (events first, then set, then notification)
        const originOrder: Record<HistoryOrigin, number> = { online: 0, offline: 1, set: 2, notification: 3 };
        return originOrder[a.origin] - originOrder[b.origin];
      });

    return {
      groupedEntries: grouped,
      propertyColumns: sortedProperties,
      propertyNames: propertyNameMap,
    };
  }, [entries, propertyDescriptions, classCode]);

  return (
    <AlertDialog open={isOpen} onOpenChange={onOpenChange}>
      <AlertDialogContent className="max-w-4xl max-h-[80vh] overflow-hidden flex flex-col">
        <AlertDialogHeader>
          <AlertDialogTitle>{dialogTitle}</AlertDialogTitle>
          <div className="text-xs text-muted-foreground space-y-0.5 text-left">
            {aliasInfo.hasAlias && <div>Device: {device.name}</div>}
            <div className="flex items-center justify-between gap-2">
              <span>{device.ip} - {device.eoj}</span>
              {lastFetchTime && (
                <span
                  className="text-xs text-muted-foreground"
                  title="Last fetched at"
                  aria-label={`Data last fetched at ${lastFetchTime}`}
                >
                  [{lastFetchTime}]
                </span>
              )}
            </div>
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
            onClick={handleRefetch}
            disabled={isLoading || !isConnected}
            title={texts.reload}
            className="h-8 w-8 p-0"
          >
            <RefreshCw className={`h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
          </Button>
        </div>

        {/* History Content and Hex Display Container */}
        <div className="flex-1 flex flex-col min-h-0 overflow-auto">
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
              <div className="relative w-full">
                {/* Use raw <table> element instead of shadcn Table wrapper for better scroll control */}
                <table className="w-full caption-bottom text-sm" aria-label="Device history with properties as columns">
                <TableHeader>
                <TableRow>
                  {/* Z-index strategy: timestamp header needs z-30 to appear above other sticky headers (z-20)
                      when both vertical (top-0) and horizontal (left-0) sticky positioning are active */}
                  <TableHead className="w-[140px] sticky top-0 left-0 bg-background dark:bg-background z-30 border-r h-8 py-1 px-0.5">{texts.timestamp}</TableHead>
                  <TableHead className="w-[100px] sticky top-0 bg-background dark:bg-background z-20 h-8 py-1 px-0.5">{texts.origin}</TableHead>
                  {propertyColumns.map((epc) => {
                    const propertyName = propertyNames.get(epc) || epc;
                    return (
                      <TableHead
                        key={epc}
                        className="min-w-[100px] max-w-[200px] sticky top-0 bg-background dark:bg-background z-20 h-8 py-1 px-0.5 text-center"
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
                  const hasEvent = group.properties.has(EVENT_KEY);
                  const eventEntry = hasEvent ? group.properties.get(EVENT_KEY) : null;
                  const isOnline = eventEntry?.origin === 'online';
                  const isOffline = eventEntry?.origin === 'offline';

                  // Determine row color based on event type or settable status
                  // Priority: online (green) > offline (red) > settable (blue) > default (no color)
                  const rowColorClass = isOnline
                    ? 'bg-green-200 dark:bg-green-900 text-green-900 dark:text-green-200 font-semibold'
                    : isOffline
                    ? 'bg-red-200 dark:bg-red-900 text-red-900 dark:text-red-200 font-semibold'
                    : group.settable
                    ? 'bg-blue-100 dark:bg-blue-950 text-blue-900 dark:text-blue-100'
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
                        <TableCell className={`font-mono text-xs sticky left-0 z-20 border-r py-1 px-0.5 ${rowColorClass}`}>
                          {formatTimestamp(group.timestamp)}
                        </TableCell>

                        {/* Origin with icon */}
                        <TableCell className="text-xs py-1 px-0.5">
                          <div className="flex items-center gap-1.5">
                            <OriginIcon className="h-3 w-3 shrink-0" aria-hidden="true" />
                            <span className="px-1.5 py-0.5 rounded bg-muted/70 whitespace-nowrap text-xs">
                              {getOriginText(group.origin)}
                            </span>
                          </div>
                        </TableCell>

                        {/* Event description spans all property columns */}
                        <TableCell colSpan={propertyColumns.length} className="font-semibold py-1 px-0.5">
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
                      <TableCell className={`font-mono text-xs sticky left-0 z-20 border-r py-1 px-0.5 ${rowColorClass || 'text-muted-foreground bg-background dark:bg-background'}`}>
                        {formatTimestamp(group.timestamp)}
                      </TableCell>

                      {/* Origin with icon */}
                      <TableCell className="text-xs py-1 px-0.5">
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
                          return <TableCell key={epc} className="text-muted-foreground text-center py-1 px-0.5">-</TableCell>;
                        }

                        const descriptor = getPropertyDescriptor(epc, propertyDescriptions, classCode);
                        const formattedValue = formatPropertyValue(entry.value, descriptor);
                        const canShowHexViewer = shouldShowHexViewer(entry.value, descriptor);
                        const propertyName = propertyNames.get(epc) || epc;

                        // Check if this hex data is currently selected
                        const isSelected = selectedHexData?.epc === epc &&
                                          selectedHexData?.edt === entry.value.EDT;

                        // Row-level coloring handles all settable property highlighting
                        // No need for cell-level coloring since grouping ensures consistent settable status per row
                        return (
                          <TableCell key={epc} className="py-1 px-0.5">
                            <div className="flex items-center justify-center gap-1">
                              <span className="font-medium font-mono text-xs truncate" title={formattedValue}>
                                {formattedValue}
                              </span>
                              {canShowHexViewer && entry.value.EDT && (
                                <Button
                                  variant={isSelected ? "default" : "outline"}
                                  size="sm"
                                  onClick={() => {
                                    if (isSelected) {
                                      setSelectedHexData(null);
                                    } else {
                                      setSelectedHexData({
                                        epc,
                                        edt: entry.value.EDT!,
                                        propertyName,
                                        timestamp: group.timestamp,
                                      });
                                    }
                                  }}
                                  className="h-4 w-4 p-0"
                                  title={isSelected ? "Hide hex data" : "Show hex data"}
                                  aria-label={isSelected ? "Hide hex data" : "Show hex data"}
                                >
                                  <Binary className="h-2 w-2" />
                                </Button>
                              )}
                            </div>
                          </TableCell>
                        );
                      })}
                    </TableRow>
                  );
                })}
              </TableBody>
              </table>
            </div>
          )}
          </div>

          {/* Hex Data Display Area */}
          {selectedHexData && (
            <div className="border-t p-2 bg-muted/30 flex-shrink-0">
              <div className="flex items-start justify-between gap-2 mb-1">
                <div className="flex-1">
                  <span className="text-xs text-muted-foreground">{formatTimestamp(selectedHexData.timestamp)}</span>
                  <span className="text-xs font-semibold ml-2">{selectedHexData.propertyName}</span>
                  <span className="text-xs text-muted-foreground ml-1">({selectedHexData.epc})</span>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setSelectedHexData(null)}
                  className="h-5 w-5 p-0"
                  title="Close hex viewer"
                  aria-label="Close hex viewer"
                >
                  <X className="h-3 w-3" />
                </Button>
              </div>
              <div className="text-xs font-mono bg-background p-2 rounded border break-words overflow-auto max-h-[300px] min-h-[60px]">
                {edtToHexString(selectedHexData.edt) || 'Invalid data'}
              </div>
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
