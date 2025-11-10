import { useCallback, useReducer, useRef, useEffect } from 'react';
import { useWebSocketConnection, type WebSocketConnection } from './useWebSocketConnection';
import { getCurrentLocale } from '../libs/languageHelper';
import type {
  ECHONETState,
  Device,
  DeviceAlias,
  DeviceGroup,
  PropertyDescriptionData,
  ServerMessage,
  ConnectionState,
  PropertyValue
} from './types';

// Type alias for in-flight property description request tracking
type PendingPropertyDescriptionRequests = Map<string, Promise<PropertyDescriptionData>>;

type ECHONETAction =
  | { type: 'SET_INITIAL_STATE'; payload: { devices: Record<string, Device>; aliases: DeviceAlias; groups: DeviceGroup; serverStartupTime?: string } }
  | { type: 'ADD_DEVICE'; payload: { device: Device } }
  | { type: 'MARK_DEVICE_OFFLINE'; payload: { ip: string; eoj: string } }
  | { type: 'MARK_DEVICE_ONLINE'; payload: { ip: string; eoj: string } }
  | { type: 'DELETE_DEVICE'; payload: { ip: string; eoj: string } }
  | { type: 'UPDATE_PROPERTY'; payload: { ip: string; eoj: string; epc: string; value: PropertyValue } }
  | { type: 'SET_ALIAS'; payload: { alias: string; target?: string; changeType: 'added' | 'updated' | 'deleted' } }
  | { type: 'SET_GROUP'; payload: { group: string; devices?: string[]; changeType: 'added' | 'updated' | 'deleted' } }
  | { type: 'SET_PROPERTY_DESCRIPTION'; payload: { classCode: string; data: PropertyDescriptionData } }
  | { type: 'SET_CONNECTION_STATE'; payload: { state: ConnectionState } };

function echonetReducer(state: ECHONETState, action: ECHONETAction): ECHONETState {
  switch (action.type) {
    case 'SET_INITIAL_STATE':
      return {
        ...state,
        devices: action.payload.devices,
        aliases: action.payload.aliases,
        groups: action.payload.groups,
        serverStartupTime: action.payload.serverStartupTime ? new Date(action.payload.serverStartupTime) : null,
        initialStateReceived: true,
      };

    case 'ADD_DEVICE': {
      const deviceKey = `${action.payload.device.ip} ${action.payload.device.eoj}`;
      
      return {
        ...state,
        devices: {
          ...state.devices,
          [deviceKey]: {
            ...action.payload.device,
            isOffline: false, // Clear offline flag when device is added/comes back online
          },
        },
      };
    }

    case 'MARK_DEVICE_OFFLINE': {
      const deviceKey = `${action.payload.ip} ${action.payload.eoj}`;
      const device = state.devices[deviceKey];
      if (!device) return state;
      
      return {
        ...state,
        devices: {
          ...state.devices,
          [deviceKey]: {
            ...device,
            isOffline: true,
          },
        },
      };
    }

    case 'MARK_DEVICE_ONLINE': {
      const deviceKey = `${action.payload.ip} ${action.payload.eoj}`;
      const device = state.devices[deviceKey];
      if (!device) return state;
      
      return {
        ...state,
        devices: {
          ...state.devices,
          [deviceKey]: {
            ...device,
            isOffline: false,
            lastSeen: new Date().toISOString(),
          },
        },
      };
    }

    case 'DELETE_DEVICE': {
      const deviceKey = `${action.payload.ip} ${action.payload.eoj}`;
      const { [deviceKey]: _removed, ...remainingDevices } = state.devices;
      void _removed; // Explicitly void the unused variable
      return {
        ...state,
        devices: remainingDevices,
      };
    }

    case 'UPDATE_PROPERTY': {
      const deviceKey = `${action.payload.ip} ${action.payload.eoj}`;
      const device = state.devices[deviceKey];
      if (!device) return state;

      return {
        ...state,
        devices: {
          ...state.devices,
          [deviceKey]: {
            ...device,
            properties: {
              ...device.properties,
              [action.payload.epc]: action.payload.value,
            },
            lastSeen: new Date().toISOString(),
          },
        },
      };
    }

    case 'SET_ALIAS': {
      const { alias, target, changeType } = action.payload;
      const newAliases = { ...state.aliases };

      if (changeType === 'deleted') {
        delete newAliases[alias];
      } else if (target) {
        newAliases[alias] = target;
      }

      return {
        ...state,
        aliases: newAliases,
      };
    }

    case 'SET_GROUP': {
      const { group, devices, changeType } = action.payload;
      const newGroups = { ...state.groups };

      if (changeType === 'deleted') {
        delete newGroups[group];
      } else if (devices) {
        newGroups[group] = devices;
      }

      return {
        ...state,
        groups: newGroups,
      };
    }

    case 'SET_PROPERTY_DESCRIPTION':
      return {
        ...state,
        propertyDescriptions: {
          ...state.propertyDescriptions,
          [action.payload.classCode]: action.payload.data,
        },
      };

    case 'SET_CONNECTION_STATE':
      return {
        ...state,
        connectionState: action.payload.state,
      };

    default:
      return state;
  }
}

const initialState: ECHONETState = {
  devices: {},
  aliases: {},
  groups: {},
  connectionState: 'disconnected',
  propertyDescriptions: {},
  initialStateReceived: false,
  serverStartupTime: null,
};

export type ECHONETHook = {
  // State
  devices: Record<string, Device>;
  aliases: DeviceAlias;
  groups: DeviceGroup;
  connectionState: ConnectionState;
  propertyDescriptions: Record<string, PropertyDescriptionData>;
  initialStateReceived: boolean;
  connectedAt: Date | null;
  serverStartupTime: Date | null;
  connection: WebSocketConnection;

  // Device operations
  listDevices: (targets: string[]) => Promise<unknown>;
  setDeviceProperties: (target: string, properties: Record<string, PropertyValue>) => Promise<unknown>;
  updateDeviceProperties: (targets?: string[], force?: boolean) => Promise<unknown>;
  discoverDevices: () => Promise<unknown>;
  deleteDevice: (target: string) => Promise<unknown>;

  // Alias operations
  addAlias: (alias: string, target: string) => Promise<unknown>;
  deleteAlias: (alias: string) => Promise<unknown>;

  // Group operations
  addGroup: (group: string, devices: string[]) => Promise<unknown>;
  addToGroup: (group: string, devices: string[]) => Promise<unknown>;
  removeFromGroup: (group: string, devices: string[]) => Promise<unknown>;
  deleteGroup: (group: string) => Promise<unknown>;
  listGroups: (group: string) => Promise<unknown>;

  // Property description operations
  getPropertyDescription: (classCode: string, lang?: string) => Promise<PropertyDescriptionData>;

  // Connection operations
  connect: () => void;
  disconnect: () => void;
  checkConnection: () => Promise<boolean>;

  // Message handler for additional processing
  onMessage?: (message: ServerMessage) => void;
};

export function useECHONET(
  url: string,
  onMessage?: (message: ServerMessage) => void,
  onWebSocketConnected?: () => void
): ECHONETHook {
  const [state, dispatch] = useReducer(echonetReducer, initialState);

  // useRef to avoid circular dependency between handleServerMessage and functions defined later
  const listDevicesRef = useRef<((targets: string[]) => Promise<unknown>) | null>(null);
  const updateDevicePropertiesRef = useRef<((targets?: string[], force?: boolean) => Promise<unknown>) | null>(null);

  const handleServerMessage = useCallback((message: ServerMessage) => {
    // Call external handler if provided
    onMessage?.(message);
    if (import.meta.env.DEV) {
      console.log('📨 Received server message:', message.type);
    }
    switch (message.type) {
      case 'initial_state':
        if (import.meta.env.DEV) {
          console.log('🎉 Received initial_state with', Object.keys(message.payload.devices || {}).length, 'devices');
        }
        dispatch({
          type: 'SET_INITIAL_STATE',
          payload: {
            devices: message.payload.devices,
            aliases: message.payload.aliases,
            groups: message.payload.groups,
            serverStartupTime: message.payload.serverStartupTime,
          },
        });
        break;

      case 'device_added': {
        const addedDevice = message.payload.device;
        
        dispatch({
          type: 'ADD_DEVICE',
          payload: { device: addedDevice },
        });
        
        // プロパティが空の場合（オンライン復旧時など）は自動的にプロパティを取得
        const deviceId = `${addedDevice.ip} ${addedDevice.eoj}`;
        if (Object.keys(addedDevice.properties).length === 0) {
          // プロパティが空の場合（オンライン復旧時など）は自動的にキャッシュからプロパティを取得
          (async () => {
            try {
              // list_devices でキャッシュされたプロパティを取得（ネットワーク通信なし）
              if (listDevicesRef.current) {
                const result = await listDevicesRef.current([deviceId]);
                
                // list_devicesの応答にはデバイス情報が含まれているので、それでstateを更新
                if (result && typeof result === 'object' && 'ip' in result && 'eoj' in result) {
                  const device = result as Device;
                  const propertyCount = device.properties ? Object.keys(device.properties).length : 0;
                  dispatch({
                    type: 'ADD_DEVICE',
                    payload: { device },
                  });
                  
                  if (propertyCount === 0) {
                    // フォールバック: update_propertiesで再試行
                    try {
                      if (updateDevicePropertiesRef.current) {
                        await updateDevicePropertiesRef.current([deviceId], true);
                      }
                    } catch {
                      // フォールバックも失敗した場合は静かに処理終了
                    }
                  }
                }
              }
            } catch {
              // エラーは静かに処理（ログスパム回避）
            }
          })();
        }
        break;
      }

      case 'device_offline':
        dispatch({
          type: 'MARK_DEVICE_OFFLINE',
          payload: { ip: message.payload.ip, eoj: message.payload.eoj },
        });
        break;

      case 'device_online': {
        dispatch({
          type: 'MARK_DEVICE_ONLINE',
          payload: { ip: message.payload.ip, eoj: message.payload.eoj },
        });
        
        // デバイスが復帰した場合、プロパティが空の可能性があるため再取得を試行
        const deviceKey = `${message.payload.ip} ${message.payload.eoj}`;
        (async () => {
          try {
            if (!listDevicesRef.current) return;
            const deviceListResponse = await listDevicesRef.current([deviceKey]);
            if (deviceListResponse && typeof deviceListResponse === 'object' && 'devices' in deviceListResponse) {
              const devices = deviceListResponse.devices as Record<string, Device>;
              const deviceData = devices[deviceKey];
              if (deviceData && Object.keys(deviceData.properties || {}).length > 0) {
                // 成功した場合、デバイスを再追加（プロパティ付きで）
                dispatch({
                  type: 'ADD_DEVICE',
                  payload: { device: deviceData },
                });
              }
            }
          } catch (error) {
            // エラーが発生した場合は静かに処理（既にオンラインフラグは設定済み）
            console.warn('Failed to fetch properties for recovered device:', deviceKey, error);
          }
        })();
        break;
      }
      
      case 'device_deleted':
        dispatch({
          type: 'DELETE_DEVICE',
          payload: { ip: message.payload.ip, eoj: message.payload.eoj },
        });
        break;

      case 'property_changed':
        dispatch({
          type: 'UPDATE_PROPERTY',
          payload: {
            ip: message.payload.ip,
            eoj: message.payload.eoj,
            epc: message.payload.epc,
            value: message.payload.value,
          },
        });
        break;

      case 'alias_changed':
        dispatch({
          type: 'SET_ALIAS',
          payload: {
            alias: message.payload.alias,
            target: message.payload.target,
            changeType: message.payload.change_type,
          },
        });
        break;

      case 'group_changed':
        dispatch({
          type: 'SET_GROUP',
          payload: {
            group: message.payload.group,
            devices: message.payload.devices,
            changeType: message.payload.change_type,
          },
        });
        break;

      case 'error_notification': {
        // Convert server error notification to log notification for NotificationBell
        const errorLogMessage = {
          type: 'log_notification' as const,
          payload: {
            level: 'ERROR' as const,
            message: `Server Error (${message.payload.code}): ${message.payload.message}`,
            time: new Date().toISOString(),
            attributes: {
              component: 'Server',
              errorCode: message.payload.code,
              originalMessage: message.payload.message
            }
          }
        };
        // Also call external handler if provided
        onMessage?.(errorLogMessage);
        break;
      }

      case 'timeout_notification':
        // Handle timeout notification if needed
        console.warn('Device timeout:', message.payload);
        break;

      default:
        console.log('Unhandled server message:', message);
    }
  }, [onMessage]);

  const handleConnectionStateChange = useCallback((connectionState: ConnectionState) => {
    if (import.meta.env.DEV) {
      console.log('🔄 Connection state changed:', connectionState);
    }
    dispatch({ type: 'SET_CONNECTION_STATE', payload: { state: connectionState } });
  }, []);

  const connection = useWebSocketConnection({
    url,
    // 開発環境では再接続を無効化、本番環境では有効
    reconnectAttempts: import.meta.env.DEV ? 0 : 5,
    reconnectDelay: 1000,
    maxReconnectDelay: 30000,
    onMessage: handleServerMessage,
    onConnectionStateChange: handleConnectionStateChange,
    onWebSocketConnected,
  });

  // Device operations
  const listDevices = useCallback(async (targets: string[]) => {
    return connection.sendMessage({
      type: 'list_devices',
      payload: { targets },
      requestId: '', // Will be set by sendMessage
    });
  }, [connection]);

  const setDeviceProperties = useCallback(async (target: string, properties: Record<string, PropertyValue>) => {
    const response = await connection.sendMessage({
      type: 'set_properties',
      payload: { target, properties },
      requestId: '',
    });

    return response;
  }, [connection]);

  const updateDeviceProperties = useCallback(async (targets?: string[], force?: boolean) => {
    return connection.sendMessage({
      type: 'update_properties',
      payload: { targets, force },
      requestId: '',
    });
  }, [connection]);

  // Set the refs to avoid circular dependency
  useEffect(() => {
    listDevicesRef.current = listDevices;
    updateDevicePropertiesRef.current = updateDeviceProperties;
  }, [listDevices, updateDeviceProperties]);

  const discoverDevices = useCallback(async () => {
    return connection.sendMessage({
      type: 'discover_devices',
      payload: {},
      requestId: '',
    });
  }, [connection]);

  const deleteDevice = useCallback(async (target: string) => {
    return connection.sendMessage({
      type: 'delete_device',
      payload: { target },
      requestId: '',
    });
  }, [connection]);

  // Alias operations
  const addAlias = useCallback(async (alias: string, target: string) => {
    return connection.sendMessage({
      type: 'manage_alias',
      payload: { action: 'add', alias, target },
      requestId: '',
    });
  }, [connection]);

  const deleteAlias = useCallback(async (alias: string) => {
    return connection.sendMessage({
      type: 'manage_alias',
      payload: { action: 'delete', alias },
      requestId: '',
    });
  }, [connection]);

  // Group operations
  const addGroup = useCallback(async (group: string, devices: string[]) => {
    return connection.sendMessage({
      type: 'manage_group',
      payload: { action: 'add', group, devices },
      requestId: '',
    });
  }, [connection]);

  const addToGroup = useCallback(async (group: string, devices: string[]) => {
    return connection.sendMessage({
      type: 'manage_group',
      payload: { action: 'add', group, devices },
      requestId: '',
    });
  }, [connection]);

  const removeFromGroup = useCallback(async (group: string, devices: string[]) => {
    return connection.sendMessage({
      type: 'manage_group',
      payload: { action: 'remove', group, devices },
      requestId: '',
    });
  }, [connection]);

  const deleteGroup = useCallback(async (group: string) => {
    return connection.sendMessage({
      type: 'manage_group',
      payload: { action: 'delete', group },
      requestId: '',
    });
  }, [connection]);

  const listGroups = useCallback(async (group: string) => {
    return connection.sendMessage({
      type: 'manage_group',
      payload: { action: 'list', group },
      requestId: '',
    });
  }, [connection]);

  // Property description operations
  // Track in-progress requests to prevent duplicate requests (e.g., from React Strict Mode double mounting)
  const pendingPropertyDescriptionRequestsRef = useRef<PendingPropertyDescriptionRequests>(new Map());

  const getPropertyDescription = useCallback(async (classCode: string, lang?: string): Promise<PropertyDescriptionData> => {
    const currentLang = lang || getCurrentLocale();
    const cacheKey = `${classCode}:${currentLang}`;

    // Check cache first (with language-specific key)
    const cached = state.propertyDescriptions[cacheKey];
    if (cached) {
      return cached;
    }

    // Check if request is already in progress
    const pending = pendingPropertyDescriptionRequestsRef.current.get(cacheKey);
    if (pending) {
      return pending;
    }

    const payload: { classCode: string; lang?: string } = { classCode };
    if (currentLang !== 'en') {
      payload.lang = currentLang;
    }

    // Create and store the pending request
    const requestPromise = connection.sendMessage({
      type: 'get_property_description',
      payload,
      requestId: '',
    }).then(data => {
      const result = data as PropertyDescriptionData;
      // Cache the result with language-specific key
      dispatch({
        type: 'SET_PROPERTY_DESCRIPTION',
        payload: { classCode: cacheKey, data: result },
      });
      return result;
    }).finally(() => {
      // Always remove from pending requests, even if dispatch throws
      pendingPropertyDescriptionRequestsRef.current.delete(cacheKey);
    });

    pendingPropertyDescriptionRequestsRef.current.set(cacheKey, requestPromise);
    return requestPromise;
  }, [connection, state.propertyDescriptions]);

  // Clean up pending requests on unmount
  useEffect(() => {
    const pendingRequests = pendingPropertyDescriptionRequestsRef.current;
    return () => {
      pendingRequests.clear();
    };
  }, []);

  return {
    // State
    devices: state.devices,
    aliases: state.aliases,
    groups: state.groups,
    connectionState: state.connectionState,
    propertyDescriptions: state.propertyDescriptions,
    initialStateReceived: state.initialStateReceived,
    connectedAt: connection.connectedAt,
    serverStartupTime: state.serverStartupTime,
    connection,

    // Device operations
    listDevices,
    setDeviceProperties,
    updateDeviceProperties,
    discoverDevices,
    deleteDevice,

    // Alias operations
    addAlias,
    deleteAlias,

    // Group operations
    addGroup,
    addToGroup,
    removeFromGroup,
    deleteGroup,
    listGroups,

    // Property description operations
    getPropertyDescription,

    // Connection operations
    connect: connection.connect,
    disconnect: connection.disconnect,
    checkConnection: connection.checkConnection,
  };
}