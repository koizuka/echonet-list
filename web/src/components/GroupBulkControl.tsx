import { useState, useMemo } from 'react';
import { Button } from '@/components/ui/button';
import { Power, PowerOff } from 'lucide-react';
import { isOperationStatusSettable } from '@/libs/propertyHelper';
import type { Device, PropertyValue } from '@/hooks/types';

type GroupBulkControlProps = {
  devices: Device[];
  onPropertyChange: (target: string, epc: string, value: PropertyValue) => Promise<void>;
};

export function GroupBulkControl({ devices, onPropertyChange }: GroupBulkControlProps) {
  const [isOperating, setIsOperating] = useState(false);

  // Filter devices that support operation status control (EPC 0x80)
  const controllableDevices = useMemo(
    () => devices.filter(device => isOperationStatusSettable(device)),
    [devices]
  );

  const hasControllableDevices = controllableDevices.length > 0;

  const handleBulkPowerControl = async (powerState: 'on' | 'off') => {
    if (isOperating || !hasControllableDevices) return;

    setIsOperating(true);

    try {
      // Execute all operations in parallel, continue even if some fail
      await Promise.allSettled(
        controllableDevices.map(device => {
          const target = `${device.ip} ${device.eoj}`;
          return onPropertyChange(target, '80', { string: powerState });
        })
      );
    } finally {
      setIsOperating(false);
    }
  };

  return (
    <>
      <Button
        onClick={() => handleBulkPowerControl('on')}
        disabled={!hasControllableDevices || isOperating}
        variant="outline"
        size="sm"
        className="flex items-center gap-2"
        aria-label="すべてのデバイスをONにする"
      >
        <Power className="h-4 w-4" />
        すべてON
      </Button>
      <Button
        onClick={() => handleBulkPowerControl('off')}
        disabled={!hasControllableDevices || isOperating}
        variant="outline"
        size="sm"
        className="flex items-center gap-2"
        aria-label="すべてのデバイスをOFFにする"
      >
        <PowerOff className="h-4 w-4" />
        すべてOFF
      </Button>
    </>
  );
}
