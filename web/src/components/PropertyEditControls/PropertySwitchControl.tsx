import { Switch } from '@/components/ui/switch';
import { cn } from '@/libs/utils';

interface PropertySwitchControlProps {
  value: string;
  onChange: (value: string) => void;
  disabled: boolean;
  testId?: string;
  compact?: boolean;
}

export function PropertySwitchControl({ value, onChange, disabled, testId, compact = false }: PropertySwitchControlProps) {
  return (
    <div className={cn('inline-flex items-center px-1', compact ? 'py-0' : 'py-2')}>
      <Switch
        checked={value === 'on'}
        onCheckedChange={(checked) => onChange(checked ? 'on' : 'off')}
        disabled={disabled}
        data-testid={testId}
        className="data-[state=checked]:bg-green-600 data-[state=unchecked]:bg-gray-400 touch-none"
      />
    </div>
  );
}