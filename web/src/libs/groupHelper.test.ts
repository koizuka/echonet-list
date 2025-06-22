import { describe, it, expect } from 'vitest';
import { validateGroupName } from './groupHelper';

describe('validateGroupName', () => {
  describe('without existing groups', () => {
    it('should return undefined for valid group names', () => {
      expect(validateGroupName('@group1')).toBeUndefined();
      expect(validateGroupName('@テストグループ')).toBeUndefined();
      expect(validateGroupName('@123')).toBeUndefined();
      expect(validateGroupName('@group-name')).toBeUndefined();
      expect(validateGroupName('@group_name')).toBeUndefined();
    });

    it('should return error for names not starting with @', () => {
      expect(validateGroupName('group1')).toBe('グループ名は @ で始まる必要があります');
      expect(validateGroupName('!group1')).toBe('グループ名は @ で始まる必要があります');
      expect(validateGroupName('')).toBe('グループ名は @ で始まる必要があります');
    });

    it('should return error for @ only', () => {
      expect(validateGroupName('@')).toBe('グループ名は @ の後に少なくとも1文字必要です');
    });

    it('should return error for names with whitespace', () => {
      expect(validateGroupName('@group 1')).toBe('グループ名に空白文字を含めることはできません');
      expect(validateGroupName('@group\t1')).toBe('グループ名に空白文字を含めることはできません');
      expect(validateGroupName('@group\n1')).toBe('グループ名に空白文字を含めることはできません');
      expect(validateGroupName('@group\r1')).toBe('グループ名に空白文字を含めることはできません');
      expect(validateGroupName('@ group')).toBe('グループ名に空白文字を含めることはできません');
    });
  });

  describe('with existing groups', () => {
    const existingGroups = ['@group1', '@group2', '@テストグループ'];

    it('should return undefined for new valid group names', () => {
      expect(validateGroupName('@group3', existingGroups)).toBeUndefined();
      expect(validateGroupName('@新しいグループ', existingGroups)).toBeUndefined();
    });

    it('should return error for duplicate group names', () => {
      expect(validateGroupName('@group1', existingGroups)).toBe('このグループ名は既に使用されています');
      expect(validateGroupName('@group2', existingGroups)).toBe('このグループ名は既に使用されています');
      expect(validateGroupName('@テストグループ', existingGroups)).toBe('このグループ名は既に使用されています');
    });

    it('should still validate format even with existing groups', () => {
      expect(validateGroupName('group3', existingGroups)).toBe('グループ名は @ で始まる必要があります');
      expect(validateGroupName('@', existingGroups)).toBe('グループ名は @ の後に少なくとも1文字必要です');
      expect(validateGroupName('@group 3', existingGroups)).toBe('グループ名に空白文字を含めることはできません');
    });
  });
});