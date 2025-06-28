import { 
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { translateLocationId } from '@/libs/locationHelper';

interface PropertySelectControlProps {
  value: string;
  aliases: Record<string, string>;
  onChange: (value: string) => void;
  disabled: boolean;
  isInstallationLocation?: boolean;
  testId?: string;
}

export function PropertySelectControl({ 
  value, 
  aliases, 
  onChange, 
  disabled, 
  isInstallationLocation = false,
  testId 
}: PropertySelectControlProps) {
  return (
    <Select
      value={value || ''}
      onValueChange={onChange}
      disabled={disabled}
    >
      <SelectTrigger className="h-7 w-[120px]" data-testid={testId}>
        <SelectValue>
          {value ? 
            (isInstallationLocation ? translateLocationId(value) : value) 
            : 'Select...'}
        </SelectValue>
      </SelectTrigger>
      <SelectContent data-testid={testId ? `${testId}-content` : undefined}>
        {Object.keys(aliases).map((aliasName) => (
          <SelectItem key={aliasName} value={aliasName} data-testid={`alias-option-${aliasName}`}>
            {isInstallationLocation ? translateLocationId(aliasName) : aliasName}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}