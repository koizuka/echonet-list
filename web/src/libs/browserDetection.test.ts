import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { isIOSSafari, getUserAgent } from './browserDetection';

describe('browserDetection', () => {
  let originalNavigator: Navigator;

  beforeEach(() => {
    originalNavigator = global.navigator;
  });

  afterEach(() => {
    global.navigator = originalNavigator;
  });

  describe('isIOSSafari', () => {
    it('should return true for iOS Safari', () => {
      Object.defineProperty(global, 'navigator', {
        value: {
          userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1',
        },
        writable: true,
        configurable: true,
      });

      expect(isIOSSafari()).toBe(true);
    });

    it('should return true for iPad Safari', () => {
      Object.defineProperty(global, 'navigator', {
        value: {
          userAgent: 'Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1',
        },
        writable: true,
        configurable: true,
      });

      expect(isIOSSafari()).toBe(true);
    });

    it('should return false for Chrome on iOS', () => {
      Object.defineProperty(global, 'navigator', {
        value: {
          userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/119.0.6045.109 Mobile/15E148 Safari/604.1',
        },
        writable: true,
        configurable: true,
      });

      expect(isIOSSafari()).toBe(false);
    });

    it('should return false for Firefox on iOS', () => {
      Object.defineProperty(global, 'navigator', {
        value: {
          userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/119.0 Mobile/15E148 Safari/605.1.15',
        },
        writable: true,
        configurable: true,
      });

      expect(isIOSSafari()).toBe(false);
    });

    it('should return false for Edge on iOS', () => {
      Object.defineProperty(global, 'navigator', {
        value: {
          userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) EdgiOS/119.0.2151.65 Mobile/15E148 Safari/605.1.15',
        },
        writable: true,
        configurable: true,
      });

      expect(isIOSSafari()).toBe(false);
    });

    it('should return false for Opera on iOS', () => {
      Object.defineProperty(global, 'navigator', {
        value: {
          userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) OPiOS/3.2.28 Mobile/15E148 Safari/9537.53',
        },
        writable: true,
        configurable: true,
      });

      expect(isIOSSafari()).toBe(false);
    });

    it('should return false for desktop Safari', () => {
      Object.defineProperty(global, 'navigator', {
        value: {
          userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15',
        },
        writable: true,
        configurable: true,
      });

      expect(isIOSSafari()).toBe(false);
    });

    it('should return false for Chrome on desktop', () => {
      Object.defineProperty(global, 'navigator', {
        value: {
          userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36',
        },
        writable: true,
        configurable: true,
      });

      expect(isIOSSafari()).toBe(false);
    });

    it('should return false when navigator is undefined', () => {
      Object.defineProperty(global, 'navigator', {
        value: undefined,
        writable: true,
        configurable: true,
      });

      expect(isIOSSafari()).toBe(false);
    });
  });

  describe('getUserAgent', () => {
    it('should return the user agent string', () => {
      const testUA = 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Safari/604.1';
      Object.defineProperty(global, 'navigator', {
        value: {
          userAgent: testUA,
        },
        writable: true,
        configurable: true,
      });

      expect(getUserAgent()).toBe(testUA);
    });

    it('should return empty string when navigator is undefined', () => {
      Object.defineProperty(global, 'navigator', {
        value: undefined,
        writable: true,
        configurable: true,
      });

      expect(getUserAgent()).toBe('');
    });
  });
});
