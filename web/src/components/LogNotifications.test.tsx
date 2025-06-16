import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { LogNotifications } from './LogNotifications';
import type { LogNotification } from '../hooks/types';

describe('LogNotifications', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders nothing when no notifications', () => {
    const { container } = render(<LogNotifications />);
    expect(container.firstChild).toBeNull();
  });

  it('displays error notification', () => {
    const notification: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'Test error message',
        time: '2023-04-01T12:00:00Z',
        attributes: { device: '192.168.1.1' }
      }
    };

    render(<LogNotifications notification={notification} />);

    expect(screen.getByText('Test error message')).toBeInTheDocument();
    expect(screen.getByText('device: 192.168.1.1')).toBeInTheDocument();
  });

  it('displays warning notification', () => {
    const notification: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'WARN',
        message: 'Test warning message',
        time: '2023-04-01T12:00:00Z',
        attributes: {}
      }
    };

    render(<LogNotifications notification={notification} />);

    expect(screen.getByText('Test warning message')).toBeInTheDocument();
  });

  it('auto-hides warning notifications after delay', async () => {
    const notification: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'WARN',
        message: 'Auto-hide warning',
        time: '2023-04-01T12:00:00Z',
        attributes: {}
      }
    };

    const { rerender } = render(
      <LogNotifications notification={notification} autoHideDelay={1000} />
    );

    // Initially visible
    expect(screen.getByText('Auto-hide warning')).toBeInTheDocument();

    // Should still be in log history
    expect(screen.getAllByText('Auto-hide warning')).toHaveLength(2); // Toast + history

    // Advance time
    vi.advanceTimersByTime(1000);

    // Wait for state update
    await waitFor(() => {
      // Toast should be hidden, but still in history
      expect(screen.getAllByText('Auto-hide warning')).toHaveLength(1);
    });
  });

  it('does not auto-hide error notifications', async () => {
    const notification: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'Persistent error',
        time: '2023-04-01T12:00:00Z',
        attributes: {}
      }
    };

    render(<LogNotifications notification={notification} autoHideDelay={1000} />);

    expect(screen.getAllByText('Persistent error')).toHaveLength(2); // Toast + history

    vi.advanceTimersByTime(2000);

    // Should still show both toast and history
    expect(screen.getAllByText('Persistent error')).toHaveLength(2);
  });

  it('dismisses toast when X is clicked', () => {
    const notification: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'Dismissible error',
        time: '2023-04-01T12:00:00Z',
        attributes: {}
      }
    };

    render(<LogNotifications notification={notification} />);

    const dismissButtons = screen.getAllByLabelText('Dismiss');
    fireEvent.click(dismissButtons[0]); // Click the toast dismiss button

    // Toast should be gone but still in history
    expect(screen.getAllByText('Dismissible error')).toHaveLength(1);
  });

  it('clears all logs when Clear button is clicked', () => {
    const notification: LogNotification = {
      type: 'log_notification',
      payload: {
        level: 'ERROR',
        message: 'Clear test',
        time: '2023-04-01T12:00:00Z',
        attributes: {}
      }
    };

    const { container } = render(<LogNotifications notification={notification} />);

    fireEvent.click(screen.getByText('Clear'));

    expect(container.firstChild).toBeNull();
  });

  it('respects maxLogs limit', () => {
    const { rerender } = render(<LogNotifications maxLogs={2} />);

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
      rerender(<LogNotifications notification={notification} maxLogs={2} />);
    }

    // Should only show the last 2
    expect(screen.queryByText('Message 1')).not.toBeInTheDocument();
    expect(screen.getByText('Message 2')).toBeInTheDocument();
    expect(screen.getByText('Message 3')).toBeInTheDocument();
  });
});