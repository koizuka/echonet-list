import { DEVICE_CLASS_ICONS, DEFAULT_DEVICE_ICON, getDeviceIconColor } from '@/libs/deviceIconHelper';
import { isDeviceOperational, isDeviceFaulty, isOperationStatusSettable } from '@/libs/propertyHelper';
import type { Device } from '@/hooks/types';

interface DeviceIconProps {
  device: Device;
  classCode: string;
  className?: string;
}

export function DeviceIcon({ device, classCode, className = '' }: DeviceIconProps) {
  const IconComponent = DEVICE_CLASS_ICONS[classCode] ?? DEFAULT_DEVICE_ICON;
  const isOperational = isDeviceOperational(device);
  const isFaulty = isDeviceFaulty(device);
  const isControllable = isOperationStatusSettable(device);
  const iconColor = getDeviceIconColor(isOperational, isFaulty, device.isOffline || false, isControllable);
  
  // Get device type name for tooltip
  const deviceTypeName = getDeviceTypeName(classCode);
  const statusText = device.isOffline 
    ? 'Offline' 
    : isFaulty 
    ? 'Error' 
    : isOperational 
    ? 'ON' 
    : 'OFF';
  
  return (
    <div title={`${deviceTypeName} - ${statusText}`}>
      <IconComponent className={`w-4 h-4 ${iconColor} ${className}`} />
    </div>
  );
}

// Helper function to get human-readable device type names
function getDeviceTypeName(classCode: string): string {
  const deviceTypes: Record<string, string> = {
    '0130': 'Air Conditioner',
    '027B': 'Floor Heating',
    '0291': 'Lighting',
    '02A3': 'Lighting System',
    '026B': 'Water Heater',
    '0272': 'Bath Heater',
    '03B7': 'Refrigerator',
    '05FF': 'Controller',
    '0EF0': 'Node Profile',
  };
  
  return deviceTypes[classCode] || 'Unknown Device';
}