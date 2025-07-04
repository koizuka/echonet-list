import { 
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { translateLocationId } from '@/libs/locationHelper';
import { getCurrentLocale } from '@/libs/languageHelper';

interface PropertySelectControlProps {
  value: string;
  aliases: Record<string, string>;
  aliasTranslations?: Record<string, Record<string, string>>;
  onChange: (value: string) => void;
  disabled: boolean;
  isInstallationLocation?: boolean;
  testId?: string;
}

export function PropertySelectControl({ 
  value, 
  aliases, 
  aliasTranslations,
  onChange, 
  disabled, 
  isInstallationLocation = false,
  testId 
}: PropertySelectControlProps) {
  const currentLang = getCurrentLocale();
  
  // Function to get display text for an alias
  const getDisplayText = (aliasName: string) => {
    if (isInstallationLocation) {
      return translateLocationId(aliasName);
    }
    
    // Use translation if available and not English
    if (aliasTranslations && currentLang !== 'en' && aliasTranslations[currentLang]?.[aliasName]) {
      return aliasTranslations[currentLang][aliasName];
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
        <SelectValue>
          {value ? getDisplayText(value) : 'Select...'}
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