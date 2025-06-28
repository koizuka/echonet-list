import { Switch } from '@/components/ui/switch';

interface PropertySwitchControlProps {
  value: string;
  onChange: (value: string) => void;
  disabled: boolean;
  testId?: string;
}

export function PropertySwitchControl({ value, onChange, disabled, testId }: PropertySwitchControlProps) {
  return (
    <Switch
      checked={value === 'on'}
      onCheckedChange={(checked) => onChange(checked ? 'on' : 'off')}
      disabled={disabled}
      data-testid={testId}
      className="data-[state=checked]:bg-green-600 data-[state=unchecked]:bg-gray-400"
    />
  );
}