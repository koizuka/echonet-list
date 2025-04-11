package client

import (
	"echonet-list/echonet_lite"
)

// GetAllPropertyAliases gets all property aliases
func (c *WebSocketClient) GetAllPropertyAliases() []string {
	return echonet_lite.GetAllAliases()
}

// GetPropertyInfo gets information about a property
func (c *WebSocketClient) GetPropertyInfo(classCode EOJClassCode, e EPCType) (*PropertyInfo, bool) {
	return echonet_lite.GetPropertyInfo(classCode, e)
}

// IsPropertyDefaultEPC checks if a property is a default property
func (c *WebSocketClient) IsPropertyDefaultEPC(classCode EOJClassCode, epc EPCType) bool {
	return echonet_lite.IsPropertyDefaultEPC(classCode, epc)
}

// FindPropertyAlias finds a property by its alias
func (c *WebSocketClient) FindPropertyAlias(classCode EOJClassCode, alias string) (Property, bool) {
	return echonet_lite.FindPropertyAlias(classCode, alias)
}

// AvailablePropertyAliases gets all available property aliases for a class
func (c *WebSocketClient) AvailablePropertyAliases(classCode EOJClassCode) map[string]string {
	return echonet_lite.AvailablePropertyAliases(classCode)
}
