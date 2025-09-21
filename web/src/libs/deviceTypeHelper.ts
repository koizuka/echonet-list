// Device type specific property definitions for compact display

import type { PropertyValue } from '@/hooks/types';

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