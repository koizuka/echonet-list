// Device type specific property definitions for compact display

/**
 * Essential properties that should always be shown in compact view
 * These are common across all device types
 * Order matters: Operation Status should always be first
 */
export const ESSENTIAL_PROPERTIES = ['80'] as const; // Operation Status

/**
 * Device type specific primary properties for compact display
 * Key: EOJ class code (first 4 characters)
 * Value: Array of EPC codes to display in compact view
 */
export const DEVICE_PRIMARY_PROPERTIES: Record<string, string[]> = {
  // Home Air Conditioner (0130)
  '0130': ['BB', 'BA', 'BE', 'B0', 'B3', 'B4', 'A0', 'A3'], // Target temp, Target humidity, Target flow, Operation mode, Temperature, Humidity, Air flow, etc.
  
  // Floor Heating (027B)
  '027B': ['E2', 'F3', 'F4', 'E1'], // Various temperature sensors
  
  // Add more device types as needed
  // Single Function Lighting (0291)
  '0291': ['B0'], // Illuminance level
  
  // Lighting System (02A3)
  '02A3': ['C0'], // Scene control
  
  // Electric Water Heater (026B)
  '026B': ['D1', 'D2'], // Hot water temperature, volume
  
  // Bath Room Heating and Air Conditioning (0272)
  '0272': ['B0', 'B3'], // Operation mode, Temperature
};

/**
 * Gets the primary properties for a device based on its class code
 * Returns essential properties plus device-specific primary properties
 */
export function getDevicePrimaryProperties(classCode: string): string[] {
  const essentialProps = [...ESSENTIAL_PROPERTIES];
  const deviceSpecificProps = DEVICE_PRIMARY_PROPERTIES[classCode] || [];
  
  // Combine essential and device-specific properties, removing duplicates
  return [...new Set([...essentialProps, ...deviceSpecificProps])];
}

/**
 * Checks if a property is considered primary for a device
 */
export function isPropertyPrimary(epc: string, classCode: string): boolean {
  const primaryProperties = getDevicePrimaryProperties(classCode);
  return primaryProperties.includes(epc);
}

/**
 * Gets secondary (non-primary) properties for a device
 */
export function getDeviceSecondaryProperties(device: { properties: Record<string, unknown>; eoj: string }): string[] {
  const classCode = device.eoj.split(':')[0];
  const primaryProperties = getDevicePrimaryProperties(classCode);
  
  return Object.keys(device.properties).filter(epc => !primaryProperties.includes(epc));
}

/**
 * Checks if a device is a Node Profile device
 * Node Profile devices have class code 0EF0
 */
export function isNodeProfileDevice(device: { eoj: string }): boolean {
  const classCode = device.eoj.split(':')[0];
  return classCode === '0EF0';
}

/**
 * Gets primary properties in the correct display order
 * Ensures properties are displayed in the order they are defined
 */
export function getSortedPrimaryProperties(device: { properties: Record<string, unknown>; eoj: string }): [string, unknown][] {
  const classCode = device.eoj.split(':')[0];
  const primaryProperties = getDevicePrimaryProperties(classCode);
  
  // Filter to only properties that exist on the device and maintain the order
  return primaryProperties
    .filter(epc => epc in device.properties)
    .map(epc => [epc, device.properties[epc]] as [string, unknown]);
}