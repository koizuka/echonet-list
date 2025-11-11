import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { GroupBulkControl } from '@/components/GroupBulkControl';
import type { Device } from '@/hooks/types';

describe('GroupBulkControl', () => {
  const mockDevices: Device[] = [
    {
      ip: '192.168.1.100',
      eoj: '0130:1',
      name: 'Air Conditioner',
      id: undefined,
      lastSeen: '2025-01-01T00:00:00Z',
      properties: {
        '80': { string: 'on', number: 0x30 }, // Operation Status
        '9E': { EDT: 'AYA=' } // Set Property Map: [0x01, 0x80] = 1 property (0x80)
      }
    },
    {
      ip: '192.168.1.101',
      eoj: '0291:1',
      name: 'Lighting',
      id: undefined,
      lastSeen: '2025-01-01T00:00:00Z',
      properties: {
        '80': { string: 'off', number: 0x31 },
        '9E': { EDT: 'AYA=' } // Set Property Map: [0x01, 0x80] = 1 property (0x80)
      }
    },
    {
      ip: '192.168.1.102',
      eoj: '0288:1',
      name: 'Sensor',
      id: undefined,
      lastSeen: '2025-01-01T00:00:00Z',
      properties: {
        '80': { string: 'on', number: 0x30 },
        '9F': { EDT: 'AYA=' } // Get Property Map only (not settable)
      }
    }
  ];

  it('renders power control buttons', () => {
    const mockOnPropertyChange = vi.fn();
    render(
      <GroupBulkControl
        devices={mockDevices}
        onPropertyChange={mockOnPropertyChange}
      />
    );

    expect(screen.getByText('All ON')).toBeInTheDocument();
    expect(screen.getByText('All OFF')).toBeInTheDocument();
  });

  it('disables buttons when no controllable devices', () => {
    const nonControllableDevices: Device[] = [
      {
        ip: '192.168.1.102',
        eoj: '0288:1',
        name: 'Sensor',
        id: undefined,
        lastSeen: '2025-01-01T00:00:00Z',
        properties: {
          '80': { string: 'on', number: 0x30 },
          '9F': { EDT: 'AYA=' } // Get only (no Set Property Map 0x9E)
        }
      }
    ];
    const mockOnPropertyChange = vi.fn();
    render(
      <GroupBulkControl
        devices={nonControllableDevices}
        onPropertyChange={mockOnPropertyChange}
      />
    );

    const onButton = screen.getByText('All ON');
    const offButton = screen.getByText('All OFF');

    expect(onButton).toBeDisabled();
    expect(offButton).toBeDisabled();
  });

  it('calls onPropertyChange for all controllable devices on "All ON" click', async () => {
    const mockOnPropertyChange = vi.fn().mockResolvedValue(undefined);
    render(
      <GroupBulkControl
        devices={mockDevices}
        onPropertyChange={mockOnPropertyChange}
      />
    );

    const onButton = screen.getByText('All ON');
    fireEvent.click(onButton);

    await waitFor(() => {
      expect(mockOnPropertyChange).toHaveBeenCalledTimes(2); // Only controllable devices
    });

    // Check that only devices with settable 0x80 are controlled
    expect(mockOnPropertyChange).toHaveBeenCalledWith(
      '192.168.1.100 0130:1',
      '80',
      { string: 'on' }
    );
    expect(mockOnPropertyChange).toHaveBeenCalledWith(
      '192.168.1.101 0291:1',
      '80',
      { string: 'on' }
    );
    // Non-settable device should not be called
    expect(mockOnPropertyChange).not.toHaveBeenCalledWith(
      '192.168.1.102 0288:1',
      expect.anything(),
      expect.anything()
    );
  });

  it('calls onPropertyChange for all controllable devices on "All OFF" click', async () => {
    const mockOnPropertyChange = vi.fn().mockResolvedValue(undefined);
    render(
      <GroupBulkControl
        devices={mockDevices}
        onPropertyChange={mockOnPropertyChange}
      />
    );

    const offButton = screen.getByText('All OFF');
    fireEvent.click(offButton);

    await waitFor(() => {
      expect(mockOnPropertyChange).toHaveBeenCalledTimes(2);
    });

    expect(mockOnPropertyChange).toHaveBeenCalledWith(
      '192.168.1.100 0130:1',
      '80',
      { string: 'off' }
    );
    expect(mockOnPropertyChange).toHaveBeenCalledWith(
      '192.168.1.101 0291:1',
      '80',
      { string: 'off' }
    );
  });

  it('continues operation even if some devices fail', async () => {
    const mockOnPropertyChange = vi.fn()
      .mockResolvedValueOnce(undefined) // First device succeeds
      .mockRejectedValueOnce(new Error('Network error')); // Second device fails

    render(
      <GroupBulkControl
        devices={mockDevices}
        onPropertyChange={mockOnPropertyChange}
      />
    );

    const onButton = screen.getByText('All ON');
    fireEvent.click(onButton);

    await waitFor(() => {
      expect(mockOnPropertyChange).toHaveBeenCalledTimes(2);
    });

    // Should have attempted both devices despite error
    expect(mockOnPropertyChange).toHaveBeenCalledTimes(2);
  });

  it('shows loading state during bulk operation', async () => {
    const mockOnPropertyChange = vi.fn((): Promise<void> =>
      new Promise(resolve => setTimeout(resolve, 100))
    );

    render(
      <GroupBulkControl
        devices={mockDevices}
        onPropertyChange={mockOnPropertyChange}
      />
    );

    const onButton = screen.getByText('All ON');
    fireEvent.click(onButton);

    // During operation, buttons should be disabled
    expect(onButton).toBeDisabled();
    expect(screen.getByText('All OFF')).toBeDisabled();

    await waitFor(() => {
      expect(onButton).not.toBeDisabled();
    }, { timeout: 200 });
  });

  it('filters devices correctly based on Set Property Map', async () => {
    const devicesWithMixedSettability: Device[] = [
      {
        ip: '192.168.1.100',
        eoj: '0130:1',
        name: 'Air Conditioner',
        id: undefined,
        lastSeen: '2025-01-01T00:00:00Z',
        properties: {
          '80': { string: 'on' },
          '9E': { EDT: 'AYA=' } // Set Property Map: [0x01, 0x80] = has 0x80
        }
      },
      {
        ip: '192.168.1.101',
        eoj: '0291:1',
        name: 'Lighting',
        id: undefined,
        lastSeen: '2025-01-01T00:00:00Z',
        properties: {
          '80': { string: 'off' },
          '9E': { EDT: 'AYEE' } // Set Property Map: [0x01, 0x81] = has 0x81 but not 0x80
        }
      },
    ];

    const mockOnPropertyChange = vi.fn().mockResolvedValue(undefined);
    render(
      <GroupBulkControl
        devices={devicesWithMixedSettability}
        onPropertyChange={mockOnPropertyChange}
      />
    );

    const onButton = screen.getByText('All ON');
    fireEvent.click(onButton);

    // Only the first device should be controlled
    await waitFor(() => {
      expect(mockOnPropertyChange).toHaveBeenCalledTimes(1);
      expect(mockOnPropertyChange).toHaveBeenCalledWith(
        '192.168.1.100 0130:1',
        '80',
        { string: 'on' }
      );
    });
  });

  it('does not add notification when all operations succeed', async () => {
    const mockOnPropertyChange = vi.fn().mockResolvedValue(undefined);
    const mockAddLogEntry = vi.fn();

    render(
      <GroupBulkControl
        devices={mockDevices}
        onPropertyChange={mockOnPropertyChange}
        addLogEntry={mockAddLogEntry}
      />
    );

    const onButton = screen.getByText('All ON');
    fireEvent.click(onButton);

    await waitFor(() => {
      expect(mockOnPropertyChange).toHaveBeenCalledTimes(2);
    });

    // No notification should be added when all succeed
    expect(mockAddLogEntry).not.toHaveBeenCalled();
  });

  it('adds ERROR notification when all operations fail', async () => {
    const mockOnPropertyChange = vi.fn().mockRejectedValue(new Error('Network error'));
    const mockAddLogEntry = vi.fn();

    render(
      <GroupBulkControl
        devices={mockDevices}
        onPropertyChange={mockOnPropertyChange}
        addLogEntry={mockAddLogEntry}
      />
    );

    const offButton = screen.getByText('All OFF');
    fireEvent.click(offButton);

    await waitFor(() => {
      expect(mockAddLogEntry).toHaveBeenCalledTimes(1);
    });

    const logEntry = mockAddLogEntry.mock.calls[0][0];
    expect(logEntry.level).toBe('ERROR');
    expect(logEntry.message).toBe('Failed to turn OFF 2 devices');
    expect(logEntry.attributes.successCount).toBe(0);
    expect(logEntry.attributes.failureCount).toBe(2);
  });

  it('adds WARN notification when some operations fail', async () => {
    const mockOnPropertyChange = vi.fn()
      .mockResolvedValueOnce(undefined) // First device succeeds
      .mockRejectedValueOnce(new Error('Network error')); // Second device fails
    const mockAddLogEntry = vi.fn();

    render(
      <GroupBulkControl
        devices={mockDevices}
        onPropertyChange={mockOnPropertyChange}
        addLogEntry={mockAddLogEntry}
      />
    );

    const onButton = screen.getByText('All ON');
    fireEvent.click(onButton);

    await waitFor(() => {
      expect(mockAddLogEntry).toHaveBeenCalledTimes(1);
    });

    const logEntry = mockAddLogEntry.mock.calls[0][0];
    expect(logEntry.level).toBe('WARN');
    expect(logEntry.message).toBe('Turned ON 1/2 devices (1 failed)');
    expect(logEntry.attributes.successCount).toBe(1);
    expect(logEntry.attributes.failureCount).toBe(1);
  });

  it('does not add notification when addLogEntry is not provided', async () => {
    const mockOnPropertyChange = vi.fn().mockResolvedValue(undefined);

    render(
      <GroupBulkControl
        devices={mockDevices}
        onPropertyChange={mockOnPropertyChange}
      />
    );

    const onButton = screen.getByText('All ON');
    fireEvent.click(onButton);

    await waitFor(() => {
      expect(mockOnPropertyChange).toHaveBeenCalledTimes(2);
    });

    // No errors should occur even without addLogEntry
  });
});
