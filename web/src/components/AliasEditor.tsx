import { useState, useRef, useEffect } from 'react';
import { Edit2, Trash2, Plus, Check, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { validateDeviceAlias } from '@/libs/aliasHelper';
import type { Device } from '@/hooks/types';

interface AliasEditorProps {
  device: Device;
  currentAlias?: string;
  onAddAlias: (alias: string, target: string) => Promise<void>;
  onDeleteAlias: (alias: string) => Promise<void>;
  isLoading?: boolean;
  deviceIdentifier: string;
}

export function AliasEditor({
  device: _device, // Not used but kept for future extensibility
  currentAlias,
  onAddAlias,
  onDeleteAlias,
  isLoading = false,
  deviceIdentifier
}: AliasEditorProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [inputValue, setInputValue] = useState(currentAlias || '');
  const [error, setError] = useState<string | undefined>();
  const [isSaving, setIsSaving] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  // Focus input when entering edit mode
  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
      // Select all text for easy replacement
      inputRef.current.select();
    }
  }, [isEditing]);

  const handleStartEdit = () => {
    setIsEditing(true);
    setInputValue(currentAlias || '');
    setError(undefined);
  };

  const handleCancel = () => {
    setIsEditing(false);
    setInputValue(currentAlias || '');
    setError(undefined);
  };

  const handleSave = async () => {
    const validationError = validateDeviceAlias(inputValue);
    if (validationError) {
      setError(validationError);
      return;
    }

    setIsSaving(true);
    try {
      // If updating existing alias, delete old one first
      if (currentAlias && currentAlias !== inputValue) {
        await onDeleteAlias(currentAlias);
      }
      
      // Add the new alias using the correct device identifier
      await onAddAlias(inputValue, deviceIdentifier);
      
      setIsEditing(false);
      setError(undefined);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'エイリアスの保存に失敗しました');
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!currentAlias) return;
    
    setIsSaving(true);
    try {
      await onDeleteAlias(currentAlias);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'エイリアスの削除に失敗しました');
    } finally {
      setIsSaving(false);
    }
  };

  if (isEditing) {
    return (
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <Input
            ref={inputRef}
            value={inputValue}
            onChange={(e) => {
              setInputValue(e.target.value);
              setError(undefined);
            }}
            placeholder="エイリアス名を入力"
            className="h-7 text-xs flex-1"
            disabled={isSaving}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                handleSave();
              } else if (e.key === 'Escape') {
                handleCancel();
              }
            }}
          />
          <Button
            variant="ghost"
            size="sm"
            onClick={handleSave}
            disabled={isSaving}
            className="h-7 w-7 p-0"
            title="保存"
          >
            <Check className="h-3 w-3" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleCancel}
            disabled={isSaving}
            className="h-7 w-7 p-0"
            title="キャンセル"
          >
            <X className="h-3 w-3" />
          </Button>
        </div>
        {error && (
          <p className="text-xs text-destructive">{error}</p>
        )}
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2">
      {currentAlias ? (
        <>
          <Badge variant="secondary" className="text-xs px-2 py-0.5">
            {currentAlias}
          </Badge>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleStartEdit}
            disabled={isLoading || isSaving}
            className="h-6 w-6 p-0"
            title="エイリアスを編集"
          >
            <Edit2 className="h-3 w-3" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleDelete}
            disabled={isLoading || isSaving}
            className="h-6 w-6 p-0"
            title="エイリアスを削除"
          >
            <Trash2 className="h-3 w-3" />
          </Button>
        </>
      ) : (
        <Button
          variant="ghost"
          size="sm"
          onClick={handleStartEdit}
          disabled={isLoading || isSaving}
          className="h-6 w-6 p-0"
          title="エイリアスを追加"
        >
          <Plus className="h-3 w-3" />
        </Button>
      )}
    </div>
  );
}