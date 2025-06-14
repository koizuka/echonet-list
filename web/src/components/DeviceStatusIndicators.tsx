import { isDeviceOperational, isDeviceFaulty, isOperationStatusSettable } from '@/libs/propertyHelper';
import type { Device } from '@/hooks/types';

interface DeviceStatusIndicatorsProps {
  device: Device;
}

export function DeviceStatusIndicators({ device }: DeviceStatusIndicatorsProps) {
  const isOperational = isDeviceOperational(device);
  const isFaulty = isDeviceFaulty(device);
  const canSetOperationStatus = isOperationStatusSettable(device);

  return (
    <div className="flex items-center gap-2">
      {/* Operation Status Indicator - only show if Operation Status is settable */}
      {device.properties['80'] && canSetOperationStatus && (
        <div 
          className={`w-3 h-3 rounded-full ${
            isOperational ? 'bg-green-500' : 'bg-gray-400'
          }`}
          title={`Operation Status: ${isOperational ? 'ON' : 'OFF'}`}
        />
      )}
      
      {/* Fault Status Indicator */}
      {isFaulty && (
        <div 
          className="w-3 h-3 rounded-full bg-red-500"
          title="Fault detected"
        />
      )}
    </div>
  );
}