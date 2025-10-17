package protocol

import (
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/handler"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// PropertyData represents the data for a single property, including its raw EDT and string representation.
type PropertyData struct {
	EDT    string `json:"EDT,omitempty"`    // Base64 encoded EDT, omitted if empty
	String string `json:"string,omitempty"` // String representation of EDT, omitted if empty
	Number *int   `json:"number,omitempty"` // Numeric value, omitted if nil. Only usable when PropertyDesc has NumberDesc.
}

// HistoryOrigin identifies the source of a history entry.
type HistoryOrigin string

const (
	// HistoryOriginNotification indicates that the entry originated from a property change notification.
	HistoryOriginNotification HistoryOrigin = "notification"
	// HistoryOriginSet indicates that the entry originated from a set_properties command.
	HistoryOriginSet HistoryOrigin = "set"
	// HistoryOriginOnline indicates that the entry originated from a device online event.
	HistoryOriginOnline HistoryOrigin = "online"
	// HistoryOriginOffline indicates that the entry originated from a device offline event.
	HistoryOriginOffline HistoryOrigin = "offline"
)

// HistoryEntry represents a single history record for a device.
type HistoryEntry struct {
	Timestamp time.Time     `json:"timestamp"`
	EPC       string        `json:"epc,omitempty"` // EPC is omitted for event entries (online/offline)
	Value     PropertyData  `json:"value"`
	Origin    HistoryOrigin `json:"origin"`
	Settable  bool          `json:"settable"`
}

// DeviceHistoryResponse is the payload returned for get_device_history.
type DeviceHistoryResponse struct {
	Entries []HistoryEntry `json:"entries"`
}

// MessageType defines the type of message being sent between client and server
type MessageType string

const (
	// Server -> Client message types
	MessageTypeInitialState        MessageType = "initial_state"
	MessageTypeDeviceAdded         MessageType = "device_added"
	MessageTypeAliasChanged        MessageType = "alias_changed"
	MessageTypeGroupChanged        MessageType = "group_changed"
	MessageTypePropertyChanged     MessageType = "property_changed"
	MessageTypeTimeoutNotification MessageType = "timeout_notification"
	MessageTypeDeviceOffline       MessageType = "device_offline"
	MessageTypeDeviceOnline        MessageType = "device_online"
	MessageTypeDeviceDeleted       MessageType = "device_deleted"
	MessageTypeErrorNotification   MessageType = "error_notification"
	MessageTypeCommandResult       MessageType = "command_result"

	// Client -> Server message types
	MessageTypeGetProperties          MessageType = "get_properties"
	MessageTypeSetProperties          MessageType = "set_properties"
	MessageTypeUpdateProperties       MessageType = "update_properties"
	MessageTypeListDevices            MessageType = "list_devices"
	MessageTypeManageAlias            MessageType = "manage_alias"
	MessageTypeManageGroup            MessageType = "manage_group"
	MessageTypeDiscoverDevices        MessageType = "discover_devices"
	MessageTypeGetPropertyDescription MessageType = "get_property_description"
	MessageTypeDeleteDevice           MessageType = "delete_device"
	MessageTypeDebugSetOffline        MessageType = "debug_set_offline"
	MessageTypeGetDeviceHistory       MessageType = "get_device_history"
)

// AliasChangeType defines the type of alias change
type AliasChangeType string

const (
	AliasChangeTypeAdded   AliasChangeType = "added"
	AliasChangeTypeUpdated AliasChangeType = "updated"
	AliasChangeTypeDeleted AliasChangeType = "deleted"
)

// AliasAction defines the action to perform on an alias
type AliasAction string

const (
	AliasActionAdd    AliasAction = "add"
	AliasActionDelete AliasAction = "delete"
)

// ErrorCode defines error codes for error messages
type ErrorCode string

// Client Request Related
const (
	ErrorCodeInvalidRequestFormat ErrorCode = "INVALID_REQUEST_FORMAT"
	ErrorCodeInvalidParameters    ErrorCode = "INVALID_PARAMETERS"
	ErrorCodeTargetNotFound       ErrorCode = "TARGET_NOT_FOUND" // not used
	ErrorCodeAliasOperationFailed ErrorCode = "ALIAS_OPERATION_FAILED"
	ErrorCodeAliasAlreadyExists   ErrorCode = "ALIAS_ALREADY_EXISTS" // not used
	ErrorCodeInvalidAliasName     ErrorCode = "INVALID_ALIAS_NAME"   // not used
	ErrorCodeAliasNotFound        ErrorCode = "ALIAS_NOT_FOUND"      // not used
)

// Server/Communication Related
const (
	ErrorCodeEchonetTimeout            ErrorCode = "ECHONET_TIMEOUT"
	ErrorCodeEchonetDeviceError        ErrorCode = "ECHONET_DEVICE_ERROR" // not used
	ErrorCodeEchonetCommunicationError ErrorCode = "ECHONET_COMMUNICATION_ERROR"
	ErrorCodeInternalServerError       ErrorCode = "INTERNAL_SERVER_ERROR"
)

// Message is the base structure for all WebSocket messages
type Message struct {
	Type      MessageType     `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	RequestID string          `json:"requestId,omitempty"`
}

// Device represents an ECHONET Lite device
type Device struct {
	IP         string                  `json:"ip"`
	EOJ        string                  `json:"eoj"`
	Name       string                  `json:"name"`
	ID         handler.IDString        `json:"id,omitempty"`
	Properties map[string]PropertyData `json:"properties"`
	LastSeen   time.Time               `json:"lastSeen"`
	IsOffline  bool                    `json:"isOffline,omitempty"`
}

// Error represents an error in the WebSocket protocol
type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// InitialStatePayload is the payload for the initial_state message
type InitialStatePayload struct {
	Devices           map[string]Device             `json:"devices"`
	Aliases           map[string]handler.IDString   `json:"aliases"`
	Groups            map[string][]handler.IDString `json:"groups"`
	ServerStartupTime time.Time                     `json:"serverStartupTime"`
}

// DeviceAddedPayload is the payload for the device_added message
type DeviceAddedPayload struct {
	Device Device `json:"device"`
}

// AliasChangedPayload is the payload for the alias_changed message
type AliasChangedPayload struct {
	ChangeType AliasChangeType  `json:"change_type"`
	Alias      string           `json:"alias"`
	Target     handler.IDString `json:"target"`
}

// PropertyChangedPayload is the payload for the property_changed message
type PropertyChangedPayload struct {
	IP    string       `json:"ip"`
	EOJ   string       `json:"eoj"`
	EPC   string       `json:"epc"`
	Value PropertyData `json:"value"`
}

// TimeoutNotificationPayload is the payload for the timeout_notification message
type TimeoutNotificationPayload struct {
	IP      string    `json:"ip"`
	EOJ     string    `json:"eoj"`
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// DeviceOfflinePayload is the payload for the device_offline message
type DeviceOfflinePayload struct {
	IP  string `json:"ip"`
	EOJ string `json:"eoj"`
}

// DeviceOnlinePayload is the payload for the device_online message
type DeviceOnlinePayload struct {
	IP  string `json:"ip"`
	EOJ string `json:"eoj"`
}

// DeviceDeletedPayload is the payload for the device_deleted message
type DeviceDeletedPayload struct {
	IP  string `json:"ip"`
	EOJ string `json:"eoj"`
}

// ErrorNotificationPayload is the payload for the error_notification message
type ErrorNotificationPayload struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// CommandResultPayload is the payload for the command_result message
type CommandResultPayload struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// GetPropertiesPayload is the payload for the get_properties message
type GetPropertiesPayload struct {
	Targets []string `json:"targets"`
	EPCs    []string `json:"epcs"`
}

// SetPropertiesPayload is the payload for the set_properties message
type SetPropertiesPayload struct {
	Target     string                  `json:"target"`
	Properties map[string]PropertyData `json:"properties"`
}

// UpdatePropertiesPayload is the payload for the update_properties message
type UpdatePropertiesPayload struct {
	Targets []string `json:"targets"`
	Force   bool     `json:"force,omitempty"`
}

// GetDeviceHistoryPayload is the payload for the get_device_history message
type GetDeviceHistoryPayload struct {
	Target       string `json:"target"`
	Limit        *int   `json:"limit,omitempty"`
	Since        string `json:"since,omitempty"`
	SettableOnly *bool  `json:"settableOnly,omitempty"`
}

// ManageAliasPayload is the payload for the manage_alias message
type ManageAliasPayload struct {
	Action AliasAction      `json:"action"`
	Alias  string           `json:"alias"`
	Target handler.IDString `json:"target,omitempty"`
}

// GroupChangeType defines the type of group change
type GroupChangeType string

const (
	GroupChangeTypeAdded   GroupChangeType = "added"
	GroupChangeTypeUpdated GroupChangeType = "updated"
	GroupChangeTypeDeleted GroupChangeType = "deleted"
)

// GroupAction defines the action to perform on a group
type GroupAction string

const (
	GroupActionAdd    GroupAction = "add"
	GroupActionRemove GroupAction = "remove"
	GroupActionDelete GroupAction = "delete"
	GroupActionList   GroupAction = "list"
)

// GroupChangedPayload is the payload for the group_changed message
type GroupChangedPayload struct {
	ChangeType GroupChangeType    `json:"change_type"`
	Group      string             `json:"group"`
	Devices    []handler.IDString `json:"devices,omitempty"`
}

// ManageGroupPayload is the payload for the manage_group message
type ManageGroupPayload struct {
	Action  GroupAction        `json:"action"`
	Group   string             `json:"group"`
	Devices []handler.IDString `json:"devices,omitempty"`
}

// DiscoverDevicesPayload is the payload for the discover_devices message
type DiscoverDevicesPayload struct {
	// Empty payload
}

// ListDevicesPayload is the payload for the list_devices message
type ListDevicesPayload struct {
	Targets []string `json:"targets,omitempty"` // Specific device identifiers to filter (optional)
}

// GetPropertyDescriptionPayload is the payload for the get_property_description message
type GetPropertyDescriptionPayload struct {
	ClassCode string `json:"classCode"`
	Lang      string `json:"lang,omitempty"` // Language code (e.g., "ja", "en"). Defaults to "en" if not specified
}

// DeleteDevicePayload is the payload for the delete_device message
type DeleteDevicePayload struct {
	Target string `json:"target"` // Device identifier (IP EOJ format)
}

// DebugSetOfflinePayload is the payload for the debug_set_offline command
type DebugSetOfflinePayload struct {
	Target  string `json:"target"`  // Device identifier (IP EOJ format)
	Offline bool   `json:"offline"` // true to set offline, false to set online
}

// PropertyDescriptionData is the data for the command_result message when success is true
// It's included in the 'data' field of CommandResultPayload for get_property_description requests
type PropertyDescriptionData struct {
	ClassCode  string             `json:"classCode"`
	Properties map[string]EPCDesc `json:"properties"` // EPC in hex format (e.g. "80") -> EPCDesc
}

// ProtocolNumberDesc defines the structure for numeric property details in the protocol.
type ProtocolNumberDesc struct {
	Min    int    `json:"min"`              // Minimum value
	Max    int    `json:"max"`              // Maximum value
	Offset int    `json:"offset"`           // Offset value used in ECHONET Lite
	Unit   string `json:"unit,omitempty"`   // Unit of the value (e.g., "C", "%"), omitted if empty
	EdtLen int    `json:"edtLen,omitempty"` // Length of EDT in bytes, omitted if 1 (default)
}

// ProtocolStringDesc defines the structure for string property details in the protocol.
type ProtocolStringDesc struct {
	MinEDTLen int `json:"minEDTLen,omitempty"` // Minimum EDT length (padded with NUL), omitted if 0
	MaxEDTLen int `json:"maxEDTLen,omitempty"` // Maximum EDT length, omitted if 0 (no limit)
}

// EPCDesc contains information about an EPC, including its description, aliases, and potentially numeric/string details.
type EPCDesc struct {
	Description       string              `json:"description"`                 // EPC description (e.g. "Operation status")
	Aliases           map[string]string   `json:"aliases,omitempty"`           // Alias name -> EDT in base64 format (optional)
	AliasTranslations map[string]string   `json:"aliasTranslations,omitempty"` // Alias translation table for current language (e.g., "on" -> "オン") (optional)
	NumberDesc        *ProtocolNumberDesc `json:"numberDesc,omitempty"`        // Details if the property is numeric (optional)
	StringDesc        *ProtocolStringDesc `json:"stringDesc,omitempty"`        // Details if the property is a string (optional)
	StringSettable    bool                `json:"stringSettable,omitempty"`    // Indicates if the property is settable as a string (optional)
}

// Helper functions for converting between ECHONET Lite types and protocol types

func MakePropertyData(classCode echonet_lite.EOJClassCode, property echonet_lite.Property) PropertyData {
	edtString := ""
	var number *int

	if desc, ok := echonet_lite.GetPropertyDesc(classCode, property.EPC); ok {
		edtString = desc.EDTToString(property.EDT)

		// If the property has a NumberDesc, try to get the numeric value
		if converter, ok := desc.Decoder.(echonet_lite.PropertyIntConverter); ok {
			if num, _, ok := converter.ToInt(property.EDT); ok {
				number = &num
			}
		}
	}

	return PropertyData{
		EDT:    base64.StdEncoding.EncodeToString(property.EDT),
		String: edtString,
		Number: number, // omitempty により、nilの場合はJSONに出力されない
	}
}

type PropertyMap map[string]PropertyData

func (props PropertyMap) Set(epc echonet_lite.EPCType, data PropertyData) {
	props[fmt.Sprintf("%02X", byte(epc))] = data
}

// DeviceToProtocol converts an ECHONET Lite device to a protocol Device
func DeviceToProtocol(ipAndEOJ echonet_lite.IPAndEOJ, properties echonet_lite.Properties, lastSeen time.Time, isOffline bool) Device {
	protoProps := make(PropertyMap)
	for _, prop := range properties {
		protoProps.Set(prop.EPC, MakePropertyData(ipAndEOJ.EOJ.ClassCode(), prop))
	}

	// Generate IDString from EOJ and properties
	var ids handler.IDString
	if id := properties.GetIdentificationNumber(); id != nil {
		ids = handler.MakeIDString(ipAndEOJ.EOJ, *id)
	}

	return Device{
		IP:         ipAndEOJ.IP.String(),
		EOJ:        ipAndEOJ.EOJ.Specifier(),
		Name:       ipAndEOJ.EOJ.ClassCode().String(),
		ID:         ids,
		Properties: protoProps,
		LastSeen:   lastSeen,
		IsOffline:  isOffline,
	}
}

// DeviceFromProtocol converts a protocol Device to ECHONET Lite types
func DeviceFromProtocol(device Device) (echonet_lite.IPAndEOJ, echonet_lite.Properties, error) {
	ipAndEOJ, err := handler.ParseDeviceIdentifier(device.IP + " " + device.EOJ)
	if err != nil {
		return echonet_lite.IPAndEOJ{}, nil, err
	}

	// Convert properties map to Properties slice
	var properties echonet_lite.Properties
	for epcStr, propData := range device.Properties {
		// Parse EPC string to EPCType
		epc, err := handler.ParseEPCString(epcStr)
		if err != nil {
			return echonet_lite.IPAndEOJ{}, nil, fmt.Errorf("error parsing EPC: %v", err)
		}

		// Determine EDT value with priority: Number > EDT > String
		var edt []byte

		if propData.EDT != "" {
			// Decode base64 string to bytes
			e, err := base64.StdEncoding.DecodeString(propData.EDT)
			if err != nil {
				return echonet_lite.IPAndEOJ{}, nil, fmt.Errorf("error decoding EDT: %v", err)
			}
			edt = e
		} else {
			// Get property description - needed for Number and String conversions
			desc, ok := echonet_lite.GetPropertyDesc(ipAndEOJ.EOJ.ClassCode(), epc)
			if !ok {
				return echonet_lite.IPAndEOJ{}, nil, fmt.Errorf("property description not found for EPC: %s", epcStr)
			}

			if propData.String != "" {
				// Convert string to EDT bytes
				bytes, ok := desc.ToEDT(propData.String)
				if !ok {
					return echonet_lite.IPAndEOJ{}, nil, fmt.Errorf("invalid string value '%s' for property %s", propData.String, epcStr)
				}

				edt = bytes
			} else if propData.Number != nil {
				converter, ok := desc.Decoder.(echonet_lite.PropertyIntConverter)
				if !ok {
					return echonet_lite.IPAndEOJ{}, nil, fmt.Errorf("property %s does not support numeric values", epcStr)
				}

				bytes, ok := converter.FromInt(*propData.Number)
				if !ok {
					return echonet_lite.IPAndEOJ{}, nil, fmt.Errorf("invalid numeric value %d for property %s", *propData.Number, epcStr)
				}

				edt = bytes
			}
		}

		// If we don't have EDT data, skip this property
		if edt == nil {
			continue
		}

		// Add property to properties slice
		properties = append(properties, echonet_lite.Property{
			EPC: epc,
			EDT: edt,
		})
	}

	return ipAndEOJ, properties, nil
}

// CreateMessage creates a new Message with the given type and payload
func CreateMessage(msgType MessageType, payload interface{}, requestID string) ([]byte, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	msg := Message{
		Type:      msgType,
		Payload:   payloadBytes,
		RequestID: requestID,
	}

	return json.Marshal(msg)
}

// ParseMessage parses a JSON message into a Message struct
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ParsePayload parses the payload of a message into the given struct
func ParsePayload(msg *Message, payload interface{}) error {
	return json.Unmarshal(msg.Payload, payload)
}
