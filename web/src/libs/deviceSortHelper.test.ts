import { describe, it, expect } from 'vitest';
import { sortDevicesByEOJAndLocation, sortDevicesWithComparator } from './deviceSortHelper';
import type { Device } from '@/hooks/types';

describe('deviceSortHelper', () => {
  const createDevice = (ip: string, eoj: string, installationLocation?: string): Device => ({
    id: `${ip}_${eoj}`,
    ip,
    eoj,
    name: `Device ${eoj}`,
    lastSeen: new Date().toISOString(),
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

    it('should sort IP addresses numerically not lexicographically', () => {
      const devices = [
        createDevice('192.168.0.90', '0130:1', 'living'),
        createDevice('192.168.0.128', '0130:1', 'living'),
        createDevice('192.168.0.2', '0130:1', 'living'),
        createDevice('192.168.0.10', '0130:1', 'living'),
      ];

      const sorted = sortDevicesByEOJAndLocation(devices);
      const ipOrder = sorted.map(d => d.ip);

      // Should be sorted numerically: 2 < 10 < 90 < 128
      expect(ipOrder).toEqual(['192.168.0.2', '192.168.0.10', '192.168.0.90', '192.168.0.128']);
    });

    it('should handle IP addresses with different octets correctly', () => {
      const devices = [
        createDevice('10.0.0.1', '0130:1', 'living'),
        createDevice('192.168.1.1', '0130:1', 'living'),
        createDevice('172.16.0.1', '0130:1', 'living'),
        createDevice('10.0.0.2', '0130:1', 'living'),
      ];

      const sorted = sortDevicesByEOJAndLocation(devices);
      const ipOrder = sorted.map(d => d.ip);

      expect(ipOrder).toEqual(['10.0.0.1', '10.0.0.2', '172.16.0.1', '192.168.1.1']);
    });

    it('should handle invalid IP addresses gracefully', () => {
      const devices = [
        createDevice('192.168.1.1', '0130:1', 'living'),
        createDevice('invalid.ip', '0130:1', 'living'),
        createDevice('192.168.1.2', '0130:1', 'living'),
        createDevice('256.256.256.256', '0130:1', 'living'), // Invalid octets
      ];

      const sorted = sortDevicesByEOJAndLocation(devices);
      const ipOrder = sorted.map(d => d.ip);

      // Valid IPs should come first, then invalid ones
      expect(ipOrder[0]).toBe('192.168.1.1');
      expect(ipOrder[1]).toBe('192.168.1.2');
      // Invalid IPs should be at the end
      expect(ipOrder.slice(2)).toContain('invalid.ip');
      expect(ipOrder.slice(2)).toContain('256.256.256.256');
    });

    it('should sort IPv6 addresses correctly', () => {
      const devices = [
        createDevice('2001:db8::8', '0130:1', 'living'),
        createDevice('2001:db8::1', '0130:1', 'living'),
        createDevice('::1', '0130:1', 'living'), // loopback
        createDevice('fe80::1', '0130:1', 'living'), // link-local
      ];

      const sorted = sortDevicesByEOJAndLocation(devices);
      const ipOrder = sorted.map(d => d.ip);

      // Should be sorted in normalized order
      expect(ipOrder).toEqual(['::1', '2001:db8::1', '2001:db8::8', 'fe80::1']);
    });

    it('should sort IPv4 addresses before IPv6 addresses', () => {
      const devices = [
        createDevice('::1', '0130:1', 'living'),
        createDevice('192.168.1.1', '0130:1', 'living'),
        createDevice('2001:db8::1', '0130:1', 'living'),
        createDevice('10.0.0.1', '0130:1', 'living'),
      ];

      const sorted = sortDevicesByEOJAndLocation(devices);
      const ipOrder = sorted.map(d => d.ip);

      // IPv4 addresses should come first
      expect(ipOrder[0]).toBe('10.0.0.1');
      expect(ipOrder[1]).toBe('192.168.1.1');
      expect(ipOrder[2]).toBe('::1');
      expect(ipOrder[3]).toBe('2001:db8::1');
    });

    it('should handle IPv6 compressed notation correctly', () => {
      const devices = [
        createDevice('2001:0db8:0000:0000:0000:0000:0000:0001', '0130:1', 'living'), // full form
        createDevice('2001:db8::1', '0130:1', 'living'), // compressed form
        createDevice('2001:0DB8::0001', '0130:1', 'living'), // different case and padding
      ];

      const sorted = sortDevicesByEOJAndLocation(devices);
      
      // All three should be treated as the same IP (stable sort keeps original order)
      expect(sorted.length).toBe(3);
      // They should remain in original order since they're equal after normalization
      expect(sorted[0].ip).toBe('2001:0db8:0000:0000:0000:0000:0000:0001');
      expect(sorted[1].ip).toBe('2001:db8::1');
      expect(sorted[2].ip).toBe('2001:0DB8::0001');
    });

    it('should handle IPv4-mapped IPv6 addresses', () => {
      const devices = [
        createDevice('::ffff:192.168.1.1', '0130:1', 'living'), // IPv4-mapped IPv6
        createDevice('192.168.1.1', '0130:1', 'living'), // Regular IPv4
        createDevice('::ffff:10.0.0.1', '0130:1', 'living'), // Another IPv4-mapped
      ];

      const sorted = sortDevicesByEOJAndLocation(devices);
      const ipOrder = sorted.map(d => d.ip);

      // Regular IPv4 should come first, then IPv4-mapped IPv6
      expect(ipOrder[0]).toBe('192.168.1.1');
      expect(ipOrder[1]).toBe('::ffff:10.0.0.1'); // Lower IP in mapped form
      expect(ipOrder[2]).toBe('::ffff:192.168.1.1');
    });

    it('should handle IPv6 addresses with brackets', () => {
      const devices = [
        createDevice('[2001:db8::1]', '0130:1', 'living'), // With brackets
        createDevice('2001:db8::2', '0130:1', 'living'), // Without brackets
        createDevice('[::1]', '0130:1', 'living'), // Loopback with brackets
      ];

      const sorted = sortDevicesByEOJAndLocation(devices);
      const ipOrder = sorted.map(d => d.ip);

      // Should normalize and sort correctly despite brackets
      expect(ipOrder).toEqual(['[::1]', '[2001:db8::1]', '2001:db8::2']);
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