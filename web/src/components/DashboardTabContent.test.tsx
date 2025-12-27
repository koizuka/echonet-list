import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { DashboardTabContent } from './DashboardTabContent';
import type { Device, PropertyDescriptionData } from '@/hooks/types';

// Mock ResizeObserver for tests
global.ResizeObserver = vi.fn(() => ({
  observe: vi.fn(),
  disconnect: vi.fn(),
  unobserve: vi.fn(),
}));

// Mock deviceIdHelper functions
vi.mock('@/libs/deviceIdHelper', () => ({
  deviceHasAlias: vi.fn(() => ({ hasAlias: false, aliasName: undefined, deviceIdentifier: 'test' })),
  getDeviceIdentifierForAlias: vi.fn(() => 'test'),
}));

// Mock languageHelper to always return 'en' for consistent test behavior
vi.mock('@/libs/languageHelper', () => ({
  isJapanese: vi.fn(() => false),
  getCurrentLocale: vi.fn(() => 'en')
}));

describe('DashboardTabContent', () => {
  const createDevice = (ip: string, eoj: string, location: string): Device => ({
    ip,
    eoj,
    name: `Device ${ip}-${eoj}`,
    id: undefined,
    lastSeen: new Date().toISOString(),
    properties: {
      '80': { string: 'on' },
      '81': { string: location },
      '9E': { EDT: btoa(String.fromCharCode(0x01, 0x80)) }
    }
  });

  const createNodeProfileDevice = (ip: string): Device => ({
    ip,
    eoj: '0EF0:1',
    name: `Node Profile ${ip}`,
    id: undefined,
    lastSeen: new Date().toISOString(),
    properties: {}
  });

  const mockPropertyDescriptions: Record<string, PropertyDescriptionData> = {
    '': {
      classCode: '',
      properties: {
        '80': { description: 'Operation Status' },
        '81': {
          description: 'Installation Location',
          aliases: { living: 'Living Room', kitchen: 'Kitchen', bedroom: 'Bedroom' }
        }
      }
    }
  };

  const mockOnPropertyChange = vi.fn().mockResolvedValue(undefined);

  const mockLocationSettings = {
    aliases: {},
    order: [],
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('empty state', () => {
    it('should render empty message when no devices', () => {
      render(
        <DashboardTabContent
          devices={{}}
          aliases={{}}
          propertyDescriptions={mockPropertyDescriptions}
          locationSettings={mockLocationSettings}
          onPropertyChange={mockOnPropertyChange}
          isConnected={true}
        />
      );

      expect(screen.getByTestId('dashboard-empty')).toBeInTheDocument();
      expect(screen.getByText('No devices found.')).toBeInTheDocument();
    });

    it('should render empty message when only Node Profile devices exist', () => {
      const devices = {
        'np1': createNodeProfileDevice('192.168.1.1'),
        'np2': createNodeProfileDevice('192.168.1.2')
      };

      render(
        <DashboardTabContent
          devices={devices}
          aliases={{}}
          propertyDescriptions={mockPropertyDescriptions}
          locationSettings={mockLocationSettings}
          onPropertyChange={mockOnPropertyChange}
          isConnected={true}
        />
      );

      expect(screen.getByTestId('dashboard-empty')).toBeInTheDocument();
    });
  });

  describe('location grouping', () => {
    it('should group devices by installation location', () => {
      const devices = {
        'd1': createDevice('192.168.1.1', '0130:1', 'living'),
        'd2': createDevice('192.168.1.2', '0130:2', 'living'),
        'd3': createDevice('192.168.1.3', '0291:1', 'kitchen')
      };

      render(
        <DashboardTabContent
          devices={devices}
          aliases={{}}
          propertyDescriptions={mockPropertyDescriptions}
          locationSettings={mockLocationSettings}
          onPropertyChange={mockOnPropertyChange}
          isConnected={true}
        />
      );

      expect(screen.getByTestId('dashboard-content')).toBeInTheDocument();
      expect(screen.getByTestId('dashboard-location-living')).toBeInTheDocument();
      expect(screen.getByTestId('dashboard-location-kitchen')).toBeInTheDocument();
    });

    it('should render location headers', () => {
      const devices = {
        'd1': createDevice('192.168.1.1', '0130:1', 'living'),
        'd2': createDevice('192.168.1.2', '0291:1', 'kitchen')
      };

      render(
        <DashboardTabContent
          devices={devices}
          aliases={{}}
          propertyDescriptions={mockPropertyDescriptions}
          locationSettings={mockLocationSettings}
          onPropertyChange={mockOnPropertyChange}
          isConnected={true}
        />
      );

      // Location headers should be visible as h3 elements
      const locationLiving = screen.getByTestId('dashboard-location-living');
      const locationKitchen = screen.getByTestId('dashboard-location-kitchen');
      expect(locationLiving.querySelector('h3')).toHaveTextContent(/living/i);
      expect(locationKitchen.querySelector('h3')).toHaveTextContent(/kitchen/i);
    });

    it('should render location label inside grid when only floor heaters exist (no AC)', () => {
      // Floor heater class code is 027B
      const devices = {
        'd1': createDevice('192.168.1.1', '027B:1', 'living')
      };

      render(
        <DashboardTabContent
          devices={devices}
          aliases={{}}
          propertyDescriptions={mockPropertyDescriptions}
          locationSettings={mockLocationSettings}
          onPropertyChange={mockOnPropertyChange}
          isConnected={true}
        />
      );

      const locationDiv = screen.getByTestId('dashboard-location-living');
      const gridContainer = locationDiv.querySelector('.grid');

      // When only floor heaters exist, the label should be inside the grid (first cell)
      // instead of being in a separate row above the grid
      expect(gridContainer).toBeInTheDocument();
      expect(gridContainer?.querySelector('h3')).toHaveTextContent(/living/i);

      // The label should NOT be outside the grid (as a direct child of locationDiv)
      const directH3Children = locationDiv.querySelectorAll(':scope > h3');
      expect(directH3Children.length).toBe(0);
    });

    it('should render location label outside grid when AC exists', () => {
      // Air conditioner class code is 0130
      const devices = {
        'd1': createDevice('192.168.1.1', '0130:1', 'living')
      };

      render(
        <DashboardTabContent
          devices={devices}
          aliases={{}}
          propertyDescriptions={mockPropertyDescriptions}
          locationSettings={mockLocationSettings}
          onPropertyChange={mockOnPropertyChange}
          isConnected={true}
        />
      );

      const locationDiv = screen.getByTestId('dashboard-location-living');

      // When AC exists, label should be outside the grid (in a div wrapper)
      const directDivWithH3 = locationDiv.querySelector(':scope > div:first-child > h3');
      expect(directDivWithH3).toHaveTextContent(/living/i);
    });

    it('should render DashboardCard for each device', () => {
      const devices = {
        'd1': createDevice('192.168.1.1', '0130:1', 'living'),
        'd2': createDevice('192.168.1.2', '0130:2', 'living'),
        'd3': createDevice('192.168.1.3', '0291:1', 'kitchen')
      };

      render(
        <DashboardTabContent
          devices={devices}
          aliases={{}}
          propertyDescriptions={mockPropertyDescriptions}
          locationSettings={mockLocationSettings}
          onPropertyChange={mockOnPropertyChange}
          isConnected={true}
        />
      );

      expect(screen.getByTestId('dashboard-card-192.168.1.1-0130:1')).toBeInTheDocument();
      expect(screen.getByTestId('dashboard-card-192.168.1.2-0130:2')).toBeInTheDocument();
      expect(screen.getByTestId('dashboard-card-192.168.1.3-0291:1')).toBeInTheDocument();
    });
  });

  describe('Node Profile exclusion', () => {
    it('should not render Node Profile devices', () => {
      const devices = {
        'd1': createDevice('192.168.1.1', '0130:1', 'living'),
        'np1': createNodeProfileDevice('192.168.1.1')
      };

      render(
        <DashboardTabContent
          devices={devices}
          aliases={{}}
          propertyDescriptions={mockPropertyDescriptions}
          locationSettings={mockLocationSettings}
          onPropertyChange={mockOnPropertyChange}
          isConnected={true}
        />
      );

      expect(screen.getByTestId('dashboard-card-192.168.1.1-0130:1')).toBeInTheDocument();
      expect(screen.queryByTestId('dashboard-card-192.168.1.1-0EF0:1')).not.toBeInTheDocument();
    });
  });

  describe('connection state', () => {
    it('should pass isConnected to DashboardCard', () => {
      const devices = {
        'd1': createDevice('192.168.1.1', '0130:1', 'living')
      };

      const { rerender } = render(
        <DashboardTabContent
          devices={devices}
          aliases={{}}
          propertyDescriptions={mockPropertyDescriptions}
          locationSettings={mockLocationSettings}
          onPropertyChange={mockOnPropertyChange}
          isConnected={true}
        />
      );

      // Switch should be enabled when connected
      let switchElement = screen.getByRole('switch');
      expect(switchElement).not.toBeDisabled();

      // Re-render with disconnected state
      rerender(
        <DashboardTabContent
          devices={devices}
          aliases={{}}
          propertyDescriptions={mockPropertyDescriptions}
          locationSettings={mockLocationSettings}
          onPropertyChange={mockOnPropertyChange}
          isConnected={false}
        />
      );

      // Switch should be disabled when disconnected
      switchElement = screen.getByRole('switch');
      expect(switchElement).toBeDisabled();
    });
  });

  describe('location label navigation', () => {
    it('should call onSelectTab when location label is clicked', () => {
      const mockOnSelectTab = vi.fn();
      const devices = {
        'd1': createDevice('192.168.1.1', '0130:1', 'living')
      };

      render(
        <DashboardTabContent
          devices={devices}
          aliases={{}}
          propertyDescriptions={mockPropertyDescriptions}
          locationSettings={mockLocationSettings}
          onPropertyChange={mockOnPropertyChange}
          isConnected={true}
          onSelectTab={mockOnSelectTab}
        />
      );

      const locationButton = screen.getByRole('button', { name: /living.*タブを開く/i });
      fireEvent.click(locationButton);
      expect(mockOnSelectTab).toHaveBeenCalledWith('living');
    });

    it('should render as h3 when onSelectTab is not provided', () => {
      const devices = {
        'd1': createDevice('192.168.1.1', '0130:1', 'living')
      };

      render(
        <DashboardTabContent
          devices={devices}
          aliases={{}}
          propertyDescriptions={mockPropertyDescriptions}
          locationSettings={mockLocationSettings}
          onPropertyChange={mockOnPropertyChange}
          isConnected={true}
        />
      );

      const locationDiv = screen.getByTestId('dashboard-location-living');
      expect(locationDiv.querySelector('h3')).toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /タブを開く/i })).not.toBeInTheDocument();
    });

    it('should have accessibility attributes when clickable', () => {
      const mockOnSelectTab = vi.fn();
      const devices = {
        'd1': createDevice('192.168.1.1', '0130:1', 'living')
      };

      render(
        <DashboardTabContent
          devices={devices}
          aliases={{}}
          propertyDescriptions={mockPropertyDescriptions}
          locationSettings={mockLocationSettings}
          onPropertyChange={mockOnPropertyChange}
          isConnected={true}
          onSelectTab={mockOnSelectTab}
        />
      );

      const locationButton = screen.getByRole('button', { name: /living.*タブを開く/i });
      expect(locationButton).toHaveAttribute('aria-label');
      expect(locationButton).toHaveAttribute('title');
    });
  });
});
