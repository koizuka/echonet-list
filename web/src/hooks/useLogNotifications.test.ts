import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useLogNotifications } from './useLogNotifications';
import type { LogNotification, DeviceOnline, DeviceOffline } from './types';

const createLogNotification = (overrides: Partial<LogNotification['payload']> = {}): LogNotification => ({
  type: 'log_notification',
  payload: {
    level: 'ERROR',
    message: 'Test message',
    time: '2023-04-01T12:00:00Z',
    attributes: {},
    ...overrides
  }
});

const createDeviceOnline = (overrides: Partial<DeviceOnline['payload']> = {}): DeviceOnline => ({
  type: 'device_online',
  payload: {
    ip: '192.168.1.100',
    eoj: '0291:1',
    ...overrides
  }
});

const createDeviceOffline = (overrides: Partial<DeviceOffline['payload']> = {}): DeviceOffline => ({
  type: 'device_offline',
  payload: {
    ip: '192.168.1.101',
    eoj: '0130:1',
    ...overrides
  }
});

describe('useLogNotifications', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('starts with empty logs', () => {
    const { result } = renderHook(() => useLogNotifications({}));

    expect(result.current.logs).toEqual([]);
  });

  it('adds new log entry when notification is received', () => {
    const { result } = renderHook(() => useLogNotifications({}));
    const notification = createLogNotification({ message: 'Test error message', attributes: { device: '192.168.1.1' } });

    act(() => {
      result.current.handleLogNotification(notification);
    });

    expect(result.current.logs).toHaveLength(1);
    expect(result.current.logs[0].message).toBe('Test error message');
    expect(result.current.logs[0].level).toBe('ERROR');
    expect(result.current.logs[0].isRead).toBe(false);
  });

  it('maintains max log limit', () => {
    const { result } = renderHook(() => useLogNotifications({ maxLogs: 3 }));

    act(() => {
      for (let i = 1; i <= 5; i++) {
        result.current.handleLogNotification(createLogNotification({ message: `Message ${i}` }));
      }
    });

    expect(result.current.logs).toHaveLength(3);
    expect(result.current.logs[0].message).toBe('Message 5');
    expect(result.current.logs[1].message).toBe('Message 4');
    expect(result.current.logs[2].message).toBe('Message 3');
  });

  it('marks all logs as read', () => {
    const { result } = renderHook(() => useLogNotifications({}));

    act(() => {
      result.current.handleLogNotification(createLogNotification({ message: 'Message 1' }));
      result.current.handleLogNotification(createLogNotification({ message: 'Message 2', level: 'WARN' }));
    });

    expect(result.current.logs).toHaveLength(2);
    expect(result.current.logs.every(log => !log.isRead)).toBe(true);

    act(() => {
      result.current.markAllAsRead();
    });

    expect(result.current.logs.every(log => log.isRead)).toBe(true);
  });

  it('clears all logs', () => {
    const { result } = renderHook(() => useLogNotifications({}));

    act(() => {
      result.current.handleLogNotification(createLogNotification());
    });

    expect(result.current.logs).toHaveLength(1);

    act(() => {
      result.current.clearAllLogs();
    });

    expect(result.current.logs).toHaveLength(0);
  });

  it('clears logs by category', () => {
    const { result } = renderHook(() => useLogNotifications({}));

    act(() => {
      result.current.handleLogNotification(createLogNotification({ message: 'WebSocket error', attributes: { component: 'WebSocket' } }));
      result.current.handleLogNotification(createLogNotification({ message: 'Other error', attributes: { component: 'Other' } }));
    });

    expect(result.current.logs).toHaveLength(2);

    act(() => {
      result.current.clearByCategory('WebSocket');
    });

    expect(result.current.logs).toHaveLength(1);
    expect(result.current.logs[0].message).toBe('Other error');
  });

  it('calls onLogsChange callback when logs change', () => {
    const onLogsChange = vi.fn();
    const { result } = renderHook(() => useLogNotifications({ onLogsChange }));

    expect(onLogsChange).toHaveBeenCalledWith([], 0);

    act(() => {
      result.current.handleLogNotification(createLogNotification({ message: 'Test message' }));
    });

    expect(onLogsChange).toHaveBeenLastCalledWith(
      expect.arrayContaining([
        expect.objectContaining({
          message: 'Test message',
          isRead: false
        })
      ]),
      1
    );

    act(() => {
      result.current.markAllAsRead();
    });

    expect(onLogsChange).toHaveBeenLastCalledWith(
      expect.arrayContaining([
        expect.objectContaining({
          message: 'Test message',
          isRead: true
        })
      ]),
      0
    );
  });

  it('handles logs without component attribute in clearByCategory', () => {
    const { result } = renderHook(() => useLogNotifications({}));

    act(() => {
      result.current.handleLogNotification(createLogNotification({ message: 'Error without component', attributes: {} }));
    });

    expect(result.current.logs).toHaveLength(1);

    act(() => {
      result.current.clearByCategory('WebSocket');
    });

    expect(result.current.logs).toHaveLength(1);
  });

  it('adds logs in reverse chronological order', () => {
    const { result } = renderHook(() => useLogNotifications({}));

    act(() => {
      result.current.handleLogNotification(createLogNotification({ message: 'First message' }));
      result.current.handleLogNotification(createLogNotification({ message: 'Second message' }));
    });

    expect(result.current.logs[0].message).toBe('Second message');
    expect(result.current.logs[1].message).toBe('First message');
  });

  it('generates unique IDs for each log entry', () => {
    const { result } = renderHook(() => useLogNotifications({}));

    act(() => {
      result.current.handleLogNotification(createLogNotification());
      result.current.handleLogNotification(createLogNotification());
    });

    expect(result.current.logs).toHaveLength(2);
    expect(result.current.logs[0].id).not.toBe(result.current.logs[1].id);
  });

  describe('device online notifications', () => {
    it('adds device online notification without alias', () => {
      const { result } = renderHook(() => useLogNotifications({}));
      const notification = createDeviceOnline();

      act(() => {
        result.current.handleDeviceOnlineNotification(notification);
      });

      expect(result.current.logs).toHaveLength(1);
      expect(result.current.logs[0].message).toBe('Device 192.168.1.100 0291:1 came online');
      expect(result.current.logs[0].level).toBe('INFO');
      expect(result.current.logs[0].attributes.alias).toBeUndefined();
      expect(result.current.logs[0].attributes.event).toBe('device_online');
    });

    it('adds device online notification with alias', () => {
      const mockResolveAlias = vi.fn(() => 'Living Room Light');
      const { result } = renderHook(() => useLogNotifications({ resolveAlias: mockResolveAlias }));
      const notification = createDeviceOnline();

      act(() => {
        result.current.handleDeviceOnlineNotification(notification);
      });

      expect(result.current.logs).toHaveLength(1);
      expect(result.current.logs[0].message).toBe('Device Living Room Light (192.168.1.100 0291:1) came online');
      expect(result.current.logs[0].attributes.alias).toBe('Living Room Light');
    });
  });

  describe('device offline notifications', () => {
    it('adds device offline notification without alias', () => {
      const { result } = renderHook(() => useLogNotifications({}));
      const notification = createDeviceOffline();

      act(() => {
        result.current.handleDeviceOfflineNotification(notification);
      });

      expect(result.current.logs).toHaveLength(1);
      expect(result.current.logs[0].message).toBe('Device 192.168.1.101 0130:1 went offline');
      expect(result.current.logs[0].level).toBe('WARN');
      expect(result.current.logs[0].attributes.alias).toBeUndefined();
      expect(result.current.logs[0].attributes.event).toBe('device_offline');
    });

    it('adds device offline notification with alias', () => {
      const mockResolveAlias = vi.fn(() => 'Bedroom AC');
      const { result } = renderHook(() => useLogNotifications({ resolveAlias: mockResolveAlias }));
      const notification = createDeviceOffline();

      act(() => {
        result.current.handleDeviceOfflineNotification(notification);
      });

      expect(result.current.logs).toHaveLength(1);
      expect(result.current.logs[0].message).toBe('Device Bedroom AC (192.168.1.101 0130:1) went offline');
      expect(result.current.logs[0].attributes.alias).toBe('Bedroom AC');
    });
  });

  describe('mixed notification types', () => {
    it('handles multiple notification types correctly', () => {
      const onLogsChange = vi.fn();
      const mockResolveAlias = vi.fn((ip: string, eoj: string) => {
        if (ip === '192.168.1.100' && eoj === '0291:1') {
          return 'Test Device';
        }
        return null;
      });

      const { result } = renderHook(() =>
        useLogNotifications({ resolveAlias: mockResolveAlias, onLogsChange })
      );

      act(() => {
        result.current.handleLogNotification(createLogNotification({ message: 'Test error' }));
        result.current.handleDeviceOnlineNotification(createDeviceOnline());
        result.current.handleDeviceOfflineNotification(createDeviceOffline());
      });

      expect(result.current.logs).toHaveLength(3);
      expect(result.current.logs[0].message).toBe('Device 192.168.1.101 0130:1 went offline');
      expect(result.current.logs[1].message).toBe('Device Test Device (192.168.1.100 0291:1) came online');
      expect(result.current.logs[2].message).toBe('Test error');
    });
  });
});
