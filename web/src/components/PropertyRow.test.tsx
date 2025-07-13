import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { PropertyRow } from './PropertyRow';
import type { Device } from '@/hooks/types';

// Mock child components
vi.mock('./PropertyEditor', () => ({
  PropertyEditor: ({ epc }: { epc: string }) => <div data-testid={`property-editor-${epc}`}>PropertyEditor</div>
}));

vi.mock('./PropertyDisplay', () => ({
  PropertyDisplay: ({ epc }: { epc: string }) => <div data-testid={`property-display-${epc}`}>PropertyDisplay</div>
}));

// Mock helper functions
vi.mock('@/libs/propertyHelper', () => ({
  getPropertyName: (epc: string) => `Property ${epc}`,
  getPropertyDescriptor: (epc: string) => ({ 
    description: `Property ${epc}`,
    // Make properties have edit capability based on EPC
    ...(epc === '80' ? { aliases: { 'on': 'MA==', 'off': 'MQ==' } } : {})
  }),
  isPropertySettable: (epc: string) => epc === '80', // Only EPC 80 is settable
}));

// Mock sensor property helper
vi.mock('@/libs/sensorPropertyHelper', () => ({
  isSensorProperty: (_classCode: string, epc: string) => ['BB', 'BA', 'BE'].includes(epc),
  getSensorIcon: (_classCode: string, epc: string) => {
    const iconMap: Record<string, any> = {
      'BB': ({ className }: { className?: string }) => 
        <svg data-testid="thermometer-icon" className={className}>Thermometer</svg>,
      'BA': ({ className }: { className?: string }) => 
        <svg data-testid="droplets-icon" className={className}>Droplets</svg>,
      'BE': ({ className }: { className?: string }) => 
        <svg data-testid="cloudsun-icon" className={className}>CloudSun</svg>
    };
    return iconMap[epc];
  },
  getSensorIconColor: (_classCode: string, _epc: string, _value: any) => 'text-muted-foreground'
}));

describe('PropertyRow', () => {
  const mockDevice: Device = {
    ip: '192.168.1.100',
    eoj: '0130:1',
    name: 'Test Device',
    id: undefined,
    lastSeen: new Date().toISOString(),
    properties: {
      '80': { string: 'on' },
      '9E': { EDT: btoa(String.fromCharCode(0x01, 0x80)) }
    }
  };

  const mockOnPropertyChange = vi.fn();
  const mockPropertyDescriptions = {
    '': {
      classCode: '',
      properties: {
        '80': { description: 'Operation Status' }
      }
    }
  };

  it('should render property name and value in compact mode', () => {
    render(
      <PropertyRow
        device={mockDevice}
        epc="80"
        value={{ string: 'on' }}
        isCompact={true}
        onPropertyChange={mockOnPropertyChange}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
      />
    );

    expect(screen.getByText('Property 80:')).toBeInTheDocument();
    expect(screen.getByTestId('property-editor-80')).toBeInTheDocument();
  });

  it('should render property name and editor in full mode', () => {
    render(
      <PropertyRow
        device={mockDevice}
        epc="80"
        value={{ string: 'on' }}
        isCompact={false}
        onPropertyChange={mockOnPropertyChange}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
      />
    );

    expect(screen.getByText('Property 80:')).toBeInTheDocument();
    expect(screen.getByTestId('property-editor-80')).toBeInTheDocument();
  });

  it('should use PropertyDisplay for non-settable properties in compact mode', () => {
    render(
      <PropertyRow
        device={mockDevice}
        epc="90"
        value={{ string: 'test' }}
        isCompact={true}
        onPropertyChange={mockOnPropertyChange}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
      />
    );

    expect(screen.getByTestId('property-display-90')).toBeInTheDocument();
    expect(screen.queryByTestId('property-editor-90')).not.toBeInTheDocument();
  });

  it('should apply different styles for compact mode', () => {
    const { container } = render(
      <PropertyRow
        device={mockDevice}
        epc="80"
        value={{ string: 'on' }}
        isCompact={true}
        onPropertyChange={mockOnPropertyChange}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
      />
    );

    const wrapper = container.firstChild;
    expect(wrapper).toHaveClass('text-xs');
  });

  it('should apply different styles for full mode', () => {
    const { container } = render(
      <PropertyRow
        device={mockDevice}
        epc="80"
        value={{ string: 'on' }}
        isCompact={false}
        onPropertyChange={mockOnPropertyChange}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
      />
    );

    const wrapper = container.firstChild;
    expect(wrapper).toHaveClass('space-y-1');
  });

  it('should render sensor properties with icon in compact mode', () => {
    const { container } = render(
      <PropertyRow
        device={mockDevice}
        epc="BB" // Temperature sensor
        value={{ number: 25 }}
        isCompact={true}
        onPropertyChange={mockOnPropertyChange}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
      />
    );

    // Should use inline-flex layout for sensor properties
    const wrapper = container.firstChild;
    expect(wrapper).toHaveClass('inline-flex');
    expect(wrapper).toHaveClass('items-center');
    
    // Should not show property name text for sensor properties
    expect(screen.queryByText('Property BB:')).not.toBeInTheDocument();
  });

  it('should show property name as tooltip on sensor component hover', () => {
    const { container } = render(
      <PropertyRow
        device={mockDevice}
        epc="BB" // Temperature sensor
        value={{ number: 25 }}
        isCompact={true}
        onPropertyChange={mockOnPropertyChange}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
      />
    );

    // Find the entire sensor component wrapper and check it has the title attribute
    const sensorWrapper = container.firstChild;
    expect(sensorWrapper).toHaveAttribute('title', 'Property BB');
  });

  it('should render non-sensor properties with traditional layout in compact mode', () => {
    render(
      <PropertyRow
        device={mockDevice}
        epc="80" // Operation status (non-sensor)
        value={{ string: 'on' }}
        isCompact={true}
        onPropertyChange={mockOnPropertyChange}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
      />
    );

    // Should show property name for non-sensor properties
    expect(screen.getByText('Property 80:')).toBeInTheDocument();
  });

  it('should render sensor properties normally in expanded mode', () => {
    render(
      <PropertyRow
        device={mockDevice}
        epc="BB" // Temperature sensor
        value={{ number: 25 }}
        isCompact={false}
        onPropertyChange={mockOnPropertyChange}
        propertyDescriptions={mockPropertyDescriptions}
        classCode="0130"
      />
    );

    // In expanded mode, sensor properties should show property name
    expect(screen.getByText('Property BB:')).toBeInTheDocument();
  });
});