import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { 
  DropdownMenu, 
  DropdownMenuContent, 
  DropdownMenuItem, 
  DropdownMenuTrigger 
} from '@/components/ui/dropdown-menu';
import { ChevronDown, Edit3, Check, X } from 'lucide-react';
import { isPropertySettable } from '@/libs/propertyHelper';
import type { PropertyDescriptor, PropertyValue, Device } from '@/hooks/types';

interface PropertyEditorProps {
  device: Device;
  epc: string;
  currentValue: PropertyValue;
  descriptor?: PropertyDescriptor;
  onPropertyChange: (target: string, epc: string, value: PropertyValue) => Promise<void>;
}

export function PropertyEditor({ 
  device, 
  epc, 
  currentValue, 
  descriptor, 
  onPropertyChange 
}: PropertyEditorProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const deviceId = `${device.ip} ${device.eoj}`;

  const hasAliases = descriptor?.aliases && Object.keys(descriptor.aliases).length > 0;
  const hasNumberDesc = descriptor?.numberDesc;
  const hasStringDesc = descriptor?.stringDesc;
  
  // Check if property is settable based on:
  // 1. Property descriptor indicates it's settable (stringSettable, numberDesc, or aliases)
  // 2. Property is listed in Set Property Map (EPC 0x9E)
  const hasEditCapability = descriptor?.stringSettable || hasNumberDesc || hasAliases;
  const isInSetPropertyMap = isPropertySettable(epc, device);
  const isSettable = hasEditCapability && isInSetPropertyMap;

  // Handle alias selection
  const handleAliasSelect = async (aliasName: string) => {
    if (!descriptor?.aliases) return;
    
    setIsLoading(true);
    try {
      await onPropertyChange(deviceId, epc, { string: aliasName });
    } catch (error) {
      console.error('Failed to set property:', error);
    } finally {
      setIsLoading(false);
    }
  };

  // Handle string/number editing
  const startEditing = () => {
    if (currentValue.string) {
      setEditValue(currentValue.string);
    } else if (currentValue.number !== undefined) {
      setEditValue(currentValue.number.toString());
    } else {
      setEditValue('');
    }
    setIsEditing(true);
  };

  const cancelEditing = () => {
    setIsEditing(false);
    setEditValue('');
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

      await onPropertyChange(deviceId, epc, propertyValue);
      setIsEditing(false);
      setEditValue('');
    } catch (error) {
      console.error('Failed to set property:', error);
    } finally {
      setIsLoading(false);
    }
  };

  // Don't render anything if property is not settable
  if (!isSettable) {
    return null;
  }

  return (
    <div className="flex items-center gap-2">
      {/* Alias dropdown menu */}
      {hasAliases && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button 
              variant="outline" 
              size="sm" 
              disabled={isLoading}
              className="h-7 px-2"
            >
              <ChevronDown className="h-3 w-3" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {Object.keys(descriptor.aliases!).map((aliasName) => {
              const isSelected = currentValue.string === aliasName;
              return (
                <DropdownMenuItem 
                  key={aliasName}
                  onClick={() => handleAliasSelect(aliasName)}
                  disabled={isLoading}
                  className={`flex items-center justify-between ${
                    isSelected 
                      ? 'bg-accent/50 text-accent-foreground data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground' 
                      : 'data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground'
                  }`}
                >
                  <span>{aliasName}</span>
                  {isSelected && <Check className="h-3 w-3 ml-2" />}
                </DropdownMenuItem>
              );
            })}
          </DropdownMenuContent>
        </DropdownMenu>
      )}

      {/* String/Number editing */}
      {(hasStringDesc || hasNumberDesc) && !isEditing && (
        <Button 
          variant="outline" 
          size="sm" 
          onClick={startEditing}
          disabled={isLoading}
          className="h-7 px-2"
        >
          <Edit3 className="h-3 w-3" />
        </Button>
      )}

      {/* Editing mode */}
      {isEditing && (
        <div className="flex items-center gap-1">
          <Input
            value={editValue}
            onChange={(e) => setEditValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                saveEdit();
              } else if (e.key === 'Escape') {
                cancelEditing();
              }
            }}
            placeholder={
              hasNumberDesc 
                ? `${descriptor.numberDesc!.min}-${descriptor.numberDesc!.max}${descriptor.numberDesc!.unit}` 
                : 'Enter value'
            }
            className="h-7 text-xs w-20"
            disabled={isLoading}
          />
          <Button 
            variant="outline" 
            size="sm" 
            onClick={saveEdit}
            disabled={isLoading || !editValue.trim()}
            className="h-7 px-1"
          >
            <Check className="h-3 w-3" />
          </Button>
          <Button 
            variant="outline" 
            size="sm" 
            onClick={cancelEditing}
            disabled={isLoading}
            className="h-7 px-1"
          >
            <X className="h-3 w-3" />
          </Button>
        </div>
      )}
    </div>
  );
}