import React, { useState, useCallback } from 'react';
import { Slider } from '@/components/ui/slider';
import { formatPropertyValue } from '@/libs/propertyHelper';
import { getCurrentLocale } from '@/libs/languageHelper';
import type { PropertyValue, PropertyDescriptor } from '@/hooks/types';
import type { LogEntry } from '@/hooks/useLogNotifications';

interface PropertySliderControlProps {
  currentValue: PropertyValue;
  descriptor?: PropertyDescriptor;
  onSave: (value: PropertyValue) => Promise<void>;
  disabled: boolean;
  testId?: string;
  onError?: (error: LogEntry) => void;
}

/**
 * PropertySliderControl - An immediate slider control for numeric ECHONET properties
 *
 * This component provides a slider interface that allows users to change property values
 * without entering edit mode. The value is sent to the device when the user finishes
 * dragging the slider (onValueCommit).
 *
 * @param currentValue - The current property value
 * @param descriptor - Property descriptor containing min/max/unit information
 * @param onSave - Callback function to save the new value
 * @param disabled - Whether the slider should be disabled
 * @param testId - Test identifier for automated testing
 * @param onError - Optional callback for error notifications
 */
export function PropertySliderControl({
  currentValue,
  descriptor,
  onSave,
  disabled,
  testId,
  onError
}: PropertySliderControlProps) {
  const [isLoading, setIsLoading] = useState(false);
  const [sliderValue, setSliderValue] = useState<number[]>([0]);
  const currentLang = getCurrentLocale();


  // Extract number description for slider configuration
  const numberDesc = descriptor?.numberDesc;

  // Get current value as number
  const currentNum = currentValue.number ?? (numberDesc?.min || 0);

  // Initialize slider value when current value changes
  React.useEffect(() => {
    setSliderValue([currentNum]);
  }, [currentNum]);

  const handleValueChange = useCallback((values: number[]) => {
    // Update local slider state during dragging
    setSliderValue(values);
  }, []);

  const handleValueCommit = useCallback(async (values: number[]) => {
    const newValue = values[0];

    // Only send if value actually changed
    if (newValue === currentNum) return;

    // Range validation (only validate if numberDesc exists)
    if (numberDesc && (newValue < numberDesc.min || newValue > numberDesc.max)) {
      const errorLog: LogEntry = {
        id: `slider-error-${Date.now()}`,
        level: 'ERROR',
        message: `Slider value ${newValue} is outside valid range ${numberDesc.min}-${numberDesc.max}`,
        time: new Date().toISOString(),
        attributes: { value: newValue, min: numberDesc.min, max: numberDesc.max },
        isRead: false
      };
      onError?.(errorLog);
      return;
    }

    setIsLoading(true);
    try {
      await onSave({ number: newValue });
    } catch (error) {
      console.error('Failed to set property:', error);

      // Send error notification
      const errorLog: LogEntry = {
        id: `slider-error-${Date.now()}`,
        level: 'ERROR',
        message: `Failed to set ${descriptor?.description || 'property'}: ${error instanceof Error ? error.message : 'Unknown error'}`,
        time: new Date().toISOString(),
        attributes: { property: descriptor?.description, value: newValue, error },
        isRead: false
      };
      onError?.(errorLog);
    } finally {
      setIsLoading(false);
    }
  }, [currentNum, numberDesc, onSave, onError, descriptor?.description]);

  if (!numberDesc) {
    // Fallback: show formatted value if no number descriptor
    return (
      <span className="text-sm text-muted-foreground">
        {formatPropertyValue(currentValue, descriptor, currentLang)}
      </span>
    );
  }

  return (
    <div className="flex flex-col gap-2 w-48">
      {/* Current value display */}
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium">
          {formatPropertyValue(currentValue, descriptor, currentLang)}
        </span>
        {isLoading && (
          <span className="text-xs text-muted-foreground">Updating...</span>
        )}
      </div>

      {/* Slider with min/max labels */}
      <div className="px-1">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-xs text-muted-foreground">{numberDesc.min}</span>
          <Slider
            value={sliderValue}
            onValueChange={handleValueChange}
            onValueCommit={handleValueCommit}
            min={numberDesc.min}
            max={numberDesc.max}
            step={1}
            className="flex-1"
            disabled={disabled || isLoading}
            data-testid={testId ? `immediate-slider-${testId}` : undefined}
          />
          <span className="text-xs text-muted-foreground">{numberDesc.max}</span>
        </div>
        <div className="text-center text-xs text-muted-foreground">
          {sliderValue[0]}{numberDesc.unit}
        </div>
      </div>
    </div>
  );
}