import { describe, it, expect } from 'vitest';
import { arrangeDashboardDevices, isPlaceholder, DashboardLayoutItem } from './dashboardLayoutHelper';
import type { Device } from '@/hooks/types';

// Helper to create mock device
function createMockDevice(eoj: string, ip: string = '192.168.1.1'): Device {
  return {
    ip,
    eoj,
    name: `Device ${eoj}`,
    id: `${eoj}:0001:001`,
    properties: {},
    lastSeen: new Date().toISOString(),
  };
}

// Helper to extract device EOJs or 'placeholder' from layout
function getLayoutPattern(items: DashboardLayoutItem[]): string[] {
  return items.map(item => isPlaceholder(item) ? 'placeholder' : item.eoj);
}

describe('arrangeDashboardDevices', () => {
  describe('air conditioner and floor heater pairing', () => {
    it('should pair 1 air conditioner with 1 floor heater', () => {
      const devices = [
        createMockDevice('0130:1'), // Air conditioner
        createMockDevice('027B:1'), // Floor heater
      ];

      const result = arrangeDashboardDevices(devices);

      expect(getLayoutPattern(result)).toEqual(['0130:1', '027B:1']);
    });

    it('should pair 2 air conditioners with 1 floor heater (placeholder for missing floor heater)', () => {
      const devices = [
        createMockDevice('0130:1'), // Air conditioner 1
        createMockDevice('0130:2'), // Air conditioner 2
        createMockDevice('027B:1'), // Floor heater
      ];

      const result = arrangeDashboardDevices(devices);

      // Row 1: AC1 | FH1
      // Row 2: AC2 | placeholder
      expect(getLayoutPattern(result)).toEqual(['0130:1', '027B:1', '0130:2', 'placeholder']);
    });

    it('should pair 1 air conditioner with 2 floor heaters (placeholder for missing air conditioner)', () => {
      const devices = [
        createMockDevice('0130:1'), // Air conditioner
        createMockDevice('027B:1'), // Floor heater 1
        createMockDevice('027B:2'), // Floor heater 2
      ];

      const result = arrangeDashboardDevices(devices);

      // Row 1: AC1 | FH1
      // Row 2: placeholder | FH2
      expect(getLayoutPattern(result)).toEqual(['0130:1', '027B:1', 'placeholder', '027B:2']);
    });

    it('should handle air conditioner only (placeholder on right)', () => {
      const devices = [
        createMockDevice('0130:1'), // Air conditioner only
      ];

      const result = arrangeDashboardDevices(devices);

      // Row 1: AC | placeholder
      expect(getLayoutPattern(result)).toEqual(['0130:1', 'placeholder']);
    });

    it('should handle floor heater only (placeholder on left)', () => {
      const devices = [
        createMockDevice('027B:1'), // Floor heater only
      ];

      const result = arrangeDashboardDevices(devices);

      // Row 1: placeholder | FH
      expect(getLayoutPattern(result)).toEqual(['placeholder', '027B:1']);
    });

    it('should handle 2 air conditioners with 3 floor heaters', () => {
      const devices = [
        createMockDevice('0130:1'),
        createMockDevice('0130:2'),
        createMockDevice('027B:1'),
        createMockDevice('027B:2'),
        createMockDevice('027B:3'),
      ];

      const result = arrangeDashboardDevices(devices);

      // Row 1: AC1 | FH1
      // Row 2: AC2 | FH2
      // Row 3: placeholder | FH3
      expect(getLayoutPattern(result)).toEqual([
        '0130:1', '027B:1',
        '0130:2', '027B:2',
        'placeholder', '027B:3'
      ]);
    });
  });

  describe('other devices placement', () => {
    it('should place other devices after air conditioner/floor heater pairs', () => {
      const devices = [
        createMockDevice('0130:1'), // Air conditioner
        createMockDevice('027B:1'), // Floor heater
        createMockDevice('0291:1'), // Lighting
        createMockDevice('0291:2'), // Lighting
      ];

      const result = arrangeDashboardDevices(devices);

      // Row 1: AC | FH
      // Row 2: Light1 | Light2
      expect(getLayoutPattern(result)).toEqual(['0130:1', '027B:1', '0291:1', '0291:2']);
    });

    it('should handle only other devices (no special arrangement)', () => {
      const devices = [
        createMockDevice('0291:1'), // Lighting 1
        createMockDevice('0291:2'), // Lighting 2
      ];

      const result = arrangeDashboardDevices(devices);

      // No special arrangement, just sorted by EOJ
      expect(getLayoutPattern(result)).toEqual(['0291:1', '0291:2']);
    });

    it('should handle mixed devices with uneven AC/FH counts', () => {
      const devices = [
        createMockDevice('0130:1'), // Air conditioner
        createMockDevice('027B:1'), // Floor heater 1
        createMockDevice('027B:2'), // Floor heater 2
        createMockDevice('0291:1'), // Lighting
      ];

      const result = arrangeDashboardDevices(devices);

      // Row 1: AC | FH1
      // Row 2: placeholder | FH2
      // Row 3: Light1 (normal flow)
      expect(getLayoutPattern(result)).toEqual([
        '0130:1', '027B:1',
        'placeholder', '027B:2',
        '0291:1'
      ]);
    });
  });

  describe('sorting within categories', () => {
    it('should sort air conditioners by EOJ', () => {
      const devices = [
        createMockDevice('0130:2'), // Air conditioner 2 (added first)
        createMockDevice('0130:1'), // Air conditioner 1
        createMockDevice('027B:1'), // Floor heater
      ];

      const result = arrangeDashboardDevices(devices);

      // AC1 should come before AC2
      expect(getLayoutPattern(result)).toEqual(['0130:1', '027B:1', '0130:2', 'placeholder']);
    });

    it('should sort floor heaters by EOJ', () => {
      const devices = [
        createMockDevice('0130:1'), // Air conditioner
        createMockDevice('027B:2'), // Floor heater 2 (added first)
        createMockDevice('027B:1'), // Floor heater 1
      ];

      const result = arrangeDashboardDevices(devices);

      // FH1 should come before FH2
      expect(getLayoutPattern(result)).toEqual(['0130:1', '027B:1', 'placeholder', '027B:2']);
    });

    it('should sort other devices by EOJ', () => {
      const devices = [
        createMockDevice('0291:2'), // Lighting 2
        createMockDevice('0291:1'), // Lighting 1
      ];

      const result = arrangeDashboardDevices(devices);

      expect(getLayoutPattern(result)).toEqual(['0291:1', '0291:2']);
    });
  });

  describe('edge cases', () => {
    it('should handle empty device array', () => {
      const result = arrangeDashboardDevices([]);
      expect(result).toEqual([]);
    });

    it('should handle multiple floor heaters only', () => {
      const devices = [
        createMockDevice('027B:1'),
        createMockDevice('027B:2'),
      ];

      const result = arrangeDashboardDevices(devices);

      // Row 1: placeholder | FH1
      // Row 2: placeholder | FH2
      expect(getLayoutPattern(result)).toEqual([
        'placeholder', '027B:1',
        'placeholder', '027B:2'
      ]);
    });

    it('should handle multiple air conditioners only', () => {
      const devices = [
        createMockDevice('0130:1'),
        createMockDevice('0130:2'),
      ];

      const result = arrangeDashboardDevices(devices);

      // Row 1: AC1 | placeholder
      // Row 2: AC2 | placeholder
      expect(getLayoutPattern(result)).toEqual([
        '0130:1', 'placeholder',
        '0130:2', 'placeholder'
      ]);
    });
  });
});

describe('isPlaceholder', () => {
  it('should return true for placeholder object', () => {
    expect(isPlaceholder({ type: 'placeholder' })).toBe(true);
  });

  it('should return false for device object', () => {
    const device = createMockDevice('0130:1');
    expect(isPlaceholder(device)).toBe(false);
  });
});
