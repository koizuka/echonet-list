import { useState, useMemo } from 'react';

export type DummyDevice = {
  id: string;
  alias: string;
  location: string;
  type: string;
  status: 'on' | 'off';
  temperature?: number;
};

const initialDevices: DummyDevice[] = [
  { id: '1', alias: 'リビングエアコン', location: 'リビング', type: 'エアコン', status: 'on', temperature: 25 },
  { id: '2', alias: '寝室エアコン', location: '寝室', type: 'エアコン', status: 'off', temperature: 22 },
  { id: '3', alias: 'キッチン照明', location: 'キッチン', type: '照明', status: 'on' },
  { id: '4', alias: 'ダイニング照明', location: 'ダイニング', type: '照明', status: 'off' },
];

export function useDummyDevices() {
  const [devices, setDevices] = useState<DummyDevice[]>(initialDevices);

  const toggleDevice = (deviceId: string) => {
    setDevices(prevDevices =>
      prevDevices.map(device =>
        device.id === deviceId
          ? { ...device, status: device.status === 'on' ? 'off' : 'on' }
          : device
      )
    );
  };

  const groupedDevices = useMemo(() => {
    return devices.reduce((acc, device) => {
      if (!acc[device.location]) {
        acc[device.location] = [];
      }
      acc[device.location].push(device);
      return acc;
    }, {} as Record<string, DummyDevice[]>);
  }, [devices]);

  return {
    devices,
    groupedDevices,
    toggleDevice,
  };
}