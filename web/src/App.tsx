import { usePropertyDescriptions } from '@/hooks/usePropertyDescriptions';
import { getAllTabs, getDevicesForTab as getDevicesForTabHelper } from '@/libs/locationHelper';
import { useCardExpansion } from '@/hooks/useCardExpansion';
import { usePersistedTab } from '@/hooks/usePersistedTab';
import { DeviceCard } from '@/components/DeviceCard';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ExpandIcon, ShrinkIcon } from 'lucide-react';
import type { PropertyValue } from '@/hooks/types';

function App() {
  // 開発環境と本番環境でWebSocket URLを切り替え
  const wsUrl = import.meta.env.DEV 
    ? 'wss://localhost:8080/ws'  // 開発時は直接接続
    : `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`; // 本番時は現在のホストを使用
  
  const echonet = usePropertyDescriptions(wsUrl);
  const cardExpansion = useCardExpansion();
  
  // Get all tabs (locations + groups)
  const tabs = getAllTabs(echonet.devices, echonet.aliases, echonet.groups);
  
  // Use persistent tab selection
  const { selectedTab, selectTab } = usePersistedTab(tabs, 'All');

  // Property change handler
  const handlePropertyChange = async (target: string, epc: string, value: PropertyValue) => {
    try {
      await echonet.setDeviceProperties(target, { [epc]: value });
    } catch (error) {
      console.error('Failed to change property:', error);
      // TODO: Show user-friendly error message
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
            <div className="w-full mb-4 overflow-x-auto">
              <TabsList className="w-max min-w-full h-auto p-1 bg-muted flex flex-nowrap justify-start gap-1 sm:flex-wrap sm:w-full">
              {tabs.map((tab) => (
                <TabsTrigger 
                  key={tab} 
                  value={tab} 
                  className="data-[state=active]:bg-primary data-[state=active]:text-primary-foreground px-2 sm:px-3 py-1.5 sm:py-2 text-xs sm:text-sm whitespace-nowrap shrink-0"
                >
                  <span className="hidden sm:inline">{tab}</span>
                  <span className="sm:hidden">{tab.length > 6 ? tab.substring(0, 6) + '...' : tab}</span>
                  {tab !== 'All' && (
                    <span className="ml-1 hidden sm:inline">({getDevicesForTab(tab).length})</span>
                  )}
                </TabsTrigger>
              ))}
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