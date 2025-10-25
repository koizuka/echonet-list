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

  it('should format timestamp in local time', () => {
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

    // Should display timestamp (exact format depends on locale)
    // Just verify it's rendered
    const timestampElements = screen.getAllByText(/2024/);
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

    // Find the online event card by data-testid
    const eventCard = screen.getByTestId('history-event-online');
    expect(eventCard).toBeInTheDocument();

    // Verify green color classes are applied
    expect(eventCard.className).toMatch(/border-green-200/);
    expect(eventCard.className).toMatch(/bg-green-50/);
    expect(eventCard.className).toMatch(/dark:border-green-800/);
    expect(eventCard.className).toMatch(/dark:bg-green-950/);

    // Verify event description is displayed
    expect(eventCard.textContent).toMatch(/Device came online/i);
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

    // Find the offline event card by data-testid
    const eventCard = screen.getByTestId('history-event-offline');
    expect(eventCard).toBeInTheDocument();

    // Verify red color classes are applied
    expect(eventCard.className).toMatch(/border-red-200/);
    expect(eventCard.className).toMatch(/bg-red-50/);
    expect(eventCard.className).toMatch(/dark:border-red-800/);
    expect(eventCard.className).toMatch(/dark:bg-red-950/);

    // Verify event description is displayed
    expect(eventCard.textContent).toMatch(/Device went offline/i);
  });

  it('should apply blue styling for settable property entries', () => {
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

    // Settable property entries should have blue background
    // Use document.querySelectorAll since AlertDialog uses portal
    const blueCards = document.querySelectorAll('.border-blue-200');
    expect(blueCards.length).toBe(1);
    expect(blueCards[0].className).toMatch(/bg-blue-50/);
    expect(blueCards[0].className).toMatch(/dark:border-blue-800/);
    expect(blueCards[0].className).toMatch(/dark:bg-blue-950/);
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

    // Non-settable entries should not have colored backgrounds
    // Since this dialog only shows one entry, we need to verify no colored backgrounds exist
    const dialog = screen.getByRole('alertdialog');
    const allCards = dialog.querySelectorAll('.border');

    // Check that none of the cards have event colors
    let hasColoredBackground = false;
    allCards.forEach((card) => {
      if (
        card.className.includes('border-green-200') ||
        card.className.includes('border-red-200') ||
        card.className.includes('border-blue-200')
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

    // Verify each type has correct colors
    // Use document.querySelectorAll since AlertDialog uses portal
    const blueCards = document.querySelectorAll('.border-blue-200');
    const greenCards = document.querySelectorAll('.border-green-200');
    const redCards = document.querySelectorAll('.border-red-200');

    // settable -> blue, online -> green, offline -> red, non-settable -> no color
    expect(blueCards.length).toBe(1);
    expect(greenCards.length).toBe(1);
    expect(redCards.length).toBe(1);
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
});
