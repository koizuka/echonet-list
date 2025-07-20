import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { DeviceDeleteConfirmDialog } from './DeviceDeleteConfirmDialog';
import type { Device } from '@/hooks/types';

// Mock the language helper
vi.mock('@/libs/languageHelper', () => ({
  isJapanese: vi.fn(),
}));

const { isJapanese } = vi.mocked(await import('@/libs/languageHelper'));

describe('DeviceDeleteConfirmDialog', () => {
  const mockDevice: Device = {
    ip: '192.168.1.100',
    eoj: '013001',
    name: 'Air Conditioner',
    id: '013001:0000:001234',
    properties: {},
    lastSeen: '2024-01-01T12:00:00Z',
    isOffline: true,
  };

  const defaultProps = {
    device: mockDevice,
    onDeleteDevice: vi.fn(),
    isDeletingDevice: false,
    isConnected: true,
    isOpen: true,
    onOpenChange: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    isJapanese.mockReturnValue(false); // Default to English
  });

  describe('English Language', () => {
    beforeEach(() => {
      isJapanese.mockReturnValue(false);
    });

    it('should render English texts when language is not Japanese', () => {
      render(<DeviceDeleteConfirmDialog {...defaultProps} />);

      expect(screen.getByText('Delete Offline Device')).toBeInTheDocument();
      expect(screen.getByText(/Are you sure you want to delete "Air Conditioner"\?/)).toBeInTheDocument();
      expect(screen.getByText(/This action cannot be undone/)).toBeInTheDocument();
      expect(screen.getByText('Cancel')).toBeInTheDocument();
      expect(screen.getByText('Delete Device')).toBeInTheDocument();
    });

    it('should display device IP and EOJ in English', () => {
      render(<DeviceDeleteConfirmDialog {...defaultProps} />);

      expect(screen.getByText('192.168.1.100 - 013001')).toBeInTheDocument();
    });

    it('should use alias name in English description when provided', () => {
      render(
        <DeviceDeleteConfirmDialog 
          {...defaultProps} 
          aliasName="Living Room AC"
        />
      );

      expect(screen.getByText(/Are you sure you want to delete "Living Room AC"\?/)).toBeInTheDocument();
    });

    it('should show deleting state in English', () => {
      render(
        <DeviceDeleteConfirmDialog 
          {...defaultProps} 
          isDeletingDevice={true}
        />
      );

      expect(screen.getByText('Deleting...')).toBeInTheDocument();
    });
  });

  describe('Japanese Language', () => {
    beforeEach(() => {
      isJapanese.mockReturnValue(true);
    });

    it('should render Japanese texts when language is Japanese', () => {
      render(<DeviceDeleteConfirmDialog {...defaultProps} />);

      expect(screen.getByText('オフラインデバイスを削除')).toBeInTheDocument();
      expect(screen.getByText(/「Air Conditioner」を削除してもよろしいですか？/)).toBeInTheDocument();
      expect(screen.getByText(/この操作は取り消すことができません/)).toBeInTheDocument();
      expect(screen.getByText('キャンセル')).toBeInTheDocument();
      expect(screen.getByText('デバイスを削除')).toBeInTheDocument();
    });

    it('should display device IP and EOJ in Japanese (same format)', () => {
      render(<DeviceDeleteConfirmDialog {...defaultProps} />);

      expect(screen.getByText('192.168.1.100 - 013001')).toBeInTheDocument();
    });

    it('should use alias name in Japanese description when provided', () => {
      render(
        <DeviceDeleteConfirmDialog 
          {...defaultProps} 
          aliasName="リビングエアコン"
        />
      );

      expect(screen.getByText(/「リビングエアコン」を削除してもよろしいですか？/)).toBeInTheDocument();
    });

    it('should show deleting state in Japanese', () => {
      render(
        <DeviceDeleteConfirmDialog 
          {...defaultProps} 
          isDeletingDevice={true}
        />
      );

      expect(screen.getByText('削除中...')).toBeInTheDocument();
    });
  });

  describe('Functionality', () => {
    it('should call onDeleteDevice with correct target when delete button is clicked', async () => {
      const mockOnDeleteDevice = vi.fn();
      render(
        <DeviceDeleteConfirmDialog 
          {...defaultProps} 
          onDeleteDevice={mockOnDeleteDevice}
        />
      );

      const deleteButton = screen.getByText('Delete Device');
      fireEvent.click(deleteButton);

      await waitFor(() => {
        expect(mockOnDeleteDevice).toHaveBeenCalledWith('192.168.1.100 013001');
      });
    });

    it('should call onOpenChange when cancel button is clicked', async () => {
      const mockOnOpenChange = vi.fn();
      render(
        <DeviceDeleteConfirmDialog 
          {...defaultProps} 
          onOpenChange={mockOnOpenChange}
        />
      );

      const cancelButton = screen.getByText('Cancel');
      fireEvent.click(cancelButton);

      await waitFor(() => {
        expect(mockOnOpenChange).toHaveBeenCalledWith(false);
      });
    });

    it('should disable delete button when isDeletingDevice is true', () => {
      render(
        <DeviceDeleteConfirmDialog 
          {...defaultProps} 
          isDeletingDevice={true}
        />
      );

      const deleteButton = screen.getByText('Deleting...');
      expect(deleteButton).toBeDisabled();
    });

    it('should disable delete button when isConnected is false', () => {
      render(
        <DeviceDeleteConfirmDialog 
          {...defaultProps} 
          isConnected={false}
        />
      );

      const deleteButton = screen.getByText('Delete Device');
      expect(deleteButton).toBeDisabled();
    });

    it('should disable cancel button when isDeletingDevice is true', () => {
      render(
        <DeviceDeleteConfirmDialog 
          {...defaultProps} 
          isDeletingDevice={true}
        />
      );

      const cancelButton = screen.getByText('Cancel');
      expect(cancelButton).toBeDisabled();
    });

    it('should not render when isOpen is false', () => {
      render(
        <DeviceDeleteConfirmDialog 
          {...defaultProps} 
          isOpen={false}
        />
      );

      expect(screen.queryByText('Delete Offline Device')).not.toBeInTheDocument();
      expect(screen.queryByText('オフラインデバイスを削除')).not.toBeInTheDocument();
    });
  });

  describe('Props Handling', () => {
    it('should handle missing aliasName gracefully', () => {
      render(<DeviceDeleteConfirmDialog {...defaultProps} />);

      expect(screen.getByText(/Are you sure you want to delete "Air Conditioner"\?/)).toBeInTheDocument();
    });

    it('should prefer aliasName over device.name when both are provided', () => {
      render(
        <DeviceDeleteConfirmDialog 
          {...defaultProps} 
          aliasName="My Custom Name"
        />
      );

      expect(screen.getByText(/Are you sure you want to delete "My Custom Name"\?/)).toBeInTheDocument();
      expect(screen.queryByText(/Are you sure you want to delete "Air Conditioner"\?/)).not.toBeInTheDocument();
    });

    it('should construct correct device target string', () => {
      const mockOnDeleteDevice = vi.fn();
      render(
        <DeviceDeleteConfirmDialog 
          {...defaultProps} 
          device={{
            ...mockDevice,
            ip: '10.0.0.50',
            eoj: '026001',
          }}
          onDeleteDevice={mockOnDeleteDevice}
        />
      );

      const deleteButton = screen.getByText('Delete Device');
      fireEvent.click(deleteButton);

      expect(mockOnDeleteDevice).toHaveBeenCalledWith('10.0.0.50 026001');
    });
  });
});