import { SimpleDeviceCard } from '@/components/SimpleDeviceCard';
import { HexViewer } from '@/components/HexViewer';
import { decodeInstanceList } from '@/libs/propertyHelper';
import type { PropertyValue, Device, DeviceAlias, PropertyDescriptionData } from '@/hooks/types';
import { getCurrentLocale } from '@/libs/languageHelper';

const messages = {
  en: {
    invalidData: 'Invalid instance list data',
    deviceNotFound: 'Device not found',
    instanceList: 'Instance List',
  },
  ja: {
    invalidData: 'インスタンスリストのデータが不正です',
    deviceNotFound: 'デバイスが見つかりません',
    instanceList: 'インスタンスリスト',
  },
};

interface SelfNodeInstanceListSDisplayProps {
  currentValue: PropertyValue;
  device: Device;
  allDevices: Record<string, Device>;
  aliases: DeviceAlias;
  propertyDescriptions: Record<string, PropertyDescriptionData>;
  getDeviceClassCode: (device: Device) => string;
  isCompact?: boolean;
}

export function SelfNodeInstanceListSDisplay({
  currentValue,
  device,
  allDevices,
  aliases,
  propertyDescriptions,
  getDeviceClassCode,
  isCompact = false,
}: SelfNodeInstanceListSDisplayProps) {
  const texts = messages[getCurrentLocale()];

  // Decode the instance list from EDT
  const instances = currentValue.EDT ? decodeInstanceList(currentValue.EDT) : null;
  
  if (!instances) {
    return (
      <div className="relative">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">
            {texts.invalidData}
          </span>
          <HexViewer 
            canShowHexViewer={true} 
            currentValue={currentValue}
          />
        </div>
      </div>
    );
  }
  
  // Create device keys by combining IP with EOJ
  const deviceCards = instances.map((instance, index) => {
    const eoj = `${instance.classCode}:${parseInt(instance.instanceCode, 16)}`;
    const deviceKey = `${device.ip} ${eoj}`;
    const targetDevice = allDevices[deviceKey];
    
    if (!targetDevice) {
      // Device not found in current devices list
      // Add debugging information for device key format issues
      if (process.env.NODE_ENV === 'development') {
        console.debug(`Device not found: expected "${deviceKey}", available keys:`, Object.keys(allDevices).slice(0, 3));
      }
      return (
        <div key={index} className={`p-2 border rounded text-muted-foreground ${
          isCompact ? "text-xs" : "text-sm"
        }`}>
          <div className="font-mono">{eoj}</div>
          <div className={isCompact ? "text-xs" : "text-sm"}>{texts.deviceNotFound}</div>
        </div>
      );
    }
    
    return (
      <SimpleDeviceCard
        key={deviceKey}
        deviceKey={deviceKey}
        device={targetDevice}
        allDevices={allDevices}
        aliases={aliases}
        propertyDescriptions={propertyDescriptions}
        getDeviceClassCode={getDeviceClassCode}
        isDraggable={false}
        className={isCompact ? "text-xs w-32 flex-shrink-0" : ""}
        isCompact={isCompact}
      />
    );
  });
  
  return (
    <div className={isCompact ? "space-y-1" : "space-y-2"}>
      {/* Show header with count and hex viewer only in full mode */}
      {!isCompact && (
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">
            {texts.instanceList} ({instances.length})
          </span>
          <HexViewer 
            canShowHexViewer={true} 
            currentValue={currentValue}
          />
        </div>
      )}
      <div className={isCompact 
        ? "flex flex-wrap gap-1 justify-end" 
        : "grid gap-2 grid-cols-1"
      }>
        {deviceCards}
      </div>
    </div>
  );
}