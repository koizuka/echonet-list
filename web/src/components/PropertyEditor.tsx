import { useState } from 'react';
import { PropertySwitchControl } from './PropertyEditControls/PropertySwitchControl';
import { PropertySelectControl } from './PropertyEditControls/PropertySelectControl';
import { PropertyInputControl } from './PropertyEditControls/PropertyInputControl';
import { PropertyDisplay } from './PropertyDisplay';
import { HexViewer } from './HexViewer';
import { isPropertySettable, formatPropertyValueWithTranslation, shouldShowHexViewer } from '@/libs/propertyHelper';
import { translateLocationId } from '@/libs/locationHelper';
import { getCurrentLocale } from '@/libs/languageHelper';
import type { PropertyDescriptor, PropertyValue, Device, PropertyDescriptionData } from '@/hooks/types';

interface PropertyEditorProps {
  device: Device;
  epc: string;
  currentValue: PropertyValue;
  descriptor?: PropertyDescriptor;
  onPropertyChange: (target: string, epc: string, value: PropertyValue) => Promise<void>;
  propertyDescriptions?: Record<string, PropertyDescriptionData>;
  isConnected?: boolean;
}

export function PropertyEditor({ 
  device, 
  epc, 
  currentValue, 
  descriptor, 
  onPropertyChange,
  propertyDescriptions,
  isConnected 
}: PropertyEditorProps) {
  const deviceId = `${device.ip} ${device.eoj}`;

  const hasAliases = descriptor?.aliases && Object.keys(descriptor.aliases).length > 0;
  const hasNumberDesc = descriptor?.numberDesc;
  const hasStringDesc = descriptor?.stringDesc;
  const currentLang = getCurrentLocale();
  const canShowHexViewer = shouldShowHexViewer(currentValue, descriptor, currentLang);
  
  // Check if property is settable based on:
  // 1. Property descriptor indicates it's settable (stringSettable, numberDesc, or aliases)
  // 2. Property is listed in Set Property Map (EPC 0x9E)
  // 3. WebSocket connection is active (defaults to connected if not specified)
  const hasEditCapability = descriptor?.stringSettable || hasNumberDesc || hasAliases;
  const isInSetPropertyMap = isPropertySettable(epc, device);
  const isConnectionActive = isConnected !== false; // Default to true if not specified
  const isSettable = hasEditCapability && isInSetPropertyMap && isConnectionActive;
  
  // Check if this is Installation Location property (EPC 0x81)
  const isInstallationLocation = epc === '81';
  
  // Check if this property has exactly 'on' and 'off' aliases (for switch UI)
  const hasOnOffAliases = hasAliases && descriptor?.aliases && 
    Object.keys(descriptor.aliases).length === 2 &&
    'on' in descriptor.aliases && 'off' in descriptor.aliases;

  // Handle alias selection
  const [isLoading, setIsLoading] = useState(false);
  const [isInputEditing, setIsInputEditing] = useState(false);
  
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
  if (!isSettable) {
    return (
      <PropertyDisplay
        currentValue={currentValue}
        descriptor={descriptor}
        epc={epc}
        propertyDescriptions={propertyDescriptions}
        device={device}
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
          isInstallationLocation={isInstallationLocation}
          testId={`alias-select-trigger-${epc}`}
        />
      )}

      {/* String/Number editing */}
      {(hasStringDesc || hasNumberDesc) && (
        <div className="relative">
          <div className="flex items-center gap-2">
            {!hasAliases && !isInputEditing && (
              <span className="text-sm font-medium">
                {formatPropertyValueWithTranslation(currentValue, descriptor, epc, translateLocationId, currentLang)}
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