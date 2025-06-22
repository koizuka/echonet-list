import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
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

  it('should render three management buttons', () => {
    render(<GroupManagementPanel {...defaultProps} />);
    
    expect(screen.getByText('グループ名を変更')).toBeInTheDocument();
    expect(screen.getByText('メンバーを編集')).toBeInTheDocument();
    expect(screen.getByText('グループを削除')).toBeInTheDocument();
  });

  it('should not display management title', () => {
    render(<GroupManagementPanel {...defaultProps} />);
    
    expect(screen.queryByText('@testgroup の管理')).not.toBeInTheDocument();
  });

  it('should call onRename when rename button is clicked', () => {
    render(<GroupManagementPanel {...defaultProps} />);
    
    const renameButton = screen.getByText('グループ名を変更');
    fireEvent.click(renameButton);
    
    expect(defaultProps.onRename).toHaveBeenCalled();
  });

  it('should call onEditMembers when edit members button is clicked', () => {
    render(<GroupManagementPanel {...defaultProps} />);
    
    const editButton = screen.getByText('メンバーを編集');
    fireEvent.click(editButton);
    
    expect(defaultProps.onEditMembers).toHaveBeenCalled();
  });

  it('should show confirmation dialog when delete button is clicked', () => {
    render(<GroupManagementPanel {...defaultProps} />);
    
    const deleteButton = screen.getByText('グループを削除');
    fireEvent.click(deleteButton);
    
    // Check if confirmation dialog appears
    expect(screen.getByText('グループの削除確認')).toBeInTheDocument();
    expect(screen.getByText('@testgroup を削除してもよろしいですか？')).toBeInTheDocument();
    expect(screen.getByText('この操作は取り消せません。')).toBeInTheDocument();
  });

  it('should call onDelete when deletion is confirmed', () => {
    render(<GroupManagementPanel {...defaultProps} />);
    
    // Open confirmation dialog
    const deleteButton = screen.getByText('グループを削除');
    fireEvent.click(deleteButton);
    
    // Confirm deletion
    const confirmButton = screen.getByText('削除する');
    fireEvent.click(confirmButton);
    
    expect(defaultProps.onDelete).toHaveBeenCalled();
  });

  it('should not call onDelete when deletion is cancelled', () => {
    render(<GroupManagementPanel {...defaultProps} />);
    
    // Open confirmation dialog
    const deleteButton = screen.getByText('グループを削除');
    fireEvent.click(deleteButton);
    
    // Cancel deletion
    const cancelButton = screen.getByText('キャンセル');
    fireEvent.click(cancelButton);
    
    expect(defaultProps.onDelete).not.toHaveBeenCalled();
  });

  it('should close confirmation dialog when cancelled', () => {
    render(<GroupManagementPanel {...defaultProps} />);
    
    // Open confirmation dialog
    const deleteButton = screen.getByText('グループを削除');
    fireEvent.click(deleteButton);
    
    // Verify dialog is open
    expect(screen.getByText('グループの削除確認')).toBeInTheDocument();
    
    // Cancel deletion
    const cancelButton = screen.getByText('キャンセル');
    fireEvent.click(cancelButton);
    
    // Verify dialog is closed
    expect(screen.queryByText('グループの削除確認')).not.toBeInTheDocument();
  });

  it('should have appropriate button styles', () => {
    render(<GroupManagementPanel {...defaultProps} />);
    
    const renameButton = screen.getByText('グループ名を変更');
    const editButton = screen.getByText('メンバーを編集');
    const deleteButton = screen.getByText('グループを削除');
    
    // Check that delete button has destructive class
    expect(deleteButton.closest('button')).toHaveClass('destructive');
    
    // Check that all buttons have small height (h-9 is small size)
    expect(renameButton.closest('button')).toHaveClass('h-9');
    expect(editButton.closest('button')).toHaveClass('h-9');
    expect(deleteButton.closest('button')).toHaveClass('h-9');
  });
});