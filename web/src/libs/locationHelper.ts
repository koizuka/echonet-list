// Utility functions for managing device locations

import type { Device, DeviceAlias, DeviceGroup, PropertyDescriptionData } from '@/hooks/types';
import { getDeviceIdentifierForAlias } from './deviceIdHelper';
import { sortDevicesByEOJAndLocation } from './deviceSortHelper';
import { isDeviceOperational, isOperationStatusSettable, isDeviceFaulty, getPropertyDescriptor, formatPropertyValue } from './propertyHelper';
import { isNodeProfileDevice } from './deviceTypeHelper';

// ECHONET Installation Location EPC
const EPC_INSTALLATION_LOCATION = '81';



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
    // Return the raw string value - translation will be handled by formatPropertyValue
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
 * @deprecated Use getLocationDisplayName instead which uses server-side translations
 */
export function translateLocationId(locationId: string): string {
  if (locationId === 'All') {
    return locationId;
  }
  
  // Return capitalized ID - translation is handled by server
  return locationId.charAt(0).toUpperCase() + locationId.slice(1);
}

/**
 * Get display name for a location tab using server-side translations
 * Looks for a device with the given location ID and uses its translated value
 */
export function getLocationDisplayName(
  locationId: string,
  devices: Record<string, Device>,
  propertyDescriptions: Record<string, PropertyDescriptionData>,
  lang?: string
): string {
  if (locationId === 'All') {
    return locationId;
  }
  
  // Find a device with this location ID to get the translated value
  const devicesInLocation = Object.values(devices).filter(device => {
    const rawLocation = extractRawLocationFromDevice(device);
    return rawLocation === locationId;
  });
  
  if (devicesInLocation.length > 0) {
    // Get the first device and use its translated location value
    const device = devicesInLocation[0];
    const installationLocationProperty = device.properties[EPC_INSTALLATION_LOCATION];
    
    if (installationLocationProperty?.string) {
      // Get property descriptor for Installation Location
      const classCode = device.eoj.split(':')[0];
      const descriptor = getPropertyDescriptor(
        EPC_INSTALLATION_LOCATION,
        propertyDescriptions,
        classCode,
        lang
      );
      
      // Format the value with translations
      const translatedValue = formatPropertyValue(
        installationLocationProperty,
        descriptor,
        lang
      );
      
      if (translatedValue && translatedValue !== 'Raw data') {
        return translatedValue;
      }
    }
  }
  
  // Fallback to capitalized ID
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
      
      // Check exact matches first
      if (groupDeviceIds.includes(deviceIdentifier) || 
          (directDeviceId && groupDeviceIds.includes(directDeviceId))) {
        return true;
      }
      
      return false;
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