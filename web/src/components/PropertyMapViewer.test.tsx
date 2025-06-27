import { render, screen, fireEvent } from '@testing-library/react';
import { PropertyMapViewer } from './PropertyMapViewer';
import type { Device } from '@/hooks/types';

const mockPropertyDescriptions = {
  '': {
    classCode: '',
    properties: {
      '80': { description: 'Operation Status' },
      '81': { description: 'Installation Location' },
      '88': { description: 'Fault Occurrence Status' },
      '9D': { description: 'Status Change Announcement Property Map' },
      '9E': { description: 'Set Property Map' },
      '9F': { description: 'Get Property Map' },
    },
  },
  '0130': {
    classCode: '0130',
    properties: {
      'B0': { description: 'Illuminance Level' },
      'B1': { description: 'Light Color' },
    },
  },
};

const createMockDevice = (propertyMaps: Record<string, string>): Device => ({
  ip: '192.168.1.100',
  eoj: '0130:1',
  name: 'Test Light',
  id: '0130:01:test123',
  properties: {
    '80': { string: 'on', EDT: btoa('\x30') },
    '81': { string: 'living_room', EDT: btoa('\x01') },
    ...Object.fromEntries(
      Object.entries(propertyMaps).map(([epc, edtHex]) => [
        epc,
        { EDT: btoa(String.fromCharCode(...edtHex.match(/.{2}/g)!.map(h => parseInt(h, 16)))) }
      ])
    ),
  },
  lastSeen: '2024-01-01T00:00:00Z',
});

describe('PropertyMapViewer', () => {
  it('renders all three property maps when available', () => {
    const device = createMockDevice({
      '9D': '0280B0', // Status announcement: 2 properties (80, B0)
      '9E': '0380B0B1', // Set: 3 properties (80, B0, B1)
      '9F': '0480818BB0', // Get: 4 properties (81, 88, B0, B1)
    });

    render(
      <PropertyMapViewer
        device={device}
        propertyDescriptions={mockPropertyDescriptions}
      />
    );

    expect(screen.getByText('Status Change Announcement Property Map (2)')).toBeInTheDocument();
    expect(screen.getByText('Set Property Map (3)')).toBeInTheDocument();
    expect(screen.getByText('Get Property Map (4)')).toBeInTheDocument();
  });

  it('shows property details when expanded', () => {
    const device = createMockDevice({
      '9E': '0280B0', // Set: 2 properties (80, B0)
    });

    render(
      <PropertyMapViewer
        device={device}
        propertyDescriptions={mockPropertyDescriptions}
      />
    );

    // Initially collapsed
    expect(screen.queryByText('80: Operation Status')).not.toBeInTheDocument();

    // Click to expand
    fireEvent.click(screen.getByText('Set Property Map (2)'));

    // Now properties should be visible (as separate elements)
    expect(screen.getByText('80')).toBeInTheDocument();
    expect(screen.getByText('Operation Status')).toBeInTheDocument();
    expect(screen.getByText('B0')).toBeInTheDocument();
    expect(screen.getByText('Illuminance Level')).toBeInTheDocument();
  });

  it('handles malformed property map data gracefully', () => {
    const device = createMockDevice({
      '9E': '00', // Invalid: count is 0 but should have properties
    });

    render(
      <PropertyMapViewer
        device={device}
        propertyDescriptions={mockPropertyDescriptions}
      />
    );

    expect(screen.getByText('Set Property Map (0)')).toBeInTheDocument();
  });

  it('displays unknown EPC codes with fallback description', () => {
    const device = createMockDevice({
      '9E': '01FF', // Set: 1 property (FF - unknown EPC)
    });

    render(
      <PropertyMapViewer
        device={device}
        propertyDescriptions={mockPropertyDescriptions}
      />
    );

    fireEvent.click(screen.getByText('Set Property Map (1)'));
    expect(screen.getByText('FF')).toBeInTheDocument();
    expect(screen.getByText('EPC FF')).toBeInTheDocument();
  });

  it('handles Base64 decode errors gracefully', () => {
    const device: Device = {
      ip: '192.168.1.100',
      eoj: '0130:1',
      name: 'Test Light',
      id: '0130:01:test123',
      properties: {
        '9E': { EDT: 'invalid-base64!' }, // Invalid Base64
      },
      lastSeen: '2024-01-01T00:00:00Z',
    };

    render(
      <PropertyMapViewer
        device={device}
        propertyDescriptions={mockPropertyDescriptions}
      />
    );

    // Should handle error gracefully and not render the property map
    expect(screen.queryByText(/Set Property Map/)).not.toBeInTheDocument();
  });

  it('does not render property maps that are not present', () => {
    const device = createMockDevice({
      '9E': '0180', // Only Set Property Map
    });

    render(
      <PropertyMapViewer
        device={device}
        propertyDescriptions={mockPropertyDescriptions}
      />
    );

    expect(screen.getByText('Set Property Map (1)')).toBeInTheDocument();
    expect(screen.queryByText(/Status Change Announcement Property Map/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Get Property Map/)).not.toBeInTheDocument();
  });

  it('toggles expansion state correctly', () => {
    const device = createMockDevice({
      '9E': '0180', // Set: 1 property (80)
    });

    render(
      <PropertyMapViewer
        device={device}
        propertyDescriptions={mockPropertyDescriptions}
      />
    );

    const toggleButton = screen.getByText('Set Property Map (1)');

    // Initially collapsed
    expect(screen.queryByText('80')).not.toBeInTheDocument();

    // Click to expand
    fireEvent.click(toggleButton);
    expect(screen.getByText('80')).toBeInTheDocument();
    expect(screen.getByText('Operation Status')).toBeInTheDocument();

    // Click to collapse
    fireEvent.click(toggleButton);
    expect(screen.queryByText('80')).not.toBeInTheDocument();
  });

  it('handles large property maps with bitmap format (16+ properties)', () => {
    // Create a bitmap format property map with 20 properties
    // Format: 1 byte count + 16 bytes bitmap
    // Properties: 0x80, 0x81, 0x82, ..., 0x93 (20 properties)
    const count = 20;
    const bitmap = new Array(16).fill(0);
    
    // Set bits for properties 0x80 to 0x93
    for (let epc = 0x80; epc <= 0x93; epc++) {
      const byteIndex = (epc & 0x0f);
      const bitIndex = (epc >> 4) - 8;
      bitmap[byteIndex] |= 1 << bitIndex;
    }
    
    const edtBytes = [count, ...bitmap];
    const edtHex = edtBytes.map(b => b.toString(16).padStart(2, '0')).join('').toUpperCase();
    
    const device = createMockDevice({
      '9F': edtHex, // Get Property Map with 20 properties
    });

    render(
      <PropertyMapViewer
        device={device}
        propertyDescriptions={mockPropertyDescriptions}
      />
    );

    expect(screen.getByText('Get Property Map (20)')).toBeInTheDocument();

    // Expand to see properties
    fireEvent.click(screen.getByText('Get Property Map (20)'));
    
    // Check some of the properties are displayed
    expect(screen.getByText('80')).toBeInTheDocument();
    expect(screen.getByText('81')).toBeInTheDocument();
    expect(screen.getByText('93')).toBeInTheDocument();
    
    // Check property descriptions
    expect(screen.getByText('Operation Status')).toBeInTheDocument();
    expect(screen.getByText('Installation Location')).toBeInTheDocument();
  });
});