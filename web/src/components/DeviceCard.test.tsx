import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { DeviceCard } from './DeviceCard';
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
  deviceHasAlias: vi.fn(() => ({ hasAlias: false, aliasName: undefined, deviceIdentifier: '192.168.1.100 0291:1' })),
  getDeviceIdentifierForAlias: vi.fn(() => '192.168.1.100 0291:1'),
  getDeviceAliases: vi.fn(() => ({ aliases: [], deviceIdentifier: '192.168.1.100 0291:1' }))
}));

// Mock AliasEditor component
vi.mock('@/components/AliasEditor', () => ({
  AliasEditor: () => <input placeholder="Enter alias name" />
}));

describe('DeviceCard', () => {
  const mockDevice: Device = {
    ip: '192.168.1.100',
    eoj: '0291:1',
    name: 'Single Function Lighting',
    id: undefined,
    lastSeen: new Date().toISOString(),
    properties: {
      '80': { string: 'on' },
      '81': { string: 'living' },
      'B0': { number: 50 },
      '9E': { EDT: btoa(String.fromCharCode(0x02, 0x80, 0xB0)) }
    }
  };

  const mockPropertyDescriptions: Record<string, PropertyDescriptionData> = {
    '': {
      classCode: '',
      properties: {
        '80': { description: 'Operation Status' },
        '81': { description: 'Installation Location' },
        '9E': { description: 'Set Property Map' }
      }
    },
    '0291': {
      classCode: '0291',
      properties: {
        'B0': { description: 'Illuminance Level' }
      }
    }
  };

  const mockOnPropertyChange = vi.fn();
  const mockOnUpdateProperties = vi.fn();
  const mockOnAddAlias = vi.fn();
  const mockOnDeleteAlias = vi.fn();
  const mockGetDeviceClassCode = vi.fn().mockReturnValue('0291');

  beforeEach(() => {
    vi.clearAllMocks();
    // Reset mocks to default behavior
    vi.mocked(deviceIdHelper.deviceHasAlias).mockReturnValue({ hasAlias: false, aliasName: undefined, deviceIdentifier: '192.168.1.100 0291:1' });
    vi.mocked(deviceIdHelper.getDeviceAliases).mockReturnValue({ aliases: [], deviceIdentifier: '192.168.1.100 0291:1' });
  });

  describe('Compact Mode (collapsed)', () => {
    it('should show only primary properties in compact mode', () => {
      render(
        <DeviceCard
          device={mockDevice}
          isExpanded={false}
          onToggleExpansion={vi.fn()}
          onPropertyChange={mockOnPropertyChange}
          onUpdateProperties={mockOnUpdateProperties}
          propertyDescriptions={mockPropertyDescriptions}
          getDeviceClassCode={mockGetDeviceClassCode}
          devices={{ [`${mockDevice.ip} ${mockDevice.eoj}`]: mockDevice }}
          aliases={{}}
        />
      );

      // Should show primary properties (Operation Status and Illuminance Level)
      expect(screen.getByText('Operation Status:')).toBeInTheDocument();
      expect(screen.getByText('Illuminance Level:')).toBeInTheDocument();
      
      // Should NOT show secondary properties (Installation Location)
      expect(screen.queryByText('Installation Location:')).not.toBeInTheDocument();
      
      // Should NOT show "Other Properties" section
      expect(screen.queryByText('Other Properties')).not.toBeInTheDocument();
    });

    it('should use compact styling for property rows', () => {
      render(
        <DeviceCard
          device={mockDevice}
          isExpanded={false}
          onToggleExpansion={vi.fn()}
          onPropertyChange={mockOnPropertyChange}
          onUpdateProperties={mockOnUpdateProperties}
          propertyDescriptions={mockPropertyDescriptions}
          getDeviceClassCode={mockGetDeviceClassCode}
          devices={{ [`${mockDevice.ip} ${mockDevice.eoj}`]: mockDevice }}
          aliases={{}}
        />
      );

      // Check for compact text size
      const propertyRows = screen.getAllByText(/:/);
      propertyRows.forEach(row => {
        const container = row.closest('.text-xs');
        expect(container).toBeInTheDocument();
      });
    });

    it('should not show alias editor in compact mode', () => {
      render(
        <DeviceCard
          device={mockDevice}
          isExpanded={false}
          onToggleExpansion={vi.fn()}
          onPropertyChange={mockOnPropertyChange}
          onUpdateProperties={mockOnUpdateProperties}
          propertyDescriptions={mockPropertyDescriptions}
          getDeviceClassCode={mockGetDeviceClassCode}
          devices={{ [`${mockDevice.ip} ${mockDevice.eoj}`]: mockDevice }}
          aliases={{}}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
        />
      );

      // Alias editor should not be present
      expect(screen.queryByPlaceholderText(/alias/i)).not.toBeInTheDocument();
    });

    it('should not show device IP and EOJ in compact mode', () => {
      render(
        <DeviceCard
          device={mockDevice}
          isExpanded={false}
          onToggleExpansion={vi.fn()}
          onPropertyChange={mockOnPropertyChange}
          onUpdateProperties={mockOnUpdateProperties}
          propertyDescriptions={mockPropertyDescriptions}
          getDeviceClassCode={mockGetDeviceClassCode}
          devices={{ [`${mockDevice.ip} ${mockDevice.eoj}`]: mockDevice }}
          aliases={{}}
        />
      );

      // IP and EOJ should not be visible
      expect(screen.queryByText(/192\.168\.1\.100/)).not.toBeInTheDocument();
      expect(screen.queryByText(/0291:1/)).not.toBeInTheDocument();
    });
  });

  describe('Full Mode (expanded)', () => {
    it('should show all properties in full mode', () => {
      render(
        <DeviceCard
          device={mockDevice}
          isExpanded={true}
          onToggleExpansion={vi.fn()}
          onPropertyChange={mockOnPropertyChange}
          onUpdateProperties={mockOnUpdateProperties}
          propertyDescriptions={mockPropertyDescriptions}
          getDeviceClassCode={mockGetDeviceClassCode}
          devices={{ [`${mockDevice.ip} ${mockDevice.eoj}`]: mockDevice }}
          aliases={{}}
        />
      );

      // Should show primary properties
      expect(screen.getByText('Operation Status:')).toBeInTheDocument();
      expect(screen.getByText('Illuminance Level:')).toBeInTheDocument();
      
      // Should show secondary properties under "Other Properties"
      expect(screen.getByText('Other Properties')).toBeInTheDocument();
      expect(screen.getByText('Installation Location:')).toBeInTheDocument();
    });

    it('should show device IP and EOJ in full mode', () => {
      render(
        <DeviceCard
          device={mockDevice}
          isExpanded={true}
          onToggleExpansion={vi.fn()}
          onPropertyChange={mockOnPropertyChange}
          onUpdateProperties={mockOnUpdateProperties}
          propertyDescriptions={mockPropertyDescriptions}
          getDeviceClassCode={mockGetDeviceClassCode}
          devices={{ [`${mockDevice.ip} ${mockDevice.eoj}`]: mockDevice }}
          aliases={{}}
        />
      );

      // IP and EOJ should be visible
      expect(screen.getByText(/192\.168\.1\.100.*0291:1/)).toBeInTheDocument();
    });

    it('should show alias editor in full mode when props are provided', () => {
      render(
        <DeviceCard
          device={mockDevice}
          isExpanded={true}
          onToggleExpansion={vi.fn()}
          onPropertyChange={mockOnPropertyChange}
          onUpdateProperties={mockOnUpdateProperties}
          propertyDescriptions={mockPropertyDescriptions}
          getDeviceClassCode={mockGetDeviceClassCode}
          devices={{ [`${mockDevice.ip} ${mockDevice.eoj}`]: mockDevice }}
          aliases={{}}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
        />
      );

      // Alias editor should be present
      expect(screen.getByPlaceholderText(/Enter alias name/i)).toBeInTheDocument();
    });

    it('should use full styling with more spacing', () => {
      render(
        <DeviceCard
          device={mockDevice}
          isExpanded={true}
          onToggleExpansion={vi.fn()}
          onPropertyChange={mockOnPropertyChange}
          onUpdateProperties={mockOnUpdateProperties}
          propertyDescriptions={mockPropertyDescriptions}
          getDeviceClassCode={mockGetDeviceClassCode}
          devices={{ [`${mockDevice.ip} ${mockDevice.eoj}`]: mockDevice }}
          aliases={{}}
        />
      );

      // Check for full mode spacing
      const mainContent = screen.getByText('Operation Status:').closest('.space-y-3');
      expect(mainContent).toBeInTheDocument();
    });
  });

  describe('Expand/Collapse Toggle', () => {
    it('should call onToggleExpansion when chevron button is clicked', () => {
      const mockToggle = vi.fn();
      
      render(
        <DeviceCard
          device={mockDevice}
          isExpanded={false}
          onToggleExpansion={mockToggle}
          onPropertyChange={mockOnPropertyChange}
          onUpdateProperties={mockOnUpdateProperties}
          propertyDescriptions={mockPropertyDescriptions}
          getDeviceClassCode={mockGetDeviceClassCode}
          devices={{ [`${mockDevice.ip} ${mockDevice.eoj}`]: mockDevice }}
          aliases={{}}
        />
      );

      const toggleButton = screen.getByTestId('expand-collapse-button');
      fireEvent.click(toggleButton);

      expect(mockToggle).toHaveBeenCalledTimes(1);
    });

    it('should show ChevronDown when collapsed', () => {
      render(
        <DeviceCard
          device={mockDevice}
          isExpanded={false}
          onToggleExpansion={vi.fn()}
          onPropertyChange={mockOnPropertyChange}
          onUpdateProperties={mockOnUpdateProperties}
          propertyDescriptions={mockPropertyDescriptions}
          getDeviceClassCode={mockGetDeviceClassCode}
          devices={{ [`${mockDevice.ip} ${mockDevice.eoj}`]: mockDevice }}
          aliases={{}}
        />
      );

      const toggleButton = screen.getByTestId('expand-collapse-button');
      // ChevronDown has viewBox 0 0 24 24
      expect(toggleButton.querySelector('svg')).toBeInTheDocument();
    });

    it('should show ChevronUp when expanded', () => {
      render(
        <DeviceCard
          device={mockDevice}
          isExpanded={true}
          onToggleExpansion={vi.fn()}
          onPropertyChange={mockOnPropertyChange}
          onUpdateProperties={mockOnUpdateProperties}
          propertyDescriptions={mockPropertyDescriptions}
          getDeviceClassCode={mockGetDeviceClassCode}
          devices={{ [`${mockDevice.ip} ${mockDevice.eoj}`]: mockDevice }}
          aliases={{}}
        />
      );

      const toggleButton = screen.getByTestId('expand-collapse-button');
      // ChevronUp has viewBox 0 0 24 24
      expect(toggleButton.querySelector('svg')).toBeInTheDocument();
    });
  });

  describe('Update Properties Button', () => {
    it('should show refresh button and call handler when clicked', () => {
      render(
        <DeviceCard
          device={mockDevice}
          isExpanded={false}
          onToggleExpansion={vi.fn()}
          onPropertyChange={mockOnPropertyChange}
          onUpdateProperties={mockOnUpdateProperties}
          propertyDescriptions={mockPropertyDescriptions}
          getDeviceClassCode={mockGetDeviceClassCode}
          devices={{ [`${mockDevice.ip} ${mockDevice.eoj}`]: mockDevice }}
          aliases={{}}
        />
      );

      const refreshButton = screen.getByTestId('update-properties-button');
      fireEvent.click(refreshButton);

      expect(mockOnUpdateProperties).toHaveBeenCalledWith('192.168.1.100 0291:1');
    });

    it('should show spinning animation when updating', () => {
      render(
        <DeviceCard
          device={mockDevice}
          isExpanded={false}
          onToggleExpansion={vi.fn()}
          onPropertyChange={mockOnPropertyChange}
          onUpdateProperties={mockOnUpdateProperties}
          propertyDescriptions={mockPropertyDescriptions}
          getDeviceClassCode={mockGetDeviceClassCode}
          devices={{ [`${mockDevice.ip} ${mockDevice.eoj}`]: mockDevice }}
          aliases={{}}
          isUpdating={true}
        />
      );

      const refreshButton = screen.getByTestId('update-properties-button');
      expect(refreshButton).toBeDisabled();
      expect(refreshButton.querySelector('.animate-spin')).toBeInTheDocument();
    });
  });

  describe('Alias Display', () => {
    it('should show alias name instead of device name when alias exists', () => {
      // Mock to return that device has alias
      vi.mocked(deviceIdHelper.deviceHasAlias).mockReturnValue({ hasAlias: true, aliasName: 'Living Light', deviceIdentifier: '192.168.1.100 0291:1' });
      
      const aliasedDevice = { ...mockDevice };
      const devices = { [`${mockDevice.ip} ${mockDevice.eoj}`]: aliasedDevice };
      const aliases = { 'Living Light': `${mockDevice.ip} ${mockDevice.eoj}` };

      render(
        <DeviceCard
          device={aliasedDevice}
          isExpanded={false}
          onToggleExpansion={vi.fn()}
          onPropertyChange={mockOnPropertyChange}
          onUpdateProperties={mockOnUpdateProperties}
          propertyDescriptions={mockPropertyDescriptions}
          getDeviceClassCode={mockGetDeviceClassCode}
          devices={devices}
          aliases={aliases}
        />
      );

      expect(screen.getByTestId('device-title')).toHaveTextContent('Living Light');
    });

    it('should show multiple alias indicator in compact mode', () => {
      // Mock to return multiple aliases
      vi.mocked(deviceIdHelper.getDeviceAliases).mockReturnValue({ 
        aliases: ['Living Light', 'Main Light'],
        deviceIdentifier: '192.168.1.100 0291:1'
      });
      
      const aliasedDevice = { ...mockDevice };
      const devices = { [`${mockDevice.ip} ${mockDevice.eoj}`]: aliasedDevice };
      const aliases = { 
        'Living Light': `${mockDevice.ip} ${mockDevice.eoj}`,
        'Main Light': `${mockDevice.ip} ${mockDevice.eoj}` 
      };

      render(
        <DeviceCard
          device={aliasedDevice}
          isExpanded={false}
          onToggleExpansion={vi.fn()}
          onPropertyChange={mockOnPropertyChange}
          onUpdateProperties={mockOnUpdateProperties}
          propertyDescriptions={mockPropertyDescriptions}
          getDeviceClassCode={mockGetDeviceClassCode}
          devices={devices}
          aliases={aliases}
        />
      );

      // Should show the count badge
      expect(screen.getByText('2')).toBeInTheDocument();
      expect(screen.getByTitle('2個のエイリアス')).toBeInTheDocument();
    });

    it('should show device name beneath alias in expanded mode', () => {
      // Mock to return that device has alias
      vi.mocked(deviceIdHelper.deviceHasAlias).mockReturnValue({ hasAlias: true, aliasName: 'Living Light', deviceIdentifier: '192.168.1.100 0291:1' });
      
      const aliasedDevice = { ...mockDevice };
      const devices = { [`${mockDevice.ip} ${mockDevice.eoj}`]: aliasedDevice };
      const aliases = { 'Living Light': `${mockDevice.ip} ${mockDevice.eoj}` };

      render(
        <DeviceCard
          device={aliasedDevice}
          isExpanded={true}
          onToggleExpansion={vi.fn()}
          onPropertyChange={mockOnPropertyChange}
          onUpdateProperties={mockOnUpdateProperties}
          propertyDescriptions={mockPropertyDescriptions}
          getDeviceClassCode={mockGetDeviceClassCode}
          devices={devices}
          aliases={aliases}
        />
      );

      expect(screen.getByText(/Device:.*Single Function Lighting/)).toBeInTheDocument();
    });
  });
});