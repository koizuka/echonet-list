import { PropertyEditor } from './PropertyEditor';
import { PropertyDisplay } from './PropertyDisplay';
import { getPropertyName, getPropertyDescriptor, isPropertySettable } from '@/libs/propertyHelper';
import type { Device, PropertyValue, PropertyDescriptionData } from '@/hooks/types';

interface PropertyRowProps {
  device: Device;
  epc: string;
  value: PropertyValue;
  isCompact: boolean;
  onPropertyChange: (target: string, epc: string, value: PropertyValue) => Promise<void>;
  propertyDescriptions: Record<string, PropertyDescriptionData>;
  classCode: string;
}

export function PropertyRow({
  device,
  epc,
  value,
  isCompact,
  onPropertyChange,
  propertyDescriptions,
  classCode
}: PropertyRowProps) {
  const propertyName = getPropertyName(epc, propertyDescriptions, classCode);
  const propertyDescriptor = getPropertyDescriptor(epc, propertyDescriptions, classCode);
  
  // Check if property is settable
  const hasEditCapability = propertyDescriptor?.stringSettable || 
    propertyDescriptor?.numberDesc || 
    (propertyDescriptor?.aliases && Object.keys(propertyDescriptor.aliases).length > 0);
  const isInSetPropertyMap = propertyDescriptor && isPropertySettable(epc, device);
  const isSettable = hasEditCapability && isInSetPropertyMap;

  if (isCompact) {
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
              />
            ) : (
              <PropertyDisplay
                currentValue={value}
                descriptor={propertyDescriptor}
                epc={epc}
                propertyDescriptions={propertyDescriptions}
                device={device}
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
          />
        </div>
      </div>
    </div>
  );
}