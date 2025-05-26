import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useDummyDevices } from '@/hooks/useDummyDevices';

export function DummyDeviceList() {
  const { groupedDevices, toggleDevice } = useDummyDevices();

  return (
    <div className="space-y-6">
      {Object.entries(groupedDevices).map(([location, devices]) => (
        <div key={location}>
          <h2 className="text-2xl font-semibold mb-4">{location}</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {devices.map((device) => (
              <Card key={device.id} className="hover:shadow-lg transition-shadow">
                <CardHeader>
                  <div className="flex justify-between items-start">
                    <div>
                      <CardTitle className="text-lg">{device.alias}</CardTitle>
                      <CardDescription>{device.type}</CardDescription>
                    </div>
                    <Badge variant={device.status === 'on' ? 'default' : 'secondary'}>
                      {device.status === 'on' ? 'ON' : 'OFF'}
                    </Badge>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    {device.temperature && (
                      <div className="text-sm text-muted-foreground">
                        温度: {device.temperature}°C
                      </div>
                    )}
                    <Button 
                      onClick={() => toggleDevice(device.id)}
                      variant={device.status === 'on' ? 'destructive' : 'default'}
                      className="w-full"
                    >
                      {device.status === 'on' ? 'OFF にする' : 'ON にする'}
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}