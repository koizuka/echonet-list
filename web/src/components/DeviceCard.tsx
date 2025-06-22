import { useState } from 'react';
import { ChevronDown, ChevronUp, RefreshCw, Binary } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { PropertyEditor } from '@/components/PropertyEditor';
import { AliasEditor } from '@/components/AliasEditor';
import { DeviceStatusIndicators } from '@/components/DeviceStatusIndicators';
import { getPropertyName, formatPropertyValueWithTranslation, getPropertyDescriptor, isPropertySettable, shouldShowHexViewer, edtToHexString } from '@/libs/propertyHelper';
import { translateLocationId } from '@/libs/locationHelper';
import { isPropertyPrimary, getSortedPrimaryProperties } from '@/libs/deviceTypeHelper';
import { deviceHasAlias } from '@/libs/deviceIdHelper';
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
  const classCode = getDeviceClassCode(device);

  // Get primary properties in sorted order (Operation Status first)
  const primaryProps = getSortedPrimaryProperties(device);
  const secondaryProps = Object.entries(device.properties).filter(([epc]) => 
    !isPropertyPrimary(epc, classCode)
  );

  const PropertyRow = ({ epc, value, isCompact = false }: { 
    epc: string; 
    value: PropertyValue; 
    isCompact?: boolean;
  }) => {
    const [showHexData, setShowHexData] = useState(false);
    const propertyName = getPropertyName(epc, propertyDescriptions, classCode);
    const propertyDescriptor = getPropertyDescriptor(epc, propertyDescriptions, classCode);
    const formattedValue = formatPropertyValueWithTranslation(value, propertyDescriptor, epc, translateLocationId);

    // Check if property is settable
    const hasEditCapability = propertyDescriptor?.stringSettable || propertyDescriptor?.numberDesc || (propertyDescriptor?.aliases && Object.keys(propertyDescriptor.aliases).length > 0);
    const isInSetPropertyMap = propertyDescriptor && isPropertySettable(epc, device);
    const isSettable = hasEditCapability && isInSetPropertyMap;
    const canShowHexViewer = shouldShowHexViewer(value, propertyDescriptor);

    if (isCompact) {
      return (
        <div className="text-xs relative">
          <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <span className="font-medium text-muted-foreground">
              {propertyName}:
            </span>
            <div className="ml-auto">
              {isSettable ? (
                <PropertyEditor
                  device={device}
                  epc={epc}
                  currentValue={value}
                  descriptor={propertyDescriptor}
                  onPropertyChange={onPropertyChange}
                />
              ) : (
                <div className="flex items-center gap-1">
                  <span className="font-medium">
                    {formattedValue}
                  </span>
                  {canShowHexViewer && (
                    <Button
                      variant={showHexData ? "default" : "outline"}
                      size="sm"
                      onClick={() => setShowHexData(!showHexData)}
                      className="h-4 w-4 p-0"
                      title={showHexData ? "Hide hex data" : "Show hex data"}
                    >
                      <Binary className="h-2 w-2" />
                    </Button>
                  )}
                </div>
              )}
            </div>
          </div>
          {showHexData && canShowHexViewer && value.EDT && (
            <div className="absolute top-full left-0 right-0 mt-1 text-xs font-mono bg-muted p-1 rounded border break-all shadow-md z-10">
              {edtToHexString(value.EDT) || 'Invalid data'}
            </div>
          )}
        </div>
      );
    }

    return (
      <div className="space-y-1 relative">
        <div className="flex flex-wrap items-start gap-x-2 gap-y-1">
          <span className="text-sm font-medium text-muted-foreground">
            {propertyName}:
          </span>
          <div className="ml-auto">
            {isSettable ? (
              <PropertyEditor
                device={device}
                epc={epc}
                currentValue={value}
                descriptor={propertyDescriptor}
                onPropertyChange={onPropertyChange}
              />
            ) : (
              <div className="flex items-center gap-2">
                <div className="text-sm font-medium">
                  {formattedValue}
                </div>
                {canShowHexViewer && (
                  <Button
                    variant={showHexData ? "default" : "outline"}
                    size="sm"
                    onClick={() => setShowHexData(!showHexData)}
                    className="h-6 w-6 p-0"
                    title={showHexData ? "Hide hex data" : "Show hex data"}
                  >
                    <Binary className="h-3 w-3" />
                  </Button>
                )}
              </div>
            )}
          </div>
        </div>
        {showHexData && canShowHexViewer && value.EDT && (
          <div className="absolute top-full left-0 right-0 mt-1 text-xs font-mono bg-muted p-2 rounded border break-all shadow-md z-10">
            {edtToHexString(value.EDT) || 'Invalid data'}
          </div>
        )}
      </div>
    );
  };

  return (
    <Card className="transition-all duration-200 w-full max-w-sm flex flex-col" data-testid={`device-card-${device.ip}-${device.eoj}`}>
      <CardHeader className="pb-2 px-3 pt-3">
        <div className="flex items-start justify-between gap-2">
          <div className="space-y-1 flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <CardTitle className="text-sm font-semibold truncate" data-testid="device-title">
                {aliasInfo.aliasName || device.name}
              </CardTitle>
              <DeviceStatusIndicators device={device} />
            </div>
            {aliasInfo.hasAlias && isExpanded && (
              <p className="text-xs text-muted-foreground truncate">
                Device: {device.name}
              </p>
            )}
            {isExpanded && (
              <>
                <p className="text-xs text-muted-foreground">
                  {device.ip} - {device.eoj}
                </p>
                {onAddAlias && onDeleteAlias && device.id && (
                  <div className="mt-2">
                    <AliasEditor
                      device={device}
                      currentAlias={aliasInfo.aliasName}
                      onAddAlias={onAddAlias}
                      onDeleteAlias={onDeleteAlias}
                      isLoading={isAliasLoading}
                    />
                  </div>
                )}
              </>
            )}
          </div>
          <div className="flex items-center gap-1 shrink-0">
            {aliasInfo.hasAlias && (
              <Badge variant="secondary" className="text-xs px-1">
                Alias
              </Badge>
            )}
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
      
      <CardContent className="pt-0 px-3 pb-0 flex flex-col flex-1">
        {/* Main content area that grows to fill space */}
        <div className="flex-1">
          {/* Always show primary properties in compact form */}
          {primaryProps.length > 0 && (
            <div className={`${isExpanded ? 'space-y-3' : 'space-y-0.5'} ${!isExpanded ? 'mb-2' : 'mb-3'}`}>
              {primaryProps.map(([epc, value]) => (
                <PropertyRow key={epc} epc={epc} value={value as PropertyValue} isCompact={!isExpanded} />
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
                  <PropertyRow key={epc} epc={epc} value={value} />
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