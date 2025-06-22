import { render, screen, fireEvent, waitFor } from '@testing-library/react';
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
    it('should show error for empty alias', async () => {
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
      fireEvent.click(saveButton);

      expect(screen.getByText('エイリアス名を入力してください')).toBeInTheDocument();
    });

    it('should show error for hex-like alias', async () => {
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
      fireEvent.click(saveButton);

      expect(screen.getByText('16進数として読める偶数桁の名前は使用できません')).toBeInTheDocument();
    });

    it('should show error for alias starting with symbol', async () => {
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
      fireEvent.click(saveButton);

      expect(screen.getByText('記号で始まる名前は使用できません')).toBeInTheDocument();
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
        expect(mockOnDeleteAlias).toHaveBeenCalledWith('living_ac');
        expect(mockOnAddAlias).toHaveBeenCalledWith('bedroom_ac', '013001:00000B:ABCDEF0123456789ABCDEF012345');
      });
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
});