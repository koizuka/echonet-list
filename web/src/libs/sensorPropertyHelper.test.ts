import { describe, it, expect } from 'vitest';
import {
  Thermometer,
  CloudSun,
  Home,
  ThermometerSun,
  ThermometerSnowflake,
  Droplets
} from 'lucide-react';
import {
  isSensorProperty,
  getSensorIcon,
  getSensorEPCs
} from './sensorPropertyHelper';

describe('sensorPropertyHelper', () => {
  describe('isSensorProperty', () => {
    it('should return true for room temperature EPCs', () => {
      expect(isSensorProperty('BB')).toBe(true); // Air Conditioner
      expect(isSensorProperty('E2')).toBe(true); // Floor Heating
    });

    it('should return true for outside temperature EPC', () => {
      expect(isSensorProperty('BE')).toBe(true);
    });

    it('should return true for floor temperature EPC', () => {
      expect(isSensorProperty('E3')).toBe(true);
    });

    it('should return true for water temperature sensor EPCs', () => {
      expect(isSensorProperty('F3')).toBe(true); // Temperature sensor 1
      expect(isSensorProperty('F4')).toBe(true); // Temperature sensor 2
    });

    it('should return true for humidity EPC', () => {
      expect(isSensorProperty('BA')).toBe(true);
    });

    it('should return false for non-sensor EPCs', () => {
      expect(isSensorProperty('80')).toBe(false); // Operation status
      expect(isSensorProperty('B3')).toBe(false); // Temperature setting
      expect(isSensorProperty('B4')).toBe(false); // Humidity setting
      expect(isSensorProperty('B0')).toBe(false); // Operation mode
      expect(isSensorProperty('XX')).toBe(false); // Unknown EPC
    });
  });

  describe('getSensorIcon', () => {
    it('should return Thermometer for room temperature EPCs', () => {
      expect(getSensorIcon('BB')).toBe(Thermometer);
      expect(getSensorIcon('E2')).toBe(Thermometer);
    });

    it('should return CloudSun for outside temperature', () => {
      expect(getSensorIcon('BE')).toBe(CloudSun);
    });

    it('should return Home for floor temperature', () => {
      expect(getSensorIcon('E3')).toBe(Home);
    });

    it('should return ThermometerSun for temperature sensor 1', () => {
      expect(getSensorIcon('F3')).toBe(ThermometerSun);
    });

    it('should return ThermometerSnowflake for temperature sensor 2', () => {
      expect(getSensorIcon('F4')).toBe(ThermometerSnowflake);
    });

    it('should return Droplets for humidity', () => {
      expect(getSensorIcon('BA')).toBe(Droplets);
    });

    it('should return undefined for non-sensor EPCs', () => {
      expect(getSensorIcon('80')).toBeUndefined();
      expect(getSensorIcon('B3')).toBeUndefined();
      expect(getSensorIcon('XX')).toBeUndefined();
    });
  });

  describe('getSensorEPCs', () => {
    it('should return all sensor EPCs', () => {
      const sensorEPCs = getSensorEPCs();
      expect(sensorEPCs).toContain('BB');
      expect(sensorEPCs).toContain('E2');
      expect(sensorEPCs).toContain('BE');
      expect(sensorEPCs).toContain('E3');
      expect(sensorEPCs).toContain('F3');
      expect(sensorEPCs).toContain('F4');
      expect(sensorEPCs).toContain('BA');
      expect(sensorEPCs).toHaveLength(7);
    });

    it('should not contain non-sensor EPCs', () => {
      const sensorEPCs = getSensorEPCs();
      expect(sensorEPCs).not.toContain('80');
      expect(sensorEPCs).not.toContain('B3');
      expect(sensorEPCs).not.toContain('B4');
    });
  });
});