import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { isJapanese, getCurrentLocale } from './languageHelper';

describe('languageHelper', () => {
  let originalNavigatorLanguage: any;
  let originalNavigatorLanguages: any;

  beforeEach(() => {
    // Save original values
    originalNavigatorLanguage = Object.getOwnPropertyDescriptor(window.navigator, 'language');
    originalNavigatorLanguages = Object.getOwnPropertyDescriptor(window.navigator, 'languages');
  });

  afterEach(() => {
    // Restore original values
    if (originalNavigatorLanguage) {
      Object.defineProperty(window.navigator, 'language', originalNavigatorLanguage);
    }
    if (originalNavigatorLanguages) {
      Object.defineProperty(window.navigator, 'languages', originalNavigatorLanguages);
    }
  });

  const mockNavigatorLanguage = (language: string, languages?: string[]) => {
    Object.defineProperty(window.navigator, 'language', {
      value: language,
      writable: true,
      configurable: true
    });

    if (languages) {
      Object.defineProperty(window.navigator, 'languages', {
        value: languages,
        writable: true,
        configurable: true
      });
    }
  };

  describe('isJapanese', () => {
    it('should return true for Japanese language codes', () => {
      mockNavigatorLanguage('ja');
      expect(isJapanese()).toBe(true);

      mockNavigatorLanguage('ja-JP');
      expect(isJapanese()).toBe(true);

      mockNavigatorLanguage('JA-JP');
      expect(isJapanese()).toBe(true);
    });

    it('should return false for non-Japanese language codes', () => {
      mockNavigatorLanguage('en');
      expect(isJapanese()).toBe(false);

      mockNavigatorLanguage('en-US');
      expect(isJapanese()).toBe(false);

      mockNavigatorLanguage('zh-CN');
      expect(isJapanese()).toBe(false);
    });

    it('should use first language from languages array if language is not set', () => {
      mockNavigatorLanguage('', ['ja-JP', 'en-US']);
      expect(isJapanese()).toBe(true);

      mockNavigatorLanguage('', ['en-US', 'ja-JP']);
      expect(isJapanese()).toBe(false);
    });

    it('should default to false if no language is set', () => {
      mockNavigatorLanguage('', []);
      expect(isJapanese()).toBe(false);
    });
  });

  describe('getCurrentLocale', () => {
    it('should return "ja" for Japanese language', () => {
      mockNavigatorLanguage('ja-JP');
      expect(getCurrentLocale()).toBe('ja');
    });

    it('should return "ja" for various Japanese locales', () => {
      mockNavigatorLanguage('ja');
      expect(getCurrentLocale()).toBe('ja');

      mockNavigatorLanguage('ja-JP');
      expect(getCurrentLocale()).toBe('ja');

      mockNavigatorLanguage('JA');
      expect(getCurrentLocale()).toBe('ja');

      mockNavigatorLanguage('JA-JP');
      expect(getCurrentLocale()).toBe('ja');
    });

    it('should return "en" for non-Japanese language', () => {
      mockNavigatorLanguage('en-US');
      expect(getCurrentLocale()).toBe('en');

      mockNavigatorLanguage('fr-FR');
      expect(getCurrentLocale()).toBe('en');
    });

    it('should handle Japanese language in languages array', () => {
      mockNavigatorLanguage('', ['ja-JP', 'en-US']);
      expect(getCurrentLocale()).toBe('ja');

      mockNavigatorLanguage('', ['ja', 'en']);
      expect(getCurrentLocale()).toBe('ja');
    });

    it('should prioritize navigator.language over languages array', () => {
      mockNavigatorLanguage('ja-JP', ['en-US', 'fr-FR']);
      expect(getCurrentLocale()).toBe('ja');

      mockNavigatorLanguage('en-US', ['ja-JP', 'fr-FR']);
      expect(getCurrentLocale()).toBe('en');
    });
  });
});