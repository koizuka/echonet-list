import { render, screen, fireEvent } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import App from './App';

// Mock deviceIdHelper
vi.mock('@/libs/deviceIdHelper', () => ({
  deviceHasAlias: () => ({ hasAlias: false, aliasName: undefined, deviceIdentifier: '192.168.1.100 0291:1' }),
  getDeviceAliases: () => ({ aliases: [], deviceIdentifier: '192.168.1.100 0291:1' }),
  getDeviceClassCode: () => '0291',
  getDeviceIdentifierForAlias: () => '192.168.1.100 0291:1',
}));

// Mock languageHelper
vi.mock('@/libs/languageHelper', () => ({
  getCurrentLocale: () => 'en',
}));

// Mock usePropertyDescriptions hook
vi.mock('@/hooks/usePropertyDescriptions', () => ({
  usePropertyDescriptions: () => ({
    devices: {
      '192.168.1.100 0291:1': {
        ip: '192.168.1.100',
        eoj: '0291:1',
        properties: {
          '80': { edt: 'MA==' }, // Operation status: ON
          '81': { edt: 'AA==' }, // Installation location
        },
        isOffline: false,
      },
    },
    aliases: {},
    groups: {},
    locationSettings: { aliases: {}, order: [] },
    propertyDescriptions: {},
    connectionState: 'connected',
    initialStateReceived: true,
    connectedAt: new Date(),
    serverStartupTime: null,
    setProperty: vi.fn().mockResolvedValue({}),
    updateDeviceProperties: vi.fn().mockResolvedValue({}),
    discoverDevices: vi.fn(),
    setAlias: vi.fn().mockResolvedValue({}),
    deleteAlias: vi.fn().mockResolvedValue({}),
    setGroup: vi.fn().mockResolvedValue({}),
    deleteGroup: vi.fn().mockResolvedValue({}),
    deleteDevice: vi.fn().mockResolvedValue({}),
    setLocationAlias: vi.fn().mockResolvedValue({}),
    deleteLocationAlias: vi.fn().mockResolvedValue({}),
    setLocationOrder: vi.fn().mockResolvedValue({}),
    checkConnection: vi.fn().mockResolvedValue(true),
    getDeviceClassCode: () => '0291',
  }),
}));

// Mock useWebSocketConnection hook
vi.mock('@/hooks/useWebSocketConnection', () => ({
  useWebSocketConnection: () => ({
    connectionState: 'connected',
    connect: vi.fn(),
    disconnect: vi.fn(),
    sendMessage: vi.fn().mockResolvedValue({}),
  }),
}));

// Mock lazy-loaded LocationSettingsDialog
vi.mock('@/components/LocationSettingsDialog', () => ({
  LocationSettingsDialog: ({ open }: { open: boolean; onOpenChange: (open: boolean) => void }) => (
    open ? <div data-testid="location-settings-dialog">Location Settings Dialog</div> : null
  ),
}));

// Mock usePersistedTab to return Dashboard as default
vi.mock('@/hooks/usePersistedTab', () => ({
  usePersistedTab: () => ({
    selectedTab: 'Dashboard',
    selectTab: vi.fn(),
  }),
}));

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders ECHONET List title', () => {
    render(<App />);
    expect(screen.getByText('ECHONET List')).toBeInTheDocument();
  });

  describe('Tab list', () => {
    it('renders location settings button in tab list', () => {
      render(<App />);
      const settingsButton = screen.getByTestId('location-settings-button');
      expect(settingsButton).toBeInTheDocument();
    });

    it('location settings button has aria-label for accessibility', () => {
      render(<App />);
      const settingsButton = screen.getByTestId('location-settings-button');
      expect(settingsButton).toHaveAttribute('aria-label', '設置場所の設定');
    });

    it('location settings button has title attribute', () => {
      render(<App />);
      const settingsButton = screen.getByTestId('location-settings-button');
      expect(settingsButton).toHaveAttribute('title', 'Location Settings');
    });

    it('settings button is clickable', () => {
      render(<App />);
      const settingsButton = screen.getByTestId('location-settings-button');

      // Verify the button is clickable (not disabled)
      expect(settingsButton).not.toBeDisabled();
      expect(() => fireEvent.click(settingsButton)).not.toThrow();
    });

    it('renders Dashboard tab', () => {
      render(<App />);
      expect(screen.getByTestId('tab-Dashboard')).toBeInTheDocument();
    });

    it('renders All tab', () => {
      render(<App />);
      expect(screen.getByTestId('tab-All')).toBeInTheDocument();
    });

    it('applies horizontal scroll class when Dashboard is selected', () => {
      render(<App />);
      // Dashboard is selected by default
      const tabsList = screen.getByRole('tablist');
      expect(tabsList).toHaveClass('scrollbar-hide');
      expect(tabsList).toHaveClass('flex-nowrap');
      expect(tabsList).toHaveClass('overflow-x-auto');
    });

  });
});
