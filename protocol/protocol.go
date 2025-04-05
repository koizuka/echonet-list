package protocol

import (
	"echonet-list/echonet_lite"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// MessageType defines the type of message being sent between client and server
type MessageType string

const (
	// Server -> Client message types
	MessageTypeInitialState          MessageType = "initial_state"
	MessageTypeDeviceAdded           MessageType = "device_added"
	MessageTypeDeviceUpdated         MessageType = "device_updated"
	MessageTypeDeviceRemoved         MessageType = "device_removed"
	MessageTypeAliasChanged          MessageType = "alias_changed"
	MessageTypeGroupChanged          MessageType = "group_changed"
	MessageTypePropertyChanged       MessageType = "property_changed"
	MessageTypeTimeoutNotification   MessageType = "timeout_notification"
	MessageTypeErrorNotification     MessageType = "error_notification"
	MessageTypeCommandResult         MessageType = "command_result"
	MessageTypePropertyAliasesResult MessageType = "property_aliases_result"

	// Client -> Server message types
	MessageTypeGetProperties      MessageType = "get_properties"
	MessageTypeSetProperties      MessageType = "set_properties"
	MessageTypeUpdateProperties   MessageType = "update_properties"
	MessageTypeManageAlias        MessageType = "manage_alias"
	MessageTypeManageGroup        MessageType = "manage_group"
	MessageTypeDiscoverDevices    MessageType = "discover_devices"
	MessageTypeGetPropertyAliases MessageType = "get_property_aliases"
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

const (
	// Client Request Related
	ErrorCodeInvalidRequestFormat ErrorCode = "INVALID_REQUEST_FORMAT"
	ErrorCodeInvalidParameters    ErrorCode = "INVALID_PARAMETERS"
	ErrorCodeTargetNotFound       ErrorCode = "TARGET_NOT_FOUND"
	ErrorCodeAliasOperationFailed ErrorCode = "ALIAS_OPERATION_FAILED"
	ErrorCodeAliasAlreadyExists   ErrorCode = "ALIAS_ALREADY_EXISTS"
	ErrorCodeInvalidAliasName     ErrorCode = "INVALID_ALIAS_NAME"
	ErrorCodeAliasNotFound        ErrorCode = "ALIAS_NOT_FOUND"

	// Server/Communication Related
	ErrorCodeEchonetTimeout            ErrorCode = "ECHONET_TIMEOUT"
	ErrorCodeEchonetDeviceError        ErrorCode = "ECHONET_DEVICE_ERROR"
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
	IP         string            `json:"ip"`
	EOJ        string            `json:"eoj"`
	Name       string            `json:"name"`
	Properties map[string]string `json:"properties"`
	LastSeen   time.Time         `json:"lastSeen"`
}

// Error represents an error in the WebSocket protocol
type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// InitialStatePayload is the payload for the initial_state message
type InitialStatePayload struct {
	Devices map[string]Device   `json:"devices"`
	Aliases map[string]string   `json:"aliases"`
	Groups  map[string][]string `json:"groups"`
}

// DeviceAddedPayload is the payload for the device_added message
type DeviceAddedPayload struct {
	Device Device `json:"device"`
}

// DeviceUpdatedPayload is the payload for the device_updated message
type DeviceUpdatedPayload struct {
	Device Device `json:"device"`
}

// DeviceRemovedPayload is the payload for the device_removed message
type DeviceRemovedPayload struct {
	IP  string `json:"ip"`
	EOJ string `json:"eoj"`
}

// AliasChangedPayload is the payload for the alias_changed message
type AliasChangedPayload struct {
	ChangeType AliasChangeType `json:"change_type"`
	Alias      string          `json:"alias"`
	Target     string          `json:"target"`
}

// PropertyChangedPayload is the payload for the property_changed message
type PropertyChangedPayload struct {
	IP    string `json:"ip"`
	EOJ   string `json:"eoj"`
	EPC   string `json:"epc"`
	Value string `json:"value"`
}

// TimeoutNotificationPayload is the payload for the timeout_notification message
type TimeoutNotificationPayload struct {
	IP      string    `json:"ip"`
	EOJ     string    `json:"eoj"`
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
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
	Target     string            `json:"target"`
	Properties map[string]string `json:"properties"`
}

// UpdatePropertiesPayload is the payload for the update_properties message
type UpdatePropertiesPayload struct {
	Targets []string `json:"targets"`
}

// ManageAliasPayload is the payload for the manage_alias message
type ManageAliasPayload struct {
	Action AliasAction `json:"action"`
	Alias  string      `json:"alias"`
	Target string      `json:"target,omitempty"`
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
	ChangeType GroupChangeType `json:"change_type"`
	Group      string          `json:"group"`
	Devices    []string        `json:"devices,omitempty"`
}

// ManageGroupPayload is the payload for the manage_group message
type ManageGroupPayload struct {
	Action  GroupAction `json:"action"`
	Group   string      `json:"group"`
	Devices []string    `json:"devices,omitempty"`
}

// DiscoverDevicesPayload is the payload for the discover_devices message
type DiscoverDevicesPayload struct {
	// Empty payload
}

// GetPropertyAliasesPayload is the payload for the get_property_aliases message
type GetPropertyAliasesPayload struct {
	ClassCode string `json:"classCode"`
}

// PropertyAliasesResultPayload is the payload for the property_aliases_result message
type PropertyAliasesResultPayload struct {
	Success bool                 `json:"success"`
	Data    *PropertyAliasesData `json:"data,omitempty"`
	Error   *Error               `json:"error,omitempty"`
}

// EPCInfo contains information about an EPC, including its description and aliases
type EPCInfo struct {
	Description string            `json:"description"` // EPC description (e.g. "Operation status")
	Aliases     map[string]string `json:"aliases"`     // Alias name -> EDT in base64 format
}

// PropertyAliasesData is the data for the property_aliases_result message
type PropertyAliasesData struct {
	ClassCode  string             `json:"classCode"`
	Properties map[string]EPCInfo `json:"properties"` // EPC in hex format (e.g. "80") -> EPCInfo
}

// Helper functions for converting between ECHONET Lite types and protocol types

// DeviceToProtocol converts an ECHONET Lite device to a protocol Device
func DeviceToProtocol(ip string, eoj echonet_lite.EOJ, properties echonet_lite.Properties, lastSeen time.Time) Device {
	protoProps := make(map[string]string)
	for _, prop := range properties {
		protoProps[fmt.Sprintf("%02X", byte(prop.EPC))] = base64.StdEncoding.EncodeToString(prop.EDT)
	}

	return Device{
		IP:         ip,
		EOJ:        eoj.Specifier(),
		Name:       eoj.ClassCode().String(),
		Properties: protoProps,
		LastSeen:   lastSeen,
	}
}

// DeviceFromProtocol converts a protocol Device to ECHONET Lite types
func DeviceFromProtocol(device Device) (string, echonet_lite.EOJ, echonet_lite.Properties, error) {
	ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(device.IP + " " + device.EOJ)
	if err != nil {
		return "", 0, nil, err
	}

	// Convert properties map to Properties slice
	var properties echonet_lite.Properties
	for epcStr, edtStr := range device.Properties {
		// Parse EPC string to EPCType
		epc, err := echonet_lite.ParseEPCString(epcStr)
		if err != nil {
			return "", 0, nil, fmt.Errorf("error parsing EPC: %v", err)
		}

		// Decode EDT string from base64
		edt, err := base64.StdEncoding.DecodeString(edtStr)
		if err != nil {
			return "", 0, nil, fmt.Errorf("error decoding EDT: %v", err)
		}

		// Add property to properties slice
		properties = append(properties, echonet_lite.Property{
			EPC: epc,
			EDT: edt,
		})
	}

	return ipAndEOJ.IP.String(), ipAndEOJ.EOJ, properties, nil
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
