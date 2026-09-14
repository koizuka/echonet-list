import '@testing-library/jest-dom/vitest';

// Mock ResizeObserver for tests (needed by Radix UI components)
global.ResizeObserver = class ResizeObserver {
  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
  }

  callback: ResizeObserverCallback;

  observe() {}
  unobserve() {}
  disconnect() {}
};
