import { useState } from 'react';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { getPropertyName, extractClassCodeFromEOJ, decodePropertyMap } from '@/libs/propertyHelper';
import type { Device, PropertyDescriptionData } from '@/hooks/types';

interface PropertyMapViewerProps {
  device: Device;
  propertyDescriptions: Record<string, PropertyDescriptionData>;
}

interface PropertyMapData {
  epc: string;
  description: string;
  propertyCount: number;
  properties: Array<{ epc: string; description: string }>;
}

export function PropertyMapViewer({ device, propertyDescriptions }: PropertyMapViewerProps) {
  const [expandedMaps, setExpandedMaps] = useState<Set<string>>(new Set());

  const classCode = extractClassCodeFromEOJ(device.eoj);

  // Parse property map from EDT
  const parsePropertyMap = (epc: string): PropertyMapData | null => {
    const propertyMap = device.properties[epc];
    if (!propertyMap?.EDT) {
      return null;
    }

    const epcs = decodePropertyMap(propertyMap.EDT);
    if (!epcs) {
      return null;
    }

    const properties = epcs.map(propertyEpc => ({
      epc: propertyEpc,
      description: getPropertyName(propertyEpc, propertyDescriptions, classCode)
    }));

    // Get property count from the original EDT data
    let propertyCount = epcs.length;
    try {
      const mapBytes = atob(propertyMap.EDT);
      if (mapBytes.length > 0) {
        propertyCount = mapBytes.charCodeAt(0);
      }
    } catch {
      // Use the decoded EPC count as fallback
    }

    const mapDescription = getPropertyName(epc, propertyDescriptions, classCode);

    return {
      epc,
      description: mapDescription,
      propertyCount,
      properties,
    };
  };

  // Property map EPCs in order of display preference
  const propertyMapEpcs = ['9D', '9E', '9F'] as const;
  const propertyMaps = propertyMapEpcs
    .map(epc => parsePropertyMap(epc))
    .filter((map): map is PropertyMapData => map !== null);

  if (propertyMaps.length === 0) {
    return null;
  }

  const toggleExpanded = (epc: string) => {
    const newExpanded = new Set(expandedMaps);
    if (newExpanded.has(epc)) {
      newExpanded.delete(epc);
    } else {
      newExpanded.add(epc);
    }
    setExpandedMaps(newExpanded);
  };

  const getMapTitle = (epc: string): string => {
    return getPropertyName(epc, propertyDescriptions, classCode);
  };

  return (
    <div className="space-y-2">
      {propertyMaps.map((map) => {
        const isExpanded = expandedMaps.has(map.epc);
        const title = getMapTitle(map.epc);

        return (
          <div key={map.epc} className="border rounded-lg">
            <Button
              variant="ghost"
              onClick={() => toggleExpanded(map.epc)}
              className="w-full justify-start p-3 h-auto font-normal"
            >
              <div className="flex items-center justify-between w-full">
                <div className="flex items-center gap-2">
                  {isExpanded ? (
                    <ChevronDown className="h-4 w-4" />
                  ) : (
                    <ChevronRight className="h-4 w-4" />
                  )}
                  <span className="text-sm font-medium">
                    {title} ({map.propertyCount})
                  </span>
                </div>
              </div>
            </Button>
            
            {isExpanded && (
              <div className="px-3 pb-3 space-y-1">
                {map.properties.map((property) => (
                  <div
                    key={property.epc}
                    className="flex items-center gap-2 text-sm text-muted-foreground pl-6"
                  >
                    <span className="font-mono text-xs bg-muted px-1 py-0.5 rounded">
                      {property.epc}
                    </span>
                    <span>{property.description}</span>
                  </div>
                ))}
                {map.properties.length === 0 && (
                  <div className="text-sm text-muted-foreground pl-6">
                    No properties in this map
                  </div>
                )}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}