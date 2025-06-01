import { renderHook, act } from '@testing-library/react';
import { useDummyDevices } from './useDummyDevices';

describe('useDummyDevices', () => {
  it('should initialize with dummy devices', () => {
    const { result } = renderHook(() => useDummyDevices());
    
    expect(result.current.devices).toHaveLength(4);
    expect(result.current.devices[0]).toEqual({
      id: '1',
      alias: 'リビングエアコン',
      location: 'リビング',
      type: 'エアコン',
      status: 'on',
      temperature: 25,
    });
  });

  it('should toggle device status', () => {
    const { result } = renderHook(() => useDummyDevices());
    
    const initialDevice = result.current.devices.find(d => d.id === '1');
    expect(initialDevice?.status).toBe('on');
    
    act(() => {
      result.current.toggleDevice('1');
    });
    
    const updatedDevice = result.current.devices.find(d => d.id === '1');
    expect(updatedDevice?.status).toBe('off');
  });

  it('should group devices by location', () => {
    const { result } = renderHook(() => useDummyDevices());
    
    const grouped = result.current.groupedDevices;
    
    expect(Object.keys(grouped)).toContain('リビング');
    expect(Object.keys(grouped)).toContain('寝室');
    expect(Object.keys(grouped)).toContain('キッチン');
    expect(Object.keys(grouped)).toContain('ダイニング');
    
    expect(grouped['リビング']).toHaveLength(1);
    expect(grouped['リビング'][0].alias).toBe('リビングエアコン');
  });
});