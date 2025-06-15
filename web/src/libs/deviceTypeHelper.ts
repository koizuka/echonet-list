// Device type specific property definitions for compact display

/**
 * Essential properties that should always be shown in compact view
 * These are common across all device types
 * Order matters: Operation Status should always be first
 */
export const ESSENTIAL_PROPERTIES = ['80', '81'] as const; // Operation Status, Installation Location

/**
 * Device type specific primary properties for compact display
 * Key: EOJ class code (first 4 characters)
 * Value: Array of EPC codes to display in compact view
 */
export const DEVICE_PRIMARY_PROPERTIES: Record<string, string[]> = {
  // Home Air Conditioner (0130)
  '0130': ['B0', 'B3', 'B4', 'BA', 'BB', 'BE'], // Operation mode, Temperature, Humidity, Target temp, etc.
  
  // Floor Heating (027B)
  '027B': ['E1', 'E2', 'F3', 'F4'], // Various temperature sensors
  
  // Add more device types as needed
  // Single Function Lighting (0291)
  '0291': ['B0'], // Illuminance level
  
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
 * Ensures Operation Status (0x80) is always first if present
 */
export function getSortedPrimaryProperties(device: { properties: Record<string, unknown>; eoj: string }): [string, unknown][] {
  const classCode = device.eoj.split(':')[0];
  const primaryProperties = getDevicePrimaryProperties(classCode);
  
  // Filter to only properties that exist on the device
  const availableProps = Object.entries(device.properties).filter(([epc]) => 
    primaryProperties.includes(epc)
  );
  
  // Sort by priority: Operation Status first, then Installation Location, then others
  return availableProps.sort(([epcA], [epcB]) => {
    if (epcA === '80') return -1; // Operation Status always first
    if (epcB === '80') return 1;
    if (epcA === '81') return -1; // Installation Location second
    if (epcB === '81') return 1;
    return epcA.localeCompare(epcB); // Alphabetical for others
  });
}