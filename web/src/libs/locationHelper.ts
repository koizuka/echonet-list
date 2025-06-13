// Utility functions for managing device locations

import type { Device, DeviceAlias, DeviceGroup } from '@/hooks/types';
import { getDeviceIdentifierForAlias } from './deviceIdHelper';
import { sortDevicesByEOJAndLocation } from './deviceSortHelper';

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
    if (deviceIdentifier) {
      aliasName = Object.entries(aliases).find(([, id]) => id === deviceIdentifier)?.[0];
    }
  } else {
    // Fallback to old logic if allDevices not provided
    // Only try alias lookup if device.id is defined
    if (device.id) {
      aliasName = Object.entries(aliases).find(([, id]) => id === device.id)?.[0];
    }
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
 * Get all tabs including locations and device groups
 * Device groups are prefixed with "@" to distinguish from locations
 */
export function getAllTabs(
  devices: Record<string, Device>,
  aliases: DeviceAlias,
  groups: DeviceGroup
): string[] {
  // Get location tabs
  const locationTabs = getAllLocations(devices, aliases);
  
  // Get group tabs (prefixed with "@")
  const groupTabs = Object.keys(groups)
    .filter(groupName => groupName.startsWith('@'))
    .sort();
  
  // Combine: All, locations, then groups
  return [...locationTabs, ...groupTabs];
}

/**
 * Get devices for a specific tab (location or group)
 * Returns devices sorted by EOJ (classCode:instance) and installation location
 */
export function getDevicesForTab(
  tabName: string,
  devices: Record<string, Device>,
  aliases: DeviceAlias,
  groups: DeviceGroup
): Device[] {
  let filteredDevices: Device[];
  
  if (tabName === 'All') {
    filteredDevices = Object.values(devices);
  } else if (tabName.startsWith('@')) {
    // It's a group tab (starts with "@")
    const groupDeviceIds = groups[tabName] || [];
    filteredDevices = Object.values(devices).filter(device => {
      // Generate device identifier using same logic as aliases
      const deviceIdentifier = getDeviceIdentifierForAlias(device, devices);
      
      // Skip devices without valid identifier
      if (deviceIdentifier === undefined) {
        return false;
      }
      
      // Also check device.id directly as fallback (if defined)
      const directDeviceId = device.id;
      
      // Extract EOJ part from device for partial matching
      const deviceEOJPart = deviceIdentifier.split(':')[0]; // e.g., "027B04"
      
      // Check exact matches first
      if (groupDeviceIds.includes(deviceIdentifier) || 
          (directDeviceId && groupDeviceIds.includes(directDeviceId))) {
        return true;
      }
      
      // Check if any group device ID starts with the same EOJ
      return groupDeviceIds.some(groupId => groupId.startsWith(deviceEOJPart + ':'));
    });
  } else {
    // It's a location tab
    const groupedDevices = groupDevicesByLocation(devices, aliases);
    filteredDevices = groupedDevices[tabName] || [];
  }
  
  // Sort all devices by EOJ (classCode:instance) and installation location
  return sortDevicesByEOJAndLocation(filteredDevices);
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
    if (deviceIdentifier) {
      aliasName = Object.entries(aliases).find(([, id]) => id === deviceIdentifier)?.[0];
    }
  } else {
    // Fallback to old logic if allDevices not provided
    // Only try alias lookup if device.id is defined
    if (device.id) {
      aliasName = Object.entries(aliases).find(([, id]) => id === device.id)?.[0];
    }
  }
  
  return aliasName || device.name || device.eoj;
}