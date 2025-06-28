import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { PropertySwitchControl } from './PropertySwitchControl';

describe('PropertySwitchControl', () => {
  const mockOnChange = vi.fn();

  beforeEach(() => {
    mockOnChange.mockClear();
  });

  it('should render switch with on state', () => {
    render(
      <PropertySwitchControl
        value="on"
        onChange={mockOnChange}
        disabled={false}
        testId="test-switch"
      />
    );

    const switchElement = screen.getByTestId('test-switch');
    expect(switchElement).toBeInTheDocument();
    expect(switchElement).toHaveAttribute('aria-checked', 'true');
  });

  it('should render switch with off state', () => {
    render(
      <PropertySwitchControl
        value="off"
        onChange={mockOnChange}
        disabled={false}
        testId="test-switch"
      />
    );

    const switchElement = screen.getByTestId('test-switch');
    expect(switchElement).toHaveAttribute('aria-checked', 'false');
  });

  it('should call onChange with "off" when toggled from on', async () => {
    render(
      <PropertySwitchControl
        value="on"
        onChange={mockOnChange}
        disabled={false}
        testId="test-switch"
      />
    );

    const switchElement = screen.getByTestId('test-switch');
    fireEvent.click(switchElement);

    await waitFor(() => {
      expect(mockOnChange).toHaveBeenCalledWith('off');
    });
  });

  it('should call onChange with "on" when toggled from off', async () => {
    render(
      <PropertySwitchControl
        value="off"
        onChange={mockOnChange}
        disabled={false}
        testId="test-switch"
      />
    );

    const switchElement = screen.getByTestId('test-switch');
    fireEvent.click(switchElement);

    await waitFor(() => {
      expect(mockOnChange).toHaveBeenCalledWith('on');
    });
  });

  it('should be disabled when disabled prop is true', () => {
    render(
      <PropertySwitchControl
        value="on"
        onChange={mockOnChange}
        disabled={true}
        testId="test-switch"
      />
    );

    const switchElement = screen.getByTestId('test-switch');
    expect(switchElement).toBeDisabled();
  });

  it('should have correct styling classes', () => {
    render(
      <PropertySwitchControl
        value="on"
        onChange={mockOnChange}
        disabled={false}
        testId="test-switch"
      />
    );

    const switchElement = screen.getByTestId('test-switch');
    expect(switchElement).toHaveClass('data-[state=checked]:bg-green-600');
    expect(switchElement).toHaveClass('data-[state=unchecked]:bg-gray-400');
  });
});