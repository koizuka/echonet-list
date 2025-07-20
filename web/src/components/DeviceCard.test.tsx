import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
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

// Mock languageHelper to always return 'en' for consistent test behavior
vi.mock('@/libs/languageHelper', () => ({
  isJapanese: vi.fn(() => false),
  getCurrentLocale: vi.fn(() => 'en')
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

      // Should show primary properties in compact mode
      expect(screen.getByText('Operation Status:')).toBeInTheDocument();
      expect(screen.getByText('Illuminance Level:')).toBeInTheDocument();
      
      // Should NOT show secondary properties or "Other Properties" section in compact mode
      expect(screen.queryByText('Other Properties')).not.toBeInTheDocument();
      expect(screen.queryByText('Installation Location:')).not.toBeInTheDocument();
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

      // Check for compact text size by looking for specific property
      const operationStatus = screen.getByText('Operation Status:');
      const container = operationStatus.closest('.text-xs');
      expect(container).toBeInTheDocument();
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

    it('should show device IP and EOJ in compact mode when no alias exists', () => {
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

      // IP and EOJ should be visible when no alias exists
      expect(screen.getByText(/192\.168\.1\.100.*0291:1/)).toBeInTheDocument();
    });

    it('should not show device IP and EOJ in compact mode when alias exists', () => {
      // Mock to return that device has alias
      vi.mocked(deviceIdHelper.deviceHasAlias).mockReturnValue({ hasAlias: true, aliasName: 'Living Light', deviceIdentifier: '192.168.1.100 0291:1' });
      
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
          aliases={{ 'Living Light': `${mockDevice.ip} ${mockDevice.eoj}` }}
        />
      );

      // IP and EOJ should not be visible when alias exists
      expect(screen.queryByText(/192\.168\.1\.100.*0291:1/)).not.toBeInTheDocument();
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

      // Should show properties with their actual rendered names
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

      // Check for full mode spacing by looking for specific property
      expect(screen.getByText('Operation Status:')).toBeInTheDocument();
      // Check that spacing class exists somewhere in the document
      const spacingElements = document.querySelectorAll('.space-y-3');
      expect(spacingElements.length).toBeGreaterThan(0);
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
          isConnected={true}
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

  describe('WebSocket connection state', () => {
    it('should disable update button when disconnected', () => {
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
          isConnected={false}
        />
      );

      const updateButton = screen.getByTestId('update-properties-button');
      expect(updateButton).toBeDisabled();
    });

    it('should enable update button when connected', () => {
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
          isConnected={true}
        />
      );

      const updateButton = screen.getByTestId('update-properties-button');
      expect(updateButton).not.toBeDisabled();
    });

    it('should default to enabled when isConnected is not specified', () => {
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

      const updateButton = screen.getByTestId('update-properties-button');
      expect(updateButton).not.toBeDisabled();
    });
  });

  describe('Last seen timestamp', () => {
    it('should not show last seen in compact mode', () => {
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

      // Should not show last seen text in compact mode
      expect(screen.queryByText(/Last seen:/)).not.toBeInTheDocument();
    });

    it('should show last seen in expanded mode', () => {
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

      // Should show last seen text in expanded mode
      expect(screen.getByText(/Last seen:/)).toBeInTheDocument();
    });

    it('should show last seen with normal color in expanded mode', () => {
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

      const lastSeenText = screen.getByText(/Last seen:/);
      // Check for normal muted color class
      expect(lastSeenText).toHaveClass('text-muted-foreground');
    });
  });

  describe('device deletion', () => {
    it('should show delete button only for offline devices', () => {
      const offlineDevice = { ...mockDevice, isOffline: true };
      const onDeleteDevice = vi.fn();

      render(
        <DeviceCard
          device={offlineDevice}
          isExpanded={false}
          onToggleExpansion={vi.fn()}
          onPropertyChange={vi.fn()}
          propertyDescriptions={{}}
          getDeviceClassCode={() => '0130'}
          devices={{ [`${offlineDevice.ip} ${offlineDevice.eoj}`]: offlineDevice }}
          aliases={{}}
          onDeleteDevice={onDeleteDevice}
        />
      );

      const deleteButton = screen.getByTestId('delete-device-button');
      expect(deleteButton).toBeInTheDocument();
      expect(deleteButton).toHaveAttribute('title', 'Delete offline device');
    });

    it('should not show delete button for online devices', () => {
      const onlineDevice = { ...mockDevice, isOffline: false };

      render(
        <DeviceCard
          device={onlineDevice}
          isExpanded={false}
          onToggleExpansion={vi.fn()}
          onPropertyChange={vi.fn()}
          propertyDescriptions={{}}
          getDeviceClassCode={() => '0130'}
          devices={{ [`${onlineDevice.ip} ${onlineDevice.eoj}`]: onlineDevice }}
          aliases={{}}
          onDeleteDevice={vi.fn()}
        />
      );

      const deleteButton = screen.queryByTestId('delete-device-button');
      expect(deleteButton).not.toBeInTheDocument();
    });

    it('should show confirmation dialog when delete button is clicked', async () => {
      const offlineDevice = { ...mockDevice, isOffline: true };
      const onDeleteDevice = vi.fn();

      render(
        <DeviceCard
          device={offlineDevice}
          isExpanded={false}
          onToggleExpansion={vi.fn()}
          onPropertyChange={vi.fn()}
          propertyDescriptions={{}}
          getDeviceClassCode={() => '0130'}
          devices={{ [`${offlineDevice.ip} ${offlineDevice.eoj}`]: offlineDevice }}
          aliases={{}}
          onDeleteDevice={onDeleteDevice}
        />
      );

      // Click the delete button
      const deleteButton = screen.getByTestId('delete-device-button');
      await userEvent.click(deleteButton);

      // Check if confirmation dialog appears
      expect(screen.getByRole('alertdialog')).toBeInTheDocument();
      expect(screen.getByText('Delete Offline Device')).toBeInTheDocument();
      expect(screen.getByText(/Are you sure you want to delete/)).toBeInTheDocument();
      expect(screen.getByText('Cancel')).toBeInTheDocument();
      expect(screen.getByText('Delete Device')).toBeInTheDocument();
    });

    it('should call onDeleteDevice when confirmed in dialog', async () => {
      const offlineDevice = { ...mockDevice, isOffline: true };
      const onDeleteDevice = vi.fn();

      render(
        <DeviceCard
          device={offlineDevice}
          isExpanded={false}
          onToggleExpansion={vi.fn()}
          onPropertyChange={vi.fn()}
          propertyDescriptions={{}}
          getDeviceClassCode={() => '0130'}
          devices={{ [`${offlineDevice.ip} ${offlineDevice.eoj}`]: offlineDevice }}
          aliases={{}}
          onDeleteDevice={onDeleteDevice}
        />
      );

      // Click the delete button to open dialog
      const deleteButton = screen.getByTestId('delete-device-button');
      await userEvent.click(deleteButton);

      // Click the confirm button in dialog
      const confirmButton = screen.getByText('Delete Device');
      await userEvent.click(confirmButton);

      // Check if onDeleteDevice was called
      expect(onDeleteDevice).toHaveBeenCalledWith('192.168.1.100 0291:1');
    });

    it('should not call onDeleteDevice when cancelled in dialog', async () => {
      const offlineDevice = { ...mockDevice, isOffline: true };
      const onDeleteDevice = vi.fn();

      render(
        <DeviceCard
          device={offlineDevice}
          isExpanded={false}
          onToggleExpansion={vi.fn()}
          onPropertyChange={vi.fn()}
          propertyDescriptions={{}}
          getDeviceClassCode={() => '0130'}
          devices={{ [`${offlineDevice.ip} ${offlineDevice.eoj}`]: offlineDevice }}
          aliases={{}}
          onDeleteDevice={onDeleteDevice}
        />
      );

      // Click the delete button to open dialog
      const deleteButton = screen.getByTestId('delete-device-button');
      await userEvent.click(deleteButton);

      // Click the cancel button in dialog
      const cancelButton = screen.getByText('Cancel');
      await userEvent.click(cancelButton);

      // Check if onDeleteDevice was not called
      expect(onDeleteDevice).not.toHaveBeenCalled();
    });

    it('should disable delete button when device is being deleted', () => {
      const offlineDevice = { ...mockDevice, isOffline: true };

      render(
        <DeviceCard
          device={offlineDevice}
          isExpanded={false}
          onToggleExpansion={vi.fn()}
          onPropertyChange={vi.fn()}
          propertyDescriptions={{}}
          getDeviceClassCode={() => '0130'}
          devices={{ [`${offlineDevice.ip} ${offlineDevice.eoj}`]: offlineDevice }}
          aliases={{}}
          onDeleteDevice={vi.fn()}
          isDeletingDevice={true}
        />
      );

      const deleteButton = screen.getByTestId('delete-device-button');
      expect(deleteButton).toBeDisabled();
    });
  });
});