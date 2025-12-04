import { Card } from '@/components/ui/card';
import { DeviceIcon } from '@/components/DeviceIcon';
import { PropertySwitchControl } from './PropertyEditControls/PropertySwitchControl';
import { getDashboardStatusProperties } from '@/libs/deviceTypeHelper';
import { formatPropertyValue, getPropertyDescriptor, isPropertySettable } from '@/libs/propertyHelper';
import { deviceHasAlias } from '@/libs/deviceIdHelper';
import type { Device, PropertyDescriptionData, DeviceAlias } from '@/hooks/types';

interface DashboardCardProps {
  device: Device;
  onPropertyChange: (target: string, epc: string, value: { string: string }) => Promise<void>;
  propertyDescriptions: Record<string, PropertyDescriptionData>;
  devices: Record<string, Device>;
  aliases: DeviceAlias;
  isConnected?: boolean;
}

export function DashboardCard({
  device,
  onPropertyChange,
  propertyDescriptions,
  devices,
  aliases,
  isConnected = true
}: DashboardCardProps) {
  const classCode = device.eoj.split(':')[0];
  const aliasInfo = deviceHasAlias(device, devices, aliases);
  const deviceName = aliasInfo.aliasName || device.name;

  // Get operation status for on/off control
  const operationStatus = device.properties['80'];
  const isOperationSettable = isPropertySettable('80', device);

  // Get dashboard status properties and format their values
  const statusEpcs = getDashboardStatusProperties(classCode) || [];
  const statusValues = statusEpcs
    .map(epc => {
      const property = device.properties[epc];
      if (!property) return null;
      const descriptor = getPropertyDescriptor(epc, propertyDescriptions, classCode);
      return formatPropertyValue(property, descriptor);
    })
    .filter((v): v is string => v !== null);

  // Join status values with separator
  const statusDisplay = statusValues.length > 0 ? statusValues.join(' / ') : '---';

  // Determine card styling based on device status
  const isOperational = operationStatus?.string === 'on';
  const isOffline = device.isOffline || false;

  const handlePowerChange = async (value: string) => {
    await onPropertyChange(`${device.ip} ${device.eoj}`, '80', { string: value });
  };

  return (
    <Card
      className={`p-2 border-2 ${isOffline ? 'opacity-50' : ''} ${isOperational ? 'border-green-500/60' : 'border-border'}`}
      data-testid={`dashboard-card-${device.ip}-${device.eoj}`}
    >
      {/* Line 1: Device name */}
      <div className="flex items-center gap-2 mb-1">
        <DeviceIcon device={device} classCode={classCode} className="flex-shrink-0" />
        <span className="text-sm font-medium truncate" title={deviceName}>
          {deviceName}
        </span>
      </div>

      {/* Line 2: Status + On/Off control */}
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs text-muted-foreground truncate flex-1">
          {statusDisplay}
        </span>

        {isOperationSettable && operationStatus && (
          <PropertySwitchControl
            value={operationStatus.string || 'off'}
            onChange={handlePowerChange}
            disabled={!isConnected || isOffline}
            testId={`dashboard-power-${device.ip}-${device.eoj}`}
          />
        )}
      </div>
    </Card>
  );
}
