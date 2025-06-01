// Utility functions for managing device locations

import type { Device, DeviceAlias } from '@/hooks/types';
import { getDeviceIdentifierForAlias } from './deviceIdHelper';

// ECHONET Installation Location EPC
const EPC_INSTALLATION_LOCATION = '81';

// ECHONET Installation Location mapping (based on ECHONET Lite spec)
const INSTALLATION_LOCATION_NAMES: Record<string, string> = {
  'living': 'Living Room',
  'dining': 'Dining Room', 
  'kitchen': 'Kitchen',
  'bathroom': 'Bathroom',
  'lavatory': 'Lavatory',
  'washroom': 'Washroom',
  'passageway': 'Passageway',
  'room': 'Room',
  'storeroom': 'Storeroom',
  'entrance': 'Entrance',
  'storage': 'Storage',
  'garden': 'Garden',
  'garage': 'Garage',
  'balcony': 'Balcony',
  'others': 'Others',
  'unspecified': 'Unspecified',
  'undetermined': 'Undetermined'
};

/**
 * Extract location from device's installation location property (EPC 0x81)
 * Falls back to alias-based extraction if installation location is not available
 */
export function extractLocationFromDevice(
  device: Device, 
  aliases: DeviceAlias,
  allDevices?: Record<string, Device>
): string {
  // First priority: Check Installation Location property (EPC 0x81)
  const installationLocationProperty = device.properties[EPC_INSTALLATION_LOCATION];
  if (installationLocationProperty?.string) {
    const locationKey = installationLocationProperty.string.toLowerCase();
    const locationName = INSTALLATION_LOCATION_NAMES[locationKey];
    if (locationName) {
      return locationName;
    }
    // If it's a valid string but not in our mapping, use it as-is (capitalized)
    return installationLocationProperty.string.charAt(0).toUpperCase() + 
           installationLocationProperty.string.slice(1);
  }

  // Second priority: Extract from device alias
  let aliasName: string | undefined;
  if (allDevices) {
    const deviceIdentifier = getDeviceIdentifierForAlias(device, allDevices);
    aliasName = Object.entries(aliases).find(([, id]) => id === deviceIdentifier)?.[0];
  } else {
    // Fallback to old logic if allDevices not provided
    aliasName = Object.entries(aliases).find(([, id]) => id === device.id)?.[0];
  }
  
  if (aliasName) {
    // Try to extract location from alias
    // Look for patterns like "Location - Device" or "Location Device"
    const dashMatch = aliasName.match(/^([^-]+)\s*-\s*/);
    if (dashMatch) {
      return dashMatch[1].trim();
    }
    
    // Look for common location keywords at the beginning
    const locationKeywords = [
      'Living Room', 'Kitchen', 'Bedroom', 'Bathroom', 'Office', 
      'Dining Room', 'Garage', 'Basement', 'Attic', 'Guest Room',
      'Master Bedroom', 'Kids Room', 'Study', 'Laundry', 'Hallway',
      // Japanese locations
      'リビング', 'キッチン', '寝室', 'お風呂', 'トイレ', '書斎', 
      '玄関', '廊下', '洗面所', '子供部屋', '主寝室'
    ];
    
    for (const keyword of locationKeywords) {
      if (aliasName.toLowerCase().startsWith(keyword.toLowerCase())) {
        return keyword;
      }
    }
    
    // If no pattern matches, use the first word as location
    const firstWord = aliasName.split(/[\s-]+/)[0];
    if (firstWord && firstWord.length > 1) {
      return firstWord;
    }
  }
  
  // Third priority: try to extract from device name
  if (device.name && device.name !== device.eoj) {
    const locationFromName = device.name.split(/[\s-]+/)[0];
    if (locationFromName && locationFromName.length > 1) {
      return locationFromName;
    }
  }
  
  // Ultimate fallback
  return 'Unknown';
}

/**
 * Group devices by their extracted locations
 */
export function groupDevicesByLocation(
  devices: Record<string, Device>,
  aliases: DeviceAlias
): Record<string, Device[]> {
  const grouped: Record<string, Device[]> = {};
  
  Object.values(devices).forEach(device => {
    const location = extractLocationFromDevice(device, aliases, devices);
    if (!grouped[location]) {
      grouped[location] = [];
    }
    grouped[location].push(device);
  });
  
  return grouped;
}

/**
 * Get all unique locations from devices, sorted alphabetically
 * with "All" as the first option
 */
export function getAllLocations(
  devices: Record<string, Device>,
  aliases: DeviceAlias
): string[] {
  const locations = new Set<string>();
  
  Object.values(devices).forEach(device => {
    const location = extractLocationFromDevice(device, aliases, devices);
    locations.add(location);
  });
  
  const sortedLocations = Array.from(locations).sort();
  return ['All', ...sortedLocations];
}

/**
 * Get display name for device (alias if available, otherwise device name)
 */
export function getDeviceDisplayName(
  device: Device,
  aliases: DeviceAlias,
  allDevices?: Record<string, Device>
): string {
  let aliasName: string | undefined;
  if (allDevices) {
    const deviceIdentifier = getDeviceIdentifierForAlias(device, allDevices);
    aliasName = Object.entries(aliases).find(([, id]) => id === deviceIdentifier)?.[0];
  } else {
    // Fallback to old logic if allDevices not provided
    aliasName = Object.entries(aliases).find(([, id]) => id === device.id)?.[0];
  }
  
  return aliasName || device.name || device.eoj;
}