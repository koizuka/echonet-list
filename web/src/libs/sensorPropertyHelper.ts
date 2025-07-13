import {
  Thermometer,
  CloudSun,
  Home,
  ThermometerSun,
  ThermometerSnowflake,
  Droplets,
  type LucideIcon
} from 'lucide-react';
import type { PropertyValue } from '@/hooks/types';

// Sensor property EPCs and their corresponding icons using classCode:EPC format
const SENSOR_PROPERTY_ICONS: Record<string, LucideIcon> = {
  // Air Conditioner (0130) sensors
  '0130:BB': Thermometer,  // Room temperature
  '0130:BE': CloudSun,     // Outside temperature
  '0130:BA': Droplets,     // Room humidity
  
  // Floor Heating (027B) sensors
  '027B:E2': Thermometer,        // Room temperature
  '027B:E3': Home,               // Floor temperature
  '027B:F3': ThermometerSun,     // Temperature sensor 1 (outgoing water temp)
  '027B:F4': ThermometerSnowflake, // Temperature sensor 2 (return water temp)
};

// Temperature sensor EPCs (for color calculation)
const TEMPERATURE_SENSOR_EPCS = new Set([
  '0130:BB', '0130:BE',  // Air Conditioner temperature sensors
  '027B:E2', '027B:E3', '027B:F3', '027B:F4'  // Floor Heating temperature sensors
]);

/**
 * Gets the temperature color class based on the temperature value
 * Only applies to temperature sensors, returns muted for others
 */
function getTemperatureColor(value: number): string {
  if (value <= 10) return 'text-blue-600';      // Very cold
  if (value <= 15) return 'text-blue-400';      // Cold
  if (value <= 24) return 'text-muted-foreground'; // Normal
  if (value <= 29) return 'text-orange-400';    // Warm
  return 'text-red-600';                         // Hot
}

/**
 * Checks if a property EPC is a sensor property
 */
export function isSensorProperty(classCode: string, epc: string): boolean {
  const key = `${classCode}:${epc}`;
  return key in SENSOR_PROPERTY_ICONS;
}

/**
 * Gets the icon component for a sensor property
 * Returns undefined if the property is not a sensor
 */
export function getSensorIcon(classCode: string, epc: string): LucideIcon | undefined {
  const key = `${classCode}:${epc}`;
  return SENSOR_PROPERTY_ICONS[key];
}

/**
 * Checks if a property is a temperature sensor
 */
export function isTemperatureSensor(classCode: string, epc: string): boolean {
  const key = `${classCode}:${epc}`;
  return TEMPERATURE_SENSOR_EPCS.has(key);
}

/**
 * Gets the color class for a sensor icon based on its value
 * Only temperature sensors get color-coded, others use muted color
 */
export function getSensorIconColor(classCode: string, epc: string, value: PropertyValue): string {
  // Only apply color to sensor properties
  if (!isSensorProperty(classCode, epc)) {
    return 'text-muted-foreground';
  }
  
  // Only temperature sensors get color-coded
  if (!isTemperatureSensor(classCode, epc)) {
    return 'text-muted-foreground';
  }
  
  // Must have a numeric value
  if (value.number === undefined) {
    return 'text-muted-foreground';
  }
  
  return getTemperatureColor(value.number);
}

/**
 * Gets all sensor EPCs as an array with classCode:EPC format
 */
export function getSensorEPCs(): string[] {
  return Object.keys(SENSOR_PROPERTY_ICONS);
}