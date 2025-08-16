import { render, screen, fireEvent } from '@testing-library/react';
import { RefreshAllOfflineButton } from './RefreshAllOfflineButton';
import type { Device } from '@/hooks/types';

// Mock the language helper
vi.mock('@/libs/languageHelper', () => ({
  getCurrentLocale: vi.fn(() => 'en')
}));

const mockOfflineDevice: Device = {
  id: '192.168.1.100 0291:1',
  ip: '192.168.1.100',
  eoj: '0291:1',
  name: 'Test Device',
  properties: {},
  lastSeen: '2023-01-01T00:00:00Z',
  isOffline: true
};

describe('RefreshAllOfflineButton', () => {
  it('renders button when there are offline devices', () => {
    const onRefreshAll = vi.fn();
    
    render(
      <RefreshAllOfflineButton
        offlineDevices={[mockOfflineDevice]}
        onRefreshAll={onRefreshAll}
        isUpdating={false}
        isConnected={true}
      />
    );

    const button = screen.getByRole('button', { name: /refresh all offline/i });
    expect(button).toBeInTheDocument();
    expect(button).not.toBeDisabled();
    expect(screen.getByText('(1)')).toBeInTheDocument();
  });

  it('is disabled when not connected', () => {
    const onRefreshAll = vi.fn();
    
    render(
      <RefreshAllOfflineButton
        offlineDevices={[mockOfflineDevice]}
        onRefreshAll={onRefreshAll}
        isUpdating={false}
        isConnected={false}
      />
    );

    const button = screen.getByRole('button', { name: /refresh all offline/i });
    expect(button).toBeDisabled();
  });

  it('is disabled when updating', () => {
    const onRefreshAll = vi.fn();
    
    render(
      <RefreshAllOfflineButton
        offlineDevices={[mockOfflineDevice]}
        onRefreshAll={onRefreshAll}
        isUpdating={true}
        isConnected={true}
      />
    );

    const button = screen.getByRole('button', { name: /refresh all offline/i });
    expect(button).toBeDisabled();
  });

  it('shows spinner when updating', () => {
    const onRefreshAll = vi.fn();
    
    render(
      <RefreshAllOfflineButton
        offlineDevices={[mockOfflineDevice]}
        onRefreshAll={onRefreshAll}
        isUpdating={true}
        isConnected={true}
      />
    );

    const spinner = screen.getByTestId('refresh-spinner');
    expect(spinner).toHaveClass('animate-spin');
  });

  it('calls onRefreshAll when clicked', () => {
    const onRefreshAll = vi.fn();
    
    render(
      <RefreshAllOfflineButton
        offlineDevices={[mockOfflineDevice]}
        onRefreshAll={onRefreshAll}
        isUpdating={false}
        isConnected={true}
      />
    );

    const button = screen.getByRole('button', { name: /refresh all offline/i });
    fireEvent.click(button);
    
    expect(onRefreshAll).toHaveBeenCalledTimes(1);
  });

  it('shows correct count of offline devices', () => {
    const onRefreshAll = vi.fn();
    const multipleOfflineDevices = [mockOfflineDevice, { ...mockOfflineDevice, id: '192.168.1.102 0291:1', ip: '192.168.1.102' }];
    
    render(
      <RefreshAllOfflineButton
        offlineDevices={multipleOfflineDevices}
        onRefreshAll={onRefreshAll}
        isUpdating={false}
        isConnected={true}
      />
    );

    expect(screen.getByText('(2)')).toBeInTheDocument();
  });

  it('renders nothing when no offline devices', () => {
    const onRefreshAll = vi.fn();
    
    const { container } = render(
      <RefreshAllOfflineButton
        offlineDevices={[]}
        onRefreshAll={onRefreshAll}
        isUpdating={false}
        isConnected={true}
      />
    );

    expect(container.firstChild).toBeNull();
  });

  it('renders nothing when only online devices', () => {
    const onRefreshAll = vi.fn();
    
    const { container } = render(
      <RefreshAllOfflineButton
        offlineDevices={[]}
        onRefreshAll={onRefreshAll}
        isUpdating={false}
        isConnected={true}
      />
    );

    expect(container.firstChild).toBeNull();
  });

  it('renders Japanese text when locale is Japanese', async () => {
    const { getCurrentLocale } = await import('@/libs/languageHelper');
    vi.mocked(getCurrentLocale).mockReturnValue('ja');
    
    const onRefreshAll = vi.fn();
    
    render(
      <RefreshAllOfflineButton
        offlineDevices={[mockOfflineDevice]}
        onRefreshAll={onRefreshAll}
        isUpdating={false}
        isConnected={true}
      />
    );

    expect(screen.getByText('オフライン端末を更新')).toBeInTheDocument();
    
    // Restore the mock
    vi.mocked(getCurrentLocale).mockReturnValue('en');
  });

  it('shows correct title with Japanese locale', async () => {
    const { getCurrentLocale } = await import('@/libs/languageHelper');
    vi.mocked(getCurrentLocale).mockReturnValue('ja');
    
    const onRefreshAll = vi.fn();
    
    render(
      <RefreshAllOfflineButton
        offlineDevices={[mockOfflineDevice]}
        onRefreshAll={onRefreshAll}
        isUpdating={false}
        isConnected={true}
      />
    );

    const button = screen.getByRole('button');
    expect(button).toHaveAttribute('title', '1台のオフライン端末を更新');
    
    // Restore the mock
    vi.mocked(getCurrentLocale).mockReturnValue('en');
  });
});