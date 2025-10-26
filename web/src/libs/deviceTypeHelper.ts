// Device type specific property definitions for compact display

import type { PropertyValue, Device } from '@/hooks/types';

/**
 * Visibility condition for properties in compact mode
 */
export type PropertyVisibilityCondition = {
  epc: string;           // EPC of the property to potentially hide
  hideWhen: {
    epc: string;         // EPC of the condition property
    values?: number[];   // Hide when condition property equals any of these values
    notValues?: number[]; // Hide when condition property does not equal any of these values
  };
};

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
  // Node Profile (0EF0)
  '0EF0': [
    'D6', // SelfNodeInstanceListS
  ],

  // Home Air Conditioner (0130)
  '0130': [
    'BB', 'BA', // temperature, humidity
    'BE', // outside temperature
    'B0', // operation mode setting
    'B3', // temperature setting
    'B4', // relative humidity setting
    // 'A0', // air volume setting
    // 'A3', // air direction setting
  ],

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

  // Refrigerator (03b7)
  '03B7': ['89', 'B0', 'B1', 'B2', 'B3'], // Fault description, Door open status, Door open alert status, Refrigerator door open status, Freezer door open status
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
export function getDeviceSecondaryProperties(device: { properties: Record<string, PropertyValue>; eoj: string }): string[] {
  const classCode = device.eoj.split(':')[0];
  const primaryProperties = getDevicePrimaryProperties(classCode);

  return Object.keys(device.properties).filter(epc => !primaryProperties.includes(epc));
}

/**
 * Device type and property combinations that should use immediate slider control
 * Key: Device class code
 * Value: Array of EPC codes that should use immediate slider
 */
export const IMMEDIATE_SLIDER_PROPERTIES: Record<string, string[]> = {
  // Single Function Lighting (0291)
  '0291': ['B0'], // Illuminance level
};

/**
 * Checks if a property should use immediate slider control
 *
 * Determines whether a specific property for a given device class should
 * display an immediate slider interface instead of the traditional edit button.
 *
 * @param epc - The property EPC code (e.g., 'B0' for illuminance)
 * @param classCode - The device class code (e.g., '0291' for Single Function Lighting)
 * @returns true if the property should show a slider without edit mode
 */
export function shouldUseImmediateSlider(epc: string, classCode: string): boolean {
  const immediateSliderEPCs = IMMEDIATE_SLIDER_PROPERTIES[classCode] || [];
  // Normalize EPC to uppercase for comparison
  const normalizedEPC = epc.toUpperCase();
  return immediateSliderEPCs.includes(normalizedEPC);
}

/**
 * Property visibility conditions for compact mode
 * Defines when certain properties should be hidden based on other property values
 * Key: Device class code
 * Value: Array of visibility conditions
 */
export const PROPERTY_VISIBILITY_CONDITIONS: Record<string, PropertyVisibilityCondition[]> = {
  // Home Air Conditioner (0130)
  '0130': [
    {
      epc: 'B3', // Temperature setting
      hideWhen: {
        epc: 'B0', // Operation mode setting
        values: [0x41, 0x45] // Hide when mode is auto (0x41) or fan (0x45)
      }
    },
    {
      epc: 'B4', // Relative humidity setting for dehumidification mode
      hideWhen: {
        epc: 'B0', // Operation mode setting
        notValues: [0x44] // Hide when mode is NOT dry (0x44)
      }
    }
  ]
};

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
export function getSortedPrimaryProperties(device: { properties: Record<string, PropertyValue>; eoj: string }): [string, PropertyValue][] {
  const classCode = device.eoj.split(':')[0];
  const primaryProperties = getDevicePrimaryProperties(classCode);

  // Filter to only properties that exist on the device and maintain the order
  return primaryProperties
    .filter(epc => epc in device.properties)
    .map(epc => [epc, device.properties[epc]] as [string, PropertyValue]);
}

/**
 * Extracts numeric value from a property
 * Tries to get the number from EDT field (Base64 decoded) or number field
 *
 * @param property - The property value object
 * @returns The numeric value or undefined if not available
 */
function getPropertyNumericValue(property: PropertyValue): number | undefined {
  // First try the number field (for numeric properties like temperature)
  if (typeof property.number === 'number') {
    return property.number;
  }

  // Try to decode EDT field (for properties with aliases like operation mode)
  if (property.EDT && typeof property.EDT === 'string') {
    try {
      // Decode Base64 to get raw bytes
      const decoded = atob(property.EDT);
      // Get the first byte as the numeric value
      if (decoded.length > 0) {
        return decoded.charCodeAt(0);
      }
    } catch {
      // If decoding fails, continue
    }
  }

  return undefined;
}

/**
 * Determines if a property should be shown in compact mode
 * Checks visibility conditions for the device class and property
 *
 * @param epc - The property EPC code to check
 * @param device - The device object containing all properties
 * @param classCode - The device class code
 * @returns true if the property should be displayed in compact mode
 */
export function shouldShowPropertyInCompactMode(epc: string, device: Device, classCode: string): boolean {
  // Get visibility conditions for this device class
  const conditions = PROPERTY_VISIBILITY_CONDITIONS[classCode];

  // If no conditions defined for this device class, always show
  if (!conditions) {
    return true;
  }

  // Find conditions for this specific property
  const propertyConditions = conditions.filter(condition => condition.epc === epc);

  // If no conditions for this property, always show
  if (propertyConditions.length === 0) {
    return true;
  }

  // Check each condition - if ANY condition says to hide, then hide
  for (const condition of propertyConditions) {
    const conditionProperty = device.properties[condition.hideWhen.epc];

    // If condition property doesn't exist, show the property
    if (!conditionProperty) {
      continue;
    }

    // Get the condition property value (from number field or EDT field)
    const conditionValue = getPropertyNumericValue(conditionProperty);

    // If condition value is not available, show the property
    if (conditionValue === undefined) {
      continue;
    }

    // Check condition types
    const { values, notValues } = condition.hideWhen;

    // Check 'values' condition (hide if condition property equals any of these values)
    if (values !== undefined && values.includes(conditionValue)) {
      return false;
    }

    // Check 'notValues' condition (hide if condition property does NOT equal any of these values)
    if (notValues !== undefined && !notValues.includes(conditionValue)) {
      return false;
    }
  }

  // If no conditions matched to hide, show the property
  return true;
}