import { describe, it, expect } from 'vitest';
import { validateDeviceAlias } from './aliasHelper';

describe('validateDeviceAlias', () => {
  describe('valid aliases', () => {
    it('should accept simple alphanumeric aliases', () => {
      expect(validateDeviceAlias('kitchen_ac')).toBeUndefined();
      expect(validateDeviceAlias('AC1')).toBeUndefined();
      expect(validateDeviceAlias('living-room')).toBeUndefined();
      expect(validateDeviceAlias('エアコン1')).toBeUndefined();
    });

    it('should accept aliases starting with letters', () => {
      expect(validateDeviceAlias('ac123')).toBeUndefined();
      expect(validateDeviceAlias('test')).toBeUndefined();
    });

    it('should accept aliases starting with numbers', () => {
      expect(validateDeviceAlias('1st_floor_ac')).toBeUndefined();
      expect(validateDeviceAlias('2ndRoom')).toBeUndefined();
    });

    it('should accept odd-length hex strings', () => {
      expect(validateDeviceAlias('ABC')).toBeUndefined(); // 3 chars
      expect(validateDeviceAlias('12345')).toBeUndefined(); // 5 chars
    });
  });

  describe('invalid aliases', () => {
    it('should reject empty string', () => {
      expect(validateDeviceAlias('')).toBe('エイリアス名を入力してください');
    });

    it('should reject even-length hex strings', () => {
      expect(validateDeviceAlias('80')).toBe('16進数として読める偶数桁の名前は使用できません');
      expect(validateDeviceAlias('0130')).toBe('16進数として読める偶数桁の名前は使用できません');
      expect(validateDeviceAlias('ABCD')).toBe('16進数として読める偶数桁の名前は使用できません');
      expect(validateDeviceAlias('1234567890ABCDEF')).toBe('16進数として読める偶数桁の名前は使用できません');
    });

    it('should reject aliases starting with symbols', () => {
      expect(validateDeviceAlias('!test')).toBe('記号で始まる名前は使用できません');
      expect(validateDeviceAlias('@group')).toBe('記号で始まる名前は使用できません');
      expect(validateDeviceAlias('#tag')).toBe('記号で始まる名前は使用できません');
      expect(validateDeviceAlias('-dash')).toBe('記号で始まる名前は使用できません');
      expect(validateDeviceAlias('[bracket')).toBe('記号で始まる名前は使用できません');
      expect(validateDeviceAlias('{brace')).toBe('記号で始まる名前は使用できません');
    });

    it('should allow symbols in the middle or end', () => {
      expect(validateDeviceAlias('test-name')).toBeUndefined();
      expect(validateDeviceAlias('test_name')).toBeUndefined();
      expect(validateDeviceAlias('test!')).toBeUndefined();
    });
  });

  describe('edge cases', () => {
    it('should handle mixed case hex strings', () => {
      expect(validateDeviceAlias('aB')).toBe('16進数として読める偶数桁の名前は使用できません');
      expect(validateDeviceAlias('aBc')).toBeUndefined(); // odd length
    });

    it('should handle non-hex even-length strings', () => {
      expect(validateDeviceAlias('GH')).toBeUndefined(); // contains non-hex char
      expect(validateDeviceAlias('test')).toBeUndefined(); // contains non-hex chars
    });

    it('should handle Unicode characters', () => {
      expect(validateDeviceAlias('エアコン')).toBeUndefined();
      expect(validateDeviceAlias('🏠room')).toBe('記号で始まる名前は使用できません'); // emoji is considered a symbol
    });
  });
});