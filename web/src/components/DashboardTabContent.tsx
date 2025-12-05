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
      className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 items-start"
      data-testid="dashboard-content"
    >
      {locationIds.map(locationId => {
        const locationDevices = groupedDevices[locationId];
        const locationName = getLocationDisplayName(locationId, devices, propertyDescriptions);

        return (
          <div
            key={locationId}
            className="md:rounded-lg md:border md:bg-card md:p-3 md:shadow-sm"
            data-testid={`dashboard-location-${locationId}`}
          >
            {/* Location header */}
            <h3 className="text-sm font-semibold text-muted-foreground mb-2 px-1 md:px-0">
              {locationName}
            </h3>

            {/* Device grid within location */}
            <div className="grid grid-cols-2 gap-2">
              {arrangeDashboardDevices(locationDevices).map((item, index) => {
                if (isPlaceholder(item)) {
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
