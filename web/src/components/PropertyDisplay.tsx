import { HexViewer } from './HexViewer';
import { PropertyMapDisplay } from './PropertyMapDisplay';
import { SelfNodeInstanceListSDisplay } from './SelfNodeInstanceListSDisplay';
import { formatPropertyValue, shouldShowHexViewer, decodeInstanceList } from '@/libs/propertyHelper';
import { getCurrentLocale } from '@/libs/languageHelper';
import type { PropertyValue, PropertyDescriptor, PropertyDescriptionData, Device, DeviceAlias } from '@/hooks/types';

interface PropertyDisplayProps {
  currentValue: PropertyValue;
  descriptor?: PropertyDescriptor;
  epc: string;
  propertyDescriptions?: Record<string, PropertyDescriptionData>;
  device?: Device;
  allDevices?: Record<string, Device>;
  aliases?: DeviceAlias;
  getDeviceClassCode?: (device: Device) => string;
  isCompact?: boolean;
}

export function PropertyDisplay({ 
  currentValue, 
  descriptor, 
  epc,
  propertyDescriptions,
  device,
  allDevices,
  aliases,
  getDeviceClassCode,
  isCompact = false
}: PropertyDisplayProps) {
  const currentLang = getCurrentLocale();
  const canShowHexViewer = shouldShowHexViewer(currentValue, descriptor, currentLang);
  const isPropertyMap = ['9D', '9E', '9F'].includes(epc);
  
  // Check if this is NodeProfile SelfNodeInstanceListS  
  // NodeProfile class code is 0x0ef0, so check for various formats
  const isNodeProfile = device && device.eoj.startsWith('0EF0:');
  const isSelfNodeInstanceListS = isNodeProfile && (epc === 'D6');
  
  
  // Handle SelfNodeInstanceListS special display
  if (isSelfNodeInstanceListS) {
    // If we have all required props, show the full display
    if (allDevices && aliases && getDeviceClassCode) {
      return (
        <SelfNodeInstanceListSDisplay
          currentValue={currentValue}
          device={device}
          allDevices={allDevices}
          aliases={aliases}
          propertyDescriptions={propertyDescriptions || {}}
          getDeviceClassCode={getDeviceClassCode}
          isCompact={isCompact}
        />
      );
    } else {
      // Fallback: just show the count with hex viewer
      const instances = currentValue.EDT ? decodeInstanceList(currentValue.EDT) : null;
      const instanceCount = instances ? instances.length : 0;
      return (
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">
            Instance List ({instanceCount})
          </span>
          <HexViewer 
            canShowHexViewer={true} 
            currentValue={currentValue}
          />
        </div>
      );
    }
  }
  
  // Handle Property Map display
  if (isPropertyMap && propertyDescriptions && currentValue.EDT) {
    return (
      <PropertyMapDisplay
        currentValue={currentValue}
        descriptor={descriptor}
        propertyDescriptions={propertyDescriptions}
        device={device}
      />
    );
  }
  
  // Default display
  const formattedValue = formatPropertyValue(currentValue, descriptor, currentLang);
  
  return (
    <div className="relative">
      <div className="flex items-center gap-2">
        <span className="text-sm font-medium">
          {formattedValue}
        </span>
        <HexViewer 
          canShowHexViewer={canShowHexViewer} 
          currentValue={currentValue}
        />
      </div>
    </div>
  );
}