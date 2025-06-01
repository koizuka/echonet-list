import { usePropertyDescriptions } from '@/hooks/usePropertyDescriptions';
import { getPropertyName, formatPropertyValue, getPropertyDescriptor } from '@/libs/propertyHelper';
import { getAllTabs, getDevicesForTab as getDevicesForTabHelper } from '@/libs/locationHelper';
import { deviceHasAlias } from '@/libs/deviceIdHelper';
import { PropertyEditor } from '@/components/PropertyEditor';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import type { PropertyValue } from '@/hooks/types';

function App() {
  // 開発環境と本番環境でWebSocket URLを切り替え
  const wsUrl = import.meta.env.DEV 
    ? 'wss://localhost:8080/ws'  // 開発時も直接接続
    : 'wss://localhost:8080/ws'; // 本番時
  
  const echonet = usePropertyDescriptions(wsUrl);

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

  // Get all tabs (locations + groups)
  const tabs = getAllTabs(echonet.devices, echonet.aliases, echonet.groups);


  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="container mx-auto p-4">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold">ECHONET List</h1>
          <Badge variant="outline" className={`${getConnectionColor(echonet.connectionState)} text-white`}>
            {echonet.connectionState}
          </Badge>
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
          <Tabs defaultValue={tabs[0]} className="w-full">
            <div className="w-full mb-6 overflow-x-auto">
              <TabsList className="w-max min-w-full h-auto p-1 bg-muted flex flex-nowrap justify-start gap-1 sm:flex-wrap sm:w-full">
              {tabs.map((tab) => (
                <TabsTrigger 
                  key={tab} 
                  value={tab} 
                  className="data-[state=active]:bg-primary data-[state=active]:text-primary-foreground px-3 py-2 text-sm whitespace-nowrap flex-shrink-0"
                >
                  <span className="hidden sm:inline">{tab}</span>
                  <span className="sm:hidden">{tab.length > 8 ? tab.substring(0, 8) + '...' : tab}</span>
                  {tab !== 'All' && (
                    <span className="ml-1 hidden sm:inline">({getDevicesForTab(tab).length})</span>
                  )}
                </TabsTrigger>
              ))}
              </TabsList>
            </div>
            
            {tabs.map((tab) => (
              <TabsContent key={tab} value={tab} className="space-y-4">
                <div className="grid gap-4">
                  {getDevicesForTab(tab).map((device) => {
                    const deviceKey = `${device.ip} ${device.eoj}`;
                    const aliasInfo = deviceHasAlias(device, echonet.devices, echonet.aliases);
                    
                    return (
                      <Card key={deviceKey}>
                        <CardHeader>
                          <div className="flex items-start justify-between">
                            <div className="space-y-1 flex-1">
                              <CardTitle>
                                {aliasInfo.aliasName || device.name}
                              </CardTitle>
                              {aliasInfo.hasAlias && (
                                <p className="text-sm text-muted-foreground">
                                  Device: {device.name}
                                </p>
                              )}
                              <p className="text-sm text-muted-foreground">
                                {device.ip} - {device.eoj}
                              </p>
                            </div>
                            {aliasInfo.hasAlias && (
                              <Badge variant="secondary" className="ml-2 text-xs">
                                Alias
                              </Badge>
                            )}
                          </div>
                        </CardHeader>
                        <CardContent>
                          <div className="grid gap-2">
                            {Object.entries(device.properties).map(([epc, value]) => {
                              const classCode = echonet.getDeviceClassCode(device);
                              const propertyName = getPropertyName(epc, echonet.propertyDescriptions, classCode);
                              const propertyDescriptor = getPropertyDescriptor(epc, echonet.propertyDescriptions, classCode);
                              const formattedValue = formatPropertyValue(value, propertyDescriptor);
                              
                              return (
                                <div key={epc} className="flex justify-between items-center">
                                  <span className="text-sm font-medium">{propertyName}:</span>
                                  <div className="flex items-center gap-2">
                                    <span className="text-sm">
                                      {formattedValue}
                                    </span>
                                    <PropertyEditor
                                      device={device}
                                      epc={epc}
                                      currentValue={value}
                                      descriptor={propertyDescriptor}
                                      onPropertyChange={handlePropertyChange}
                                    />
                                  </div>
                                </div>
                              );
                            })}
                          </div>
                          <p className="text-xs text-muted-foreground mt-2">
                            Last seen: {new Date(device.lastSeen).toLocaleString()}
                          </p>
                        </CardContent>
                      </Card>
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