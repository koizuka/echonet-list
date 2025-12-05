import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { DashboardCard } from './DashboardCard';
import type { Device, PropertyDescriptionData } from '@/hooks/types';
import * as deviceIdHelper from '@/libs/deviceIdHelper';

// Mock ResizeObserver for tests
global.ResizeObserver = vi.fn(() => ({
  observe: vi.fn(),
  disconnect: vi.fn(),
  unobserve: vi.fn(),
}));

// Mock deviceIdHelper functions
vi.mock('@/libs/deviceIdHelper', () => ({
  deviceHasAlias: vi.fn(() => ({ hasAlias: false, aliasName: undefined, deviceIdentifier: '192.168.1.100 0130:1' })),
  getDeviceIdentifierForAlias: vi.fn(() => '192.168.1.100 0130:1'),
}));

// Mock languageHelper to always return 'en' for consistent test behavior
vi.mock('@/libs/languageHelper', () => ({
  isJapanese: vi.fn(() => false),
  getCurrentLocale: vi.fn(() => 'en')
}));

describe('DashboardCard', () => {
  const createDevice = (overrides: Partial<Device> = {}): Device => ({
    ip: '192.168.1.100',
    eoj: '0130:1',
    name: 'Home Air Conditioner',
    id: undefined,
    lastSeen: new Date().toISOString(),
    properties: {
      '80': { string: 'on' },
      '81': { string: 'living' },
      'BB': { number: 24 },
      'B0': { string: 'cooling' },
      '9E': { EDT: btoa(String.fromCharCode(0x01, 0x80)) } // Set Property Map with EPC 0x80
    },
    ...overrides
  });

  const mockPropertyDescriptions: Record<string, PropertyDescriptionData> = {
    '': {
      classCode: '',
      properties: {
        '80': { description: 'Operation Status' },
        '81': {
          description: 'Installation Location',
          aliases: { living: 'Living Room', kitchen: 'Kitchen' }
        }
      }
    },
    '0130': {
      classCode: '0130',
      properties: {
        'BB': {
          description: 'Room Temperature',
          numberDesc: { min: -50, max: 50, offset: 0, unit: '\u00B0C', edtLen: 1 }
        },
        'B0': {
          description: 'Operation Mode',
          aliases: { cooling: 'Cooling', heating: 'Heating', auto: 'Auto', dry: 'Dry', fan: 'Fan' }
        }
      }
    }
  };

  const mockDevices: Record<string, Device> = {
    '192.168.1.100 0130:1': createDevice()
  };

  const mockOnPropertyChange = vi.fn().mockResolvedValue(undefined);

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(deviceIdHelper.deviceHasAlias).mockReturnValue({
      hasAlias: false,
      aliasName: undefined,
      deviceIdentifier: '192.168.1.100 0130:1'
    });
  });

  describe('rendering', () => {
    it('should not render device name by default (collapsed state)', () => {
      render(
        <DashboardCard
          device={createDevice()}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          devices={mockDevices}
          aliases={{}}
          isConnected={true}
        />
      );

      expect(screen.queryByText('Home Air Conditioner')).not.toBeInTheDocument();
    });

    it('should render device name when expanded', () => {
      render(
        <DashboardCard
          device={createDevice()}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          devices={mockDevices}
          aliases={{}}
          isConnected={true}
        />
      );

      // Click to expand
      const expandableArea = screen.getByTestId('dashboard-card-expandable-192.168.1.100-0130:1');
      fireEvent.click(expandableArea);

      expect(screen.getByText('Home Air Conditioner')).toBeInTheDocument();
    });

    it('should hide device name when collapsed after expand', () => {
      render(
        <DashboardCard
          device={createDevice()}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          devices={mockDevices}
          aliases={{}}
          isConnected={true}
        />
      );

      const expandableArea = screen.getByTestId('dashboard-card-expandable-192.168.1.100-0130:1');

      // Expand
      fireEvent.click(expandableArea);
      expect(screen.getByText('Home Air Conditioner')).toBeInTheDocument();

      // Collapse
      fireEvent.click(expandableArea);
      expect(screen.queryByText('Home Air Conditioner')).not.toBeInTheDocument();
    });

    it('should render device name from alias when expanded', () => {
      vi.mocked(deviceIdHelper.deviceHasAlias).mockReturnValue({
        hasAlias: true,
        aliasName: 'Living Room AC',
        deviceIdentifier: '192.168.1.100 0130:1'
      });

      render(
        <DashboardCard
          device={createDevice()}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          devices={mockDevices}
          aliases={{ 'Living Room AC': '192.168.1.100 0130:1' }}
          isConnected={true}
        />
      );

      // Click to expand
      const expandableArea = screen.getByTestId('dashboard-card-expandable-192.168.1.100-0130:1');
      fireEvent.click(expandableArea);

      expect(screen.getByText('Living Room AC')).toBeInTheDocument();
    });

    it('should render status property values', () => {
      render(
        <DashboardCard
          device={createDevice()}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          devices={mockDevices}
          aliases={{}}
          isConnected={true}
        />
      );

      // Should display temperature and operation mode
      expect(screen.getByText('24°C')).toBeInTheDocument();
      expect(screen.getByText('cooling')).toBeInTheDocument();
    });

    it('should render "---" when status properties are not available', () => {
      const deviceWithoutStatus = createDevice({
        properties: {
          '80': { string: 'on' },
          '81': { string: 'living' },
          '9E': { EDT: btoa(String.fromCharCode(0x01, 0x80)) }
          // No BB or B0 properties
        }
      });

      render(
        <DashboardCard
          device={deviceWithoutStatus}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          devices={mockDevices}
          aliases={{}}
          isConnected={true}
        />
      );

      expect(screen.getByText('---')).toBeInTheDocument();
    });
  });

  describe('on/off switch', () => {
    it('should render switch when operation status is settable', () => {
      render(
        <DashboardCard
          device={createDevice()}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          devices={mockDevices}
          aliases={{}}
          isConnected={true}
        />
      );

      const switchElement = screen.getByRole('switch');
      expect(switchElement).toBeInTheDocument();
    });

    it('should not render switch when operation status is not settable', () => {
      const deviceNotSettable = createDevice({
        properties: {
          '80': { string: 'on' },
          '81': { string: 'living' },
          'BB': { number: 24 },
          '9E': { EDT: btoa(String.fromCharCode(0x00)) } // Empty Set Property Map
        }
      });

      render(
        <DashboardCard
          device={deviceNotSettable}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          devices={mockDevices}
          aliases={{}}
          isConnected={true}
        />
      );

      expect(screen.queryByRole('switch')).not.toBeInTheDocument();
    });

    it('should call onPropertyChange when switch is toggled', async () => {
      render(
        <DashboardCard
          device={createDevice()}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          devices={mockDevices}
          aliases={{}}
          isConnected={true}
        />
      );

      const switchElement = screen.getByRole('switch');
      fireEvent.click(switchElement);

      expect(mockOnPropertyChange).toHaveBeenCalledWith(
        '192.168.1.100 0130:1',
        '80',
        { string: 'off' }
      );
    });

    it('should disable switch when not connected', () => {
      render(
        <DashboardCard
          device={createDevice()}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          devices={mockDevices}
          aliases={{}}
          isConnected={false}
        />
      );

      const switchElement = screen.getByRole('switch');
      expect(switchElement).toBeDisabled();
    });

    it('should disable switch when device is offline', () => {
      const offlineDevice = createDevice({ isOffline: true });

      render(
        <DashboardCard
          device={offlineDevice}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          devices={mockDevices}
          aliases={{}}
          isConnected={true}
        />
      );

      const switchElement = screen.getByRole('switch');
      expect(switchElement).toBeDisabled();
    });

    it('should not expand when switch is clicked', () => {
      render(
        <DashboardCard
          device={createDevice()}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          devices={mockDevices}
          aliases={{}}
          isConnected={true}
        />
      );

      // Click the switch
      const switchElement = screen.getByRole('switch');
      fireEvent.click(switchElement);

      // Device name should not be visible (card should not expand)
      expect(screen.queryByText('Home Air Conditioner')).not.toBeInTheDocument();
    });
  });

  describe('styling', () => {
    it('should have opacity-50 when device is offline', () => {
      const offlineDevice = createDevice({ isOffline: true });

      render(
        <DashboardCard
          device={offlineDevice}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          devices={mockDevices}
          aliases={{}}
          isConnected={true}
        />
      );

      const card = screen.getByTestId('dashboard-card-192.168.1.100-0130:1');
      expect(card).toHaveClass('opacity-50');
    });

    it('should have green border when device is operational (on)', () => {
      render(
        <DashboardCard
          device={createDevice()}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          devices={mockDevices}
          aliases={{}}
          isConnected={true}
        />
      );

      const card = screen.getByTestId('dashboard-card-192.168.1.100-0130:1');
      expect(card).toHaveClass('border-green-500/60');
    });

    it('should not have green border when device is off', () => {
      const offDevice = createDevice({
        properties: {
          '80': { string: 'off' },
          '81': { string: 'living' },
          'BB': { number: 24 },
          '9E': { EDT: btoa(String.fromCharCode(0x01, 0x80)) }
        }
      });

      render(
        <DashboardCard
          device={offDevice}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          devices={mockDevices}
          aliases={{}}
          isConnected={true}
        />
      );

      const card = screen.getByTestId('dashboard-card-192.168.1.100-0130:1');
      expect(card).not.toHaveClass('border-green-500/60');
    });
  });
});
