import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PropertyInputControl } from './PropertyInputControl';
import { isJapanese } from '@/libs/languageHelper';

vi.mock('@/libs/languageHelper', () => ({
  isJapanese: vi.fn(() => false),
  getCurrentLocale: vi.fn(() => 'en'),
}));

describe('PropertyInputControl', () => {
  const defaultProps = {
    currentValue: { string: 'hello' },
    onSave: vi.fn().mockResolvedValue(undefined),
    disabled: false,
  };

  beforeEach(() => {
    vi.mocked(isJapanese).mockReturnValue(false);
  });

  it('should give the icon-only edit button an accessible name', () => {
    render(<PropertyInputControl {...defaultProps} />);
    expect(screen.getByRole('button', { name: 'Edit value' })).toBeInTheDocument();
  });

  it('should give the icon-only save and cancel buttons accessible names while editing', () => {
    render(<PropertyInputControl {...defaultProps} />);
    fireEvent.click(screen.getByRole('button', { name: 'Edit value' }));

    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
  });

  it('should use Japanese button names in a Japanese locale', () => {
    vi.mocked(isJapanese).mockReturnValue(true);
    render(<PropertyInputControl {...defaultProps} />);
    fireEvent.click(screen.getByRole('button', { name: '値を編集' }));

    expect(screen.getByRole('button', { name: '保存' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'キャンセル' })).toBeInTheDocument();
  });
});
