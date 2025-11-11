import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { GroupManagementPanel } from './GroupManagementPanel';

describe('GroupManagementPanel', () => {
  const defaultProps = {
    groupName: '@testgroup',
    onRename: vi.fn(),
    onDelete: vi.fn(),
    onEditMembers: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render group settings button', () => {
    render(<GroupManagementPanel {...defaultProps} />);

    expect(screen.getByTitle('グループ設定')).toBeInTheDocument();
  });

  it('should not display management title', () => {
    render(<GroupManagementPanel {...defaultProps} />);
    
    expect(screen.queryByText('@testgroup の管理')).not.toBeInTheDocument();
  });

  it('should call onRename when rename menu item is clicked', async () => {
    const user = userEvent.setup();
    render(<GroupManagementPanel {...defaultProps} />);

    // Open dropdown menu
    const menuButton = screen.getByTitle('グループ設定');
    await user.click(menuButton);

    // Click rename option
    const renameButton = await screen.findByText('グループ名を変更');
    await user.click(renameButton);

    expect(defaultProps.onRename).toHaveBeenCalled();
  });

  it('should call onEditMembers when edit members menu item is clicked', async () => {
    const user = userEvent.setup();
    render(<GroupManagementPanel {...defaultProps} />);

    // Open dropdown menu
    const menuButton = screen.getByTitle('グループ設定');
    await user.click(menuButton);

    // Click edit members option
    const editButton = await screen.findByText('メンバーを編集');
    await user.click(editButton);

    expect(defaultProps.onEditMembers).toHaveBeenCalled();
  });

  it('should show confirmation dialog when delete menu item is clicked', async () => {
    const user = userEvent.setup();
    render(<GroupManagementPanel {...defaultProps} />);

    // Open dropdown menu
    const menuButton = screen.getByTitle('グループ設定');
    await user.click(menuButton);

    // Click delete option
    const deleteButton = await screen.findByText('グループを削除');
    await user.click(deleteButton);

    // Check if confirmation dialog appears
    expect(await screen.findByText('グループの削除確認')).toBeInTheDocument();
    expect(screen.getByText('@testgroup を削除してもよろしいですか？')).toBeInTheDocument();
    expect(screen.getByText('この操作は取り消せません。')).toBeInTheDocument();
  });

  it('should call onDelete when deletion is confirmed', async () => {
    const user = userEvent.setup();
    render(<GroupManagementPanel {...defaultProps} />);

    // Open dropdown menu
    const menuButton = screen.getByTitle('グループ設定');
    await user.click(menuButton);

    // Open confirmation dialog
    const deleteButton = await screen.findByText('グループを削除');
    await user.click(deleteButton);

    // Confirm deletion
    const confirmButton = await screen.findByText('削除する');
    await user.click(confirmButton);

    expect(defaultProps.onDelete).toHaveBeenCalled();
  });

  it('should not call onDelete when deletion is cancelled', async () => {
    const user = userEvent.setup();
    render(<GroupManagementPanel {...defaultProps} />);

    // Open dropdown menu
    const menuButton = screen.getByTitle('グループ設定');
    await user.click(menuButton);

    // Open confirmation dialog
    const deleteButton = await screen.findByText('グループを削除');
    await user.click(deleteButton);

    // Cancel deletion
    const cancelButton = await screen.findByText('キャンセル');
    await user.click(cancelButton);

    expect(defaultProps.onDelete).not.toHaveBeenCalled();
  });

  it('should close confirmation dialog when cancelled', async () => {
    const user = userEvent.setup();
    render(<GroupManagementPanel {...defaultProps} />);

    // Open dropdown menu
    const menuButton = screen.getByTitle('グループ設定');
    await user.click(menuButton);

    // Open confirmation dialog
    const deleteButton = await screen.findByText('グループを削除');
    await user.click(deleteButton);

    // Verify dialog is open
    expect(await screen.findByText('グループの削除確認')).toBeInTheDocument();

    // Cancel deletion
    const cancelButton = screen.getByText('キャンセル');
    await user.click(cancelButton);

    // Verify dialog is closed
    await waitFor(() => {
      expect(screen.queryByText('グループの削除確認')).not.toBeInTheDocument();
    });
  });

  it('should have appropriate menu button styles', () => {
    render(<GroupManagementPanel {...defaultProps} />);

    const menuButton = screen.getByTitle('グループ設定');

    // Check that menu button has small height (h-9 is small size)
    expect(menuButton).toHaveClass('h-9');
  });
});