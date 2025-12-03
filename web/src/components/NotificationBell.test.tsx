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
    // Set current time to 2023-12-15 14:30:00 JST
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2023-12-15T05:30:00Z')); // UTC = JST - 9
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns time with seconds for today logs', () => {
    // Today at 10:25:30 JST (01:25:30 UTC)
    const todayLog = new Date('2023-12-15T01:25:30Z');
    const result = formatLogTime(todayLog);
    expect(result).toBe('10:25:30');
  });

  it('returns date and time for yesterday logs', () => {
    // Yesterday at 10:25 JST
    const yesterdayLog = new Date('2023-12-14T01:25:00Z');
    const result = formatLogTime(yesterdayLog);
    expect(result).toBe('12/14 10:25');
  });

  it('returns date and time for older logs', () => {
    // A week ago at 08:05 JST
    const oldLog = new Date('2023-12-08T23:05:00Z'); // Dec 9 08:05 JST
    const result = formatLogTime(oldLog);
    expect(result).toBe('12/9 8:05');
  });

  it('pads minutes and seconds with zero when needed', () => {
    // Today at 09:05:03 JST (00:05:03 UTC)
    const todayLog = new Date('2023-12-15T00:05:03Z');
    const result = formatLogTime(todayLog);
    expect(result).toBe('9:05:03');
  });
});