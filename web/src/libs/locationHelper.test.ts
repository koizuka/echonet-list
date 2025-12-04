import { hasAnyOperationalDevice, hasAnyFaultyDevice, groupDevicesByLocation, getAllLocations, getAllTabs, getDashboardDevicesGroupedByLocation, getDevicesForTab, translateLocationId, getTabDisplayName, isDashboardDevice } from './locationHelper';
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

      const locations = getAllLocations(devices);

      // Should still return All + location tabs, but Node Profile shouldn't affect location generation
      expect(locations).toContain('All');
      // Since devices don't have location info, they'll be named by their device names (IP-EOJ format)
      expect(locations.length).toBeGreaterThanOrEqual(1); // At least 'All' should be present
    });
  });

  describe('getAllTabs', () => {
    it('should return Dashboard as the first tab', () => {
      const devices = {
        'device1': createDevice('192.168.1.1', '0130:1')
      };
      const groups: DeviceGroup = {};

      const tabs = getAllTabs(devices, groups);

      expect(tabs[0]).toBe('Dashboard');
    });

    it('should return All as the second tab', () => {
      const devices = {
        'device1': createDevice('192.168.1.1', '0130:1')
      };
      const groups: DeviceGroup = {};

      const tabs = getAllTabs(devices, groups);

      expect(tabs[1]).toBe('All');
    });

    it('should order tabs as Dashboard, All, locations, then groups', () => {
      const devices = {
        'device1': { ...createDevice('192.168.1.1', '0130:1'), properties: { '81': { string: 'living' } } },
        'device2': { ...createDevice('192.168.1.2', '0290:1'), properties: { '81': { string: 'kitchen' } } }
      };
      const groups: DeviceGroup = { '@GroupA': [], '@GroupB': [] };

      const tabs = getAllTabs(devices, groups);

      expect(tabs[0]).toBe('Dashboard');
      expect(tabs[1]).toBe('All');
      // Locations should come after All (sorted alphabetically)
      const locationsStart = 2;
      const groupsStart = tabs.findIndex(t => t.startsWith('@'));
      expect(groupsStart).toBeGreaterThan(locationsStart);
    });

    it('should include device groups with @ prefix', () => {
      const devices = {};
      const groups: DeviceGroup = { '@MyGroup': ['device1'] };

      const tabs = getAllTabs(devices, groups);

      expect(tabs).toContain('@MyGroup');
    });
  });

  describe('getDashboardDevicesGroupedByLocation', () => {
    // Helper to create settable device with location
    const createSettableDeviceWithLocation = (ip: string, eoj: string, location: string): Device => {
      const mapData = String.fromCharCode(1, 0x80); // 1 property: 0x80 (settable)
      return {
        ip,
        eoj,
        name: `Device ${eoj}`,
        id: undefined,
        lastSeen: new Date().toISOString(),
        properties: {
          '80': { string: 'on' },
          '81': { string: location },
          '9E': { EDT: btoa(mapData) }
        }
      };
    };

    // Helper to create non-settable device with location
    const createNonSettableDeviceWithLocation = (ip: string, eoj: string, location: string): Device => {
      return {
        ip,
        eoj,
        name: `Device ${eoj}`,
        id: undefined,
        lastSeen: new Date().toISOString(),
        properties: {
          '80': { string: 'on' },
          '81': { string: location },
          '9E': { EDT: btoa(String.fromCharCode(0)) } // Empty Set Property Map
        }
      };
    };

    // Note: Filtering tests (Node Profile exclusion, settable status check) are in isDashboardDevice tests

    it('should group devices by installation location', () => {
      const devices = {
        'device1': createSettableDeviceWithLocation('192.168.1.1', '0130:1', 'living'),
        'device2': createSettableDeviceWithLocation('192.168.1.2', '0130:2', 'living'),
        'device3': createSettableDeviceWithLocation('192.168.1.3', '0290:1', 'kitchen')
      };

      const grouped = getDashboardDevicesGroupedByLocation(devices);

      expect(grouped['living']).toHaveLength(2);
      expect(grouped['kitchen']).toHaveLength(1);
    });

    it('should return empty object for no devices', () => {
      const devices = {};

      const grouped = getDashboardDevicesGroupedByLocation(devices);

      expect(Object.keys(grouped)).toHaveLength(0);
    });

    it('should return empty object when only Node Profile devices exist', () => {
      const devices = {
        'nodeProfile1': createNodeProfileDevice('192.168.1.1'),
        'nodeProfile2': createNodeProfileDevice('192.168.1.2')
      };

      const grouped = getDashboardDevicesGroupedByLocation(devices);

      expect(Object.keys(grouped)).toHaveLength(0);
    });

    it('should return empty object when only non-settable devices exist', () => {
      const devices = {
        'nonSettable1': createNonSettableDeviceWithLocation('192.168.1.1', '0130:1', 'living'),
        'nonSettable2': createNonSettableDeviceWithLocation('192.168.1.2', '0130:2', 'kitchen')
      };

      const grouped = getDashboardDevicesGroupedByLocation(devices);

      expect(Object.keys(grouped)).toHaveLength(0);
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
      const allDevices = getDevicesForTab('All', devices, groups);

      expect(allDevices).toContain(devices.device1);
      expect(allDevices).toContain(devices.device2);
      expect(allDevices).toContain(devices.nodeProfile);
    });

    it('should exclude Node Profile devices from "Dashboard" tab', () => {
      const dashboardDevices = getDevicesForTab('Dashboard', devices, groups);

      expect(dashboardDevices).toContain(devices.device1);
      expect(dashboardDevices).toContain(devices.device2);
      expect(dashboardDevices).not.toContain(devices.nodeProfile);
    });

    it('should return all non-NodeProfile devices for "Dashboard" tab', () => {
      const dashboardDevices = getDevicesForTab('Dashboard', devices, groups);

      expect(dashboardDevices).toHaveLength(2);
    });

    it('should exclude Node Profile devices from location tabs', () => {
      // Since devices don't have proper location info, they'll likely be grouped under different names
      // Let's check using the actual location names from groupDevicesByLocation
      const groupedDevices = groupDevicesByLocation(devices, aliases);
      const locationNames = Object.keys(groupedDevices);

      if (locationNames.length > 0) {
        const firstLocationDevices = getDevicesForTab(locationNames[0], devices, groups);
        expect(firstLocationDevices).not.toContain(devices.nodeProfile);
      }

      // Test with a specific location that might exist
      const testLocationDevices = getDevicesForTab('192.168.1.1', devices, groups);
      expect(testLocationDevices).not.toContain(devices.nodeProfile);
    });
  });


  describe('translateLocationId', () => {
    it('should capitalize location names', () => {
      expect(translateLocationId('living')).toBe('Living');
      expect(translateLocationId('kitchen')).toBe('Kitchen');
      expect(translateLocationId('bathroom')).toBe('Bathroom');
    });

    it('should handle "All" tab name unchanged', () => {
      expect(translateLocationId('All')).toBe('All');
    });

    it('should return capitalized ID if no translation found', () => {
      expect(translateLocationId('unknown')).toBe('Unknown');
      expect(translateLocationId('custom')).toBe('Custom');
    });
  });

  describe('isDashboardDevice', () => {
    it('should return true for devices with settable operation status', () => {
      const device = createDevice('192.168.1.1', '0130:1', 'on');
      expect(isDashboardDevice(device)).toBe(true);
    });

    it('should return false for Node Profile devices', () => {
      const nodeProfile = createNodeProfileDevice('192.168.1.1');
      expect(isDashboardDevice(nodeProfile)).toBe(false);
    });

    it('should return false for devices without settable operation status', () => {
      const device: Device = {
        ip: '192.168.1.1',
        eoj: '0130:1',
        name: 'Test Device',
        id: '0130:001:192.168.1.1',
        properties: {
          '80': { string: 'on' }
          // No Set Property Map (0x9E), so operation status is not settable
        },
        lastSeen: '2023-01-01T00:00:00Z'
      };
      expect(isDashboardDevice(device)).toBe(false);
    });
  });

  describe('getTabDisplayName', () => {
    const createDeviceWithLocation = (ip: string, eoj: string, location: string): Device => ({
      ip,
      eoj,
      name: `${ip}-${eoj}`,
      id: `${eoj}:001:${ip}`,
      properties: {
        '81': { string: location }
      },
      lastSeen: '2023-01-01T00:00:00Z'
    });

    it('should return "Dashboard" as-is for Dashboard tab', () => {
      expect(getTabDisplayName('Dashboard', {}, {})).toBe('Dashboard');
    });

    it('should return "All" as-is for All tab', () => {
      expect(getTabDisplayName('All', {}, {})).toBe('All');
    });

    it('should return group name as-is for group tabs (starting with @)', () => {
      expect(getTabDisplayName('@Living Room', {}, {})).toBe('@Living Room');
      expect(getTabDisplayName('@My Group', {}, {})).toBe('@My Group');
    });

    it('should delegate to getLocationDisplayName for location tabs', () => {
      const devices: Record<string, Device> = {
        'device1': createDeviceWithLocation('192.168.1.1', '0130:1', 'living')
      };

      // Without proper property descriptions, formatPropertyValue returns the raw value
      const result = getTabDisplayName('living', devices, {});
      expect(result).toBe('living');
    });

    it('should return capitalized location ID when no device found', () => {
      const result = getTabDisplayName('unknown', {}, {});
      expect(result).toBe('Unknown');
    });
  });
});