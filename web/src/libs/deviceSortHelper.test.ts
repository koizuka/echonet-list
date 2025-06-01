import { describe, it, expect } from 'vitest';
import { sortDevicesByEOJAndLocation, sortDevicesWithComparator } from './deviceSortHelper';
import type { Device } from '@/hooks/types';

describe('deviceSortHelper', () => {
  const createDevice = (ip: string, eoj: string, installationLocation?: string): Device => ({
    id: `${ip}_${eoj}`,
    ip,
    eoj,
    name: `Device ${eoj}`,
    lastSeen: Date.now(),
    properties: installationLocation ? {
      '81': { string: installationLocation }
    } : {}
  });

  describe('sortDevicesByEOJAndLocation', () => {
    it('should sort devices by EOJ (classCode:instance) as primary key', () => {
      const devices = [
        createDevice('192.168.1.103', '0290:1'), // Single Function Lighting
        createDevice('192.168.1.101', '0130:1'), // Home Air Conditioner
        createDevice('192.168.1.102', '0130:2'), // Home Air Conditioner
        createDevice('192.168.1.104', '027B:1'), // Floor Heating
      ];

      const sorted = sortDevicesByEOJAndLocation(devices);
      const eojOrder = sorted.map(d => d.eoj);

      expect(eojOrder).toEqual(['0130:1', '0130:2', '027B:1', '0290:1']);
    });

    it('should sort by installation location as secondary key when EOJ is same', () => {
      const devices = [
        createDevice('192.168.1.103', '0130:1', 'living'),    // Living Room
        createDevice('192.168.1.101', '0130:1', 'kitchen'),   // Kitchen
        createDevice('192.168.1.102', '0130:1', 'bedroom'),   // Bedroom
      ];

      const sorted = sortDevicesByEOJAndLocation(devices);
      const locationOrder = sorted.map(d => d.properties['81']?.string);

      expect(locationOrder).toEqual(['bedroom', 'kitchen', 'living']);
    });

    it('should sort by IP address as tertiary key when EOJ and location are same', () => {
      const devices = [
        createDevice('192.168.1.103', '0130:1', 'living'),
        createDevice('192.168.1.101', '0130:1', 'living'),
        createDevice('192.168.1.102', '0130:1', 'living'),
      ];

      const sorted = sortDevicesByEOJAndLocation(devices);
      const ipOrder = sorted.map(d => d.ip);

      expect(ipOrder).toEqual(['192.168.1.101', '192.168.1.102', '192.168.1.103']);
    });

    it('should handle devices without installation location', () => {
      const devices = [
        createDevice('192.168.1.102', '0130:1', 'living'),
        createDevice('192.168.1.101', '0130:1'), // No installation location
        createDevice('192.168.1.103', '0130:1', 'kitchen'),
      ];

      const sorted = sortDevicesByEOJAndLocation(devices);
      
      // Device without location should come first (empty string sorts first)
      expect(sorted[0].ip).toBe('192.168.1.101');
      expect(sorted[1].properties['81']?.string).toBe('kitchen');
      expect(sorted[2].properties['81']?.string).toBe('living');
    });

    it('should handle mixed EOJ and location scenarios', () => {
      const devices = [
        createDevice('192.168.1.105', '0290:1', 'living'),    // Lighting - Living
        createDevice('192.168.1.102', '0130:1', 'kitchen'),   // AC - Kitchen
        createDevice('192.168.1.104', '0130:2', 'bedroom'),   // AC - Bedroom
        createDevice('192.168.1.103', '0130:1', 'living'),    // AC - Living
        createDevice('192.168.1.101', '027B:1', 'bathroom'),  // Floor Heating - Bathroom
      ];

      const sorted = sortDevicesByEOJAndLocation(devices);
      const expectedOrder = [
        '0130:1', // AC 1 - Kitchen (alphabetically first)
        '0130:1', // AC 1 - Living
        '0130:2', // AC 2 - Bedroom
        '027B:1', // Floor Heating - Bathroom
        '0290:1', // Lighting - Living
      ];

      const actualOrder = sorted.map(d => d.eoj);
      expect(actualOrder).toEqual(expectedOrder);

      // Verify location order for same EOJ
      const ac1Devices = sorted.filter(d => d.eoj === '0130:1');
      expect(ac1Devices[0].properties['81']?.string).toBe('kitchen');
      expect(ac1Devices[1].properties['81']?.string).toBe('living');
    });

    it('should not modify the original array', () => {
      const devices = [
        createDevice('192.168.1.102', '0290:1'),
        createDevice('192.168.1.101', '0130:1'),
      ];
      const originalOrder = devices.map(d => d.eoj);

      sortDevicesByEOJAndLocation(devices);

      // Original array should remain unchanged
      expect(devices.map(d => d.eoj)).toEqual(originalOrder);
    });
  });

  describe('sortDevicesWithComparator', () => {
    it('should sort devices using custom comparator', () => {
      const devices = [
        createDevice('192.168.1.103', '0130:3'),
        createDevice('192.168.1.101', '0130:1'),
        createDevice('192.168.1.102', '0130:2'),
      ];

      // Sort by IP address
      const sortedByIp = sortDevicesWithComparator(devices, (a, b) => a.ip.localeCompare(b.ip));
      const ipOrder = sortedByIp.map(d => d.ip);

      expect(ipOrder).toEqual(['192.168.1.101', '192.168.1.102', '192.168.1.103']);
    });

    it('should not modify the original array', () => {
      const devices = [
        createDevice('192.168.1.102', '0130:2'),
        createDevice('192.168.1.101', '0130:1'),
      ];
      const originalOrder = devices.map(d => d.ip);

      sortDevicesWithComparator(devices, (a, b) => a.ip.localeCompare(b.ip));

      // Original array should remain unchanged
      expect(devices.map(d => d.ip)).toEqual(originalOrder);
    });
  });
});