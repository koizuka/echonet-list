import { usePropertyDescriptions } from '@/hooks/usePropertyDescriptions';
import { getAllTabs, getDevicesForTab as getDevicesForTabHelper, hasAnyOperationalDevice, hasAnyFaultyDevice, translateLocationId, getLocationDisplayName } from '@/libs/locationHelper';
import { useCardExpansion } from '@/hooks/useCardExpansion';
import { usePersistedTab } from '@/hooks/usePersistedTab';
import { useAutoReconnect } from '@/hooks/useAutoReconnect';
import { DeviceCard } from '@/components/DeviceCard';
import { useLogNotifications } from '@/hooks/useLogNotifications';
import { NotificationBell } from '@/components/NotificationBell';
import { GroupNameEditor } from '@/components/GroupNameEditor';
import { GroupMemberEditor } from '@/components/GroupMemberEditor';
import { GroupManagementPanel } from '@/components/GroupManagementPanel';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ExpandIcon, ShrinkIcon, Plus } from 'lucide-react';
import { useState, useEffect, useMemo, useRef } from 'react';
import type { PropertyValue, LogNotification } from '@/hooks/types';
import type { LogEntry } from '@/hooks/useLogNotifications';

function App() {
  // 開発環境と本番環境でWebSocket URLを切り替え
  const wsUrl = import.meta.env.DEV 
    ? (import.meta.env.VITE_WS_URL || 'wss://localhost:8080/ws')  // 開発時は環境変数またはデフォルト値
    : `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`; // 本番時は現在のホストを使用
  
  // Track the latest log notification
  const [latestLogNotification, setLatestLogNotification] = useState<LogNotification | undefined>();
  
  // Log notification state
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);

  // Log notification handlers
  const logManager = useLogNotifications({ 
    notification: latestLogNotification,
    onLogsChange: (newLogs, newUnreadCount) => {
      setLogs(newLogs);
      setUnreadCount(newUnreadCount);
    }
  });

  const echonet = usePropertyDescriptions(wsUrl, (message) => {
    // Handle log notifications
    if (message.type === 'log_notification') {
      setLatestLogNotification(message);
    }
  }, () => {
    // Clear WebSocket connection errors when successfully connected
    logManager.clearByCategory('WebSocket');
  });
  const cardExpansion = useCardExpansion();
  
  // Compute isConnected from connectionState to avoid unnecessary re-renders
  const isConnected = echonet.connectionState === 'connected';
  
  // Auto-reconnect when page/browser becomes active and auto-disconnect when hidden
  useAutoReconnect({
    connectionState: echonet.connectionState,
    connect: echonet.connect,
    disconnect: echonet.disconnect,
  });
  
  // Loading state for update operations
  const [updatingDevices, setUpdatingDevices] = useState<Set<string>>(new Set());
  // Track alias operations loading state
  const [aliasLoading, setAliasLoading] = useState(false);
  // Loading state for delete operations
  const [deletingDevices, setDeletingDevices] = useState<Set<string>>(new Set());
  
  // Group management states
  const [isCreatingGroup, setIsCreatingGroup] = useState(false);
  const [editingGroupName, setEditingGroupName] = useState<string | null>(null);
  const [editingGroupMembers, setEditingGroupMembers] = useState<string | null>(null);
  const [groupOperationLoading, setGroupOperationLoading] = useState(false);
  const [newGroupTabName, setNewGroupTabName] = useState<string | null>(null);
  const [pendingGroupName, setPendingGroupName] = useState<string | null>(null);
  const isAutoSelectingRef = useRef(false);

  // Get all tab IDs (location IDs + groups + new group tab if creating)
  const tabIds = useMemo(() => {
    const baseTabIds = getAllTabs(echonet.devices, echonet.groups);
    const additionalTabs = [];
    if (newGroupTabName) additionalTabs.push(newGroupTabName);
    if (pendingGroupName && !baseTabIds.includes(pendingGroupName)) additionalTabs.push(pendingGroupName);
    return [...baseTabIds, ...additionalTabs];
  }, [echonet.devices, echonet.groups, newGroupTabName, pendingGroupName]);
  
  // Use persistent tab selection
  const { selectedTab, selectTab } = usePersistedTab(tabIds, 'All');

  // Auto-select new group tab when it's created
  useEffect(() => {
    if (newGroupTabName && isCreatingGroup && tabIds.includes(newGroupTabName)) {
      isAutoSelectingRef.current = true;
      selectTab(newGroupTabName);
      // Reset the flag after a short delay
      setTimeout(() => {
        isAutoSelectingRef.current = false;
      }, 100);
    }
  }, [newGroupTabName, isCreatingGroup, tabIds, selectTab]);

  // Auto-select pending group tab when it's created
  useEffect(() => {
    if (pendingGroupName && editingGroupMembers === pendingGroupName && tabIds.includes(pendingGroupName)) {
      isAutoSelectingRef.current = true;
      selectTab(pendingGroupName);
      // Reset the flag after a short delay
      setTimeout(() => {
        isAutoSelectingRef.current = false;
      }, 100);
    }
  }, [pendingGroupName, editingGroupMembers, tabIds, selectTab]);

  // Cancel group creation when switching away from the new group tab
  useEffect(() => {
    if (isCreatingGroup && newGroupTabName && selectedTab !== newGroupTabName && !isAutoSelectingRef.current) {
      setIsCreatingGroup(false);
      setNewGroupTabName(null);
    }
    // Also cancel pending group creation
    if (pendingGroupName && editingGroupMembers === pendingGroupName && selectedTab !== pendingGroupName && !isAutoSelectingRef.current) {
      setPendingGroupName(null);
      setEditingGroupMembers(null);
    }
  }, [selectedTab, isCreatingGroup, newGroupTabName, pendingGroupName, editingGroupMembers]);

  // Property change handler
  const handlePropertyChange = async (target: string, epc: string, value: PropertyValue) => {
    try {
      await echonet.setDeviceProperties(target, { [epc]: value });
    } catch (error) {
      console.error('Failed to change property:', error);
      // TODO: Show user-friendly error message
    }
  };

  // Update properties handler
  const handleUpdateProperties = async (target: string) => {
    try {
      // Add device to updating set
      setUpdatingDevices(prev => new Set([...prev, target]));
      
      console.log('Updating properties for:', target);
      await echonet.updateDeviceProperties([target]);
    } catch (error) {
      console.error('Failed to update properties:', error);
      // TODO: Show user-friendly error message
    } finally {
      // Remove device from updating set
      setUpdatingDevices(prev => {
        const newSet = new Set(prev);
        newSet.delete(target);
        return newSet;
      });
    }
  };

  // Delete device handler
  const handleDeleteDevice = async (target: string) => {
    try {
      // Add device to deleting set
      setDeletingDevices(prev => new Set([...prev, target]));
      
      console.log('Deleting device:', target);
      await echonet.deleteDevice(target);
    } catch (error) {
      console.error('Failed to delete device:', error);
      // TODO: Show user-friendly error message
    } finally {
      // Remove device from deleting set
      setDeletingDevices(prev => {
        const newSet = new Set(prev);
        newSet.delete(target);
        return newSet;
      });
    }
  };

  // Add alias handler
  const handleAddAlias = async (alias: string, target: string) => {
    try {
      setAliasLoading(true);
      await echonet.addAlias(alias, target);
    } catch (error) {
      console.error('Failed to add alias:', error);
      throw error; // Re-throw to let AliasEditor handle the error
    } finally {
      setAliasLoading(false);
    }
  };

  // Delete alias handler
  const handleDeleteAlias = async (alias: string) => {
    try {
      setAliasLoading(true);
      await echonet.deleteAlias(alias);
    } catch (error) {
      console.error('Failed to delete alias:', error);
      throw error; // Re-throw to let AliasEditor handle the error
    } finally {
      setAliasLoading(false);
    }
  };

  // Group handlers
  const handleCreateGroup = async (groupName: string) => {
    // Instead of creating empty group, transition to member editing mode
    setIsCreatingGroup(false);
    setNewGroupTabName(null);
    setPendingGroupName(groupName);
    setEditingGroupMembers(groupName);
    // selectTab will be called via useEffect when tabIds is updated
  };

  const handleRenameGroup = async (oldName: string, newName: string) => {
    try {
      setGroupOperationLoading(true);
      // Get current members
      const members = echonet.groups[oldName] || [];
      // Create new group with members
      await echonet.addGroup(newName, members);
      // Delete old group
      await echonet.deleteGroup(oldName);
      setEditingGroupName(null);
      selectTab(newName); // Switch to the renamed group tab
    } catch (error) {
      console.error('Failed to rename group:', error);
      throw error;
    } finally {
      setGroupOperationLoading(false);
    }
  };

  const handleDeleteGroup = async (groupName: string) => {
    try {
      setGroupOperationLoading(true);
      await echonet.deleteGroup(groupName);
      selectTab('All'); // Switch to All tab after deletion
    } catch (error) {
      console.error('Failed to delete group:', error);
    } finally {
      setGroupOperationLoading(false);
    }
  };

  const getConnectionColor = (state: string) => {
    switch (state) {
      case 'connected':
        return 'bg-green-500';
      case 'connecting':
        return 'bg-yellow-500';
      case 'error':
        return 'bg-red-500';
      default:
        return 'bg-gray-500';
    }
  };

  // Function to get devices for a specific tab ID
  const getDevicesForTab = (tabId: string) => {
    return getDevicesForTabHelper(tabId, echonet.devices, echonet.groups);
  };

  // Get all device keys for expand/collapse all functionality
  const allDeviceKeys = Object.keys(echonet.devices).map(key => {
    const device = echonet.devices[key];
    return `${device.ip} ${device.eoj}`;
  });

  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="container mx-auto p-4">
        <div className="flex justify-between items-center mb-4 sm:mb-6">
          <h1 className="text-2xl sm:text-3xl font-bold">ECHONET List</h1>
          <div className="flex items-center gap-1 sm:gap-2">
            {/* Expand/Collapse All Controls */}
            {Object.keys(echonet.devices).length > 0 && (
              <div className="flex items-center gap-1">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => cardExpansion.expandAll(allDeviceKeys)}
                  className="h-7 sm:h-8 px-2 sm:px-3"
                  data-testid="expand-all-button"
                >
                  <ExpandIcon className="h-3 w-3 sm:mr-1" />
                  <span className="hidden sm:inline ml-1">Expand All</span>
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => cardExpansion.collapseAll()}
                  className="h-7 sm:h-8 px-2 sm:px-3"
                  data-testid="collapse-all-button"
                >
                  <ShrinkIcon className="h-3 w-3 sm:mr-1" />
                  <span className="hidden sm:inline ml-1">Collapse All</span>
                </Button>
              </div>
            )}
            <Badge variant="outline" className={`${getConnectionColor(echonet.connectionState)} text-white text-xs`} data-testid="connection-status">
              {echonet.connectionState}
            </Badge>
            
            {/* Notification Bell */}
            <NotificationBell
              logs={logs}
              unreadCount={unreadCount}
              onMarkAllAsRead={logManager.markAllAsRead}
              onClearAll={logManager.clearAllLogs}
              connectedAt={echonet.connectedAt}
              onDiscoverDevices={echonet.discoverDevices}
            />
          </div>
        </div>
        

        {Object.keys(echonet.devices).length === 0 ? (
          <Card>
            <CardContent className="pt-6">
              <p className="text-center text-muted-foreground">
                {echonet.connectionState === 'connected' 
                  ? (echonet.initialStateReceived 
                      ? 'No devices found. Click refresh to discover devices.'
                      : 'サーバーから初期情報が受信できていません'
                    )
                  : `サーバーに接続できません (${echonet.connectionState})`
                }
              </p>
            </CardContent>
          </Card>
        ) : (
          <Tabs value={selectedTab} onValueChange={selectTab} className="w-full" data-testid="device-tabs">
            <div className="w-full mb-4">
              <TabsList className="w-full h-auto p-2 bg-muted flex flex-wrap justify-between gap-2">
              {tabIds.map((tabId) => {
                const tabDevices = getDevicesForTab(tabId);
                const hasOperationalDevice = hasAnyOperationalDevice(tabDevices);
                const hasFaultyDevice = hasAnyFaultyDevice(tabDevices);
                const displayName = tabId.startsWith('@') 
                  ? tabId // Group tabs keep their name as-is
                  : getLocationDisplayName(tabId, echonet.devices, echonet.propertyDescriptions);
                return (
                  <TabsTrigger 
                    key={tabId} 
                    value={tabId} 
                    className="data-[state=active]:bg-primary data-[state=active]:text-primary-foreground data-[state=active]:border-primary border-2 border-muted-foreground/30 bg-background px-2 sm:px-3 py-1.5 sm:py-2 text-xs sm:text-sm rounded-lg"
                    data-testid={`tab-${tabId}`}
                  >
                    <div className="flex items-center gap-1">
                      {tabId !== 'All' && (
                        <div className="flex items-center gap-1">
                          <div 
                            className={`w-2 h-2 rounded-full ${
                              hasOperationalDevice ? 'bg-green-500' : 'bg-gray-400'
                            }`}
                            title={`Power Status: ${hasOperationalDevice ? 'At least one device is ON' : 'All devices are OFF or no devices'}`}
                          />
                          {hasFaultyDevice && (
                            <div 
                              className="w-2 h-2 rounded-full bg-red-500"
                              title="At least one device has a fault"
                            />
                          )}
                        </div>
                      )}
                      <span>{displayName}</span>
                      {tabId !== 'All' && (
                        <span className="ml-1 text-[10px] sm:text-xs">({tabDevices.length})</span>
                      )}
                    </div>
                  </TabsTrigger>
                );
              })}
              {/* Add Group Button */}
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  const tempTabName = '@新規グループ';
                  setNewGroupTabName(tempTabName);
                  setIsCreatingGroup(true);
                }}
                disabled={isCreatingGroup || !isConnected}
                className="h-8 px-2 sm:px-3 text-xs sm:text-sm"
                data-testid="add-group-button"
              >
                <Plus className="h-3 w-3 sm:mr-1" />
                <span className="hidden sm:inline">新規グループ</span>
              </Button>
              </TabsList>
            </div>
            
            
            {tabIds.map((tabId) => (
              <TabsContent key={tabId} value={tabId} className="space-y-4" data-testid={`tab-content-${tabId}`}>
                {/* Show group creation interface if creating a new group in this tab */}
                {tabId === newGroupTabName && isCreatingGroup && (
                  <Card className="mb-4">
                    <CardContent className="pt-6">
                      <GroupNameEditor
                        groupName="@"
                        existingGroups={[...Object.keys(echonet.groups), ...(pendingGroupName ? [pendingGroupName] : [])]}
                        onSave={handleCreateGroup}
                        onCancel={() => {
                          setIsCreatingGroup(false);
                          setNewGroupTabName(null);
                          selectTab('All');
                        }}
                        isLoading={false}
                        isConnected={isConnected}
                      />
                    </CardContent>
                  </Card>
                )}
                
                {/* Show group management panel for group tabs (but not for pending groups) */}
                {tabId.startsWith('@') && !editingGroupName && tabId !== newGroupTabName && tabId !== pendingGroupName && (
                  <GroupManagementPanel
                    groupName={tabId}
                    onRename={() => setEditingGroupName(tabId)}
                    onDelete={() => handleDeleteGroup(tabId)}
                    onEditMembers={() => setEditingGroupMembers(tabId)}
                    isEditingMembers={editingGroupMembers === tabId}
                    onDoneEditingMembers={() => {
                      setEditingGroupMembers(null);
                      // Clear pending group name if this was a new group
                      if (pendingGroupName === tabId) {
                        setPendingGroupName(null);
                      }
                    }}
                    isConnected={isConnected}
                  />
                )}
                
                {/* Show group name editor if editing group name */}
                {editingGroupName === tabId && (
                  <Card>
                    <CardContent className="pt-6">
                      <GroupNameEditor
                        groupName={tabId}
                        existingGroups={Object.keys(echonet.groups).filter(g => g !== tabId)}
                        onSave={(newName) => handleRenameGroup(tabId, newName)}
                        onCancel={() => setEditingGroupName(null)}
                        isLoading={groupOperationLoading}
                        isConnected={isConnected}
                      />
                    </CardContent>
                  </Card>
                )}
                
                {/* Show member editor if editing members */}
                {editingGroupMembers === tabId ? (
                  <GroupMemberEditor
                    groupName={tabId}
                    groupMembers={echonet.groups[tabId] || []}
                    allDevices={echonet.devices}
                    aliases={echonet.aliases}
                    onAddToGroup={async (group, devices) => {
                      await echonet.addToGroup(group, devices);
                      // If this is a pending group, mark it as created
                      if (pendingGroupName === group) {
                        setPendingGroupName(null);
                      }
                    }}
                    onRemoveFromGroup={async (group, devices) => {
                      await echonet.removeFromGroup(group, devices);
                    }}
                    propertyDescriptions={echonet.propertyDescriptions}
                    getDeviceClassCode={echonet.getDeviceClassCode}
                    isLoading={groupOperationLoading}
                    onDone={pendingGroupName === tabId ? () => {
                      setEditingGroupMembers(null);
                      setPendingGroupName(null);
                    } : undefined}
                    isConnected={isConnected}
                  />
                ) : tabId !== newGroupTabName && (
                  <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-6 gap-3 sm:gap-4">
                  {getDevicesForTab(tabId).map((device) => {
                    const deviceKey = `${device.ip} ${device.eoj}`;
                    
                    return (
                      <DeviceCard
                        key={deviceKey}
                        device={device}
                        isExpanded={cardExpansion.isCardExpanded(deviceKey)}
                        onToggleExpansion={() => cardExpansion.toggleCard(deviceKey)}
                        onPropertyChange={handlePropertyChange}
                        onUpdateProperties={handleUpdateProperties}
                        isUpdating={updatingDevices.has(deviceKey)}
                        propertyDescriptions={echonet.propertyDescriptions}
                        getDeviceClassCode={echonet.getDeviceClassCode}
                        devices={echonet.devices}
                        aliases={echonet.aliases}
                        onAddAlias={handleAddAlias}
                        onDeleteAlias={handleDeleteAlias}
                        isAliasLoading={aliasLoading}
                        isConnected={isConnected}
                        onDeleteDevice={handleDeleteDevice}
                        isDeletingDevice={deletingDevices.has(deviceKey)}
                      />
                    );
                  })}
                  </div>
                )}
                
                {!editingGroupMembers && tabId !== newGroupTabName && getDevicesForTab(tabId).length === 0 && (
                  <Card>
                    <CardContent className="pt-6">
                      <p className="text-center text-muted-foreground">
                        {tabId === 'All' 
                          ? 'No devices found.' 
                          : tabId.startsWith('@') 
                            ? `No devices found in group ${tabId}.`
                            : `No devices found in ${translateLocationId(tabId)}.`
                        }
                      </p>
                    </CardContent>
                  </Card>
                )}
              </TabsContent>
            ))}
          </Tabs>
        )}
      </div>
    </div>
  );
}

export default App;