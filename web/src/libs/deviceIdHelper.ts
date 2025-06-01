// Device ID generation utilities that match the Go backend logic

import type { Device } from '@/hooks/types';

// ECHONET Node Profile Object EOJ
const NODE_PROFILE_OBJECT_EOJ = '0EF0:1';

// EPC for Identification Number
const EPC_IDENTIFICATION_NUMBER = '83';

/**
 * Generate EOJ ID string (6-digit hex) from EOJ string
 * Converts "0130:1" to "013001"
 */
function generateEOJIDString(eoj: string): string {
  const parts = eoj.split(':');
  if (parts.length >= 2) {
    const classCode = parts[0]; // e.g., "0130"
    const instanceCode = parseInt(parts[1], 10); // e.g., 1
    return `${classCode}${instanceCode.toString(16).padStart(2, '0').toUpperCase()}`;
  }
  return '';
}

/**
 * Parse IdentificationNumber from hex string to manufacturer:unique format
 * Matches Go backend's IdentificationNumber.String() method
 * Format: FE + 3-byte manufacturer + 13-byte unique identifier
 * Returns: "manufacturer:unique" (e.g., "000005:00112233445566778899AABBCCDD")
 */
function parseIdentificationNumber(hexString: string): string {
  // Expected format: FE + 6 hex digits (manufacturer) + 26 hex digits (unique)
  // Total: 2 + 6 + 26 = 34 hex characters
  if (hexString.length !== 34 || !hexString.startsWith('FE')) {
    return hexString; // Return as-is if format doesn't match
  }
  
  // Extract manufacturer code (3 bytes = 6 hex chars)
  const manufacturerCode = hexString.substring(2, 8);
  
  // Extract unique identifier (13 bytes = 26 hex chars)
  const uniqueIdentifier = hexString.substring(8);
  
  return `${manufacturerCode}:${uniqueIdentifier}`;
}

/**
 * Get device identifier for alias matching
 * This matches the Go backend's GetIDString() logic:
 * 1. Find NodeProfileObject device for the same IP
 * 2. Get IdentificationNumber (EPC 83) from NPO
 * 3. Parse IdentificationNumber to manufacturer:unique format
 * 4. Generate ID as: EOJ.IDString() + ":" + parsed IdentificationNumber
 */
export function getDeviceIdentifierForAlias(
  device: Device,
  allDevices: Record<string, Device>
): string {
  // Find NodeProfileObject device for the same IP
  const npoKey = `${device.ip} ${NODE_PROFILE_OBJECT_EOJ}`;
  const npoDevice = allDevices[npoKey];
  
  if (!npoDevice) {
    // No NodeProfileObject found, fallback to device.id
    return device.id;
  }

  // Get IdentificationNumber (EPC 83) from NPO device
  const identificationNumberProp = npoDevice.properties[EPC_IDENTIFICATION_NUMBER];
  if (!identificationNumberProp?.string) {
    // No IdentificationNumber property, fallback to device.id
    return device.id;
  }

  // Generate EOJ ID string (6-digit hex)
  const eojIdString = generateEOJIDString(device.eoj);
  if (!eojIdString) {
    // Invalid EOJ format, fallback to device.id
    return device.id;
  }

  // Parse IdentificationNumber to match Go backend format
  const parsedIdNumber = parseIdentificationNumber(identificationNumberProp.string);

  // Generate device identifier: EOJ.IDString() + ":" + IdentificationNumber.String()
  return `${eojIdString}:${parsedIdNumber}`;
}

/**
 * Get display name for device using correct device identifier for alias matching
 */
export function getDeviceDisplayNameWithCorrectId(
  device: Device,
  allDevices: Record<string, Device>,
  aliases: Record<string, string>
): string {
  const deviceIdentifier = getDeviceIdentifierForAlias(device, allDevices);
  const aliasName = Object.entries(aliases).find(([, id]) => id === deviceIdentifier)?.[0];
  
  return aliasName || device.name || device.eoj;
}

/**
 * Check if device has an alias using correct device identifier
 */
export function deviceHasAlias(
  device: Device,
  allDevices: Record<string, Device>,
  aliases: Record<string, string>
): { hasAlias: boolean; aliasName?: string; deviceIdentifier: string } {
  const deviceIdentifier = getDeviceIdentifierForAlias(device, allDevices);
  const aliasName = Object.entries(aliases).find(([, id]) => id === deviceIdentifier)?.[0];
  
  return {
    hasAlias: !!aliasName,
    aliasName,
    deviceIdentifier
  };
}