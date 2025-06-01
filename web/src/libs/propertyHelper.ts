// Utility functions for working with ECHONET property descriptions

import type { PropertyDescriptionData, PropertyDescriptor } from '@/hooks/types';

/**
 * Gets the human-readable property name for an EPC
 * Falls back to "EPC {epc}" if no description is found
 */
export function getPropertyName(
  epc: string,
  propertyDescriptions: Record<string, PropertyDescriptionData>,
  classCode?: string
): string {
  // Try to find the property description for the specific class code
  if (classCode && propertyDescriptions[classCode]) {
    const property = propertyDescriptions[classCode].properties[epc];
    if (property?.description) {
      return property.description;
    }
  }

  // Try to find in common properties (classCode "")
  const commonProperties = propertyDescriptions[""];
  if (commonProperties?.properties[epc]?.description) {
    return commonProperties.properties[epc].description;
  }

  // Fallback to EPC hex display
  return `EPC ${epc}`;
}

/**
 * Gets the property descriptor for an EPC
 * Returns undefined if not found
 */
export function getPropertyDescriptor(
  epc: string,
  propertyDescriptions: Record<string, PropertyDescriptionData>,
  classCode?: string
): PropertyDescriptor | undefined {
  // Try to find the property description for the specific class code
  if (classCode && propertyDescriptions[classCode]) {
    const property = propertyDescriptions[classCode].properties[epc];
    if (property) {
      return property;
    }
  }

  // Try to find in common properties (classCode "")
  const commonProperties = propertyDescriptions[""];
  if (commonProperties?.properties[epc]) {
    return commonProperties.properties[epc];
  }

  return undefined;
}

/**
 * Extracts the class code from an EOJ string (e.g., "01:30:01" -> "0130")
 */
export function extractClassCodeFromEOJ(eoj: string): string {
  const parts = eoj.split(':');
  if (parts.length >= 2) {
    return parts[0] + parts[1];
  }
  return '';
}

/**
 * Formats a property value using aliases if available
 */
export function formatPropertyValue(
  value: { EDT?: string; string?: string; number?: number },
  descriptor?: PropertyDescriptor
): string {
  // If we have a string representation, use it
  if (value.string) {
    return value.string;
  }

  // If we have a number, format it with unit if available
  if (value.number !== undefined) {
    const unit = descriptor?.numberDesc?.unit || '';
    return `${value.number}${unit}`;
  }

  // If we have EDT but no string, try to decode using aliases
  if (value.EDT && descriptor?.aliases) {
    try {
      const edtBytes = atob(value.EDT);
      // Find matching alias
      for (const [aliasName, aliasEDT] of Object.entries(descriptor.aliases)) {
        if (atob(aliasEDT) === edtBytes) {
          return aliasName;
        }
      }
    } catch {
      // Ignore decode errors
    }
  }

  return 'Raw data';
}