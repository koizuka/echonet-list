import { DashboardCard } from './DashboardCard';
import { TooltipProvider } from '@/components/ui/tooltip';
import { getDashboardDevicesGroupedByLocation, getLocationDisplayName, sortLocationIds } from '@/libs/locationHelper';
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
        {locationIds.map((locationId, index) => {
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

        return (
          <div
            key={locationId}
            className={cn(
              // Mobile: left accent line + alternating background
              'border-l-2 border-l-primary/40 pl-2',
              index % 2 === 1 && 'bg-muted/60',
              // Tablet+: reset mobile styles and apply card container
              'md:border-l-0 md:pl-0 md:bg-card',
              'md:rounded-lg md:border md:p-3 md:shadow-sm'
            )}
            data-testid={`dashboard-location-${locationId}`}
          >
            {/* Location header - only show outside grid if first item is not a placeholder */}
            {!firstIsPlaceholder && (
              <div className="mb-2">{locationLabel}</div>
            )}

            {/* Device grid within location */}
            <div className="grid grid-cols-2 gap-2">
              {arranged.map((item, index) => {
                if (isPlaceholder(item)) {
                  // First placeholder becomes the location label
                  if (index === 0) {
                    return (
                      <div key={`label-${locationId}`} className="flex items-start">
                        {locationLabel}
                      </div>
                    );
                  }
                  return <div key={`placeholder-${index}`} />;
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
        })}
      </div>
    </TooltipProvider>
  );
}
