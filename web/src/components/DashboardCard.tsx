import { Card } from '@/components/ui/card';
import { DeviceIcon } from '@/components/DeviceIcon';
import { PropertySwitchControl } from './PropertyEditControls/PropertySwitchControl';
import { getDashboardStatusProperties } from '@/libs/deviceTypeHelper';
import { formatPropertyValue, getPropertyDescriptor, isPropertySettable } from '@/libs/propertyHelper';
import { isTemperatureSensor, getTemperatureColor } from '@/libs/sensorPropertyHelper';
import { deviceHasAlias } from '@/libs/deviceIdHelper';
import { cn } from '@/libs/utils';
import type { Device, PropertyDescriptionData, DeviceAlias, PropertyValue } from '@/hooks/types';

interface StatusItem {
  value: string;
  colorClass: string;
}

interface DashboardCardProps {
  device: Device;
  onPropertyChange: (target: string, epc: string, value: { string: string }) => Promise<void>;
  propertyDescriptions: Record<string, PropertyDescriptionData>;
  devices: Record<string, Device>;
  aliases: DeviceAlias;
  isConnected?: boolean;
  isExpanded?: boolean;
  onToggleExpand?: () => void;
}

export function DashboardCard({
  device,
  onPropertyChange,
  propertyDescriptions,
  devices,
  aliases,
  isConnected = true,
  isExpanded = false,
  onToggleExpand
}: DashboardCardProps) {
  const classCode = device.eoj.split(':')[0];
  const aliasInfo = deviceHasAlias(device, devices, aliases);
  const deviceName = aliasInfo.aliasName || device.name;

  // Get operation status for on/off control
  const operationStatus = device.properties['80'];
  const isOperationSettable = isPropertySettable('80', device);

  // Get dashboard status properties and format their values with colors
  const statusEpcs = getDashboardStatusProperties(classCode) || [];
  const getStatusColorClass = (epc: string, property: PropertyValue): string => {
    if (isTemperatureSensor(classCode, epc) && property.number !== undefined) {
      return getTemperatureColor(property.number);
    }
    return 'text-muted-foreground';
  };

  const statusItems: StatusItem[] = statusEpcs
    .map(epc => {
      const property = device.properties[epc];
      if (!property) return null;
      const descriptor = getPropertyDescriptor(epc, propertyDescriptions, classCode);
      const value = formatPropertyValue(property, descriptor);
      if (!value) return null;
      return {
        value,
        colorClass: getStatusColorClass(epc, property)
      };
    })
    .filter((v): v is StatusItem => v !== null);

  // Determine card styling based on device status
  const isOperational = operationStatus?.string === 'on';
  const isOffline = device.isOffline || false;

  const handlePowerChange = async (value: string) => {
    try {
      await onPropertyChange(`${device.ip} ${device.eoj}`, '80', { string: value });
    } catch (error) {
      console.error('Failed to change power state:', error);
    }
  };

  return (
    <Card
      className={cn(
        'py-1 px-2 border-2',
        isOffline && 'opacity-50',
        isOperational ? 'border-green-500/60' : 'border-border'
      )}
      data-testid={`dashboard-card-${device.ip}-${device.eoj}`}
    >
      {/* Line 1: Icon + Status + On/Off control */}
      <div className="flex items-center justify-between gap-2">
        {/* Expandable area: Icon + Status */}
        <div
          className="flex items-center gap-2 flex-1 min-w-0 cursor-pointer"
          onClick={onToggleExpand}
          onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); onToggleExpand?.(); } }}
          onKeyUp={(e) => { if (e.key === ' ') { e.preventDefault(); onToggleExpand?.(); } }}
          role="button"
          tabIndex={0}
          aria-expanded={isExpanded}
          aria-label={`${deviceName}: ${isExpanded ? 'collapse' : 'expand'}`}
          data-testid={`dashboard-card-expandable-${device.ip}-${device.eoj}`}
        >
          <DeviceIcon device={device} classCode={classCode} className="flex-shrink-0" />
          <span className="text-xs truncate flex-1">
            {statusItems.length > 0 ? (
              statusItems.map((item, index) => (
                <span key={index}>
                  {index > 0 && <span className="text-muted-foreground" aria-hidden="true"> / </span>}
                  <span className={item.colorClass} aria-label={`Status: ${item.value}`}>{item.value}</span>
                </span>
              ))
            ) : (
              <span className="text-muted-foreground" aria-label="No status data">---</span>
            )}
          </span>
        </div>

        {isOperationSettable && operationStatus && (
          <PropertySwitchControl
            value={operationStatus.string || 'off'}
            onChange={handlePowerChange}
            disabled={!isConnected || isOffline}
            testId={`dashboard-power-${device.ip}-${device.eoj}`}
            compact
          />
        )}
      </div>

      {/* Line 2: Device name (only when expanded) */}
      {isExpanded && (
        <div className="text-xs text-muted-foreground truncate mt-0.5" title={deviceName}>
          {deviceName}
        </div>
      )}
    </Card>
  );
}
