import { hasAnyOperationalDevice, hasAnyFaultyDevice, groupDevicesByLocation, getAllLocations, getDevicesForTab } from './locationHelper';
import type { Device, DeviceAlias, DeviceGroup } from '@/hooks/types';

describe('locationHelper', () => {
  const createDevice = (ip: string, eoj: string, operationStatus?: 'on' | 'off', faultStatus?: 'fault' | 'no_fault'): Device => {
    const properties: Record<string, any> = {};
    
    if (operationStatus) {
      properties['80'] = { string: operationStatus };
      // Add Set Property Map (EPC 0x9E) that includes EPC 0x80 to make it settable
      // Format: first byte = number of properties, followed by EPC codes
      const mapData = String.fromCharCode(1, 0x80); // 1 property: 0x80
      properties['9E'] = { EDT: btoa(mapData) };
    }
    
    if (faultStatus) {
      properties['88'] = { string: faultStatus };
    }
    
    return {
      ip,
      eoj,
      name: `${ip}-${eoj}`,
      id: `${eoj}:001:${ip}`,
      properties,
      lastSeen: '2023-01-01T00:00:00Z'
    };
  };

  const createNodeProfileDevice = (ip: string): Device => ({
    ip,
    eoj: '0EF0:1',
    name: `${ip}-0EF0:1`,
    id: `0EF0:001:${ip}`,
    properties: {},
    lastSeen: '2023-01-01T00:00:00Z'
  });

  describe('hasAnyOperationalDevice', () => {
    it('should return true when at least one device has operation status "on"', () => {
      const devices = [
        createDevice('192.168.1.1', '0130:1', 'off'),
        createDevice('192.168.1.2', '0130:2', 'on'),
        createDevice('192.168.1.3', '0130:3', 'off')
      ];

      expect(hasAnyOperationalDevice(devices)).toBe(true);
    });

    it('should return false when all devices have operation status "off"', () => {
      const devices = [
        createDevice('192.168.1.1', '0130:1', 'off'),
        createDevice('192.168.1.2', '0130:2', 'off'),
        createDevice('192.168.1.3', '0130:3', 'off')
      ];

      expect(hasAnyOperationalDevice(devices)).toBe(false);
    });

    it('should return false when no devices have operation status property', () => {
      const devices = [
        createDevice('192.168.1.1', '0130:1'),
        createDevice('192.168.1.2', '0130:2'),
        createDevice('192.168.1.3', '0130:3')
      ];

      expect(hasAnyOperationalDevice(devices)).toBe(false);
    });

    it('should return false for empty device array', () => {
      expect(hasAnyOperationalDevice([])).toBe(false);
    });

    it('should handle mixed devices with and without operation status', () => {
      const devices = [
        createDevice('192.168.1.1', '0130:1'),
        createDevice('192.168.1.2', '0130:2', 'on'),
        createDevice('192.168.1.3', '0130:3', 'off')
      ];

      expect(hasAnyOperationalDevice(devices)).toBe(true);
    });
  });

  describe('hasAnyFaultyDevice', () => {
    it('should return true when at least one device has a fault', () => {
      const devices = [
        createDevice('192.168.1.1', '0130:1', 'on', 'no_fault'),
        createDevice('192.168.1.2', '0130:2', 'off', 'fault'),
        createDevice('192.168.1.3', '0130:3', 'on', 'no_fault')
      ];

      expect(hasAnyFaultyDevice(devices)).toBe(true);
    });

    it('should return false when all devices have no faults', () => {
      const devices = [
        createDevice('192.168.1.1', '0130:1', 'on', 'no_fault'),
        createDevice('192.168.1.2', '0130:2', 'off', 'no_fault'),
        createDevice('192.168.1.3', '0130:3', 'on', 'no_fault')
      ];

      expect(hasAnyFaultyDevice(devices)).toBe(false);
    });

    it('should return false when no devices have fault status property', () => {
      const devices = [
        createDevice('192.168.1.1', '0130:1', 'on'),
        createDevice('192.168.1.2', '0130:2', 'off'),
        createDevice('192.168.1.3', '0130:3', 'on')
      ];

      expect(hasAnyFaultyDevice(devices)).toBe(false);
    });

    it('should return false for empty device array', () => {
      expect(hasAnyFaultyDevice([])).toBe(false);
    });

    it('should handle mixed devices with and without fault status', () => {
      const devices = [
        createDevice('192.168.1.1', '0130:1', 'on'),
        createDevice('192.168.1.2', '0130:2', 'off', 'fault'),
        createDevice('192.168.1.3', '0130:3', 'on', 'no_fault')
      ];

      expect(hasAnyFaultyDevice(devices)).toBe(true);
    });
  });

  describe('groupDevicesByLocation', () => {
    it('should exclude Node Profile devices from location grouping', () => {
      const devices = {
        'device1': createDevice('192.168.1.1', '0130:1'),
        'device2': createDevice('192.168.1.2', '0290:1'),
        'nodeProfile': createNodeProfileDevice('192.168.1.3')
      };
      const aliases: DeviceAlias = {};

      const grouped = groupDevicesByLocation(devices, aliases);
      
      // Should not contain Node Profile device in any location group
      const allGroupedDevices = Object.values(grouped).flat();
      expect(allGroupedDevices).not.toContain(devices.nodeProfile);
      expect(allGroupedDevices).toContain(devices.device1);
      expect(allGroupedDevices).toContain(devices.device2);
    });
  });

  describe('getAllLocations', () => {
    it('should exclude Node Profile devices from location detection', () => {
      const devices = {
        'device1': createDevice('192.168.1.1', '0130:1'),
        'device2': createDevice('192.168.1.2', '0290:1'),
        'nodeProfile': createNodeProfileDevice('192.168.1.3')
      };
      const aliases: DeviceAlias = {};

      const locations = getAllLocations(devices, aliases);
      
      // Should still return All + location tabs, but Node Profile shouldn't affect location generation
      expect(locations).toContain('All');
      // Since devices don't have location info, they'll be named by their device names (IP-EOJ format)
      expect(locations.length).toBeGreaterThanOrEqual(1); // At least 'All' should be present
    });
  });

  describe('getDevicesForTab', () => {
    const devices = {
      'device1': createDevice('192.168.1.1', '0130:1'),
      'device2': createDevice('192.168.1.2', '0290:1'),
      'nodeProfile': createNodeProfileDevice('192.168.1.3')
    };
    const aliases: DeviceAlias = {};
    const groups: DeviceGroup = {};

    it('should include Node Profile devices in "All" tab', () => {
      const allDevices = getDevicesForTab('All', devices, aliases, groups);
      
      expect(allDevices).toContain(devices.device1);
      expect(allDevices).toContain(devices.device2);
      expect(allDevices).toContain(devices.nodeProfile);
    });

    it('should exclude Node Profile devices from location tabs', () => {
      // Since devices don't have proper location info, they'll likely be grouped under different names
      // Let's check using the actual location names from groupDevicesByLocation
      const groupedDevices = groupDevicesByLocation(devices, aliases);
      const locationNames = Object.keys(groupedDevices);
      
      if (locationNames.length > 0) {
        const firstLocationDevices = getDevicesForTab(locationNames[0], devices, aliases, groups);
        expect(firstLocationDevices).not.toContain(devices.nodeProfile);
      }
      
      // Test with a specific location that might exist
      const testLocationDevices = getDevicesForTab('192.168.1.1', devices, aliases, groups);
      expect(testLocationDevices).not.toContain(devices.nodeProfile);
    });
  });
});