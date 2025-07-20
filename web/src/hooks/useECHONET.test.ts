import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useECHONET } from './useECHONET';
import type { Device, ServerMessage } from './types';

// Mock the WebSocket connection hook
vi.mock('./useWebSocketConnection', () => ({
  useWebSocketConnection: vi.fn(),
}));

import { useWebSocketConnection } from './useWebSocketConnection';

describe('useECHONET', () => {
  let mockSendMessage: ReturnType<typeof vi.fn>;
  let mockConnect: ReturnType<typeof vi.fn>;
  let mockDisconnect: ReturnType<typeof vi.fn>;
  let capturedCallbacks: {
    onMessage?: (message: ServerMessage) => void;
    onConnectionStateChange?: (state: any) => void;
    onError?: (error: any) => void;
  } = {};

  beforeEach(() => {
    mockSendMessage = vi.fn();
    mockConnect = vi.fn();
    mockDisconnect = vi.fn();
    capturedCallbacks = {};

    (useWebSocketConnection as any).mockReturnValue({
      connectionState: 'disconnected',
      sendMessage: mockSendMessage,
      connect: mockConnect,
      disconnect: mockDisconnect,
      error: null,
    });

    // Capture the callbacks passed to useWebSocketConnection
    (useWebSocketConnection as any).mockImplementation((options: any) => {
      capturedCallbacks = {
        onMessage: options.onMessage,
        onConnectionStateChange: options.onConnectionStateChange,
      };
      return {
        connectionState: 'disconnected',
        sendMessage: mockSendMessage,
        connect: mockConnect,
        disconnect: mockDisconnect,
      };
    });
  });

  const testUrl = 'ws://localhost:8080/ws';

  it('should initialize with empty state', () => {
    const { result } = renderHook(() => useECHONET(testUrl));
    
    expect(result.current.devices).toEqual({});
    expect(result.current.aliases).toEqual({});
    expect(result.current.groups).toEqual({});
    expect(result.current.connectionState).toBe('disconnected');
    expect(result.current.propertyDescriptions).toEqual({});
  });

  it('should handle initial_state message', () => {
    const { result } = renderHook(() => useECHONET(testUrl));

    const testDevice: Device = {
      ip: '192.168.1.10',
      eoj: '0130:1',
      name: 'HomeAirConditioner',
      id: '013001:00000B:ABCDEF0123456789ABCDEF012345',
      properties: {
        '80': { EDT: 'MzA=', string: 'on' },
      },
      lastSeen: '2023-04-01T12:34:56Z',
    };

    const initialStateMessage: ServerMessage = {
      type: 'initial_state',
      payload: {
        devices: { '192.168.1.10 0130:1': testDevice },
        aliases: { living_ac: '013001:00000B:ABCDEF0123456789ABCDEF012345' },
        groups: { '@living_room': ['013001:00000B:ABCDEF0123456789ABCDEF012345'] },
      },
    };

    act(() => {
      capturedCallbacks.onMessage?.(initialStateMessage);
    });

    expect(result.current.devices).toEqual({ '192.168.1.10 0130:1': testDevice });
    expect(result.current.aliases).toEqual({ living_ac: '013001:00000B:ABCDEF0123456789ABCDEF012345' });
    expect(result.current.groups).toEqual({ '@living_room': ['013001:00000B:ABCDEF0123456789ABCDEF012345'] });
  });

  it('should handle device_added message', () => {
    const { result } = renderHook(() => useECHONET(testUrl));

    const newDevice: Device = {
      ip: '192.168.1.11',
      eoj: '0290:1',
      name: 'LightingSystem',
      id: '029001:000005:FEDCBA9876543210FEDCBA987654',
      properties: {},
      lastSeen: '2023-04-01T12:35:00Z',
    };

    const deviceAddedMessage: ServerMessage = {
      type: 'device_added',
      payload: { device: newDevice },
    };

    act(() => {
      capturedCallbacks.onMessage?.(deviceAddedMessage);
    });

    expect(result.current.devices['192.168.1.11 0290:1']).toEqual({
      ...newDevice,
      isOffline: false, // ADD_DEVICE action adds this field
    });
  });

  it('should auto-fetch cached data for device_added with empty properties', async () => {
    renderHook(() => useECHONET(testUrl));

    const newDeviceEmptyProps: Device = {
      ip: '192.168.1.12',
      eoj: '0130:1',
      name: 'AirConditioner',
      id: '192.168.1.12 0130:1',
      properties: {}, // 空のプロパティ
      lastSeen: '2023-04-01T12:35:00Z',
    };

    const deviceAddedMessage: ServerMessage = {
      type: 'device_added',
      payload: { device: newDeviceEmptyProps },
    };

    await act(async () => {
      capturedCallbacks.onMessage?.(deviceAddedMessage);
      // Wait for async list_devices call
      await new Promise(resolve => setTimeout(resolve, 50));
    });

    // プロパティが空の場合はlist_devicesが自動的に呼ばれることを確認（キャッシュベース）
    expect(mockSendMessage).toHaveBeenCalledWith({
      type: 'list_devices',
      payload: { targets: ['192.168.1.12 0130:1'] },
      requestId: '',
    });
  });

  it('should handle property_changed message', () => {
    const { result } = renderHook(() => useECHONET(testUrl));

    // First add a device
    const testDevice: Device = {
      ip: '192.168.1.10',
      eoj: '0130:1',
      name: 'HomeAirConditioner',
      id: '013001:00000B:ABCDEF0123456789ABCDEF012345',
      properties: {
        '80': { EDT: 'MzA=', string: 'on' },
      },
      lastSeen: '2023-04-01T12:34:56Z',
    };

    const initialStateMessage: ServerMessage = {
      type: 'initial_state',
      payload: {
        devices: { '192.168.1.10 0130:1': testDevice },
        aliases: {},
        groups: {},
      },
    };

    act(() => {
      capturedCallbacks.onMessage?.(initialStateMessage);
    });

    // Now send property change
    const propertyChangedMessage: ServerMessage = {
      type: 'property_changed',
      payload: {
        ip: '192.168.1.10',
        eoj: '0130:1',
        epc: 'B3',
        value: { EDT: 'MjY=', string: '26', number: 26 },
      },
    };

    act(() => {
      capturedCallbacks.onMessage?.(propertyChangedMessage);
    });

    const updatedDevice = result.current.devices['192.168.1.10 0130:1'];
    expect(updatedDevice.properties['B3']).toEqual({ EDT: 'MjY=', string: '26', number: 26 });
    expect(new Date(updatedDevice.lastSeen).getTime()).toBeGreaterThan(new Date(testDevice.lastSeen).getTime());
  });

  it('should handle alias_changed message', () => {
    const { result } = renderHook(() => useECHONET(testUrl));

    // Add alias
    const aliasAddedMessage: ServerMessage = {
      type: 'alias_changed',
      payload: {
        change_type: 'added',
        alias: 'kitchen_ac',
        target: '013001:00000B:ABCDEF0123456789ABCDEF012345',
      },
    };

    act(() => {
      capturedCallbacks.onMessage?.(aliasAddedMessage);
    });

    expect(result.current.aliases.kitchen_ac).toBe('013001:00000B:ABCDEF0123456789ABCDEF012345');

    // Delete alias
    const aliasDeletedMessage: ServerMessage = {
      type: 'alias_changed',
      payload: {
        change_type: 'deleted',
        alias: 'kitchen_ac',
      },
    };

    act(() => {
      capturedCallbacks.onMessage?.(aliasDeletedMessage);
    });

    expect(result.current.aliases.kitchen_ac).toBeUndefined();
  });

  it('should handle group_changed message', () => {
    const { result } = renderHook(() => useECHONET(testUrl));

    // Add group
    const groupAddedMessage: ServerMessage = {
      type: 'group_changed',
      payload: {
        change_type: 'added',
        group: '@kitchen',
        devices: ['013001:00000B:ABCDEF0123456789ABCDEF012345'],
      },
    };

    act(() => {
      capturedCallbacks.onMessage?.(groupAddedMessage);
    });

    expect(result.current.groups['@kitchen']).toEqual(['013001:00000B:ABCDEF0123456789ABCDEF012345']);

    // Delete group
    const groupDeletedMessage: ServerMessage = {
      type: 'group_changed',
      payload: {
        change_type: 'deleted',
        group: '@kitchen',
      },
    };

    act(() => {
      capturedCallbacks.onMessage?.(groupDeletedMessage);
    });

    expect(result.current.groups['@kitchen']).toBeUndefined();
  });

  it('should handle device_offline message', () => {
    const { result } = renderHook(() => useECHONET(testUrl));

    // First add a device
    const testDevice: Device = {
      ip: '192.168.1.10',
      eoj: '0130:1',
      name: 'HomeAirConditioner',
      id: '013001:00000B:ABCDEF0123456789ABCDEF012345',
      properties: {},
      lastSeen: '2023-04-01T12:34:56Z',
    };

    const initialStateMessage: ServerMessage = {
      type: 'initial_state',
      payload: {
        devices: { '192.168.1.10 0130:1': testDevice },
        aliases: {},
        groups: {},
      },
    };

    act(() => {
      capturedCallbacks.onMessage?.(initialStateMessage);
    });

    expect(result.current.devices['192.168.1.10 0130:1']).toBeDefined();

    // Now mark device as offline
    const deviceOfflineMessage: ServerMessage = {
      type: 'device_offline',
      payload: {
        ip: '192.168.1.10',
        eoj: '0130:1',
      },
    };

    act(() => {
      capturedCallbacks.onMessage?.(deviceOfflineMessage);
    });

    expect(result.current.devices['192.168.1.10 0130:1']).toEqual({
      ...testDevice,
      isOffline: true, // MARK_DEVICE_OFFLINE action sets this field
    });
  });

  it('should handle device_deleted message', () => {
    const { result } = renderHook(() => useECHONET(testUrl));
    
    // First add a device
    const testDevice: Device = {
      ip: '192.168.1.10',
      eoj: '0130:1',
      name: 'HomeAirConditioner',
      id: '013001:00000B:ABCDEF0123456789ABCDEF012345',
      properties: {},
      lastSeen: '2023-04-01T12:34:56Z',
      isOffline: true,
    };
    
    const initialStateMessage: ServerMessage = {
      type: 'initial_state',
      payload: {
        devices: { '192.168.1.10 0130:1': testDevice },
        aliases: {},
        groups: {},
      },
    };
    
    act(() => {
      capturedCallbacks.onMessage?.(initialStateMessage);
    });
    
    // Verify device is present
    expect(result.current.devices['192.168.1.10 0130:1']).toBeDefined();
    
    // Send device_deleted message
    const deviceDeletedMessage: ServerMessage = {
      type: 'device_deleted',
      payload: {
        ip: '192.168.1.10',
        eoj: '0130:1',
      },
    };
    
    act(() => {
      capturedCallbacks.onMessage?.(deviceDeletedMessage);
    });
    
    // Device should be removed
    expect(result.current.devices['192.168.1.10 0130:1']).toBeUndefined();
  });

  it('should handle device_online message', async () => {
    renderHook(() => useECHONET(testUrl));

    // Device online message should be received without errors
    const deviceOnlineMessage: ServerMessage = {
      type: 'device_online',
      payload: {
        ip: '192.168.1.10',
        eoj: '0130:1',
      },
    };

    // Should not throw an error
    await act(async () => {
      capturedCallbacks.onMessage?.(deviceOnlineMessage);
    });

    // The actual device restoration will be handled by subsequent device_added message
  });

  it('should send device operation messages', async () => {
    const { result } = renderHook(() => useECHONET(testUrl));

    await act(async () => {
      result.current.listDevices(['192.168.1.10 0130:1']);
    });

    expect(mockSendMessage).toHaveBeenCalledWith({
      type: 'list_devices',
      payload: {
        targets: ['192.168.1.10 0130:1'],
      },
      requestId: '',
    });

    await act(async () => {
      result.current.setDeviceProperties('192.168.1.10 0130:1', {
        '80': { string: 'on' },
      });
    });

    expect(mockSendMessage).toHaveBeenCalledWith({
      type: 'set_properties',
      payload: {
        target: '192.168.1.10 0130:1',
        properties: { '80': { string: 'on' } },
      },
      requestId: '',
    });
  });

  it('should send alias operation messages', async () => {
    const { result } = renderHook(() => useECHONET(testUrl));

    await act(async () => {
      result.current.addAlias('living_ac', '013001:00000B:ABCDEF0123456789ABCDEF012345');
    });

    expect(mockSendMessage).toHaveBeenCalledWith({
      type: 'manage_alias',
      payload: {
        action: 'add',
        alias: 'living_ac',
        target: '013001:00000B:ABCDEF0123456789ABCDEF012345',
      },
      requestId: '',
    });

    await act(async () => {
      result.current.deleteAlias('living_ac');
    });

    expect(mockSendMessage).toHaveBeenCalledWith({
      type: 'manage_alias',
      payload: {
        action: 'delete',
        alias: 'living_ac',
      },
      requestId: '',
    });
  });

  it('should send group operation messages', async () => {
    const { result } = renderHook(() => useECHONET(testUrl));

    await act(async () => {
      result.current.addGroup('@living_room', ['013001:00000B:ABCDEF0123456789ABCDEF012345']);
    });

    expect(mockSendMessage).toHaveBeenCalledWith({
      type: 'manage_group',
      payload: {
        action: 'add',
        group: '@living_room',
        devices: ['013001:00000B:ABCDEF0123456789ABCDEF012345'],
      },
      requestId: '',
    });
  });

  it('should cache property descriptions', async () => {
    const { result } = renderHook(() => useECHONET(testUrl));

    const mockPropertyData = {
      classCode: '0130',
      properties: {
        '80': {
          description: 'Operation status',
          aliases: { on: 'MzA=', off: 'MzE=' },
        },
      },
    };

    mockSendMessage.mockResolvedValueOnce(mockPropertyData);

    await act(async () => {
      const data = await result.current.getPropertyDescription('0130');
      expect(data).toEqual(mockPropertyData);
    });

    expect(mockSendMessage).toHaveBeenCalledTimes(1);

    // Second call should use cache
    await act(async () => {
      const data = await result.current.getPropertyDescription('0130');
      expect(data).toEqual(mockPropertyData);
    });

    // Should not call sendMessage again
    expect(mockSendMessage).toHaveBeenCalledTimes(1);
  });
});