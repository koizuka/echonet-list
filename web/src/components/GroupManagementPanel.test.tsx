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

    expect(screen.getByTitle('Group settings')).toBeInTheDocument();
  });

  it('should not display management title', () => {
    render(<GroupManagementPanel {...defaultProps} />);

    expect(screen.queryByText('@testgroup management')).not.toBeInTheDocument();
  });

  it('should call onRename when rename menu item is clicked', async () => {
    const user = userEvent.setup();
    render(<GroupManagementPanel {...defaultProps} />);

    // Open dropdown menu
    const menuButton = screen.getByTitle('Group settings');
    await user.click(menuButton);

    // Click rename option
    const renameButton = await screen.findByText('Rename group');
    await user.click(renameButton);

    expect(defaultProps.onRename).toHaveBeenCalled();
  });

  it('should call onEditMembers when edit members menu item is clicked', async () => {
    const user = userEvent.setup();
    render(<GroupManagementPanel {...defaultProps} />);

    // Open dropdown menu
    const menuButton = screen.getByTitle('Group settings');
    await user.click(menuButton);

    // Click edit members option
    const editButton = await screen.findByText('Edit members');
    await user.click(editButton);

    expect(defaultProps.onEditMembers).toHaveBeenCalled();
  });

  it('should show confirmation dialog when delete menu item is clicked', async () => {
    const user = userEvent.setup();
    render(<GroupManagementPanel {...defaultProps} />);

    // Open dropdown menu
    const menuButton = screen.getByTitle('Group settings');
    await user.click(menuButton);

    // Click delete option
    const deleteButton = await screen.findByText('Delete group');
    await user.click(deleteButton);

    // Check if confirmation dialog appears
    expect(await screen.findByText('Delete group confirmation')).toBeInTheDocument();
    expect(screen.getByText('Are you sure you want to delete @testgroup?')).toBeInTheDocument();
    expect(screen.getByText('This action cannot be undone.')).toBeInTheDocument();
  });

  it('should call onDelete when deletion is confirmed', async () => {
    const user = userEvent.setup();
    render(<GroupManagementPanel {...defaultProps} />);

    // Open dropdown menu
    const menuButton = screen.getByTitle('Group settings');
    await user.click(menuButton);

    // Open confirmation dialog
    const deleteButton = await screen.findByText('Delete group');
    await user.click(deleteButton);

    // Confirm deletion
    const confirmButton = await screen.findByText('Delete');
    await user.click(confirmButton);

    expect(defaultProps.onDelete).toHaveBeenCalled();
  });

  it('should not call onDelete when deletion is cancelled', async () => {
    const user = userEvent.setup();
    render(<GroupManagementPanel {...defaultProps} />);

    // Open dropdown menu
    const menuButton = screen.getByTitle('Group settings');
    await user.click(menuButton);

    // Open confirmation dialog
    const deleteButton = await screen.findByText('Delete group');
    await user.click(deleteButton);

    // Cancel deletion
    const cancelButton = await screen.findByText('Cancel');
    await user.click(cancelButton);

    expect(defaultProps.onDelete).not.toHaveBeenCalled();
  });

  it('should close confirmation dialog when cancelled', async () => {
    const user = userEvent.setup();
    render(<GroupManagementPanel {...defaultProps} />);

    // Open dropdown menu
    const menuButton = screen.getByTitle('Group settings');
    await user.click(menuButton);

    // Open confirmation dialog
    const deleteButton = await screen.findByText('Delete group');
    await user.click(deleteButton);

    // Verify dialog is open
    expect(await screen.findByText('Delete group confirmation')).toBeInTheDocument();

    // Cancel deletion
    const cancelButton = screen.getByText('Cancel');
    await user.click(cancelButton);

    // Verify dialog is closed
    await waitFor(() => {
      expect(screen.queryByText('Delete group confirmation')).not.toBeInTheDocument();
    });
  });

  it('should have appropriate menu button styles', () => {
    render(<GroupManagementPanel {...defaultProps} />);

    const menuButton = screen.getByTitle('Group settings');

    // Check that menu button has small height (h-9 is small size)
    expect(menuButton).toHaveClass('h-9');
  });
});