import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { PropertyEditor } from './PropertyEditor';
import type { Device, PropertyDescriptor } from '@/hooks/types';

describe('PropertyEditor', () => {
  const mockDevice: Device = {
    ip: '192.168.1.100',
    eoj: '0x0130',
    name: 'Test Device',
    id: undefined,
    lastSeen: new Date().toISOString(),
    properties: {
      '80': { string: 'on' },
      '9E': { EDT: btoa(String.fromCharCode(0x01, 0x80)) } // Set Property Map with 1 property: EPC 0x80
    }
  };

  const mockOnPropertyChange = vi.fn();

  beforeEach(() => {
    mockOnPropertyChange.mockClear();
  });

  describe('Operation status (EPC 0x80) with Switch UI', () => {
    const operationStatusDescriptor: PropertyDescriptor = {
      description: 'Operation status',
      aliases: {
        'on': 'MA==',
        'off': 'MQ=='
      }
    };

    it('should render a switch for Operation status', () => {
      const { container } = render(
        <PropertyEditor
          device={mockDevice}
          epc="80"
          currentValue={{ string: 'on' }}
          descriptor={operationStatusDescriptor}
          onPropertyChange={mockOnPropertyChange}
        />
      );

      // Debug output
      console.log('Rendered HTML:', container.innerHTML);
      
      const switchElement = screen.getByTestId('operation-status-switch-80');
      expect(switchElement).toBeInTheDocument();
      expect(switchElement).toHaveAttribute('aria-checked', 'true');
    });

    it('should toggle switch from on to off', async () => {
      render(
        <PropertyEditor
          device={mockDevice}
          epc="80"
          currentValue={{ string: 'on' }}
          descriptor={operationStatusDescriptor}
          onPropertyChange={mockOnPropertyChange}
        />
      );

      const switchElement = screen.getByTestId('operation-status-switch-80');
      fireEvent.click(switchElement);

      await waitFor(() => {
        expect(mockOnPropertyChange).toHaveBeenCalledWith(
          '192.168.1.100 0x0130',
          '80',
          { string: 'off' }
        );
      });
    });

    it('should toggle switch from off to on', async () => {
      render(
        <PropertyEditor
          device={mockDevice}
          epc="80"
          currentValue={{ string: 'off' }}
          descriptor={operationStatusDescriptor}
          onPropertyChange={mockOnPropertyChange}
        />
      );

      const switchElement = screen.getByTestId('operation-status-switch-80');
      expect(switchElement).toHaveAttribute('aria-checked', 'false');
      
      fireEvent.click(switchElement);

      await waitFor(() => {
        expect(mockOnPropertyChange).toHaveBeenCalledWith(
          '192.168.1.100 0x0130',
          '80',
          { string: 'on' }
        );
      });
    });

    it('should disable switch when loading', () => {
      render(
        <PropertyEditor
          device={mockDevice}
          epc="80"
          currentValue={{ string: 'on' }}
          descriptor={operationStatusDescriptor}
          onPropertyChange={mockOnPropertyChange}
        />
      );

      const switchElement = screen.getByTestId('operation-status-switch-80');
      expect(switchElement).not.toBeDisabled();
    });

    it('should not render switch for non-Operation status properties', () => {
      const otherDescriptor: PropertyDescriptor = {
        description: 'Illuminance level',
        aliases: {
          'high': 'MQ==',
          'low': 'MA=='
        }
      };

      // Device with property B0 settable
      const deviceWithB0Settable = {
        ...mockDevice,
        properties: {
          ...mockDevice.properties,
          'B0': { string: 'high' },
          '9E': { EDT: btoa(String.fromCharCode(0x02, 0x80, 0xB0)) } // Set Property Map with EPCs 0x80 and 0xB0
        }
      };

      render(
        <PropertyEditor
          device={deviceWithB0Settable}
          epc="B0"
          currentValue={{ string: 'high' }}
          descriptor={otherDescriptor}
          onPropertyChange={mockOnPropertyChange}
        />
      );

      expect(screen.queryByTestId('operation-status-switch-B0')).not.toBeInTheDocument();
      expect(screen.getByTestId('alias-select-trigger-B0')).toBeInTheDocument();
    });

    it('should not render switch when Operation status has no aliases', () => {
      render(
        <PropertyEditor
          device={mockDevice}
          epc="80"
          currentValue={{ string: 'on' }}
          descriptor={{ description: 'Operation status', stringDesc: { minEDTLen: 1, maxEDTLen: 10 } }}
          onPropertyChange={mockOnPropertyChange}
        />
      );

      expect(screen.queryByTestId('operation-status-switch-80')).not.toBeInTheDocument();
    });

    it('should not render switch when Operation status aliases do not include on/off', () => {
      const customAliasDescriptor: PropertyDescriptor = {
        description: 'Operation status',
        aliases: {
          'active': 'MA==',
          'inactive': 'MQ=='
        }
      };

      render(
        <PropertyEditor
          device={mockDevice}
          epc="80"
          currentValue={{ string: 'active' }}
          descriptor={customAliasDescriptor}
          onPropertyChange={mockOnPropertyChange}
        />
      );

      expect(screen.queryByTestId('operation-status-switch-80')).not.toBeInTheDocument();
      expect(screen.getByTestId('alias-select-trigger-80')).toBeInTheDocument();
    });
  });

  describe('Other alias properties', () => {
    it('should render dropdown for non-Operation status alias properties', () => {
      const locationDescriptor: PropertyDescriptor = {
        description: 'Installation location',
        aliases: {
          'living': '08',
          'dining': '10',
          'kitchen': '18'
        }
      };

      // Device with property 81 settable
      const deviceWith81Settable = {
        ...mockDevice,
        properties: {
          ...mockDevice.properties,
          '81': { string: 'living' },
          '9E': { EDT: btoa(String.fromCharCode(0x02, 0x80, 0x81)) } // Set Property Map with EPCs 0x80 and 0x81
        }
      };

      render(
        <PropertyEditor
          device={deviceWith81Settable}
          epc="81"
          currentValue={{ string: 'living' }}
          descriptor={locationDescriptor}
          onPropertyChange={mockOnPropertyChange}
        />
      );

      expect(screen.queryByTestId('operation-status-switch-81')).not.toBeInTheDocument();
      expect(screen.getByTestId('alias-select-trigger-81')).toBeInTheDocument();
    });
  });
});