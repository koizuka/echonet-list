import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { GroupNameEditor } from './GroupNameEditor';

describe('GroupNameEditor', () => {
  const defaultProps = {
    groupName: '@testgroup',
    existingGroups: ['@group1', '@group2'],
    onSave: vi.fn(),
    onCancel: vi.fn(),
    isLoading: false,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render input with initial group name', () => {
    render(<GroupNameEditor {...defaultProps} />);
    const input = screen.getByRole('textbox');
    expect(input).toHaveValue('@testgroup');
  });

  it('should render save and cancel buttons', () => {
    render(<GroupNameEditor {...defaultProps} />);
    expect(screen.getByTitle('保存')).toBeInTheDocument();
    expect(screen.getByTitle('キャンセル')).toBeInTheDocument();
  });

  it('should disable save button when value has not changed', () => {
    render(<GroupNameEditor {...defaultProps} />);
    const saveButton = screen.getByTitle('保存');
    expect(saveButton).toBeDisabled();
  });

  it('should enable save button when value changes to valid name', () => {
    render(<GroupNameEditor {...defaultProps} />);
    
    const input = screen.getByRole('textbox');
    const saveButton = screen.getByTitle('保存');
    
    // Clear and type new value
    fireEvent.change(input, { target: { value: '@newgroup' } });
    
    expect(saveButton).not.toBeDisabled();
  });

  it('should show validation error for invalid names', () => {
    render(<GroupNameEditor {...defaultProps} />);
    
    const input = screen.getByRole('textbox');
    
    // Test missing @
    fireEvent.change(input, { target: { value: 'group' } });
    expect(screen.getByText('グループ名は @ で始まる必要があります')).toBeInTheDocument();
    
    // Test whitespace
    fireEvent.change(input, { target: { value: '@group name' } });
    expect(screen.getByText('グループ名に空白文字を含めることはできません')).toBeInTheDocument();
  });

  it('should show error for duplicate group names', () => {
    render(<GroupNameEditor {...defaultProps} />);
    
    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: '@group1' } });
    
    expect(screen.getByText('このグループ名は既に使用されています')).toBeInTheDocument();
  });

  it('should disable save button when validation fails', () => {
    render(<GroupNameEditor {...defaultProps} />);
    
    const input = screen.getByRole('textbox');
    const saveButton = screen.getByTitle('保存');
    
    fireEvent.change(input, { target: { value: '@group 1' } });
    
    expect(saveButton).toBeDisabled();
  });

  it('should call onSave with new name when save is clicked', () => {
    render(<GroupNameEditor {...defaultProps} />);
    
    const input = screen.getByRole('textbox');
    const saveButton = screen.getByTitle('保存');
    
    fireEvent.change(input, { target: { value: '@newgroup' } });
    fireEvent.click(saveButton);
    
    expect(defaultProps.onSave).toHaveBeenCalledWith('@newgroup');
  });

  it('should call onCancel when cancel is clicked', () => {
    render(<GroupNameEditor {...defaultProps} />);
    
    const cancelButton = screen.getByTitle('キャンセル');
    fireEvent.click(cancelButton);
    
    expect(defaultProps.onCancel).toHaveBeenCalled();
  });

  it('should handle Enter key to save', () => {
    render(<GroupNameEditor {...defaultProps} />);
    
    const input = screen.getByRole('textbox');
    
    fireEvent.change(input, { target: { value: '@newgroup' } });
    fireEvent.keyDown(input, { key: 'Enter' });
    
    expect(defaultProps.onSave).toHaveBeenCalledWith('@newgroup');
  });

  it('should handle Escape key to cancel', () => {
    render(<GroupNameEditor {...defaultProps} />);
    
    const input = screen.getByRole('textbox');
    fireEvent.keyDown(input, { key: 'Escape' });
    
    expect(defaultProps.onCancel).toHaveBeenCalled();
  });

  it('should disable all controls when isLoading is true', () => {
    render(<GroupNameEditor {...defaultProps} isLoading={true} />);
    
    expect(screen.getByRole('textbox')).toBeDisabled();
    expect(screen.getByTitle('保存')).toBeDisabled();
    expect(screen.getByTitle('キャンセル')).toBeDisabled();
  });

  it('should not call onSave during Japanese IME composition', async () => {
    render(<GroupNameEditor {...defaultProps} />);
    
    const input = screen.getByRole('textbox');
    
    // Clear and type new value
    fireEvent.change(input, { target: { value: '@新しいグループ' } });
    
    // Start composition
    fireEvent.compositionStart(input);
    
    // Try to submit with Enter
    fireEvent.keyDown(input, { key: 'Enter' });
    
    expect(defaultProps.onSave).not.toHaveBeenCalled();
    
    // End composition
    fireEvent.compositionEnd(input);
    
    // Now Enter should work
    fireEvent.keyDown(input, { key: 'Enter' });
    
    expect(defaultProps.onSave).toHaveBeenCalledWith('@新しいグループ');
  });

  it('should auto-focus and select text on mount', async () => {
    render(<GroupNameEditor {...defaultProps} />);
    
    const input = screen.getByRole('textbox') as HTMLInputElement;
    
    await waitFor(() => {
      expect(document.activeElement).toBe(input);
    });
  });
});