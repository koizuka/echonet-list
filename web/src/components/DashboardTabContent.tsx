import React from 'react';
import { DashboardCard } from './DashboardCard';
import { TooltipProvider } from '@/components/ui/tooltip';
import { getDashboardDevicesGroupedByLocation, getLocationDisplayName, sortLocationIds, splitLocationsBySepar } from '@/libs/locationHelper';
import { arrangeDashboardDevices, isPlaceholder } from '@/libs/dashboardLayoutHelper';
import { useDashboardCardExpansion } from '@/hooks/useDashboardCardExpansion';
import { cn } from '@/libs/utils';
import { isJapanese } from '@/libs/languageHelper';
import type { Device, PropertyDescriptionData, DeviceAlias, LocationSettings } from '@/hooks/types';

interface DashboardTabContentProps {
  devices: Record<string, Device>;
  aliases: DeviceAlias;
  propertyDescriptions: Record<string, PropertyDescriptionData>;
  locationSettings: LocationSettings;
  onPropertyChange: (target: string, epc: string, value: { string: string }) => Promise<void>;
  isConnected: boolean;
  onSelectTab?: (tabId: string) => void;
}

export function DashboardTabContent({
  devices,
  aliases,
  propertyDescriptions,
  locationSettings,
  onPropertyChange,
  isConnected,
  onSelectTab
}: DashboardTabContentProps) {
  const { isExpanded, toggleExpansion } = useDashboardCardExpansion();
  const groupedDevices = getDashboardDevicesGroupedByLocation(devices);
  const locationIds = sortLocationIds(Object.keys(groupedDevices), locationSettings);

  // Split locations into groups based on separator positions
  const locationGroups = splitLocationsBySepar(locationIds, locationSettings.order);

  // Pre-compute global indices for alternating backgrounds (avoids mutable variable in render)
  const globalIndexMap = new Map<string, number>();
  let idx = 0;
  for (const group of locationGroups) {
    for (const locationId of group) {
      globalIndexMap.set(locationId, idx++);
    }
  }

  if (locationIds.length === 0) {
    return (
      <div className="text-center text-muted-foreground py-8" data-testid="dashboard-empty">
        No devices found.
      </div>
    );
  }

  return (
    <TooltipProvider delayDuration={300}>
      <div
        className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 items-start pb-16 safe-area-bottom"
        data-testid="dashboard-content"
      >
        {locationGroups.map((group, groupIndex) => {
          const isLastGroup = groupIndex === locationGroups.length - 1;
          const groupElements = group.map((locationId) => {
            const locationDevices = groupedDevices[locationId];
            const locationName = getLocationDisplayName(locationId, devices, propertyDescriptions, locationSettings);

            const arranged = arrangeDashboardDevices(locationDevices);
            const firstIsPlaceholder = arranged.length > 0 && isPlaceholder(arranged[0]);

            const locationLabelClassName = "text-sm font-semibold font-display text-muted-foreground/80 uppercase tracking-wide px-1 md:px-0 translate-y-1.5 md:translate-y-0";
            const buttonLabel = isJapanese() ? `${locationName} タブを開く` : `Open ${locationName} tab`;
            const locationLabel = onSelectTab ? (
              <button
                type="button"
                onClick={() => onSelectTab(locationId)}
                className={cn(locationLabelClassName, "cursor-pointer hover:text-primary transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 rounded")}
                aria-label={buttonLabel}
                title={buttonLabel}
              >
                {locationName}
              </button>
            ) : (
              <h3 className={locationLabelClassName}>
                {locationName}
              </h3>
            );

            const globalIndex = globalIndexMap.get(locationId) ?? 0;

            return (
              <div
                key={locationId}
                className={cn(
                  // Mobile: left accent line + alternating background
                  'border-l-2 border-l-primary/40 pl-2',
                  globalIndex % 2 === 1 && 'bg-muted/60',
                  // Tablet+: reset mobile styles and apply card container
                  'md:border-l-0 md:pl-0 md:bg-card',
                  'md:rounded-lg md:border md:p-3 md:shadow-sm'
                )}
                data-testid={`dashboard-location-${locationId}`}
              >
                {/* Location header - intentionally duplicated for responsive layout:
                    - This div: visible on PC (md:block), hidden on mobile when placeholder exists
                    - Placeholder span below: visible on mobile (md:hidden), hidden on PC
                    Only one is visible at a time based on viewport */}
                <div
                  className={cn("mb-2", firstIsPlaceholder && "hidden md:block")}
                  data-testid={`location-label-pc-${locationId}`}
                >
                  {locationLabel}
                </div>

                {/* Device grid within location */}
                <div className="grid grid-cols-2 gap-2">
                  {arranged.map((item, itemIndex) => {
                    if (isPlaceholder(item)) {
                      // First placeholder: show label on mobile only, keep empty placeholder on PC
                      if (itemIndex === 0) {
                        return (
                          <div
                            key={`label-${locationId}`}
                            className="flex items-start"
                            data-testid={`location-label-mobile-${locationId}`}
                          >
                            <span className="md:hidden">{locationLabel}</span>
                          </div>
                        );
                      }
                      return <div key={`placeholder-${itemIndex}`} />;
                    }
                    const deviceKey = `${item.ip} ${item.eoj}`;
                    return (
                      <DashboardCard
                        key={deviceKey}
                        device={item}
                        onPropertyChange={onPropertyChange}
                        propertyDescriptions={propertyDescriptions}
                        devices={devices}
                        aliases={aliases}
                        isConnected={isConnected}
                        isExpanded={isExpanded(deviceKey)}
                        onToggleExpand={() => toggleExpansion(deviceKey)}
                      />
                    );
                  })}
                </div>
              </div>
            );
          });

          // Return group elements with separator between groups
          // Show separator even for empty groups to help users notice configuration issues
          return (
            <React.Fragment key={`group-${groupIndex}`}>
              {groupElements}
              {!isLastGroup && (
                <div className="col-span-full my-2">
                  <div className="h-px bg-border" />
                </div>
              )}
            </React.Fragment>
          );
        })}
      </div>
    </TooltipProvider>
  );
}
