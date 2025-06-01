import { usePropertyDescriptions } from '@/hooks/usePropertyDescriptions';
import { getPropertyName, formatPropertyValue, getPropertyDescriptor } from '@/libs/propertyHelper';
import { getAllLocations, groupDevicesByLocation, getDeviceDisplayName, extractLocationFromDevice } from '@/libs/locationHelper';
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

  // Get locations and grouped devices
  const locations = getAllLocations(echonet.devices, echonet.aliases);
  const groupedDevices = groupDevicesByLocation(echonet.devices, echonet.aliases);

  // Debug: Log device locations (only in development)
  if (import.meta.env.DEV && Object.keys(echonet.devices).length > 0) {
    console.log('🏠 Device locations debug:');
    Object.values(echonet.devices).forEach(device => {
      const deviceId = `${device.ip} ${device.eoj}`;
      const installationLocationProp = device.properties['81'];
      const aliasName = Object.entries(echonet.aliases).find(([, id]) => id === deviceId)?.[0];
      
      console.log(`Device: ${device.name} (${deviceId})`);
      console.log(`  Installation Location (EPC 81): ${installationLocationProp?.string || 'none'}`);
      console.log(`  Alias: ${aliasName || 'none'}`);
      console.log(`  Extracted Location: ${extractLocationFromDevice(device, echonet.aliases)}`);
      console.log('---');
    });
    console.log('Grouped devices:', groupedDevices);
  }

  // Function to get devices for a specific tab
  const getDevicesForTab = (location: string) => {
    if (location === 'All') {
      return Object.values(echonet.devices);
    }
    return groupedDevices[location] || [];
  };

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
          <Tabs defaultValue={locations[0]} className="w-full">
            <TabsList className="grid w-full grid-cols-auto mb-6" style={{ gridTemplateColumns: `repeat(${locations.length}, minmax(0, 1fr))` }}>
              {locations.map((location) => (
                <TabsTrigger key={location} value={location} className="data-[state=active]:bg-primary data-[state=active]:text-primary-foreground">
                  {location} {location !== 'All' && `(${getDevicesForTab(location).length})`}
                </TabsTrigger>
              ))}
            </TabsList>
            
            {locations.map((location) => (
              <TabsContent key={location} value={location} className="space-y-4">
                <div className="grid gap-4">
                  {getDevicesForTab(location).map((device) => {
                    const deviceKey = `${device.ip} ${device.eoj}`;
                    const displayName = getDeviceDisplayName(device, echonet.aliases);
                    
                    return (
                      <Card key={deviceKey}>
                        <CardHeader>
                          <CardTitle>{displayName}</CardTitle>
                          <p className="text-sm text-muted-foreground">
                            {device.ip} - {device.eoj}
                          </p>
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
                
                {getDevicesForTab(location).length === 0 && (
                  <Card>
                    <CardContent className="pt-6">
                      <p className="text-center text-muted-foreground">
                        No devices found in {location === 'All' ? 'any location' : location}.
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