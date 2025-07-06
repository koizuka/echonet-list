import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { HexViewer } from './HexViewer';
import { formatPropertyValueWithTranslation, shouldShowHexViewer, decodePropertyMap, getPropertyName, extractClassCodeFromEOJ } from '@/libs/propertyHelper';
import { getCurrentLocale } from '@/libs/languageHelper';
import type { PropertyValue, PropertyDescriptor, PropertyDescriptionData, Device } from '@/hooks/types';

interface PropertyDisplayProps {
  currentValue: PropertyValue;
  descriptor?: PropertyDescriptor;
  epc: string;
  propertyDescriptions?: Record<string, PropertyDescriptionData>;
  device?: Device;
}

export function PropertyDisplay({ 
  currentValue, 
  descriptor, 
  epc,
  propertyDescriptions,
  device
}: PropertyDisplayProps) {
  const [showPropertyMap, setShowPropertyMap] = useState(false);
  const currentLang = getCurrentLocale();
  const canShowHexViewer = shouldShowHexViewer(currentValue, descriptor, currentLang);
  const isPropertyMap = ['9D', '9E', '9F'].includes(epc);
  
  // Parse property map if applicable
  const parsePropertyMap = () => {
    if (!isPropertyMap || !currentValue.EDT || !propertyDescriptions || !device) return null;
    
    const epcs = decodePropertyMap(currentValue.EDT);
    if (!epcs) return null;
    
    const classCode = extractClassCodeFromEOJ(device.eoj);
    const properties = epcs.map(epc => ({
      epc,
      description: getPropertyName(epc, propertyDescriptions, classCode, currentLang)
    }));
    
    // Get property count from the original EDT data
    try {
      const mapBytes = atob(currentValue.EDT);
      const propertyCount = mapBytes.length > 0 ? mapBytes.charCodeAt(0) : 0;
      return { propertyCount, properties };
    } catch {
      return { propertyCount: epcs.length, properties };
    }
  };
  
  const formattedValue = formatPropertyValueWithTranslation(currentValue, descriptor, epc, currentLang);
  
  if (isPropertyMap && propertyDescriptions && currentValue.EDT) {
    const mapData = parsePropertyMap();
    if (mapData) {
      return (
        <div className="relative">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">
              {formattedValue} ({mapData.propertyCount})
            </span>
            <Button
              variant={showPropertyMap ? "default" : "outline"}
              size="sm"
              onClick={() => setShowPropertyMap(!showPropertyMap)}
              className="h-6 w-6 p-0"
              title={showPropertyMap ? "Hide property details" : "Show property details"}
            >
              {showPropertyMap ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
            </Button>
            <HexViewer 
              canShowHexViewer={canShowHexViewer} 
              currentValue={currentValue}
            />
          </div>
          {showPropertyMap && (
            <div className="absolute top-full left-0 right-0 mt-1 bg-background border rounded shadow-md z-20 min-w-max">
              <div className="p-2 space-y-1">
                {mapData.properties.map((property) => (
                  <div
                    key={property.epc}
                    className="flex items-center gap-2 text-sm"
                  >
                    <span className="font-mono text-xs bg-muted px-1 py-0.5 rounded">
                      {property.epc}
                    </span>
                    <span>{property.description}</span>
                  </div>
                ))}
                {mapData.properties.length === 0 && (
                  <div className="text-sm text-muted-foreground">
                    No properties in this map
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      );
    }
  }
  
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