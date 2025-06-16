import { usePropertyDescriptions } from '@/hooks/usePropertyDescriptions';
import { getAllTabs, getDevicesForTab as getDevicesForTabHelper, hasAnyOperationalDevice, hasAnyFaultyDevice } from '@/libs/locationHelper';
import { useCardExpansion } from '@/hooks/useCardExpansion';
import { usePersistedTab } from '@/hooks/usePersistedTab';
import { useAutoReconnect } from '@/hooks/useAutoReconnect';
import { DeviceCard } from '@/components/DeviceCard';
import { LogNotifications } from '@/components/LogNotifications';
import { NotificationBell } from '@/components/NotificationBell';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ExpandIcon, ShrinkIcon } from 'lucide-react';
import { useState } from 'react';
import type { PropertyValue, LogNotification } from '@/hooks/types';
import type { LogEntry } from '@/components/LogNotifications';

function App() {
  // 開発環境と本番環境でWebSocket URLを切り替え
  const wsUrl = import.meta.env.DEV 
    ? (import.meta.env.VITE_WS_URL || 'wss://localhost:8080/ws')  // 開発時は環境変数またはデフォルト値
    : `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`; // 本番時は現在のホストを使用
  
  const echonet = usePropertyDescriptions(wsUrl, (message) => {
    // Handle log notifications
    if (message.type === 'log_notification') {
      setLatestLogNotification(message);
    }
  });
  const cardExpansion = useCardExpansion();
  
  // Auto-reconnect when page/browser becomes active
  useAutoReconnect({
    connectionState: echonet.connectionState,
    connect: echonet.connect,
  });
  
  // Get all tabs (locations + groups)
  const tabs = getAllTabs(echonet.devices, echonet.aliases, echonet.groups);
  
  // Use persistent tab selection
  const { selectedTab, selectTab } = usePersistedTab(tabs, 'All');
  
  // Loading state for update operations
  const [updatingDevices, setUpdatingDevices] = useState<Set<string>>(new Set());
  
  // Track the latest log notification
  const [latestLogNotification, setLatestLogNotification] = useState<LogNotification | undefined>();
  
  // Log notification state
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);

  // Property change handler
  const handlePropertyChange = async (target: string, epc: string, value: PropertyValue) => {
    try {
      await echonet.setDeviceProperties(target, { [epc]: value });
    } catch (error) {
      console.error('Failed to change property:', error);
      // TODO: Show user-friendly error message
    }
  };

  // Update properties handler
  const handleUpdateProperties = async (target: string) => {
    try {
      // Add device to updating set
      setUpdatingDevices(prev => new Set([...prev, target]));
      
      console.log('Updating properties for:', target);
      await echonet.updateDeviceProperties([target]);
    } catch (error) {
      console.error('Failed to update properties:', error);
      // TODO: Show user-friendly error message
    } finally {
      // Remove device from updating set
      setUpdatingDevices(prev => {
        const newSet = new Set(prev);
        newSet.delete(target);
        return newSet;
      });
    }
  };

  const getConnectionColor = (state: string) => {
    switch (state) {
      case 'connected':
        return 'bg-green-500';
      case 'connecting':
        return 'bg-yellow-500';
      case 'error':
        return 'bg-red-500';
      default:
        return 'bg-gray-500';
    }
  };

  // Function to get devices for a specific tab
  const getDevicesForTab = (tabName: string) => {
    return getDevicesForTabHelper(tabName, echonet.devices, echonet.aliases, echonet.groups);
  };

  // Get all device keys for expand/collapse all functionality
  const allDeviceKeys = Object.keys(echonet.devices).map(key => {
    const device = echonet.devices[key];
    return `${device.ip} ${device.eoj}`;
  });

  // Log notification handlers
  const logManager = LogNotifications({ 
    notification: latestLogNotification,
    onLogsChange: (newLogs, newUnreadCount) => {
      setLogs(newLogs);
      setUnreadCount(newUnreadCount);
    }
  });

  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="container mx-auto p-4">
        <div className="flex justify-between items-center mb-4 sm:mb-6">
          <h1 className="text-2xl sm:text-3xl font-bold">ECHONET List</h1>
          <div className="flex items-center gap-1 sm:gap-2">
            {/* Expand/Collapse All Controls */}
            {Object.keys(echonet.devices).length > 0 && (
              <div className="flex items-center gap-1">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => cardExpansion.expandAll(allDeviceKeys)}
                  className="h-7 sm:h-8 px-2 sm:px-3"
                >
                  <ExpandIcon className="h-3 w-3 sm:mr-1" />
                  <span className="hidden sm:inline ml-1">Expand All</span>
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => cardExpansion.collapseAll()}
                  className="h-7 sm:h-8 px-2 sm:px-3"
                >
                  <ShrinkIcon className="h-3 w-3 sm:mr-1" />
                  <span className="hidden sm:inline ml-1">Collapse All</span>
                </Button>
              </div>
            )}
            <Badge variant="outline" className={`${getConnectionColor(echonet.connectionState)} text-white text-xs`}>
              {echonet.connectionState}
            </Badge>
            
            {/* Notification Bell */}
            <NotificationBell
              logs={logs}
              unreadCount={unreadCount}
              onMarkAsRead={logManager.markAsRead}
              onMarkAllAsRead={logManager.markAllAsRead}
              onClearAll={logManager.clearAllLogs}
            />
          </div>
        </div>
        
        {echonet.error && (
          <Card className="mb-4 border-red-500">
            <CardHeader>
              <CardTitle className="text-red-500">Error</CardTitle>
            </CardHeader>
            <CardContent>
              <p>{echonet.error.message}</p>
            </CardContent>
          </Card>
        )}

        {Object.keys(echonet.devices).length === 0 && echonet.connectionState === 'connected' ? (
          <Card>
            <CardContent className="pt-6">
              <p className="text-center text-muted-foreground">
                No devices found. Click refresh to discover devices.
              </p>
            </CardContent>
          </Card>
        ) : (
          <Tabs value={selectedTab} onValueChange={selectTab} className="w-full">
            <div className="w-full mb-4">
              <TabsList className="w-full h-auto p-2 bg-muted flex flex-wrap justify-between gap-2">
              {tabs.map((tab) => {
                const tabDevices = getDevicesForTab(tab);
                const hasOperationalDevice = hasAnyOperationalDevice(tabDevices);
                const hasFaultyDevice = hasAnyFaultyDevice(tabDevices);
                return (
                  <TabsTrigger 
                    key={tab} 
                    value={tab} 
                    className="data-[state=active]:bg-primary data-[state=active]:text-primary-foreground data-[state=active]:border-primary border-2 border-muted-foreground/30 bg-background px-2 sm:px-3 py-1.5 sm:py-2 text-xs sm:text-sm rounded-lg"
                  >
                    <div className="flex items-center gap-1">
                      {tab !== 'All' && (
                        <div className="flex items-center gap-1">
                          <div 
                            className={`w-2 h-2 rounded-full ${
                              hasOperationalDevice ? 'bg-green-500' : 'bg-gray-400'
                            }`}
                            title={`Power Status: ${hasOperationalDevice ? 'At least one device is ON' : 'All devices are OFF or no devices'}`}
                          />
                          {hasFaultyDevice && (
                            <div 
                              className="w-2 h-2 rounded-full bg-red-500"
                              title="At least one device has a fault"
                            />
                          )}
                        </div>
                      )}
                      <span>{tab}</span>
                      {tab !== 'All' && (
                        <span className="ml-1 text-[10px] sm:text-xs">({tabDevices.length})</span>
                      )}
                    </div>
                  </TabsTrigger>
                );
              })}
              </TabsList>
            </div>
            
            {tabs.map((tab) => (
              <TabsContent key={tab} value={tab} className="space-y-4">
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-6 gap-3 sm:gap-4">
                  {getDevicesForTab(tab).map((device) => {
                    const deviceKey = `${device.ip} ${device.eoj}`;
                    
                    return (
                      <DeviceCard
                        key={deviceKey}
                        device={device}
                        isExpanded={cardExpansion.isCardExpanded(deviceKey)}
                        onToggleExpansion={() => cardExpansion.toggleCard(deviceKey)}
                        onPropertyChange={handlePropertyChange}
                        onUpdateProperties={handleUpdateProperties}
                        isUpdating={updatingDevices.has(deviceKey)}
                        propertyDescriptions={echonet.propertyDescriptions}
                        getDeviceClassCode={echonet.getDeviceClassCode}
                        devices={echonet.devices}
                        aliases={echonet.aliases}
                      />
                    );
                  })}
                </div>
                
                {getDevicesForTab(tab).length === 0 && (
                  <Card>
                    <CardContent className="pt-6">
                      <p className="text-center text-muted-foreground">
                        {tab === 'All' 
                          ? 'No devices found.' 
                          : tab.startsWith('@') 
                            ? `No devices found in group ${tab}.`
                            : `No devices found in ${tab}.`
                        }
                      </p>
                    </CardContent>
                  </Card>
                )}
              </TabsContent>
            ))}
          </Tabs>
        )}
      </div>
    </div>
  );
}

export default App;