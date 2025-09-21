import { describe, it, expect, vi } from 'vitest';
import { generateUniqueId, generateLogEntryId } from './idHelper';

describe('idHelper', () => {
  describe('generateUniqueId', () => {
    it('should generate unique IDs without prefix', () => {
      const id1 = generateUniqueId();
      const id2 = generateUniqueId();

      expect(id1).not.toBe(id2);
      expect(id1).toMatch(/^\d+-[a-z0-9]+$/);
      expect(id2).toMatch(/^\d+-[a-z0-9]+$/);
    });

    it('should generate unique IDs with prefix', () => {
      const id1 = generateUniqueId('error');
      const id2 = generateUniqueId('error');

      expect(id1).not.toBe(id2);
      expect(id1).toMatch(/^error-\d+-[a-z0-9]+$/);
      expect(id2).toMatch(/^error-\d+-[a-z0-9]+$/);
    });

    it('should handle different prefixes', () => {
      const errorId = generateUniqueId('error');
      const sliderId = generateUniqueId('slider-error');
      const infoId = generateUniqueId('info');

      expect(errorId).toMatch(/^error-\d+-[a-z0-9]+$/);
      expect(sliderId).toMatch(/^slider-error-\d+-[a-z0-9]+$/);
      expect(infoId).toMatch(/^info-\d+-[a-z0-9]+$/);
    });

    it('should include timestamp component', () => {
      const mockTime = 1640123456789;
      vi.useFakeTimers();
      vi.setSystemTime(mockTime);

      const id = generateUniqueId('test');
      expect(id).toContain(mockTime.toString());

      vi.useRealTimers();
    });

    it('should generate different IDs even when called rapidly', () => {
      const ids = new Set();

      // Generate multiple IDs in rapid succession
      for (let i = 0; i < 100; i++) {
        ids.add(generateUniqueId());
      }

      // All IDs should be unique
      expect(ids.size).toBe(100);
    });
  });

  describe('generateLogEntryId', () => {
    it('should generate IDs formatted for LogEntry usage', () => {
      const errorId = generateLogEntryId('error');
      const warnId = generateLogEntryId('warn');
      const infoId = generateLogEntryId('info');

      expect(errorId).toMatch(/^error-\d+-[a-z0-9]+$/);
      expect(warnId).toMatch(/^warn-\d+-[a-z0-9]+$/);
      expect(infoId).toMatch(/^info-\d+-[a-z0-9]+$/);
    });

    it('should generate unique IDs for same type', () => {
      const id1 = generateLogEntryId('error');
      const id2 = generateLogEntryId('error');

      expect(id1).not.toBe(id2);
    });

    it('should handle various log entry types', () => {
      const types = ['online', 'offline', 'slider-error', 'validation-error', 'network-error'];
      const ids = types.map(type => generateLogEntryId(type));

      // All should be unique
      const uniqueIds = new Set(ids);
      expect(uniqueIds.size).toBe(types.length);

      // All should match expected format
      ids.forEach((id, index) => {
        expect(id).toMatch(new RegExp(`^${types[index]}-\\d+-[a-z0-9]+$`));
      });
    });
  });
});