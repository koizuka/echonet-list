import { ChevronDown, ChevronUp, RefreshCw } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { PropertyRow } from '@/components/PropertyRow';
import { AliasEditor } from '@/components/AliasEditor';
import { DeviceStatusIndicators } from '@/components/DeviceStatusIndicators';
import { isPropertyPrimary, getSortedPrimaryProperties } from '@/libs/deviceTypeHelper';
import { deviceHasAlias, getDeviceIdentifierForAlias, getDeviceAliases } from '@/libs/deviceIdHelper';
import type { Device, PropertyValue, PropertyDescriptionData } from '@/hooks/types';

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
  isAliasLoading = false
}: DeviceCardProps) {
  const aliasInfo = deviceHasAlias(device, devices, aliases);
  const deviceAliasesInfo = getDeviceAliases(device, devices, aliases);
  const classCode = getDeviceClassCode(device);
  const deviceIdentifier = getDeviceIdentifierForAlias(device, devices);

  // Get primary properties in sorted order (Operation Status first)
  const primaryProps = getSortedPrimaryProperties(device);
  const secondaryProps = Object.entries(device.properties).filter(([epc]) => 
    !isPropertyPrimary(epc, classCode)
  );


  return (
    <Card className="transition-all duration-200 w-full max-w-sm flex flex-col" data-testid={`device-card-${device.ip}-${device.eoj}`}>
      <CardHeader className="pb-2 px-3 pt-3">
        <div className="flex items-start justify-between gap-2">
          <div className="space-y-1 flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <CardTitle className="text-sm font-semibold truncate" data-testid="device-title">
                {aliasInfo.aliasName || device.name}
              </CardTitle>
              {/* Multiple alias indicator in compact mode */}
              {!isExpanded && deviceAliasesInfo.aliases.length > 1 && (
                <div className="inline-flex items-center justify-center w-4 h-4 text-xs font-bold text-primary-foreground bg-primary rounded-full" title={`${deviceAliasesInfo.aliases.length}個のエイリアス`}>
                  {deviceAliasesInfo.aliases.length}
                </div>
              )}
              <DeviceStatusIndicators device={device} />
            </div>
            {aliasInfo.hasAlias && isExpanded && (
              <p className="text-xs text-muted-foreground truncate">
                Device: {device.name}
              </p>
            )}
            {isExpanded && (
              <p className="text-xs text-muted-foreground">
                {device.ip} - {device.eoj}
              </p>
            )}
          </div>
          <div className="flex items-center gap-1 shrink-0">
            {onUpdateProperties && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onUpdateProperties(`${device.ip} ${device.eoj}`)}
                className="h-6 w-6 p-0"
                title={isUpdating ? "Updating..." : "Update device properties"}
                disabled={isUpdating}
                data-testid="update-properties-button"
              >
                <RefreshCw className={`h-3 w-3 ${isUpdating ? 'animate-spin' : ''}`} />
              </Button>
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
          />
        </div>
      )}
      
      <CardContent className="pt-0 px-3 pb-0 flex flex-col flex-1">
        {/* Main content area that grows to fill space */}
        <div className="flex-1">
          {/* Always show primary properties in compact form */}
          {primaryProps.length > 0 && (
            <div className={`${isExpanded ? 'space-y-3' : 'space-y-0.5'} ${!isExpanded ? 'mb-2' : 'mb-3'}`}>
              {primaryProps.map(([epc, value]) => (
                <PropertyRow 
                  key={epc} 
                  device={device}
                  epc={epc} 
                  value={value as PropertyValue} 
                  isCompact={!isExpanded}
                  onPropertyChange={onPropertyChange}
                  propertyDescriptions={propertyDescriptions}
                  classCode={classCode}
                />
              ))}
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
                    value={value as PropertyValue}
                    isCompact={false}
                    onPropertyChange={onPropertyChange}
                    propertyDescriptions={propertyDescriptions}
                    classCode={classCode}
                  />
                ))}
              </div>
            </div>
          )}

        </div>

        {/* Last seen timestamp - always at bottom */}
        <div className="border-t pt-2 pb-3 mt-2">
          <p className="text-xs text-muted-foreground">
            Last seen: {new Date(device.lastSeen).toLocaleString()}
          </p>
        </div>
      </CardContent>
    </Card>
  );
}