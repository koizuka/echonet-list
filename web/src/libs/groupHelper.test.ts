import { describe, it, expect, vi, beforeEach } from 'vitest';
import { validateGroupName, getUnusedGroupName } from './groupHelper';
import { getCurrentLocale } from '@/libs/languageHelper';

// Existing assertions use Japanese messages; English is covered separately
vi.mock('@/libs/languageHelper', () => ({
  getCurrentLocale: vi.fn(() => 'ja'),
}));

describe('getUnusedGroupName', () => {
  it('should return the base name when it is unused', () => {
    expect(getUnusedGroupName('@new-group', ['@living'])).toBe('@new-group');
  });

  it('should append a numeric suffix when the base name is taken', () => {
    expect(getUnusedGroupName('@new-group', ['@new-group'])).toBe('@new-group-1');
    expect(getUnusedGroupName('@new-group', ['@new-group', '@new-group-1'])).toBe('@new-group-2');
  });
});

describe('validateGroupName', () => {
  beforeEach(() => {
    // mockReturnValue persists across tests, so reset the locale for each test
    vi.mocked(getCurrentLocale).mockReturnValue('ja');
  });

  describe('English locale', () => {
    it('should return English messages', () => {
      vi.mocked(getCurrentLocale).mockReturnValue('en');
      expect(validateGroupName('group1')).toBe('Group names must start with @');
      expect(validateGroupName('@')).toBe('Group names need at least one character after @');
      expect(validateGroupName('@a b')).toBe('Group names cannot contain whitespace');
      expect(validateGroupName('@g', ['@g'])).toBe('This group name is already in use');
    });
  });

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