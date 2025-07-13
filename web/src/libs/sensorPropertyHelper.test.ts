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
  getSensorIconColor,
  getSensorEPCs,
  isTemperatureSensor
} from './sensorPropertyHelper';
import type { PropertyValue } from '@/hooks/types';

describe('sensorPropertyHelper', () => {
  describe('isSensorProperty', () => {
    it('should return true for valid sensor properties with classCode:EPC format', () => {
      // Air Conditioner sensors
      expect(isSensorProperty('0130', 'BB')).toBe(true); // Room temperature
      expect(isSensorProperty('0130', 'BE')).toBe(true); // Outside temperature
      expect(isSensorProperty('0130', 'BA')).toBe(true); // Room humidity
      
      // Floor Heating sensors
      expect(isSensorProperty('027B', 'E2')).toBe(true); // Room temperature
      expect(isSensorProperty('027B', 'E3')).toBe(true); // Floor temperature
      expect(isSensorProperty('027B', 'F3')).toBe(true); // Temperature sensor 1
      expect(isSensorProperty('027B', 'F4')).toBe(true); // Temperature sensor 2
    });

    it('should return false for non-sensor properties', () => {
      expect(isSensorProperty('0130', '80')).toBe(false); // Power status
      expect(isSensorProperty('0130', 'B0')).toBe(false); // Operation mode
      expect(isSensorProperty('027B', '81')).toBe(false); // Installation location
    });

    it('should return false for wrong classCode combinations', () => {
      // EPC BB exists for Air Conditioner but not for Floor Heating
      expect(isSensorProperty('027B', 'BB')).toBe(false);
      // EPC E2 exists for Floor Heating but not for Air Conditioner
      expect(isSensorProperty('0130', 'E2')).toBe(false);
    });

    it('should return false for unknown classCode', () => {
      expect(isSensorProperty('FFFF', 'BB')).toBe(false);
      expect(isSensorProperty('0000', 'E2')).toBe(false);
    });
  });

  describe('getSensorIcon', () => {
    it('should return correct icons for temperature sensors', () => {
      expect(getSensorIcon('0130', 'BB')).toBe(Thermometer);
      expect(getSensorIcon('027B', 'E2')).toBe(Thermometer);
      expect(getSensorIcon('0130', 'BE')).toBe(CloudSun);
      expect(getSensorIcon('027B', 'E3')).toBe(Home);
      expect(getSensorIcon('027B', 'F3')).toBe(ThermometerSun);
      expect(getSensorIcon('027B', 'F4')).toBe(ThermometerSnowflake);
    });

    it('should return correct icon for humidity sensor', () => {
      expect(getSensorIcon('0130', 'BA')).toBe(Droplets);
    });

    it('should return undefined for non-sensor properties', () => {
      expect(getSensorIcon('0130', '80')).toBeUndefined();
      expect(getSensorIcon('027B', '81')).toBeUndefined();
    });

    it('should return undefined for wrong classCode combinations', () => {
      expect(getSensorIcon('027B', 'BB')).toBeUndefined();
      expect(getSensorIcon('0130', 'E2')).toBeUndefined();
    });
  });

  describe('isTemperatureSensor', () => {
    it('should return true for temperature sensor properties', () => {
      expect(isTemperatureSensor('0130', 'BB')).toBe(true);
      expect(isTemperatureSensor('027B', 'E2')).toBe(true);
      expect(isTemperatureSensor('0130', 'BE')).toBe(true);
      expect(isTemperatureSensor('027B', 'E3')).toBe(true);
      expect(isTemperatureSensor('027B', 'F3')).toBe(true);
      expect(isTemperatureSensor('027B', 'F4')).toBe(true);
    });

    it('should return false for non-temperature sensors', () => {
      expect(isTemperatureSensor('0130', 'BA')).toBe(false); // Humidity
    });

    it('should return false for non-sensor properties', () => {
      expect(isTemperatureSensor('0130', '80')).toBe(false);
      expect(isTemperatureSensor('027B', '81')).toBe(false);
    });
  });

  describe('getSensorIconColor', () => {
    it('should return blue colors for cold temperatures', () => {
      const coldValue: PropertyValue = { number: 5 };
      const veryColdValue: PropertyValue = { number: -10 };
      
      expect(getSensorIconColor('0130', 'BB', coldValue)).toBe('text-blue-600');
      expect(getSensorIconColor('027B', 'E2', veryColdValue)).toBe('text-blue-600');
      
      const coolValue: PropertyValue = { number: 12 };
      expect(getSensorIconColor('0130', 'BB', coolValue)).toBe('text-blue-400');
    });

    it('should return red colors for hot temperatures', () => {
      const hotValue: PropertyValue = { number: 35 };
      const warmValue: PropertyValue = { number: 27 };
      
      expect(getSensorIconColor('0130', 'BB', hotValue)).toBe('text-red-600');
      expect(getSensorIconColor('027B', 'E2', warmValue)).toBe('text-orange-400');
    });

    it('should return muted color for normal temperatures', () => {
      const normalValue: PropertyValue = { number: 22 };
      
      expect(getSensorIconColor('0130', 'BB', normalValue)).toBe('text-muted-foreground');
      expect(getSensorIconColor('027B', 'E2', normalValue)).toBe('text-muted-foreground');
    });

    it('should return muted color for non-temperature sensors', () => {
      const humidityValue: PropertyValue = { number: 60 };
      
      expect(getSensorIconColor('0130', 'BA', humidityValue)).toBe('text-muted-foreground');
    });

    it('should return muted color for non-sensor properties', () => {
      const value: PropertyValue = { number: 25 };
      
      expect(getSensorIconColor('0130', '80', value)).toBe('text-muted-foreground');
      expect(getSensorIconColor('027B', '81', value)).toBe('text-muted-foreground');
    });

    it('should return muted color for non-numeric values', () => {
      const stringValue: PropertyValue = { string: 'ON' };
      const edtValue: PropertyValue = { EDT: 'AQ==' }; // Base64 encoded value
      
      expect(getSensorIconColor('0130', 'BB', stringValue)).toBe('text-muted-foreground');
      expect(getSensorIconColor('0130', 'BB', edtValue)).toBe('text-muted-foreground');
    });

    it('should handle edge temperature values correctly', () => {
      // Boundary testing
      expect(getSensorIconColor('0130', 'BB', { number: 10 })).toBe('text-blue-600');
      expect(getSensorIconColor('0130', 'BB', { number: 11 })).toBe('text-blue-400');
      expect(getSensorIconColor('0130', 'BB', { number: 15 })).toBe('text-blue-400');
      expect(getSensorIconColor('0130', 'BB', { number: 16 })).toBe('text-muted-foreground');
      expect(getSensorIconColor('0130', 'BB', { number: 24 })).toBe('text-muted-foreground');
      expect(getSensorIconColor('0130', 'BB', { number: 25 })).toBe('text-orange-400');
      expect(getSensorIconColor('0130', 'BB', { number: 29 })).toBe('text-orange-400');
      expect(getSensorIconColor('0130', 'BB', { number: 30 })).toBe('text-red-600');
    });
  });

  describe('getSensorEPCs', () => {
    it('should return all sensor EPCs with their classCode mappings', () => {
      const sensorEPCs = getSensorEPCs();
      
      expect(sensorEPCs).toContain('0130:BB');
      expect(sensorEPCs).toContain('0130:BE');
      expect(sensorEPCs).toContain('0130:BA');
      expect(sensorEPCs).toContain('027B:E2');
      expect(sensorEPCs).toContain('027B:E3');
      expect(sensorEPCs).toContain('027B:F3');
      expect(sensorEPCs).toContain('027B:F4');
    });

    it('should return unique values', () => {
      const sensorEPCs = getSensorEPCs();
      const uniqueEPCs = [...new Set(sensorEPCs)];
      
      expect(sensorEPCs.length).toBe(uniqueEPCs.length);
    });
  });
});