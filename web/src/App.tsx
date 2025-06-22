import { usePropertyDescriptions } from '@/hooks/usePropertyDescriptions';
import { getAllTabs, getDevicesForTab as getDevicesForTabHelper, hasAnyOperationalDevice, hasAnyFaultyDevice, translateLocationId } from '@/libs/locationHelper';
import { useCardExpansion } from '@/hooks/useCardExpansion';
import { usePersistedTab } from '@/hooks/usePersistedTab';
import { useAutoReconnect } from '@/hooks/useAutoReconnect';
import { DeviceCard } from '@/components/DeviceCard';
import { useLogNotifications } from '@/hooks/useLogNotifications';
import { NotificationBell } from '@/components/NotificationBell';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ExpandIcon, ShrinkIcon } from 'lucide-react';
import { useState } from 'react';
import type { PropertyValue, LogNotification } from '@/hooks/types';
import type { LogEntry } from '@/hooks/useLogNotifications';

function App() {
  // 開発環境と本番環境でWebSocket URLを切り替え
  const wsUrl = import.meta.env.DEV 
    ? (import.meta.env.VITE_WS_URL || 'wss://localhost:8080/ws')  // 開発時は環境変数またはデフォルト値
    : `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`; // 本番時は現在のホストを使用
  
  // Track the latest log notification
  const [latestLogNotification, setLatestLogNotification] = useState<LogNotification | undefined>();
  
  // Log notification state
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);

  // Log notification handlers
  const logManager = useLogNotifications({ 
    notification: latestLogNotification,
    onLogsChange: (newLogs, newUnreadCount) => {
      setLogs(newLogs);
      setUnreadCount(newUnreadCount);
    }
  });

  const echonet = usePropertyDescriptions(wsUrl, (message) => {
    // Handle log notifications
    if (message.type === 'log_notification') {
      setLatestLogNotification(message);
    }
  }, () => {
    // Clear WebSocket connection errors when successfully connected
    logManager.clearByCategory('WebSocket');
  });
  const cardExpansion = useCardExpansion();
  
  // Auto-reconnect when page/browser becomes active
  useAutoReconnect({
    connectionState: echonet.connectionState,
    connect: echonet.connect,
  });
  
  // Get all tab IDs (location IDs + groups)
  const tabIds = getAllTabs(echonet.devices, echonet.groups);
  
  // Use persistent tab selection
  const { selectedTab, selectTab } = usePersistedTab(tabIds, 'All');
  
  // Loading state for update operations
  const [updatingDevices, setUpdatingDevices] = useState<Set<string>>(new Set());
  // Track alias operations loading state
  const [aliasLoading, setAliasLoading] = useState(false);

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

  // Add alias handler
  const handleAddAlias = async (alias: string, target: string) => {
    try {
      setAliasLoading(true);
      await echonet.addAlias(alias, target);
    } catch (error) {
      console.error('Failed to add alias:', error);
      throw error; // Re-throw to let AliasEditor handle the error
    } finally {
      setAliasLoading(false);
    }
  };

  // Delete alias handler
  const handleDeleteAlias = async (alias: string) => {
    try {
      setAliasLoading(true);
      await echonet.deleteAlias(alias);
    } catch (error) {
      console.error('Failed to delete alias:', error);
      throw error; // Re-throw to let AliasEditor handle the error
    } finally {
      setAliasLoading(false);
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

  // Function to get devices for a specific tab ID
  const getDevicesForTab = (tabId: string) => {
    return getDevicesForTabHelper(tabId, echonet.devices, echonet.groups);
  };

  // Get all device keys for expand/collapse all functionality
  const allDeviceKeys = Object.keys(echonet.devices).map(key => {
    const device = echonet.devices[key];
    return `${device.ip} ${device.eoj}`;
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
                  data-testid="expand-all-button"
                >
                  <ExpandIcon className="h-3 w-3 sm:mr-1" />
                  <span className="hidden sm:inline ml-1">Expand All</span>
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => cardExpansion.collapseAll()}
                  className="h-7 sm:h-8 px-2 sm:px-3"
                  data-testid="collapse-all-button"
                >
                  <ShrinkIcon className="h-3 w-3 sm:mr-1" />
                  <span className="hidden sm:inline ml-1">Collapse All</span>
                </Button>
              </div>
            )}
            <Badge variant="outline" className={`${getConnectionColor(echonet.connectionState)} text-white text-xs`} data-testid="connection-status">
              {echonet.connectionState}
            </Badge>
            
            {/* Notification Bell */}
            <NotificationBell
              logs={logs}
              unreadCount={unreadCount}
              onMarkAllAsRead={logManager.markAllAsRead}
              onClearAll={logManager.clearAllLogs}
            />
          </div>
        </div>
        

        {Object.keys(echonet.devices).length === 0 ? (
          <Card>
            <CardContent className="pt-6">
              <p className="text-center text-muted-foreground">
                {echonet.connectionState === 'connected' 
                  ? (echonet.initialStateReceived 
                      ? 'No devices found. Click refresh to discover devices.'
                      : 'サーバーから初期情報が受信できていません'
                    )
                  : `サーバーに接続できません (${echonet.connectionState})`
                }
              </p>
            </CardContent>
          </Card>
        ) : (
          <Tabs value={selectedTab} onValueChange={selectTab} className="w-full" data-testid="device-tabs">
            <div className="w-full mb-4">
              <TabsList className="w-full h-auto p-2 bg-muted flex flex-wrap justify-between gap-2">
              {tabIds.map((tabId) => {
                const tabDevices = getDevicesForTab(tabId);
                const hasOperationalDevice = hasAnyOperationalDevice(tabDevices);
                const hasFaultyDevice = hasAnyFaultyDevice(tabDevices);
                const displayName = translateLocationId(tabId);
                return (
                  <TabsTrigger 
                    key={tabId} 
                    value={tabId} 
                    className="data-[state=active]:bg-primary data-[state=active]:text-primary-foreground data-[state=active]:border-primary border-2 border-muted-foreground/30 bg-background px-2 sm:px-3 py-1.5 sm:py-2 text-xs sm:text-sm rounded-lg"
                    data-testid={`tab-${tabId}`}
                  >
                    <div className="flex items-center gap-1">
                      {tabId !== 'All' && (
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
                      <span>{displayName}</span>
                      {tabId !== 'All' && (
                        <span className="ml-1 text-[10px] sm:text-xs">({tabDevices.length})</span>
                      )}
                    </div>
                  </TabsTrigger>
                );
              })}
              </TabsList>
            </div>
            
            {tabIds.map((tabId) => (
              <TabsContent key={tabId} value={tabId} className="space-y-4" data-testid={`tab-content-${tabId}`}>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-6 gap-3 sm:gap-4">
                  {getDevicesForTab(tabId).map((device) => {
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
                        onAddAlias={handleAddAlias}
                        onDeleteAlias={handleDeleteAlias}
                        isAliasLoading={aliasLoading}
                      />
                    );
                  })}
                </div>
                
                {getDevicesForTab(tabId).length === 0 && (
                  <Card>
                    <CardContent className="pt-6">
                      <p className="text-center text-muted-foreground">
                        {tabId === 'All' 
                          ? 'No devices found.' 
                          : tabId.startsWith('@') 
                            ? `No devices found in group ${tabId}.`
                            : `No devices found in ${translateLocationId(tabId)}.`
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