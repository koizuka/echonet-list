import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { PropertyDisplay } from './PropertyDisplay';
import type { PropertyDescriptor } from '@/hooks/types';

// Mock the translateLocationId function
vi.mock('@/libs/locationHelper', () => ({
  translateLocationId: (id: string) => {
    const translations: Record<string, string> = {
      'living': 'リビング',
      'dining': 'ダイニング',
    };
    return translations[id] || id;
  }
}));

describe('PropertyDisplay', () => {
  it('should display formatted property value', () => {
    const descriptor: PropertyDescriptor = {
      description: 'Temperature',
      numberDesc: {
        min: 0,
        max: 100,
        unit: '°C',
        offset: 0,
        edtLen: 1
      }
    };

    render(
      <PropertyDisplay
        currentValue={{ number: 25 }}
        descriptor={descriptor}
        epc="B0"
      />
    );

    expect(screen.getByText('25°C')).toBeInTheDocument();
  });

  it('should display alias value with translation for location', () => {
    const descriptor: PropertyDescriptor = {
      description: 'Installation location',
      aliases: {
        'living': '08',
        'dining': '10'
      }
    };

    render(
      <PropertyDisplay
        currentValue={{ string: 'living' }}
        descriptor={descriptor}
        epc="81"
      />
    );

    expect(screen.getByText('リビング')).toBeInTheDocument();
  });

  it('should display raw data when no descriptor', () => {
    render(
      <PropertyDisplay
        currentValue={{ EDT: btoa('test') }}
        descriptor={undefined}
        epc="FF"
      />
    );

    expect(screen.getByText('Raw data')).toBeInTheDocument();
  });

  it('should integrate HexViewer when applicable', () => {
    render(
      <PropertyDisplay
        currentValue={{ EDT: btoa('test') }}
        descriptor={undefined}
        epc="FF"
      />
    );

    expect(screen.getByTitle('Show hex data')).toBeInTheDocument();
  });

  it('should display property map with expand button', () => {
    const mockDevice = {
      ip: '192.168.1.100',
      eoj: '0291:1',
      name: 'Test Device',
      id: undefined,
      lastSeen: new Date().toISOString(),
      properties: {}
    };

    render(
      <PropertyDisplay
        currentValue={{ EDT: btoa(String.fromCharCode(0x02, 0x80, 0xB0)) }}
        descriptor={{ description: 'Set Property Map' }}
        epc="9E"
        propertyDescriptions={{
          '': {
            classCode: '',
            properties: {
              '80': { description: 'Operation Status' },
              'B0': { description: 'Illuminance Level' }
            }
          }
        }}
        device={mockDevice}
      />
    );

    expect(screen.getByText(/Raw data.*\(2\)/)).toBeInTheDocument();
    expect(screen.getByTitle('Show property details')).toBeInTheDocument();
  });
});