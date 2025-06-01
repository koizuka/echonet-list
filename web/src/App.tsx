import { usePropertyDescriptions } from '@/hooks/usePropertyDescriptions';
import { getPropertyName, formatPropertyValue, getPropertyDescriptor } from '@/libs/propertyHelper';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';

function App() {
  // 開発環境と本番環境でWebSocket URLを切り替え
  const wsUrl = import.meta.env.DEV 
    ? 'wss://localhost:8080/ws'  // 開発時も直接接続
    : 'wss://localhost:8080/ws'; // 本番時
  
  const echonet = usePropertyDescriptions(wsUrl);

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

        <div className="grid gap-4">
          {Object.entries(echonet.devices).map(([key, device]) => (
            <Card key={key}>
              <CardHeader>
                <CardTitle>{device.name}</CardTitle>
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
                      <div key={epc} className="flex justify-between">
                        <span className="text-sm font-medium">{propertyName}:</span>
                        <span className="text-sm">
                          {formattedValue}
                        </span>
                      </div>
                    );
                  })}
                </div>
                <p className="text-xs text-muted-foreground mt-2">
                  Last seen: {new Date(device.lastSeen).toLocaleString()}
                </p>
              </CardContent>
            </Card>
          ))}
        </div>

        {Object.keys(echonet.devices).length === 0 && echonet.connectionState === 'connected' && (
          <Card>
            <CardContent className="pt-6">
              <p className="text-center text-muted-foreground">
                No devices found. Click refresh to discover devices.
              </p>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}

export default App;