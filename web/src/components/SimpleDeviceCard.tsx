import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { GripVertical, Plus, Minus } from 'lucide-react';
import { getDeviceAliases } from '@/libs/deviceIdHelper';
import { formatPropertyValue, getPropertyDescriptor } from '@/libs/propertyHelper';
import type { Device, DeviceAlias, PropertyDescriptionData } from '@/hooks/types';

interface SimpleDeviceCardProps {
  deviceKey: string;
  device: Device;
  allDevices: Record<string, Device>;
  aliases: DeviceAlias;
  propertyDescriptions: Record<string, PropertyDescriptionData>;
  getDeviceClassCode: (device: Device) => string;
  isDraggable?: boolean;
  onDragStart?: (e: React.DragEvent, deviceKey: string) => void;
  onDragEnd?: () => void;
  isDragging?: boolean;
  isLoading?: boolean;
  actionButton?: {
    type: 'add' | 'remove' | 'custom';
    onClick: () => void;
    icon?: React.ReactNode;
    title?: string;
    disabled?: boolean;
  };
  className?: string;
  isCompact?: boolean;
}

export function SimpleDeviceCard({
  deviceKey,
  device,
  allDevices,
  aliases,
  propertyDescriptions,
  getDeviceClassCode,
  isDraggable = false,
  onDragStart,
  onDragEnd,
  isDragging = false,
  isLoading = false,
  actionButton,
  className = '',
  isCompact = false,
}: SimpleDeviceCardProps) {
  const { aliases: deviceAliases } = getDeviceAliases(device, allDevices, aliases);
  const locationProperty = device.properties['81'];
  
  // Get device class code and property descriptor like original DeviceCard does
  const classCode = getDeviceClassCode(device);
  const propertyDescriptor = getPropertyDescriptor('81', propertyDescriptions, classCode);
  
  // Format installation location using the same method as original DeviceCard
  const locationDisplay = locationProperty 
    ? formatPropertyValue(locationProperty, propertyDescriptor)
    : '';
  
  // Use device.name for better readability (e.g., "0EF0[Node Profile]")
  const deviceDisplayName = device.name || device.eoj;
  
  return (
    <Card
      data-testid={`device-card-${deviceKey.replace(/\s+/g, '-')}`}
      draggable={isDraggable && !isLoading}
      onDragStart={isDraggable ? (e) => onDragStart?.(e, deviceKey) : undefined}
      onDragEnd={isDraggable ? onDragEnd : undefined}
      className={`transition-opacity overflow-hidden ${
        isDragging ? 'opacity-50' : ''
      } ${isLoading ? 'cursor-not-allowed opacity-50' : ''} ${className}`}
    >
      <CardContent className="p-3">
        <div className="flex items-start gap-2">
          {isDraggable && (
            <GripVertical className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
          )}
          <div className="flex-1 min-w-0 space-y-2">
            {/* Primary identification */}
            <div className="space-y-1">
              {deviceAliases.length > 0 ? (
                <div className={`text-sm font-medium ${isCompact ? 'truncate' : ''}`}>{deviceAliases[0]}</div>
              ) : (
                <div className={`text-sm font-medium ${isCompact ? 'truncate' : ''}`}>{deviceDisplayName}</div>
              )}
              {deviceAliases.length > 0 && !isCompact ? (
                <div className={`text-xs text-muted-foreground ${isCompact ? 'truncate' : ''}`}>{deviceDisplayName}</div>
              ) : !deviceAliases.length ? (
                <div className={`text-xs text-muted-foreground ${isCompact ? 'truncate' : ''}`}>{device.ip} {device.eoj}</div>
              ) : null}
              {/* Installation location display */}
              {locationDisplay && (
                <div className={`text-xs text-muted-foreground ${isCompact ? 'truncate' : ''}`} aria-label="Installation location">
                  設置場所: {locationDisplay}
                </div>
              )}
            </div>
            
            {/* Device info badges */}
            <div className="flex flex-wrap gap-1">
              {deviceAliases.length > 1 && (
                <Badge variant="outline" className="text-xs">
                  +{deviceAliases.length - 1}
                </Badge>
              )}
            </div>
          </div>
          
          {/* Action Button */}
          {actionButton && (
            <div className="flex-shrink-0">
              <Button
                variant="outline"
                size="sm"
                onClick={actionButton.onClick}
                disabled={actionButton.disabled || isLoading}
                className="h-8 w-8 p-0"
                title={actionButton.title}
                data-testid={`${actionButton.type}-device-${deviceKey.replace(/\s+/g, '-')}`}
              >
                {actionButton.icon || (
                  actionButton.type === 'add' ? <Plus className="h-4 w-4" /> :
                  actionButton.type === 'remove' ? <Minus className="h-4 w-4" /> :
                  null
                )}
              </Button>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}