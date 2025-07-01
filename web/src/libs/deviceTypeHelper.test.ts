import { describe, it, expect } from 'vitest';
import {
  getDevicePrimaryProperties,
  isPropertyPrimary,
  getDeviceSecondaryProperties,
  getSortedPrimaryProperties,
  isNodeProfileDevice,
  ESSENTIAL_PROPERTIES,
  DEVICE_PRIMARY_PROPERTIES
} from './deviceTypeHelper';

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
      const device = {
        eoj: '0130:1', // Home Air Conditioner
        properties: {
          '80': { string: 'on' },     // Primary (essential)
          '81': { string: 'living' }, // Secondary (not essential anymore)
          'B0': { string: 'cool' },   // Primary (device-specific)
          '9F': { EDT: 'base64' },    // Secondary
          'FF': { string: 'test' }    // Secondary
        }
      };

      const secondaryProps = getDeviceSecondaryProperties(device);
      expect(secondaryProps).toContain('9F');
      expect(secondaryProps).toContain('FF');
      expect(secondaryProps).not.toContain('80');
      expect(secondaryProps).not.toContain('B0');
    });

    it('should return empty array when all properties are primary', () => {
      const device = {
        eoj: '0291:1', // Single Function Lighting
        properties: {
          '80': { string: 'on' },     // Primary (essential)
          '81': { string: 'bedroom' }, // Secondary (not essential anymore)
          'B0': { number: 50 }        // Primary (device-specific)
        }
      };

      const secondaryProps = getDeviceSecondaryProperties(device);
      expect(secondaryProps).toEqual(['81']); // '81' is now secondary (not essential)
    });

    it('should handle devices with no properties', () => {
      const device = {
        eoj: '0130:1',
        properties: {}
      };

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
      const device = {
        eoj: '0130:1', // Home Air Conditioner
        properties: {
          'B0': { string: 'cool' },   // Operation mode
          '81': { string: 'living' }, // Installation Location
          '80': { string: 'on' },     // Operation Status
          'B3': { number: 25 },       // Temperature
          'A0': { string: 'auto' },   // Air flow rate
          'A3': { string: 'swing' },  // Air flow direction
          'BA': { number: 24 },       // Target temperature
          'BB': { number: 50 },       // Target humidity
          'BE': { string: 'auto' }    // Target flow
        }
      };

      const sortedProps = getSortedPrimaryProperties(device);
      const epcs = sortedProps.map(([epc]) => epc);

      // Should follow the order: ESSENTIAL_PROPERTIES first, then DEVICE_PRIMARY_PROPERTIES order
      expect(epcs).toEqual(['80', 'BB', 'BA', 'BE', 'B0', 'B3']);
    });

    it('should handle missing properties gracefully', () => {
      const device = {
        eoj: '0130:1',
        properties: {
          'B0': { string: 'cool' },
          'B3': { number: 25 },
          'BE': { number: 22 }
        }
      };

      const sortedProps = getSortedPrimaryProperties(device);
      const epcs = sortedProps.map(([epc]) => epc);

      // Should only include properties that exist, in defined order
      expect(epcs).toEqual(['BE', 'B0', 'B3']);
    });

    it('should only return primary properties', () => {
      const device = {
        eoj: '0130:1',
        properties: {
          '80': { string: 'on' },     // Primary
          '81': { string: 'living' }, // Primary
          'B0': { string: 'cool' },   // Primary
          '9F': { EDT: 'base64' },    // Secondary
          'FF': { string: 'test' }    // Secondary
        }
      };

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
      const device = {
        eoj: '0130:1',
        properties: {
          '9F': { EDT: 'base64' },    // Secondary
          'FF': { string: 'test' }    // Secondary
        }
      };

      const sortedProps = getSortedPrimaryProperties(device);
      expect(sortedProps).toEqual([]);
    });

    it('should maintain device-specific property order', () => {
      const device = {
        eoj: '0130:1',
        properties: {
          'BE': { number: 22 },       // Target flow
          '80': { string: 'on' },     // Operation Status (essential, should be first)
          'B3': { number: 25 },       // Temperature
          'BB': { number: 50 },       // Target humidity
          'B0': { string: 'cool' },   // Operation mode
          'BA': { number: 24 }        // Target temperature
        }
      };

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
});