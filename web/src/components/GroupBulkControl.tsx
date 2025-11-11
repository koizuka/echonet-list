import { useState, useMemo, useRef } from 'react';
import { Button } from '@/components/ui/button';
import { Power, PowerOff } from 'lucide-react';
import { isOperationStatusSettable } from '@/libs/propertyHelper';
import { generateLogEntryId } from '@/libs/idHelper';
import type { Device, PropertyValue } from '@/hooks/types';
import type { LogEntry } from '@/hooks/useLogNotifications';

type GroupBulkControlProps = {
  devices: Device[];
  onPropertyChange: (target: string, epc: string, value: PropertyValue) => Promise<void>;
  addLogEntry?: (log: LogEntry) => void;
  resolveAlias?: (ip: string, eoj: string) => string | null;
};

export function GroupBulkControl({ devices, onPropertyChange, addLogEntry, resolveAlias: _resolveAlias }: GroupBulkControlProps) {
  const [isOperating, setIsOperating] = useState(false);
  const isOperatingRef = useRef(false);

  // Filter devices that support operation status control (EPC 0x80)
  const controllableDevices = useMemo(
    () => devices.filter(device => isOperationStatusSettable(device)),
    [devices]
  );

  const hasControllableDevices = controllableDevices.length > 0;

  const handleBulkPowerControl = async (powerState: 'on' | 'off') => {
    // Use ref for robust race condition protection
    if (isOperatingRef.current || !hasControllableDevices) return;

    isOperatingRef.current = true;
    setIsOperating(true);

    try {
      // Execute all operations in parallel, continue even if some fail
      const results = await Promise.allSettled(
        controllableDevices.map(device => {
          const target = `${device.ip} ${device.eoj}`;
          return onPropertyChange(target, '80', { string: powerState });
        })
      );

      // Count successes and failures
      const successCount = results.filter(r => r.status === 'fulfilled').length;
      const failureCount = results.filter(r => r.status === 'rejected').length;

      // Add notification if there are any results
      if (addLogEntry && (successCount > 0 || failureCount > 0)) {
        const actionText = powerState === 'on' ? 'ON' : 'OFF';
        let message: string;
        let level: 'INFO' | 'WARN' | 'ERROR';

        if (failureCount === 0) {
          // All succeeded
          message = `${successCount}台のデバイスを${actionText}にしました`;
          level = 'INFO';
        } else if (successCount === 0) {
          // All failed
          message = `${failureCount}台のデバイスを${actionText}にできませんでした`;
          level = 'ERROR';
        } else {
          // Partial success
          message = `${successCount}/${controllableDevices.length}台のデバイスを${actionText}にしました（${failureCount}台失敗）`;
          level = 'WARN';
        }

        addLogEntry({
          id: generateLogEntryId('bulk_control'),
          level,
          message,
          time: new Date().toISOString(),
          attributes: {
            component: 'GroupBulkControl',
            powerState,
            successCount,
            failureCount,
            totalDevices: controllableDevices.length
          },
          isRead: false
        });
      }
    } finally {
      isOperatingRef.current = false;
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
