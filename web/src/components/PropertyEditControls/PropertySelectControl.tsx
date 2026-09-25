import { 
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { getCurrentLocale } from '@/libs/languageHelper';

const messages = {
  en: { placeholder: 'Select...' },
  ja: { placeholder: '選択...' },
};
import type { AliasTranslations } from '@/hooks/types';

interface PropertySelectControlProps {
  value: string;
  aliases: Record<string, string>;
  aliasTranslations?: AliasTranslations;
  onChange: (value: string) => void;
  disabled: boolean;
  testId?: string;
}

export function PropertySelectControl({ 
  value, 
  aliases, 
  aliasTranslations,
  onChange, 
  disabled, 
  testId 
}: PropertySelectControlProps) {
  const currentLang = getCurrentLocale();
  
  // Function to get display text for an alias
  const getDisplayText = (aliasName: string) => {
    // Use translation if available and not English
    if (aliasTranslations && currentLang !== 'en') {
      const translation = aliasTranslations[aliasName];
      if (translation) {
        return translation;
      }
    }
    
    return aliasName;
  };
  
  return (
    <Select
      value={value || ''}
      onValueChange={onChange}
      disabled={disabled}
    >
      <SelectTrigger className="h-7 w-[120px]" data-testid={testId}>
        {/* Radix renders `placeholder` (not children) while the value is empty */}
        <SelectValue placeholder={messages[currentLang].placeholder}>
          {value ? getDisplayText(value) : null}
        </SelectValue>
      </SelectTrigger>
      <SelectContent data-testid={testId ? `${testId}-content` : undefined}>
        {Object.keys(aliases).map((aliasName) => (
          <SelectItem key={aliasName} value={aliasName} data-testid={`alias-option-${aliasName}`}>
            {getDisplayText(aliasName)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}