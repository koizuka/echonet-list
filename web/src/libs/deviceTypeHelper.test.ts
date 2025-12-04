import { describe, it, expect } from 'vitest';
import {
  getDevicePrimaryProperties,
  isPropertyPrimary,
  getDeviceSecondaryProperties,
  getSortedPrimaryProperties,
  isNodeProfileDevice,
  shouldShowPropertyInCompactMode,
  getDashboardStatusProperties,
  ESSENTIAL_PROPERTIES,
  DEVICE_PRIMARY_PROPERTIES,
  DEVICE_DASHBOARD_PROPERTIES
} from './deviceTypeHelper';
import type { PropertyValue, Device } from '@/hooks/types';

// Test helper functions for creating properly typed property values
const createPropertyValue = (value: Partial<PropertyValue>): PropertyValue => value as PropertyValue;

const createDevice = (eoj: string, properties: Record<string, Partial<PropertyValue>>) => ({
  eoj,
  properties: Object.fromEntries(
    Object.entries(properties).map(([key, value]) => [key, createPropertyValue(value)])
  ) as Record<string, PropertyValue>
});

describe('deviceTypeHelper', () => {
  describe('getDevicePrimaryProperties', () => {
    it('should return essential properties for unknown device types', () => {
      const properties = getDevicePrimaryProperties('9999'); // Unknown class code
      expect(properties).toEqual(['80']); // Essential properties only
    });

    it('should return essential + device-specific properties for air conditioner', () => {
      const properties = getDevicePrimaryProperties('0130'); // Home Air Conditioner
      expect(properties).toContain('80'); // Operation Status (essential)
      expect(properties).toContain('B0'); // Operation mode (device-specific)
      expect(properties).toContain('B3'); // Temperature (device-specific)
    });

    it('should return essential + device-specific properties for lighting', () => {
      const properties = getDevicePrimaryProperties('0291'); // Single Function Lighting
      expect(properties).toContain('80'); // Operation Status (essential)
      expect(properties).toContain('B0'); // Illuminance level (device-specific)
    });

    it('should not contain duplicates', () => {
      const properties = getDevicePrimaryProperties('0130');
      const uniqueProperties = [...new Set(properties)];
      expect(properties).toEqual(uniqueProperties);
    });
  });

  describe('isPropertyPrimary', () => {
    it('should return true for essential properties', () => {
      expect(isPropertyPrimary('80', '9999')).toBe(true); // Operation Status
      expect(isPropertyPrimary('81', '9999')).toBe(false); // Installation Location (not essential anymore)
    });

    it('should return true for device-specific primary properties', () => {
      expect(isPropertyPrimary('B0', '0130')).toBe(true); // Air conditioner operation mode
      expect(isPropertyPrimary('B0', '0291')).toBe(true); // Lighting illuminance
    });

    it('should return false for non-primary properties', () => {
      expect(isPropertyPrimary('9F', '0130')).toBe(false); // Get Property Map
      expect(isPropertyPrimary('FF', '0291')).toBe(false); // Unknown property
    });

    it('should handle case sensitivity', () => {
      expect(isPropertyPrimary('b0', '0130')).toBe(false); // Lowercase should not match
      expect(isPropertyPrimary('B0', '0130')).toBe(true); // Uppercase should match
    });
  });

  describe('getDeviceSecondaryProperties', () => {
    it('should return properties that are not primary', () => {
      const device = createDevice('0130:1', { // Home Air Conditioner
        '80': { string: 'on' },     // Primary (essential)
        '81': { string: 'living' }, // Secondary (not essential anymore)
        'B0': { string: 'cool' },   // Primary (device-specific)
        '9F': { EDT: 'base64' },    // Secondary
        'FF': { string: 'test' }    // Secondary
      });

      const secondaryProps = getDeviceSecondaryProperties(device);
      expect(secondaryProps).toContain('9F');
      expect(secondaryProps).toContain('FF');
      expect(secondaryProps).not.toContain('80');
      expect(secondaryProps).not.toContain('B0');
    });

    it('should return empty array when all properties are primary', () => {
      const device = createDevice('0291:1', { // Single Function Lighting
        '80': { string: 'on' },     // Primary (essential)
        '81': { string: 'bedroom' }, // Secondary (not essential anymore)
        'B0': { number: 50 }        // Primary (device-specific)
      });

      const secondaryProps = getDeviceSecondaryProperties(device);
      expect(secondaryProps).toEqual(['81']); // '81' is now secondary (not essential)
    });

    it('should handle devices with no properties', () => {
      const device = createDevice('0130:1', {});

      const secondaryProps = getDeviceSecondaryProperties(device);
      expect(secondaryProps).toEqual([]);
    });
  });

  describe('constants', () => {
    it('should have correct essential properties', () => {
      expect(ESSENTIAL_PROPERTIES).toEqual(['80']);
    });

    it('should have device-specific properties defined', () => {
      expect(DEVICE_PRIMARY_PROPERTIES['0130']).toContain('B0'); // Air conditioner
      expect(DEVICE_PRIMARY_PROPERTIES['0291']).toContain('B0'); // Lighting
      expect(DEVICE_PRIMARY_PROPERTIES['027B']).toContain('E1'); // Floor heating
    });
  });

  describe('getSortedPrimaryProperties', () => {
    it('should maintain the order defined in primary properties', () => {
      const device = createDevice('0130:1', { // Home Air Conditioner
        'B0': { string: 'cool' },   // Operation mode
        '81': { string: 'living' }, // Installation Location
        '80': { string: 'on' },     // Operation Status
        'B3': { number: 25 },       // Temperature
        'A0': { string: 'auto' },   // Air flow rate
        'A3': { string: 'swing' },  // Air flow direction
        'BA': { number: 24 },       // Target temperature
        'BB': { number: 50 },       // Target humidity
        'BE': { string: 'auto' }    // Target flow
      });

      const sortedProps = getSortedPrimaryProperties(device);
      const epcs = sortedProps.map(([epc]) => epc);

      // Should follow the order: ESSENTIAL_PROPERTIES first, then DEVICE_PRIMARY_PROPERTIES order
      expect(epcs).toEqual(['80', 'BB', 'BA', 'BE', 'B0', 'B3']);
    });

    it('should handle missing properties gracefully', () => {
      const device = createDevice('0130:1', {
        'B0': { string: 'cool' },
        'B3': { number: 25 },
        'BE': { number: 22 }
      });

      const sortedProps = getSortedPrimaryProperties(device);
      const epcs = sortedProps.map(([epc]) => epc);

      // Should only include properties that exist, in defined order
      expect(epcs).toEqual(['BE', 'B0', 'B3']);
    });

    it('should only return primary properties', () => {
      const device = createDevice('0130:1', {
        '80': { string: 'on' },     // Primary
        '81': { string: 'living' }, // Primary
        'B0': { string: 'cool' },   // Primary
        '9F': { EDT: 'base64' },    // Secondary
        'FF': { string: 'test' }    // Secondary
      });

      const sortedProps = getSortedPrimaryProperties(device);

      // Should only contain primary properties
      const epcs = sortedProps.map(([epc]) => epc);
      expect(epcs).toContain('80');
      expect(epcs).toContain('B0');
      expect(epcs).not.toContain('9F');
      expect(epcs).not.toContain('FF');
      expect(epcs).not.toContain('81'); // 81 is not essential anymore
    });

    it('should return empty array for device with no primary properties', () => {
      const device = createDevice('0130:1', {
        '9F': { EDT: 'base64' },    // Secondary
        'FF': { string: 'test' }    // Secondary
      });

      const sortedProps = getSortedPrimaryProperties(device);
      expect(sortedProps).toEqual([]);
    });

    it('should maintain device-specific property order', () => {
      const device = createDevice('0130:1', {
        'BE': { number: 22 },       // Target flow
        '80': { string: 'on' },     // Operation Status (essential, should be first)
        'B3': { number: 25 },       // Temperature
        'BB': { number: 50 },       // Target humidity
        'B0': { string: 'cool' },   // Operation mode
        'BA': { number: 24 }        // Target temperature
      });

      const sortedProps = getSortedPrimaryProperties(device);
      const epcs = sortedProps.map(([epc]) => epc);

      // Should follow definition order: essential first, then device-specific in order
      expect(epcs).toEqual(['80', 'BB', 'BA', 'BE', 'B0', 'B3']);
    });
  });

  describe('isNodeProfileDevice', () => {
    it('should return true for Node Profile devices (0EF0)', () => {
      const nodeProfileDevice = { eoj: '0EF0:1' };
      expect(isNodeProfileDevice(nodeProfileDevice)).toBe(true);
    });

    it('should return false for non-Node Profile devices', () => {
      const airConditioner = { eoj: '0130:1' };
      const lighting = { eoj: '0291:1' };
      expect(isNodeProfileDevice(airConditioner)).toBe(false);
      expect(isNodeProfileDevice(lighting)).toBe(false);
    });

    it('should handle various EOJ formats', () => {
      const nodeProfileWithInstance = { eoj: '0EF0:2' };
      const nodeProfileSimple = { eoj: '0EF0' };
      expect(isNodeProfileDevice(nodeProfileWithInstance)).toBe(true);
      expect(isNodeProfileDevice(nodeProfileSimple)).toBe(true);
    });
  });

  describe('shouldShowPropertyInCompactMode', () => {
    describe('Home Air Conditioner (0130)', () => {
      it('should hide temperature setting (B3) when operation mode is auto (string alias)', () => {
        const device = createDevice('0130:1', {
          'B0': { string: 'auto' }, // auto mode
          'B3': { number: 25 }      // temperature setting
        }) as unknown as Device;

        expect(shouldShowPropertyInCompactMode('B3', device, '0130')).toBe(false);
      });

      it('should hide temperature setting (B3) when operation mode is fan (string alias)', () => {
        const device = createDevice('0130:1', {
          'B0': { string: 'fan' }, // fan mode
          'B3': { number: 25 }     // temperature setting
        }) as unknown as Device;

        expect(shouldShowPropertyInCompactMode('B3', device, '0130')).toBe(false);
      });

      it('should show temperature setting (B3) when operation mode is cooling (string alias)', () => {
        const device = createDevice('0130:1', {
          'B0': { string: 'cooling' }, // cooling mode
          'B3': { number: 25 }         // temperature setting
        }) as unknown as Device;

        expect(shouldShowPropertyInCompactMode('B3', device, '0130')).toBe(true);
      });

      it('should show temperature setting (B3) when operation mode is heating (string alias)', () => {
        const device = createDevice('0130:1', {
          'B0': { string: 'heating' }, // heating mode
          'B3': { number: 25 }         // temperature setting
        }) as unknown as Device;

        expect(shouldShowPropertyInCompactMode('B3', device, '0130')).toBe(true);
      });

      it('should show relative humidity setting (B4) when operation mode is dry (string alias)', () => {
        const device = createDevice('0130:1', {
          'B0': { string: 'dry' }, // dry mode
          'B4': { number: 60 }     // relative humidity setting
        }) as unknown as Device;

        expect(shouldShowPropertyInCompactMode('B4', device, '0130')).toBe(true);
      });

      it('should hide relative humidity setting (B4) when operation mode is cooling (string alias)', () => {
        const device = createDevice('0130:1', {
          'B0': { string: 'cooling' }, // cooling mode
          'B4': { number: 60 }         // relative humidity setting
        }) as unknown as Device;

        expect(shouldShowPropertyInCompactMode('B4', device, '0130')).toBe(false);
      });

      it('should hide relative humidity setting (B4) when operation mode is heating (string alias)', () => {
        const device = createDevice('0130:1', {
          'B0': { string: 'heating' }, // heating mode
          'B4': { number: 60 }         // relative humidity setting
        }) as unknown as Device;

        expect(shouldShowPropertyInCompactMode('B4', device, '0130')).toBe(false);
      });

      it('should hide relative humidity setting (B4) when operation mode is fan (string alias)', () => {
        const device = createDevice('0130:1', {
          'B0': { string: 'fan' }, // fan mode
          'B4': { number: 60 }     // relative humidity setting
        }) as unknown as Device;

        expect(shouldShowPropertyInCompactMode('B4', device, '0130')).toBe(false);
      });

      it('should show property when condition property does not exist', () => {
        const device = createDevice('0130:1', {
          'B3': { number: 25 }    // temperature setting without operation mode
        }) as unknown as Device;

        expect(shouldShowPropertyInCompactMode('B3', device, '0130')).toBe(true);
      });

      it('should show other properties unconditionally', () => {
        const device = createDevice('0130:1', {
          'B0': { string: 'auto' }, // auto mode
          '80': { number: 0x30 }    // operation status
        }) as unknown as Device;

        expect(shouldShowPropertyInCompactMode('80', device, '0130')).toBe(true);
        expect(shouldShowPropertyInCompactMode('BB', device, '0130')).toBe(true); // room temperature
      });
    });

    describe('other device types', () => {
      it('should show all properties for devices without visibility conditions', () => {
        const device = createDevice('0291:1', { // Single Function Lighting
          'B0': { number: 50 }    // illuminance level
        }) as unknown as Device;

        expect(shouldShowPropertyInCompactMode('B0', device, '0291')).toBe(true);
      });

      it('should show all properties for unknown device types', () => {
        const device = createDevice('9999:1', {
          'B0': { number: 42 }
        }) as unknown as Device;

        expect(shouldShowPropertyInCompactMode('B0', device, '9999')).toBe(true);
      });
    });

    describe('edge cases', () => {
      it('should handle device with no properties', () => {
        const device = createDevice('0130:1', {}) as unknown as Device;

        expect(shouldShowPropertyInCompactMode('B3', device, '0130')).toBe(true);
      });

      it('should prioritize string alias over numeric value', () => {
        const device = createDevice('0130:1', {
          'B0': { string: 'auto', number: 0x42 }, // string says auto, but number would be cooling
          'B3': { number: 25 }
        }) as unknown as Device;

        // Should use string alias (auto) and hide the property
        expect(shouldShowPropertyInCompactMode('B3', device, '0130')).toBe(false);
      });

      it('should fall back to EDT-decoded number when string alias is not available', () => {
        const device = createDevice('0130:1', {
          'B0': { EDT: 'QQ==' }, // Base64 for 0x41 (auto)
          'B3': { number: 25 }
        }) as unknown as Device;

        // EDT decodes to 0x41, but condition uses string 'auto', so won't match
        // Property should be shown since EDT numeric value doesn't match string condition
        expect(shouldShowPropertyInCompactMode('B3', device, '0130')).toBe(true);
      });

      it('should handle property with no comparable value', () => {
        const device = createDevice('0130:1', {
          'B0': {}, // No string, number, or EDT
          'B3': { number: 25 }
        }) as unknown as Device;

        // Should show the property if condition value cannot be determined
        expect(shouldShowPropertyInCompactMode('B3', device, '0130')).toBe(true);
      });
    });
  });

  describe('getDashboardStatusProperties', () => {
    it('should return room temperature and operation mode for Home Air Conditioner', () => {
      expect(getDashboardStatusProperties('0130')).toEqual(['BB', 'B0']);
    });

    it('should return illuminance level for Single Function Lighting', () => {
      expect(getDashboardStatusProperties('0291')).toEqual(['B0']);
    });

    it('should return room temperature for Floor Heating', () => {
      expect(getDashboardStatusProperties('027B')).toEqual(['E2']);
    });

    it('should return scene control for Lighting System', () => {
      expect(getDashboardStatusProperties('02A3')).toEqual(['C0']);
    });

    it('should return hot water temperature for Electric Water Heater', () => {
      expect(getDashboardStatusProperties('026B')).toEqual(['D1']);
    });

    it('should return operation mode for Bath Room Heating', () => {
      expect(getDashboardStatusProperties('0272')).toEqual(['B0']);
    });

    it('should return door open status for Refrigerator', () => {
      expect(getDashboardStatusProperties('03B7')).toEqual(['B0']);
    });

    it('should return undefined for unknown device types', () => {
      expect(getDashboardStatusProperties('9999')).toBeUndefined();
    });

    it('should return undefined for Node Profile devices', () => {
      expect(getDashboardStatusProperties('0EF0')).toBeUndefined();
    });
  });

  describe('DEVICE_DASHBOARD_PROPERTIES', () => {
    it('should have entries for common device types', () => {
      expect(DEVICE_DASHBOARD_PROPERTIES['0130']).toBeDefined(); // Air Conditioner
      expect(DEVICE_DASHBOARD_PROPERTIES['0291']).toBeDefined(); // Lighting
      expect(DEVICE_DASHBOARD_PROPERTIES['027B']).toBeDefined(); // Floor Heating
    });

    it('should not include Node Profile devices', () => {
      expect(DEVICE_DASHBOARD_PROPERTIES['0EF0']).toBeUndefined();
    });

    it('dashboard properties should exist in primary properties', () => {
      // Each dashboard property should be part of the device's primary properties
      for (const [classCode, dashboardEpcs] of Object.entries(DEVICE_DASHBOARD_PROPERTIES)) {
        const primaryProps = DEVICE_PRIMARY_PROPERTIES[classCode];
        if (primaryProps) {
          for (const epc of dashboardEpcs) {
            expect(primaryProps).toContain(epc);
          }
        }
      }
    });

    it('should return arrays for all device types', () => {
      for (const dashboardEpcs of Object.values(DEVICE_DASHBOARD_PROPERTIES)) {
        expect(Array.isArray(dashboardEpcs)).toBe(true);
        expect(dashboardEpcs.length).toBeGreaterThan(0);
      }
    });
  });
});