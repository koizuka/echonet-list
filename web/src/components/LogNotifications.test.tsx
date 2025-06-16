import { renderHook, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { LogNotifications } from './LogNotifications';
import type { LogNotification } from '../hooks/types';

describe('LogNotifications', () => {
  it('starts with empty logs', () => {
    const { result } = renderHook(() => LogNotifications({}));
    
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
      ({ notification }) => LogNotifications({ notification }),
      { initialProps: { notification: undefined } }
    );

    expect(result.current.logs).toEqual([]);

    rerender({ notification });

    expect(result.current.logs).toHaveLength(1);
    expect(result.current.logs[0].message).toBe('Test error message');
    expect(result.current.logs[0].level).toBe('ERROR');
    expect(result.current.logs[0].isRead).toBe(false);
  });

  it('marks individual log as read', () => {
    const notification: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'Test message',
        time: '2023-04-01T12:00:00Z',
        attributes: {}
      }
    };

    const { result, rerender } = renderHook(
      ({ notification }) => LogNotifications({ notification }),
      { initialProps: { notification: undefined } }
    );

    rerender({ notification });
    
    const logId = result.current.logs[0].id;
    
    act(() => {
      result.current.markAsRead(logId);
    });

    expect(result.current.logs[0].isRead).toBe(true);
  });

  it('marks all logs as read', () => {
    const { result, rerender } = renderHook(
      ({ notification }) => LogNotifications({ notification }),
      { initialProps: { notification: undefined } }
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
    const notification: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'Test message',
        time: '2023-04-01T12:00:00Z',
        attributes: {}
      }
    };

    const { result, rerender } = renderHook(
      ({ notification }) => LogNotifications({ notification }),
      { initialProps: { notification: undefined } }
    );

    rerender({ notification });
    
    expect(result.current.logs).toHaveLength(1);

    act(() => {
      result.current.clearAllLogs();
    });

    expect(result.current.logs).toEqual([]);
  });

  it('respects maxLogs limit', () => {
    const { result, rerender } = renderHook(
      ({ notification }) => LogNotifications({ notification, maxLogs: 2 }),
      { initialProps: { notification: undefined } }
    );

    // Add 3 notifications
    for (let i = 1; i <= 3; i++) {
      const notification: LogNotification = {
        type: 'log_notification',
        payload: {
          level: 'ERROR',
          message: `Message ${i}`,
          time: `2023-04-01T12:0${i}:00Z`,
          attributes: {}
        }
      };
      rerender({ notification });
    }

    // Should only keep the last 2
    expect(result.current.logs).toHaveLength(2);
    expect(result.current.logs[0].message).toBe('Message 3'); // Most recent first
    expect(result.current.logs[1].message).toBe('Message 2');
  });

  it('calls onLogsChange when logs change', () => {
    const onLogsChange = vi.fn();
    const notification: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'Test message',
        time: '2023-04-01T12:00:00Z',
        attributes: {}
      }
    };

    const { rerender } = renderHook(
      ({ notification }) => LogNotifications({ notification, onLogsChange }),
      { initialProps: { notification: undefined } }
    );

    expect(onLogsChange).toHaveBeenCalledWith([], 0);

    rerender({ notification });

    expect(onLogsChange).toHaveBeenCalledWith(
      expect.arrayContaining([
        expect.objectContaining({
          message: 'Test message',
          isRead: false
        })
      ]),
      1
    );
  });
});