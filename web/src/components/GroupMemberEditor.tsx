import { useState } from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Users } from 'lucide-react';
import { SimpleDeviceCard } from '@/components/SimpleDeviceCard';
import { getDeviceAliases } from '@/libs/deviceIdHelper';
import type { Device, DeviceAlias, PropertyDescriptionData } from '@/hooks/types';

interface GroupMemberEditorProps {
  groupName: string;
  groupMembers: string[];
  allDevices: Record<string, Device>;
  aliases: DeviceAlias;
  onAddToGroup: (group: string, devices: string[]) => Promise<void>;
  onRemoveFromGroup: (group: string, devices: string[]) => Promise<void>;
  propertyDescriptions: Record<string, PropertyDescriptionData>;
  getDeviceClassCode: (device: Device) => string;
  isLoading?: boolean;
  onDone?: () => void;
  isConnected?: boolean;
}

export function GroupMemberEditor({
  groupName,
  groupMembers,
  allDevices,
  aliases,
  onAddToGroup,
  onRemoveFromGroup,
  propertyDescriptions,
  getDeviceClassCode,
  isLoading = false,
  onDone,
  isConnected = true,
}: GroupMemberEditorProps) {
  const [dragOverSection, setDragOverSection] = useState<'members' | 'available' | null>(null);
  const [draggingDevice, setDraggingDevice] = useState<string | null>(null);


  // Helper function to find device by matching various ID formats
  const findDeviceByMemberId = (memberId: string) => {
    // Direct match first
    if (allDevices[memberId]) {
      return { id: memberId, device: allDevices[memberId] };
    }
    
    // Search by device identifier using deviceIdHelper
    for (const [deviceKey, device] of Object.entries(allDevices)) {
      const { deviceIdentifier } = getDeviceAliases(device, allDevices, aliases);
      if (deviceIdentifier === memberId) {
        return { id: deviceKey, device };
      }
    }
    
    return null;
  };

  // Split devices into members and non-members
  const memberDevices = groupMembers
    .map(memberId => findDeviceByMemberId(memberId))
    .filter(result => result !== null)
    .map(result => result!);
  
  const memberDeviceKeys = memberDevices.map(item => item.id);
  
  const availableDevices = Object.entries(allDevices)
    .filter(([deviceKey]) => !memberDeviceKeys.includes(deviceKey))
    .map(([id, device]) => ({ id, device }));

  const handleDragStart = (e: React.DragEvent, deviceKey: string) => {
    e.dataTransfer.setData('text/plain', deviceKey);
    e.dataTransfer.effectAllowed = 'move';
    setDraggingDevice(deviceKey);
  };

  const handleDragEnd = () => {
    setDraggingDevice(null);
    setDragOverSection(null);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
  };

  const handleDragEnter = (section: 'members' | 'available') => {
    setDragOverSection(section);
  };

  const handleDragLeave = (e: React.DragEvent, section: 'members' | 'available') => {
    // Check if we're leaving to a child element
    const relatedTarget = e.relatedTarget as Node;
    if (relatedTarget && e.currentTarget.contains(relatedTarget)) {
      return;
    }
    
    if (dragOverSection === section) {
      setDragOverSection(null);
    }
  };

  const handleDrop = async (e: React.DragEvent, section: 'members' | 'available') => {
    e.preventDefault();
    const deviceKey = e.dataTransfer.getData('text/plain');
    
    if (!deviceKey || !allDevices[deviceKey]) return;

    const device = allDevices[deviceKey];
    const { deviceIdentifier } = getDeviceAliases(device, allDevices, aliases);
    
    // Fallback to deviceKey if deviceIdentifier is not available
    const finalDeviceId = deviceIdentifier || deviceKey;
    
    if (!finalDeviceId) {
      console.error('Could not determine device identifier for group operation');
      return;
    }

    try {
      const isCurrentlyMember = memberDeviceKeys.includes(deviceKey);
      
      if (section === 'members' && !isCurrentlyMember) {
        await onAddToGroup(groupName, [finalDeviceId]);
      } else if (section === 'available' && isCurrentlyMember) {
        await onRemoveFromGroup(groupName, [finalDeviceId]);
      }
    } catch (error) {
      console.error('Failed to update group membership:', error);
    } finally {
      setDragOverSection(null);
      setDraggingDevice(null);
    }
  };

  // Button click handlers for add/remove functionality
  const handleAddDevice = async (deviceKey: string) => {
    if (isLoading) return;
    
    const device = allDevices[deviceKey];
    if (!device) return;

    const { deviceIdentifier } = getDeviceAliases(device, allDevices, aliases);
    const finalDeviceId = deviceIdentifier || deviceKey;
    
    if (!finalDeviceId) {
      console.error('Could not determine device identifier for group operation');
      return;
    }

    try {
      await onAddToGroup(groupName, [finalDeviceId]);
    } catch (error) {
      console.error('Failed to add device to group:', error);
    }
  };

  const handleRemoveDevice = async (deviceKey: string) => {
    if (isLoading) return;
    
    const device = allDevices[deviceKey];
    if (!device) return;

    const { deviceIdentifier } = getDeviceAliases(device, allDevices, aliases);
    const finalDeviceId = deviceIdentifier || deviceKey;
    
    if (!finalDeviceId) {
      console.error('Could not determine device identifier for group operation');
      return;
    }

    try {
      await onRemoveFromGroup(groupName, [finalDeviceId]);
    } catch (error) {
      console.error('Failed to remove device from group:', error);
    }
  };

  const renderDeviceCard = (deviceKey: string, device: Device, isMember: boolean) => {
    return (
      <SimpleDeviceCard
        key={deviceKey}
        deviceKey={deviceKey}
        device={device}
        allDevices={allDevices}
        aliases={aliases}
        propertyDescriptions={propertyDescriptions}
        getDeviceClassCode={getDeviceClassCode}
        isDraggable={true}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
        isDragging={draggingDevice === deviceKey}
        isLoading={isLoading}
        actionButton={{
          type: isMember ? 'remove' : 'add',
          onClick: () => isMember ? handleRemoveDevice(deviceKey) : handleAddDevice(deviceKey),
          title: isMember ? "グループから削除" : "グループに追加",
          disabled: isLoading || !isConnected
        }}
      />
    );
  };

  return (
    <div className="space-y-4">
      {/* Done Button - only show if onDone callback is provided */}
      {onDone && (
        <Card>
          <CardContent className="flex flex-wrap gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={onDone}
              disabled={isLoading || !isConnected}
              title="メンバー編集を終了"
              data-testid="done-editing-button"
            >
              <Users className="h-4 w-4 sm:mr-2" />
              <span className="hidden sm:inline">編集を終了</span>
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Group Members Section */}
      <div className="space-y-2">
        <h3 className="text-sm font-medium">{groupName} のメンバー</h3>
        <div
          data-testid="group-members-section"
          className={`min-h-[200px] p-4 border-2 border-dashed rounded-lg transition-colors ${
            dragOverSection === 'members' ? 'border-primary bg-primary/10 drag-over' : 'border-muted-foreground/30'
          }`}
          onDragOver={handleDragOver}
          onDragEnter={() => handleDragEnter('members')}
          onDragLeave={(e) => handleDragLeave(e, 'members')}
          onDrop={(e) => handleDrop(e, 'members')}
        >
          {memberDevices.length === 0 ? (
            <p className="text-center text-muted-foreground text-sm">
              デバイスをここにドラッグしてグループに追加
            </p>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
              {memberDevices.map(({ id, device }) => renderDeviceCard(id, device, true))}
            </div>
          )}
        </div>
      </div>

      {/* Available Devices Section */}
      <div className="space-y-2">
        <h3 className="text-sm font-medium">利用可能なデバイス</h3>
        <div
          data-testid="available-devices-section"
          className={`min-h-[200px] p-4 border-2 border-dashed rounded-lg transition-colors ${
            dragOverSection === 'available' ? 'border-primary bg-primary/10 drag-over' : 'border-muted-foreground/30'
          }`}
          onDragOver={handleDragOver}
          onDragEnter={() => handleDragEnter('available')}
          onDragLeave={(e) => handleDragLeave(e, 'available')}
          onDrop={(e) => handleDrop(e, 'available')}
        >
          {Object.keys(allDevices).length === 0 ? (
            <p className="text-center text-muted-foreground text-sm">
              利用可能なデバイスがありません
            </p>
          ) : availableDevices.length === 0 ? (
            <p className="text-center text-muted-foreground text-sm">
              すべてのデバイスがグループに登録されています
            </p>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
              {availableDevices.map(({ id, device }) => renderDeviceCard(id, device, false))}
            </div>
          )}
        </div>
      </div>

    </div>
  );
}