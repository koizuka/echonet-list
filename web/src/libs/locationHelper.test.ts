import { hasAnyOperationalDevice } from './locationHelper';
import type { Device } from '@/hooks/types';

describe('hasAnyOperationalDevice', () => {
  const createDevice = (ip: string, eoj: string, operationStatus?: 'on' | 'off'): Device => ({
    ip,
    eoj,
    name: `${ip}-${eoj}`,
    id: `${eoj}:001:${ip}`,
    properties: operationStatus ? {
      '80': { string: operationStatus }
    } : {},
    lastSeen: '2023-01-01T00:00:00Z'
  });

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