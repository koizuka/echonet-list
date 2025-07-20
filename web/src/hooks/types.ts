// ECHONET WebSocket Protocol Types

export type Device = {
  ip: string;
  eoj: string;
  name: string;
  id: string | undefined; // Format: EOJ:ManufacturerCode:UniqueIdentifier, undefined when IdentificationNumber (EPC 0x83) is not available
  properties: Record<string, PropertyValue>;
  lastSeen: string; // ISO 8601 format
  isOffline?: boolean; // true when device is offline
};

export type PropertyValue = {
  EDT?: string; // Base64 encoded string
  string?: string; // Human readable string representation
  number?: number; // Numeric representation (if NumberDesc exists)
};

export type DeviceAlias = Record<string, string>; // alias -> device ID string
export type DeviceGroup = Record<string, string[]>; // group name -> device ID strings

export type ErrorInfo = {
  code: string;
  message: string;
};

// Server -> Client Messages (Notifications)
export type InitialState = {
  type: 'initial_state';
  payload: {
    devices: Record<string, Device>;
    aliases: DeviceAlias;
    groups: DeviceGroup;
  };
};

export type DeviceAdded = {
  type: 'device_added';
  payload: {
    device: Device;
  };
};

export type AliasChanged = {
  type: 'alias_changed';
  payload: {
    change_type: 'added' | 'updated' | 'deleted';
    alias: string;
    target?: string; // device ID string, omitted for "deleted"
  };
};

export type PropertyChanged = {
  type: 'property_changed';
  payload: {
    ip: string;
    eoj: string;
    epc: string; // 2-digit hex string
    value: PropertyValue;
  };
};

export type TimeoutNotification = {
  type: 'timeout_notification';
  payload: {
    ip: string;
    eoj: string;
    code: string;
    message: string;
  };
};

export type DeviceOffline = {
  type: 'device_offline';
  payload: {
    ip: string;
    eoj: string;
  };
};

export type DeviceOnline = {
  type: 'device_online';
  payload: {
    ip: string;
    eoj: string;
  };
};

export type DeviceDeleted = {
  type: 'device_deleted';
  payload: {
    ip: string;
    eoj: string;
  };
};

export type GroupChanged = {
  type: 'group_changed';
  payload: {
    change_type: 'added' | 'updated' | 'deleted';
    group: string;
    devices?: string[]; // device ID strings, omitted for "deleted"
  };
};

export type ErrorNotification = {
  type: 'error_notification';
  payload: ErrorInfo;
};

export type LogNotification = {
  type: 'log_notification';
  payload: {
    level: 'ERROR' | 'WARN';
    message: string;
    time: string; // ISO 8601 format
    attributes: Record<string, unknown>;
  };
};

export type ServerMessage = 
  | InitialState
  | DeviceAdded
  | AliasChanged
  | PropertyChanged
  | TimeoutNotification
  | DeviceOffline
  | DeviceOnline
  | DeviceDeleted
  | GroupChanged
  | ErrorNotification
  | LogNotification;

// Client -> Server Messages (Requests)
export type BaseRequest<T = Record<string, unknown>> = {
  type: string;
  payload: T;
  requestId: string;
};

export type GetPropertiesRequest = BaseRequest<{
  targets: string[]; // device ID strings (IP EOJ format)
  epcs: string[]; // EPC strings
}>;

export type SetPropertiesRequest = BaseRequest<{
  target: string; // device ID string (IP EOJ format)
  properties: Record<string, PropertyValue>;
}>;

export type UpdatePropertiesRequest = BaseRequest<{
  targets?: string[]; // device ID strings, if omitted all devices are updated
  force?: boolean; // force update flag
}>;

export type ManageAliasRequest = BaseRequest<{
  action: 'add' | 'delete';
  alias: string;
  target?: string; // device ID string, required for "add"
}>;

export type ManageGroupRequest = BaseRequest<{
  action: 'add' | 'remove' | 'delete' | 'list';
  group: string;
  devices?: string[]; // device ID strings, required for "add" or "remove"
}>;

export type DiscoverDevicesRequest = BaseRequest<Record<string, never>>;

export type GetPropertyDescriptionRequest = BaseRequest<{
  classCode: string; // 4-digit hex string
}>;

export type DeleteDeviceRequest = BaseRequest<{
  target: string; // device ID string (IP EOJ format)
}>;

export type ClientMessage = 
  | GetPropertiesRequest
  | SetPropertiesRequest
  | UpdatePropertiesRequest
  | ManageAliasRequest
  | ManageGroupRequest
  | DiscoverDevicesRequest
  | GetPropertyDescriptionRequest
  | DeleteDeviceRequest;

// Command Result Response
export type CommandResult = {
  type: 'command_result';
  payload: {
    success: boolean;
    data?: unknown; // success data
    error?: ErrorInfo; // error info
  };
  requestId: string;
};

// Property Description Types
export type PropertyAlias = Record<string, string>; // alias name -> EDT (Base64)

export type NumberDesc = {
  min: number;
  max: number;
  offset: number;
  unit: string;
  edtLen: number;
};

export type StringDesc = {
  minEDTLen: number;
  maxEDTLen: number;
};

// Type for alias translations - backend sends flat structure for requested language
export type AliasTranslations = Record<string, string>; // Flat: { cooling: "冷房" }

export type PropertyDescriptor = {
  description: string;
  aliases?: PropertyAlias;
  aliasTranslations?: AliasTranslations;
  numberDesc?: NumberDesc;
  stringDesc?: StringDesc;
  stringSettable?: boolean;
};

export type PropertyDescriptionData = {
  classCode: string;
  properties: Record<string, PropertyDescriptor>; // EPC -> PropertyDescriptor
};

// WebSocket Connection State
export type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'error';

// Hook State Types
export type ECHONETState = {
  devices: Record<string, Device>;
  aliases: DeviceAlias;
  groups: DeviceGroup;
  connectionState: ConnectionState;
  propertyDescriptions: Record<string, PropertyDescriptionData>; // classCode -> PropertyDescriptionData
  initialStateReceived: boolean;
};