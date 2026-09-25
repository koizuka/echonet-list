import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { PropertySelectControl } from './PropertySelectControl';
import { getCurrentLocale } from '@/libs/languageHelper';

vi.mock('@/libs/languageHelper', () => ({
  getCurrentLocale: vi.fn(() => 'en'),
}));

describe('PropertySelectControl', () => {
  const props = {
    value: '',
    aliases: { on: 'MzA=', off: 'MzE=' },
    onChange: vi.fn(),
    disabled: false,
    testId: 'op-status',
  };

  beforeEach(() => {
    // mockReturnValue persists across tests, so reset the locale for each test
    vi.mocked(getCurrentLocale).mockReturnValue('en');
  });

  it('should show an English placeholder when no value is selected', () => {
    render(<PropertySelectControl {...props} />);
    expect(screen.getByTestId('op-status')).toHaveTextContent('Select...');
  });

  it('should show a Japanese placeholder in a Japanese locale', () => {
    vi.mocked(getCurrentLocale).mockReturnValue('ja');
    render(<PropertySelectControl {...props} />);
    expect(screen.getByTestId('op-status')).toHaveTextContent('選択...');
  });
});
