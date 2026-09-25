import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PropertyInputControl } from './PropertyInputControl';

describe('PropertyInputControl', () => {
  const defaultProps = {
    currentValue: { string: 'hello' },
    onSave: vi.fn().mockResolvedValue(undefined),
    disabled: false,
  };

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
});
