import { PropertyEditor } from './PropertyEditor';
import { PropertyDisplay } from './PropertyDisplay';
import { getPropertyName, getPropertyDescriptor, isPropertySettable } from '@/libs/propertyHelper';
import { isSensorProperty, getSensorIcon, getSensorIconColor } from '@/libs/sensorPropertyHelper';
import { getCurrentLocale } from '@/libs/languageHelper';
import type { Device, PropertyValue, PropertyDescriptionData, DeviceAlias } from '@/hooks/types';

interface PropertyRowProps {
  device: Device;
  epc: string;
  value: PropertyValue;
  isCompact: boolean;
  onPropertyChange: (target: string, epc: string, value: PropertyValue) => Promise<void>;
  propertyDescriptions: Record<string, PropertyDescriptionData>;
  classCode: string;
  isConnected?: boolean;
  allDevices?: Record<string, Device>;
  aliases?: DeviceAlias;
  getDeviceClassCode?: (device: Device) => string;
}

export function PropertyRow({
  device,
  epc,
  value,
  isCompact,
  onPropertyChange,
  propertyDescriptions,
  classCode,
  isConnected,
  allDevices,
  aliases,
  getDeviceClassCode
}: PropertyRowProps) {
  
  const currentLang = getCurrentLocale();
  const propertyName = getPropertyName(epc, propertyDescriptions, classCode, currentLang);
  const propertyDescriptor = getPropertyDescriptor(epc, propertyDescriptions, classCode, currentLang);
  
  // Check if property is settable
  const hasEditCapability = propertyDescriptor?.stringSettable || 
    propertyDescriptor?.numberDesc || 
    (propertyDescriptor?.aliases && Object.keys(propertyDescriptor.aliases).length > 0);
  const isInSetPropertyMap = propertyDescriptor && isPropertySettable(epc, device);
  const isSettable = hasEditCapability && isInSetPropertyMap;

  // Check if this is a sensor property for special compact display
  const isSensor = isSensorProperty(classCode, epc);
  const SensorIcon = getSensorIcon(classCode, epc);
  const sensorIconColor = getSensorIconColor(classCode, epc, value);

  if (isCompact && isSensor && SensorIcon) {
    // Sensor properties in compact mode: icon + value only
    return (
      <div className="inline-flex items-center gap-1 text-xs mr-3 mb-1" title={propertyName}>
        <SensorIcon className={`h-3 w-3 ${sensorIconColor}`} />
        <div>
          {isSettable ? (
            <PropertyEditor
              device={device}
              epc={epc}
              currentValue={value}
              descriptor={propertyDescriptor}
              onPropertyChange={onPropertyChange}
              propertyDescriptions={propertyDescriptions}
              isConnected={isConnected}
              allDevices={allDevices}
              aliases={aliases}
              getDeviceClassCode={getDeviceClassCode}
              isCompact={isCompact}
            />
          ) : (
            <PropertyDisplay
              currentValue={value}
              descriptor={propertyDescriptor}
              epc={epc}
              propertyDescriptions={propertyDescriptions}
              device={device}
              allDevices={allDevices}
              aliases={aliases}
              getDeviceClassCode={getDeviceClassCode}
              isCompact={isCompact}
            />
          )}
        </div>
      </div>
    );
  }

  if (isCompact) {
    // Non-sensor properties in compact mode: traditional label + value display
    return (
      <div className="text-xs relative">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
          <span className="font-medium text-muted-foreground">
            {propertyName}:
          </span>
          <div className="ml-auto">
            {isSettable ? (
              <PropertyEditor
                device={device}
                epc={epc}
                currentValue={value}
                descriptor={propertyDescriptor}
                onPropertyChange={onPropertyChange}
                propertyDescriptions={propertyDescriptions}
                isConnected={isConnected}
                allDevices={allDevices}
                aliases={aliases}
                getDeviceClassCode={getDeviceClassCode}
                isCompact={isCompact}
              />
            ) : (
              <PropertyDisplay
                currentValue={value}
                descriptor={propertyDescriptor}
                epc={epc}
                propertyDescriptions={propertyDescriptions}
                device={device}
                allDevices={allDevices}
                aliases={aliases}
                getDeviceClassCode={getDeviceClassCode}
                isCompact={isCompact}
              />
            )}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-1 relative">
      <div className="flex flex-wrap items-start gap-x-2 gap-y-1">
        <span className="text-sm font-medium text-muted-foreground">
          {propertyName}:
        </span>
        <div className="ml-auto">
          <PropertyEditor
            device={device}
            epc={epc}
            currentValue={value}
            descriptor={propertyDescriptor}
            onPropertyChange={onPropertyChange}
            propertyDescriptions={propertyDescriptions}
            isConnected={isConnected}
            allDevices={allDevices}
            aliases={aliases}
            getDeviceClassCode={getDeviceClassCode}
            isCompact={isCompact}
          />
        </div>
      </div>
    </div>
  );
}