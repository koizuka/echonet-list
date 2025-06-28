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
});