import type { LucideIcon } from 'lucide-react';
import {
  AirVent,
  Heater,
  Lightbulb,
  LampCeiling,
  ThermometerSun,
  Refrigerator,
  Settings,
  Info,
  CircleHelp
} from 'lucide-react';

/**
 * Mapping of ECHONET Lite device class codes to Lucide icons
 */
export const DEVICE_CLASS_ICONS: Record<string, LucideIcon> = {
  // Home Air Conditioner
  '0130': AirVent,
  
  // Floor Heating
  '027B': Heater,
  
  // Single Function Lighting
  '0291': Lightbulb,
  
  // Lighting System
  '02A3': LampCeiling,
  
  // Electric Water Heater
  '026B': ThermometerSun,
  
  // Bath Room Heating and Air Conditioning
  '0272': Heater,
  
  // Refrigerator
  '03B7': Refrigerator,
  
  // Controller
  '05FF': Settings,
  
  // Node Profile
  '0EF0': Info,
};

/**
 * Get the appropriate icon for a device based on its class code
 * @param classCode The ECHONET Lite device class code
 * @returns The corresponding Lucide icon component or a default icon
 */
export function getDeviceIcon(classCode: string): LucideIcon {
  return DEVICE_CLASS_ICONS[classCode] || CircleHelp;
}

/**
 * Get the icon color based on device status
 * @param isOperational Whether the device is operational (ON)
 * @param isFaulty Whether the device has a fault
 * @param isOffline Whether the device is offline
 * @param isControllable Whether the device is user-controllable
 * @returns The appropriate Tailwind CSS color class
 */
export function getDeviceIconColor(
  isOperational: boolean,
  isFaulty: boolean,
  isOffline: boolean,
  isControllable: boolean = true
): string {
  // For non-controllable devices, only show offline and fault states
  if (!isControllable) {
    if (isOffline) {
      return 'text-muted-foreground';
    }
    if (isFaulty) {
      return 'text-red-500';
    }
    return 'text-gray-400'; // No status coloring for non-controllable devices
  }
  
  // For controllable devices, show full status coloring
  if (isOffline) {
    return 'text-muted-foreground';
  }
  if (isFaulty) {
    return 'text-red-500';
  }
  if (isOperational) {
    return 'text-green-500';
  }
  return 'text-gray-400';
}