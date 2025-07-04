import { describe, it, expect, vi } from 'vitest';
import { 
  isPropertySettable, 
  isDeviceOperational, 
  isDeviceFaulty, 
  decodePropertyMap,
  getPropertyName,
  getPropertyDescriptor,
  formatPropertyValue 
} from './propertyHelper';
import type { Device, PropertyDescriptionData, PropertyDescriptor } from '@/hooks/types';

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
      // Suppress console warnings for this test
      const originalWarn = console.warn;
      console.warn = vi.fn();
      
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
      
      // Verify console.warn was called with the error
      expect(console.warn).toHaveBeenCalledWith(
        'Failed to parse Set Property Map for device 192.168.1.100 0130:1:',
        expect.any(Error)
      );
      
      // Restore console.warn
      console.warn = originalWarn;
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

describe('decodePropertyMap', () => {
  it('should decode direct list format (< 16 properties)', () => {
    // Create a property map with 3 properties: 80, 81, B0
    const epcs = [0x80, 0x81, 0xB0];
    const data = [epcs.length, ...epcs];
    const edt = btoa(String.fromCharCode(...data));
    
    const result = decodePropertyMap(edt);
    
    expect(result).toEqual(['80', '81', 'B0']);
  });

  it('should decode bitmap format (>= 16 properties)', () => {
    // Create a bitmap with specific properties
    const propertyEpcs = [
      0x80, // Operation status
      0x81, // Installation location
      0x88, // Fault occurrence status
      0x9E, // Set property map
      0xA0, // Test property in range A0-AF
      0xB0, // Test property in range B0-BF
    ];

    const bitmapData = new Array(17).fill(0);
    bitmapData[0] = 16; // Property count >= 16 triggers bitmap format

    // Set bits according to Go formula: EPC = i + (j << 4) + 0x80
    propertyEpcs.forEach(epc => {
      const offset = epc - 0x80;
      const i = offset & 0x0F; // byte index (0-15)
      const j = (offset & 0xF0) >> 4; // bit index (0-7)
      if (i < 16 && j < 8) {
        bitmapData[i + 1] |= (1 << j);
      }
    });

    const edt = btoa(String.fromCharCode(...bitmapData));
    const result = decodePropertyMap(edt);
    
    expect(result).toEqual(['80', '81', '88', '9E', 'A0', 'B0']);
  });

  it('should sort EPCs in ascending order', () => {
    // Create property map with unsorted EPCs
    const epcs = [0xB0, 0x80, 0x9F, 0x81]; // Unsorted
    const data = [epcs.length, ...epcs];
    const edt = btoa(String.fromCharCode(...data));
    
    const result = decodePropertyMap(edt);
    
    expect(result).toEqual(['80', '81', '9F', 'B0']); // Sorted
  });

  it('should handle empty property map', () => {
    const data = [0]; // Count = 0
    const edt = btoa(String.fromCharCode(...data));
    
    const result = decodePropertyMap(edt);
    
    expect(result).toEqual([]);
  });

  it('should handle invalid Base64 input', () => {
    // Mock console.warn to suppress expected error output
    const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
    
    const result = decodePropertyMap('invalid-base64!');
    
    expect(result).toBeNull();
    
    consoleSpy.mockRestore();
  });

  it('should handle empty EDT', () => {
    const result = decodePropertyMap('');
    
    expect(result).toBeNull();
  });

  it('should handle insufficient bitmap data', () => {
    // Property count >= 16 but insufficient bitmap data
    const data = [16, 0x80]; // Count = 16 but only 2 bytes total
    const edt = btoa(String.fromCharCode(...data));
    
    const result = decodePropertyMap(edt);
    
    expect(result).toBeNull();
  });

  it('should handle realistic air conditioner property map', () => {
    // Test with realistic air conditioner properties (22 properties)
    const propertyEpcs = [
      0x80, 0x81, 0x88, 0x8A, 0x8B, 0x8C, 0x8D, 0x8E, 0x8F,
      0x9D, 0x9E, 0x9F, 0xA0, 0xA1, 0xA3, 0xA4, 0xAA,
      0xB0, 0xB1, 0xB3, 0xBA, 0xBB
    ];

    const bitmapData = new Array(17).fill(0);
    bitmapData[0] = propertyEpcs.length; // Property count

    propertyEpcs.forEach(epc => {
      const offset = epc - 0x80;
      const i = offset & 0x0F;
      const j = (offset & 0xF0) >> 4;
      if (i < 16 && j < 8) {
        bitmapData[i + 1] |= (1 << j);
      }
    });

    const edt = btoa(String.fromCharCode(...bitmapData));
    const result = decodePropertyMap(edt);
    
    // Should return all EPCs in sorted order
    expect(result).toEqual([
      '80', '81', '88', '8A', '8B', '8C', '8D', '8E', '8F',
      '9D', '9E', '9F', 'A0', 'A1', 'A3', 'A4', 'AA',
      'B0', 'B1', 'B3', 'BA', 'BB'
    ]);
  });

  it('should handle EPC values with correct hex padding', () => {
    // Test single-digit hex values get padded correctly
    const epcs = [0x01, 0x0A, 0x80]; // Should become '01', '0A', '80'
    const data = [epcs.length, ...epcs];
    const edt = btoa(String.fromCharCode(...data));
    
    const result = decodePropertyMap(edt);
    
    expect(result).toEqual(['01', '0A', '80']);
  });
});

describe('Internationalization (i18n)', () => {
  // Mock getCurrentLocale function
  vi.mock('./languageHelper', () => ({
    getCurrentLocale: vi.fn(() => 'en')
  }));

  const mockPropertyDescriptions: Record<string, PropertyDescriptionData> = {
    '': {
      classCode: '',
      properties: {
        '80': { description: 'Operation Status' },
        '81': { description: 'Installation Location' }
      }
    },
    ':ja': {
      classCode: '',
      properties: {
        '80': { description: '動作状態' },
        '81': { description: '設置場所' }
      }
    },
    '0130': {
      classCode: '0130',
      properties: {
        'B0': { description: 'Set Temperature Value' }
      }
    },
    '0130:ja': {
      classCode: '0130',
      properties: {
        'B0': { description: '設定温度値' }
      }
    }
  };

  describe('getPropertyName', () => {
    it('should return English property names when lang is "en"', () => {
      expect(getPropertyName('80', mockPropertyDescriptions, '', 'en')).toBe('Operation Status');
      expect(getPropertyName('81', mockPropertyDescriptions, '', 'en')).toBe('Installation Location');
      expect(getPropertyName('B0', mockPropertyDescriptions, '0130', 'en')).toBe('Set Temperature Value');
    });

    it('should return Japanese property names when lang is "ja"', () => {
      expect(getPropertyName('80', mockPropertyDescriptions, '', 'ja')).toBe('動作状態');
      expect(getPropertyName('81', mockPropertyDescriptions, '', 'ja')).toBe('設置場所');
      expect(getPropertyName('B0', mockPropertyDescriptions, '0130', 'ja')).toBe('設定温度値');
    });

    it('should fallback to English when Japanese translation is not available', () => {
      // Test with a property that only has English description
      const limitedDescriptions = {
        '': {
          classCode: '',
          properties: {
            '80': { description: 'Operation Status' }
          }
        }
      };

      expect(getPropertyName('80', limitedDescriptions, '', 'ja')).toBe('Operation Status');
    });

    it('should fallback to EPC format when no description is found', () => {
      expect(getPropertyName('FF', mockPropertyDescriptions, '', 'en')).toBe('EPC FF');
      expect(getPropertyName('FF', mockPropertyDescriptions, '', 'ja')).toBe('EPC FF');
    });

    it('should prioritize class-specific properties over common properties', () => {
      const descriptionsWithConflict = {
        '': {
          classCode: '',
          properties: {
            'B0': { description: 'Common B0 Property' }
          }
        },
        '0130': {
          classCode: '0130',
          properties: {
            'B0': { description: 'Set Temperature Value' }
          }
        }
      };

      expect(getPropertyName('B0', descriptionsWithConflict, '0130', 'en')).toBe('Set Temperature Value');
    });
  });

  describe('getPropertyDescriptor', () => {
    const mockDescriptor: PropertyDescriptor = {
      description: 'Operation Status',
      aliases: { 'on': 'MA==', 'off': 'MQ==' }
    };

    const descriptionsWithDescriptor = {
      '': {
        classCode: '',
        properties: {
          '80': mockDescriptor
        }
      },
      ':ja': {
        classCode: '',
        properties: {
          '80': {
            description: '動作状態',
            aliases: { 'on': 'MA==', 'off': 'MQ==' },
            aliasTranslations: { 'on': 'オン', 'off': 'オフ' }
          }
        }
      }
    };

    it('should return English descriptor when lang is "en"', () => {
      const result = getPropertyDescriptor('80', descriptionsWithDescriptor, '', 'en');
      expect(result?.description).toBe('Operation Status');
    });

    it('should return Japanese descriptor when lang is "ja"', () => {
      const result = getPropertyDescriptor('80', descriptionsWithDescriptor, '', 'ja');
      expect(result?.description).toBe('動作状態');
      // Check aliasTranslations
      expect(result?.aliasTranslations?.on).toBe('オン');
      expect(result?.aliasTranslations?.off).toBe('オフ');
    });

    it('should fallback to English descriptor when Japanese is not available', () => {
      const result = getPropertyDescriptor('80', { '': descriptionsWithDescriptor[''] }, '', 'ja');
      expect(result?.description).toBe('Operation Status');
    });
  });

  describe('formatPropertyValue', () => {
    const descriptorWithTranslations: PropertyDescriptor = {
      description: 'Operation Status',
      aliases: {
        'on': 'MA==',
        'off': 'MQ=='
      },
      aliasTranslations: {
        'on': '動作中',
        'off': '停止中'
      }
    };

    it('should return English alias values when lang is "en"', () => {
      expect(formatPropertyValue({ string: 'on' }, descriptorWithTranslations, 'en')).toBe('on');
      expect(formatPropertyValue({ string: 'off' }, descriptorWithTranslations, 'en')).toBe('off');
    });

    it('should return Japanese translated alias values when lang is "ja"', () => {
      const descriptorWithTranslations: PropertyDescriptor = {
        description: 'Operation Status',
        aliases: {
          'on': 'MA==',
          'off': 'MQ=='
        },
        aliasTranslations: {
          'on': '動作中',
          'off': '停止中'
        }
      };
      expect(formatPropertyValue({ string: 'on' }, descriptorWithTranslations, 'ja')).toBe('動作中');
      expect(formatPropertyValue({ string: 'off' }, descriptorWithTranslations, 'ja')).toBe('停止中');
    });

    it('should fallback to original string when translation is not available', () => {
      expect(formatPropertyValue({ string: 'unknown' }, descriptorWithTranslations, 'ja')).toBe('unknown');
    });

    it('should handle numeric values with units', () => {
      const numericDescriptor: PropertyDescriptor = {
        description: 'Temperature',
        numberDesc: { min: 0, max: 50, offset: 0, unit: '°C', edtLen: 1 }
      };

      expect(formatPropertyValue({ number: 25 }, numericDescriptor, 'en')).toBe('25°C');
      expect(formatPropertyValue({ number: 25 }, numericDescriptor, 'ja')).toBe('25°C');
    });
  });
});