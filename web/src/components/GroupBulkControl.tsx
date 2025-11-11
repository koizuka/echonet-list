import { useState, useMemo, useRef } from 'react';
import { Button } from '@/components/ui/button';
import { Power, PowerOff } from 'lucide-react';
import { isOperationStatusSettable } from '@/libs/propertyHelper';
import { generateLogEntryId } from '@/libs/idHelper';
import { getCurrentLocale } from '@/libs/languageHelper';
import type { Device, PropertyValue } from '@/hooks/types';
import type { LogEntry } from '@/hooks/useLogNotifications';

type GroupBulkControlProps = {
  devices: Device[];
  onPropertyChange: (target: string, epc: string, value: PropertyValue) => Promise<void>;
  addLogEntry?: (log: LogEntry) => void;
};

// Localized message templates for bulk control operations
const BULK_CONTROL_MESSAGES = {
  partial_failure: {
    en: 'Turned {powerState} {successCount}/{totalDevices} devices ({failureCount} failed)',
    ja: '{successCount}/{totalDevices}台のデバイスを{powerState}にしました（{failureCount}台失敗）'
  },
  all_failed: {
    en: 'Failed to turn {powerState} {failureCount} devices',
    ja: '{failureCount}台のデバイスを{powerState}にできませんでした'
  },
  power_state: {
    on: {
      en: 'ON',
      ja: 'ON'
    },
    off: {
      en: 'OFF',
      ja: 'OFF'
    }
  },
  tooltip: {
    operating: {
      en: 'Operating...',
      ja: '操作中...'
    },
    no_controllable_devices: {
      en: 'No controllable devices',
      ja: '操作可能なデバイスがありません'
    }
  },
  button_label: {
    all_on: {
      en: 'All ON',
      ja: 'すべてON'
    },
    all_off: {
      en: 'All OFF',
      ja: 'すべてOFF'
    }
  }
} as const;

export function GroupBulkControl({ devices, onPropertyChange, addLogEntry }: GroupBulkControlProps) {
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

      // Add notification only if there are failures (partial or complete failure)
      if (addLogEntry && failureCount > 0) {
        const locale = getCurrentLocale();
        const powerStateText = BULK_CONTROL_MESSAGES.power_state[powerState][locale];
        let message: string;
        let level: 'WARN' | 'ERROR';

        if (successCount === 0) {
          // All failed
          const template = BULK_CONTROL_MESSAGES.all_failed[locale];
          message = template
            .replace('{powerState}', powerStateText)
            .replace('{failureCount}', String(failureCount));
          level = 'ERROR';
        } else {
          // Partial success
          const template = BULK_CONTROL_MESSAGES.partial_failure[locale];
          message = template
            .replace('{powerState}', powerStateText)
            .replace('{successCount}', String(successCount))
            .replace('{totalDevices}', String(controllableDevices.length))
            .replace('{failureCount}', String(failureCount));
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

  const locale = getCurrentLocale();

  const getButtonTitle = () => {
    if (isOperating) return BULK_CONTROL_MESSAGES.tooltip.operating[locale];
    if (!hasControllableDevices) return BULK_CONTROL_MESSAGES.tooltip.no_controllable_devices[locale];
    return undefined;
  };

  return (
    <>
      <Button
        onClick={() => handleBulkPowerControl('on')}
        disabled={!hasControllableDevices || isOperating}
        variant="outline"
        size="sm"
        className="flex items-center gap-2"
        aria-label={BULK_CONTROL_MESSAGES.button_label.all_on[locale]}
        title={getButtonTitle()}
      >
        <Power className="h-4 w-4" />
        {BULK_CONTROL_MESSAGES.button_label.all_on[locale]}
      </Button>
      <Button
        onClick={() => handleBulkPowerControl('off')}
        disabled={!hasControllableDevices || isOperating}
        variant="outline"
        size="sm"
        className="flex items-center gap-2"
        aria-label={BULK_CONTROL_MESSAGES.button_label.all_off[locale]}
        title={getButtonTitle()}
      >
        <PowerOff className="h-4 w-4" />
        {BULK_CONTROL_MESSAGES.button_label.all_off[locale]}
      </Button>
    </>
  );
}
