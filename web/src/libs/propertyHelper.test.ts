import { describe, it, expect } from 'vitest';
import { isPropertySettable, isDeviceOperational, isDeviceFaulty } from './propertyHelper';
import type { Device } from '@/hooks/types';

describe('propertyHelper', () => {
  describe('isPropertySettable', () => {
    it('should return true for properties listed in Set Property Map', () => {
      // Mock device with Set Property Map (EPC 0x9E)
      // Property map: 2 properties (0x80 and 0xB3)
      // Encoded as: count(2) + 0x80 + 0xB3
      const setPropertyMapEDT = btoa('\x02\x80\xB3');
      
      const device: Device = {
        id: 'test',
        ip: '192.168.1.100',
        eoj: '0130:1',
        name: 'Test Device',
        lastSeen: new Date().toISOString(),
        properties: {
          '9E': { EDT: setPropertyMapEDT }
        }
      };

      // Should return true for properties in the map
      expect(isPropertySettable('80', device)).toBe(true);
      expect(isPropertySettable('B3', device)).toBe(true);
      
      // Should return false for properties not in the map
      expect(isPropertySettable('81', device)).toBe(false);
      expect(isPropertySettable('9D', device)).toBe(false);
    });

    it('should return false when Set Property Map is not available', () => {
      const device: Device = {
        id: 'test',
        ip: '192.168.1.100',
        eoj: '0130:1',
        name: 'Test Device',
    lastSeen: new Date().toISOString(),
        properties: {}
      };

      expect(isPropertySettable('80', device)).toBe(false);
    });

    it('should handle invalid Set Property Map gracefully', () => {
      const device: Device = {
        id: 'test',
        ip: '192.168.1.100',
        eoj: '0130:1',
        name: 'Test Device',
    lastSeen: new Date().toISOString(),
        properties: {
          '9E': { EDT: 'invalid-base64' }
        }
      };

      expect(isPropertySettable('80', device)).toBe(false);
    });

    it('should handle empty Set Property Map', () => {
      // Empty property map: count(0)
      const setPropertyMapEDT = btoa('\x00');
      
      const device: Device = {
        id: 'test',
        ip: '192.168.1.100',
        eoj: '0130:1',
        name: 'Test Device',
        lastSeen: new Date().toISOString(),
        properties: {
          '9E': { EDT: setPropertyMapEDT }
        }
      };

      expect(isPropertySettable('80', device)).toBe(false);
    });

    it('should handle hex EPC codes correctly', () => {
      // Property map with hex codes: 0xB0, 0xB3, 0xFF
      const setPropertyMapEDT = btoa('\x03\xB0\xB3\xFF');
      
      const device: Device = {
        id: 'test',
        ip: '192.168.1.100',
        eoj: '0130:1',
        name: 'Test Device',
        lastSeen: new Date().toISOString(),
        properties: {
          '9E': { EDT: setPropertyMapEDT }
        }
      };

      // Test with both uppercase and lowercase hex
      expect(isPropertySettable('B0', device)).toBe(true);
      expect(isPropertySettable('b0', device)).toBe(true);
      expect(isPropertySettable('B3', device)).toBe(true);
      expect(isPropertySettable('FF', device)).toBe(true);
      expect(isPropertySettable('80', device)).toBe(false);
    });
  });

  describe('isDeviceOperational', () => {
    it('should return true when operation status is on (string)', () => {
      const device: Device = {
        id: 'test',
        ip: '192.168.1.100',
        eoj: '0130:1',
        name: 'Test Device',
        lastSeen: new Date().toISOString(),
        properties: {
          '80': { string: 'on' }
        }
      };

      expect(isDeviceOperational(device)).toBe(true);
    });

    it('should return false when operation status is off (string)', () => {
      const device: Device = {
        id: 'test',
        ip: '192.168.1.100',
        eoj: '0130:1',
        name: 'Test Device',
        lastSeen: new Date().toISOString(),
        properties: {
          '80': { string: 'off' }
        }
      };

      expect(isDeviceOperational(device)).toBe(false);
    });

    it('should return true when operation status is on (EDT 0x30)', () => {
      const operationStatusEDT = btoa('\x30'); // 0x30 = on
      
      const device: Device = {
        id: 'test',
        ip: '192.168.1.100',
        eoj: '0130:1',
        name: 'Test Device',
        lastSeen: new Date().toISOString(),
        properties: {
          '80': { EDT: operationStatusEDT }
        }
      };

      expect(isDeviceOperational(device)).toBe(true);
    });

    it('should return false when operation status is off (EDT 0x31)', () => {
      const operationStatusEDT = btoa('\x31'); // 0x31 = off
      
      const device: Device = {
        id: 'test',
        ip: '192.168.1.100',
        eoj: '0130:1',
        name: 'Test Device',
        lastSeen: new Date().toISOString(),
        properties: {
          '80': { EDT: operationStatusEDT }
        }
      };

      expect(isDeviceOperational(device)).toBe(false);
    });

    it('should return false when no operation status property', () => {
      const device: Device = {
        id: 'test',
        ip: '192.168.1.100',
        eoj: '0130:1',
        name: 'Test Device',
        lastSeen: new Date().toISOString(),
        properties: {}
      };

      expect(isDeviceOperational(device)).toBe(false);
    });
  });

  describe('isDeviceFaulty', () => {
    it('should return false when fault status is no_fault (string)', () => {
      const device: Device = {
        id: 'test',
        ip: '192.168.1.100',
        eoj: '0130:1',
        name: 'Test Device',
        lastSeen: new Date().toISOString(),
        properties: {
          '88': { string: 'no_fault' }
        }
      };

      expect(isDeviceFaulty(device)).toBe(false);
    });

    it('should return true when fault status indicates fault (string)', () => {
      const device: Device = {
        id: 'test',
        ip: '192.168.1.100',
        eoj: '0130:1',
        name: 'Test Device',
        lastSeen: new Date().toISOString(),
        properties: {
          '88': { string: 'fault_detected' }
        }
      };

      expect(isDeviceFaulty(device)).toBe(true);
    });

    it('should return false when fault status is no_fault (EDT 0x42)', () => {
      const faultStatusEDT = btoa('\x42'); // 0x42 = no_fault
      
      const device: Device = {
        id: 'test',
        ip: '192.168.1.100',
        eoj: '0130:1',
        name: 'Test Device',
        lastSeen: new Date().toISOString(),
        properties: {
          '88': { EDT: faultStatusEDT }
        }
      };

      expect(isDeviceFaulty(device)).toBe(false);
    });

    it('should return true when fault status indicates fault (EDT not 0x42)', () => {
      const faultStatusEDT = btoa('\x41'); // 0x41 = fault
      
      const device: Device = {
        id: 'test',
        ip: '192.168.1.100',
        eoj: '0130:1',
        name: 'Test Device',
        lastSeen: new Date().toISOString(),
        properties: {
          '88': { EDT: faultStatusEDT }
        }
      };

      expect(isDeviceFaulty(device)).toBe(true);
    });

    it('should return false when no fault status property', () => {
      const device: Device = {
        id: 'test',
        ip: '192.168.1.100',
        eoj: '0130:1',
        name: 'Test Device',
        lastSeen: new Date().toISOString(),
        properties: {}
      };

      expect(isDeviceFaulty(device)).toBe(false);
    });
  });
});