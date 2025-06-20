// Utility functions for managing device locations

import type { Device, DeviceAlias, DeviceGroup } from '@/hooks/types';
import { getDeviceIdentifierForAlias } from './deviceIdHelper';
import { sortDevicesByEOJAndLocation } from './deviceSortHelper';
import { isDeviceOperational, isOperationStatusSettable, isDeviceFaulty } from './propertyHelper';
import { isNodeProfileDevice } from './deviceTypeHelper';
import { getCurrentLocale } from './languageHelper';

// ECHONET Installation Location EPC
const EPC_INSTALLATION_LOCATION = '81';

// ECHONET Installation Location mapping - English (based on ECHONET Lite spec)
const INSTALLATION_LOCATION_NAMES_EN: Record<string, string> = {
  'living': 'Living Room',
  'dining': 'Dining Room', 
  'kitchen': 'Kitchen',
  'bathroom': 'Bathroom',
  'lavatory': 'Lavatory',
  'washroom': 'Washroom',
  'passageway': 'Passageway',
  'room': 'Room',
  'staircase': 'Staircase',
  'entrance': 'Entrance',
  'storage': 'Storage',
  'garden': 'Garden',
  'garage': 'Garage',
  'balcony': 'Balcony',
  'others': 'Others',
  'unspecified': 'Unspecified',
  'undetermined': 'Undetermined'
};

// ECHONET Installation Location mapping - Japanese
const INSTALLATION_LOCATION_NAMES_JA: Record<string, string> = {
  'living': 'リビング',
  'dining': 'ダイニング', 
  'kitchen': 'キッチン',
  'bathroom': '浴室',
  'lavatory': 'トイレ',
  'washroom': '洗面所',
  'passageway': '廊下',
  'room': '部屋',
  'staircase': '階段室',
  'entrance': '玄関',
  'storage': '納戸',
  'garden': '庭',
  'garage': 'ガレージ',
  'balcony': 'バルコニー',
  'others': 'その他',
  'unspecified': '未指定',
  'undetermined': '未定'
};

/**
 * Get installation location names based on current locale
 */
export function getInstallationLocationNames(): Record<string, string> {
  const locale = getCurrentLocale();
  return locale === 'ja' ? INSTALLATION_LOCATION_NAMES_JA : INSTALLATION_LOCATION_NAMES_EN;
}


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
    const locationString = installationLocationProperty.string.toLowerCase();
    const installationLocationNames = getInstallationLocationNames();
    
    // Extract base location key and any trailing number
    const match = locationString.match(/^([a-z]+)(\d*)$/);
    if (match) {
      const [, baseKey, number] = match;
      const locationName = installationLocationNames[baseKey];
      if (locationName) {
        // If there's a number, append it to the location name
        return number ? `${locationName} ${number}` : locationName;
      }
    }
    
    // Try exact match if pattern doesn't match
    const locationName = installationLocationNames[locationString];
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
 * Excludes Node Profile devices from location grouping
 */
export function groupDevicesByLocation(
  devices: Record<string, Device>,
  aliases: DeviceAlias
): Record<string, Device[]> {
  const grouped: Record<string, Device[]> = {};
  
  Object.values(devices).forEach(device => {
    // Skip Node Profile devices for location grouping
    if (isNodeProfileDevice(device)) {
      return;
    }
    
    const location = extractLocationFromDevice(device, aliases, devices);
    if (!grouped[location]) {
      grouped[location] = [];
    }
    grouped[location].push(device);
  });
  
  return grouped;
}

/**
 * Extract raw installation location value from device (used for tab IDs)
 */
export function extractRawLocationFromDevice(device: Device): string {
  const installationLocationProperty = device.properties[EPC_INSTALLATION_LOCATION];
  return installationLocationProperty?.string || 'unknown';
}

/**
 * Get all unique locations from devices, sorted alphabetically
 * with "All" as the first option
 * Excludes Node Profile devices from location detection
 * Returns location IDs for internal use
 */
export function getAllLocations(
  devices: Record<string, Device>
): string[] {
  const locationIds = new Set<string>();
  
  Object.values(devices).forEach(device => {
    // Skip Node Profile devices for location detection
    if (isNodeProfileDevice(device)) {
      return;
    }
    
    const locationId = extractRawLocationFromDevice(device);
    locationIds.add(locationId);
  });
  
  const sortedLocationIds = Array.from(locationIds).sort();
  return ['All', ...sortedLocationIds];
}


/**
 * Translate location ID to display name
 */
export function translateLocationId(locationId: string): string {
  if (locationId === 'All') {
    return locationId;
  }
  
  const locationNames = getInstallationLocationNames();
  const lowerLocationId = locationId.toLowerCase();
  
  // Try direct match
  if (locationNames[lowerLocationId]) {
    return locationNames[lowerLocationId];
  }
  
  // Try to extract base location and number (e.g., "living2" -> "Living Room 2")
  const match = lowerLocationId.match(/^([a-z]+)(\d*)$/);
  if (match) {
    const [, baseKey, number] = match;
    const locationName = locationNames[baseKey];
    if (locationName) {
      return number ? `${locationName} ${number}` : locationName;
    }
  }
  
  // Return capitalized ID if no translation found
  return locationId.charAt(0).toUpperCase() + locationId.slice(1);
}

/**
 * Get all tabs including locations and device groups
 * Device groups are prefixed with "@" to distinguish from locations
 * Returns tab IDs for internal use
 */
export function getAllTabs(
  devices: Record<string, Device>,
  groups: DeviceGroup
): string[] {
  // Get location tab IDs
  const locationTabIds = getAllLocations(devices);
  
  // Get group tab IDs (prefixed with "@")
  const groupTabIds = Object.keys(groups)
    .filter(groupName => groupName.startsWith('@'))
    .sort();
  
  // Combine: All, locations, then groups
  return [...locationTabIds, ...groupTabIds];
}


/**
 * Group devices by their raw location IDs
 * Excludes Node Profile devices from location grouping
 */
export function groupDevicesByLocationId(
  devices: Record<string, Device>
): Record<string, Device[]> {
  const grouped: Record<string, Device[]> = {};
  
  Object.values(devices).forEach(device => {
    // Skip Node Profile devices for location grouping
    if (isNodeProfileDevice(device)) {
      return;
    }
    
    const locationId = extractRawLocationFromDevice(device);
    if (!grouped[locationId]) {
      grouped[locationId] = [];
    }
    grouped[locationId].push(device);
  });
  
  return grouped;
}

/**
 * Get devices for a specific tab (location or group)
 * Returns devices sorted by EOJ (classCode:instance) and installation location
 * For location tabs, excludes Node Profile devices. For 'All' tab, includes all devices.
 * Takes tab ID as input (raw location ID or group name)
 */
export function getDevicesForTab(
  tabId: string,
  devices: Record<string, Device>,
  groups: DeviceGroup
): Device[] {
  let filteredDevices: Device[];
  
  if (tabId === 'All') {
    // Show all devices including Node Profile in the 'All' tab
    filteredDevices = Object.values(devices);
  } else if (tabId.startsWith('@')) {
    // It's a group tab (starts with "@")
    const groupDeviceIds = groups[tabId] || [];
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
    // It's a location tab - use raw location ID
    const groupedDevices = groupDevicesByLocationId(devices);
    filteredDevices = groupedDevices[tabId] || [];
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

/**
 * Check if any device in the array has operation status "on"
 * Only considers devices where Operation Status (EPC 0x80) is settable
 */
export function hasAnyOperationalDevice(devices: Device[]): boolean {
  return devices.some(device => 
    isOperationStatusSettable(device) && isDeviceOperational(device)
  );
}

/**
 * Check if any device in the array has a fault
 * Returns true if at least one device has a fault status
 */
export function hasAnyFaultyDevice(devices: Device[]): boolean {
  return devices.some(device => isDeviceFaulty(device));
}