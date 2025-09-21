import { useState } from 'react';
import { PropertySwitchControl } from './PropertyEditControls/PropertySwitchControl';
import { PropertySelectControl } from './PropertyEditControls/PropertySelectControl';
import { PropertyInputControl } from './PropertyEditControls/PropertyInputControl';
import { PropertySliderControl } from './PropertyEditControls/PropertySliderControl';
import { PropertyDisplay } from './PropertyDisplay';
import { HexViewer } from './HexViewer';
import { isPropertySettable, formatPropertyValue, shouldShowHexViewer } from '@/libs/propertyHelper';
import { getCurrentLocale } from '@/libs/languageHelper';
import { shouldUseImmediateSlider } from '@/libs/deviceTypeHelper';
import type { PropertyDescriptor, PropertyValue, Device, PropertyDescriptionData, DeviceAlias } from '@/hooks/types';
import type { LogEntry } from '@/hooks/useLogNotifications';

/**
 * Determines if a property is settable based on various conditions
 *
 * @param params - Object containing all the necessary parameters for settability check
 * @returns true if the property can be edited
 */
function determinePropertySettability({
  descriptor,
  hasNumberDesc,
  hasAliases,
  epc,
  device,
  useImmediateSlider,
  isConnected
}: {
  descriptor?: PropertyDescriptor;
  hasNumberDesc: boolean;
  hasAliases: boolean;
  epc: string;
  device: Device;
  useImmediateSlider: boolean;
  isConnected?: boolean;
}): boolean {
  // 1. Property descriptor indicates it's settable (stringSettable, numberDesc, or aliases)
  const hasEditCapability = descriptor?.stringSettable || hasNumberDesc || hasAliases;

  // 2. Property is listed in Set Property Map (EPC 0x9E) OR is an immediate slider property
  const isInSetPropertyMap = isPropertySettable(epc, device);

  // 3. WebSocket connection is active (defaults to connected if not specified)
  const isConnectionActive = isConnected !== false; // Default to true if not specified

  return hasEditCapability && (isInSetPropertyMap || useImmediateSlider) && isConnectionActive;
}

interface PropertyEditorProps {
  device: Device;
  epc: string;
  currentValue: PropertyValue;
  descriptor?: PropertyDescriptor;
  onPropertyChange: (target: string, epc: string, value: PropertyValue) => Promise<void>;
  propertyDescriptions?: Record<string, PropertyDescriptionData>;
  isConnected?: boolean;
  allDevices?: Record<string, Device>;
  aliases?: DeviceAlias;
  getDeviceClassCode?: (device: Device) => string;
  isCompact?: boolean;
  onError?: (error: LogEntry) => void;
}

export function PropertyEditor({
  device,
  epc,
  currentValue,
  descriptor,
  onPropertyChange,
  propertyDescriptions,
  isConnected,
  allDevices,
  aliases,
  getDeviceClassCode,
  isCompact = false,
  onError
}: PropertyEditorProps) {
  
  const deviceId = `${device.ip} ${device.eoj}`;

  const hasAliases = descriptor?.aliases && Object.keys(descriptor.aliases).length > 0;
  const hasNumberDesc = descriptor?.numberDesc;
  const hasStringDesc = descriptor?.stringDesc;
  const currentLang = getCurrentLocale();
  const canShowHexViewer = shouldShowHexViewer(currentValue, descriptor, currentLang);

  // Check if this property should use immediate slider control
  const classCode = device.eoj.split(':')[0];
  const useImmediateSlider = hasNumberDesc && shouldUseImmediateSlider(epc, classCode);

  // Check if property is settable
  const isSettable = determinePropertySettability({
    descriptor,
    hasNumberDesc: !!hasNumberDesc,
    hasAliases: !!hasAliases,
    epc,
    device,
    useImmediateSlider: !!useImmediateSlider,
    isConnected
  });

  // For component use, we need these values separately
  const isConnectionActive = isConnected !== false;



  // Handle alias selection
  const [isLoading, setIsLoading] = useState(false);
  const [isInputEditing, setIsInputEditing] = useState(false);

  // Check if this property has exactly 'on' and 'off' aliases (for switch UI)
  const hasOnOffAliases = hasAliases && descriptor?.aliases &&
    Object.keys(descriptor.aliases).length === 2 &&
    'on' in descriptor.aliases && 'off' in descriptor.aliases;
  
  const handleAliasSelect = async (aliasName: string) => {
    if (!descriptor?.aliases) return;
    
    setIsLoading(true);
    try {
      await onPropertyChange(deviceId, epc, { string: aliasName });
    } catch (error) {
      console.error('Failed to set property:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleInputSave = async (value: PropertyValue) => {
    await onPropertyChange(deviceId, epc, value);
  };

  // For read-only properties, use PropertyDisplay component
  // Exception: immediate slider properties should always be editable regardless of Set Property Map
  if (!isSettable && !useImmediateSlider) {
    return (
      <PropertyDisplay
        currentValue={currentValue}
        descriptor={descriptor}
        epc={epc}
        propertyDescriptions={propertyDescriptions}
        device={device}
        allDevices={allDevices}
        aliases={aliases}
        getDeviceClassCode={getDeviceClassCode}
        isCompact={isCompact}
      />
    );
  }

  return (
    <div className="flex items-center gap-2">
      {/* Switch for properties with only on/off aliases */}
      {hasOnOffAliases && (
        <PropertySwitchControl
          value={currentValue.string || 'off'}
          onChange={handleAliasSelect}
          disabled={isLoading || !isConnectionActive}
          testId={`operation-status-switch-${epc}`}
        />
      )}
      
      {/* Alias select - not for on/off properties */}
      {hasAliases && !hasOnOffAliases && (
        <PropertySelectControl
          value={currentValue.string || ''}
          aliases={descriptor.aliases!}
          aliasTranslations={descriptor.aliasTranslations}
          onChange={handleAliasSelect}
          disabled={isLoading || !isConnectionActive}
          testId={`alias-select-trigger-${epc}`}
        />
      )}

      {/* Immediate Slider for specific properties */}
      {useImmediateSlider && (
        <div className="relative">
          <div className="flex items-center gap-2">
            <PropertySliderControl
              currentValue={currentValue}
              descriptor={descriptor}
              onSave={handleInputSave}
              disabled={isLoading || !isConnectionActive}
              testId={epc}
              onError={onError}
            />
            <HexViewer
              canShowHexViewer={canShowHexViewer}
              currentValue={currentValue}
            />
          </div>
        </div>
      )}

      {/* String/Number editing (traditional mode) */}
      {(hasStringDesc || hasNumberDesc) && !useImmediateSlider && (
        <div className="relative">
          <div className="flex items-center gap-2">
            {!hasAliases && !isInputEditing && (
              <span className="text-sm font-medium">
                {formatPropertyValue(currentValue, descriptor, currentLang)}
              </span>
            )}
            <PropertyInputControl
              currentValue={currentValue}
              descriptor={descriptor}
              onSave={handleInputSave}
              disabled={isLoading || !isConnectionActive}
              testId={epc}
              onEditModeChange={setIsInputEditing}
            />
            <HexViewer
              canShowHexViewer={canShowHexViewer}
              currentValue={currentValue}
            />
          </div>
        </div>
      )}
    </div>
  );
}