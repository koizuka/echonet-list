import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PropertyInputControl } from './PropertyInputControl';
import { getCurrentLocale } from '@/libs/languageHelper';

vi.mock('@/libs/languageHelper', () => ({
  getCurrentLocale: vi.fn(() => 'en'),
}));

describe('PropertyInputControl', () => {
  const defaultProps = {
    currentValue: { string: 'hello' },
    onSave: vi.fn().mockResolvedValue(undefined),
    disabled: false,
  };

  beforeEach(() => {
    vi.mocked(getCurrentLocale).mockReturnValue('en');
  });

  it('should give the icon-only edit button an accessible name', () => {
    render(<PropertyInputControl {...defaultProps} />);
    expect(screen.getByRole('button', { name: 'Edit value' })).toHaveAttribute('aria-label', 'Edit value');
  });

  it('should give the icon-only save and cancel buttons accessible names while editing', () => {
    render(<PropertyInputControl {...defaultProps} />);
    fireEvent.click(screen.getByRole('button', { name: 'Edit value' }));

    expect(screen.getByRole('button', { name: 'Save' })).toHaveAttribute('aria-label', 'Save');
    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveAttribute('aria-label', 'Cancel');
  });

  it('should use Japanese button names in a Japanese locale', () => {
    vi.mocked(getCurrentLocale).mockReturnValue('ja');
    render(<PropertyInputControl {...defaultProps} />);
    fireEvent.click(screen.getByRole('button', { name: '値を編集' }));

    expect(screen.getByRole('button', { name: '保存' })).toHaveAttribute('aria-label', '保存');
    expect(screen.getByRole('button', { name: 'キャンセル' })).toHaveAttribute('aria-label', 'キャンセル');
  });
});
