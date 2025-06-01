import { ChevronDown, ChevronUp } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { PropertyEditor } from '@/components/PropertyEditor';
import { DeviceStatusIndicators } from '@/components/DeviceStatusIndicators';
import { getPropertyName, formatPropertyValue, getPropertyDescriptor } from '@/libs/propertyHelper';
import { isPropertyPrimary, getSortedPrimaryProperties } from '@/libs/deviceTypeHelper';
import { deviceHasAlias } from '@/libs/deviceIdHelper';
import type { Device, PropertyValue, PropertyDescriptionData } from '@/hooks/types';

interface DeviceCardProps {
  device: Device;
  isExpanded: boolean;
  onToggleExpansion: () => void;
  onPropertyChange: (target: string, epc: string, value: PropertyValue) => Promise<void>;
  propertyDescriptions: Record<string, PropertyDescriptionData>;
  getDeviceClassCode: (device: Device) => string;
  devices: Record<string, Device>;
  aliases: Record<string, string>;
}

export function DeviceCard({
  device,
  isExpanded,
  onToggleExpansion,
  onPropertyChange,
  propertyDescriptions,
  getDeviceClassCode,
  devices,
  aliases
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
    const propertyName = getPropertyName(epc, propertyDescriptions, classCode);
    const propertyDescriptor = getPropertyDescriptor(epc, propertyDescriptions, classCode);
    const formattedValue = formatPropertyValue(value, propertyDescriptor);

    if (isCompact) {
      return (
        <div className="flex items-center justify-between gap-1 text-xs">
          <div className="flex items-center gap-1 min-w-0 flex-1">
            <span className="font-medium text-muted-foreground truncate">
              {propertyName}:
            </span>
            <span className="font-medium truncate">
              {formattedValue}
            </span>
          </div>
          <PropertyEditor
            device={device}
            epc={epc}
            currentValue={value}
            descriptor={propertyDescriptor}
            onPropertyChange={onPropertyChange}
          />
        </div>
      );
    }

    return (
      <div className="space-y-1">
        <div className="flex items-start justify-between gap-2">
          <span className="text-sm font-medium text-muted-foreground flex-shrink-0">
            {propertyName}:
          </span>
          <PropertyEditor
            device={device}
            epc={epc}
            currentValue={value}
            descriptor={propertyDescriptor}
            onPropertyChange={onPropertyChange}
          />
        </div>
        <div className="text-sm font-medium break-words">
          {formattedValue}
        </div>
      </div>
    );
  };

  return (
    <Card className="transition-all duration-200 w-full max-w-sm flex flex-col">
      <CardHeader className="pb-2 px-3 pt-3">
        <div className="flex items-start justify-between gap-2">
          <div className="space-y-1 flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <CardTitle className="text-sm font-semibold truncate">
                {aliasInfo.aliasName || device.name}
              </CardTitle>
              <DeviceStatusIndicators device={device} />
            </div>
            {aliasInfo.hasAlias && (
              <p className="text-xs text-muted-foreground truncate">
                Device: {device.name}
              </p>
            )}
            <p className="text-xs text-muted-foreground">
              {device.ip} - {device.eoj}
            </p>
          </div>
          <div className="flex items-center gap-1 flex-shrink-0">
            {aliasInfo.hasAlias && (
              <Badge variant="secondary" className="text-xs px-1">
                Alias
              </Badge>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={onToggleExpansion}
              className="h-6 w-6 p-0"
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