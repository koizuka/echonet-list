import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { SelfNodeInstanceListSDisplay } from './SelfNodeInstanceListSDisplay';
import { getCurrentLocale } from '@/libs/languageHelper';
import type { Device } from '@/hooks/types';

vi.mock('@/libs/languageHelper', () => ({
  getCurrentLocale: vi.fn(() => 'en'),
}));

describe('SelfNodeInstanceListSDisplay', () => {
  const nodeProfile: Device = {
    ip: '192.168.1.10',
    eoj: '0EF0:1',
    name: 'Node Profile',
    id: undefined,
    properties: {},
    lastSeen: '2024-01-01T00:00:00Z',
  };

  // One instance (0130:1) that is not in allDevices
  const oneMissingInstance = { EDT: btoa(String.fromCharCode(0x01, 0x01, 0x30, 0x01)) };

  const baseProps = {
    device: nodeProfile,
    allDevices: {},
    aliases: {},
    propertyDescriptions: {},
    getDeviceClassCode: () => '0EF0',
  };

  beforeEach(() => {
    // mockReturnValue persists across tests, so reset the locale for each test
    vi.mocked(getCurrentLocale).mockReturnValue('en');
  });

  it('should show English texts in an English locale', () => {
    render(<SelfNodeInstanceListSDisplay {...baseProps} currentValue={oneMissingInstance} />);
    expect(screen.getByText('Instance List (1)')).toBeInTheDocument();
    expect(screen.getByText('Device not found')).toBeInTheDocument();
  });

  it('should show Japanese texts in a Japanese locale', () => {
    vi.mocked(getCurrentLocale).mockReturnValue('ja');
    render(<SelfNodeInstanceListSDisplay {...baseProps} currentValue={oneMissingInstance} />);
    expect(screen.getByText('インスタンスリスト (1)')).toBeInTheDocument();
    expect(screen.getByText('デバイスが見つかりません')).toBeInTheDocument();
  });

  it('should show a localized message for invalid data', () => {
    const invalid = { EDT: btoa(String.fromCharCode(0x02, 0x01)) };
    const { unmount } = render(<SelfNodeInstanceListSDisplay {...baseProps} currentValue={invalid} />);
    expect(screen.getByText('Invalid instance list data')).toBeInTheDocument();
    unmount();

    vi.mocked(getCurrentLocale).mockReturnValue('ja');
    render(<SelfNodeInstanceListSDisplay {...baseProps} currentValue={invalid} />);
    expect(screen.getByText('インスタンスリストのデータが不正です')).toBeInTheDocument();
  });
});
