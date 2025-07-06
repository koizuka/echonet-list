import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { PropertyEditor } from './PropertyEditor';
import type { Device, PropertyDescriptor, PropertyDescriptionData } from '@/hooks/types';

// Mock ResizeObserver for tests
global.ResizeObserver = vi.fn(() => ({
  observe: vi.fn(),
  disconnect: vi.fn(),
  unobserve: vi.fn(),
}));

// Mock languageHelper to always return 'en' for consistent test behavior
vi.mock('@/libs/languageHelper', () => ({
  isJapanese: vi.fn(() => false),
  getCurrentLocale: vi.fn(() => 'en')
}));

describe('PropertyEditor', () => {
  const mockDevice: Device = {
    ip: '192.168.1.100',
    eoj: '0130:1',
    name: 'Test Device',
    id: undefined,
    lastSeen: new Date().toISOString(),
    properties: {
      '80': { string: 'on' },
      '9E': { EDT: btoa(String.fromCharCode(0x01, 0x80)) } // Set Property Map with 1 property: EPC 0x80
    }
  };

  const mockPropertyDescriptions: Record<string, PropertyDescriptionData> = {
    '': {
      classCode: '',
      properties: {
        '80': { description: 'Operation Status' },
        '9E': { description: 'Set Property Map' },
      },
    },
    '0130': {
      classCode: '0130',
      properties: {
        'B0': { description: 'Illuminance Level' },
      },
    },
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
          propertyDescriptions={mockPropertyDescriptions}
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
          propertyDescriptions={mockPropertyDescriptions}
        />
      );

      const switchElement = screen.getByTestId('operation-status-switch-80');
      fireEvent.click(switchElement);

      await waitFor(() => {
        expect(mockOnPropertyChange).toHaveBeenCalledWith(
          '192.168.1.100 0130:1',
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
          propertyDescriptions={mockPropertyDescriptions}
        />
      );

      const switchElement = screen.getByTestId('operation-status-switch-80');
      expect(switchElement).toHaveAttribute('aria-checked', 'false');
      
      fireEvent.click(switchElement);

      await waitFor(() => {
        expect(mockOnPropertyChange).toHaveBeenCalledWith(
          '192.168.1.100 0130:1',
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
          propertyDescriptions={mockPropertyDescriptions}
        />
      );

      const switchElement = screen.getByTestId('operation-status-switch-80');
      expect(switchElement).not.toBeDisabled();
    });

    it('should show as read-only when WebSocket is disconnected', () => {
      render(
        <PropertyEditor
          device={mockDevice}
          epc="80"
          currentValue={{ string: 'on' }}
          descriptor={operationStatusDescriptor}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          isConnected={false}
        />
      );

      // Should not have switch control when disconnected
      expect(screen.queryByTestId('operation-status-switch-80')).not.toBeInTheDocument();
      // Should display the value as read-only
      expect(screen.getByText('on')).toBeInTheDocument();
    });

    it('should enable switch when WebSocket is connected', () => {
      render(
        <PropertyEditor
          device={mockDevice}
          epc="80"
          currentValue={{ string: 'on' }}
          descriptor={operationStatusDescriptor}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          isConnected={true}
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
          propertyDescriptions={mockPropertyDescriptions}
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
          propertyDescriptions={mockPropertyDescriptions}
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
          propertyDescriptions={mockPropertyDescriptions}
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
          propertyDescriptions={mockPropertyDescriptions}
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
          propertyDescriptions={mockPropertyDescriptions}
        />
      );

      expect(screen.queryByTestId('operation-status-switch-81')).not.toBeInTheDocument();
      expect(screen.getByTestId('alias-select-trigger-81')).toBeInTheDocument();
    });

    it('should show as read-only when WebSocket is disconnected', () => {
      const locationDescriptor: PropertyDescriptor = {
        description: 'Installation location',
        aliases: {
          'living': '08',
          'dining': '10',
          'kitchen': '18'
        },
        aliasTranslations: {
          'living': 'リビング',
          'dining': 'ダイニング',
          'kitchen': 'キッチン'
        }
      };

      const deviceWith81Settable = {
        ...mockDevice,
        properties: {
          ...mockDevice.properties,
          '81': { string: 'living' },
          '9E': { EDT: btoa(String.fromCharCode(0x02, 0x80, 0x81)) }
        }
      };

      render(
        <PropertyEditor
          device={deviceWith81Settable}
          epc="81"
          currentValue={{ string: 'living' }}
          descriptor={locationDescriptor}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          isConnected={false}
        />
      );

      // Should not have dropdown when disconnected
      expect(screen.queryByTestId('alias-select-trigger-81')).not.toBeInTheDocument();
      // Should display the alias value as read-only
      expect(screen.getByText('living')).toBeInTheDocument();
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
          propertyDescriptions={mockPropertyDescriptions}
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

    it('should show as read-only when WebSocket is disconnected', () => {
      render(
        <PropertyEditor
          device={deviceWithTemperatureSettable}
          epc="B3"
          currentValue={{ number: 22 }}
          descriptor={temperatureDescriptor}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          isConnected={false}
        />
      );

      // Should not have edit button when disconnected
      expect(screen.queryByTestId('edit-button-B3')).not.toBeInTheDocument();
      // Should display value as read-only
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
          propertyDescriptions={mockPropertyDescriptions}
        />
      );

      // Enter edit mode
      const editButton = screen.getByTestId('edit-button-B3');
      fireEvent.click(editButton);

      const input = screen.getByTestId('edit-input-B3');
      
      // Change input value
      fireEvent.change(input, { target: { value: '25' } });
      expect(input).toHaveValue(25);
      
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
          propertyDescriptions={mockPropertyDescriptions}
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
      expect(input).toHaveValue(null);
      
      // Should show min value on slider (0%)
      expect(screen.getByText('0%')).toBeInTheDocument();
    });
  });

  describe('WebSocket connection state handling', () => {
    it('should show as read-only when WebSocket is disconnected', () => {
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
          '9E': { EDT: btoa(String.fromCharCode(0x02, 0x80, 0xB3)) }
        }
      };

      render(
        <PropertyEditor
          device={deviceWithTemperatureSettable}
          epc="B3"
          currentValue={{ number: 22 }}
          descriptor={temperatureDescriptor}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
          isConnected={false}
        />
      );

      // Should not have edit button
      expect(screen.queryByTestId('edit-button-B3')).not.toBeInTheDocument();
      // Should show read-only value
      expect(screen.getByText('22°C')).toBeInTheDocument();
    });
  });

  describe('Property Map Display', () => {
    const deviceWithPropertyMap: Device = {
      ...mockDevice,
      properties: {
        ...mockDevice.properties,
        '9E': { EDT: btoa(String.fromCharCode(0x02, 0x80, 0xB0)) } // Set Property Map with 2 properties: 80, B0
      }
    };

    it('should render property map with expand button', () => {
      render(
        <PropertyEditor
          device={deviceWithPropertyMap}
          epc="9E"
          currentValue={{ EDT: btoa(String.fromCharCode(0x02, 0x80, 0xB0)) }}
          descriptor={{ description: 'Set Property Map' }}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
        />
      );

      // Should show property count
      expect(screen.getByText(/Raw data.*\(2\)/)).toBeInTheDocument();
      
      // Should show expand button (chevron right)
      const expandButton = screen.getByTitle('Show property details');
      expect(expandButton).toBeInTheDocument();
    });

    it('should expand and show property details when clicked', () => {
      render(
        <PropertyEditor
          device={deviceWithPropertyMap}
          epc="9E"
          currentValue={{ EDT: btoa(String.fromCharCode(0x02, 0x80, 0xB0)) }}
          descriptor={{ description: 'Set Property Map' }}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
        />
      );

      // Initially collapsed
      expect(screen.queryByText('80')).not.toBeInTheDocument();
      expect(screen.queryByText('Operation Status')).not.toBeInTheDocument();

      // Click to expand
      const expandButton = screen.getByTitle('Show property details');
      fireEvent.click(expandButton);

      // Should show property details
      expect(screen.getByText('80')).toBeInTheDocument();
      expect(screen.getByText('Operation Status')).toBeInTheDocument();
      expect(screen.getByText('B0')).toBeInTheDocument();
      expect(screen.getByText('Illuminance Level')).toBeInTheDocument();
    });

    it('should handle empty property map gracefully', () => {
      const deviceWithEmptyMap: Device = {
        ...mockDevice,
        properties: {
          ...mockDevice.properties,
          '9E': { EDT: btoa(String.fromCharCode(0x00)) } // Empty property map
        }
      };

      render(
        <PropertyEditor
          device={deviceWithEmptyMap}
          epc="9E"
          currentValue={{ EDT: btoa(String.fromCharCode(0x00)) }}
          descriptor={{ description: 'Set Property Map' }}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
        />
      );

      expect(screen.getByText(/Raw data.*\(0\)/)).toBeInTheDocument();
      
      // Click to expand
      const expandButton = screen.getByTitle('Show property details');
      fireEvent.click(expandButton);

      expect(screen.getByText('No properties in this map')).toBeInTheDocument();
    });

    it('should handle bitmap format for property maps with 16+ properties', () => {
      // Create a bitmap with properties at specific positions
      // Using Go formula: EPC = i + (j << 4) + 0x80
      // For i=0, j=0: EPC = 0 + 0 + 0x80 = 0x80
      // For i=1, j=1: EPC = 1 + 16 + 0x80 = 0x91
      const bitmapData = new Array(17).fill(0);
      bitmapData[0] = 16; // Property count >= 16 triggers bitmap format
      bitmapData[1] = 0x01; // Bit 0 set: EPC 0x80
      bitmapData[2] = 0x02; // Bit 1 set: EPC 0x91
      
      const deviceWithBitmapMap: Device = {
        ...mockDevice,
        properties: {
          ...mockDevice.properties,
          '9F': { EDT: btoa(String.fromCharCode(...bitmapData)) }
        }
      };

      render(
        <PropertyEditor
          device={deviceWithBitmapMap}
          epc="9F"
          currentValue={{ EDT: btoa(String.fromCharCode(...bitmapData)) }}
          descriptor={{ description: 'Get Property Map' }}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
        />
      );

      expect(screen.getByText(/Raw data.*\(16\)/)).toBeInTheDocument();
      
      // Click to expand
      const expandButton = screen.getByTitle('Show property details');
      fireEvent.click(expandButton);

      // Should show the specific EPCs from bitmap
      expect(screen.getByText('80')).toBeInTheDocument();
      expect(screen.getByText('Operation Status')).toBeInTheDocument();
      expect(screen.getByText('91')).toBeInTheDocument();
      expect(screen.getByText('EPC 91')).toBeInTheDocument(); // Unknown EPC fallback
    });

    it('should handle realistic bitmap format for air conditioner with many properties', () => {
      // Simulate a realistic air conditioner with 20+ properties that would require bitmap format
      // Common ECHONET Lite properties for air conditioner (Home Air Conditioner: 0x0130)
      const propertyEpcs = [
        0x80, // Operation status
        0x81, // Installation location  
        0x88, // Fault occurrence status
        0x8A, // Manufacturer code
        0x8B, // Business facility code
        0x8C, // Product code
        0x8D, // Production number
        0x8E, // Production date
        0x8F, // Power saving operation setting
        0x9D, // Status change announcement property map
        0x9E, // Set property map
        0x9F, // Get property map
        0xA0, // Operation mode setting
        0xA1, // Automatic temperature control setting
        0xA3, // Automatic swing setting
        0xA4, // Air flow rate setting
        0xAA, // Relative humidity in dehumidification mode
        0xB0, // Set temperature value
        0xB1, // Relative humidity setting value
        0xB3, // Indoor relative humidity
        0xBA, // Indoor temperature
        0xBB, // Outdoor temperature
      ];

      // Create bitmap data (17 bytes: 1 count + 16 bitmap bytes)
      const bitmapData = new Array(17).fill(0);
      bitmapData[0] = propertyEpcs.length; // Property count

      // Set bits in bitmap according to Go formula: EPC = i + (j << 4) + 0x80
      // To reverse: offset = EPC - 0x80
      // From formula: offset = i + (j << 4), where i = offset & 0x0F, j = (offset & 0xF0) >> 4
      propertyEpcs.forEach(epc => {
        const offset = epc - 0x80;
        const i = offset & 0x0F; // byte index (0-15) - lower 4 bits
        const j = (offset & 0xF0) >> 4; // bit index (0-7) - upper 4 bits
        if (i < 16 && j < 8) {
          bitmapData[i + 1] |= (1 << j);
        }
      });

      const deviceWithRealisticMap: Device = {
        ...mockDevice,
        eoj: '0130:1', // Home Air Conditioner
        properties: {
          ...mockDevice.properties,
          '9F': { EDT: btoa(String.fromCharCode(...bitmapData)) }
        }
      };

      // Add air conditioner specific property descriptions
      const airConditionerPropertyDescriptions = {
        ...mockPropertyDescriptions,
        '0130': {
          classCode: '0130',
          properties: {
            'A0': { description: 'Operation Mode Setting' },
            'A1': { description: 'Automatic Temperature Control Setting' },
            'A3': { description: 'Automatic Swing Setting' },
            'A4': { description: 'Air Flow Rate Setting' },
            'AA': { description: 'Relative Humidity in Dehumidification Mode' },
            'B0': { description: 'Set Temperature Value' },
            'B1': { description: 'Relative Humidity Setting Value' },
            'B3': { description: 'Indoor Relative Humidity' },
            'BA': { description: 'Indoor Temperature' },
            'BB': { description: 'Outdoor Temperature' },
          },
        },
      };

      render(
        <PropertyEditor
          device={deviceWithRealisticMap}
          epc="9F"
          currentValue={{ EDT: btoa(String.fromCharCode(...bitmapData)) }}
          descriptor={{ description: 'Get Property Map' }}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={airConditionerPropertyDescriptions}
        />
      );

      expect(screen.getByText(new RegExp(`Raw data.*\\(${propertyEpcs.length}\\)`))).toBeInTheDocument();
      
      // Click to expand
      const expandButton = screen.getByTitle('Show property details');
      fireEvent.click(expandButton);

      // Verify some key properties are displayed (should be sorted)
      expect(screen.getByText('80')).toBeInTheDocument();
      expect(screen.getByText('Operation Status')).toBeInTheDocument();
      expect(screen.getByText('A0')).toBeInTheDocument();
      expect(screen.getByText('Operation Mode Setting')).toBeInTheDocument();
      expect(screen.getByText('B0')).toBeInTheDocument();
      expect(screen.getByText('Set Temperature Value')).toBeInTheDocument();
      expect(screen.getByText('BA')).toBeInTheDocument();
      expect(screen.getByText('Indoor Temperature')).toBeInTheDocument();
    });

    it('should correctly sort properties in ascending EPC order', () => {
      // Create property map with EPCs in non-sorted order to verify sorting
      // Use direct list format (< 16 properties) for simpler testing
      const unsortedEpcs = [0xB0, 0x80, 0xA0, 0x81, 0x9F, 0x88]; // Mixed order
      
      // Use direct list format since we have < 16 properties
      const directListData = [unsortedEpcs.length, ...unsortedEpcs];

      const deviceWithUnsortedMap: Device = {
        ...mockDevice,
        properties: {
          ...mockDevice.properties,
          '9E': { EDT: btoa(String.fromCharCode(...directListData)) }
        }
      };

      render(
        <PropertyEditor
          device={deviceWithUnsortedMap}
          epc="9E"
          currentValue={{ EDT: btoa(String.fromCharCode(...directListData)) }}
          descriptor={{ description: 'Set Property Map' }}
          onPropertyChange={mockOnPropertyChange}
          propertyDescriptions={mockPropertyDescriptions}
        />
      );

      // Click to expand
      const expandButton = screen.getByTitle('Show property details');
      fireEvent.click(expandButton);

      // Get all EPC elements and verify they are in sorted order
      const epcElements = screen.getAllByText(/^[0-9A-F]{2}$/);
      const epcTexts = epcElements.map(el => el.textContent);
      const sortedEpcs = [...epcTexts].sort();
      
      expect(epcTexts).toEqual(sortedEpcs);
      
      // Verify specific order: 80, 81, 88, 9F, A0, B0
      expect(epcTexts).toEqual(['80', '81', '88', '9F', 'A0', 'B0']);
    });
  });
});