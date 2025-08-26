import { describe, it, expect } from 'vitest';
import {
  AirVent,
  Heater,
  Lightbulb,
  LampCeiling,
  ThermometerSun,
  Refrigerator,
  Settings,
  Info,
  CircleHelp
} from 'lucide-react';
import { getDeviceIcon, getDeviceIconColor } from './deviceIconHelper';

describe('deviceIconHelper', () => {
  describe('getDeviceIcon', () => {
    it('should return AirVent icon for air conditioner (0130)', () => {
      expect(getDeviceIcon('0130')).toBe(AirVent);
    });

    it('should return Heater icon for floor heating (027B)', () => {
      expect(getDeviceIcon('027B')).toBe(Heater);
    });

    it('should return Lightbulb icon for single function lighting (0291)', () => {
      expect(getDeviceIcon('0291')).toBe(Lightbulb);
    });

    it('should return LampCeiling icon for lighting system (02A3)', () => {
      expect(getDeviceIcon('02A3')).toBe(LampCeiling);
    });

    it('should return ThermometerSun icon for water heater (026B)', () => {
      expect(getDeviceIcon('026B')).toBe(ThermometerSun);
    });

    it('should return Heater icon for bath room heater (0272)', () => {
      expect(getDeviceIcon('0272')).toBe(Heater);
    });

    it('should return Refrigerator icon for refrigerator (03B7)', () => {
      expect(getDeviceIcon('03B7')).toBe(Refrigerator);
    });

    it('should return Settings icon for controller (05FF)', () => {
      expect(getDeviceIcon('05FF')).toBe(Settings);
    });

    it('should return Info icon for node profile (0EF0)', () => {
      expect(getDeviceIcon('0EF0')).toBe(Info);
    });

    it('should return CircleHelp icon for unknown device class', () => {
      expect(getDeviceIcon('9999')).toBe(CircleHelp);
    });
  });

  describe('getDeviceIconColor', () => {
    it('should return muted color for offline device', () => {
      expect(getDeviceIconColor(true, false, true)).toBe('text-muted-foreground');
      expect(getDeviceIconColor(false, true, true)).toBe('text-muted-foreground');
    });

    it('should return red color for faulty device', () => {
      expect(getDeviceIconColor(true, true, false)).toBe('text-red-500');
      expect(getDeviceIconColor(false, true, false)).toBe('text-red-500');
    });

    it('should return green color for operational controllable device', () => {
      expect(getDeviceIconColor(true, false, false, true)).toBe('text-green-500');
    });

    it('should return gray color for operational non-controllable device', () => {
      expect(getDeviceIconColor(true, false, false, false)).toBe('text-gray-400');
    });

    it('should return gray color for non-operational device', () => {
      expect(getDeviceIconColor(false, false, false)).toBe('text-gray-400');
    });

    it('should show fault state even for non-controllable devices', () => {
      expect(getDeviceIconColor(false, true, false, false)).toBe('text-red-500');
    });

    it('should show offline state even for non-controllable devices', () => {
      expect(getDeviceIconColor(false, false, true, false)).toBe('text-muted-foreground');
    });
  });
});