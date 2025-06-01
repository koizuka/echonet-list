// Device sorting utilities for consistent display order

import type { Device } from '@/hooks/types';

/**
 * Gets the installation location from device properties (EPC 0x81)
 * Returns empty string if not available
 */
function getInstallationLocation(device: Device): string {
  const installationLocation = device.properties['81'];
  if (installationLocation?.string) {
    return installationLocation.string;
  }
  return '';
}

/**
 * Sorts devices by classCode:instance as primary key and installation location as secondary key
 * 
 * Primary sort: EOJ (classCode:instance) - e.g., "0130:1" comes before "0130:2", "0130:2" comes before "0290:1"
 * Secondary sort: Installation location alphabetically
 */
export function sortDevicesByEOJAndLocation(devices: Device[]): Device[] {
  return [...devices].sort((a, b) => {
    // Primary sort by EOJ (classCode:instance)
    const eojCompare = a.eoj.localeCompare(b.eoj);
    if (eojCompare !== 0) {
      return eojCompare;
    }
    
    // Secondary sort by installation location
    const locationA = getInstallationLocation(a);
    const locationB = getInstallationLocation(b);
    const locationCompare = locationA.localeCompare(locationB);
    if (locationCompare !== 0) {
      return locationCompare;
    }
    
    // Tertiary sort by IP address for consistent ordering
    return a.ip.localeCompare(b.ip);
  });
}

/**
 * Sorts devices with custom comparison logic
 * Useful for more complex sorting requirements
 */
export function sortDevicesWithComparator(
  devices: Device[], 
  compareFn: (a: Device, b: Device) => number
): Device[] {
  return [...devices].sort(compareFn);
}