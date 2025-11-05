import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DeviceHistoryDialog } from './DeviceHistoryDialog';
import type { Device, PropertyDescriptionData, DeviceHistoryEntry } from '@/hooks/types';
import type { WebSocketConnection } from '@/hooks/useWebSocketConnection';

// Mock the useDeviceHistory hook
vi.mock('@/hooks/useDeviceHistory', () => ({
  useDeviceHistory: vi.fn(),
}));

import { useDeviceHistory } from '@/hooks/useDeviceHistory';

describe('DeviceHistoryDialog', () => {
  let mockDevice: Device;
  let mockConnection: WebSocketConnection;
  let mockPropertyDescriptions: Record<string, PropertyDescriptionData>;

  beforeEach(() => {
    mockDevice = {
      ip: '192.168.1.10',
      eoj: '0130:1',
      name: 'HomeAirConditioner',
      id: '013001:00000B:ABCDEF0123456789ABCDEF012345',
      properties: {
        '80': { string: 'on', EDT: 'MzA=' },
        B3: { number: 25, EDT: 'MjU=' },
      },
      lastSeen: '2024-05-01T12:00:00Z',
    };

    mockConnection = {
      connectionState: 'connected',
      sendMessage: vi.fn(),
      connect: vi.fn(),
      disconnect: vi.fn(),
      connectedAt: new Date(),
      checkConnection: vi.fn().mockResolvedValue(true),
    };

    mockPropertyDescriptions = {
      '0130': {
        classCode: '0130',
        properties: {
          '80': {
            description: 'Operation Status',
            aliases: { on: 'MzA=', off: 'MzE=' },
          },
          B3: {
            description: 'Temperature Setting',
            numberDesc: {
              min: 0,
              max: 50,
              offset: 0,
              unit: 'C',
              edtLen: 1,
            },
          },
        },
      },
    };

    vi.mocked(useDeviceHistory).mockReturnValue({
      entries: [],
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });
  });

  it('should not render when isOpen is false', () => {
    render(
      <DeviceHistoryDialog
        device={mockDevice}
        connection={mockConnection}
        isOpen={false}
        onOpenChange={vi.fn()}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
        isConnected={true}
      />
    );

    expect(screen.queryByText(/Device History/i)).not.toBeInTheDocument();
  });

  it('should render loading state', () => {
    vi.mocked(useDeviceHistory).mockReturnValue({
      entries: [],
      isLoading: true,
      error: null,
      refetch: vi.fn(),
    });

    render(
      <DeviceHistoryDialog
        device={mockDevice}
        connection={mockConnection}
        isOpen={true}
        onOpenChange={vi.fn()}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
        isConnected={true}
      />
    );

    expect(screen.getByText(/Loading/i)).toBeInTheDocument();
  });

  it('should render error state', () => {
    const mockError = new Error('Failed to fetch history');
    vi.mocked(useDeviceHistory).mockReturnValue({
      entries: [],
      isLoading: false,
      error: mockError,
      refetch: vi.fn(),
    });

    render(
      <DeviceHistoryDialog
        device={mockDevice}
        connection={mockConnection}
        isOpen={true}
        onOpenChange={vi.fn()}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
        isConnected={true}
      />
    );

    expect(screen.getByText(/Failed to fetch history/i)).toBeInTheDocument();
  });

  it('should render empty state', () => {
    vi.mocked(useDeviceHistory).mockReturnValue({
      entries: [],
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });

    render(
      <DeviceHistoryDialog
        device={mockDevice}
        connection={mockConnection}
        isOpen={true}
        onOpenChange={vi.fn()}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
        isConnected={true}
      />
    );

    expect(screen.getByText(/No history available/i)).toBeInTheDocument();
  });

  it('should render history entries', () => {
    const mockEntries: DeviceHistoryEntry[] = [
      {
        timestamp: '2024-05-01T12:34:56.789Z',
        epc: '80',
        value: { string: 'on', EDT: 'MzA=' },
        origin: 'set',
        settable: true,
      },
      {
        timestamp: '2024-05-01T12:35:10.123Z',
        epc: 'B3',
        value: { number: 25, EDT: 'MjU=' },
        origin: 'notification',
        settable: true,
      },
    ];

    vi.mocked(useDeviceHistory).mockReturnValue({
      entries: mockEntries,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });

    render(
      <DeviceHistoryDialog
        device={mockDevice}
        connection={mockConnection}
        isOpen={true}
        onOpenChange={vi.fn()}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
        isConnected={true}
      />
    );

    expect(screen.getByText(/Operation Status/i)).toBeInTheDocument();
    expect(screen.getByText(/Temperature Setting/i)).toBeInTheDocument();
    // Verify both entries are displayed
    const container = screen.getByRole('alertdialog');
    expect(container.textContent).toContain('on');
    expect(container.textContent).toContain('25');
  });

  it('should toggle settableOnly filter', async () => {
    const user = userEvent.setup();
    const mockRefetch = vi.fn();

    vi.mocked(useDeviceHistory).mockReturnValue({
      entries: [],
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });

    render(
      <DeviceHistoryDialog
        device={mockDevice}
        connection={mockConnection}
        isOpen={true}
        onOpenChange={vi.fn()}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
        isConnected={true}
      />
    );

    const toggle = screen.getByRole('switch');
    expect(toggle).toBeInTheDocument();
    expect(toggle).toHaveAttribute('aria-checked', 'false');

    await user.click(toggle);

    // After clicking, the toggle should be checked
    await waitFor(() => {
      expect(toggle).toHaveAttribute('aria-checked', 'true');
    });

    // The component should call useDeviceHistory with settableOnly=true
    // (This will trigger a re-render and new hook call)
    expect(useDeviceHistory).toHaveBeenLastCalledWith(
      expect.objectContaining({
        settableOnly: true,
      })
    );
  });

  it('should call refetch when reload button is clicked', async () => {
    const user = userEvent.setup();
    const mockRefetch = vi.fn();

    vi.mocked(useDeviceHistory).mockReturnValue({
      entries: [],
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    });

    render(
      <DeviceHistoryDialog
        device={mockDevice}
        connection={mockConnection}
        isOpen={true}
        onOpenChange={vi.fn()}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
        isConnected={true}
      />
    );

    const reloadButton = screen.getByTitle(/Reload/i);
    await user.click(reloadButton);

    expect(mockRefetch).toHaveBeenCalledTimes(1);
  });

  it('should call onOpenChange when close button is clicked', async () => {
    const user = userEvent.setup();
    const mockOnOpenChange = vi.fn();

    vi.mocked(useDeviceHistory).mockReturnValue({
      entries: [],
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });

    render(
      <DeviceHistoryDialog
        device={mockDevice}
        connection={mockConnection}
        isOpen={true}
        onOpenChange={mockOnOpenChange}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
        isConnected={true}
      />
    );

    const closeButton = screen.getByText(/Close/i);
    await user.click(closeButton);

    expect(mockOnOpenChange).toHaveBeenCalledWith(false);
  });

  it('should format timestamp in MM/DD HH:MM:SS format', () => {
    const mockEntries: DeviceHistoryEntry[] = [
      {
        timestamp: '2024-05-01T03:34:56Z', // UTC
        epc: '80',
        value: { string: 'on', EDT: 'MzA=' },
        origin: 'set',
        settable: true,
      },
    ];

    vi.mocked(useDeviceHistory).mockReturnValue({
      entries: mockEntries,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });

    render(
      <DeviceHistoryDialog
        device={mockDevice}
        connection={mockConnection}
        isOpen={true}
        onOpenChange={vi.fn()}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
        isConnected={true}
      />
    );

    // Should display timestamp in MM/DD HH:MM:SS format
    // Format: 05/01 03:34:56 or similar depending on timezone
    const timestampElements = screen.getAllByText(/\d{2}\/\d{2}\s+\d{2}:\d{2}:\d{2}/);
    expect(timestampElements.length).toBeGreaterThan(0);
  });

  it('should display origin as readable text', () => {
    const mockEntries: DeviceHistoryEntry[] = [
      {
        timestamp: '2024-05-01T12:34:56Z',
        epc: '80',
        value: { string: 'on', EDT: 'MzA=' },
        origin: 'set',
        settable: true,
      },
      {
        timestamp: '2024-05-01T12:35:00Z',
        epc: 'B3',
        value: { number: 25, EDT: 'MjU=' },
        origin: 'notification',
        settable: true,
      },
    ];

    vi.mocked(useDeviceHistory).mockReturnValue({
      entries: mockEntries,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });

    render(
      <DeviceHistoryDialog
        device={mockDevice}
        connection={mockConnection}
        isOpen={true}
        onOpenChange={vi.fn()}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
        isConnected={true}
      />
    );

    // Origin should be displayed as readable text
    // (Actual text depends on language - we'll handle this in implementation)
    const container = screen.getByRole('alertdialog');
    expect(container).toBeInTheDocument();
    // Verify origin text is displayed
    expect(container.textContent).toContain('Operation');
    expect(container.textContent).toContain('Notification');
  });

  it('should apply green styling for online events', () => {
    const mockEntries: DeviceHistoryEntry[] = [
      {
        timestamp: '2024-05-01T12:00:00Z',
        origin: 'online',
        settable: false,
        value: { EDT: '' }, // Empty value for event entries
      },
    ];

    vi.mocked(useDeviceHistory).mockReturnValue({
      entries: mockEntries,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });

    render(
      <DeviceHistoryDialog
        device={mockDevice}
        connection={mockConnection}
        isOpen={true}
        onOpenChange={vi.fn()}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
        isConnected={true}
      />
    );

    // Find the online event row by data-testid
    const eventRow = screen.getByTestId('history-event-online');
    expect(eventRow).toBeInTheDocument();

    // Verify green color classes are applied (table row styling)
    expect(eventRow.className).toMatch(/bg-green-200/);
    expect(eventRow.className).toMatch(/dark:bg-green-900/);
    expect(eventRow.className).toMatch(/font-semibold/);

    // Verify event description is displayed
    expect(eventRow.textContent).toMatch(/Device came online/i);
  });

  it('should apply red styling for offline events', () => {
    const mockEntries: DeviceHistoryEntry[] = [
      {
        timestamp: '2024-05-01T12:00:00Z',
        origin: 'offline',
        settable: false,
        value: { EDT: '' }, // Empty value for event entries
      },
    ];

    vi.mocked(useDeviceHistory).mockReturnValue({
      entries: mockEntries,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });

    render(
      <DeviceHistoryDialog
        device={mockDevice}
        connection={mockConnection}
        isOpen={true}
        onOpenChange={vi.fn()}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
        isConnected={true}
      />
    );

    // Find the offline event row by data-testid
    const eventRow = screen.getByTestId('history-event-offline');
    expect(eventRow).toBeInTheDocument();

    // Verify red color classes are applied (table row styling)
    expect(eventRow.className).toMatch(/bg-red-200/);
    expect(eventRow.className).toMatch(/dark:bg-red-900/);
    expect(eventRow.className).toMatch(/font-semibold/);

    // Verify event description is displayed
    expect(eventRow.textContent).toMatch(/Device went offline/i);
  });

  it('should apply blue styling for settable property entries (row-level)', () => {
    const mockEntries: DeviceHistoryEntry[] = [
      {
        timestamp: '2024-05-01T12:00:00Z',
        epc: '80',
        value: { string: 'on', EDT: 'MzA=' },
        origin: 'notification',
        settable: true,
      },
    ];

    vi.mocked(useDeviceHistory).mockReturnValue({
      entries: mockEntries,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });

    render(
      <DeviceHistoryDialog
        device={mockDevice}
        connection={mockConnection}
        isOpen={true}
        onOpenChange={vi.fn()}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
        isConnected={true}
      />
    );

    // Settable property entries should have blue background for the entire row
    // Use document.querySelectorAll since AlertDialog uses portal
    const blueRows = document.querySelectorAll('.bg-blue-100');
    expect(blueRows.length).toBeGreaterThan(0);
    expect(blueRows[0].className).toMatch(/dark:bg-blue-950/);
    expect(blueRows[0].className).toMatch(/text-blue-900/);
    expect(blueRows[0].className).toMatch(/dark:text-blue-100/);
  });

  it('should not apply background colors for non-settable entries', () => {
    const mockEntries: DeviceHistoryEntry[] = [
      {
        timestamp: '2024-05-01T12:01:00Z',
        epc: 'B3',
        value: { number: 25, EDT: 'MjU=' },
        origin: 'notification',
        settable: false,
      },
    ];

    vi.mocked(useDeviceHistory).mockReturnValue({
      entries: mockEntries,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });

    render(
      <DeviceHistoryDialog
        device={mockDevice}
        connection={mockConnection}
        isOpen={true}
        onOpenChange={vi.fn()}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
        isConnected={true}
      />
    );

    // Non-settable entries should not have colored backgrounds (updated for new log-style)
    // Since this dialog only shows one entry, we need to verify no colored backgrounds exist
    const dialog = screen.getByRole('alertdialog');
    const allCards = dialog.querySelectorAll('[class*="bg-"]');

    // Check that none of the cards have event colors
    let hasColoredBackground = false;
    allCards.forEach((card) => {
      if (
        card.className.includes('bg-green-200') ||
        card.className.includes('bg-red-200') ||
        card.className.includes('bg-blue-200')
      ) {
        hasColoredBackground = true;
      }
    });

    expect(hasColoredBackground).toBe(false);
  });

  it('should apply different colors for different entry types', () => {
    const mockEntries: DeviceHistoryEntry[] = [
      {
        timestamp: '2024-05-01T12:00:00Z',
        epc: '80',
        value: { string: 'on', EDT: 'MzA=' },
        origin: 'set',
        settable: true,
      },
      {
        timestamp: '2024-05-01T12:01:00Z',
        epc: 'B3',
        value: { number: 25, EDT: 'MjU=' },
        origin: 'notification',
        settable: false,
      },
      {
        timestamp: '2024-05-01T12:02:00Z',
        origin: 'online',
        settable: false,
        value: { EDT: '' },
      },
      {
        timestamp: '2024-05-01T12:03:00Z',
        origin: 'offline',
        settable: false,
        value: { EDT: '' },
      },
    ];

    vi.mocked(useDeviceHistory).mockReturnValue({
      entries: mockEntries,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });

    render(
      <DeviceHistoryDialog
        device={mockDevice}
        connection={mockConnection}
        isOpen={true}
        onOpenChange={vi.fn()}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
        isConnected={true}
      />
    );

    // Verify each type has correct colors (updated for row-level coloring)
    // Use document.querySelectorAll since AlertDialog uses portal
    const blueRows = document.querySelectorAll('.bg-blue-100'); // settable row uses bg-blue-100
    const greenCards = document.querySelectorAll('.bg-green-200');
    const redCards = document.querySelectorAll('.bg-red-200');

    // settable -> blue (row-level: bg-blue-100), online -> green (row + timestamp cell = 2), offline -> red (row + timestamp cell = 2), non-settable -> no color
    expect(blueRows.length).toBeGreaterThan(0); // At least one settable row
    expect(greenCards.length).toBe(2); // Event row applies color to both row and timestamp cell
    expect(redCards.length).toBe(2); // Event row applies color to both row and timestamp cell
  });

  it('should display event description for online/offline events', () => {
    const mockEntries: DeviceHistoryEntry[] = [
      {
        timestamp: '2024-05-01T12:00:00Z',
        origin: 'online',
        settable: false,
        value: { EDT: '' }, // Empty value for event entries
      },
      {
        timestamp: '2024-05-01T12:01:00Z',
        origin: 'offline',
        settable: false,
        value: { EDT: '' }, // Empty value for event entries
      },
    ];

    vi.mocked(useDeviceHistory).mockReturnValue({
      entries: mockEntries,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });

    render(
      <DeviceHistoryDialog
        device={mockDevice}
        connection={mockConnection}
        isOpen={true}
        onOpenChange={vi.fn()}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
        isConnected={true}
      />
    );

    // Should display event descriptions (language-dependent)
    const container = screen.getByRole('alertdialog');
    // The actual text depends on the language, but events should be displayed
    expect(container.textContent).toContain('online');
    expect(container.textContent).toContain('offline');
  });

  describe('Device name display', () => {
    it('should display device alias name when alias exists', () => {
      const mockAliases = {
        'Living Room AC': '013001:00000B:ABCDEF0123456789ABCDEF012345',
      };

      const mockAllDevices: Record<string, Device> = {
        '192.168.1.10 0130:1': mockDevice,
        '192.168.1.10 0EF0:1': {
          ip: '192.168.1.10',
          eoj: '0EF0:1',
          name: 'NodeProfile',
          id: '0EF001:00000B:ABCDEF0123456789ABCDEF012345',
          properties: {
            '83': { EDT: 'QUJDREVGMDI5NjU3ODlBQkNERUYwMTIzNDU=' }, // Identification Number
          },
          lastSeen: '2024-05-01T12:00:00Z',
        },
      };

      vi.mocked(useDeviceHistory).mockReturnValue({
        entries: [],
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      });

      render(
        <DeviceHistoryDialog
          device={mockDevice}
          connection={mockConnection}
          isOpen={true}
          onOpenChange={vi.fn()}
          propertyDescriptions={mockPropertyDescriptions}
          classCode="0130"
          isConnected={true}
          aliases={mockAliases}
          allDevices={mockAllDevices}
        />
      );

      const container = screen.getByRole('alertdialog');
      // Should display alias name
      expect(container.textContent).toContain('Living Room AC');
    });

    it('should display device physical name when alias exists', () => {
      const mockAliases = {
        'Living Room AC': '013001:00000B:ABCDEF0123456789ABCDEF012345',
      };

      const mockAllDevices: Record<string, Device> = {
        '192.168.1.10 0130:1': mockDevice,
        '192.168.1.10 0EF0:1': {
          ip: '192.168.1.10',
          eoj: '0EF0:1',
          name: 'NodeProfile',
          id: '0EF001:00000B:ABCDEF0123456789ABCDEF012345',
          properties: {
            '83': { EDT: 'QUJDREVGMDI5NjU3ODlBQkNERUYwMTIzNDU=' }, // Identification Number
          },
          lastSeen: '2024-05-01T12:00:00Z',
        },
      };

      vi.mocked(useDeviceHistory).mockReturnValue({
        entries: [],
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      });

      render(
        <DeviceHistoryDialog
          device={mockDevice}
          connection={mockConnection}
          isOpen={true}
          onOpenChange={vi.fn()}
          propertyDescriptions={mockPropertyDescriptions}
          classCode="0130"
          isConnected={true}
          aliases={mockAliases}
          allDevices={mockAllDevices}
        />
      );

      const container = screen.getByRole('alertdialog');
      // Should display physical device name with "Device:" prefix
      expect(container.textContent).toContain('Device:');
      expect(container.textContent).toContain('HomeAirConditioner');
    });

    it('should display device name when no alias exists', () => {
      const mockAliases = {};
      const mockAllDevices: Record<string, Device> = {
        '192.168.1.10 0130:1': mockDevice,
      };

      vi.mocked(useDeviceHistory).mockReturnValue({
        entries: [],
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      });

      render(
        <DeviceHistoryDialog
          device={mockDevice}
          connection={mockConnection}
          isOpen={true}
          onOpenChange={vi.fn()}
          propertyDescriptions={mockPropertyDescriptions}
          classCode="0130"
          isConnected={true}
          aliases={mockAliases}
          allDevices={mockAllDevices}
        />
      );

      const container = screen.getByRole('alertdialog');
      // Should display device name without "Device:" prefix
      expect(container.textContent).toContain('HomeAirConditioner');
      expect(container.textContent).not.toContain('Device:');
    });

    it('should display IP address and EOJ for all cases', () => {
      const mockAliases = {};
      const mockAllDevices: Record<string, Device> = {
        '192.168.1.10 0130:1': mockDevice,
      };

      vi.mocked(useDeviceHistory).mockReturnValue({
        entries: [],
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      });

      render(
        <DeviceHistoryDialog
          device={mockDevice}
          connection={mockConnection}
          isOpen={true}
          onOpenChange={vi.fn()}
          propertyDescriptions={mockPropertyDescriptions}
          classCode="0130"
          isConnected={true}
          aliases={mockAliases}
          allDevices={mockAllDevices}
        />
      );

      const container = screen.getByRole('alertdialog');
      // Should display IP address and EOJ
      expect(container.textContent).toContain('192.168.1.10');
      expect(container.textContent).toContain('0130:1');
    });
  });

  describe('Row compression for same timestamp, origin, and settable status', () => {
    it('should merge multiple properties with same timestamp, same origin, and same settable status into one row', () => {
      const mockEntries: DeviceHistoryEntry[] = [
        {
          timestamp: '2024-05-01T12:00:00.123Z',
          epc: '80',
          value: { string: 'on', EDT: 'MzA=' },
          origin: 'set',
          settable: true,
        },
        {
          timestamp: '2024-05-01T12:00:00.456Z', // Different milliseconds, same second
          epc: 'B3',
          value: { number: 25, EDT: 'MjU=' },
          origin: 'set', // Same origin as previous entry
          settable: true,
        },
      ];

      vi.mocked(useDeviceHistory).mockReturnValue({
        entries: mockEntries,
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      });

      render(
        <DeviceHistoryDialog
          device={mockDevice}
          connection={mockConnection}
          isOpen={true}
          onOpenChange={vi.fn()}
          propertyDescriptions={mockPropertyDescriptions}
          classCode="0130"
          isConnected={true}
        />
      );

      // Should have only ONE row for both properties (excluding header)
      const table = screen.getByRole('table');
      const rows = table.querySelectorAll('tbody tr');
      expect(rows.length).toBe(1);

      // The row should contain both property values
      const row = rows[0];
      expect(row.textContent).toContain('on');
      expect(row.textContent).toContain('25');
    });

    it('should create separate rows for same timestamp but different origins', () => {
      const mockEntries: DeviceHistoryEntry[] = [
        {
          timestamp: '2024-05-01T12:00:00.123Z',
          epc: '80',
          value: { string: 'on', EDT: 'MzA=' },
          origin: 'set',
          settable: true,
        },
        {
          timestamp: '2024-05-01T12:00:00.456Z', // Different milliseconds, same second
          epc: 'B3',
          value: { number: 25, EDT: 'MjU=' },
          origin: 'notification', // Different origin
          settable: true,
        },
      ];

      vi.mocked(useDeviceHistory).mockReturnValue({
        entries: mockEntries,
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      });

      render(
        <DeviceHistoryDialog
          device={mockDevice}
          connection={mockConnection}
          isOpen={true}
          onOpenChange={vi.fn()}
          propertyDescriptions={mockPropertyDescriptions}
          classCode="0130"
          isConnected={true}
        />
      );

      // Should have TWO rows for different origins (excluding header)
      const table = screen.getByRole('table');
      const rows = table.querySelectorAll('tbody tr');
      expect(rows.length).toBe(2);

      // Each row should have its own origin
      const rowTexts = Array.from(rows).map((r) => r.textContent);
      expect(rowTexts.some((text) => text?.includes('Operation'))).toBe(true); // 'set' origin
      expect(rowTexts.some((text) => text?.includes('Notification'))).toBe(true); // 'notification' origin
    });

    it('should create separate rows for same timestamp and origin but different settable status', () => {
      const mockEntries: DeviceHistoryEntry[] = [
        {
          timestamp: '2024-05-01T12:00:00.123Z',
          epc: '80',
          value: { string: 'on', EDT: 'MzA=' },
          origin: 'notification',
          settable: true, // settable=true
        },
        {
          timestamp: '2024-05-01T12:00:00.456Z', // Different milliseconds, same second
          epc: 'B3',
          value: { number: 25, EDT: 'MjU=' },
          origin: 'notification', // Same origin
          settable: false, // settable=false (different from above)
        },
      ];

      vi.mocked(useDeviceHistory).mockReturnValue({
        entries: mockEntries,
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      });

      render(
        <DeviceHistoryDialog
          device={mockDevice}
          connection={mockConnection}
          isOpen={true}
          onOpenChange={vi.fn()}
          propertyDescriptions={mockPropertyDescriptions}
          classCode="0130"
          isConnected={true}
        />
      );

      // Should have TWO rows for different settable status (excluding header)
      const table = screen.getByRole('table');
      const rows = table.querySelectorAll('tbody tr');
      expect(rows.length).toBe(2);

      // Verify one row has blue styling (settable=true), the other does not
      // Note: bg-blue-100 appears on both the row element and the timestamp cell
      // Therefore, we expect 2 DOM elements with the blue class (1 row × 2 elements)
      const blueRows = document.querySelectorAll('.bg-blue-100');
      expect(blueRows.length).toBe(2); // Row element + timestamp cell = 2 elements total
    });

    it('should keep online/offline events in separate rows regardless of timestamp', () => {
      const mockEntries: DeviceHistoryEntry[] = [
        {
          timestamp: '2024-05-01T12:00:00.123Z',
          epc: '80',
          value: { string: 'on', EDT: 'MzA=' },
          origin: 'set',
          settable: true,
        },
        {
          timestamp: '2024-05-01T12:00:00.456Z', // Different milliseconds, same second
          origin: 'online',
          settable: false,
          value: { EDT: '' },
        },
      ];

      vi.mocked(useDeviceHistory).mockReturnValue({
        entries: mockEntries,
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      });

      render(
        <DeviceHistoryDialog
          device={mockDevice}
          connection={mockConnection}
          isOpen={true}
          onOpenChange={vi.fn()}
          propertyDescriptions={mockPropertyDescriptions}
          classCode="0130"
          isConnected={true}
        />
      );

      // Should have TWO rows (one for property, one for event)
      const table = screen.getByRole('table');
      const rows = table.querySelectorAll('tbody tr');
      expect(rows.length).toBe(2);

      // Event row should be separate
      const eventRow = screen.getByTestId('history-event-online');
      expect(eventRow).toBeInTheDocument();
    });

    it('should handle complex scenario with multiple timestamps and origins', () => {
      const mockEntries: DeviceHistoryEntry[] = [
        // Time 1: Two properties with 'set' origin (same second, different milliseconds)
        {
          timestamp: '2024-05-01T12:00:00.123Z',
          epc: '80',
          value: { string: 'on', EDT: 'MzA=' },
          origin: 'set',
          settable: true,
        },
        {
          timestamp: '2024-05-01T12:00:00.456Z',
          epc: 'B3',
          value: { number: 25, EDT: 'MjU=' },
          origin: 'set',
          settable: true,
        },
        // Time 1: One property with 'notification' origin (same second, different origin)
        {
          timestamp: '2024-05-01T12:00:00.789Z',
          epc: 'B3',
          value: { number: 26, EDT: 'MjY=' },
          origin: 'notification',
          settable: true,
        },
        // Time 2: Online event
        {
          timestamp: '2024-05-01T12:01:00.000Z',
          origin: 'online',
          settable: false,
          value: { EDT: '' },
        },
      ];

      vi.mocked(useDeviceHistory).mockReturnValue({
        entries: mockEntries,
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      });

      render(
        <DeviceHistoryDialog
          device={mockDevice}
          connection={mockConnection}
          isOpen={true}
          onOpenChange={vi.fn()}
          propertyDescriptions={mockPropertyDescriptions}
          classCode="0130"
          isConnected={true}
        />
      );

      // Should have THREE rows:
      // 1. Time 12:00:00, origin 'set' with 80 and B3
      // 2. Time 12:00:00, origin 'notification' with B3
      // 3. Time 12:01:00, origin 'online' event
      const table = screen.getByRole('table');
      const rows = table.querySelectorAll('tbody tr');
      expect(rows.length).toBe(3);
    });

    it('should use the latest value when same property is updated multiple times in the same second', () => {
      const mockEntries: DeviceHistoryEntry[] = [
        {
          timestamp: '2024-05-01T12:00:00.100Z',
          epc: 'B3',
          value: { number: 20, EDT: 'MjA=' },
          origin: 'set',
          settable: true,
        },
        {
          timestamp: '2024-05-01T12:00:00.200Z',
          epc: 'B3',
          value: { number: 25, EDT: 'MjU=' },
          origin: 'set',
          settable: true,
        },
        {
          timestamp: '2024-05-01T12:00:00.900Z', // Latest value in the same second
          epc: 'B3',
          value: { number: 30, EDT: 'MzA=' },
          origin: 'set',
          settable: true,
        },
      ];

      vi.mocked(useDeviceHistory).mockReturnValue({
        entries: mockEntries,
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      });

      render(
        <DeviceHistoryDialog
          device={mockDevice}
          connection={mockConnection}
          isOpen={true}
          onOpenChange={vi.fn()}
          propertyDescriptions={mockPropertyDescriptions}
          classCode="0130"
          isConnected={true}
        />
      );

      // Should have only ONE row (all entries are in the same second with same origin)
      const table = screen.getByRole('table');
      const rows = table.querySelectorAll('tbody tr');
      expect(rows.length).toBe(1);

      // The row should contain the LATEST value (30), not earlier values
      const row = rows[0];
      expect(row.textContent).toContain('30');
      expect(row.textContent).not.toContain('20');
      expect(row.textContent).not.toContain('25');
    });
  });

  describe('Table format display with properties as columns', () => {
    it('should render history entries in table format with properties as columns', () => {
      const mockEntries: DeviceHistoryEntry[] = [
        {
          timestamp: '2024-05-01T12:34:56.789Z',
          epc: '80',
          value: { string: 'on', EDT: 'MzA=' },
          origin: 'set',
          settable: true,
        },
        {
          timestamp: '2024-05-01T12:35:10.123Z',
          epc: 'B3',
          value: { number: 25, EDT: 'MjU=' },
          origin: 'notification',
          settable: true,
        },
      ];

      vi.mocked(useDeviceHistory).mockReturnValue({
        entries: mockEntries,
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      });

      render(
        <DeviceHistoryDialog
          device={mockDevice}
          connection={mockConnection}
          isOpen={true}
          onOpenChange={vi.fn()}
          propertyDescriptions={mockPropertyDescriptions}
          classCode="0130"
          isConnected={true}
        />
      );

      // Table should exist
      const table = screen.getByRole('table');
      expect(table).toBeInTheDocument();

      // Table headers should be present
      expect(screen.getByText(/Time/i)).toBeInTheDocument();
      expect(screen.getByText(/Origin/i)).toBeInTheDocument();

      // Property names should be in column headers
      expect(screen.getByText(/Operation Status/i)).toBeInTheDocument();
      expect(screen.getByText(/Temperature Setting/i)).toBeInTheDocument();
    });

    it('should display event entries with special styling in table', () => {
      const mockEntries: DeviceHistoryEntry[] = [
        {
          timestamp: '2024-05-01T12:00:00Z',
          origin: 'online',
          settable: false,
          value: { EDT: '' },
        },
        {
          timestamp: '2024-05-01T12:01:00Z',
          origin: 'offline',
          settable: false,
          value: { EDT: '' },
        },
      ];

      vi.mocked(useDeviceHistory).mockReturnValue({
        entries: mockEntries,
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      });

      render(
        <DeviceHistoryDialog
          device={mockDevice}
          connection={mockConnection}
          isOpen={true}
          onOpenChange={vi.fn()}
          propertyDescriptions={mockPropertyDescriptions}
          classCode="0130"
          isConnected={true}
        />
      );

      // Event entries should be displayed
      const onlineEvent = screen.getByTestId('history-event-online');
      const offlineEvent = screen.getByTestId('history-event-offline');

      expect(onlineEvent).toBeInTheDocument();
      expect(offlineEvent).toBeInTheDocument();

      // Event descriptions should be in the table
      expect(screen.getByText(/Device came online/i)).toBeInTheDocument();
      expect(screen.getByText(/Device went offline/i)).toBeInTheDocument();
    });

    it('should truncate long property names with title attribute', () => {
      const mockPropertyDescriptionsWithLongName = {
        '0130': {
          classCode: '0130',
          properties: {
            B3: {
              description: 'Very Long Property Name That Should Be Truncated In The UI',
              numberDesc: {
                min: 0,
                max: 50,
                offset: 0,
                unit: 'C',
                edtLen: 1,
              },
            },
          },
        },
      };

      const mockEntries: DeviceHistoryEntry[] = [
        {
          timestamp: '2024-05-01T12:34:56.789Z',
          epc: 'B3',
          value: { number: 25, EDT: 'MjU=' },
          origin: 'set',
          settable: true,
        },
      ];

      vi.mocked(useDeviceHistory).mockReturnValue({
        entries: mockEntries,
        isLoading: false,
        error: null,
        refetch: vi.fn(),
      });

      render(
        <DeviceHistoryDialog
          device={mockDevice}
          connection={mockConnection}
          isOpen={true}
          onOpenChange={vi.fn()}
          propertyDescriptions={mockPropertyDescriptionsWithLongName}
          classCode="0130"
          isConnected={true}
        />
      );

      // Long property name should exist in table header
      const propertyHeader = screen.getByText(/Very Long Property Name/i);
      expect(propertyHeader).toBeInTheDocument();

      // The parent TableHead element should have title attribute
      const tableHead = propertyHeader.closest('th');
      expect(tableHead).toHaveAttribute('title', 'Very Long Property Name That Should Be Truncated In The UI');

      // Should have truncate class
      expect(propertyHeader.className).toMatch(/truncate/);
    });
  });
});
