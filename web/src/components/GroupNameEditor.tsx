import { useState, useRef, useEffect } from 'react';
import { Check, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { validateGroupName } from '@/libs/groupHelper';
import { getCurrentLocale } from '@/libs/languageHelper';

const messages = {
  en: { placeholder: 'Enter group name', save: 'Save', cancel: 'Cancel' },
  ja: { placeholder: 'グループ名を入力', save: '保存', cancel: 'キャンセル' },
};

interface GroupNameEditorProps {
  groupName: string;
  existingGroups: string[];
  onSave: (groupName: string) => void;
  onCancel: () => void;
  isLoading?: boolean;
  isConnected?: boolean;
}

export function GroupNameEditor({
  groupName,
  existingGroups,
  onSave,
  onCancel,
  isLoading = false,
  isConnected = true,
}: GroupNameEditorProps) {
  // Remove '@' prefix from initial value if present
  const [inputValue, setInputValue] = useState(groupName.startsWith('@') ? groupName.slice(1) : groupName);
  const [error, setError] = useState<string | undefined>();
  const [isComposing, setIsComposing] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const texts = messages[getCurrentLocale()];

  // Focus input when component mounts
  useEffect(() => {
    if (inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, []);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    let newValue = e.target.value;
    
    // Remove leading '@' if user types it
    if (newValue.startsWith('@')) {
      newValue = newValue.slice(1);
    }
    
    setInputValue(newValue);
    
    // Validate the new value with '@' prefix
    const fullGroupName = '@' + newValue;
    const validationError = validateGroupName(fullGroupName, existingGroups);
    setError(validationError);
  };

  const getIsSaveDisabled = () => {
    // Disabled while loading or disconnected
    if (isLoading || !isConnected) return true;
    
    // Check validation with '@' prefix
    const fullGroupName = '@' + inputValue;
    const validationError = validateGroupName(fullGroupName, existingGroups);
    if (validationError) return true;
    
    // Check if value has changed
    if (fullGroupName === groupName) return true;
    
    return false;
  };

  const handleSave = () => {
    if (!getIsSaveDisabled()) {
      // Save with '@' prefix
      onSave('@' + inputValue);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !isComposing) {
      if (!getIsSaveDisabled()) {
        handleSave();
      }
    } else if (e.key === 'Escape') {
      onCancel();
    }
  };

  return (
    <div className="space-y-2 max-w-sm">
      <div className="flex gap-2">
        <div className="flex items-center gap-1">
          <span className="text-xs font-medium text-muted-foreground">@</span>
          <Input
            ref={inputRef}
            value={inputValue}
            onChange={handleInputChange}
            placeholder={texts.placeholder}
            className="h-7 text-xs flex-1"
            disabled={isLoading || !isConnected}
            onCompositionStart={() => setIsComposing(true)}
            onCompositionEnd={() => setIsComposing(false)}
            onKeyDown={handleKeyDown}
          />
        </div>
        <div className="flex items-center gap-1 flex-shrink-0">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleSave}
            disabled={getIsSaveDisabled()}
            className="h-7 w-7 p-0"
            aria-label={texts.save}
            title={texts.save}
          >
            <Check className="h-3 w-3" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={onCancel}
            disabled={isLoading || !isConnected}
            className="h-7 w-7 p-0"
            aria-label={texts.cancel}
            title={texts.cancel}
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