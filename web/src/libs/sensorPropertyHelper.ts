import {
  Thermometer,
  CloudSun,
  Home,
  ThermometerSun,
  ThermometerSnowflake,
  Droplets,
  type LucideIcon
} from 'lucide-react';

// Sensor property EPCs and their corresponding icons
const SENSOR_PROPERTY_ICONS: Record<string, LucideIcon> = {
  // Room temperature (Air Conditioner & Floor Heating)
  'BB': Thermometer,
  'E2': Thermometer,
  
  // Outside temperature (Air Conditioner)
  'BE': CloudSun,
  
  // Floor temperature (Floor Heating)
  'E3': Home,
  
  // Temperature sensor 1 (Floor Heating - outgoing water temp)
  'F3': ThermometerSun,
  
  // Temperature sensor 2 (Floor Heating - return water temp)
  'F4': ThermometerSnowflake,
  
  // Room humidity (Air Conditioner)
  'BA': Droplets,
};

/**
 * Checks if a property EPC is a sensor property
 */
export function isSensorProperty(epc: string): boolean {
  return epc in SENSOR_PROPERTY_ICONS;
}

/**
 * Gets the icon component for a sensor property
 * Returns undefined if the property is not a sensor
 */
export function getSensorIcon(epc: string): LucideIcon | undefined {
  return SENSOR_PROPERTY_ICONS[epc];
}

/**
 * Gets all sensor EPCs as an array
 */
export function getSensorEPCs(): string[] {
  return Object.keys(SENSOR_PROPERTY_ICONS);
}