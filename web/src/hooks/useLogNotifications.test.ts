import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { useLogNotifications } from './useLogNotifications';
import type { LogNotification } from './types';

describe('useLogNotifications', () => {
  it('starts with empty logs', () => {
    const { result } = renderHook(() => useLogNotifications({}));
    
    expect(result.current.logs).toEqual([]);
  });

  it('adds new log entry when notification is received', () => {
    const notification: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'Test error message',
        time: '2023-04-01T12:00:00Z',
        attributes: { device: '192.168.1.1' }
      }
    };

    const { result, rerender } = renderHook(
      ({ notification }) => useLogNotifications({ notification }),
      { initialProps: { notification: undefined as LogNotification | undefined } }
    );

    expect(result.current.logs).toEqual([]);

    rerender({ notification });

    expect(result.current.logs).toHaveLength(1);
    expect(result.current.logs[0].message).toBe('Test error message');
    expect(result.current.logs[0].level).toBe('ERROR');
    expect(result.current.logs[0].isRead).toBe(false);
  });

  it('maintains max log limit', () => {
    const { result, rerender } = renderHook(
      ({ notification }) => useLogNotifications({ notification, maxLogs: 3 }),
      { initialProps: { notification: undefined as LogNotification | undefined } }
    );

    // Add 5 notifications
    for (let i = 1; i <= 5; i++) {
      const notification: LogNotification = {
        type: 'log_notification',
        payload: {
          level: 'ERROR',
          message: `Message ${i}`,
          time: '2023-04-01T12:00:00Z',
          attributes: {}
        }
      };
      rerender({ notification });
    }

    // Should only keep the last 3
    expect(result.current.logs).toHaveLength(3);
    expect(result.current.logs[0].message).toBe('Message 5');
    expect(result.current.logs[1].message).toBe('Message 4');
    expect(result.current.logs[2].message).toBe('Message 3');
  });

  it('marks all logs as read', () => {
    const { result, rerender } = renderHook(
      ({ notification }) => useLogNotifications({ notification }),
      { initialProps: { notification: undefined as LogNotification | undefined } }
    );

    // Add two notifications
    const notification1: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'Message 1',
        time: '2023-04-01T12:00:00Z',
        attributes: {}
      }
    };

    const notification2: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'WARN',
        message: 'Message 2',
        time: '2023-04-01T12:01:00Z',
        attributes: {}
      }
    };

    rerender({ notification: notification1 });
    rerender({ notification: notification2 });

    expect(result.current.logs).toHaveLength(2);
    expect(result.current.logs.every(log => !log.isRead)).toBe(true);

    act(() => {
      result.current.markAllAsRead();
    });

    expect(result.current.logs.every(log => log.isRead)).toBe(true);
  });

  it('clears all logs', () => {
    const { result, rerender } = renderHook(
      ({ notification }) => useLogNotifications({ notification }),
      { initialProps: { notification: undefined as LogNotification | undefined } }
    );

    const notification: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'Test message',
        time: '2023-04-01T12:00:00Z',
        attributes: {}
      }
    };

    rerender({ notification });
    expect(result.current.logs).toHaveLength(1);

    act(() => {
      result.current.clearAllLogs();
    });

    expect(result.current.logs).toHaveLength(0);
  });

  it('clears logs by category', () => {
    const { result, rerender } = renderHook(
      ({ notification }) => useLogNotifications({ notification }),
      { initialProps: { notification: undefined as LogNotification | undefined } }
    );

    // Add WebSocket error
    const wsNotification: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'WebSocket error',
        time: '2023-04-01T12:00:00Z',
        attributes: { component: 'WebSocket' }
      }
    };
    rerender({ notification: wsNotification });

    // Add another error
    const otherNotification: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'Other error',
        time: '2023-04-01T12:01:00Z',
        attributes: { component: 'Other' }
      }
    };
    rerender({ notification: otherNotification });

    expect(result.current.logs).toHaveLength(2);

    act(() => {
      result.current.clearByCategory('WebSocket');
    });

    expect(result.current.logs).toHaveLength(1);
    expect(result.current.logs[0].message).toBe('Other error');
  });

  it('calls onLogsChange callback when logs change', () => {
    const onLogsChange = vi.fn();
    const { result, rerender } = renderHook(
      ({ notification }) => useLogNotifications({ notification, onLogsChange }),
      { initialProps: { notification: undefined as LogNotification | undefined } }
    );

    // Initial call with empty logs
    expect(onLogsChange).toHaveBeenCalledWith([], 0);

    const notification: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'Test message',
        time: '2023-04-01T12:00:00Z',
        attributes: {}
      }
    };

    rerender({ notification });

    // Should be called with the new log and unread count
    expect(onLogsChange).toHaveBeenCalledWith(
      expect.arrayContaining([
        expect.objectContaining({
          message: 'Test message',
          isRead: false
        })
      ]),
      1
    );

    // Mark as read
    act(() => {
      result.current.markAllAsRead();
    });

    // Should be called with updated read status
    expect(onLogsChange).toHaveBeenCalledWith(
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
    const { result, rerender } = renderHook(
      ({ notification }) => useLogNotifications({ notification }),
      { initialProps: { notification: undefined as LogNotification | undefined } }
    );

    // Add log without component attribute
    const notification: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'Error without component',
        time: '2023-04-01T12:00:00Z',
        attributes: {}
      }
    };
    rerender({ notification });

    expect(result.current.logs).toHaveLength(1);

    act(() => {
      result.current.clearByCategory('WebSocket');
    });

    // Should still have the log since it doesn't match the category
    expect(result.current.logs).toHaveLength(1);
  });

  it('adds logs in reverse chronological order', () => {
    const { result, rerender } = renderHook(
      ({ notification }) => useLogNotifications({ notification }),
      { initialProps: { notification: undefined as LogNotification | undefined } }
    );

    const notification1: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'First message',
        time: '2023-04-01T12:00:00Z',
        attributes: {}
      }
    };

    const notification2: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'Second message',
        time: '2023-04-01T12:01:00Z',
        attributes: {}
      }
    };

    rerender({ notification: notification1 });
    rerender({ notification: notification2 });

    // Newer logs should appear first
    expect(result.current.logs[0].message).toBe('Second message');
    expect(result.current.logs[1].message).toBe('First message');
  });

  it('generates unique IDs for each log entry', () => {
    const { result, rerender } = renderHook(
      ({ notification }) => useLogNotifications({ notification }),
      { initialProps: { notification: undefined as LogNotification | undefined } }
    );

    const notification1: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'Test message',
        time: '2023-04-01T12:00:00Z',
        attributes: {}
      }
    };

    const notification2: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'Test message',
        time: '2023-04-01T12:00:00Z',
        attributes: {}
      }
    };

    // Add same content but different object references
    rerender({ notification: notification1 });
    rerender({ notification: notification2 });

    expect(result.current.logs).toHaveLength(2);
    expect(result.current.logs[0].id).not.toBe(result.current.logs[1].id);
  });
});