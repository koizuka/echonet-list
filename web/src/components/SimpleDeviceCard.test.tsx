import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SimpleDeviceCard } from './SimpleDeviceCard';
import type { Device, DeviceAlias, PropertyDescriptionData } from '@/hooks/types';
import * as deviceIdHelper from '@/libs/deviceIdHelper';
import * as propertyHelper from '@/libs/propertyHelper';

// Mock deviceIdHelper functions
vi.mock('@/libs/deviceIdHelper', () => ({
  getDeviceAliases: vi.fn(() => ({ 
    aliases: [], 
    deviceIdentifier: '192.168.1.100 0291:1' 
  }))
}));

// Mock propertyHelper functions
vi.mock('@/libs/propertyHelper', () => ({
  formatPropertyValue: vi.fn((value) => value?.string || ''),
  getPropertyDescriptor: vi.fn(() => ({ name: 'Installation location' }))
}));

// Mock DeviceIcon component
vi.mock('@/components/DeviceIcon', () => ({
  DeviceIcon: ({ device, classCode }: { device: Device; classCode: string }) => (
    <div data-testid="device-icon" data-device={device.eoj} data-class-code={classCode}>
      🏠
    </div>
  )
}));

describe('SimpleDeviceCard', () => {
  const mockDevice: Device = {
    ip: '192.168.1.100',
    eoj: '0291:1',
    name: 'Single Function Lighting',
    id: undefined,
    lastSeen: new Date().toISOString(),
    properties: {
      '80': { string: 'on' },
      '81': { string: 'living' }
    }
  };

  const mockDevices: Record<string, Device> = {
    '192.168.1.100 0291:1': mockDevice
  };

  const mockAliases: DeviceAlias = {};

  const mockPropertyDescriptions: Record<string, PropertyDescriptionData> = {
    '': {
      classCode: '',
      properties: {
        '80': { description: 'Operation Status' },
        '81': { description: 'Installation Location' }
      }
    },
    '0291': {
      classCode: '0291',
      properties: {
        'B0': { description: 'Illuminance Level' }
      }
    }
  };

  const defaultProps = {
    deviceKey: '192.168.1.100 0291:1',
    device: mockDevice,
    allDevices: mockDevices,
    aliases: mockAliases,
    propertyDescriptions: mockPropertyDescriptions,
    getDeviceClassCode: vi.fn(() => '0291')
  };

  beforeEach(() => {
    vi.clearAllMocks();
    // Reset mocks to default behavior
    vi.mocked(deviceIdHelper.getDeviceAliases).mockReturnValue({
      aliases: [],
      deviceIdentifier: '192.168.1.100 0291:1'
    });
    vi.mocked(propertyHelper.formatPropertyValue).mockImplementation((value) => value?.string || '');
    vi.mocked(propertyHelper.getPropertyDescriptor).mockReturnValue({ description: 'Installation location' });
  });

  describe('Basic rendering', () => {
    it('should render device card with basic information', () => {
      render(<SimpleDeviceCard {...defaultProps} />);
      
      expect(screen.getByTestId('device-card-192.168.1.100-0291:1')).toBeInTheDocument();
      expect(screen.getByText('Single Function Lighting')).toBeInTheDocument();
    });

    it('should render DeviceIcon with correct props', () => {
      render(<SimpleDeviceCard {...defaultProps} />);
      
      const deviceIcon = screen.getByTestId('device-icon');
      expect(deviceIcon).toBeInTheDocument();
      expect(deviceIcon).toHaveAttribute('data-device', '0291:1');
      expect(deviceIcon).toHaveAttribute('data-class-code', '0291');
    });

    it('should render device with IP and EOJ when no alias', () => {
      render(<SimpleDeviceCard {...defaultProps} />);
      
      expect(screen.getByText('192.168.1.100 0291:1')).toBeInTheDocument();
    });

    it('should display alias when available', () => {
      const mockGetDeviceAliases = vi.mocked(deviceIdHelper.getDeviceAliases);
      mockGetDeviceAliases.mockReturnValue({
        aliases: ['Living Room Light'],
        deviceIdentifier: '192.168.1.100 0291:1'
      });

      render(<SimpleDeviceCard {...defaultProps} />);
      
      expect(screen.getByText('Living Room Light')).toBeInTheDocument();
    });
  });

  describe('Null safety and error handling', () => {
    it('should return null when device is null', () => {
      const { container } = render(
        <SimpleDeviceCard {...defaultProps} device={null as any} />
      );
      
      expect(container.firstChild).toBeNull();
    });

    it('should return null when getDeviceClassCode is null', () => {
      const { container } = render(
        <SimpleDeviceCard {...defaultProps} getDeviceClassCode={null as any} />
      );
      
      expect(container.firstChild).toBeNull();
    });

    it('should handle missing properties gracefully', () => {
      const deviceWithoutProperties = {
        ...mockDevice,
        properties: undefined as any
      };

      render(<SimpleDeviceCard {...defaultProps} device={deviceWithoutProperties} />);
      
      expect(screen.getByTestId('device-card-192.168.1.100-0291:1')).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('should have correct aria-label for online device', () => {
      render(<SimpleDeviceCard {...defaultProps} />);
      
      const card = screen.getByTestId('device-card-192.168.1.100-0291:1');
      expect(card).toHaveAttribute('aria-label', 'Single Function Lighting');
    });

    it('should have correct aria-label for offline device', () => {
      const offlineDevice = { ...mockDevice, isOffline: true };
      
      render(<SimpleDeviceCard {...defaultProps} device={offlineDevice} />);
      
      const card = screen.getByTestId('device-card-192.168.1.100-0291:1');
      expect(card).toHaveAttribute('aria-label', 'Single Function Lighting (オフライン)');
    });

    it('should have correct aria-label for device with alias', () => {
      const mockGetDeviceAliases = vi.mocked(deviceIdHelper.getDeviceAliases);
      mockGetDeviceAliases.mockReturnValue({
        aliases: ['Living Room Light'],
        deviceIdentifier: '192.168.1.100 0291:1'
      });

      render(<SimpleDeviceCard {...defaultProps} />);
      
      const card = screen.getByTestId('device-card-192.168.1.100-0291:1');
      expect(card).toHaveAttribute('aria-label', 'Living Room Light');
    });

    it('should have correct aria-label for offline device with alias', () => {
      const mockGetDeviceAliases = vi.mocked(deviceIdHelper.getDeviceAliases);
      mockGetDeviceAliases.mockReturnValue({
        aliases: ['Living Room Light'],
        deviceIdentifier: '192.168.1.100 0291:1'
      });
      const offlineDevice = { ...mockDevice, isOffline: true };

      render(<SimpleDeviceCard {...defaultProps} device={offlineDevice} />);
      
      const card = screen.getByTestId('device-card-192.168.1.100-0291:1');
      expect(card).toHaveAttribute('aria-label', 'Living Room Light (オフライン)');
    });
  });

  describe('Offline styling', () => {
    it('should apply offline styling when device is offline', () => {
      const offlineDevice = { ...mockDevice, isOffline: true };
      
      render(<SimpleDeviceCard {...defaultProps} device={offlineDevice} />);
      
      const card = screen.getByTestId('device-card-192.168.1.100-0291:1');
      expect(card).toHaveClass('after:absolute');
      expect(card).toHaveClass('after:inset-0');
      expect(card).toHaveClass('after:bg-background/60');
      expect(card).toHaveClass('after:pointer-events-none');
      expect(card).toHaveClass('after:rounded-lg');
    });

    it('should not apply offline styling when device is online', () => {
      render(<SimpleDeviceCard {...defaultProps} />);
      
      const card = screen.getByTestId('device-card-192.168.1.100-0291:1');
      expect(card).not.toHaveClass('after:absolute');
    });
  });

  describe('Draggable functionality', () => {
    it('should be draggable when isDraggable is true', () => {
      const onDragStart = vi.fn();
      const onDragEnd = vi.fn();

      render(
        <SimpleDeviceCard 
          {...defaultProps} 
          isDraggable={true}
          onDragStart={onDragStart}
          onDragEnd={onDragEnd}
        />
      );
      
      const card = screen.getByTestId('device-card-192.168.1.100-0291:1');
      expect(card).toHaveAttribute('draggable', 'true');
      expect(screen.getByText('Single Function Lighting')).toBeInTheDocument();
    });

    it('should not be draggable when isLoading is true', () => {
      render(
        <SimpleDeviceCard 
          {...defaultProps} 
          isDraggable={true}
          isLoading={true}
        />
      );
      
      const card = screen.getByTestId('device-card-192.168.1.100-0291:1');
      expect(card).toHaveAttribute('draggable', 'false');
    });

    it('should apply dragging opacity when isDragging is true', () => {
      render(
        <SimpleDeviceCard 
          {...defaultProps} 
          isDraggable={true}
          isDragging={true}
        />
      );
      
      const card = screen.getByTestId('device-card-192.168.1.100-0291:1');
      expect(card).toHaveClass('opacity-50');
    });

    it('should call onDragStart with correct parameters', () => {
      const onDragStart = vi.fn();

      render(
        <SimpleDeviceCard 
          {...defaultProps} 
          isDraggable={true}
          onDragStart={onDragStart}
        />
      );
      
      const card = screen.getByTestId('device-card-192.168.1.100-0291:1');
      fireEvent.dragStart(card);
      
      expect(onDragStart).toHaveBeenCalledWith(
        expect.any(Object),
        '192.168.1.100 0291:1'
      );
    });
  });

  describe('Action button', () => {
    it('should render add action button', () => {
      const onClick = vi.fn();

      render(
        <SimpleDeviceCard 
          {...defaultProps} 
          actionButton={{
            type: 'add',
            onClick,
            title: 'Add device'
          }}
        />
      );
      
      const button = screen.getByTestId('add-device-192.168.1.100-0291:1');
      expect(button).toBeInTheDocument();
      expect(button).toHaveAttribute('title', 'Add device');
    });

    it('should render remove action button', () => {
      const onClick = vi.fn();

      render(
        <SimpleDeviceCard 
          {...defaultProps} 
          actionButton={{
            type: 'remove',
            onClick,
            title: 'Remove device'
          }}
        />
      );
      
      const button = screen.getByTestId('remove-device-192.168.1.100-0291:1');
      expect(button).toBeInTheDocument();
    });

    it('should render custom action button with custom icon', () => {
      const onClick = vi.fn();
      const customIcon = <div data-testid="custom-icon">⚙️</div>;

      render(
        <SimpleDeviceCard 
          {...defaultProps} 
          actionButton={{
            type: 'custom',
            onClick,
            icon: customIcon,
            title: 'Custom action'
          }}
        />
      );
      
      const button = screen.getByTestId('custom-device-192.168.1.100-0291:1');
      expect(button).toBeInTheDocument();
      expect(screen.getByTestId('custom-icon')).toBeInTheDocument();
    });

    it('should call onClick when action button is clicked', async () => {
      const user = userEvent.setup();
      const onClick = vi.fn();

      render(
        <SimpleDeviceCard 
          {...defaultProps} 
          actionButton={{
            type: 'add',
            onClick
          }}
        />
      );
      
      const button = screen.getByTestId('add-device-192.168.1.100-0291:1');
      await user.click(button);
      
      expect(onClick).toHaveBeenCalledOnce();
    });

    it('should disable action button when disabled prop is true', () => {
      const onClick = vi.fn();

      render(
        <SimpleDeviceCard 
          {...defaultProps} 
          actionButton={{
            type: 'add',
            onClick,
            disabled: true
          }}
        />
      );
      
      const button = screen.getByTestId('add-device-192.168.1.100-0291:1');
      expect(button).toBeDisabled();
    });

    it('should disable action button when isLoading is true', () => {
      const onClick = vi.fn();

      render(
        <SimpleDeviceCard 
          {...defaultProps} 
          isLoading={true}
          actionButton={{
            type: 'add',
            onClick
          }}
        />
      );
      
      const button = screen.getByTestId('add-device-192.168.1.100-0291:1');
      expect(button).toBeDisabled();
    });
  });

  describe('Multiple aliases display', () => {
    it('should show alias count badge when multiple aliases exist', () => {
      const mockGetDeviceAliases = vi.mocked(deviceIdHelper.getDeviceAliases);
      mockGetDeviceAliases.mockReturnValue({
        aliases: ['Alias 1', 'Alias 2', 'Alias 3'],
        deviceIdentifier: '192.168.1.100 0291:1'
      });

      render(<SimpleDeviceCard {...defaultProps} />);
      
      const badge = screen.getByText('+2');
      expect(badge).toBeInTheDocument();
    });

    it('should not show alias count badge when only one alias exists', () => {
      const mockGetDeviceAliases = vi.mocked(deviceIdHelper.getDeviceAliases);
      mockGetDeviceAliases.mockReturnValue({
        aliases: ['Single Alias'],
        deviceIdentifier: '192.168.1.100 0291:1'
      });

      render(<SimpleDeviceCard {...defaultProps} />);
      
      expect(screen.queryByText('+0')).not.toBeInTheDocument();
    });
  });

  describe('Compact mode', () => {
    it('should apply truncate class in compact mode', () => {
      render(<SimpleDeviceCard {...defaultProps} isCompact={true} />);
      
      const container = screen.getByTestId('device-card-192.168.1.100-0291:1');
      expect(container).toBeInTheDocument();
    });

    it('should not show secondary device info in compact mode with alias', () => {
      const mockGetDeviceAliases = vi.mocked(deviceIdHelper.getDeviceAliases);
      mockGetDeviceAliases.mockReturnValue({
        aliases: ['Living Room Light'],
        deviceIdentifier: '192.168.1.100 0291:1'
      });

      render(<SimpleDeviceCard {...defaultProps} isCompact={true} />);
      
      expect(screen.getByText('Living Room Light')).toBeInTheDocument();
      expect(screen.queryByText('Single Function Lighting')).not.toBeInTheDocument();
    });
  });

  describe('Installation location display', () => {
    it('should display installation location when available', () => {
      const mockFormatPropertyValue = vi.mocked(propertyHelper.formatPropertyValue);
      mockFormatPropertyValue.mockReturnValue('Living Room');

      render(<SimpleDeviceCard {...defaultProps} />);
      
      expect(screen.getByText(/設置場所: Living Room/)).toBeInTheDocument();
    });

    it('should not display installation location when not available', () => {
      const deviceWithoutLocation = {
        ...mockDevice,
        properties: { '80': { string: 'on' } }
      };

      render(<SimpleDeviceCard {...defaultProps} device={deviceWithoutLocation} />);
      
      expect(screen.queryByText(/設置場所:/)).not.toBeInTheDocument();
    });
  });
});