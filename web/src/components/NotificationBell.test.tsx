import { fireEvent, render, screen, waitFor, act } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import type { LogEntry } from '../hooks/useLogNotifications';
import { NotificationBell, type NotificationBellProps, formatLogTime } from './NotificationBell';

describe('NotificationBell', () => {
  const mockLogs: LogEntry[] = [
    {
      id: '1',
      level: 'ERROR',
      message: 'Test error message',
      time: '2023-04-01T12:00:00Z',
      attributes: { device: '192.168.1.1' },
      isRead: false
    },
    {
      id: '2',
      level: 'WARN',
      message: 'Test warning message',
      time: '2023-04-01T12:01:00Z',
      attributes: {},
      isRead: true
    }
  ];

  const defaultProps: NotificationBellProps = {
    logs: mockLogs,
    unreadCount: 1,
    onMarkAllAsRead: vi.fn(),
    onClearAll: vi.fn(),
    connectedAt: new Date('2023-04-01T12:00:00Z'),
    serverStartupTime: new Date('2023-04-01T11:00:00Z')
  };

  it('renders bell icon with unread count badge', () => {
    render(<NotificationBell {...defaultProps} />);
    
    expect(screen.getByRole('button')).toBeInTheDocument();
    expect(screen.getByText('1')).toBeInTheDocument();
  });

  it('shows 99+ when unread count exceeds 99', () => {
    render(<NotificationBell {...defaultProps} unreadCount={150} />);
    
    expect(screen.getByText('99+')).toBeInTheDocument();
  });

  it('does not show badge when unread count is 0', () => {
    render(<NotificationBell {...defaultProps} unreadCount={0} />);
    
    expect(screen.queryByText('0')).not.toBeInTheDocument();
  });

  it('opens dropdown when bell is clicked', () => {
    render(<NotificationBell {...defaultProps} />);
    
    fireEvent.click(screen.getByRole('button'));
    
    expect(screen.getByText('Server Logs')).toBeInTheDocument();
    expect(screen.getByText('Test error message')).toBeInTheDocument();
    expect(screen.getByText('Test warning message')).toBeInTheDocument();
  });

  it('calls onMarkAllAsRead when dropdown is opened with unread logs', () => {
    const onMarkAllAsRead = vi.fn();
    render(<NotificationBell {...defaultProps} onMarkAllAsRead={onMarkAllAsRead} />);
    
    fireEvent.click(screen.getByRole('button'));
    
    expect(onMarkAllAsRead).toHaveBeenCalledTimes(1);
  });

  it('does not call onMarkAllAsRead when no unread logs', () => {
    const onMarkAllAsRead = vi.fn();
    render(<NotificationBell {...defaultProps} unreadCount={0} onMarkAllAsRead={onMarkAllAsRead} />);
    
    fireEvent.click(screen.getByRole('button'));
    
    expect(onMarkAllAsRead).not.toHaveBeenCalled();
  });

  it('calls onClearAll when Clear All button is clicked', () => {
    const onClearAll = vi.fn();
    render(<NotificationBell {...defaultProps} onClearAll={onClearAll} />);
    
    fireEvent.click(screen.getByRole('button'));
    fireEvent.click(screen.getByText('Clear All'));
    
    expect(onClearAll).toHaveBeenCalledTimes(1);
  });

  it('closes dropdown when X button is clicked', async () => {
    render(<NotificationBell {...defaultProps} />);
    
    fireEvent.click(screen.getByRole('button'));
    expect(screen.getByText('Server Logs')).toBeInTheDocument();
    
    fireEvent.click(screen.getAllByRole('button').find(btn => btn.querySelector('svg'))!);
    
    await waitFor(() => {
      expect(screen.queryByText('Server Logs')).not.toBeInTheDocument();
    });
  });

  it('shows empty state when no logs', () => {
    render(<NotificationBell {...defaultProps} logs={[]} />);
    
    fireEvent.click(screen.getByRole('button'));
    
    expect(screen.getByText('No logs yet')).toBeInTheDocument();
  });

  it('displays log attributes correctly', () => {
    render(<NotificationBell {...defaultProps} />);
    
    fireEvent.click(screen.getByRole('button'));
    
    expect(screen.getByText('device:')).toBeInTheDocument();
    expect(screen.getByText('192.168.1.1')).toBeInTheDocument();
  });

  it('shows correct total count in footer', () => {
    render(<NotificationBell {...defaultProps} />);
    
    fireEvent.click(screen.getByRole('button'));
    
    expect(screen.getByText('2 logs total')).toBeInTheDocument();
  });

  // Discover Devices Tests
  it('shows discover button when onDiscoverDevices is provided', () => {
    const onDiscoverDevices = vi.fn();
    render(<NotificationBell {...defaultProps} onDiscoverDevices={onDiscoverDevices} />);
    
    fireEvent.click(screen.getByRole('button'));
    
    expect(screen.getByText('Discover')).toBeInTheDocument();
  });

  it('does not show discover button when onDiscoverDevices is not provided', () => {
    render(<NotificationBell {...defaultProps} />);
    
    fireEvent.click(screen.getByRole('button'));
    
    expect(screen.queryByText('Discover')).not.toBeInTheDocument();
  });

  it('calls onDiscoverDevices when discover button is clicked', async () => {
    const onDiscoverDevices = vi.fn().mockResolvedValue({});
    render(<NotificationBell {...defaultProps} onDiscoverDevices={onDiscoverDevices} />);
    
    fireEvent.click(screen.getByRole('button'));
    
    await act(async () => {
      fireEvent.click(screen.getByText('Discover'));
    });
    
    expect(onDiscoverDevices).toHaveBeenCalledTimes(1);
  });

  it('shows loading state during discover operation', async () => {
    let resolveDiscover: () => void;
    const discoverPromise = new Promise<void>((resolve) => {
      resolveDiscover = resolve;
    });
    const onDiscoverDevices = vi.fn().mockReturnValue(discoverPromise);
    
    render(<NotificationBell {...defaultProps} onDiscoverDevices={onDiscoverDevices} />);
    
    fireEvent.click(screen.getByRole('button'));
    fireEvent.click(screen.getByText('Discover'));
    
    expect(screen.getByText('Searching...')).toBeInTheDocument();
    
    // Resolve the promise
    resolveDiscover!();
    await waitFor(() => {
      expect(screen.getByText('Discover')).toBeInTheDocument();
    });
  });

  it('disables discover button during operation', async () => {
    let resolveDiscover: () => void;
    const discoverPromise = new Promise<void>((resolve) => {
      resolveDiscover = resolve;
    });
    const onDiscoverDevices = vi.fn().mockReturnValue(discoverPromise);
    
    render(<NotificationBell {...defaultProps} onDiscoverDevices={onDiscoverDevices} />);
    
    fireEvent.click(screen.getByRole('button'));
    const discoverButton = screen.getByText('Discover');
    fireEvent.click(discoverButton);
    
    const searchingButton = screen.getByText('Searching...');
    expect(searchingButton).toBeDisabled();
    
    // Resolve the promise
    resolveDiscover!();
    await waitFor(() => {
      expect(screen.getByText('Discover')).not.toBeDisabled();
    });
  });

  it('handles discover error gracefully', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const onDiscoverDevices = vi.fn().mockRejectedValue(new Error('Network error'));
    
    render(<NotificationBell {...defaultProps} onDiscoverDevices={onDiscoverDevices} />);
    
    fireEvent.click(screen.getByRole('button'));
    fireEvent.click(screen.getByText('Discover'));
    
    await waitFor(() => {
      expect(screen.getByText('Discover')).toBeInTheDocument();
    });
    
    expect(consoleSpy).toHaveBeenCalledWith('Discover devices failed:', expect.any(Error));
    consoleSpy.mockRestore();
  });

  describe('timestamp display', () => {
    it('displays server startup time when provided', () => {
      render(<NotificationBell {...defaultProps} />);
      
      fireEvent.click(screen.getByRole('button'));
      
      expect(screen.getByTestId('server-startup-time')).toBeInTheDocument();
      expect(screen.getByTestId('server-startup-time')).toHaveTextContent('Server started:');
    });

    it('hides server startup time when null', () => {
      render(<NotificationBell {...defaultProps} serverStartupTime={null} />);
      
      fireEvent.click(screen.getByRole('button'));
      
      expect(screen.queryByTestId('server-startup-time')).not.toBeInTheDocument();
    });

    it('displays Web UI build date from environment', () => {
      // Mock the BUILD_DATE environment variable
      const originalBuildDate = import.meta.env.BUILD_DATE;
      import.meta.env.BUILD_DATE = '2023-04-01T10:00:00.000Z';
      
      render(<NotificationBell {...defaultProps} />);
      
      fireEvent.click(screen.getByRole('button'));
      
      expect(screen.getByTestId('build-time')).toBeInTheDocument();
      expect(screen.getByTestId('build-time')).toHaveTextContent('Web UI built:');
      
      // Restore original value
      import.meta.env.BUILD_DATE = originalBuildDate;
    });

    it('displays connection time when provided', () => {
      render(<NotificationBell {...defaultProps} />);
      
      fireEvent.click(screen.getByRole('button'));
      
      expect(screen.getByTestId('connection-time')).toBeInTheDocument();
      expect(screen.getByTestId('connection-time')).toHaveTextContent('Connected at:');
    });

    it('hides connection time when null', () => {
      render(<NotificationBell {...defaultProps} connectedAt={null} />);
      
      fireEvent.click(screen.getByRole('button'));
      
      expect(screen.queryByTestId('connection-time')).not.toBeInTheDocument();
    });

    it('displays all timestamps in correct order when all provided', () => {
      render(<NotificationBell {...defaultProps} />);

      fireEvent.click(screen.getByRole('button'));

      // Check that all three timestamps are present
      expect(screen.getByTestId('server-startup-time')).toBeInTheDocument();
      expect(screen.getByTestId('build-time')).toBeInTheDocument();
      expect(screen.getByTestId('connection-time')).toBeInTheDocument();
    });
  });

});

describe('formatLogTime', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    // Set current time to today at 14:30:00 (local time)
    const now = new Date();
    now.setHours(14, 30, 0, 0);
    vi.setSystemTime(now);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns time with seconds for today logs', () => {
    // Today at 10:25:30 (local time)
    const todayLog = new Date();
    todayLog.setHours(10, 25, 30, 0);
    const result = formatLogTime(todayLog);
    expect(result).toBe('10:25:30');
  });

  it('returns padded time for midnight logs', () => {
    // Today at 00:05:03 (local time)
    const midnightLog = new Date();
    midnightLog.setHours(0, 5, 3, 0);
    const result = formatLogTime(midnightLog);
    expect(result).toBe('00:05:03');
  });

  it('returns date and time for yesterday logs', () => {
    // Yesterday at 10:25 (local time)
    const yesterdayLog = new Date();
    yesterdayLog.setDate(yesterdayLog.getDate() - 1);
    yesterdayLog.setHours(10, 25, 0, 0);
    const result = formatLogTime(yesterdayLog);
    const expectedMonth = yesterdayLog.getMonth() + 1;
    const expectedDay = yesterdayLog.getDate();
    expect(result).toBe(`${expectedMonth}/${expectedDay} 10:25`);
  });

  it('returns date and time for older logs', () => {
    // A week ago at 08:05 (local time)
    const oldLog = new Date();
    oldLog.setDate(oldLog.getDate() - 7);
    oldLog.setHours(8, 5, 0, 0);
    const result = formatLogTime(oldLog);
    const expectedMonth = oldLog.getMonth() + 1;
    const expectedDay = oldLog.getDate();
    expect(result).toBe(`${expectedMonth}/${expectedDay} 08:05`);
  });

  it('pads hours, minutes and seconds with zero when needed', () => {
    // Today at 09:05:03 (local time)
    const todayLog = new Date();
    todayLog.setHours(9, 5, 3, 0);
    const result = formatLogTime(todayLog);
    expect(result).toBe('09:05:03');
  });

  it('returns date with year for logs from previous year', () => {
    // Last year at 10:25 (local time)
    const lastYearLog = new Date();
    lastYearLog.setFullYear(lastYearLog.getFullYear() - 1);
    lastYearLog.setHours(10, 25, 0, 0);
    const result = formatLogTime(lastYearLog);
    const expectedYear = lastYearLog.getFullYear();
    const expectedMonth = lastYearLog.getMonth() + 1;
    const expectedDay = lastYearLog.getDate();
    expect(result).toBe(`${expectedYear}/${expectedMonth}/${expectedDay} 10:25`);
  });
});