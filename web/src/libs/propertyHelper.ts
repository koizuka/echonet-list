// Utility functions for working with ECHONET property descriptions

import type { PropertyDescriptionData, PropertyDescriptor, Device } from '@/hooks/types';
import { getCurrentLocale } from './languageHelper';

/**
 * Gets the human-readable property name for an EPC
 * Falls back to "EPC {epc}" if no description is found
 * Supports language-aware lookups using cached property descriptions
 */
export function getPropertyName(
  epc: string,
  propertyDescriptions: Record<string, PropertyDescriptionData>,
  classCode?: string,
  lang?: string
): string {
  const currentLang = lang || getCurrentLocale();

  // Try to find the property description for the specific class code with language
  if (classCode) {
    const langSpecificKey = `${classCode}:${currentLang}`;
    if (propertyDescriptions[langSpecificKey]) {
      const property = propertyDescriptions[langSpecificKey].properties[epc];
      if (property?.description) {
        return property.description;
      }
    }

    // Fallback to English if not found in requested language
    if (propertyDescriptions[classCode]) {
      const property = propertyDescriptions[classCode].properties[epc];
      if (property?.description) {
        return property.description;
      }
    }
  }

  // Try to find in common properties (classCode "") with language
  const commonLangKey = `:${currentLang}`;
  if (propertyDescriptions[commonLangKey]?.properties[epc]?.description) {
    return propertyDescriptions[commonLangKey].properties[epc].description;
  }

  // Fallback to English common properties
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
 * Supports language-aware lookups using cached property descriptions
 */
export function getPropertyDescriptor(
  epc: string,
  propertyDescriptions: Record<string, PropertyDescriptionData>,
  classCode?: string,
  lang?: string
): PropertyDescriptor | undefined {
  const currentLang = lang || getCurrentLocale();

  // Try to find the property description for the specific class code with language
  if (classCode) {
    const langSpecificKey = `${classCode}:${currentLang}`;
    if (propertyDescriptions[langSpecificKey]) {
      const property = propertyDescriptions[langSpecificKey].properties[epc];
      if (property) {
        return property;
      }
    }

    // Fallback to English if not found in requested language
    if (propertyDescriptions[classCode]) {
      const property = propertyDescriptions[classCode].properties[epc];
      if (property) {
        return property;
      }
    }
  }

  // Try to find in common properties (classCode "") with language
  const commonLangKey = `:${currentLang}`;
  if (propertyDescriptions[commonLangKey]?.properties[epc]) {
    return propertyDescriptions[commonLangKey].properties[epc];
  }

  // Fallback to English common properties
  const commonProperties = propertyDescriptions[""];
  if (commonProperties?.properties[epc]) {
    return commonProperties.properties[epc];
  }

  return undefined;
}

/**
 * Extracts the class code from an EOJ string (e.g., "0130:1" -> "0130")
 */
export function extractClassCodeFromEOJ(eoj: string): string {
  const parts = eoj.split(':');
  if (parts.length >= 1 && parts[0].length === 4) {
    return parts[0];
  }
  return '';
}

/**
 * Checks if a property (EPC) is settable according to Set Property Map (EPC 0x9E)
 * Returns true if the property is listed in the Set Property Map
 */
export function isPropertySettable(epc: string, device: Device): boolean {
  // EPC 0x9E contains the Set Property Map
  const setPropertyMap = device.properties['9E'];

  if (!setPropertyMap?.EDT) {
    // If no Set Property Map is available, assume not settable
    return false;
  }

  try {
    // Decode the Base64 EDT to get the property map bytes
    const mapBytes = atob(setPropertyMap.EDT);

    // First byte is the number of properties
    if (mapBytes.length < 1) {
      return false;
    }

    const propertyCount = mapBytes.charCodeAt(0);

    // Check if EPC is in the list
    const epcCode = parseInt(epc, 16);
    for (let i = 1; i <= propertyCount && i < mapBytes.length; i++) {
      if (mapBytes.charCodeAt(i) === epcCode) {
        return true;
      }
    }

    return false;
  } catch (error) {
    console.warn(`Failed to parse Set Property Map for device ${device.ip} ${device.eoj}:`, error);
    return false;
  }
}

/**
 * Formats a property value using aliases if available
 * Supports localized alias translations
 */
export function formatPropertyValue(
  value: { EDT?: string; string?: string; number?: number },
  descriptor?: PropertyDescriptor,
  lang?: string
): string {
  const currentLang = lang || getCurrentLocale();

  // If we have a string representation, use it
  if (value.string) {
    // Try to translate using aliasTranslations if available
    if (descriptor?.aliasTranslations && currentLang !== 'en') {
      const translation = descriptor.aliasTranslations[value.string];
      if (translation) {
        return translation;
      }
    }
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
          // Try to translate the alias name if translations are available
          if (descriptor.aliasTranslations && currentLang !== 'en') {
            const translation = descriptor.aliasTranslations[aliasName];
            if (translation) {
              return translation;
            }
          }
          return aliasName;
        }
      }
    } catch {
      // Ignore decode errors
    }
  }

  return 'Raw data';
}

/**
 * Format property value with localization support
 * This is a wrapper around formatPropertyValue that handles translation for specific EPCs
 */
export function formatPropertyValueWithTranslation(
  value: { EDT?: string; string?: string; number?: number },
  descriptor?: PropertyDescriptor,
  _epc?: string,
  _translateFunc?: (value: string) => string,
  lang?: string
): string {
  // Translation is now handled by formatPropertyValue using server-side aliasTranslations
  return formatPropertyValue(value, descriptor, lang);
}

/**
 * Converts Base64 EDT to hex string representation
 * Returns formatted hex string like "01 23 45" or null if conversion fails
 */
export function edtToHexString(edt: string): string | null {
  if (!edt) return null;

  try {
    const bytes = atob(edt);
    const hexBytes = [];

    for (let i = 0; i < bytes.length; i++) {
      const byte = bytes.charCodeAt(i);
      hexBytes.push(byte.toString(16).toUpperCase().padStart(2, '0'));
    }

    return hexBytes.join(' ');
  } catch (error) {
    console.warn('Failed to convert EDT to hex string:', error);
    return null;
  }
}

/**
 * Checks if a property value should show hex data viewer
 * Returns true if value has EDT but formatted as "Raw data"
 */
export function shouldShowHexViewer(
  value: { EDT?: string; string?: string; number?: number },
  descriptor?: PropertyDescriptor,
  lang?: string
): boolean {
  // Only show for values that have EDT but format as "Raw data"
  if (!value.EDT) return false;

  const formattedValue = formatPropertyValue(value, descriptor, lang);
  return formattedValue === 'Raw data';
}

/**
 * Checks if device is operational (EPC 0x80 Operation Status is on)
 * Returns true if operation status is on, false otherwise
 */
export function isDeviceOperational(device: Device): boolean {
  const operationStatus = device.properties['80'];
  if (!operationStatus) {
    return false;
  }

  // Check if the string value indicates "on"
  if (operationStatus.string) {
    return operationStatus.string.toLowerCase() === 'on';
  }

  // Check by EDT value (0x30 = on, 0x31 = off for most devices)
  if (operationStatus.EDT) {
    try {
      const edtBytes = atob(operationStatus.EDT);
      if (edtBytes.length > 0) {
        const statusByte = edtBytes.charCodeAt(0);
        return statusByte === 0x30; // 0x30 = on
      }
    } catch {
      // Ignore decode errors
    }
  }

  return false;
}

/**
 * Checks if device has a fault (EPC 0x88 Fault occurrence status is not no_fault)
 * Returns true if device has a fault, false otherwise
 */
export function isDeviceFaulty(device: Device): boolean {
  const faultStatus = device.properties['88'];
  if (!faultStatus) {
    return false;
  }

  // Check if the string value indicates fault
  if (faultStatus.string) {
    return faultStatus.string.toLowerCase() !== 'no_fault';
  }

  // Check by EDT value (0x42 = no_fault for most devices)
  if (faultStatus.EDT) {
    try {
      const edtBytes = atob(faultStatus.EDT);
      if (edtBytes.length > 0) {
        const statusByte = edtBytes.charCodeAt(0);
        return statusByte !== 0x42; // 0x42 = no_fault
      }
    } catch {
      // Ignore decode errors
    }
  }

  return false;
}

/**
 * Checks if device's Operation Status (EPC 0x80) is settable
 * Returns true if EPC 0x80 is in the Set Property Map, false otherwise
 */
export function isOperationStatusSettable(device: Device): boolean {
  return isPropertySettable('80', device);
}

/**
 * Decodes ECHONET Lite property map from Base64 EDT data
 * Supports both direct list format (< 16 properties) and bitmap format (>= 16 properties)
 * 
 * @param edt Base64 encoded EDT data
 * @returns Array of EPC codes (hex strings) or null if parsing fails
 */
export function decodePropertyMap(edt: string): string[] | null {
  if (!edt) return null;

  try {
    const mapBytes = atob(edt);
    if (mapBytes.length < 1) return null;

    const propertyCount = mapBytes.charCodeAt(0);
    const epcs: string[] = [];

    if (propertyCount < 16) {
      // Direct list format: properties listed directly after count byte
      for (let i = 1; i <= propertyCount && i < mapBytes.length; i++) {
        const epc = mapBytes.charCodeAt(i).toString(16).toUpperCase().padStart(2, '0');
        epcs.push(epc);
      }
    } else {
      // Bitmap format: 17 bytes of bitmap data
      // Each bit in each byte represents a specific EPC according to Go formula: i + (j << 4) + 0x80
      // where i = byte index (0-15), j = bit index (0-7)
      if (mapBytes.length < 17) {
        console.warn(`Property map has ${propertyCount} properties but insufficient bitmap data`);
        return null;
      }

      for (let i = 0; i < 16; i++) {
        const bitmapByte = mapBytes.charCodeAt(i + 1);

        for (let j = 0; j < 8; j++) {
          if (bitmapByte & (1 << j)) {
            const epc = (i + (j << 4) + 0x80).toString(16).toUpperCase();
            epcs.push(epc);
          }
        }
      }
    }

    // Sort EPCs in ascending order
    epcs.sort((a, b) => a.localeCompare(b));

    return epcs;
  } catch (error) {
    console.warn('Failed to decode property map:', error);
    return null;
  }
}