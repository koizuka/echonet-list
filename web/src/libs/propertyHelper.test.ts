import { describe, it, expect } from 'vitest';
import { isPropertySettable } from './propertyHelper';
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
        lastSeen: Date.now(),
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
        lastSeen: Date.now(),
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
        lastSeen: Date.now(),
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
        lastSeen: Date.now(),
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
        lastSeen: Date.now(),
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
});