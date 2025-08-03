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
 * Detects if an IP address is IPv4 or IPv6
 * Returns 'ipv4', 'ipv6', or null for invalid addresses
 */
function detectIPVersion(ip: string): 'ipv4' | 'ipv6' | null {
  // IPv4 pattern: 4 groups of 1-3 digits separated by dots
  if (/^(\d{1,3}\.){3}\d{1,3}$/.test(ip)) {
    return 'ipv4';
  }
  
  // IPv6 patterns (simplified - doesn't validate all edge cases)
  // Full form: 8 groups of 1-4 hex digits separated by colons
  // Compressed form: contains ::
  // IPv4-mapped: ends with IPv4 address
  if (ip.includes(':')) {
    return 'ipv6';
  }
  
  return null;
}

/**
 * Parses an IPv4 address into an array of numbers
 * Returns null if invalid
 */
function parseIPv4(ip: string): number[] | null {
  const parts = ip.split('.');
  if (parts.length !== 4) return null;
  
  const octets = parts.map(part => {
    const num = parseInt(part, 10);
    return (isNaN(num) || num < 0 || num > 255) ? null : num;
  });
  
  return octets.includes(null) ? null : octets as number[];
}

/**
 * Normalizes an IPv6 address to its full form for comparison
 * Returns null if invalid
 */
function normalizeIPv6(ip: string): string | null {
  try {
    // Remove square brackets if present (e.g., [::1])
    const cleanIP = ip.replace(/^\[|\]$/g, '');
    
    // Handle IPv4-mapped IPv6 addresses (e.g., ::ffff:192.168.1.1)
    const ipv4Match = cleanIP.match(/::ffff:(\d+\.\d+\.\d+\.\d+)$/i);
    if (ipv4Match) {
      const ipv4Parts = parseIPv4(ipv4Match[1]);
      if (!ipv4Parts) return null;
      // Convert to full IPv6 form
      const hex = ipv4Parts.map(n => n.toString(16).padStart(2, '0'));
      return `0000:0000:0000:0000:0000:ffff:${hex[0]}${hex[1]}:${hex[2]}${hex[3]}`.toLowerCase();
    }
    
    // Split by :: to handle compressed notation
    const parts = cleanIP.split('::');
    if (parts.length > 2) return null; // Invalid: more than one ::
    
    let groups: string[] = [];
    
    if (parts.length === 1) {
      // No compression
      groups = parts[0].split(':');
      if (groups.length !== 8) return null;
    } else {
      // Has compression
      const leftGroups = parts[0] ? parts[0].split(':') : [];
      const rightGroups = parts[1] ? parts[1].split(':') : [];
      const missingGroups = 8 - leftGroups.length - rightGroups.length;
      
      if (missingGroups < 0) return null; // Too many groups
      
      groups = [
        ...leftGroups,
        ...Array(missingGroups).fill('0'),
        ...rightGroups
      ];
    }
    
    // Validate and normalize each group
    const normalized = groups.map(group => {
      if (!/^[0-9a-fA-F]{0,4}$/.test(group)) return null;
      return group.padStart(4, '0').toLowerCase();
    });
    
    return normalized.includes(null) ? null : normalized.join(':');
  } catch {
    return null;
  }
}

/**
 * Compares two IP addresses (IPv4 or IPv6)
 * Returns negative if a < b, positive if a > b, 0 if equal
 * Invalid IPs are treated as greater than valid ones
 * IPv4 addresses sort before IPv6 addresses
 */
function compareIPAddresses(a: string, b: string): number {
  const versionA = detectIPVersion(a);
  const versionB = detectIPVersion(b);
  
  // Handle invalid IPs
  if (!versionA && !versionB) return a.localeCompare(b); // Both invalid
  if (!versionA) return 1; // Invalid IPs sort after valid ones
  if (!versionB) return -1;
  
  // IPv4 sorts before IPv6
  if (versionA !== versionB) {
    return versionA === 'ipv4' ? -1 : 1;
  }
  
  if (versionA === 'ipv4') {
    // Compare IPv4 addresses
    const octetsA = parseIPv4(a);
    const octetsB = parseIPv4(b);
    
    if (!octetsA || !octetsB) return a.localeCompare(b); // Shouldn't happen
    
    for (let i = 0; i < 4; i++) {
      if (octetsA[i] !== octetsB[i]) {
        return octetsA[i] - octetsB[i];
      }
    }
    return 0;
  } else {
    // Compare IPv6 addresses
    const normalizedA = normalizeIPv6(a);
    const normalizedB = normalizeIPv6(b);
    
    if (!normalizedA || !normalizedB) return a.localeCompare(b); // Shouldn't happen
    
    // Compare normalized forms lexicographically (they're now same length and format)
    return normalizedA.localeCompare(normalizedB);
  }
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
    return compareIPAddresses(a.ip, b.ip);
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