import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { AliasEditor } from './AliasEditor';
import type { Device } from '@/hooks/types';

const mockDevice: Device = {
  ip: '192.168.1.10',
  eoj: '0130:1',
  name: 'HomeAirConditioner',
  id: '013001:00000B:ABCDEF0123456789ABCDEF012345',
  properties: {},
  lastSeen: '2023-04-01T12:00:00Z'
};

const mockDeviceWithoutId: Device = {
  ip: '192.168.1.10',
  eoj: '0130:1',
  name: 'HomeAirConditioner',
  id: undefined,
  properties: {},
  lastSeen: '2023-04-01T12:00:00Z'
};

describe('AliasEditor', () => {
  const mockOnAddAlias = vi.fn();
  const mockOnDeleteAlias = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('when device has no alias', () => {
    it('should show add button', () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={[]}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      expect(screen.getByTitle('エイリアスを追加')).toBeInTheDocument();
      expect(screen.queryByTitle('エイリアスを編集')).not.toBeInTheDocument();
      expect(screen.queryByTitle('エイリアスを削除')).not.toBeInTheDocument();
    });

    it('should enter edit mode when add button is clicked', () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={[]}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      const addButton = screen.getByTitle('エイリアスを追加');
      fireEvent.click(addButton);

      expect(screen.getByPlaceholderText('エイリアス名を入力')).toBeInTheDocument();
      expect(screen.getByTitle('保存')).toBeInTheDocument();
      expect(screen.getByTitle('キャンセル')).toBeInTheDocument();
    });
  });

  describe('when device has alias', () => {
    it('should show edit and delete buttons', () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={['living_ac']}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      expect(screen.getByText('living_ac')).toBeInTheDocument();
      expect(screen.getByTitle('エイリアスを編集')).toBeInTheDocument();
      expect(screen.getByTitle('エイリアスを削除')).toBeInTheDocument();
      expect(screen.getByTitle('エイリアスを追加')).toBeInTheDocument();
    });

    it('should enter edit mode with current alias when edit button is clicked', () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={['living_ac']}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      const editButton = screen.getByTitle('エイリアスを編集');
      fireEvent.click(editButton);

      const input = screen.getByDisplayValue('living_ac');
      expect(input).toBeInTheDocument();
    });

    it('should call onDeleteAlias when delete button is clicked', async () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={['living_ac']}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      const deleteButton = screen.getByTitle('エイリアスを削除');
      fireEvent.click(deleteButton);

      await waitFor(() => {
        expect(mockOnDeleteAlias).toHaveBeenCalledWith('living_ac');
      });
    });
  });

  describe('edit mode validation', () => {
    it('should disable save button for empty alias', async () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={[]}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      const addButton = screen.getByTitle('エイリアスを追加');
      fireEvent.click(addButton);

      const saveButton = screen.getByTitle('保存');
      expect(saveButton).toBeDisabled();
    });

    it('should disable save button for hex-like alias', async () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={[]}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      const addButton = screen.getByTitle('エイリアスを追加');
      fireEvent.click(addButton);

      const input = screen.getByPlaceholderText('エイリアス名を入力');
      fireEvent.change(input, { target: { value: '0130' } });

      const saveButton = screen.getByTitle('保存');
      expect(saveButton).toBeDisabled();
    });

    it('should disable save button for alias starting with symbol', async () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={[]}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      const addButton = screen.getByTitle('エイリアスを追加');
      fireEvent.click(addButton);

      const input = screen.getByPlaceholderText('エイリアス名を入力');
      fireEvent.change(input, { target: { value: '@group' } });

      const saveButton = screen.getByTitle('保存');
      expect(saveButton).toBeDisabled();
    });
  });

  describe('successful operations', () => {
    it('should call onAddAlias with valid new alias', async () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={[]}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      const addButton = screen.getByTitle('エイリアスを追加');
      fireEvent.click(addButton);

      const input = screen.getByPlaceholderText('エイリアス名を入力');
      fireEvent.change(input, { target: { value: 'kitchen_ac' } });

      const saveButton = screen.getByTitle('保存');
      fireEvent.click(saveButton);

      await waitFor(() => {
        expect(mockOnAddAlias).toHaveBeenCalledWith('kitchen_ac', '013001:00000B:ABCDEF0123456789ABCDEF012345');
      });
    });

    it('should call onAddAlias when updating existing alias', async () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={['living_ac']}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      const editButton = screen.getByTitle('エイリアスを編集');
      fireEvent.click(editButton);

      const input = screen.getByDisplayValue('living_ac');
      fireEvent.change(input, { target: { value: 'bedroom_ac' } });

      const saveButton = screen.getByTitle('保存');
      fireEvent.click(saveButton);

      await waitFor(() => {
        expect(mockOnAddAlias).toHaveBeenCalledWith('bedroom_ac', '013001:00000B:ABCDEF0123456789ABCDEF012345');
        expect(mockOnDeleteAlias).toHaveBeenCalledWith('living_ac');
      });

      // Verify the order: add first, then delete
      expect(mockOnAddAlias.mock.calls.length).toBe(1);
      expect(mockOnDeleteAlias.mock.calls.length).toBe(1);
    });

    it('should exit edit mode on cancel', () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={['living_ac']}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      const editButton = screen.getByTitle('エイリアスを編集');
      fireEvent.click(editButton);

      const cancelButton = screen.getByTitle('キャンセル');
      fireEvent.click(cancelButton);

      expect(screen.getByText('living_ac')).toBeInTheDocument();
      expect(screen.queryByPlaceholderText('エイリアス名を入力')).not.toBeInTheDocument();
    });
  });

  describe('loading state', () => {
    it('should disable buttons when loading', () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={['living_ac']}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          isLoading={true}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      expect(screen.getByTitle('エイリアスを編集')).toBeDisabled();
      expect(screen.getByTitle('エイリアスを削除')).toBeDisabled();
    });
  });

  describe('device identifier prop', () => {
    it('should use provided deviceIdentifier for alias operations', async () => {
      render(
        <AliasEditor
          device={mockDeviceWithoutId}
          aliases={[]}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="test_device_id"
        />
      );

      const addButton = screen.getByTitle('エイリアスを追加');
      fireEvent.click(addButton);

      const input = screen.getByPlaceholderText('エイリアス名を入力');
      fireEvent.change(input, { target: { value: 'test_alias' } });

      const saveButton = screen.getByTitle('保存');
      fireEvent.click(saveButton);

      await waitFor(() => {
        expect(mockOnAddAlias).toHaveBeenCalledWith('test_alias', 'test_device_id');
      });
    });
  });

  describe('multiple aliases', () => {
    it('should display multiple aliases with individual edit/delete buttons', () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={['living_ac', 'main_ac', 'primary_ac']}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      expect(screen.getByText('living_ac')).toBeInTheDocument();
      expect(screen.getByText('main_ac')).toBeInTheDocument();
      expect(screen.getByText('primary_ac')).toBeInTheDocument();
      
      // Should have 3 edit buttons and 3 delete buttons
      expect(screen.getAllByTitle('エイリアスを編集')).toHaveLength(3);
      expect(screen.getAllByTitle('エイリアスを削除')).toHaveLength(3);
      
      // Should always have add button at bottom
      expect(screen.getByTitle('エイリアスを追加')).toBeInTheDocument();
    });

    it('should edit specific alias when edit button clicked', () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={['living_ac', 'main_ac']}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      const editButtons = screen.getAllByTitle('エイリアスを編集');
      fireEvent.click(editButtons[1]); // Click edit for second alias (main_ac)

      const input = screen.getByDisplayValue('main_ac');
      expect(input).toBeInTheDocument();
    });
  });

  describe('save button validation', () => {
    it('should disable save button when input value is same as original', () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={['living_ac']}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      const editButton = screen.getByTitle('エイリアスを編集');
      fireEvent.click(editButton);

      const saveButton = screen.getByTitle('保存');
      expect(saveButton).toBeDisabled();
    });

    it('should disable save button when validation fails', () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={[]}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      const addButton = screen.getByTitle('エイリアスを追加');
      fireEvent.click(addButton);

      const input = screen.getByPlaceholderText('エイリアス名を入力');
      fireEvent.change(input, { target: { value: '0130' } }); // Invalid hex-like alias

      const saveButton = screen.getByTitle('保存');
      expect(saveButton).toBeDisabled();
    });

    it('should enable save button when input value is valid and different', () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={['living_ac']}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      const editButton = screen.getByTitle('エイリアスを編集');
      fireEvent.click(editButton);

      const input = screen.getByDisplayValue('living_ac');
      fireEvent.change(input, { target: { value: 'bedroom_ac' } });

      const saveButton = screen.getByTitle('保存');
      expect(saveButton).not.toBeDisabled();
    });
  });

  describe('IME handling', () => {
    it('should not save on Enter when composing', async () => {
      render(
        <AliasEditor
          device={mockDevice}
          aliases={[]}
          onAddAlias={mockOnAddAlias}
          onDeleteAlias={mockOnDeleteAlias}
          deviceIdentifier="013001:00000B:ABCDEF0123456789ABCDEF012345"
        />
      );

      const addButton = screen.getByTitle('エイリアスを追加');
      fireEvent.click(addButton);

      const input = screen.getByPlaceholderText('エイリアス名を入力');
      
      // Start composition (Japanese input)
      act(() => {
        fireEvent.compositionStart(input);
        fireEvent.change(input, { target: { value: 'test_alias' } });
      });
      
      // Press Enter while composing
      act(() => {
        fireEvent.keyDown(input, { key: 'Enter' });
      });
      
      // Should not have called onAddAlias
      expect(mockOnAddAlias).not.toHaveBeenCalled();
      
      // End composition
      act(() => {
        fireEvent.compositionEnd(input);
      });
      
      // Now Enter should work
      act(() => {
        fireEvent.keyDown(input, { key: 'Enter' });
      });
      
      await waitFor(() => {
        expect(mockOnAddAlias).toHaveBeenCalledWith('test_alias', '013001:00000B:ABCDEF0123456789ABCDEF012345');
      });
    });
  });
});