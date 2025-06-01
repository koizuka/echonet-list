import { describe, it, expect } from 'vitest';
import { 
  getDevicePrimaryProperties, 
  isPropertyPrimary, 
  getDeviceSecondaryProperties,
  getSortedPrimaryProperties,
  ESSENTIAL_PROPERTIES,
  DEVICE_PRIMARY_PROPERTIES
} from './deviceTypeHelper';

describe('deviceTypeHelper', () => {
  describe('getDevicePrimaryProperties', () => {
    it('should return essential properties for unknown device types', () => {
      const properties = getDevicePrimaryProperties('9999'); // Unknown class code
      expect(properties).toEqual(['80', '81']); // Essential properties only
    });

    it('should return essential + device-specific properties for air conditioner', () => {
      const properties = getDevicePrimaryProperties('0130'); // Home Air Conditioner
      expect(properties).toContain('80'); // Operation Status (essential)
      expect(properties).toContain('81'); // Installation Location (essential)
      expect(properties).toContain('B0'); // Operation mode (device-specific)
      expect(properties).toContain('B3'); // Temperature (device-specific)
    });

    it('should return essential + device-specific properties for lighting', () => {
      const properties = getDevicePrimaryProperties('0290'); // Single Function Lighting
      expect(properties).toContain('80'); // Operation Status (essential)
      expect(properties).toContain('81'); // Installation Location (essential)
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
      expect(isPropertyPrimary('81', '9999')).toBe(true); // Installation Location
    });

    it('should return true for device-specific primary properties', () => {
      expect(isPropertyPrimary('B0', '0130')).toBe(true); // Air conditioner operation mode
      expect(isPropertyPrimary('B0', '0290')).toBe(true); // Lighting illuminance
    });

    it('should return false for non-primary properties', () => {
      expect(isPropertyPrimary('9F', '0130')).toBe(false); // Get Property Map
      expect(isPropertyPrimary('FF', '0290')).toBe(false); // Unknown property
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
          '81': { string: 'living' }, // Primary (essential)
          'B0': { string: 'cool' },   // Primary (device-specific)
          '9F': { EDT: 'base64' },    // Secondary
          'FF': { string: 'test' }    // Secondary
        }
      };

      const secondaryProps = getDeviceSecondaryProperties(device);
      expect(secondaryProps).toContain('9F');
      expect(secondaryProps).toContain('FF');
      expect(secondaryProps).not.toContain('80');
      expect(secondaryProps).not.toContain('81');
      expect(secondaryProps).not.toContain('B0');
    });

    it('should return empty array when all properties are primary', () => {
      const device = {
        eoj: '0290:1', // Single Function Lighting
        properties: {
          '80': { string: 'on' },     // Primary (essential)
          '81': { string: 'bedroom' }, // Primary (essential)
          'B0': { number: 50 }        // Primary (device-specific)
        }
      };

      const secondaryProps = getDeviceSecondaryProperties(device);
      expect(secondaryProps).toEqual([]);
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
      expect(ESSENTIAL_PROPERTIES).toEqual(['80', '81']);
    });

    it('should have device-specific properties defined', () => {
      expect(DEVICE_PRIMARY_PROPERTIES['0130']).toContain('B0'); // Air conditioner
      expect(DEVICE_PRIMARY_PROPERTIES['0290']).toContain('B0'); // Lighting
      expect(DEVICE_PRIMARY_PROPERTIES['027B']).toContain('E1'); // Floor heating
    });
  });

  describe('getSortedPrimaryProperties', () => {
    it('should prioritize Operation Status (0x80) first', () => {
      const device = {
        eoj: '0130:1', // Home Air Conditioner
        properties: {
          'B0': { string: 'cool' },   // Operation mode
          '81': { string: 'living' }, // Installation Location
          '80': { string: 'on' },     // Operation Status
          'B3': { number: 25 }        // Temperature
        }
      };

      const sortedProps = getSortedPrimaryProperties(device);
      
      // Operation Status should be first
      expect(sortedProps[0][0]).toBe('80');
      expect(sortedProps[1][0]).toBe('81'); // Installation Location should be second
    });

    it('should handle missing Operation Status gracefully', () => {
      const device = {
        eoj: '0130:1',
        properties: {
          'B0': { string: 'cool' },
          '81': { string: 'living' },
          'B3': { number: 25 }
        }
      };

      const sortedProps = getSortedPrimaryProperties(device);
      
      // Installation Location should be first when Operation Status is missing
      expect(sortedProps[0][0]).toBe('81');
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
      expect(epcs).toContain('81');
      expect(epcs).toContain('B0');
      expect(epcs).not.toContain('9F');
      expect(epcs).not.toContain('FF');
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

    it('should sort other properties alphabetically after essential ones', () => {
      const device = {
        eoj: '0130:1',
        properties: {
          'BE': { number: 22 },       // Target temperature (alphabetically later)
          '80': { string: 'on' },     // Operation Status (should be first)
          'B3': { number: 25 },       // Temperature (alphabetically earlier)
          '81': { string: 'living' }  // Installation Location (should be second)
        }
      };

      const sortedProps = getSortedPrimaryProperties(device);
      const epcs = sortedProps.map(([epc]) => epc);
      
      expect(epcs[0]).toBe('80'); // Operation Status first
      expect(epcs[1]).toBe('81'); // Installation Location second
      expect(epcs[2]).toBe('B3'); // B3 comes before BE alphabetically
      expect(epcs[3]).toBe('BE');
    });
  });
});