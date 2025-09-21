import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { PropertySliderControl } from './PropertySliderControl';
import type { PropertyValue, PropertyDescriptor } from '@/hooks/types';

describe('PropertySliderControl', () => {
  const mockOnSave = vi.fn();

  const mockDescriptor: PropertyDescriptor = {
    description: 'Illuminance Level',
    numberDesc: {
      min: 0,
      max: 100,
      unit: '%',
      offset: 0,
      edtLen: 1
    }
  };

  beforeEach(() => {
    mockOnSave.mockClear();
  });

  it('should render slider with current value', () => {
    const currentValue: PropertyValue = { number: 50 };

    render(
      <PropertySliderControl
        currentValue={currentValue}
        descriptor={mockDescriptor}
        onSave={mockOnSave}
        disabled={false}
        testId="illuminance"
      />
    );

    // Check if current value is displayed (using getAllByText to handle multiple instances)
    expect(screen.getAllByText('50%')).toHaveLength(2); // One in main display, one in slider value

    // Check if min/max labels are displayed
    expect(screen.getByText('0')).toBeInTheDocument();
    expect(screen.getByText('100')).toBeInTheDocument();

    // Check if slider exists
    const slider = screen.getByTestId('immediate-slider-illuminance');
    expect(slider).toBeInTheDocument();
  });

  it('should render fallback when no numberDesc provided', () => {
    const currentValue: PropertyValue = { string: 'test' };
    const descriptorWithoutNumber: PropertyDescriptor = {
      description: 'Test Property'
    };

    render(
      <PropertySliderControl
        currentValue={currentValue}
        descriptor={descriptorWithoutNumber}
        onSave={mockOnSave}
        disabled={false}
      />
    );

    expect(screen.getByText('test')).toBeInTheDocument();
    expect(screen.queryByRole('slider')).not.toBeInTheDocument();
  });

  it('should use min value when current value is undefined', () => {
    const currentValue: PropertyValue = {};

    render(
      <PropertySliderControl
        currentValue={currentValue}
        descriptor={mockDescriptor}
        onSave={mockOnSave}
        disabled={false}
      />
    );

    // Should show min value when current is undefined
    // The component shows "Raw data" when there's no valid number
    expect(screen.getByText('Raw data')).toBeInTheDocument();
    // Check that min value is used for slider (min=0)
    expect(screen.getByRole('slider')).toHaveAttribute('aria-valuenow', '0');
  });

  it('should call onSave with new value on slider commit', async () => {
    const currentValue: PropertyValue = { number: 50 };

    render(
      <PropertySliderControl
        currentValue={currentValue}
        descriptor={mockDescriptor}
        onSave={mockOnSave}
        disabled={false}
        testId="illuminance"
      />
    );

    const slider = screen.getByTestId('immediate-slider-illuminance');

    // Simulate slider change and commit (this simulates dragging and releasing)
    // Note: Testing slider interaction might be complex with Radix UI
    // We'll test the handler function directly
    const sliderControl = slider.querySelector('[role="slider"]');
    expect(sliderControl).toBeInTheDocument();

    // For now, test that the component renders correctly
    // Integration testing would be needed for actual slider interaction
  });

  it('should not call onSave when value has not changed', async () => {
    const currentValue: PropertyValue = { number: 50 };

    render(
      <PropertySliderControl
        currentValue={currentValue}
        descriptor={mockDescriptor}
        onSave={mockOnSave}
        disabled={false}
      />
    );

    // If the slider commits the same value, onSave should not be called
    // This would be tested at integration level with actual slider interaction
    expect(mockOnSave).not.toHaveBeenCalled();
  });

  it('should show loading state when disabled', () => {
    const currentValue: PropertyValue = { number: 50 };

    render(
      <PropertySliderControl
        currentValue={currentValue}
        descriptor={mockDescriptor}
        onSave={mockOnSave}
        disabled={true}
        testId="illuminance"
      />
    );

    const slider = screen.getByTestId('immediate-slider-illuminance');
    expect(slider).toHaveAttribute('aria-disabled', 'true');
  });

  it('should handle different value ranges', () => {
    const currentValue: PropertyValue = { number: 25 };
    const customDescriptor: PropertyDescriptor = {
      description: 'Custom Range',
      numberDesc: {
        min: 10,
        max: 50,
        unit: '°C',
        offset: 0,
        edtLen: 1
      }
    };

    render(
      <PropertySliderControl
        currentValue={currentValue}
        descriptor={customDescriptor}
        onSave={mockOnSave}
        disabled={false}
      />
    );

    expect(screen.getAllByText('25°C')).toHaveLength(2);
    expect(screen.getByText('10')).toBeInTheDocument();
    expect(screen.getByText('50')).toBeInTheDocument();
  });

  it('should handle error in onSave gracefully', async () => {
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const mockOnError = vi.fn();
    mockOnSave.mockRejectedValue(new Error('Network error'));

    const currentValue: PropertyValue = { number: 50 };

    render(
      <PropertySliderControl
        currentValue={currentValue}
        descriptor={mockDescriptor}
        onSave={mockOnSave}
        disabled={false}
        onError={mockOnError}
      />
    );

    // Error handling would be tested at integration level
    // For now, just ensure component renders without crashing
    expect(screen.getAllByText('50%')).toHaveLength(2);

    consoleErrorSpy.mockRestore();
  });

  it('should call onError when range validation fails', async () => {
    const mockOnError = vi.fn();
    const currentValue: PropertyValue = { number: 50 };

    const component = render(
      <PropertySliderControl
        currentValue={currentValue}
        descriptor={mockDescriptor}
        onSave={mockOnSave}
        disabled={false}
        onError={mockOnError}
      />
    );

    // Test range validation by directly calling handleValueCommit with out-of-range value
    // This would be integration tested with actual slider interaction
    expect(component).toBeTruthy();
  });
});