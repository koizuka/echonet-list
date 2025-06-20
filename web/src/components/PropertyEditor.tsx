import { useState, useRef, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { 
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Edit3, Check, X, Binary } from 'lucide-react';
import { isPropertySettable, formatPropertyValueWithTranslation, shouldShowHexViewer, edtToHexString } from '@/libs/propertyHelper';
import { translateInstallationLocation } from '@/libs/locationHelper';
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
  const [showHexData, setShowHexData] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const deviceId = `${device.ip} ${device.eoj}`;

  // Auto-focus input when entering edit mode
  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
    }
  }, [isEditing]);

  const hasAliases = descriptor?.aliases && Object.keys(descriptor.aliases).length > 0;
  const hasNumberDesc = descriptor?.numberDesc;
  const hasStringDesc = descriptor?.stringDesc;
  const canShowHexViewer = shouldShowHexViewer(currentValue, descriptor);
  
  // Check if property is settable based on:
  // 1. Property descriptor indicates it's settable (stringSettable, numberDesc, or aliases)
  // 2. Property is listed in Set Property Map (EPC 0x9E)
  const hasEditCapability = descriptor?.stringSettable || hasNumberDesc || hasAliases;
  const isInSetPropertyMap = isPropertySettable(epc, device);
  const isSettable = hasEditCapability && isInSetPropertyMap;
  
  // Check if this is Installation Location property (EPC 0x81)
  const isInstallationLocation = epc === '81';

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

  // For read-only properties, only show hex viewer if applicable
  if (!isSettable) {
    if (canShowHexViewer) {
      return (
        <div className="relative">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">Raw data</span>
            <Button
              variant={showHexData ? "default" : "outline"}
              size="sm"
              onClick={() => setShowHexData(!showHexData)}
              className="h-6 w-6 p-0"
              title={showHexData ? "Hide hex data" : "Show hex data"}
            >
              <Binary className="h-3 w-3" />
            </Button>
          </div>
          {showHexData && currentValue.EDT && (
            <div className="absolute top-full left-0 right-0 mt-1 text-xs font-mono bg-muted p-2 rounded border break-all shadow-md z-10 min-w-max">
              {edtToHexString(currentValue.EDT) || 'Invalid data'}
            </div>
          )}
        </div>
      );
    }
    return null;
  }

  return (
    <div className="flex items-center gap-2">
      {/* Alias select - hidden when editing */}
      {hasAliases && !isEditing && (
        <Select
          value={currentValue.string || ''}
          onValueChange={(value) => handleAliasSelect(value)}
          disabled={isLoading}
        >
          <SelectTrigger className="h-7 w-[120px]" data-testid={`alias-select-trigger-${epc}`}>
            <SelectValue>
              {currentValue.string ? 
                (isInstallationLocation ? translateInstallationLocation(currentValue.string) : currentValue.string) 
                : 'Select...'}
            </SelectValue>
          </SelectTrigger>
          <SelectContent data-testid={`alias-select-content-${epc}`}>
            {Object.keys(descriptor.aliases!).map((aliasName) => (
              <SelectItem key={aliasName} value={aliasName} data-testid={`alias-option-${aliasName}`}>
                {isInstallationLocation ? translateInstallationLocation(aliasName) : aliasName}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}

      {/* String/Number editing */}
      {(hasStringDesc || hasNumberDesc) && !hasAliases && !isEditing && (
        <div className="relative">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">
              {formatPropertyValueWithTranslation(currentValue, descriptor, epc, translateInstallationLocation)}
            </span>
            <Button 
              variant="outline" 
              size="sm" 
              onClick={startEditing}
              disabled={isLoading}
              className="h-7 px-2"
              data-testid={`edit-button-${epc}`}
            >
              <Edit3 className="h-3 w-3" />
            </Button>
            {canShowHexViewer && (
              <Button
                variant={showHexData ? "default" : "outline"}
                size="sm"
                onClick={() => setShowHexData(!showHexData)}
                className="h-6 w-6 p-0"
                title={showHexData ? "Hide hex data" : "Show hex data"}
              >
                <Binary className="h-3 w-3" />
              </Button>
            )}
          </div>
          {showHexData && canShowHexViewer && currentValue.EDT && (
            <div className="absolute top-full left-0 right-0 mt-1 text-xs font-mono bg-muted p-2 rounded border break-all shadow-md z-10 min-w-max">
              {edtToHexString(currentValue.EDT) || 'Invalid data'}
            </div>
          )}
        </div>
      )}
      
      {/* String/Number editing - only edit button when aliases exist */}
      {(hasStringDesc || hasNumberDesc) && hasAliases && !isEditing && (
        <div className="relative">
          <div className="flex items-center gap-2">
            <Button 
              variant="outline" 
              size="sm" 
              onClick={startEditing}
              disabled={isLoading}
              className="h-7 px-2"
            >
              <Edit3 className="h-3 w-3" />
            </Button>
            {canShowHexViewer && (
              <Button
                variant={showHexData ? "default" : "outline"}
                size="sm"
                onClick={() => setShowHexData(!showHexData)}
                className="h-6 w-6 p-0"
                title={showHexData ? "Hide hex data" : "Show hex data"}
              >
                <Binary className="h-3 w-3" />
              </Button>
            )}
          </div>
          {showHexData && canShowHexViewer && currentValue.EDT && (
            <div className="absolute top-full left-0 right-0 mt-1 text-xs font-mono bg-muted p-2 rounded border break-all shadow-md z-10 min-w-max">
              {edtToHexString(currentValue.EDT) || 'Invalid data'}
            </div>
          )}
        </div>
      )}

      {/* Editing mode */}
      {isEditing && (
        <div className="flex items-center gap-1">
          <Input
            ref={inputRef}
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
            data-testid={`edit-input-${epc}`}
          />
          <Button 
            variant="outline" 
            size="sm" 
            onClick={saveEdit}
            disabled={isLoading || !editValue.trim()}
            className="h-7 px-1"
            data-testid={`save-button-${epc}`}
          >
            <Check className="h-3 w-3" />
          </Button>
          <Button 
            variant="outline" 
            size="sm" 
            onClick={cancelEditing}
            disabled={isLoading}
            className="h-7 px-1"
            data-testid={`cancel-button-${epc}`}
          >
            <X className="h-3 w-3" />
          </Button>
        </div>
      )}
    </div>
  );
}