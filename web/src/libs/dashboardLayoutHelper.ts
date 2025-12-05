// Dashboard layout helper for special air conditioner / floor heater arrangement

import type { Device } from '@/hooks/types';

// Placeholder type for empty grid slots
export type Placeholder = { type: 'placeholder' };

// Layout item can be either a Device or a Placeholder
export type DashboardLayoutItem = Device | Placeholder;

/**
 * Type guard to check if an item is a placeholder
 */
export function isPlaceholder(item: DashboardLayoutItem): item is Placeholder {
  return 'type' in item && item.type === 'placeholder';
}

// Device class codes for special layout handling
const CLASS_AIR_CONDITIONER = '0130';
const CLASS_FLOOR_HEATER = '027B';

/**
 * Extract class code from EOJ (format: "classCode:instance")
 */
function getClassCode(device: Device): string {
  return device.eoj.split(':')[0];
}

/**
 * Check if device is an air conditioner
 */
function isAirConditioner(device: Device): boolean {
  return getClassCode(device) === CLASS_AIR_CONDITIONER;
}

/**
 * Check if device is a floor heater
 */
function isFloorHeater(device: Device): boolean {
  return getClassCode(device) === CLASS_FLOOR_HEATER;
}

/**
 * Sort devices by EOJ (classCode:instance)
 */
function sortByEOJ(devices: Device[]): Device[] {
  return [...devices].sort((a, b) => a.eoj.localeCompare(b.eoj));
}

/**
 * Create a placeholder for empty grid slots
 */
function createPlaceholder(): Placeholder {
  return { type: 'placeholder' };
}

/**
 * Arrange devices for Dashboard grid layout with special handling for
 * air conditioners (class 0130) and floor heaters (class 027B).
 *
 * Layout rules:
 * - If AC or floor heater exists, pair them: left = AC, right = floor heater
 * - If counts don't match, use placeholders for empty slots
 * - Other devices follow after the AC/FH pairs in normal EOJ order
 */
export function arrangeDashboardDevices(devices: Device[]): DashboardLayoutItem[] {
  if (devices.length === 0) {
    return [];
  }

  // Categorize devices
  const airConditioners = sortByEOJ(devices.filter(isAirConditioner));
  const floorHeaters = sortByEOJ(devices.filter(isFloorHeater));
  const others = sortByEOJ(devices.filter(d => !isAirConditioner(d) && !isFloorHeater(d)));

  // If no AC or floor heaters, return others in sorted order
  if (airConditioners.length === 0 && floorHeaters.length === 0) {
    return others;
  }

  const result: DashboardLayoutItem[] = [];

  // Create paired rows for AC and floor heaters
  const maxPairs = Math.max(airConditioners.length, floorHeaters.length);

  for (let i = 0; i < maxPairs; i++) {
    // Left column: air conditioner or placeholder
    if (i < airConditioners.length) {
      result.push(airConditioners[i]);
    } else {
      result.push(createPlaceholder());
    }

    // Right column: floor heater or placeholder
    if (i < floorHeaters.length) {
      result.push(floorHeaters[i]);
    } else {
      result.push(createPlaceholder());
    }
  }

  // Add remaining devices
  result.push(...others);

  return result;
}
