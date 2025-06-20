import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { PropertyEditor } from './PropertyEditor';
import type { Device, PropertyDescriptor } from '@/hooks/types';

// Mock ResizeObserver for tests
global.ResizeObserver = vi.fn(() => ({
  observe: vi.fn(),
  disconnect: vi.fn(),
  unobserve: vi.fn(),
}));

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

  describe('Properties with only on/off aliases (Switch UI)', () => {
    const operationStatusDescriptor: PropertyDescriptor = {
      description: 'Operation status',
      aliases: {
        'on': 'MA==',
        'off': 'MQ=='
      }
    };

    it('should render a switch for properties with only on/off aliases', () => {
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

    it('should not render switch for properties with more than two aliases', () => {
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

    it('should not render switch when property has no aliases', () => {
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

    it('should render switch for non-0x80 properties with only on/off aliases', () => {
      const fanDescriptor: PropertyDescriptor = {
        description: 'Fan power',
        aliases: {
          'on': 'MA==',
          'off': 'MQ=='
        }
      };

      // Device with property CF settable (example: fan power)
      const deviceWithCFSettable = {
        ...mockDevice,
        properties: {
          ...mockDevice.properties,
          'CF': { string: 'off' },
          '9E': { EDT: btoa(String.fromCharCode(0x02, 0x80, 0xCF)) } // Set Property Map with EPCs 0x80 and 0xCF
        }
      };

      render(
        <PropertyEditor
          device={deviceWithCFSettable}
          epc="CF"
          currentValue={{ string: 'off' }}
          descriptor={fanDescriptor}
          onPropertyChange={mockOnPropertyChange}
        />
      );

      const switchElement = screen.getByTestId('operation-status-switch-CF');
      expect(switchElement).toBeInTheDocument();
      expect(switchElement).toHaveAttribute('aria-checked', 'false');
    });

    it('should not render switch when aliases are not exactly on/off', () => {
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
    it('should render dropdown for alias properties with more than two options', () => {
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

  describe('Slider functionality for numeric properties', () => {
    const temperatureDescriptor: PropertyDescriptor = {
      description: 'Temperature setting',
      numberDesc: {
        min: 16,
        max: 30,
        offset: 0,
        unit: '°C',
        edtLen: 1
      }
    };

    const deviceWithTemperatureSettable = {
      ...mockDevice,
      properties: {
        ...mockDevice.properties,
        'B3': { number: 22 },
        '9E': { EDT: btoa(String.fromCharCode(0x02, 0x80, 0xB3)) } // Set Property Map with EPCs 0x80 and 0xB3
      }
    };

    it('should render slider for numeric properties in edit mode', () => {
      render(
        <PropertyEditor
          device={deviceWithTemperatureSettable}
          epc="B3"
          currentValue={{ number: 22 }}
          descriptor={temperatureDescriptor}
          onPropertyChange={mockOnPropertyChange}
        />
      );

      // Click edit button to enter edit mode
      const editButton = screen.getByTestId('edit-button-B3');
      fireEvent.click(editButton);

      // Check that slider is present
      const slider = screen.getByTestId('slider-B3');
      expect(slider).toBeInTheDocument();

      // Check min/max labels
      expect(screen.getByText('16')).toBeInTheDocument();
      expect(screen.getByText('30')).toBeInTheDocument();
      
      // Check unit display
      expect(screen.getByText('22°C')).toBeInTheDocument();
    });

    it('should sync input and slider values', () => {
      render(
        <PropertyEditor
          device={deviceWithTemperatureSettable}
          epc="B3"
          currentValue={{ number: 22 }}
          descriptor={temperatureDescriptor}
          onPropertyChange={mockOnPropertyChange}
        />
      );

      // Enter edit mode
      const editButton = screen.getByTestId('edit-button-B3');
      fireEvent.click(editButton);

      const input = screen.getByTestId('edit-input-B3');
      
      // Change input value
      fireEvent.change(input, { target: { value: '25' } });
      expect(input).toHaveValue('25');
      
      // Check that unit display updated
      expect(screen.getByText('25°C')).toBeInTheDocument();
    });

    it('should handle alias+number property correctly', () => {
      const mixedDescriptor: PropertyDescriptor = {
        description: 'Mixed property',
        aliases: {
          'auto': 'QVU=',
          'manual': 'TUFOVA=='
        },
        numberDesc: {
          min: 0,
          max: 100,
          offset: 0,
          unit: '%',
          edtLen: 1
        }
      };

      const deviceWithMixedProperty = {
        ...mockDevice,
        properties: {
          ...mockDevice.properties,
          'CF': { string: 'auto' }, // Currently showing alias
          '9E': { EDT: btoa(String.fromCharCode(0x02, 0x80, 0xCF)) }
        }
      };

      render(
        <PropertyEditor
          device={deviceWithMixedProperty}
          epc="CF"
          currentValue={{ string: 'auto' }}
          descriptor={mixedDescriptor}
          onPropertyChange={mockOnPropertyChange}
        />
      );

      // Should show dropdown, not switch (because more than 2 aliases)
      expect(screen.getByTestId('alias-select-trigger-CF')).toBeInTheDocument();

      // Click edit button to enter edit mode
      const editButton = screen.getByTestId('edit-button-CF');
      fireEvent.click(editButton);

      // Should show slider for number editing
      const slider = screen.getByTestId('slider-CF');
      expect(slider).toBeInTheDocument();
      
      // Input should be empty (no current number value)
      const input = screen.getByTestId('edit-input-CF');
      expect(input).toHaveValue('');
      
      // Should show min value on slider (0%)
      expect(screen.getByText('0%')).toBeInTheDocument();
    });
  });
});