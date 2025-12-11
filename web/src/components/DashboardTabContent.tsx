import { DashboardCard } from './DashboardCard';
import { getDashboardDevicesGroupedByLocation, getLocationDisplayName } from '@/libs/locationHelper';
import { arrangeDashboardDevices, isPlaceholder } from '@/libs/dashboardLayoutHelper';
import { useDashboardCardExpansion } from '@/hooks/useDashboardCardExpansion';
import type { Device, PropertyDescriptionData, DeviceAlias } from '@/hooks/types';

interface DashboardTabContentProps {
  devices: Record<string, Device>;
  aliases: DeviceAlias;
  propertyDescriptions: Record<string, PropertyDescriptionData>;
  onPropertyChange: (target: string, epc: string, value: { string: string }) => Promise<void>;
  isConnected: boolean;
}

export function DashboardTabContent({
  devices,
  aliases,
  propertyDescriptions,
  onPropertyChange,
  isConnected
}: DashboardTabContentProps) {
  const { isExpanded, toggleExpansion } = useDashboardCardExpansion();
  const groupedDevices = getDashboardDevicesGroupedByLocation(devices);
  const locationIds = Object.keys(groupedDevices).sort();

  if (locationIds.length === 0) {
    return (
      <div className="text-center text-muted-foreground py-8" data-testid="dashboard-empty">
        No devices found.
      </div>
    );
  }

  return (
    <div
      className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 items-start pb-16 safe-area-bottom"
      data-testid="dashboard-content"
    >
      {locationIds.map(locationId => {
        const locationDevices = groupedDevices[locationId];
        const locationName = getLocationDisplayName(locationId, devices, propertyDescriptions);

        const arranged = arrangeDashboardDevices(locationDevices);
        const firstIsPlaceholder = arranged.length > 0 && isPlaceholder(arranged[0]);

        const locationLabel = (
          <h3 className="text-sm font-semibold font-display text-muted-foreground/80 uppercase tracking-wide px-1 md:px-0">
            {locationName}
          </h3>
        );

        return (
          <div
            key={locationId}
            className="md:rounded-lg md:border md:bg-card md:p-3 md:shadow-sm"
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
  );
}
