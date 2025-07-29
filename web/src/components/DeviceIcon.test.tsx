import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/react';
import { DeviceIcon } from './DeviceIcon';
import type { Device } from '@/hooks/types';

// Mock the helper modules
vi.mock('@/libs/propertyHelper', () => ({
  isDeviceOperational: vi.fn(),
  isDeviceFaulty: vi.fn(),
}));

vi.mock('@/libs/deviceIconHelper', () => ({
  getDeviceIcon: vi.fn(() => TestIcon),
  getDeviceIconColor: vi.fn(),
}));

// Test icon component
function TestIcon({ className }: { className?: string }) {
  return <svg className={className} data-testid="test-icon" />;
}

import { isDeviceOperational, isDeviceFaulty } from '@/libs/propertyHelper';
import { getDeviceIcon, getDeviceIconColor } from '@/libs/deviceIconHelper';

const mockIsDeviceOperational = isDeviceOperational as ReturnType<typeof vi.fn>;
const mockIsDeviceFaulty = isDeviceFaulty as ReturnType<typeof vi.fn>;
const mockGetDeviceIcon = getDeviceIcon as ReturnType<typeof vi.fn>;
const mockGetDeviceIconColor = getDeviceIconColor as ReturnType<typeof vi.fn>;

describe('DeviceIcon', () => {
  const createMockDevice = (overrides?: Partial<Device>): Device => ({
    ip: '192.168.1.1',
    eoj: '0130:01',
    name: 'Test Device',
    id: undefined,
    properties: { '80': { string: 'ON' } },
    lastSeen: new Date().toISOString(),
    isOffline: false,
    ...overrides,
  });

  beforeEach(() => {
    vi.clearAllMocks();
    mockGetDeviceIcon.mockReturnValue(TestIcon);
  });

  it('should render device icon with correct color for operational device', () => {
    mockIsDeviceOperational.mockReturnValue(true);
    mockIsDeviceFaulty.mockReturnValue(false);
    mockGetDeviceIconColor.mockReturnValue('text-green-500');

    const device = createMockDevice();
    const { container } = render(<DeviceIcon device={device} classCode="0130" />);
    
    const icon = container.querySelector('svg');
    expect(icon).toHaveClass('w-4', 'h-4', 'text-green-500');
    expect(mockGetDeviceIconColor).toHaveBeenCalledWith(true, false, false);
  });

  it('should render device icon with correct color for faulty device', () => {
    mockIsDeviceOperational.mockReturnValue(false);
    mockIsDeviceFaulty.mockReturnValue(true);
    mockGetDeviceIconColor.mockReturnValue('text-red-500');

    const device = createMockDevice();
    const { container } = render(<DeviceIcon device={device} classCode="0130" />);
    
    const icon = container.querySelector('svg');
    expect(icon).toHaveClass('text-red-500');
    expect(mockGetDeviceIconColor).toHaveBeenCalledWith(false, true, false);
  });

  it('should render device icon with correct color for offline device', () => {
    mockIsDeviceOperational.mockReturnValue(false);
    mockIsDeviceFaulty.mockReturnValue(false);
    mockGetDeviceIconColor.mockReturnValue('text-muted-foreground');

    const device = createMockDevice({ isOffline: true });
    const { container } = render(<DeviceIcon device={device} classCode="0130" />);
    
    const icon = container.querySelector('svg');
    expect(icon).toHaveClass('text-muted-foreground');
    expect(mockGetDeviceIconColor).toHaveBeenCalledWith(false, false, true);
  });

  it('should show correct tooltip for air conditioner', () => {
    mockIsDeviceOperational.mockReturnValue(true);
    mockIsDeviceFaulty.mockReturnValue(false);

    const device = createMockDevice();
    const { container } = render(<DeviceIcon device={device} classCode="0130" />);
    
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper.title).toBe('Air Conditioner - ON');
  });

  it('should show correct tooltip for offline device', () => {
    mockIsDeviceOperational.mockReturnValue(false);
    mockIsDeviceFaulty.mockReturnValue(false);

    const device = createMockDevice({ isOffline: true });
    const { container } = render(<DeviceIcon device={device} classCode="0291" />);
    
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper.title).toBe('Lighting - Offline');
  });

  it('should show correct tooltip for faulty device', () => {
    mockIsDeviceOperational.mockReturnValue(false);
    mockIsDeviceFaulty.mockReturnValue(true);

    const device = createMockDevice();
    const { container } = render(<DeviceIcon device={device} classCode="03B7" />);
    
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper.title).toBe('Refrigerator - Error');
  });

  it('should apply custom className', () => {
    mockIsDeviceOperational.mockReturnValue(false);
    mockIsDeviceFaulty.mockReturnValue(false);
    mockGetDeviceIconColor.mockReturnValue('text-gray-400');

    const device = createMockDevice();
    const { container } = render(
      <DeviceIcon device={device} classCode="0130" className="custom-class" />
    );
    
    const icon = container.querySelector('svg');
    expect(icon).toHaveClass('custom-class');
  });

  it('should handle unknown device type', () => {
    mockIsDeviceOperational.mockReturnValue(false);
    mockIsDeviceFaulty.mockReturnValue(false);

    const device = createMockDevice();
    const { container } = render(<DeviceIcon device={device} classCode="9999" />);
    
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper.title).toBe('Unknown Device - OFF');
  });
});