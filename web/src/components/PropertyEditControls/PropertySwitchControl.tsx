import { Switch } from '@/components/ui/switch';

interface PropertySwitchControlProps {
  value: string;
  onChange: (value: string) => void;
  disabled: boolean;
  testId?: string;
}

export function PropertySwitchControl({ value, onChange, disabled, testId }: PropertySwitchControlProps) {
  return (
    <div className="inline-flex items-center py-2 px-1">
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