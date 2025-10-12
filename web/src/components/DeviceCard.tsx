import { ChevronDown, ChevronUp, RefreshCw, Trash2, History } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { PropertyRow } from '@/components/PropertyRow';
import { AliasEditor } from '@/components/AliasEditor';
import { DeviceDeleteConfirmDialog } from '@/components/DeviceDeleteConfirmDialog';
import { DeviceHistoryDialog } from '@/components/DeviceHistoryDialog';
import { DeviceIcon } from '@/components/DeviceIcon';
import { useState } from 'react';
import { isPropertyPrimary, getSortedPrimaryProperties } from '@/libs/deviceTypeHelper';
import { deviceHasAlias, getDeviceIdentifierForAlias, getDeviceAliases } from '@/libs/deviceIdHelper';
import { isSensorProperty } from '@/libs/sensorPropertyHelper';
import { isDeviceOperational, isDeviceFaulty, isOperationStatusSettable } from '@/libs/propertyHelper';
import type { Device, PropertyValue, PropertyDescriptionData } from '@/hooks/types';
import type { WebSocketConnection } from '@/hooks/useWebSocketConnection';

interface DeviceCardProps {
  device: Device;
  isExpanded: boolean;
  onToggleExpansion: () => void;
  onPropertyChange: (target: string, epc: string, value: PropertyValue) => Promise<void>;
  onUpdateProperties?: (target: string) => Promise<void>;
  isUpdating?: boolean;
  propertyDescriptions: Record<string, PropertyDescriptionData>;
  getDeviceClassCode: (device: Device) => string;
  devices: Record<string, Device>;
  aliases: Record<string, string>;
  onAddAlias?: (alias: string, target: string) => Promise<void>;
  onDeleteAlias?: (alias: string) => Promise<void>;
  isAliasLoading?: boolean;
  isConnected?: boolean;
  onDeleteDevice?: (target: string) => Promise<void>;
  isDeletingDevice?: boolean;
  connection?: WebSocketConnection;
}

export function DeviceCard({
  device,
  isExpanded,
  onToggleExpansion,
  onPropertyChange,
  onUpdateProperties,
  isUpdating = false,
  propertyDescriptions,
  getDeviceClassCode,
  devices,
  aliases,
  onAddAlias,
  onDeleteAlias,
  isAliasLoading = false,
  isConnected = true,
  onDeleteDevice,
  isDeletingDevice = false,
  connection
}: DeviceCardProps) {
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [isHistoryDialogOpen, setIsHistoryDialogOpen] = useState(false);
  const aliasInfo = deviceHasAlias(device, devices, aliases);
  const deviceAliasesInfo = getDeviceAliases(device, devices, aliases);
  const classCode = getDeviceClassCode(device);
  const deviceIdentifier = getDeviceIdentifierForAlias(device, devices);

  // Device status for styling
  const isOperational = isDeviceOperational(device);
  const isFaulty = isDeviceFaulty(device);
  const isOffline = device.isOffline || false;
  const isControllable = isOperationStatusSettable(device);

  // Determine border color based on device status
  const getBorderColorClass = (): string => {
    if (isOffline) {
      return 'border-muted-foreground/30';
    }
    if (isFaulty) {
      return 'border-red-500/60';
    }
    if (isOperational && isControllable) {
      return 'border-green-500/60';
    }
    return 'border-border'; // Default border
  };

  // Get primary properties in sorted order (Operation Status first)
  const primaryProps = getSortedPrimaryProperties(device);
  const secondaryProps = Object.entries(device.properties).filter(([epc]) => 
    !isPropertyPrimary(epc, classCode)
  );

  // For compact view, separate sensor and non-sensor properties
  const primarySensorProps = primaryProps.filter(([epc]) => isSensorProperty(classCode, epc));
  const primaryNonSensorProps = primaryProps.filter(([epc]) => !isSensorProperty(classCode, epc));


  return (
    <Card 
      className={`transition-all duration-200 w-full max-w-sm flex flex-col relative border-2 ${getBorderColorClass()} ${device.isOffline ? 'after:absolute after:inset-0 after:bg-background/60 after:pointer-events-none after:rounded-lg' : ''}`} 
      data-testid={`device-card-${device.ip}-${device.eoj}`}
    >
      <CardHeader className="pb-2 px-3 pt-3">
        <div className="flex items-start justify-between gap-2">
          <div className="space-y-1 flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <DeviceIcon device={device} classCode={classCode} />
              <CardTitle className="text-sm font-semibold truncate" data-testid="device-title">
                {aliasInfo.aliasName || device.name}
              </CardTitle>
              {/* Multiple alias indicator in compact mode */}
              {!isExpanded && deviceAliasesInfo.aliases.length > 1 && (
                <div className="inline-flex items-center justify-center w-4 h-4 text-xs font-bold text-primary-foreground bg-primary rounded-full" title={`${deviceAliasesInfo.aliases.length}個のエイリアス`}>
                  {deviceAliasesInfo.aliases.length}
                </div>
              )}
            </div>
            {aliasInfo.hasAlias && isExpanded && (
              <p className="text-xs text-muted-foreground truncate">
                Device: {device.name}
              </p>
            )}
            {(isExpanded || !aliasInfo.hasAlias) && (
              <p className="text-xs text-muted-foreground">
                {device.ip} - {device.eoj}
              </p>
            )}
          </div>
          <div className="flex items-center gap-1 shrink-0 relative z-10">
            {connection && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setIsHistoryDialogOpen(true)}
                className="h-6 w-6 p-0"
                title="View device history"
                disabled={!isConnected || device.isOffline}
                data-testid="history-button"
              >
                <History className="h-3 w-3" />
              </Button>
            )}
            {onUpdateProperties && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onUpdateProperties(`${device.ip} ${device.eoj}`)}
                className="h-6 w-6 p-0"
                title={isUpdating ? "Updating..." : device.isOffline ? "Try to reconnect device" : "Update device properties"}
                disabled={isUpdating || !isConnected}
                data-testid="update-properties-button"
              >
                <RefreshCw className={`h-3 w-3 ${isUpdating ? 'animate-spin' : ''}`} />
              </Button>
            )}
            {device.isOffline && onDeleteDevice && (
              <>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-6 w-6 p-0 text-destructive hover:text-destructive hover:bg-destructive/10"
                  title="Delete offline device"
                  disabled={isDeletingDevice || !isConnected}
                  data-testid="delete-device-button"
                  onClick={() => setIsDeleteDialogOpen(true)}
                >
                  <Trash2 className="h-3 w-3" />
                </Button>
                <DeviceDeleteConfirmDialog
                  device={device}
                  aliasName={aliasInfo.aliasName}
                  onDeleteDevice={onDeleteDevice}
                  isDeletingDevice={isDeletingDevice}
                  isConnected={isConnected}
                  isOpen={isDeleteDialogOpen}
                  onOpenChange={setIsDeleteDialogOpen}
                />
              </>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={onToggleExpansion}
              className="h-6 w-6 p-0"
              data-testid="expand-collapse-button"
            >
              {isExpanded ? (
                <ChevronUp className="h-3 w-3" />
              ) : (
                <ChevronDown className="h-3 w-3" />
              )}
            </Button>
          </div>
        </div>
      </CardHeader>
      
      {/* Alias Editor - Full width below header */}
      {isExpanded && onAddAlias && onDeleteAlias && deviceIdentifier && (
        <div className="px-3 pb-2">
          <AliasEditor
            device={device}
            aliases={deviceAliasesInfo.aliases}
            onAddAlias={onAddAlias}
            onDeleteAlias={onDeleteAlias}
            isLoading={isAliasLoading}
            deviceIdentifier={deviceIdentifier}
            isConnected={isConnected}
          />
        </div>
      )}
      
      <CardContent className="pt-0 px-3 pb-0 flex flex-col flex-1">
        {/* Main content area that grows to fill space */}
        <div className="flex-1">
          {/* Show primary properties */}
          {primaryProps.length > 0 && (
            <div className={`${isExpanded ? 'space-y-3' : 'space-y-0.5'} ${!isExpanded ? 'mb-2' : 'mb-3'}`}>
              {isExpanded ? (
                // Expanded mode: show all properties as separate rows
                primaryProps.map(([epc, value]) => (
                  <PropertyRow 
                    key={epc} 
                    device={device}
                    epc={epc} 
                    value={value} 
                    isCompact={false}
                    onPropertyChange={onPropertyChange}
                    propertyDescriptions={propertyDescriptions}
                    classCode={classCode}
                    isConnected={isConnected && !device.isOffline}
                    allDevices={devices}
                    aliases={aliases}
                    getDeviceClassCode={getDeviceClassCode}
                  />
                ))
              ) : (
                // Compact mode: show only non-sensor properties here
                primaryNonSensorProps.map(([epc, value]) => (
                  <PropertyRow 
                    key={epc} 
                    device={device}
                    epc={epc} 
                    value={value} 
                    isCompact={true}
                    onPropertyChange={onPropertyChange}
                    propertyDescriptions={propertyDescriptions}
                    classCode={classCode}
                    isConnected={isConnected && !device.isOffline}
                    allDevices={devices}
                    aliases={aliases}
                    getDeviceClassCode={getDeviceClassCode}
                  />
                ))
              )}
            </div>
          )}

          {/* Show secondary properties only when expanded */}
          {isExpanded && secondaryProps.length > 0 && (
            <div className="border-t pt-2">
              <h4 className="text-xs font-medium mb-2 text-muted-foreground">
                Other Properties
              </h4>
              <div className="space-y-3">
                {secondaryProps.map(([epc, value]) => (
                  <PropertyRow 
                    key={epc} 
                    device={device}
                    epc={epc} 
                    value={value}
                    isCompact={false}
                    onPropertyChange={onPropertyChange}
                    propertyDescriptions={propertyDescriptions}
                    classCode={classCode}
                    isConnected={isConnected && !device.isOffline}
                    allDevices={devices}
                    aliases={aliases}
                    getDeviceClassCode={getDeviceClassCode}
                  />
                ))}
              </div>
            </div>
          )}

        </div>

        {/* Sensor properties in compact mode - positioned at the bottom */}
        {!isExpanded && primarySensorProps.length > 0 && (
          <div className="border-t pt-2 mt-2">
            <div className="flex flex-wrap items-center">
              {primarySensorProps.map(([epc, value]) => (
                <PropertyRow 
                  key={epc} 
                  device={device}
                  epc={epc} 
                  value={value as PropertyValue} 
                  isCompact={true}
                  onPropertyChange={onPropertyChange}
                  propertyDescriptions={propertyDescriptions}
                  classCode={classCode}
                  isConnected={isConnected}
                  allDevices={devices}
                  aliases={aliases}
                  getDeviceClassCode={getDeviceClassCode}
                />
              ))}
            </div>
          </div>
        )}

        {/* Last seen timestamp - only show in expanded mode */}
        {isExpanded && (
          <div className="border-t pt-2 pb-3 mt-2">
            <p className="text-xs text-muted-foreground">
              Last seen: {new Date(device.lastSeen).toLocaleString()}
            </p>
          </div>
        )}
      </CardContent>

      {/* Device History Dialog */}
      {connection && (
        <DeviceHistoryDialog
          device={device}
          connection={connection}
          isOpen={isHistoryDialogOpen}
          onOpenChange={setIsHistoryDialogOpen}
          propertyDescriptions={propertyDescriptions}
          classCode={classCode}
          isConnected={isConnected}
        />
      )}
    </Card>
  );
}