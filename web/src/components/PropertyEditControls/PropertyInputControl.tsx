import { useState, useRef, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Slider } from '@/components/ui/slider';
import { Edit3, Check, X } from 'lucide-react';
import type { PropertyValue, PropertyDescriptor } from '@/hooks/types';

interface PropertyInputControlProps {
  currentValue: PropertyValue;
  descriptor?: PropertyDescriptor;
  onSave: (value: PropertyValue) => Promise<void>;
  disabled: boolean;
  testId?: string;
  onEditModeChange?: (isEditing: boolean) => void;
}

export function PropertyInputControl({ 
  currentValue, 
  descriptor, 
  onSave, 
  disabled,
  testId,
  onEditModeChange
}: PropertyInputControlProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState('');
  const [sliderValue, setSliderValue] = useState<number[]>([0]);
  const [isLoading, setIsLoading] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const hasNumberDesc = descriptor?.numberDesc;

  // Auto-focus input when entering edit mode
  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
    }
  }, [isEditing]);

  const startEditing = () => {
    if (currentValue.number !== undefined) {
      setEditValue(currentValue.number.toString());
      setSliderValue([currentValue.number]);
    } else if (currentValue.string && !hasNumberDesc) {
      setEditValue(currentValue.string);
    } else {
      setEditValue('');
      if (hasNumberDesc && descriptor?.numberDesc) {
        setSliderValue([descriptor.numberDesc.min]);
      }
    }
    setIsEditing(true);
    onEditModeChange?.(true);
  };

  const cancelEditing = () => {
    setIsEditing(false);
    setEditValue('');
    setSliderValue([0]);
    onEditModeChange?.(false);
  };

  const saveEdit = async () => {
    if (!editValue.trim()) return;

    setIsLoading(true);
    try {
      let propertyValue: PropertyValue;
      
      if (hasNumberDesc) {
        const numValue = parseInt(editValue, 10);
        if (!isNaN(numValue)) {
          propertyValue = { number: numValue };
        } else {
          throw new Error('Invalid number value');
        }
      } else {
        propertyValue = { string: editValue };
      }

      await onSave(propertyValue);
      setIsEditing(false);
      setEditValue('');
      setSliderValue([0]);
      onEditModeChange?.(false);
    } catch (error) {
      console.error('Failed to set property:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSliderChange = (value: number[]) => {
    const numValue = value[0];
    setSliderValue(value);
    setEditValue(numValue.toString());
  };


  if (!isEditing) {
    return (
      <Button 
        variant="outline" 
        size="sm" 
        onClick={startEditing}
        disabled={disabled || isLoading}
        className="h-7 px-2"
        data-testid={testId ? `edit-button-${testId}` : undefined}
      >
        <Edit3 className="h-3 w-3" />
      </Button>
    );
  }

  return (
    <div className="flex flex-col gap-2 min-w-0">
      <div className="flex items-center gap-2 min-w-0">
        <Input
          ref={inputRef}
          type={hasNumberDesc ? "number" : "text"}
          value={editValue}
          onChange={(e) => {
            setEditValue(e.target.value);
            // Update slider if it's a valid number
            if (hasNumberDesc && descriptor?.numberDesc) {
              const numValue = parseInt(e.target.value, 10);
              if (!isNaN(numValue)) {
                setSliderValue([Math.max(descriptor.numberDesc.min, Math.min(descriptor.numberDesc.max, numValue))]);
              }
            }
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              saveEdit();
            } else if (e.key === 'Escape') {
              cancelEditing();
            }
          }}
          placeholder={
            hasNumberDesc
              ? `${descriptor?.numberDesc!.min}-${descriptor?.numberDesc!.max}${descriptor?.numberDesc!.unit}`
              : 'Enter value'
          }
          min={hasNumberDesc ? descriptor?.numberDesc!.min : undefined}
          max={hasNumberDesc ? descriptor?.numberDesc!.max : undefined}
          step={1}
          className="h-7 text-xs w-20 flex-shrink-0"
          disabled={isLoading}
          data-testid={testId ? `edit-input-${testId}` : undefined}
        />
        <div className="flex items-center gap-1 flex-shrink-0">
          <Button
            variant="outline"
            size="sm"
            onClick={saveEdit}
            disabled={isLoading || !editValue.trim()}
            className="h-7 px-1"
            data-testid={testId ? `save-button-${testId}` : undefined}
          >
            <Check className="h-3 w-3" />
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={cancelEditing}
            disabled={isLoading}
            className="h-7 px-1"
            data-testid={testId ? `cancel-button-${testId}` : undefined}
          >
            <X className="h-3 w-3" />
          </Button>
        </div>
      </div>

      {/* Slider for number properties */}
      {hasNumberDesc && descriptor?.numberDesc && (
        <div className="w-full max-w-48 px-1">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-xs text-muted-foreground flex-shrink-0">{descriptor.numberDesc.min}</span>
            <Slider
              value={sliderValue}
              onValueChange={handleSliderChange}
              min={descriptor.numberDesc.min}
              max={descriptor.numberDesc.max}
              step={1}
              className="flex-1 min-w-0"
              disabled={isLoading}
              data-testid={testId ? `slider-${testId}` : undefined}
            />
            <span className="text-xs text-muted-foreground flex-shrink-0">{descriptor.numberDesc.max}</span>
          </div>
          <div className="text-center text-xs text-muted-foreground">
            {sliderValue[0]}{descriptor.numberDesc.unit}
          </div>
        </div>
      )}
    </div>
  );
}