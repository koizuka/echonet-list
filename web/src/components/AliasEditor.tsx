import { useState, useRef, useEffect } from 'react';
import { Edit2, Trash2, Plus, Check, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { validateDeviceAlias } from '@/libs/aliasHelper';
import type { Device } from '@/hooks/types';

interface AliasEditorProps {
  device: Device;
  aliases: string[];
  onAddAlias: (alias: string, target: string) => Promise<void>;
  onDeleteAlias: (alias: string) => Promise<void>;
  isLoading?: boolean;
  deviceIdentifier: string;
  isConnected?: boolean;
}

export function AliasEditor({
  device: _device, // Not used but kept for future extensibility
  aliases,
  onAddAlias,
  onDeleteAlias,
  isLoading = false,
  deviceIdentifier,
  isConnected = true
}: AliasEditorProps) {
  const [editingIndex, setEditingIndex] = useState<number | null>(null); // null means not editing, -1 means adding new alias, >=0 means editing existing
  const [inputValue, setInputValue] = useState('');
  const [error, setError] = useState<string | undefined>();
  const [savingIndex, setSavingIndex] = useState<number | null>(null);
  const [isComposing, setIsComposing] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  // Focus input when entering edit mode
  useEffect(() => {
    if (editingIndex !== null && inputRef.current) {
      inputRef.current.focus();
      // Select all text for easy replacement
      inputRef.current.select();
    }
  }, [editingIndex]);

  const handleStartEdit = (index: number) => {
    setEditingIndex(index);
    setInputValue(index >= 0 ? aliases[index] : '');
    setError(undefined);
  };

  const handleStartAdd = () => {
    setEditingIndex(-1);
    setInputValue('');
    setError(undefined);
  };

  const handleCancel = () => {
    setEditingIndex(null);
    setInputValue('');
    setError(undefined);
  };

  // Check if save button should be disabled
  const getIsSaveDisabled = () => {
    // Disabled while saving
    if (savingIndex !== null) return true;
    
    // Check validation
    const validationError = validateDeviceAlias(inputValue);
    if (validationError) return true;
    
    // Check if value has changed (for existing alias)
    if (editingIndex !== null && editingIndex >= 0) {
      const originalValue = aliases[editingIndex];
      if (inputValue === originalValue) return true;
    }
    
    // For new alias, just check that it's not empty and valid
    if (editingIndex === -1 && inputValue.trim() === '') return true;
    
    return false;
  };

  const handleSave = async () => {
    const validationError = validateDeviceAlias(inputValue);
    if (validationError) {
      setError(validationError);
      return;
    }

    setSavingIndex(editingIndex);
    try {
      // Add the new alias first
      await onAddAlias(inputValue, deviceIdentifier);
      
      // If updating existing alias, delete old one after successful addition
      if (editingIndex !== null && editingIndex >= 0 && aliases[editingIndex] !== inputValue) {
        await onDeleteAlias(aliases[editingIndex]);
      }
      
      setEditingIndex(null);
      setInputValue('');
      setError(undefined);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'エイリアスの保存に失敗しました');
    } finally {
      setSavingIndex(null);
    }
  };

  const handleDelete = async (aliasToDelete: string, index: number) => {
    setSavingIndex(index);
    try {
      await onDeleteAlias(aliasToDelete);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'エイリアスの削除に失敗しました');
    } finally {
      setSavingIndex(null);
    }
  };

  // Render editing form when editing
  if (editingIndex !== null) {
    return (
      <div className="space-y-2 w-full">
        <div className="flex gap-2 w-full">
          <Input
            ref={inputRef}
            value={inputValue}
            onChange={(e) => {
              setInputValue(e.target.value);
              setError(undefined);
            }}
            placeholder="エイリアス名を入力"
            className="h-7 text-xs flex-1"
            disabled={savingIndex !== null}
            onCompositionStart={() => setIsComposing(true)}
            onCompositionEnd={() => setIsComposing(false)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !isComposing) {
                if (!getIsSaveDisabled()) {
                  handleSave();
                }
              } else if (e.key === 'Escape') {
                handleCancel();
              }
            }}
          />
          <div className="flex items-center gap-1 flex-shrink-0">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleSave}
              disabled={getIsSaveDisabled()}
              className="h-7 w-7 p-0"
              title="保存"
            >
              <Check className="h-3 w-3" />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={handleCancel}
              disabled={savingIndex !== null}
              className="h-7 w-7 p-0"
              title="キャンセル"
            >
              <X className="h-3 w-3" />
            </Button>
          </div>
        </div>
        {error && (
          <p className="text-xs text-destructive">{error}</p>
        )}
      </div>
    );
  }

  return (
    <div className="w-full space-y-2">
      {/* Display existing aliases */}
      {aliases.map((alias, index) => (
        <div key={`${alias}-${index}`} className="flex gap-2 w-full">
          <div className="flex-1 min-w-0">
            <div className="text-xs bg-secondary text-secondary-foreground px-2 py-0.5 rounded break-all leading-relaxed">
              {alias}
            </div>
          </div>
          <div className="flex items-center gap-1 flex-shrink-0">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => handleStartEdit(index)}
              disabled={isLoading || savingIndex !== null || !isConnected}
              className="h-6 w-6 p-0"
              title="エイリアスを編集"
            >
              <Edit2 className="h-3 w-3" />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => handleDelete(alias, index)}
              disabled={isLoading || savingIndex !== null || !isConnected}
              className="h-6 w-6 p-0"
              title="エイリアスを削除"
            >
              <Trash2 className="h-3 w-3" />
            </Button>
          </div>
        </div>
      ))}
      
      {/* Add new alias button - always shown at bottom */}
      <div className="flex justify-start w-full">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleStartAdd}
          disabled={isLoading || savingIndex !== null || !isConnected}
          className="h-6 w-6 p-0"
          title="エイリアスを追加"
        >
          <Plus className="h-3 w-3" />
        </Button>
      </div>
      
      {/* Error message (for delete operations) */}
      {error && (
        <p className="text-xs text-destructive">{error}</p>
      )}
    </div>
  );
}