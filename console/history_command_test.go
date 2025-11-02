package console

import (
	"bytes"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"echonet-list/client"
	"echonet-list/echonet_lite"
	"echonet-list/protocol"
)

type stubAliasManager struct{}

func (stubAliasManager) AliasList() []client.AliasIDStringPair                        { return nil }
func (stubAliasManager) AliasSet(alias *string, criteria client.FilterCriteria) error { return nil }
func (stubAliasManager) AliasDelete(alias *string) error                              { return nil }
func (stubAliasManager) AliasGet(alias *string) (*client.IPAndEOJ, error)             { return nil, nil }
func (stubAliasManager) GetAliases(device client.IPAndEOJ) []string                   { return nil }
func (stubAliasManager) GetDeviceByAlias(alias string) (client.IPAndEOJ, bool) {
	return client.IPAndEOJ{}, false
}

type stubGroupManager struct{}

func (stubGroupManager) GroupList(groupName *string) []client.GroupDevicePair { return nil }
func (stubGroupManager) GroupAdd(groupName string, devices []client.IDString) error {
	return nil
}
func (stubGroupManager) GroupRemove(groupName string, devices []client.IDString) error {
	return nil
}
func (stubGroupManager) GroupDelete(groupName string) error { return nil }
func (stubGroupManager) GetDevicesByGroup(groupName string) ([]client.IDString, bool) {
	return nil, false
}

type stubPropertyDescProvider struct{}

func (stubPropertyDescProvider) GetAllPropertyAliases() map[string]client.PropertyDescription {
	return nil
}
func (stubPropertyDescProvider) GetPropertyDesc(classCode client.EOJClassCode, e client.EPCType) (*client.PropertyDesc, bool) {
	return nil, false
}
func (stubPropertyDescProvider) IsPropertyDefaultEPC(classCode client.EOJClassCode, epc client.EPCType) bool {
	return false
}
func (stubPropertyDescProvider) FindPropertyAlias(classCode client.EOJClassCode, alias string) (client.Property, bool) {
	return client.Property{}, false
}
func (stubPropertyDescProvider) AvailablePropertyAliases(classCode client.EOJClassCode) map[string]client.PropertyDescription {
	return nil
}

func TestParseHistoryCommand(t *testing.T) {
	parser := NewCommandParser(stubPropertyDescProvider{}, stubAliasManager{}, stubGroupManager{})

	cmd, err := parser.ParseCommand("history 192.168.1.20 0130:1 -limit 10 -all", false)
	if err != nil {
		t.Fatalf("ParseCommand returned error: %v", err)
	}

	if cmd.Type != CmdHistory {
		t.Fatalf("expected CmdHistory, got %v", cmd.Type)
	}

	if cmd.HistoryOptions.Limit != 10 {
		t.Fatalf("expected limit 10, got %d", cmd.HistoryOptions.Limit)
	}
	if cmd.HistoryOptions.SettableOnly == nil || *cmd.HistoryOptions.SettableOnly {
		t.Fatalf("expected settableOnly=false, got %v", cmd.HistoryOptions.SettableOnly)
	}
}

type historyClientStub struct {
	devices        []client.IPAndEOJ
	historyEntries []client.DeviceHistoryEntry
	lastDevice     *client.IPAndEOJ
	lastOptions    client.DeviceHistoryOptions
}

func (s *historyClientStub) IsDebug() bool                                 { return false }
func (s *historyClientStub) SetDebug(bool)                                 {}
func (s *historyClientStub) DebugSetOffline(string, bool) error            { return nil }
func (s *historyClientStub) IsOfflineDevice(client.IPAndEOJ) bool          { return false }
func (s *historyClientStub) AliasList() []client.AliasIDStringPair         { return nil }
func (s *historyClientStub) AliasSet(*string, client.FilterCriteria) error { return nil }
func (s *historyClientStub) AliasDelete(*string) error                     { return nil }
func (s *historyClientStub) AliasGet(*string) (*client.IPAndEOJ, error)    { return nil, nil }
func (s *historyClientStub) GetAliases(client.IPAndEOJ) []string           { return nil }
func (s *historyClientStub) GetDeviceByAlias(string) (client.IPAndEOJ, bool) {
	return client.IPAndEOJ{}, false
}
func (s *historyClientStub) Discover() error                                    { return nil }
func (s *historyClientStub) UpdateProperties(client.FilterCriteria, bool) error { return nil }
func (s *historyClientStub) GetDevices(client.DeviceSpecifier) []client.IPAndEOJ {
	return append([]client.IPAndEOJ(nil), s.devices...)
}
func (s *historyClientStub) ListDevices(client.FilterCriteria) []client.DeviceAndProperties {
	return nil
}
func (s *historyClientStub) GetProperties(client.IPAndEOJ, []client.EPCType, bool) (client.DeviceAndProperties, error) {
	return client.DeviceAndProperties{}, nil
}
func (s *historyClientStub) SetProperties(client.IPAndEOJ, client.Properties) (client.DeviceAndProperties, error) {
	return client.DeviceAndProperties{}, nil
}
func (s *historyClientStub) GetDeviceHistory(device client.IPAndEOJ, opts client.DeviceHistoryOptions) ([]client.DeviceHistoryEntry, error) {
	s.lastDevice = &device
	s.lastOptions = opts
	return append([]client.DeviceHistoryEntry(nil), s.historyEntries...), nil
}
func (s *historyClientStub) FindDeviceByIDString(client.IDString) *client.IPAndEOJ { return nil }
func (s *historyClientStub) GetIDString(client.IPAndEOJ) client.IDString           { return "" }
func (s *historyClientStub) GetAllPropertyAliases() map[string]client.PropertyDescription {
	return nil
}
func (s *historyClientStub) GetPropertyDesc(classCode client.EOJClassCode, epc client.EPCType) (*client.PropertyDesc, bool) {
	desc := &client.PropertyDesc{Name: "Test Property"}
	return desc, true
}
func (s *historyClientStub) IsPropertyDefaultEPC(client.EOJClassCode, client.EPCType) bool {
	return false
}
func (s *historyClientStub) FindPropertyAlias(client.EOJClassCode, string) (client.Property, bool) {
	return client.Property{}, false
}
func (s *historyClientStub) AvailablePropertyAliases(client.EOJClassCode) map[string]client.PropertyDescription {
	return nil
}
func (s *historyClientStub) GroupList(*string) []client.GroupDevicePair         { return nil }
func (s *historyClientStub) GroupAdd(string, []client.IDString) error           { return nil }
func (s *historyClientStub) GroupRemove(string, []client.IDString) error        { return nil }
func (s *historyClientStub) GroupDelete(string) error                           { return nil }
func (s *historyClientStub) GetDevicesByGroup(string) ([]client.IDString, bool) { return nil, false }
func (s *historyClientStub) Close() error                                       { return nil }

func TestProcessHistoryCommand(t *testing.T) {
	device := client.IPAndEOJ{
		IP:  parseIP(t, "192.168.1.20"),
		EOJ: echonet_lite.MakeEOJ(0x0130, 0x01),
	}

	stub := &historyClientStub{
		devices: []client.IPAndEOJ{device},
		historyEntries: []client.DeviceHistoryEntry{
			{
				Timestamp: time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC),
				EPC:       0x80,
				Value:     protocol.PropertyData{String: "on"},
				Origin:    protocol.HistoryOriginSet,
			},
		},
	}

	processor := &CommandProcessor{handler: stub}

	class := client.EOJClassCode(0x0130)
	instance := client.EOJInstanceCode(0x01)
	spec := client.DeviceSpecifier{
		IP:           &device.IP,
		ClassCode:    &class,
		InstanceCode: &instance,
	}

	cmd := &Command{
		Type:           CmdHistory,
		DeviceSpec:     spec,
		HistoryOptions: client.DeviceHistoryOptions{},
	}

	output := captureOutput(func() {
		if err := processor.processHistoryCommand(cmd); err != nil {
			t.Fatalf("processHistoryCommand returned error: %v", err)
		}
	})

	if stub.lastDevice == nil || !stub.lastDevice.IP.Equal(device.IP) {
		t.Fatalf("expected history request for device %v", device)
	}

	if !strings.Contains(output, "History for") {
		t.Fatalf("expected output to contain history header, got: %s", output)
	}
	if !strings.Contains(output, "value=on") {
		t.Fatalf("expected output to contain value string, got: %s", output)
	}

	if stub.lastOptions.SettableOnly != nil {
		t.Fatalf("expected default settableOnly to be nil")
	}
}

func TestProcessHistoryCommandWithOnlineOfflineEvents(t *testing.T) {
	device := client.IPAndEOJ{
		IP:  parseIP(t, "192.168.1.20"),
		EOJ: echonet_lite.MakeEOJ(0x0130, 0x01),
	}

	stub := &historyClientStub{
		devices: []client.IPAndEOJ{device},
		historyEntries: []client.DeviceHistoryEntry{
			{
				Timestamp: time.Date(2024, 5, 1, 10, 0, 0, 0, time.UTC),
				EPC:       0, // No EPC for event entries
				Value:     protocol.PropertyData{},
				Origin:    protocol.HistoryOriginOnline,
			},
			{
				Timestamp: time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC),
				EPC:       0x80,
				Value:     protocol.PropertyData{String: "on"},
				Origin:    protocol.HistoryOriginSet,
			},
			{
				Timestamp: time.Date(2024, 5, 1, 14, 0, 0, 0, time.UTC),
				EPC:       0, // No EPC for event entries
				Value:     protocol.PropertyData{},
				Origin:    protocol.HistoryOriginOffline,
			},
		},
	}

	processor := &CommandProcessor{handler: stub}

	class := client.EOJClassCode(0x0130)
	instance := client.EOJInstanceCode(0x01)
	spec := client.DeviceSpecifier{
		IP:           &device.IP,
		ClassCode:    &class,
		InstanceCode: &instance,
	}

	cmd := &Command{
		Type:           CmdHistory,
		DeviceSpec:     spec,
		HistoryOptions: client.DeviceHistoryOptions{},
	}

	output := captureOutput(func() {
		if err := processor.processHistoryCommand(cmd); err != nil {
			t.Fatalf("processHistoryCommand returned error: %v", err)
		}
	})

	if stub.lastDevice == nil || !stub.lastDevice.IP.Equal(device.IP) {
		t.Fatalf("expected history request for device %v", device)
	}

	// Verify output contains all entries
	if !strings.Contains(output, "History for") {
		t.Fatalf("expected output to contain history header, got: %s", output)
	}

	// Verify online event is displayed
	if !strings.Contains(output, "Device came online") && !strings.Contains(output, "デバイスがオンラインになりました") {
		t.Fatalf("expected output to contain online event, got: %s", output)
	}

	// Verify offline event is displayed
	if !strings.Contains(output, "Device went offline") && !strings.Contains(output, "デバイスがオフラインになりました") {
		t.Fatalf("expected output to contain offline event, got: %s", output)
	}

	// Verify property change entry is still displayed correctly
	if !strings.Contains(output, "value=on") {
		t.Fatalf("expected output to contain property value, got: %s", output)
	}

	// Verify event entries do not show "value=" field
	lines := strings.Split(output, "\n")
	onlineEventFound := false
	offlineEventFound := false
	for _, line := range lines {
		if strings.Contains(line, "online") && strings.Contains(line, "origin=online") {
			onlineEventFound = true
			// Event entries should not display "value=" field
			if strings.Contains(line, "value=") {
				t.Fatalf("online event should not display value field, got: %s", line)
			}
		}
		if strings.Contains(line, "offline") && strings.Contains(line, "origin=offline") {
			offlineEventFound = true
			// Event entries should not display "value=" field
			if strings.Contains(line, "value=") {
				t.Fatalf("offline event should not display value field, got: %s", line)
			}
		}
	}

	if !onlineEventFound {
		t.Fatalf("expected to find online event entry in output")
	}
	if !offlineEventFound {
		t.Fatalf("expected to find offline event entry in output")
	}

	// Verify total entry count
	if !strings.Contains(output, "(3 entries)") {
		t.Fatalf("expected output to contain correct entry count, got: %s", output)
	}
}

func parseIP(t *testing.T, addr string) net.IP {
	t.Helper()
	ip := net.ParseIP(addr)
	if ip == nil {
		t.Fatalf("invalid IP: %s", addr)
	}
	return ip
}

func captureOutput(fn func()) string {
	original := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = original

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}
